import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { route } from 'preact-router';
import { isAuthenticated } from '../store.js';
import { urlApi, shortApi, configApi } from '../api.js';
import { ColorPicker } from '../components/ColorPicker.jsx';
import { ConfirmModal } from '../components/ConfirmModal.jsx';

const EMPTY_FORM = {
  link: '',
  title: '',
  description: '',
  keywords: '',
  category: '',
  tags: '',
  manual_weight: 0,
  visit_count: 0,
  short_code: '',
  ttl: 0,
  color: '',
  icon: '',
  status: '',
};

export function DetailPage({ id }) {
  const isNew = !id || id === 'new';
  const [urlData, setUrlData] = useState(null);
  const [editing, setEditing] = useState(isNew);
  const [saving, setSaving] = useState(false);
  const [reanalyzing, setReanalyzing] = useState(false);
  const [message, setMessage] = useState('');
  const [messageType, setMessageType] = useState('success');
  const [form, setForm] = useState({ ...EMPTY_FORM });
  const [enableShortCode, setEnableShortCode] = useState(false);
  const [categories, setCategories] = useState([]);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  // Auth guard
  useEffect(() => {
    if (!isAuthenticated.value) {
      route('/login', true);
    }
  }, []);

  // Check for ?edit query param
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      if (params.get('edit') !== null) {
        setEditing(true);
      }
    }
  }, []);

  // Fetch categories from config API
  useEffect(() => {
    configApi.categories().then(data => {
      setCategories(data.categories || []);
    }).catch(err => {
      console.error('Failed to load categories:', err);
    });
  }, []);

  // Load URL data
  useEffect(() => {
    if (!isNew && id) {
      loadUrl();
    }
  }, [id]);

  async function loadUrl() {
    try {
      const data = await urlApi.get(id);
      setUrlData(data);
      setForm({
        link: data.link || '',
        title: data.title || '',
        description: data.description || '',
        keywords: data.keywords || '',
        category: data.category || '',
        tags: data.tags || '',
        manual_weight: data.manual_weight || 0,
        visit_count: data.visit_count || 0,
        short_code: data.short_code || '',
        ttl: 0,
        color: data.color || '',
        icon: data.icon || '',
        status: data.status || '',
      });
      setEnableShortCode(!!data.short_code);
    } catch (err) {
      showMessage('Failed to load URL: ' + err.message, 'error');
    }
  }

  function showMessage(msg, type = 'success') {
    setMessage(msg);
    setMessageType(type);
    setTimeout(() => setMessage(''), 5000);
  }

  function updateField(field, value) {
    setForm(prev => ({ ...prev, [field]: value }));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!form.link) {
      showMessage('URL link is required', 'error');
      return;
    }

    setSaving(true);
    try {
      if (isNew) {
        // Create flow: POST with link, then PUT with extra fields
        const created = await urlApi.create({ link: form.link });
        const newId = created.ID || created.id;

        // Update with extra fields
        await urlApi.update(newId, {
          title: form.title,
          description: form.description,
          keywords: form.keywords,
          category: form.category,
          tags: form.tags,
          manual_weight: Number(form.manual_weight) || 0,
          color: form.color,
          icon: form.icon,
        });

        // Create short link if enabled
        if (enableShortCode && form.short_code) {
          try {
            await shortApi.create({
              long_url: form.link,
              code: form.short_code,
              ttl: Number(form.ttl) || 0,
            });
          } catch (err) {
            console.warn('Short link creation failed:', err);
          }
        }

        showMessage('URL created successfully');
        route(`/urls/${newId}`);
      } else {
        // Edit flow
        await urlApi.update(id, {
          link: form.link,
          title: form.title,
          description: form.description,
          keywords: form.keywords,
          category: form.category,
          tags: form.tags,
          manual_weight: Number(form.manual_weight) || 0,
          visit_count: Number(form.visit_count) || 0,
          color: form.color,
          icon: form.icon,
        });

        // Create short link if newly enabled and has code
        if (enableShortCode && form.short_code && (!urlData || !urlData.short_code)) {
          try {
            await shortApi.create({
              long_url: form.link,
              code: form.short_code,
              ttl: Number(form.ttl) || 0,
            });
          } catch (err) {
            console.warn('Short link creation failed:', err);
          }
        }

        showMessage('URL updated successfully');
        setEditing(false);
        await loadUrl();
      }
    } catch (err) {
      showMessage('Save failed: ' + err.message, 'error');
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    setShowDeleteModal(true);
  }

  async function confirmDelete() {
    setShowDeleteModal(false);
    try {
      await urlApi.delete(id);
      route('/');
    } catch (err) {
      showMessage('Delete failed: ' + err.message, 'error');
    }
  }

  async function handleReanalyze() {
    setReanalyzing(true);
    try {
      await urlApi.reanalyze(id);
      showMessage('Re-analysis started. Refreshing...');
      setTimeout(() => loadUrl(), 1000);
    } catch (err) {
      showMessage('Reanalyze failed: ' + err.message, 'error');
    } finally {
      setReanalyzing(false);
    }
  }

  if (!isAuthenticated.value) return null;

  const statusBadge = urlData ? `badge-${urlData.status}` : '';

  return (
    <div class="max-w-3xl mx-auto">
      {/* Terminal prompt header */}
      <div class="text-terminal-green font-mono text-sm mb-4 opacity-70">
        <span class="text-terminal-cyan">user@linkstash</span>
        <span class="text-terminal-gray">:</span>
        <span class="text-terminal-green">~</span>
        <span class="text-terminal-gray">$</span>
        {' '}{isNew ? 'touch /urls/new' : `cat /urls/${id}`}
      </div>

      {/* Back link */}
      <a href="/" class="text-terminal-gray hover:text-terminal-green text-sm mb-4 inline-block no-underline">
        ← cd /urls
      </a>

      <div class="terminal-card rounded-lg p-6">
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <h1 class="text-terminal-green text-xl font-mono">
              {isNew ? '> NEW URL' : `> URL #${id}`}
            </h1>
            {!isNew && urlData && urlData.status && (
              <span class={`${statusBadge} text-xs border px-2 py-0.5 rounded`}>
                {urlData.status}
              </span>
            )}
          </div>
          {!isNew && (
            <div class="flex items-center gap-2">
              {!editing && (
                <button onClick={() => setEditing(true)} class="terminal-btn text-xs px-3 py-1">
                  Edit
                </button>
              )}
              <button onClick={handleReanalyze} class="terminal-btn text-xs px-3 py-1" disabled={reanalyzing}>
                {reanalyzing ? 'Analyzing...' : 'Reanalyze'}
              </button>
              <button onClick={handleDelete} class="terminal-btn terminal-btn-danger text-xs px-3 py-1">
                Delete
              </button>
            </div>
          )}
        </div>

        {/* Message */}
        {message && (
          <div class={`text-sm mb-4 p-2 rounded border ${messageType === 'error' ? 'text-terminal-red border-terminal-red' : 'text-terminal-green border-terminal-green'}`}>
            {message}
          </div>
        )}

        {editing ? (
          <form onSubmit={handleSubmit} class="space-y-4">
            {/* Link */}
            <div>
              <label class="block text-terminal-gray text-sm mb-1">Link *</label>
              <input
                type="url"
                class="terminal-input w-full"
                value={form.link}
                onInput={(e) => updateField('link', e.target.value)}
                placeholder="https://example.com"
                required
              />
            </div>

            {/* Title */}
            <div>
              <label class="block text-terminal-gray text-sm mb-1">Title</label>
              <input
                type="text"
                class="terminal-input w-full"
                value={form.title}
                onInput={(e) => updateField('title', e.target.value)}
                placeholder="Page title"
              />
            </div>

            {/* Description */}
            <div>
              <label class="block text-terminal-gray text-sm mb-1">Description</label>
              <textarea
                class="terminal-input w-full h-24 resize-y"
                value={form.description}
                onInput={(e) => updateField('description', e.target.value)}
                placeholder="Description"
              />
            </div>

            {/* Keywords */}
            <div>
              <label class="block text-terminal-gray text-sm mb-1">Keywords</label>
              <input
                type="text"
                class="terminal-input w-full"
                value={form.keywords}
                onInput={(e) => updateField('keywords', e.target.value)}
                placeholder="keyword1, keyword2, keyword3"
              />
            </div>

            {/* Category + Tags */}
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-terminal-gray text-sm mb-1">Category</label>
                <select
                  class="terminal-input w-full"
                  value={form.category}
                  onChange={(e) => updateField('category', e.target.value)}
                >
                  <option value="">Select...</option>
                  {categories.map(cat => (
                    <option key={cat} value={cat}>{cat}</option>
                  ))}
                  {form.category && !categories.includes(form.category) && (
                    <option value={form.category}>{form.category}</option>
                  )}
                </select>
              </div>
              <div>
                <label class="block text-terminal-gray text-sm mb-1">Tags</label>
                <input
                  type="text"
                  class="terminal-input w-full"
                  value={form.tags}
                  onInput={(e) => updateField('tags', e.target.value)}
                  placeholder="tag1, tag2"
                />
              </div>
            </div>

            {/* Weight + Visit Count */}
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-terminal-gray text-sm mb-1">Manual Weight</label>
                <input
                  type="number"
                  class="terminal-input w-full"
                  value={form.manual_weight}
                  onInput={(e) => updateField('manual_weight', e.target.value)}
                />
              </div>
              <div>
                <label class="block text-terminal-gray text-sm mb-1">Visit Count</label>
                <input
                  type="number"
                  class="terminal-input w-full"
                  value={form.visit_count}
                  onInput={(e) => updateField('visit_count', e.target.value)}
                  disabled={isNew}
                />
              </div>
            </div>

            {/* Color */}
            <div>
              <label class="block text-terminal-gray text-sm mb-1">Color</label>
              <ColorPicker value={form.color} onChange={(c) => updateField('color', c)} />
            </div>

            {/* Icon */}
            <div>
              <label class="block text-terminal-gray text-sm mb-1">Icon</label>
              <input
                type="text"
                class="terminal-input w-full"
                value={form.icon}
                onInput={(e) => updateField('icon', e.target.value)}
                placeholder="Icon URL or emoji"
              />
            </div>

            {/* Short Code */}
            <div>
              <label class="flex items-center gap-2 text-terminal-gray text-sm mb-1 cursor-pointer">
                <input
                  type="checkbox"
                  checked={enableShortCode}
                  onChange={(e) => setEnableShortCode(e.target.checked)}
                />
                Enable Short Link
              </label>
              {enableShortCode && (
                <div class="grid grid-cols-2 gap-4 mt-2">
                  <div>
                    <label class="block text-terminal-gray text-xs mb-1">Short Code</label>
                    <input
                      type="text"
                      class="terminal-input w-full"
                      value={form.short_code}
                      onInput={(e) => updateField('short_code', e.target.value)}
                      placeholder="custom-code"
                    />
                  </div>
                  <div>
                    <label class="block text-terminal-gray text-xs mb-1">有效期</label>
                    <select
                      class="terminal-input w-full"
                      value={form.ttl}
                      onChange={(e) => updateField('ttl', Number(e.target.value))}
                    >
                      <option value={0}>永久</option>
                      <option value={86400}>1 天</option>
                      <option value={604800}>7 天</option>
                      <option value={2592000}>30 天</option>
                      <option value={7776000}>90 天</option>
                      <option value={31536000}>1 年</option>
                    </select>
                  </div>
                </div>
              )}
            </div>

            {/* Actions */}
            <div class="flex items-center gap-3 pt-4 border-t border-terminal-border">
              <button type="submit" class="terminal-btn text-xs px-4 py-1.5" disabled={saving}>
                {saving ? '> SAVING...' : isNew ? '> CREATE' : '> SAVE'}
              </button>
              {!isNew && (
                <button
                  type="button"
                  class="terminal-btn terminal-btn-danger text-xs px-4 py-1.5"
                  onClick={() => { setEditing(false); loadUrl(); }}
                >
                  Cancel
                </button>
              )}
            </div>
          </form>
        ) : (
          /* View mode */
          urlData && (
            <div class="space-y-4">
              {/* Title + Link */}
              <div>
                <h2 class="text-lg text-terminal-green mb-1">{urlData.title || 'Untitled'}</h2>
                <a
                  href={urlData.link}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-terminal-cyan text-sm hover:underline break-all"
                >
                  {urlData.link}
                </a>
              </div>

              {/* Description */}
              {urlData.description && (
                <div>
                  <label class="text-terminal-gray text-xs">Description</label>
                  <p class="text-sm text-gray-300 mt-1">{urlData.description}</p>
                </div>
              )}

              {/* Keywords */}
              {urlData.keywords && (
                <div>
                  <label class="text-terminal-gray text-xs">Keywords</label>
                  <p class="text-sm text-gray-300 mt-1">{urlData.keywords}</p>
                </div>
              )}

              {/* Category + Tags */}
              <div class="flex gap-4">
                {urlData.category && (
                  <div>
                    <label class="text-terminal-gray text-xs">Category</label>
                    <p class="text-sm text-terminal-cyan mt-1">{urlData.category}</p>
                  </div>
                )}
                {urlData.tags && (
                  <div>
                    <label class="text-terminal-gray text-xs">Tags</label>
                    <p class="text-sm text-gray-300 mt-1">{urlData.tags}</p>
                  </div>
                )}
              </div>

              {/* Short Link */}
              {urlData.short_code && (
                <div>
                  <label class="text-terminal-gray text-xs">Short Link</label>
                  <p class="text-sm text-terminal-cyan mt-1">
                    <a href={`/s/${urlData.short_code}`} class="hover:underline">/s/{urlData.short_code}</a>
                  </p>
                </div>
              )}

              {/* Metadata */}
              <div class="grid grid-cols-2 md:grid-cols-4 gap-3 pt-4 border-t border-terminal-border text-xs">
                <div>
                  <span class="text-terminal-gray">Manual Weight</span>
                  <p class="text-terminal-green">{urlData.manual_weight || 0}</p>
                </div>
                <div>
                  <span class="text-terminal-gray">Auto Weight</span>
                  <p class="text-terminal-green">{urlData.auto_weight || 0}</p>
                </div>
                <div>
                  <span class="text-terminal-gray">Visits</span>
                  <p class="text-terminal-green">{urlData.visit_count || 0}</p>
                </div>
                <div>
                  <span class="text-terminal-gray">Color</span>
                  <p class="text-terminal-green">{urlData.color || 'default'}</p>
                </div>
                <div>
                  <span class="text-terminal-gray">Created</span>
                  <p class="text-terminal-green">{urlData.CreatedAt ? new Date(urlData.CreatedAt).toLocaleDateString() : '-'}</p>
                </div>
                <div>
                  <span class="text-terminal-gray">Updated</span>
                  <p class="text-terminal-green">{urlData.UpdatedAt ? new Date(urlData.UpdatedAt).toLocaleDateString() : '-'}</p>
                </div>
                <div>
                  <span class="text-terminal-gray">Last Visit</span>
                  <p class="text-terminal-green">{urlData.last_visit_at ? new Date(urlData.last_visit_at).toLocaleDateString() : '-'}</p>
                </div>
                <div>
                  <span class="text-terminal-gray">Status</span>
                  <p class={`badge-${urlData.status}`}>{urlData.status || '-'}</p>
                </div>
              </div>
            </div>
          )
        )}
      </div>
      {/* Delete confirmation modal */}
      <ConfirmModal
        open={showDeleteModal}
        title="确认删除"
        message="确定要删除这个 URL 吗？此操作无法撤销。"
        confirmText="Delete"
        cancelText="Cancel"
        onConfirm={confirmDelete}
        onCancel={() => setShowDeleteModal(false)}
      />
    </div>
  );
}
