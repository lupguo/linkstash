/**
 * Shared utility functions for LinkStash
 */

/**
 * Get a cookie value by name.
 * @param {string} name
 * @returns {string}
 */
export function getCookie(name) {
  const v = document.cookie.match('(^|;)\\s*' + name + '\\s*=\\s*([^;]+)');
  return v ? v.pop() : '';
}

/**
 * Check if a JWT token is expired (or invalid).
 * Returns true if expired/invalid, false if still valid.
 * @param {string} token
 * @returns {boolean}
 */
export function isTokenExpired(token) {
  if (!token) return true;
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    // exp is in seconds, Date.now() in ms
    return !payload.exp || payload.exp * 1000 < Date.now();
  } catch {
    return true; // malformed token
  }
}

/**
 * Copy text to clipboard with fallback.
 * @param {string} text
 */
export function copyToClipboard(text) {
  navigator.clipboard.writeText(text).catch(() => {
    const ta = document.createElement('textarea');
    ta.value = text;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
  });
}
