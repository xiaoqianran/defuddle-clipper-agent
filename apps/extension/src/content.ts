import Defuddle from 'defuddle';
import { createMarkdownContent } from 'defuddle/full';
import { ContentPacket, PROTOCOL_VERSION } from './protocol';
import { loadSettings } from './settings';

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

async function extractPacket(): Promise<ContentPacket> {
  const settings = await loadSettings();
  const parser = new Defuddle(document, { url: location.href });

  let result;
  try {
    result = await parser.parseAsync();
  } catch {
    result = parser.parse();
  }

  if (!result?.content) {
    throw new Error('Defuddle returned no content for this page.');
  }

  const markdown = createMarkdownContent(result.content, location.href);
  if (!markdown.trim()) {
    throw new Error('Extracted Markdown is empty.');
  }

  const selectionHtml = selectedHtml();
  const selectionMarkdown = selectionHtml
    ? createMarkdownContent(selectionHtml, location.href)
    : undefined;

  const variables = (result.variables ?? {}) as Record<string, unknown>;
  const transcript = stringVariable(variables.transcript);

  return {
    protocolVersion: PROTOCOL_VERSION,
    captureId: crypto.randomUUID(),
    capturedAt: new Date().toISOString(),
    source: {
      url: location.href,
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
      metaTags: (result.metaTags ?? []) as Record<string, unknown>[],
      variables
    },
    ...(transcript ? { media: { transcript } } : {})
  };
}

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type !== 'DCA_EXTRACT_PAGE') return;

  void extractPacket()
    .then(packet => sendResponse({ ok: true, packet }))
    .catch(error => {
      sendResponse({
        ok: false,
        error: error instanceof Error ? error.message : String(error)
      });
    });

  return true;
});
