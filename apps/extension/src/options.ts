import { clampCaptureDelay, loadSettings, normalizeDomainList, saveSettings } from './settings';

const agentUrl = document.getElementById('agentUrl') as HTMLInputElement;
const authToken = document.getElementById('authToken') as HTMLInputElement;
const includeHtml = document.getElementById('includeHtml') as HTMLInputElement;
const autoCapture = document.getElementById('autoCapture') as HTMLInputElement;
const followBrowser = document.getElementById('followBrowser') as HTMLInputElement;
const captureDelayMs = document.getElementById('captureDelayMs') as HTMLInputElement;
const domainAllowlist = document.getElementById('domainAllowlist') as HTMLTextAreaElement;
const domainDenylist = document.getElementById('domainDenylist') as HTMLTextAreaElement;
const save = document.getElementById('save') as HTMLButtonElement;
const status = document.getElementById('status') as HTMLDivElement;

function parseDomains(value: string): string[] {
  return normalizeDomainList(value.split(/[\n,]+/));
}

void loadSettings().then(settings => {
  agentUrl.value = settings.agentUrl;
  authToken.value = settings.authToken;
  includeHtml.checked = settings.includeHtml;
  autoCapture.checked = settings.autoCapture;
  followBrowser.checked = settings.followBrowser;
  captureDelayMs.value = String(settings.captureDelayMs);
  domainAllowlist.value = settings.domainAllowlist.join('\n');
  domainDenylist.value = settings.domainDenylist.join('\n');
});

save.addEventListener('click', () => {
  void (async () => {
    const url = agentUrl.value.trim().replace(/\/+$/, '');
    if (!/^https?:\/\/(127\.0\.0\.1|localhost)(:\d+)?$/i.test(url)) {
      status.textContent = 'Local bridge URL must use localhost or 127.0.0.1.';
      return;
    }

    await saveSettings({
      agentUrl: url,
      authToken: authToken.value,
      includeHtml: includeHtml.checked,
      autoCapture: autoCapture.checked,
      followBrowser: followBrowser.checked,
      captureDelayMs: clampCaptureDelay(Number(captureDelayMs.value)),
      domainAllowlist: parseDomains(domainAllowlist.value),
      domainDenylist: parseDomains(domainDenylist.value)
    });
    status.textContent = 'Saved. New navigation will use these settings.';
  })();
});
