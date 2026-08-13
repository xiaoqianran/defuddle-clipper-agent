export interface CapturePolicy {
  revision?: number;
  autoCapture: boolean;
  archiveAll: boolean;
  captureDelayMs: number;
  domainAllowlist: string[];
  domainDenylist: string[];
}

export interface ExtensionSettings extends CapturePolicy {
  agentUrl: string;
  authToken: string;
  includeHtml: boolean;
  followBrowser: boolean;
}

export const DEFAULT_SETTINGS: ExtensionSettings = {
  agentUrl: 'http://127.0.0.1:27123',
  authToken: '',
  includeHtml: false,
  autoCapture: true,
  archiveAll: true,
  followBrowser: true,
  captureDelayMs: 1200,
  domainAllowlist: [],
  domainDenylist: []
};

const KEY = 'dca.settings';
const REMOTE_POLICY_KEY = 'dca.remotePolicy';
const MIN_CAPTURE_DELAY_MS = 250;
const MAX_CAPTURE_DELAY_MS = 30_000;

function normalizeDomain(domain: string): string {
  return domain.trim().toLowerCase().replace(/^\.+|\.+$/g, '');
}

export function normalizeDomainList(values: string[]): string[] {
  return [...new Set(values.map(normalizeDomain).filter(Boolean))];
}

export function clampCaptureDelay(value: number): number {
  if (!Number.isFinite(value)) return DEFAULT_SETTINGS.captureDelayMs;
  return Math.min(MAX_CAPTURE_DELAY_MS, Math.max(MIN_CAPTURE_DELAY_MS, Math.round(value)));
}

export function isSupportedPageUrl(rawUrl: string): boolean {
  try {
    const url = new URL(rawUrl);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

function matchesDomain(hostname: string, domain: string): boolean {
  return hostname === domain || hostname.endsWith(`.${domain}`);
}

export function isDomainAllowed(rawUrl: string, settings: ExtensionSettings): boolean {
  if (!isSupportedPageUrl(rawUrl)) return false;

  const hostname = new URL(rawUrl).hostname.toLowerCase();
  const allowlist = normalizeDomainList(settings.domainAllowlist);
  const denylist = normalizeDomainList(settings.domainDenylist);

  if (denylist.some(domain => matchesDomain(hostname, domain))) return false;
  if (allowlist.length > 0 && !allowlist.some(domain => matchesDomain(hostname, domain))) return false;
  return true;
}

function fromStored(stored: Partial<ExtensionSettings>): ExtensionSettings {
  return {
    ...DEFAULT_SETTINGS,
    ...stored,
    archiveAll: stored.archiveAll ?? DEFAULT_SETTINGS.archiveAll,
    captureDelayMs: clampCaptureDelay(stored.captureDelayMs ?? DEFAULT_SETTINGS.captureDelayMs),
    domainAllowlist: normalizeDomainList(stored.domainAllowlist ?? []),
    domainDenylist: normalizeDomainList(stored.domainDenylist ?? [])
  };
}

function applyPolicy(base: ExtensionSettings, policy: Partial<CapturePolicy> | undefined): ExtensionSettings {
  if (!policy) return base;
  return {
    ...base,
    autoCapture: policy.autoCapture ?? base.autoCapture,
    archiveAll: policy.archiveAll ?? base.archiveAll,
    captureDelayMs: clampCaptureDelay(policy.captureDelayMs ?? base.captureDelayMs),
    domainAllowlist: normalizeDomainList(policy.domainAllowlist ?? base.domainAllowlist),
    domainDenylist: normalizeDomainList(policy.domainDenylist ?? base.domainDenylist)
  };
}

export async function loadLocalSettings(): Promise<ExtensionSettings> {
  const result = await chrome.storage.sync.get(KEY);
  return fromStored((result[KEY] ?? {}) as Partial<ExtensionSettings>);
}

export async function cacheRemotePolicy(policy: CapturePolicy): Promise<void> {
  await chrome.storage.local.set({ [REMOTE_POLICY_KEY]: policy });
}

export async function loadSettings(): Promise<ExtensionSettings> {
  const local = await loadLocalSettings();
  const remote = await chrome.storage.local.get(REMOTE_POLICY_KEY);
  return applyPolicy(local, remote[REMOTE_POLICY_KEY] as Partial<CapturePolicy> | undefined);
}

export async function saveSettings(settings: ExtensionSettings): Promise<void> {
  await chrome.storage.sync.set({
    [KEY]: {
      ...settings,
      captureDelayMs: clampCaptureDelay(settings.captureDelayMs),
      domainAllowlist: normalizeDomainList(settings.domainAllowlist),
      domainDenylist: normalizeDomainList(settings.domainDenylist)
    }
  });
}
