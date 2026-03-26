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
