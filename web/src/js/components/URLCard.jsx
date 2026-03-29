import { h } from 'preact';
import { route } from 'preact-router';
import { useState } from 'preact/hooks';
import { urlApi } from '../api.js';
import { copyToClipboard } from '../utils.js';
import { ConfirmModal } from './ConfirmModal.jsx';

function getDomain(link) {
  try {
    return new URL(link).hostname;
  } catch {
    return link;
  }
}

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

  const [showDeleteModal, setShowDeleteModal] = useState(false);

  function handleDelete(e) {
    e.stopPropagation();
    setShowDeleteModal(true);
  }

  async function confirmDelete() {
    setShowDeleteModal(false);
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
  const domain = getDomain(url.link);
  const faviconSrc = url.favicon || `https://www.google.com/s2/favicons?domain=${domain}&sz=16`;

  return (
    <div class="relative card-wrapper">
      <div class={`terminal-card p-3 group card-default ${colorClass}`}>
        {/* Title — Inter sans-serif, larger, medium weight */}
        <div class="flex items-start gap-2 mb-1">
          <img src={faviconSrc} alt="" class="w-[16px] h-[16px] mt-[2px] flex-shrink-0 rounded" loading="lazy"
            onError={(e) => { e.target.style.display = 'none'; }} />
          <a
            href={url.link}
            onClick={handleVisit}
            class="text-[14px] font-medium text-white/90 truncate flex-1 hover:text-white no-underline cursor-pointer transition-colors leading-snug"
            title={url.title || url.link}
          >
            {url.title || url.link}
          </a>
          {url.score != null && (
            <span class="font-mono text-[10px] text-terminal-cyan/60 bg-terminal-cyan/5 px-1.5 py-0.5 rounded-md flex-shrink-0 tabular-nums">
              {url.score.toFixed(2)}
            </span>
          )}
        </div>

        {/* Domain — mono, small, visible */}
        <div class="font-mono text-[11px] text-gray-500 truncate mb-1">
          {domain}
        </div>

        {/* Description — Inter, readable gray */}
        {url.description && (
          <p class="card-desc text-[13px] text-gray-400 leading-tight mb-1">
            {url.description}
          </p>
        )}

        {/* Keywords (hover only) — mono */}
        {url.keywords && (
          <div class="card-hover-extra font-mono text-[11px] text-terminal-cyan/50 mb-2 leading-relaxed">
            <span class="text-gray-500">kw:</span> {url.keywords}
          </div>
        )}

        {/* Category + Tags + Status — small pills */}
        <div class="card-tags-row flex items-center gap-1.5 mb-1">
          {url.category && (
            <span class="text-[11px] font-medium text-terminal-cyan/70 bg-terminal-cyan/[0.06] px-2 py-[3px] rounded-md">
              {url.category}
            </span>
          )}
          {url.tags && (
            <div class="tags-row flex items-center gap-1">
              {url.tags.split(',').slice(0, 3).map(tag => (
                <span key={tag.trim()} class="text-[10px] text-gray-500 bg-white/[0.04] px-1.5 py-[3px] rounded-md uppercase tracking-wider">
                  {tag.trim()}
                </span>
              ))}
            </div>
          )}
          {url.status && (
            <span class={`badge-${url.status} text-[10px] font-medium border px-1.5 py-[3px] rounded-md`}>
              {url.status}
            </span>
          )}
        </div>

        {/* Bottom row — mono metadata */}
        <div class="flex items-center justify-between font-mono text-[11px] text-gray-500 mt-auto pt-1">
          <div class="flex items-center gap-3">
            {url.short_code && (
              <span class="text-terminal-cyan/40">/s/{url.short_code}</span>
            )}
            <span>w:{weight}</span>
          </div>
          <span>{date}</span>
        </div>

        {/* Action bar — hover only, bottom right */}
        <div class="card-hover-actions items-center justify-end gap-4 pt-2.5 mt-2 border-t border-white/[0.04]">
          <button
            onClick={handleEdit}
            class="text-gray-500 hover:text-terminal-green transition-colors text-[11px] font-medium tracking-wide"
            title="Edit"
          >
            Edit
          </button>
          <button
            onClick={handleCopy}
            class="text-gray-500 hover:text-terminal-cyan transition-colors text-[11px] font-medium tracking-wide"
            title="Copy link"
          >
            Copy
          </button>
          <button
            onClick={handleDelete}
            class="text-gray-500 hover:text-terminal-red transition-colors text-[11px] font-medium tracking-wide"
            title="Delete"
          >
            Delete
          </button>
        </div>
      </div>
      <ConfirmModal
        open={showDeleteModal}
        title="确认删除"
        message={`确定要删除 "${url.title || url.link}" 吗？`}
        confirmText="Delete"
        cancelText="Cancel"
        onConfirm={confirmDelete}
        onCancel={() => setShowDeleteModal(false)}
      />
    </div>
  );
}
