export interface ExtensionSettings {
  agentUrl: string;
  authToken: string;
  includeHtml: boolean;
}

export const DEFAULT_SETTINGS: ExtensionSettings = {
  agentUrl: 'http://127.0.0.1:27123',
  authToken: '',
  includeHtml: false
};

const KEY = 'dca.settings';

export async function loadSettings(): Promise<ExtensionSettings> {
  const result = await chrome.storage.sync.get(KEY);
  return {
    ...DEFAULT_SETTINGS,
    ...(result[KEY] ?? {})
  };
}

export async function saveSettings(settings: ExtensionSettings): Promise<void> {
  await chrome.storage.sync.set({ [KEY]: settings });
}
