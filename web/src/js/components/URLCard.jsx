import { h } from 'preact';
import { route } from 'preact-router';

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

export function URLCard({ url }) {
  function handleClick() {
    route(`/urls/${url.ID}`);
  }

  const colorClass = url.color ? `card-theme-${url.color}` : '';
  const domain = getDomain(url.link);
  const faviconSrc = url.favicon || `https://www.google.com/s2/favicons?domain=${domain}&sz=16`;

  return (
    <div class={`link-item ${colorClass}`} onClick={handleClick}>
      {/* Row 1: favicon + title + domain + time */}
      <div class="flex items-center gap-2 min-w-0">
        <img
          src={faviconSrc}
          alt=""
          class="w-4 h-4 flex-shrink-0 rounded"
          loading="lazy"
          onError={(e) => { e.target.style.display = 'none'; }}
        />
        <span class="text-sm font-medium text-text-primary truncate flex-1" title={url.title || url.link}>
          {url.title || url.link}
        </span>
        <span class="font-mono text-[11px] text-text-muted flex-shrink-0 hidden sm:inline">
          {domain}
        </span>
        {url.score != null && (
          <span class="font-mono text-[10px] text-accent/70 bg-accent/5 px-1.5 py-0.5 rounded flex-shrink-0 tabular-nums">
            {url.score.toFixed(2)}
          </span>
        )}
      </div>

      {/* Row 2: description + category pill + time */}
      <div class="flex items-center gap-2 mt-1 min-w-0">
        {url.description ? (
          <p class="text-xs text-text-secondary truncate flex-1">
            {url.description}
          </p>
        ) : (
          <span class="flex-1" />
        )}
        {url.category && (
          <span class="text-[10px] font-medium text-accent/70 bg-accent/[0.08] px-1.5 py-0.5 rounded-full flex-shrink-0 uppercase tracking-wider">
            {url.category}
          </span>
        )}
        <span class="text-[11px] text-text-muted flex-shrink-0 tabular-nums">
          {relativeTime(url.CreatedAt)}
        </span>
      </div>
    </div>
  );
}
