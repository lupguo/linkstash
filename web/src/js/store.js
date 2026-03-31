import { signal, computed } from '@preact/signals';
import { getCookie, isTokenExpired } from './utils.js';

// On startup: read token from cookie, clear it if expired/invalid
function getValidToken() {
  const token = getCookie('linkstash_token');
  if (token && !isTokenExpired(token)) {
    return token;
  }
  // Clear stale cookie
  if (token) {
    document.cookie = 'linkstash_token=;path=/;max-age=0';
  }
  return null;
}

export const auth = signal({ token: getValidToken() });
export const isAuthenticated = computed(() => !!auth.value.token);
