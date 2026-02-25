export type ConsoleSettings = {
  apiBaseUrl: string;
  authToken: string;
};

const SETTINGS_KEY = 'trustage.console.settings';

export function loadSettings(): ConsoleSettings {
  if (typeof window === 'undefined') {
    return { apiBaseUrl: '', authToken: '' };
  }

  const raw = window.localStorage.getItem(SETTINGS_KEY);
  if (!raw) {
    return { apiBaseUrl: '', authToken: '' };
  }

  try {
    const data = JSON.parse(raw) as ConsoleSettings;
    return {
      apiBaseUrl: data.apiBaseUrl || '',
      authToken: data.authToken || '',
    };
  } catch {
    return { apiBaseUrl: '', authToken: '' };
  }
}

export function saveSettings(settings: ConsoleSettings) {
  if (typeof window === 'undefined') {
    return;
  }

  window.localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
}
