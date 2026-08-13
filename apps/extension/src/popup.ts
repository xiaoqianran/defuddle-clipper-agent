import { ContentPacket } from './protocol';
import { loadSettings, saveSettings } from './settings';

const statusEl = document.getElementById('status') as HTMLDivElement;
const modeEl = document.getElementById('mode') as HTMLDivElement;
const toggleButton = document.getElementById('toggle') as HTMLButtonElement;
const captureButton = document.getElementById('capture') as HTMLButtonElement;
const healthButton = document.getElementById('health') as HTMLButtonElement;
const optionsButton = document.getElementById('options') as HTMLButtonElement;

function status(text: string): void {
  statusEl.textContent = text;
}

async function refreshMode(): Promise<void> {
  const settings = await loadSettings();
  modeEl.textContent = settings.autoCapture ? 'Auto Capture: ON' : 'Auto Capture: PAUSED';
  toggleButton.textContent = settings.autoCapture ? 'Pause auto capture' : 'Resume auto capture';
}

async function activeTab(): Promise<chrome.tabs.Tab> {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id) throw new Error('No active tab.');
  return tab;
}

toggleButton.addEventListener('click', () => {
  void (async () => {
    const settings = await loadSettings();
    await saveSettings({ ...settings, autoCapture: !settings.autoCapture });
    await refreshMode();
    status(settings.autoCapture ? 'Automatic capture paused.' : 'Automatic capture resumed.');
  })();
});

captureButton.addEventListener('click', () => {
  void (async () => {
    captureButton.disabled = true;
    status('Manual fallback capture…');
    try {
      const tab = await activeTab();
      const extracted = await chrome.tabs.sendMessage(tab.id!, { type: 'DCA_EXTRACT_PAGE' }) as
        { ok: boolean; packet?: ContentPacket; error?: string };
      if (!extracted?.ok || !extracted.packet) throw new Error(extracted?.error || 'Page extraction failed.');

      const submit = await chrome.runtime.sendMessage({ type: 'DCA_SUBMIT_PACKET', packet: extracted.packet });
      if (submit?.queued) {
        status(`Queued locally. Agent unavailable: ${submit.error ?? 'unknown error'}`);
      } else {
        status(`Saved: ${submit?.result?.captureId ?? extracted.packet.captureId}`);
      }
    } catch (error) {
      status(error instanceof Error ? error.message : String(error));
    } finally {
      captureButton.disabled = false;
    }
  })();
});

healthButton.addEventListener('click', () => {
  void chrome.runtime.sendMessage({ type: 'DCA_CHECK_HEALTH' }).then(result => {
    status(result?.ok ? 'Local bridge is reachable.' : `Bridge unavailable: ${result?.error ?? 'unknown error'}`);
  });
});

optionsButton.addEventListener('click', () => void chrome.runtime.openOptionsPage());
void refreshMode();
