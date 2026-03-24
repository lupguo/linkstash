# Refactor loadMore to HTMX — Design Spec

## Overview

Replace the vanilla `fetch()` + `insertAdjacentHTML` infinite scroll and `window.location.href` search/filter in LinkStash with declarative HTMX patterns. This eliminates the `urlListPage()` Alpine component entirely, moving all pagination and search state to the server.

## Goals

1. Replace `loadMore()` with HTMX `hx-trigger="revealed"` sentinel pattern
2. Replace `doSearch()` / `applyFilters()` / `clearSearch()` with HTMX form submission
3. Delete the entire `urlListPage()` Alpine component and the `<script id="page-data">` JSON block
4. Maintain identical user experience: automatic infinite scroll, URL-synced search, toggle buttons for search type

## Non-Goals

- Changing the `urlCard` Alpine component (copyToClipboard, etc.)
- Changing backend data query logic (`fetchURLs`, `parseListParams`)
- Changing the routing structure (`GET /`, `GET /cards`)
- Changing the `url_card.html` template

## Approach: Server-Driven Sentinel

### Core Mechanism — Infinite Scroll

The server returns card HTML followed by a sentinel `<div>` that carries the next page's `hx-get` URL. When the user scrolls to the sentinel, HTMX fires the request automatically. The response replaces the sentinel with new cards (via out-of-band swap) and a fresh sentinel. When no more pages exist, the server returns an end-of-list indicator instead. The OOB wrapper `<div>` is ephemeral — HTMX extracts it, appends its children to `#url-list`, then discards the wrapper.

```
Initial page: server renders cards + sentinel(page=2)
    ↓ user scrolls to sentinel
HTMX: GET /cards?page=2&... → cards(oob→#url-list) + sentinel(page=3)
    ↓ user scrolls to sentinel
HTMX: GET /cards?page=3&... → cards(oob→#url-list) + end-of-list div → done
```

### Core Mechanism — Search and Filters

The search form uses `hx-get="/"` with `hx-target="#search-results"` and `hx-push-url="true"`. The server detects `HX-Request` header and returns cards + sentinel as **inline HTML** (no OOB wrapper), replacing the entire results area.

**Important**: The search response format differs from the scroll response. Search uses direct `innerHTML` replacement — cards and sentinel are returned inline. Scroll uses OOB swap to append cards while replacing the sentinel. This avoids the OOB + innerHTML conflict.

```
User types query + submits form
    → HTMX: GET /?q=foo&search_type=keyword&sort=...
    → Server sees HX-Request header → returns inline cards + sentinel
    → hx-target="#search-results" replaces results area content
    → hx-push-url syncs browser URL bar
```

## Detailed Design

### 1. Sentinel Template

A `{{define}}` block inside `web/components/load_more_sentinel.html` (loaded as a shared component, available to all page templates):

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
{{else}}
<div class="text-center py-4">
    <span class="text-terminal-gray text-xs">// end of list</span>
</div>
{{end}}
{{end}}
```

- `hx-trigger="revealed"`: HTMX fires when the element enters the viewport (replaces IntersectionObserver)
- `hx-swap="outerHTML"`: the sentinel itself is replaced by the response
- `.NextPageQuery`: server-generated query string with page, size, sort, category, search params
- When `!HasMore`: shows end-of-list indicator (no hx-get → scrolling stops)

### 2. Two Response Formats

#### 2a. Scroll response (`GET /cards`) — OOB append + sentinel replace

Used for infinite scroll when the sentinel triggers. Cards are appended to `#url-list` via OOB, and the sentinel is replaced by `outerHTML` swap.

Template block `{{define "scroll_fragment"}}` in `web/components/card_fragment.html`:

```html
{{define "scroll_fragment"}}
<!-- Cards appended to list via out-of-band swap -->
<div hx-swap-oob="beforeend:#url-list">
    {{range .URLs}}
    {{template "url_card" .}}
    {{end}}
</div>

<!-- Sentinel replaces itself (main swap target) -->
{{template "load_more_sentinel" .}}
{{end}}
```

#### 2b. Search response (`GET /` with HX-Request) — inline innerHTML

Used for search/filter HTMX requests. Cards and sentinel are returned inline, replacing the entire `#search-results` area.

Template block `{{define "search_fragment"}}` in `web/components/card_fragment.html`:

```html
{{define "search_fragment"}}
<div id="url-list" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 items-start">
    {{range .URLs}}
    {{template "url_card" .}}
    {{end}}
</div>

{{if not .URLs}}
<div class="terminal-card p-8 rounded-lg text-center">
    <p class="text-terminal-gray">// no urls found</p>
</div>
{{end}}

{{template "load_more_sentinel" .}}
{{end}}
```

### 3. Page Layout — `#search-results` wrapper

Wrap the URL list and sentinel in a container that serves as the HTMX swap target for search:

```html
<!-- index.html -->
<div id="search-results">
    <div id="url-list" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 items-start">
        {{range .URLs}}
        {{template "url_card" .}}
        {{end}}
    </div>

    {{if not .URLs}}
    <div class="terminal-card p-8 rounded-lg text-center">
        <p class="text-terminal-gray">// no urls found</p>
    </div>
    {{end}}

    {{template "load_more_sentinel" .}}
</div>
```

### 4. Search Form

Replace Alpine-driven search with a standard HTML form enhanced by HTMX. Search type toggle buttons use radio inputs styled as buttons (preserving current UX):

```html
<form id="search-form"
      hx-get="/"
      hx-target="#search-results"
      hx-swap="innerHTML"
      hx-push-url="true"
      hx-indicator="#search-indicator">

    <div class="flex gap-2">
        <input type="text" name="q" value="{{.Query}}"
               placeholder="search bookmarks..."
               class="terminal-input flex-1 px-3 py-2 rounded text-sm">
        <button type="submit" class="terminal-btn px-4 py-2 rounded text-sm whitespace-nowrap">
            &gt; grep
        </button>
        <a href="/" class="terminal-btn terminal-btn-danger px-3 py-2 rounded text-sm whitespace-nowrap"
           {{if not .IsSearch}}style="display:none"{{end}}>
            x
        </a>
    </div>

    <!-- Search Type — radio buttons styled as toggle buttons -->
    <div class="flex items-center gap-2">
        <span class="text-terminal-gray text-xs">mode:</span>
        <label class="px-3 py-1 rounded text-xs border transition-all cursor-pointer has-[:checked]:border-terminal-green has-[:checked]:bg-terminal-green/10 has-[:checked]:text-terminal-green border-terminal-border text-terminal-gray hover:text-terminal-green">
            <input type="radio" name="search_type" value="keyword" class="hidden"
                   {{if eq .SearchType "keyword"}}checked{{end}}>
            keyword
        </label>
        <label class="px-3 py-1 rounded text-xs border transition-all cursor-pointer has-[:checked]:border-terminal-green has-[:checked]:bg-terminal-green/10 has-[:checked]:text-terminal-green border-terminal-border text-terminal-gray hover:text-terminal-green">
            <input type="radio" name="search_type" value="semantic" class="hidden"
                   {{if eq .SearchType "semantic"}}checked{{end}}>
            semantic
        </label>
        <label class="px-3 py-1 rounded text-xs border transition-all cursor-pointer has-[:checked]:border-terminal-green has-[:checked]:bg-terminal-green/10 has-[:checked]:text-terminal-green border-terminal-border text-terminal-gray hover:text-terminal-green">
            <input type="radio" name="search_type" value="hybrid" class="hidden"
                   {{if eq .SearchType "hybrid"}}checked{{end}}>
            hybrid
        </label>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap gap-3 items-center text-sm pt-2 border-t border-terminal-border/50">
        <div class="flex items-center gap-2">
            <span class="text-terminal-gray text-xs">category:</span>
            <select name="category" class="terminal-input px-2 py-1 rounded text-xs"
                    hx-trigger="change" hx-include="closest form">
                <option value="">all</option>
                {{range .Categories}}
                <option value="{{.}}" {{if eq . $.FilterCategory}}selected{{end}}>{{.}}</option>
                {{end}}
            </select>
        </div>
        <div class="flex items-center gap-2">
            <span class="text-terminal-gray text-xs">sort:</span>
            <select name="sort" class="terminal-input px-2 py-1 rounded text-xs"
                    hx-trigger="change" hx-include="closest form">
                <option value="weight">weight</option>
                <option value="time" {{if eq .FilterSort "time"}}selected{{end}}>time</option>
            </select>
        </div>
        <div class="flex items-center gap-2">
            <span class="text-terminal-gray text-xs">size:</span>
            <select name="size" class="terminal-input px-2 py-1 rounded text-xs"
                    hx-trigger="change" hx-include="closest form">
                <option value="20">20</option>
                <option value="50" {{if eq .Size 50}}selected{{end}}>50</option>
                <option value="100" {{if eq .Size 100}}selected{{end}}>100</option>
            </select>
        </div>
        <label class="flex items-center gap-1 cursor-pointer">
            <input type="checkbox" name="is_shorturl" value="1"
                   {{if .IsShortURL}}checked{{end}}
                   class="accent-green-500"
                   hx-trigger="change" hx-include="closest form">
            <span class="text-terminal-gray text-xs">short only</span>
        </label>
        <div class="flex items-center gap-2" id="min-score-filter"
             {{if not .IsSearch}}style="display:none"{{end}}>
            <span class="text-terminal-gray text-xs">score:</span>
            <select name="min_score" class="terminal-input px-2 py-1 rounded text-xs"
                    hx-trigger="change" hx-include="closest form">
                <option value="0">all</option>
                <option value="0.2">≥ 0.2</option>
                <option value="0.4">≥ 0.4</option>
                <option value="0.6" selected>≥ 0.6</option>
                <option value="0.8">≥ 0.8</option>
                <option value="1.0">= 1.0</option>
            </select>
        </div>
    </div>

    <input type="hidden" name="page" value="1" />
</form>
```

**minScore visibility**: The `#min-score-filter` div is shown/hidden based on whether a search query exists. Since HTMX search replaces `#search-results` (not the form), the form state persists. For first render, server-side `{{if not .IsSearch}}style="display:none"{{end}}` handles it. For dynamic show/hide as the user types, a small inline script or HTMX `hx-on::after-request` can toggle visibility. Alternatively, keep a minimal Alpine `x-data` just for this UI toggle (not for pagination state).

**clearSearch**: The `x` button is now a simple `<a href="/">` link — navigates to the clean URL, which reloads the page without search params. Visible only when `.IsSearch` is true (server-rendered).

### 5. `HasMore` and `NextPageQuery` Computation

Both `HandleIndex` and `HandleIndexCards` need to compute these values:

```go
// HasMore: use result count heuristic (not totalPages)
// This is more reliable than totalPages, especially in search mode
// where total may be inaccurate.
hasMore := len(displayURLs) == params.Size

// NextPageQuery: current params with page incremented
nextParams := r.URL.Query()
nextParams.Set("page", strconv.Itoa(params.Page + 1))
nextPageQuery := nextParams.Encode()
```

**Why `len(results) == size` instead of `page < totalPages`**: In search mode, `fetchURLs` returns a `total` based on the current page's post-filter count, which is unreliable for computing `totalPages`. Checking `len(results) == size` is the standard cursor-less pagination heuristic — if the server returned a full page, there are likely more results.

### 6. `HandleIndex` — HX-Request Detection

```go
func (h *WebHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
    // ... existing auth + data fetch ...

    hasMore := len(displayURLs) == params.Size
    nextParams := r.URL.Query()
    nextParams.Set("page", strconv.Itoa(params.Page + 1))

    if r.Header.Get("HX-Request") == "true" {
        // HTMX search/filter request → return search fragment (inline, no OOB)
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        t := h.tmplMap["index"]
        t.ExecuteTemplate(w, "search_fragment", struct {
            URLs          []DisplayURL
            HasMore       bool
            NextPageQuery string
        }{displayURLs, hasMore, nextParams.Encode()})
        return
    }
    // Normal request → full page (include HasMore/NextPageQuery in template data)
    h.renderTemplate(w, "index", data)
}
```

### 7. `HandleIndexCards` — Updated Response

```go
func (h *WebHandler) HandleIndexCards(w http.ResponseWriter, r *http.Request) {
    // ... existing auth + data fetch ...

    hasMore := len(displayURLs) == params.Size
    nextParams := r.URL.Query()
    nextParams.Set("page", strconv.Itoa(params.Page + 1))

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    t := h.tmplMap["index"]  // components are parsed into every page template
    t.ExecuteTemplate(w, "scroll_fragment", struct {
        URLs          []DisplayURL
        HasMore       bool
        NextPageQuery string
    }{displayURLs, hasMore, nextParams.Encode()})
}
```

Note: Both handlers use `h.tmplMap["index"]` because component `{{define}}` blocks are parsed into every page template. No new `tmplMap` entry needed.

### 8. Error Handling

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

**Empty search results**: handled by the `search_fragment` template — shows "no urls found" div when `.URLs` is empty.

**401 Unauthorized**: global HTMX event listener redirects to login:

```html
<body hx-on::response-error="if(event.detail.xhr.status===401) window.location='/login'">
```

### 9. Browser History

- `hx-push-url="true"` on search form ensures URL updates on search/filter
- Enable HTMX history cache in `<head>`:

```html
<meta name="htmx-config" content='{"historyCacheSize": 10}'>
```

### 10. Remove `page-data` Script Block

The current template includes:

```html
<script id="page-data" type="application/json">{{json .PageData}}</script>
```

This feeds Alpine's `urlListPage()` component. Since that component is being deleted, remove:
- The `<script id="page-data">` block from `index.html`
- The `PageData` field from the `HandleIndex` template data struct
- The `getPageData()` import/usage in `url-list.js`

## File Changes

### Modified

| File | Change |
|---|---|
| `web/src/js/alpine/url-list.js` | Delete `urlListPage()` entirely. Keep file only if other exports exist, otherwise delete. |
| `web/templates/index.html` | Remove `x-data="urlListPage()"`, `x-init`, Alpine bindings, `@submit.prevent`, `x-model`, `x-show`, `@change`. Add `#search-results` wrapper. Add HTMX attributes to form. Remove `<script id="page-data">` block. Use `{{template "load_more_sentinel"}}` for initial sentinel. Convert search type toggle from Alpine buttons to radio inputs. |
| `app/handler/web_handler.go` | Add `HX-Request` detection in `HandleIndex`. Refactor `HandleIndexCards` to use `scroll_fragment` template. Compute `HasMore` via `len(results) == size` and `NextPageQuery` in both handlers. Remove `PageData` from template data. |

### New

| File | Purpose |
|---|---|
| `web/components/card_fragment.html` | Contains `{{define "scroll_fragment"}}` and `{{define "search_fragment"}}` template blocks |
| `web/components/load_more_sentinel.html` | Contains `{{define "load_more_sentinel"}}` template block (sentinel + end-of-list) |

Both are automatically loaded as shared components by the existing `componentPattern` glob in `NewWebHandler`.

### Deleted

| Code | Reason |
|---|---|
| `urlListPage()` in `url-list.js` | All logic replaced by HTMX |
| `initScroll()` / IntersectionObserver | Replaced by `hx-trigger="revealed"` |
| `loadMore()` / fetch | Replaced by `hx-get` on sentinel |
| `doSearch()` / `applyFilters()` / `clearSearch()` | Replaced by HTMX form |
| `nextPage`, `totalPages`, `hasMore`, `isLoading` state | Server-driven, no client state needed |
| `<script id="page-data">` in index.html | No longer needed without Alpine data component |
| `PageData` in HandleIndex template data | No longer needed |

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
│  → Full HTML: form + #search-results(cards+sentinel)  │
└──────────────┬───────────────────────────────────────┘
               │
     ┌─────────▼──────────┐
     │  User scrolls to    │  ← hx-trigger="revealed"
     │  sentinel            │
     └─────────┬──────────┘
               │
     ┌─────────▼──────────────────────────────────────┐
     │  HTMX GET /cards?page=2&sort=...                │
     │  → scroll_fragment:                              │
     │    oob: cards append to #url-list                │
     │    sentinel(page=3) or end-of-list               │
     └─────────┬──────────────────────────────────────┘
               │ repeats until end-of-list
               ▼
     ┌────────────────────────────────────────────────┐
     │  Search/filter: form hx-get="/"                  │
     │  → HX-Request detected → search_fragment:        │
     │    inline cards + sentinel (no OOB)              │
     │  → hx-target="#search-results" innerHTML          │
     │  → hx-push-url syncs address bar                 │
     └────────────────────────────────────────────────┘
```
