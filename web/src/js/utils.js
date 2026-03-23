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

/**
 * Make an authenticated API request.
 * @param {string} url
 * @param {string} method
 * @param {object} [body]
 * @returns {Promise<Response>}
 */
export function apiRequest(url, method, body) {
  const token = getCookie('linkstash_token');
  const headers = { 'Authorization': 'Bearer ' + token };
  const opts = { method, headers };
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  return fetch(url, opts);
}

/**
 * Read page data from embedded JSON script tag.
 * @returns {object}
 */
export function getPageData() {
  const el = document.getElementById('page-data');
  if (!el) return {};
  try {
    return JSON.parse(el.textContent);
  } catch (e) {
    console.error('Failed to parse page data:', e);
    return {};
  }
}
