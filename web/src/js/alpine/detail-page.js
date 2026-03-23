import { getCookie, copyToClipboard, apiRequest, getPageData } from '../utils.js';

/**
 * Alpine.js component for the URL detail page.
 */
export function detailPage() {
  const data = getPageData();
  const isNew = data.IsNew;
  const url = data.URL || {};
  const autoEdit = !isNew && new URLSearchParams(window.location.search).has('edit');

  return {
    isNew: isNew,
    editing: autoEdit,
    saving: false,
    message: '',
    messageType: '',
    enableShortCode: isNew ? false : !!url.ShortCode,
    form: {
      link: '',
      title: isNew ? '' : (url.Title || ''),
      description: isNew ? '' : (url.Description || ''),
      keywords: isNew ? '' : (url.Keywords || ''),
      category: isNew ? '' : (url.Category || ''),
      tags: isNew ? '' : (url.Tags || ''),
      manual_weight: isNew ? 0 : (url.ManualWeight || 0),
      visit_count: isNew ? 0 : (url.VisitCount || 0),
      short_code: isNew ? '' : (url.ShortCode || ''),
      ttl: '',
      color: isNew ? '' : (url.Color || ''),
      icon: isNew ? '' : (url.Icon || ''),
      status: isNew ? 'pending' : (url.Status || 'pending'),
    },

    startEdit() {
      this.editing = true;
      this.message = '';
    },

    async clearFavicon() {
      try {
        const resp = await apiRequest('/api/urls/' + url.ID, 'PUT', { favicon: '' });
        if (resp.ok) {
          this.message = 'favicon cleared';
          this.messageType = 'success';
          setTimeout(() => window.location.reload(), 500);
        } else {
          this.message = 'failed to clear favicon';
          this.messageType = 'error';
        }
      } catch (e) {
        this.message = 'network error';
        this.messageType = 'error';
      }
    },

    copyText(text) {
      copyToClipboard(text);
      this.message = 'copied to clipboard';
      this.messageType = 'success';
      setTimeout(() => this.message = '', 1500);
    },

    async createURL() {
      if (!this.form.link.trim()) {
        this.message = 'link is required';
        this.messageType = 'error';
        return;
      }
      this.saving = true;
      this.message = '';
      try {
        // Step 1: Create URL
        const resp = await apiRequest('/api/urls', 'POST', { link: this.form.link.trim() });
        if (!resp.ok) {
          const d = await resp.json();
          this.message = (d.error && d.error.message) || 'failed to create url';
          this.messageType = 'error';
          this.saving = false;
          return;
        }
        const created = await resp.json();
        const urlId = created.ID;

        // Step 2: Update extra fields if any are filled
        const updates = {};
        if (this.form.title.trim()) updates.title = this.form.title.trim();
        if (this.form.description.trim()) updates.description = this.form.description.trim();
        if (this.form.keywords.trim()) updates.keywords = this.form.keywords.trim();
        if (this.form.category.trim()) updates.category = this.form.category.trim();
        if (this.form.tags.trim()) updates.tags = this.form.tags.trim();
        if (this.form.manual_weight) updates.manual_weight = this.form.manual_weight;
        if (this.form.color.trim()) updates.color = this.form.color.trim();
        if (this.form.icon.trim()) updates.icon = this.form.icon.trim();

        if (Object.keys(updates).length > 0) {
          await apiRequest('/api/urls/' + urlId, 'PUT', updates);
        }

        // Step 3: Create short link if enabled
        if (this.enableShortCode && (this.form.short_code.trim() || this.form.ttl)) {
          const shortBody = { long_url: this.form.link.trim() };
          if (this.form.short_code.trim()) shortBody.code = this.form.short_code.trim();
          if (this.form.ttl) shortBody.ttl = this.form.ttl;
          const shortResp = await apiRequest('/api/short-links', 'POST', shortBody);
          if (!shortResp.ok) {
            const d = await shortResp.json();
            this.message = 'URL created, but short link failed: ' + ((d.error && d.error.message) || 'unknown');
            this.messageType = 'error';
            setTimeout(() => window.location.href = '/urls/' + urlId, 1500);
            this.saving = false;
            return;
          }
        }

        window.location.href = '/urls/' + urlId;
      } catch (e) {
        this.message = 'network error';
        this.messageType = 'error';
      }
      this.saving = false;
    },

    async saveEdit() {
      this.saving = true;
      this.message = '';
      try {
        const updates = {
          title: this.form.title,
          description: this.form.description,
          keywords: this.form.keywords,
          category: this.form.category,
          tags: this.form.tags,
          manual_weight: this.form.manual_weight,
          visit_count: this.form.visit_count,
          color: this.form.color,
          icon: this.form.icon,
          status: this.form.status,
        };
        if (this.enableShortCode) {
          updates.short_code = this.form.short_code;
          updates.ttl = this.form.ttl;
        } else {
          updates.short_code = '';
          updates.ttl = '';
        }
        const resp = await apiRequest('/api/urls/' + url.ID, 'PUT', updates);
        if (!resp.ok) {
          const d = await resp.json();
          this.message = (d.error && d.error.message) || 'update failed';
          this.messageType = 'error';
          this.saving = false;
          return;
        }

        // Update short link if needed
        const origShortCode = url.ShortCode || '';
        if (this.enableShortCode && this.form.short_code.trim() !== origShortCode) {
          if (this.form.short_code.trim() && !origShortCode) {
            const shortBody = { long_url: url.Link };
            shortBody.code = this.form.short_code.trim();
            if (this.form.ttl) shortBody.ttl = this.form.ttl;
            await apiRequest('/api/short-links', 'POST', shortBody);
          }
        }

        this.message = 'updated successfully';
        this.messageType = 'success';
        this.editing = false;
        setTimeout(() => window.location.reload(), 500);
      } catch (e) {
        this.message = 'network error';
        this.messageType = 'error';
      }
      this.saving = false;
    },

    confirmDelete() {
      if (confirm('$ rm -rf /urls/' + url.ID + ' -- are you sure?')) {
        this.doDelete();
      }
    },

    async doDelete() {
      try {
        const resp = await apiRequest('/api/urls/' + url.ID, 'DELETE');
        if (resp.ok) {
          window.location.href = '/';
        } else {
          this.message = 'delete failed';
          this.messageType = 'error';
        }
      } catch (e) {
        this.message = 'network error';
        this.messageType = 'error';
      }
    },
  };
}
