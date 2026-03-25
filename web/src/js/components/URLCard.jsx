import { h } from 'preact';
import { route } from 'preact-router';
import { urlApi } from '../api.js';
import { copyToClipboard } from '../utils.js';

const STATUS_BADGES = {
  pending: 'text-yellow-400 border-yellow-400',
  analyzing: 'text-blue-400 border-blue-400',
  ready: 'text-green-400 border-green-400',
  failed: 'text-red-400 border-red-400',
};

export function URLCard({ url, onDelete }) {
  async function handleVisit(e) {
    e.preventDefault();
    try {
      await urlApi.visit(url.ID);
    } catch (err) {
      // ignore visit tracking errors
    }
    window.open(url.Link, '_blank');
  }

  function handleCopy(e) {
    e.stopPropagation();
    copyToClipboard(url.Link);
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

  const statusClass = STATUS_BADGES[url.Status] || '';
  const weight = (url.ManualWeight || 0) + (url.AutoWeight || 0);
  const date = url.CreatedAt ? new Date(url.CreatedAt).toLocaleDateString() : '';
  const faviconSrc = url.Favicon || (url.Link ? `https://www.google.com/s2/favicons?domain=${new URL(url.Link).hostname}&sz=16` : '');

  return (
    <div class="relative card-wrapper">
      <div class="terminal-card rounded-lg p-3 group card-default">
        {/* Title row */}
        <div class="flex items-start gap-2 mb-1">
          {faviconSrc && (
            <img src={faviconSrc} alt="" class="w-4 h-4 mt-0.5 flex-shrink-0" loading="lazy" />
          )}
          <a
            href={url.Link}
            onClick={handleVisit}
            class="text-terminal-green text-sm font-semibold truncate flex-1 hover:text-white no-underline cursor-pointer"
            title={url.Title || url.Link}
          >
            {url.Title || url.Link}
          </a>
          {url.Score != null && (
            <span class="text-xs text-terminal-cyan bg-terminal-dark px-1.5 py-0.5 rounded flex-shrink-0">
              {url.Score.toFixed(2)}
            </span>
          )}
        </div>

        {/* Link */}
        <div class="text-terminal-gray text-xs truncate mb-1">
          {url.Link}
        </div>

        {/* Description */}
        {url.Description && (
          <p class="text-gray-400 text-xs line-clamp-2 mb-1">
            {url.Description}
          </p>
        )}

        {/* Keywords (hover) */}
        {url.Keywords && (
          <div class="hidden group-hover:block text-xs text-terminal-cyan mb-1 truncate">
            {url.Keywords}
          </div>
        )}

        {/* Category + Tags + Status */}
        <div class="flex items-center gap-2 flex-wrap text-xs mb-1">
          {url.Category && (
            <span class="text-terminal-cyan bg-terminal-dark px-1.5 py-0.5 rounded">
              {url.Category}
            </span>
          )}
          {url.Tags && url.Tags.split(',').slice(0, 3).map(tag => (
            <span key={tag.trim()} class="text-terminal-gray bg-terminal-dark px-1 py-0.5 rounded">
              {tag.trim()}
            </span>
          ))}
          {url.Status && (
            <span class={`text-xs border px-1 py-0.5 rounded ${statusClass}`}>
              {url.Status}
            </span>
          )}
        </div>

        {/* Bottom row: short link, weight, date */}
        <div class="flex items-center justify-between text-xs text-terminal-gray">
          <div class="flex items-center gap-2">
            {url.ShortCode && (
              <span class="text-terminal-cyan">/s/{url.ShortCode}</span>
            )}
            <span>w:{weight}</span>
          </div>
          <span>{date}</span>
        </div>

        {/* Action buttons (hover) */}
        <div class="hidden group-hover:flex absolute top-2 right-2 gap-1">
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
