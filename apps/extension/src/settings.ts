export interface ExtensionSettings {
  agentUrl: string;
  authToken: string;
  includeHtml: boolean;
  autoCapture: boolean;
  followBrowser: boolean;
  captureDelayMs: number;
  domainAllowlist: string[];
  domainDenylist: string[];
}

export const DEFAULT_SETTINGS: ExtensionSettings = {
  agentUrl: 'http://127.0.0.1:27123',
  authToken: '',
  includeHtml: false,
  autoCapture: true,
  followBrowser: true,
  captureDelayMs: 1200,
  domainAllowlist: [],
  domainDenylist: []
};

const KEY = 'dca.settings';
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

export async function loadSettings(): Promise<ExtensionSettings> {
  const result = await chrome.storage.sync.get(KEY);
  const stored = (result[KEY] ?? {}) as Partial<ExtensionSettings>;
  return {
    ...DEFAULT_SETTINGS,
    ...stored,
    captureDelayMs: clampCaptureDelay(stored.captureDelayMs ?? DEFAULT_SETTINGS.captureDelayMs),
    domainAllowlist: normalizeDomainList(stored.domainAllowlist ?? []),
    domainDenylist: normalizeDomainList(stored.domainDenylist ?? [])
  };
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
