// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
