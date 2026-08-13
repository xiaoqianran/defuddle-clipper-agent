import { ExtensionSettings, isDomainAllowed } from './settings';

export type CaptureTrigger = 'initial' | 'navigation' | 'bfcache' | 'manual';

export function canonicalizeUrl(rawUrl: string): string {
  const url = new URL(rawUrl);
  url.hostname = url.hostname.toLowerCase();

  const trackingNames = new Set(['fbclid', 'gclid', 'dclid', 'mc_cid', 'mc_eid', 'ref_src']);
  for (const key of [...url.searchParams.keys()]) {
    const lower = key.toLowerCase();
    if (lower.startsWith('utm_') || trackingNames.has(lower)) url.searchParams.delete(key);
  }
  url.searchParams.sort();

  if (url.hash && !/^#(?:!?\/|\?)/.test(url.hash)) url.hash = '';
  return url.toString();
}

export function contentFingerprint(canonicalUrl: string, markdown: string): string {
  const value = `${canonicalUrl}\n${markdown.replace(/\s+/g, ' ').trim()}`;
  let hash = 0xcbf29ce484222325n;
  const prime = 0x100000001b3n;
  const mask = 0xffffffffffffffffn;
  for (let i = 0; i < value.length; i += 1) {
    hash ^= BigInt(value.charCodeAt(i));
    hash = (hash * prime) & mask;
  }
  return hash.toString(16).padStart(16, '0');
}

export function shouldAutoCapture(url: string, settings: ExtensionSettings): boolean {
  return settings.autoCapture && isDomainAllowed(url, settings);
}
