import { ContentPacket } from './protocol';

const statusEl = document.getElementById('status') as HTMLDivElement;
const captureButton = document.getElementById('capture') as HTMLButtonElement;
const healthButton = document.getElementById('health') as HTMLButtonElement;
const optionsButton = document.getElementById('options') as HTMLButtonElement;

function status(text: string): void {
  statusEl.textContent = text;
}

async function activeTab(): Promise<chrome.tabs.Tab> {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id) throw new Error('No active tab.');
  return tab;
}

captureButton.addEventListener('click', () => {
  void (async () => {
    captureButton.disabled = true;
    status('Extracting page with Defuddle…');
    try {
      const tab = await activeTab();
      const extracted = await chrome.tabs.sendMessage(tab.id!, { type: 'DCA_EXTRACT_PAGE' }) as
        { ok: boolean; packet?: ContentPacket; error?: string };

      if (!extracted?.ok || !extracted.packet) {
        throw new Error(extracted?.error || 'Page extraction failed.');
      }

      status('Sending to local agent…');
      const submit = await chrome.runtime.sendMessage({
        type: 'DCA_SUBMIT_PACKET',
        packet: extracted.packet
      });

      if (submit?.queued) {
        status(`Captured locally in extension queue.\nAgent unavailable: ${submit.error ?? 'unknown error'}`);
      } else {
        const result = submit?.result;
        status(
          `Saved: ${result?.captureId ?? extracted.packet.captureId}\n` +
          `AI: ${result?.aiStatus ?? 'unknown'}${result?.duplicate ? '\n(idempotent retry)' : ''}`
        );
      }
    } catch (error) {
      status(error instanceof Error ? error.message : String(error));
    } finally {
      captureButton.disabled = false;
    }
  })();
});

healthButton.addEventListener('click', () => {
  void (async () => {
    status('Checking local agent…');
    const result = await chrome.runtime.sendMessage({ type: 'DCA_CHECK_HEALTH' });
    status(result?.ok ? 'Local agent is reachable.' : `Agent unavailable: ${result?.error ?? 'unknown error'}`);
  })();
});

optionsButton.addEventListener('click', () => {
  void chrome.runtime.openOptionsPage();
});
