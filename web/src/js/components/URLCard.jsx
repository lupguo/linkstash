import { h } from 'preact';
import { route } from 'preact-router';
import { useState } from 'preact/hooks';
import { urlApi } from '../api.js';
import { copyToClipboard } from '../utils.js';
import { urlListVersion } from '../store.js';
import { ConfirmModal } from './ConfirmModal.jsx';

function getDomain(link) {
  try {
    return new URL(link).hostname;
  } catch {
    return link;
  }
}

function relativeTime(dateStr) {
  if (!dateStr) return '';
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = now - then;
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}

export function URLCard({ url, onDelete }) {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [analyzing, setAnalyzing] = useState(false);

  async function handleVisit(e) {
    e.stopPropagation();
    try {
      await urlApi.visit(url.id);
    } catch (err) {
      // ignore visit tracking errors
    }
    window.open(url.link, '_blank');
  }

  function handleCopy(e) {
    e.stopPropagation();
    copyToClipboard(url.link);
  }

  function handleEdit(e) {
    e.stopPropagation();
    route(`/urls/${url.id}?edit`);
  }

  function handleDelete(e) {
    e.stopPropagation();
    setShowDeleteModal(true);
  }

  async function handleAnalyze(e) {
    e.stopPropagation();
    if (analyzing) return;
    setAnalyzing(true);
    try {
      await urlApi.reanalyze(url.id);
      // Bump list version to trigger re-fetch so card shows "analyzing" status
      urlListVersion.value++;
    } catch (err) {
      console.error('Analyze failed:', err);
    } finally {
      setAnalyzing(false);
    }
  }

  async function confirmDelete() {
    setShowDeleteModal(false);
    try {
      await urlApi.delete(url.id);
      if (onDelete) onDelete(url.id);
    } catch (err) {
      console.error('Delete failed:', err);
    }
  }

  function handleClick() {
    route(`/urls/${url.id}`);
  }

  const colorClass = url.color ? `card-theme-${url.color}` : '';
  const domain = getDomain(url.link);
  const faviconSrc = url.favicon || `https://www.google.com/s2/favicons?domain=${domain}&sz=16`;
  const weight = (url.manual_weight || 0) + (url.auto_weight || 0);

  return (
    <div class={`link-item group relative ${colorClass}`} onClick={handleClick}>
      {/* Row 1: favicon + title + domain */}
      <div class="flex items-center gap-2 min-w-0 overflow-hidden">
        <img
          src={faviconSrc}
          alt=""
          class="w-4 h-4 flex-shrink-0 rounded"
          loading="lazy"
          onError={(e) => { e.target.style.display = 'none'; }}
        />
        <span class="text-sm font-medium text-text-primary truncate min-w-[40%] flex-1" title={url.title || url.link}>
          {url.title || url.link}
        </span>
        <span class="font-mono text-[11px] text-text-muted flex-shrink-0 hidden sm:inline truncate max-w-[40%]" title={domain}>
          {domain}
        </span>
        {url.score != null && (
          <span class="font-mono text-[10px] text-accent/70 bg-accent/5 px-1.5 py-0.5 rounded flex-shrink-0 tabular-nums">
            {url.score.toFixed(2)}
          </span>
        )}
      </div>

      {/* Row 2: description + tags + category + status + weight + time */}
      <div class="flex items-center gap-2 mt-1 min-w-0 overflow-hidden">
        {url.description ? (
          <p class="text-xs text-text-secondary truncate min-w-0 flex-1">
            {url.description}
          </p>
        ) : (
          <span class="flex-1 min-w-0" />
        )}
        {url.tags && (
          <span class="text-[10px] text-text-muted bg-bg-surface-hi/50 px-1.5 py-0.5 rounded flex-shrink-0 hidden md:inline truncate max-w-[80px]">
            {url.tags.split(',')[0].trim()}
          </span>
        )}
        {url.category && (
          <span class="text-[10px] font-medium text-accent/70 bg-accent/[0.08] px-1.5 py-0.5 rounded-full flex-shrink-0 uppercase tracking-wider">
            {url.category}
          </span>
        )}
        {url.status && url.status !== 'ready' && (
          <span class={`badge-${url.status} text-[10px] font-medium border px-1.5 py-0.5 rounded flex-shrink-0`}>
            {url.status}
          </span>
        )}
        {url.short_code && (
          <span class="text-[10px] text-accent/40 font-mono flex-shrink-0 hidden lg:inline">/s/{url.short_code}</span>
        )}
        <span class="font-mono text-[10px] text-text-muted flex-shrink-0 tabular-nums">w:{weight}</span>
        <span class="text-[11px] text-text-muted flex-shrink-0 tabular-nums">
          {relativeTime(url.created_at)}
        </span>
      </div>

      {/* Hover action bar — bottom-right */}
      <div class="hidden group-hover:flex absolute right-2 bottom-1 items-center gap-1 bg-bg-surface border border-border-hi rounded-md px-1 py-0.5 shadow-lg z-10">
        <button
          onClick={handleVisit}
          class="text-[11px] text-text-muted hover:text-accent px-1.5 py-0.5 rounded transition-colors"
          title="Open link"
        >
          Visit
        </button>
        <button
          onClick={handleCopy}
          class="text-[11px] text-text-muted hover:text-accent px-1.5 py-0.5 rounded transition-colors"
          title="Copy link"
        >
          Copy
        </button>
        <button
          onClick={handleEdit}
          class="text-[11px] text-text-muted hover:text-accent px-1.5 py-0.5 rounded transition-colors"
          title="Edit"
        >
          Edit
        </button>
        <button
          onClick={handleAnalyze}
          class="text-[11px] text-text-muted hover:text-accent px-1.5 py-0.5 rounded transition-colors"
          title="Re-analyze with AI"
          disabled={analyzing}
        >
          {analyzing ? (
            <svg class="animate-spin w-3 h-3" viewBox="0 0 24 24" fill="none">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          ) : 'Analyze'}
        </button>
        <button
          onClick={handleDelete}
          class="text-[11px] text-text-muted hover:text-red-400 px-1.5 py-0.5 rounded transition-colors"
          title="Delete"
        >
          Del
        </button>
      </div>

      <ConfirmModal
        open={showDeleteModal}
        title="Confirm Delete"
        message={`Delete "${url.title || url.link}"?`}
        confirmText="Delete"
        cancelText="Cancel"
        onConfirm={confirmDelete}
        onCancel={() => setShowDeleteModal(false)}
      />
    </div>
  );
}
