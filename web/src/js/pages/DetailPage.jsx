import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { route } from 'preact-router';
import { isAuthenticated, urlListVersion } from '../store.js';
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
  network_type: '',
  manual_weight: 0,
  visit_count: 0,
  short_code: '',
  ttl: 0,
  color: '',
  icon: '',
  favicon: '',
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
  const [networkTypes, setNetworkTypes] = useState([]);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  useEffect(() => {
    if (!isAuthenticated.value) {
      route('/login', true);
    }
  }, []);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      if (params.get('edit') !== null) {
        setEditing(true);
      }
    }
  }, []);

  useEffect(() => {
    configApi.categories().then(data => {
      setCategories(data.categories || []);
    }).catch(err => {
      console.error('Failed to load categories:', err);
    });
  }, []);

  useEffect(() => {
    configApi.networkTypes().then(data => {
      setNetworkTypes(data.network_types || []);
    }).catch(err => {
      console.error('Failed to load network types:', err);
    });
  }, []);

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
        network_type: data.network_type || '',
        manual_weight: data.manual_weight || 0,
        visit_count: data.visit_count || 0,
        short_code: data.short_code || '',
        ttl: 0,
        color: data.color || '',
        icon: data.icon || '',
        favicon: data.favicon || '',
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
        const created = await urlApi.create({ link: form.link });
        const newId = created.id;

        await urlApi.update(newId, {
          title: form.title,
          description: form.description,
          keywords: form.keywords,
          category: form.category,
          tags: form.tags,
          network_type: form.network_type,
          manual_weight: Number(form.manual_weight) || 0,
          color: form.color,
          icon: form.icon,
          favicon: form.favicon,
        });

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
        await urlApi.update(id, {
          link: form.link,
          title: form.title,
          description: form.description,
          keywords: form.keywords,
          category: form.category,
          tags: form.tags,
          network_type: form.network_type,
          manual_weight: Number(form.manual_weight) || 0,
          visit_count: Number(form.visit_count) || 0,
          color: form.color,
          icon: form.icon,
          favicon: form.favicon,
        });

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
        // Signal IndexPage to refetch when user navigates back
        urlListVersion.value++;
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

  return (
    <div class="max-w-3xl mx-auto">
      {/* Back link */}
      <a href="/" class="text-text-muted hover:text-accent text-sm mb-4 inline-flex items-center gap-1 no-underline transition-colors">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path d="M15 19l-7-7 7-7" /></svg>
        Back to list
      </a>

      <div class="surface-card rounded-lg p-6">
        {/* Header: title + actions */}
        <div class="flex items-start justify-between mb-5">
          <div class="flex-1 min-w-0">
            <h1 class="text-text-primary text-xl font-semibold truncate">
              {isNew ? 'New URL' : (urlData?.title || `URL #${id}`)}
            </h1>
            {!isNew && urlData && (
              <div class="flex items-center gap-3 mt-1.5 text-xs text-text-muted">
                <span>Status: <span class={`badge-${urlData.status} border px-1.5 py-0.5 rounded`}>{urlData.status || '-'}</span></span>
                <span>ID: {id}</span>
              </div>
            )}
          </div>
          {!isNew && (
            <div class="flex items-center gap-2 flex-shrink-0 ml-4">
              {!editing && (
                <button onClick={() => setEditing(true)} class="btn text-sm px-3 py-1.5">
                  Edit
                </button>
              )}
              <button onClick={handleReanalyze} class="btn text-sm px-3 py-1.5" disabled={reanalyzing}>
                {reanalyzing ? 'Analyzing...' : 'Reanalyze'}
              </button>
              <button onClick={handleDelete} class="btn btn-danger text-sm px-3 py-1.5">
                Delete
              </button>
            </div>
          )}
        </div>

        {/* Message */}
        {message && (
          <div class={`text-sm mb-4 p-2.5 rounded border ${messageType === 'error' ? 'text-red-400 border-red-400/30 bg-red-400/5' : 'text-accent border-accent/30 bg-accent/5'}`}>
            {message}
          </div>
        )}

        {editing ? (
          <form onSubmit={handleSubmit} class="space-y-4">
            {/* Link */}
            <div>
              <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Link *</label>
              <input type="url" class="input w-full text-sm" value={form.link} onInput={(e) => updateField('link', e.target.value)} placeholder="https://example.com" required />
            </div>

            {/* Title */}
            <div>
              <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Title</label>
              <input type="text" class="input w-full text-sm" value={form.title} onInput={(e) => updateField('title', e.target.value)} placeholder="Page title" />
            </div>

            {/* Description */}
            <div>
              <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Description</label>
              <textarea class="input w-full h-20 resize-y text-sm" value={form.description} onInput={(e) => updateField('description', e.target.value)} placeholder="Description" />
            </div>

            {/* Keywords */}
            <div>
              <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Keywords</label>
              <input type="text" class="input w-full text-sm" value={form.keywords} onInput={(e) => updateField('keywords', e.target.value)} placeholder="keyword1, keyword2, keyword3" />
            </div>

            {/* Category + Network Type + Tags */}
            <div class="grid grid-cols-3 gap-4">
              <div>
                <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Category</label>
                <select class="input w-full text-sm" value={form.category} onChange={(e) => updateField('category', e.target.value)}>
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
                <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Network</label>
                <select class="input w-full text-sm" value={form.network_type} onChange={(e) => updateField('network_type', e.target.value)}>
                  <option value="">Select...</option>
                  {networkTypes.map(nt => (
                    <option key={nt.key} value={nt.key}>{nt.label}</option>
                  ))}
                  {form.network_type && !networkTypes.find(nt => nt.key === form.network_type) && (
                    <option value={form.network_type}>{form.network_type}</option>
                  )}
                </select>
              </div>
              <div>
                <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Tags</label>
                <input type="text" class="input w-full text-sm" value={form.tags} onInput={(e) => updateField('tags', e.target.value)} placeholder="tag1, tag2" />
              </div>
            </div>

            {/* Auto Weight + Visit Count + Manual Weight */}
            <div class="grid grid-cols-3 gap-4">
              <div>
                <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Auto Weight</label>
                <input type="number" class="input w-full text-sm text-text-muted bg-bg-surface-hi/30" value={urlData?.auto_weight || 0} disabled />
              </div>
              <div>
                <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Visit Count</label>
                <input type="number" class="input w-full text-sm" value={form.visit_count} onInput={(e) => updateField('visit_count', e.target.value)} disabled={isNew} />
              </div>
              <div>
                <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Manual Weight</label>
                <input type="number" class="input w-full text-sm" value={form.manual_weight} onInput={(e) => updateField('manual_weight', e.target.value)} />
              </div>
            </div>

            {/* Color */}
            <div>
              <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Color</label>
              <ColorPicker value={form.color} onChange={(c) => updateField('color', c)} />
            </div>

            {/* Icon + Favicon — same row */}
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Icon</label>
                <input type="text" class="input w-full text-sm" value={form.icon} onInput={(e) => updateField('icon', e.target.value)} placeholder="Emoji icon" />
              </div>
              <div>
                <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Favicon</label>
                <div class="flex items-center gap-2">
                  {form.favicon && (
                    <img src={form.favicon} alt="favicon" class="w-5 h-5 rounded flex-shrink-0" />
                  )}
                  <span class="input flex-1 text-sm text-text-muted truncate cursor-default">{form.favicon ? 'Has favicon' : 'No favicon'}</span>
                  {form.favicon && (
                    <button
                      type="button"
                      class="btn btn-danger text-xs px-2 py-1 flex-shrink-0"
                      onClick={() => updateField('favicon', '')}
                    >
                      Clear
                    </button>
                  )}
                </div>
              </div>
            </div>

            {/* Short Link */}
            <div>
              <label class="flex items-center gap-2 text-text-secondary text-sm mb-1 cursor-pointer">
                <input type="checkbox" checked={enableShortCode} onChange={(e) => setEnableShortCode(e.target.checked)} class="accent-sky-400" />
                Enable Short Link
              </label>
              {enableShortCode && (
                <div class="grid grid-cols-2 gap-4 mt-2">
                  <div>
                    <label class="block text-text-muted text-xs mb-1">Short Code</label>
                    <input type="text" class="input w-full text-sm" value={form.short_code} onInput={(e) => updateField('short_code', e.target.value)} placeholder="custom-code" />
                  </div>
                  <div>
                    <label class="block text-text-muted text-xs mb-1">TTL</label>
                    <select class="input w-full text-sm" value={form.ttl} onChange={(e) => updateField('ttl', Number(e.target.value))}>
                      <option value={0}>Permanent</option>
                      <option value={86400}>1 day</option>
                      <option value={604800}>7 days</option>
                      <option value={2592000}>30 days</option>
                      <option value={7776000}>90 days</option>
                      <option value={31536000}>1 year</option>
                    </select>
                  </div>
                </div>
              )}
            </div>

            {/* Actions */}
            <div class="flex items-center gap-3 pt-4 border-t border-border-hi">
              <button type="submit" class="btn btn-primary text-sm px-4 py-2" disabled={saving}>
                {saving ? 'Saving...' : isNew ? 'Create' : 'Save'}
              </button>
              {!isNew && (
                <button type="button" class="btn text-sm px-4 py-2" onClick={() => { setEditing(false); loadUrl(); }}>
                  Cancel
                </button>
              )}
            </div>
          </form>
        ) : (
          /* View mode */
          urlData && (
            <div class="space-y-3">
              {/* Link */}
              <div>
                <label class="text-text-muted text-xs font-medium uppercase tracking-wider">Link</label>
                <p class="mt-1">
                  <a href={urlData.link} target="_blank" rel="noopener noreferrer" class="text-accent text-sm font-mono hover:underline break-all">
                    {urlData.link}
                  </a>
                </p>
              </div>

              {/* Description */}
              {urlData.description && (
                <div>
                  <label class="text-text-muted text-xs font-medium uppercase tracking-wider">Description</label>
                  <p class="text-sm text-text-secondary mt-1">{urlData.description}</p>
                </div>
              )}

              {/* Keywords */}
              {urlData.keywords && (
                <div>
                  <label class="text-text-muted text-xs font-medium uppercase tracking-wider">Keywords</label>
                  <p class="text-sm text-text-secondary mt-1">{urlData.keywords}</p>
                </div>
              )}

              {/* Category + Tags */}
              <div class="flex gap-6">
                {urlData.category && (
                  <div>
                    <label class="text-text-muted text-xs font-medium uppercase tracking-wider">Category</label>
                    <p class="text-sm text-accent mt-1">{urlData.category}</p>
                  </div>
                )}
                {urlData.tags && (
                  <div>
                    <label class="text-text-muted text-xs font-medium uppercase tracking-wider">Tags</label>
                    <p class="text-sm text-text-secondary mt-1">{urlData.tags}</p>
                  </div>
                )}
              </div>

              {/* Network Type */}
              {urlData.network_type && urlData.network_type !== '' && (
                <div>
                  <label class="text-text-muted text-xs font-medium uppercase tracking-wider">Network</label>
                  <p class="text-sm text-accent mt-1">
                    {(() => {
                      const nt = networkTypes.find(n => n.key === urlData.network_type);
                      return nt ? nt.label : urlData.network_type;
                    })()}
                  </p>
                </div>
              )}

              {/* Short Link */}
              {urlData.short_code && (
                <div>
                  <label class="text-text-muted text-xs font-medium uppercase tracking-wider">Short Link</label>
                  <p class="text-sm text-accent mt-1">
                    <a href={`/s/${urlData.short_code}`} class="hover:underline font-mono">/s/{urlData.short_code}</a>
                  </p>
                </div>
              )}

              {/* Metadata grid */}
              <div class="grid grid-cols-2 md:grid-cols-4 gap-3 pt-4 border-t border-border-hi text-xs">
                <div>
                  <span class="text-text-muted font-medium uppercase tracking-wider">Auto Weight</span>
                  <p class="text-text-primary mt-0.5">{urlData.auto_weight || 0}</p>
                </div>
                <div>
                  <span class="text-text-muted font-medium uppercase tracking-wider">Manual Weight</span>
                  <p class="text-text-primary mt-0.5">{urlData.manual_weight || 0}</p>
                </div>
                <div>
                  <span class="text-text-muted font-medium uppercase tracking-wider">Visits</span>
                  <p class="text-text-primary mt-0.5">{urlData.visit_count || 0}</p>
                </div>
                <div>
                  <span class="text-text-muted font-medium uppercase tracking-wider">Color</span>
                  <p class="text-text-primary mt-0.5">{urlData.color || 'default'}</p>
                </div>
                <div>
                  <span class="text-text-muted font-medium uppercase tracking-wider">Created</span>
                  <p class="text-text-primary mt-0.5">{urlData.created_at ? new Date(urlData.created_at).toLocaleDateString() : '-'}</p>
                </div>
                <div>
                  <span class="text-text-muted font-medium uppercase tracking-wider">Updated</span>
                  <p class="text-text-primary mt-0.5">{urlData.updated_at ? new Date(urlData.updated_at).toLocaleDateString() : '-'}</p>
                </div>
                <div>
                  <span class="text-text-muted font-medium uppercase tracking-wider">Last Visit</span>
                  <p class="text-text-primary mt-0.5">{urlData.last_visit_at ? new Date(urlData.last_visit_at).toLocaleDateString() : '-'}</p>
                </div>
                <div>
                  <span class="text-text-muted font-medium uppercase tracking-wider">Status</span>
                  <p class="text-text-primary mt-0.5">{urlData.status || '-'}</p>
                </div>
              </div>
            </div>
          )
        )}
      </div>
      <ConfirmModal
        open={showDeleteModal}
        title="Confirm Delete"
        message="Are you sure you want to delete this URL? This action cannot be undone."
        confirmText="Delete"
        cancelText="Cancel"
        onConfirm={confirmDelete}
        onCancel={() => setShowDeleteModal(false)}
      />
    </div>
  );
}
