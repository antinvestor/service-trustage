export type ConsoleSettings = {
  apiBaseUrl: string;
  authToken: string;
};

const SETTINGS_KEY = 'trustage.console.settings';
const TOKEN_KEY = 'trustage.console.session.token';

export function loadSettings(): ConsoleSettings {
  if (typeof window === 'undefined') {
    return { apiBaseUrl: '', authToken: '' };
  }

  const raw = window.localStorage.getItem(SETTINGS_KEY);
  const authToken = window.sessionStorage.getItem(TOKEN_KEY) || '';
  if (!raw) {
    return { apiBaseUrl: '', authToken };
  }

  try {
    const data = JSON.parse(raw) as ConsoleSettings;
    return {
      apiBaseUrl: data.apiBaseUrl || '',
      authToken,
    };
  } catch {
    return { apiBaseUrl: '', authToken };
  }
}

export function saveSettings(settings: ConsoleSettings) {
  if (typeof window === 'undefined') {
    return;
  }

  window.localStorage.setItem(
    SETTINGS_KEY,
    JSON.stringify({
      apiBaseUrl: settings.apiBaseUrl,
    }),
  );

  if (settings.authToken) {
    window.sessionStorage.setItem(TOKEN_KEY, settings.authToken);
  } else {
    window.sessionStorage.removeItem(TOKEN_KEY);
  }
}
