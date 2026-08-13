import Defuddle from 'defuddle';
import { createMarkdownContent } from 'defuddle/full';
import { canonicalizeUrl, CaptureTrigger, contentFingerprint, shouldAutoCapture } from './capture-policy';
import { waitForDOMStability } from './dom-stability';
import { ContentPacket, PROTOCOL_VERSION } from './protocol';
import { loadSettings } from './settings';

let captureGeneration = 0;
let scheduleTimer: number | undefined;
let lastActiveSignature = '';

function selectedHtml(): string | undefined {
  const selection = window.getSelection();
  if (!selection || selection.isCollapsed || selection.rangeCount === 0) return undefined;
  const container = document.createElement('div');
  container.appendChild(selection.getRangeAt(0).cloneContents());
  return container.innerHTML || undefined;
}

function stringVariable(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

async function extractPacket(trigger: CaptureTrigger): Promise<ContentPacket> {
  const settings = await loadSettings();
  const sourceUrl = location.href;
  const parser = new Defuddle(document, { url: sourceUrl });

  let result;
  try {
    result = await parser.parseAsync();
  } catch {
    result = parser.parse();
  }

  if (!result?.content) throw new Error('Defuddle returned no content for this page.');
  const markdown = createMarkdownContent(result.content, sourceUrl);
  if (!markdown.trim()) throw new Error('Extracted Markdown is empty.');

  const canonicalUrl = canonicalizeUrl(sourceUrl);
  const fingerprint = contentFingerprint(canonicalUrl, markdown);
  const selectionHtml = trigger === 'manual' ? selectedHtml() : undefined;
  const selectionMarkdown = selectionHtml ? createMarkdownContent(selectionHtml, sourceUrl) : undefined;
  const variables = (result.variables ?? {}) as Record<string, unknown>;
  const transcript = stringVariable(variables.transcript);

  return {
    protocolVersion: PROTOCOL_VERSION,
    captureId: crypto.randomUUID(),
    capturedAt: new Date().toISOString(),
    source: {
      url: sourceUrl,
      title: result.title || document.title || 'Untitled',
      site: result.site || location.hostname,
      author: result.author || undefined,
      published: result.published || undefined,
      language: result.language || document.documentElement.lang || undefined,
      description: result.description || undefined
    },
    content: {
      markdown,
      ...(settings.includeHtml ? { html: result.content } : {})
    },
    ...(selectionHtml || selectionMarkdown
      ? { selection: { html: selectionHtml, markdown: selectionMarkdown } }
      : {}),
    metadata: {
      wordCount: result.wordCount || undefined,
      image: result.image || undefined,
      favicon: result.favicon || undefined,
      schemaOrg: result.schemaOrgData,
      metaTags: result.metaTags ?? [],
      variables,
      capture: { trigger, canonicalUrl, contentFingerprint: fingerprint }
    },
    ...(transcript ? { media: { transcript } } : {})
  };
}

async function runAutoCapture(generation: number, trigger: CaptureTrigger): Promise<void> {
  const settings = await loadSettings();
  if (generation !== captureGeneration || !shouldAutoCapture(location.href, settings)) return;

  await waitForDOMStability();
  if (generation !== captureGeneration) return;

  const expectedUrl = location.href;
  try {
    const packet = await extractPacket(trigger);
    if (generation !== captureGeneration || location.href !== expectedUrl) return;
    await chrome.runtime.sendMessage({ type: 'DCA_AUTO_PACKET', packet });
  } catch (error) {
    console.debug('[DCA] automatic capture skipped:', error instanceof Error ? error.message : String(error));
  }
}

function scheduleAutoCapture(trigger: CaptureTrigger): void {
  const generation = ++captureGeneration;
  if (scheduleTimer !== undefined) window.clearTimeout(scheduleTimer);

  void loadSettings().then(settings => {
    if (generation !== captureGeneration || !shouldAutoCapture(location.href, settings)) return;
    scheduleTimer = window.setTimeout(() => void runAutoCapture(generation, trigger), settings.captureDelayMs);
  });
}

async function reportActivePage(): Promise<void> {
  if (document.visibilityState !== 'visible') return;
  const settings = await loadSettings();
  if (!settings.followBrowser || !shouldAutoCapture(location.href, { ...settings, autoCapture: true })) return;

  const signature = `${location.href}\n${document.title}`;
  if (signature === lastActiveSignature) return;
  lastActiveSignature = signature;

  await chrome.runtime.sendMessage({
    type: 'DCA_ACTIVE_PAGE',
    page: {
      url: location.href,
      title: document.title || location.hostname,
      observedAt: new Date().toISOString()
    }
  });
}

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === 'DCA_EXTRACT_PAGE') {
    void extractPacket('manual')
      .then(packet => sendResponse({ ok: true, packet }))
      .catch(error => sendResponse({ ok: false, error: error instanceof Error ? error.message : String(error) }));
    return true;
  }

  if (message?.type === 'DCA_NAVIGATION') {
    scheduleAutoCapture('navigation');
    if (document.visibilityState === 'visible') void reportActivePage();
    sendResponse({ ok: true });
    return;
  }

  if (message?.type === 'DCA_TAB_ACTIVE') {
    lastActiveSignature = '';
    void reportActivePage().then(() => sendResponse({ ok: true }));
    return true;
  }
});

document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') {
    lastActiveSignature = '';
    void reportActivePage();
  }
});

window.addEventListener('pageshow', event => {
  if (event.persisted) scheduleAutoCapture('bfcache');
  if (document.visibilityState === 'visible') {
    lastActiveSignature = '';
    void reportActivePage();
  }
});

scheduleAutoCapture('initial');
if (document.visibilityState === 'visible') void reportActivePage();
