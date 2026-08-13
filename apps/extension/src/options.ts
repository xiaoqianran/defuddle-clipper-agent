import { loadSettings, saveSettings } from './settings';

const agentUrl = document.getElementById('agentUrl') as HTMLInputElement;
const authToken = document.getElementById('authToken') as HTMLInputElement;
const includeHtml = document.getElementById('includeHtml') as HTMLInputElement;
const save = document.getElementById('save') as HTMLButtonElement;
const status = document.getElementById('status') as HTMLDivElement;

void loadSettings().then(settings => {
  agentUrl.value = settings.agentUrl;
  authToken.value = settings.authToken;
  includeHtml.checked = settings.includeHtml;
});

save.addEventListener('click', () => {
  void (async () => {
    const url = agentUrl.value.trim().replace(/\/+$/, '');
    if (!/^https?:\/\/(127\.0\.0\.1|localhost)(:\d+)?$/i.test(url)) {
      status.textContent = 'P0 only accepts localhost/127.0.0.1 agent URLs.';
      return;
    }

    await saveSettings({
      agentUrl: url,
      authToken: authToken.value,
      includeHtml: includeHtml.checked
    });
    status.textContent = 'Saved.';
  })();
});
