import { ContentPacket, SubmitResult } from './protocol';
import {
  cacheRemotePolicy,
  CapturePolicy,
  isSupportedPageUrl,
  loadLocalSettings,
  loadSettings,
  saveSettings
} from './settings';

const QUEUE_KEY = 'dca.pendingCaptures';
const SEEN_KEY = 'dca.seenCaptures';
const RETRY_ALARM = 'dca.retry';
const MAX_QUEUE = 100;
const MAX_SEEN = 500;
const inFlight = new Set<string>();

interface PendingCapture {
  packet: ContentPacket;
  attempts: number;
  lastError?: string;
  queuedAt: string;
}

interface SeenCapture {
  key: string;
  url: string;
  seenAt: string;
}

interface ActivePage {
  url: string;
  title: string;
  observedAt: string;
}

async function readQueue(): Promise<PendingCapture[]> {
  const data = await chrome.storage.local.get(QUEUE_KEY);
  return Array.isArray(data[QUEUE_KEY]) ? data[QUEUE_KEY] : [];
}

async function writeQueue(queue: PendingCapture[]): Promise<void> {
  await chrome.storage.local.set({ [QUEUE_KEY]: queue.slice(-MAX_QUEUE) });
  void sendHeartbeat(queue.length, queue.at(-1)?.lastError);
}

async function readSeen(): Promise<SeenCapture[]> {
  const data = await chrome.storage.local.get(SEEN_KEY);
  return Array.isArray(data[SEEN_KEY]) ? data[SEEN_KEY] : [];
}

function dedupKey(packet: ContentPacket): string | undefined {
  const raw = packet.metadata?.capture;
  if (!raw || typeof raw !== 'object') return undefined;
  const meta = raw as Record<string, unknown>;
  const canonicalUrl = typeof meta.canonicalUrl === 'string' ? meta.canonicalUrl : undefined;
  const fingerprint = typeof meta.contentFingerprint === 'string' ? meta.contentFingerprint : undefined;
  return canonicalUrl && fingerprint ? `${canonicalUrl}::${fingerprint}` : undefined;
}

async function isSeen(key: string): Promise<boolean> {
  return (await readSeen()).some(item => item.key === key);
}

async function rememberSeen(key: string, url: string): Promise<void> {
  const items = (await readSeen()).filter(item => item.key !== key);
  items.push({ key, url, seenAt: new Date().toISOString() });
  await chrome.storage.local.set({ [SEEN_KEY]: items.slice(-MAX_SEEN) });
}

async function enqueue(packet: ContentPacket, error: unknown): Promise<void> {
  const queue = await readQueue();
  const existing = queue.find(item => item.packet.captureId === packet.captureId);
  const message = error instanceof Error ? error.message : String(error);

  if (existing) {
    existing.attempts += 1;
    existing.lastError = message;
  } else {
    queue.push({ packet, attempts: 1, lastError: message, queuedAt: new Date().toISOString() });
  }
  await writeQueue(queue);
}

async function authHeaders(json = false): Promise<Record<string, string>> {
  const settings = await loadSettings();
  const headers: Record<string, string> = {};
  if (json) headers['Content-Type'] = 'application/json';
  if (settings.authToken) headers.Authorization = `Bearer ${settings.authToken}`;
  return headers;
}

async function postPacket(packet: ContentPacket): Promise<SubmitResult> {
  const settings = await loadSettings();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 20_000);

  try {
    const headers = await authHeaders(true);
    headers['X-DCA-Protocol'] = packet.protocolVersion;
    const response = await fetch(`${settings.agentUrl.replace(/\/+$/, '')}/v1/captures`, {
      method: 'POST',
      headers,
      body: JSON.stringify(packet),
      signal: controller.signal
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body?.error || `Local agent returned HTTP ${response.status}`);
    return body as SubmitResult;
  } finally {
    clearTimeout(timeout);
  }
}

async function submit(packet: ContentPacket): Promise<{ queued: boolean; result?: SubmitResult; error?: string }> {
  try {
    return { queued: false, result: await postPacket(packet) };
  } catch (error) {
    await enqueue(packet, error);
    return { queued: true, error: error instanceof Error ? error.message : String(error) };
  }
}

async function submitAuto(packet: ContentPacket): Promise<{ queued?: boolean; skipped?: boolean; reason?: string }> {
  const key = dedupKey(packet);
  if (!key) return submit(packet);
  if (inFlight.has(key) || await isSeen(key)) return { skipped: true, reason: 'duplicate-content' };

  inFlight.add(key);
  try {
    const result = await submit(packet);
    await rememberSeen(key, packet.source.url);
    return result;
  } finally {
    inFlight.delete(key);
  }
}

async function retryQueue(): Promise<void> {
  const queue = await readQueue();
  if (queue.length === 0) return;

  const remaining: PendingCapture[] = [];
  for (const item of queue) {
    try {
      await postPacket(item.packet);
    } catch (error) {
      remaining.push({
        ...item,
        attempts: item.attempts + 1,
        lastError: error instanceof Error ? error.message : String(error)
      });
    }
  }
  await writeQueue(remaining);
}

async function postActivePage(page: ActivePage, sender: chrome.runtime.MessageSender): Promise<void> {
  const settings = await loadSettings();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5_000);
  try {
    await fetch(`${settings.agentUrl.replace(/\/+$/, '')}/v1/browser/active`, {
      method: 'POST',
      headers: await authHeaders(true),
      body: JSON.stringify({
        ...page,
        tabId: sender.tab?.id,
        windowId: sender.tab?.windowId
      }),
      signal: controller.signal
    });
  } catch {
    // Follow Browser 状态是短暂的。捕获投递有自己的持久队列。
  } finally {
    clearTimeout(timeout);
  }
}

async function refreshRemotePolicy(): Promise<CapturePolicy | undefined> {
  const settings = await loadLocalSettings();
  try {
    const response = await fetch(`${settings.agentUrl.replace(/\/+$/, '')}/v1/policy`, {
      headers: await authHeaders()
    });
    if (!response.ok) return undefined;
    const policy = (await response.json()) as CapturePolicy;
    await cacheRemotePolicy(policy);
    return policy;
  } catch {
    return undefined;
  }
}

async function pushPolicy(partial: Partial<CapturePolicy>): Promise<CapturePolicy | undefined> {
  const current = await loadSettings();
  const next: CapturePolicy = {
    autoCapture: partial.autoCapture ?? current.autoCapture,
    archiveAll: partial.archiveAll ?? current.archiveAll,
    captureDelayMs: partial.captureDelayMs ?? current.captureDelayMs,
    domainAllowlist: partial.domainAllowlist ?? current.domainAllowlist,
    domainDenylist: partial.domainDenylist ?? current.domainDenylist
  };
  const settings = await loadLocalSettings();
  try {
    const response = await fetch(`${settings.agentUrl.replace(/\/+$/, '')}/v1/policy`, {
      method: 'PUT',
      headers: await authHeaders(true),
      body: JSON.stringify(next)
    });
    if (!response.ok) return undefined;
    const policy = (await response.json()) as CapturePolicy;
    await cacheRemotePolicy(policy);
    return policy;
  } catch {
    return undefined;
  }
}

async function sendHeartbeat(queueLength?: number, lastError?: string): Promise<void> {
  const settings = await loadLocalSettings();
  const queue = queueLength === undefined ? (await readQueue()).length : queueLength;
  try {
    await fetch(`${settings.agentUrl.replace(/\/+$/, '')}/v1/sensor/heartbeat`, {
      method: 'POST',
      headers: await authHeaders(true),
      body: JSON.stringify({
        queueLength: queue,
        lastError: lastError || undefined,
        version: chrome.runtime.getManifest().version
      })
    });
  } catch {
    // 心跳失败不影响捕获队列。
  }
}

async function health(): Promise<{ ok: boolean; error?: string }> {
  const settings = await loadLocalSettings();
  try {
    const response = await fetch(`${settings.agentUrl.replace(/\/+$/, '')}/health`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    void refreshRemotePolicy();
    void sendHeartbeat();
    return { ok: true };
  } catch (error) {
    return { ok: false, error: error instanceof Error ? error.message : String(error) };
  }
}

function notifyNavigation(tabId: number, url?: string): void {
  if (url && !isSupportedPageUrl(url)) return;
  void chrome.tabs.sendMessage(tabId, { type: 'DCA_NAVIGATION', url }).catch(() => undefined);
}

function notifyActive(tabId: number): void {
  void chrome.tabs.sendMessage(tabId, { type: 'DCA_TAB_ACTIVE' }).catch(() => undefined);
}

chrome.runtime.onInstalled.addListener(() => {
  void chrome.alarms.create(RETRY_ALARM, { periodInMinutes: 1 });
});

chrome.runtime.onStartup.addListener(() => {
  void chrome.alarms.create(RETRY_ALARM, { periodInMinutes: 1 });
});

chrome.alarms.onAlarm.addListener(alarm => {
  if (alarm.name === RETRY_ALARM) {
    void retryQueue();
    void refreshRemotePolicy();
    void sendHeartbeat();
  }
});

chrome.webNavigation.onHistoryStateUpdated.addListener(details => {
  if (details.frameId === 0) notifyNavigation(details.tabId, details.url);
});

chrome.webNavigation.onReferenceFragmentUpdated.addListener(details => {
  if (details.frameId === 0) notifyNavigation(details.tabId, details.url);
});

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === 'complete') notifyNavigation(tabId, tab.url);
});

chrome.tabs.onActivated.addListener(activeInfo => notifyActive(activeInfo.tabId));

chrome.windows.onFocusChanged.addListener(windowId => {
  if (windowId === chrome.windows.WINDOW_ID_NONE) return;
  void chrome.tabs.query({ active: true, windowId }).then(([tab]) => {
    if (tab?.id) notifyActive(tab.id);
  });
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message?.type === 'DCA_SUBMIT_PACKET') {
    void submit(message.packet as ContentPacket).then(sendResponse);
    return true;
  }
  if (message?.type === 'DCA_AUTO_PACKET') {
    void submitAuto(message.packet as ContentPacket).then(sendResponse);
    return true;
  }
  if (message?.type === 'DCA_ACTIVE_PAGE') {
    void postActivePage(message.page as ActivePage, sender).then(() => sendResponse({ ok: true }));
    return true;
  }
  if (message?.type === 'DCA_CHECK_HEALTH') {
    void health().then(sendResponse);
    return true;
  }
  if (message?.type === 'DCA_RETRY_QUEUE') {
    void retryQueue().then(() => sendResponse({ ok: true }));
    return true;
  }
  if (message?.type === 'DCA_EFFECTIVE_SETTINGS') {
    void refreshRemotePolicy().then(() => loadSettings()).then(sendResponse);
    return true;
  }
  if (message?.type === 'DCA_PUSH_POLICY') {
    void (async () => {
      const local = await loadLocalSettings();
      await saveSettings({ ...local, ...message.policy });
      const policy = await pushPolicy(message.policy ?? {});
      sendResponse({ ok: true, policy });
    })();
    return true;
  }
});
