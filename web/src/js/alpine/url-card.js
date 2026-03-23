import { getCookie, copyToClipboard } from '../utils.js';

/**
 * Alpine.js component for individual URL cards.
 * @param {object} data - { id, link, shortCode }
 */
export function urlCard(data) {
  return {
    id: data.id,
    link: data.link,
    shortCode: data.shortCode,
    linkCopied: false,
    shortCopied: false,

    trackVisit() {
      const token = getCookie('linkstash_token');
      const url = '/api/urls/' + this.id + '/visit';
      fetch(url, {
        method: 'POST',
        headers: { 'Authorization': 'Bearer ' + token },
        keepalive: true,
      }).catch(() => {});
    },

    copyLink() {
      copyToClipboard(this.link);
      this.linkCopied = true;
      setTimeout(() => this.linkCopied = false, 2000);
    },

    copyShortLink() {
      copyToClipboard(window.location.origin + '/s/' + this.shortCode);
      this.shortCopied = true;
      setTimeout(() => this.shortCopied = false, 2000);
    },

    async deleteURL() {
      if (!confirm('$ rm url/' + this.id + ' -- are you sure?')) return;
      try {
        const token = getCookie('linkstash_token');
        const resp = await fetch('/api/urls/' + this.id, {
          method: 'DELETE',
          headers: { 'Authorization': 'Bearer ' + token },
        });
        if (resp.ok) window.location.reload();
      } catch (e) {
        // silently fail
      }
    },
  };
}
