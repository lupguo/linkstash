# Refactor loadMore to HTMX — Design Spec

## Overview

Replace the vanilla `fetch()` + `insertAdjacentHTML` infinite scroll and `window.location.href` search/filter in LinkStash with declarative HTMX patterns. This eliminates the `urlListPage()` Alpine component entirely, moving all pagination and search state to the server.

## Goals

1. Replace `loadMore()` with HTMX `hx-trigger="revealed"` sentinel pattern
2. Replace `doSearch()` / `applyFilters()` with HTMX form submission
3. Delete the entire `urlListPage()` Alpine component (loadMore, initScroll, doSearch, applyFilters, clearSearch, and all pagination/search state)
4. Maintain identical user experience: automatic infinite scroll, URL-synced search

## Non-Goals

- Changing the `urlCard` Alpine component (copyToClipboard, etc.)
- Changing backend data query logic (`fetchURLs`, `parseListParams`)
- Changing the routing structure (`GET /`, `GET /cards`)
- Changing the `url_card.html` template

## Approach: Server-Driven Sentinel

### Core Mechanism — Infinite Scroll

The server returns card HTML followed by a sentinel `<div>` that carries the next page's `hx-get` URL. When the user scrolls to the sentinel, HTMX fires the request automatically. The response replaces the sentinel with new cards (via out-of-band swap) and a fresh sentinel. When no more pages exist, the server omits the sentinel and scrolling stops.

```
Initial page: server renders cards + sentinel(page=2)
    ↓ user scrolls to sentinel
HTMX: GET /cards?page=2&... → cards(oob→#url-list) + sentinel(page=3)
    ↓ user scrolls to sentinel
HTMX: GET /cards?page=3&... → cards(oob→#url-list) + no sentinel → done
```

### Core Mechanism — Search and Filters

The search form uses `hx-get="/"` with `hx-target="#url-list"` and `hx-push-url="true"`. The server detects `HX-Request` header and returns only the card fragment (not the full page). Filter `<select>` elements use `hx-trigger="change"` to submit automatically when changed.

```
User types query + submits form
    → HTMX: GET /?q=foo&search_type=keyword&sort=...
    → Server sees HX-Request header → returns card fragment + sentinel
    → hx-target="#url-list" replaces list content
    → hx-push-url syncs browser URL bar
```

## Detailed Design

### 1. Sentinel Template

A reusable Go template block for the sentinel div:

```html
{{define "load_more_sentinel"}}
{{if .HasMore}}
<div id="load-more-sentinel"
     hx-get="/cards?{{.NextPageQuery}}"
     hx-trigger="revealed"
     hx-swap="outerHTML"
     hx-target="this">
    <span class="text-terminal-gray text-sm">loading...</span>
</div>
{{end}}
{{end}}
```

- `hx-trigger="revealed"`: HTMX fires when the element enters the viewport (replaces IntersectionObserver)
- `hx-swap="outerHTML"`: the sentinel itself is replaced by the response
- `.NextPageQuery`: server-generated query string with page, size, sort, category, search params

### 2. `/cards` Response Format

The `/cards` endpoint returns two parts:

```html
<!-- Part 1: cards appended to list via out-of-band swap -->
<div hx-swap-oob="beforeend:#url-list">
    {{range .URLs}}
    {{template "url_card" .}}
    {{end}}
</div>

<!-- Part 2: new sentinel (only if more pages exist) -->
{{template "load_more_sentinel" .}}
```

`hx-swap-oob="beforeend:#url-list"` tells HTMX to append the cards to `#url-list` as an out-of-band operation, independent of the main swap target (the sentinel itself).

### 3. Search Form

Replace Alpine-driven search with a standard HTML form enhanced by HTMX:

```html
<form id="search-form"
      hx-get="/"
      hx-target="#url-list"
      hx-swap="innerHTML"
      hx-push-url="true"
      hx-indicator="#search-indicator">

    <input type="text" name="q" value="{{.Query}}" />
    <select name="search_type">
        <option value="keyword">keyword</option>
        <option value="semantic">semantic</option>
        <option value="hybrid">hybrid</option>
    </select>
    <select name="min_score">...</select>
    <select name="category" hx-trigger="change" hx-include="closest form">...</select>
    <select name="sort" hx-trigger="change" hx-include="closest form">...</select>
    <select name="size" hx-trigger="change" hx-include="closest form">...</select>
    <input type="hidden" name="page" value="1" />
    <button type="submit">search</button>
</form>
```

- Filter `<select>` elements with `hx-trigger="change"` auto-submit on change
- `hx-include="closest form"` ensures all form fields are included
- `hx-push-url="true"` syncs the URL bar for bookmarkability and back-button support
- `searchType` ↔ `minScore` visibility: change from Alpine `x-show` to vanilla JS or CSS `:has()` selector

### 4. `HandleIndex` — HX-Request Detection

```go
func (h *WebHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
    // ... existing auth + data fetch ...

    if r.Header.Get("HX-Request") == "true" {
        // HTMX search/filter request → return card fragment only
        h.renderCardFragment(w, displayURLs, hasMore, nextPageQuery)
        return
    }
    // Normal request → full page
    h.renderTemplate(w, "index", data)
}
```

### 5. `HandleIndexCards` — Updated Response

```go
func (h *WebHandler) HandleIndexCards(w http.ResponseWriter, r *http.Request) {
    // ... existing auth + data fetch ...

    // Render card fragment with oob swap + sentinel
    h.renderCardFragment(w, displayURLs, hasMore, nextPageQuery)
}
```

### 6. `renderCardFragment` — Shared Helper

New method used by both `HandleIndex` (HTMX mode) and `HandleIndexCards`:

```go
func (h *WebHandler) renderCardFragment(w http.ResponseWriter, urls []DisplayURL, hasMore bool, nextPageQuery string) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")

    data := struct {
        URLs          []DisplayURL
        HasMore       bool
        NextPageQuery string
    }{urls, hasMore, nextPageQuery}

    t := h.tmplMap["card_fragment"]
    t.Execute(w, data)
}
```

### 7. Error Handling

**Network/server error on infinite scroll**: return a retry-able sentinel:

```go
if err != nil {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprintf(w, `<div id="load-more-sentinel" class="text-center py-4">
        <span class="text-red-400 text-sm">load failed</span>
        <button hx-get="/cards?%s" hx-target="#load-more-sentinel"
                hx-swap="outerHTML" class="text-terminal-green text-sm ml-2 underline">
            retry
        </button>
    </div>`, r.URL.RawQuery)
    return
}
```

**Empty search results**: server returns an empty-state div (no sentinel):

```html
<div class="terminal-card p-8 rounded-lg text-center">
    <p class="text-terminal-gray">// no urls found</p>
</div>
```

**401 Unauthorized**: global HTMX event listener redirects to login:

```html
<body hx-on::response-error="if(event.detail.xhr.status===401) window.location='/login'">
```

### 8. Browser History

- `hx-push-url="true"` on search form ensures URL updates on search/filter
- Enable HTMX history cache in `<head>`:

```html
<meta name="htmx-config" content='{"historyCacheSize": 10}'>
```

## File Changes

### Modified

| File | Change |
|---|---|
| `web/src/js/alpine/url-list.js` | Delete `urlListPage()` entirely. File may become empty or be deleted if no other exports remain. |
| `web/templates/index.html` | Remove `x-data="urlListPage()"` and all Alpine bindings. Add HTMX attributes to search form. Add sentinel after `#url-list`. Convert searchType↔minScore toggle to vanilla JS/CSS. |
| `app/handler/web_handler.go` | Add `HX-Request` detection in `HandleIndex`. Refactor `HandleIndexCards` to use `renderCardFragment`. Add `renderCardFragment` helper. Compute `HasMore` and `NextPageQuery` in both handlers. |

### New

| File | Purpose |
|---|---|
| `web/components/card_fragment.html` | Shared template: oob-swap card list + sentinel. Used by `/cards` and `HandleIndex` HTMX mode. |

### Deleted

| Code | Reason |
|---|---|
| `urlListPage()` in `url-list.js` | All logic replaced by HTMX |
| `initScroll()` / IntersectionObserver | Replaced by `hx-trigger="revealed"` |
| `loadMore()` / fetch | Replaced by `hx-get` on sentinel |
| `doSearch()` / `applyFilters()` / `clearSearch()` | Replaced by HTMX form |
| `nextPage`, `totalPages`, `hasMore`, `isLoading` state | Server-driven, no client state needed |

### Unchanged

| File | Reason |
|---|---|
| `web/src/js/alpine/url-card.js` | Card-level interactions unrelated to pagination |
| `web/components/url_card.html` | Single card template unchanged |
| `cmd/server/main.go` | Routes unchanged |
| `app/handler/web_handler.go` `fetchURLs`, `parseListParams` | Query logic unchanged |

## Data Flow Summary

```
┌──────────────────────────────────────────────────────┐
│  GET / (browser)                                      │
│  → Full HTML: form + cards + sentinel(page=2)         │
└──────────────┬───────────────────────────────────────┘
               │
     ┌─────────▼──────────┐
     │  User scrolls to    │  ← hx-trigger="revealed"
     │  sentinel            │
     └─────────┬──────────┘
               │
     ┌─────────▼──────────────────────────────┐
     │  HTMX GET /cards?page=2&sort=...        │
     │  → oob: cards append to #url-list       │
     │  → sentinel(page=3) or no sentinel      │
     └─────────┬──────────────────────────────┘
               │ repeats until no sentinel
               ▼
     ┌────────────────────────────────────────┐
     │  Search/filter: form hx-get="/"         │
     │  → HX-Request detected → card fragment  │
     │  → hx-target="#url-list" innerHTML       │
     │  → hx-push-url syncs address bar        │
     └────────────────────────────────────────┘
```
