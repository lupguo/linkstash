import { h } from 'preact';
import { route } from 'preact-router';
import { urlApi } from '../api.js';
import { copyToClipboard } from '../utils.js';

export function URLCard({ url, onDelete }) {
  async function handleVisit(e) {
    e.preventDefault();
    try {
      await urlApi.visit(url.ID);
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
    route(`/urls/${url.ID}?edit`);
  }

  async function handleDelete(e) {
    e.stopPropagation();
    if (!confirm('Delete this URL?')) return;
    try {
      await urlApi.delete(url.ID);
      if (onDelete) onDelete(url.ID);
    } catch (err) {
      console.error('Delete failed:', err);
    }
  }

  const weight = (url.manual_weight || 0) + (url.auto_weight || 0);
  const date = url.CreatedAt ? new Date(url.CreatedAt).toLocaleDateString() : '';
  const colorClass = url.color ? `card-theme-${url.color}` : '';
  const faviconSrc = url.favicon || (url.link ? `https://www.google.com/s2/favicons?domain=${new URL(url.link).hostname}&sz=16` : '');

  return (
    <div class="relative card-wrapper">
      <div class={`terminal-card rounded-lg p-3 group card-default ${colorClass}`}>
        {/* Title row */}
        <div class="flex items-start gap-2 mb-1">
          {faviconSrc && (
            <img src={faviconSrc} alt="" class="w-4 h-4 mt-0.5 flex-shrink-0" loading="lazy" />
          )}
          <a
            href={url.link}
            onClick={handleVisit}
            class="text-terminal-green text-sm font-semibold truncate flex-1 hover:text-white no-underline cursor-pointer"
            title={url.title || url.link}
          >
            {url.title || url.link}
          </a>
          {url.score != null && (
            <span class="text-xs text-terminal-cyan bg-terminal-dark px-1.5 py-0.5 rounded flex-shrink-0">
              {url.score.toFixed(2)}
            </span>
          )}
        </div>

        {/* Link */}
        <div class="text-terminal-gray text-xs truncate mb-1">
          {url.link}
        </div>

        {/* Description */}
        {url.description && (
          <p class="card-desc text-gray-400 text-xs mb-1">
            {url.description}
          </p>
        )}

        {/* Keywords (hover) */}
        {url.keywords && (
          <div class="card-hover-extra text-xs text-terminal-cyan mb-1 truncate">
            {url.keywords}
          </div>
        )}

        {/* Category + Tags + Status */}
        <div class="flex items-center gap-2 flex-wrap text-xs mb-1">
          {url.category && (
            <span class="text-terminal-cyan bg-terminal-dark px-1.5 py-0.5 rounded">
              {url.category}
            </span>
          )}
          {url.tags && (
            <div class="tags-row flex items-center gap-1">
              {url.tags.split(',').slice(0, 3).map(tag => (
                <span key={tag.trim()} class="text-terminal-gray bg-terminal-dark px-1 py-0.5 rounded">
                  {tag.trim()}
                </span>
              ))}
            </div>
          )}
          {url.status && (
            <span class={`badge-${url.status} text-xs border px-1 py-0.5 rounded`}>
              {url.status}
            </span>
          )}
        </div>

        {/* Bottom row: short link, weight, date */}
        <div class="flex items-center justify-between text-xs text-terminal-gray">
          <div class="flex items-center gap-2">
            {url.short_code && (
              <span class="text-terminal-cyan">/s/{url.short_code}</span>
            )}
            <span>w:{weight}</span>
          </div>
          <span>{date}</span>
        </div>

        {/* Action buttons (hover) */}
        <div class="card-hover-actions absolute top-2 right-2 gap-1">
          <button
            onClick={handleEdit}
            class="text-xs bg-terminal-dark text-terminal-green border border-terminal-border rounded px-1.5 py-0.5 hover:bg-surface-hi"
            title="Edit"
          >
            ✎
          </button>
          <button
            onClick={handleCopy}
            class="text-xs bg-terminal-dark text-terminal-cyan border border-terminal-border rounded px-1.5 py-0.5 hover:bg-surface-hi"
            title="Copy link"
          >
            ⎘
          </button>
          <button
            onClick={handleDelete}
            class="text-xs bg-terminal-dark text-terminal-red border border-terminal-border rounded px-1.5 py-0.5 hover:bg-surface-hi"
            title="Delete"
          >
            ✕
          </button>
        </div>
      </div>
    </div>
  );
}
