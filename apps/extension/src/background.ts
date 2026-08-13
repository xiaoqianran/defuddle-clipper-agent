import { ContentPacket, SubmitResult } from './protocol';
import { loadSettings } from './settings';

const QUEUE_KEY = 'dca.pendingCaptures';
const RETRY_ALARM = 'dca.retry';
const MAX_QUEUE = 100;

interface PendingCapture {
  packet: ContentPacket;
  attempts: number;
  lastError?: string;
  queuedAt: string;
}

async function readQueue(): Promise<PendingCapture[]> {
  const data = await chrome.storage.local.get(QUEUE_KEY);
  return Array.isArray(data[QUEUE_KEY]) ? data[QUEUE_KEY] : [];
}

async function writeQueue(queue: PendingCapture[]): Promise<void> {
  await chrome.storage.local.set({ [QUEUE_KEY]: queue.slice(-MAX_QUEUE) });
}

async function enqueue(packet: ContentPacket, error: unknown): Promise<void> {
  const queue = await readQueue();
  const existing = queue.find(item => item.packet.captureId === packet.captureId);
  const message = error instanceof Error ? error.message : String(error);

  if (existing) {
    existing.attempts += 1;
    existing.lastError = message;
  } else {
    queue.push({
      packet,
      attempts: 1,
      lastError: message,
      queuedAt: new Date().toISOString()
    });
  }
  await writeQueue(queue);
}

async function postPacket(packet: ContentPacket): Promise<SubmitResult> {
  const settings = await loadSettings();
  const url = `${settings.agentUrl.replace(/\/+$/, '')}/v1/captures`;

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 20_000);

  try {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-DCA-Protocol': packet.protocolVersion
    };
    if (settings.authToken) {
      headers.Authorization = `Bearer ${settings.authToken}`;
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify(packet),
      signal: controller.signal
    });

    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(body?.error || `Local agent returned HTTP ${response.status}`);
    }
    return body as SubmitResult;
  } finally {
    clearTimeout(timeout);
  }
}

async function submit(packet: ContentPacket): Promise<{ queued: boolean; result?: SubmitResult; error?: string }> {
  try {
    const result = await postPacket(packet);
    return { queued: false, result };
  } catch (error) {
    await enqueue(packet, error);
    return {
      queued: true,
      error: error instanceof Error ? error.message : String(error)
    };
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

async function health(): Promise<{ ok: boolean; error?: string }> {
  const settings = await loadSettings();
  try {
    const response = await fetch(`${settings.agentUrl.replace(/\/+$/, '')}/health`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return { ok: true };
  } catch (error) {
    return { ok: false, error: error instanceof Error ? error.message : String(error) };
  }
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
  }
});

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === 'DCA_SUBMIT_PACKET') {
    void submit(message.packet as ContentPacket).then(sendResponse);
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
});
