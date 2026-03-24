# HTMX LoadMore Refactoring — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace fetch-based infinite scroll and Alpine search with declarative HTMX patterns, eliminating the `urlListPage()` component entirely.

**Architecture:** Server-driven sentinel pattern for infinite scroll (OOB append + sentinel replace). HTMX form with `hx-get` for search/filters (inline innerHTML replacement). Two distinct response formats to avoid OOB+innerHTML conflict.

**Tech Stack:** Go templates, HTMX (`hx-trigger="revealed"`, `hx-swap-oob`, `hx-push-url`), HTML radio inputs (replacing Alpine toggle buttons), Tailwind CSS `:has()` selector.

**Spec:** `docs/specs/2026-03-24-htmx-loadmore-design.md`

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `web/components/load_more_sentinel.html` | Create | `{{define "load_more_sentinel"}}` — sentinel div or end-of-list indicator |
| `web/components/card_fragment.html` | Create | `{{define "scroll_fragment"}}` and `{{define "search_fragment"}}` |
| `app/handler/web_handler.go` | Modify | Add `HX-Request` detection, refactor `HandleIndexCards`, compute `HasMore`/`NextPageQuery`, remove `PageData` |
| `web/templates/index.html` | Modify | Remove Alpine bindings, add HTMX form, `#search-results` wrapper, sentinel |
| `web/src/js/alpine/url-list.js` | Delete | Entire file — all logic replaced by HTMX |
| `web/src/js/app.js` | Modify | Remove `urlListPage` import and `window.urlListPage` assignment |
| `web/templates/layout.html` | Modify | Add HTMX config meta tag and 401 handler on `<body>` |

---

### Task 1: Create sentinel template component

**Files:**
- Create: `web/components/load_more_sentinel.html`

- [ ] **Step 1: Create the sentinel template**

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

- [ ] **Step 2: Verify template parses**

Run: `cd /data/projects/github.com/lupguo/linkstash && go build ./cmd/server/`
Expected: Build succeeds (template is auto-discovered by `componentPattern` glob)

- [ ] **Step 3: Commit**

```bash
git add web/components/load_more_sentinel.html
git commit -m "feat: add HTMX sentinel template component"
```

---

### Task 2: Create card fragment templates

**Files:**
- Create: `web/components/card_fragment.html`

- [ ] **Step 1: Create the scroll and search fragment templates**

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

{{define "search_fragment"}}
<div id="url-list" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 items-start">
    {{range .URLs}}
    {{template "url_card" .}}
    {{end}}
</div>

{{if not .URLs}}
<div class="terminal-card p-8 rounded-lg text-center">
    <p class="text-terminal-gray">// no urls found</p>
    <p class="text-terminal-gray text-xs mt-2">try adding a url or adjusting filters</p>
</div>
{{end}}

{{template "load_more_sentinel" .}}
{{end}}
```

- [ ] **Step 2: Verify template parses**

Run: `cd /data/projects/github.com/lupguo/linkstash && go build ./cmd/server/`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add web/components/card_fragment.html
git commit -m "feat: add scroll and search fragment templates for HTMX"
```

---

### Task 3: Add fragment data struct and refactor `HandleIndexCards`

**Files:**
- Modify: `app/handler/web_handler.go:325-359`

- [ ] **Step 1: Add `fragmentData` struct and `buildNextPageQuery` helper**

Add after the `listParams` type (around line 157), before `HandleIndex`:

```go
// fragmentData holds data for HTMX card fragment templates.
type fragmentData struct {
	URLs          []indexURL
	HasMore       bool
	NextPageQuery string
}

// buildNextPageQuery returns the current query params with page incremented.
func buildNextPageQuery(r *http.Request, currentPage int) string {
	nextParams := r.URL.Query()
	nextParams.Set("page", strconv.Itoa(currentPage+1))
	return nextParams.Encode()
}
```

- [ ] **Step 2: Refactor `HandleIndexCards` to use `scroll_fragment` template**

Replace lines 325-359 of `web_handler.go`:

```go
// HandleIndexCards serves GET /cards - returns HTMX scroll fragment (OOB cards + sentinel).
func (h *WebHandler) HandleIndexCards(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	params := parseListParams(r)
	displayURLs, _, _, err := h.fetchURLs(params)
	if err != nil {
		slog.Error("fetch urls error", "component", "web_handler", "error", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div id="load-more-sentinel" class="text-center py-4">
			<span class="text-red-400 text-sm">load failed</span>
			<button hx-get="/cards?%s" hx-target="#load-more-sentinel"
					hx-swap="outerHTML" class="text-terminal-green text-sm ml-2 underline">retry</button>
		</div>`, r.URL.RawQuery)
		return
	}

	hasMore := len(displayURLs) == params.Size

	t, ok := h.tmplMap["index"]
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "scroll_fragment", fragmentData{
		URLs:          displayURLs,
		HasMore:       hasMore,
		NextPageQuery: buildNextPageQuery(r, params.Page),
	}); err != nil {
		slog.Error("render scroll fragment error", "component", "web_handler", "error", err)
	}
}
```

- [ ] **Step 3: Verify build**

Run: `cd /data/projects/github.com/lupguo/linkstash && go build ./cmd/server/`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add app/handler/web_handler.go
git commit -m "refactor: HandleIndexCards uses scroll_fragment template with HTMX OOB"
```

---

### Task 4: Add HX-Request detection to `HandleIndex`

**Files:**
- Modify: `app/handler/web_handler.go:270-323`

- [ ] **Step 1: Add HX-Request branch and add `HasMore`/`NextPageQuery` to full-page data**

Replace `HandleIndex` (lines 270-323):

```go
// HandleIndex serves GET / - the URL list page with optional search.
func (h *WebHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	params := parseListParams(r)
	displayURLs, total, isSearch, err := h.fetchURLs(params)
	if err != nil {
		slog.Error("fetch urls error", "component", "web_handler", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hasMore := len(displayURLs) == params.Size

	// HTMX search/filter request → return search fragment only
	if r.Header.Get("HX-Request") == "true" {
		t, ok := h.tmplMap["index"]
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.ExecuteTemplate(w, "search_fragment", fragmentData{
			URLs:          displayURLs,
			HasMore:       hasMore,
			NextPageQuery: buildNextPageQuery(r, params.Page),
		}); err != nil {
			slog.Error("render search fragment error", "component", "web_handler", "error", err)
		}
		return
	}

	// Normal request → full page
	pd := h.newPageData(params.Page, params.Size, total, params.Category, params.Sort)
	pd.Categories = h.categories

	data := struct {
		pageData
		URLs          []indexURL
		Query         string
		SearchType    string
		IsSearch      bool
		IsShortURL    bool
		MinScore      float64
		HasMore       bool
		NextPageQuery string
	}{
		pageData:      pd,
		URLs:          displayURLs,
		Query:         params.Query,
		SearchType:    params.SearchType,
		IsSearch:      isSearch,
		IsShortURL:    params.IsShortURL,
		MinScore:      params.MinScore,
		HasMore:       hasMore,
		NextPageQuery: buildNextPageQuery(r, params.Page),
	}

	h.renderTemplate(w, "index", data)
}
```

Key changes from the old version:
- Removed `PageData` map (the JSON block for Alpine)
- Added `HasMore` and `NextPageQuery` fields to the template data struct
- Added `HX-Request` branch returning `search_fragment`

- [ ] **Step 2: Verify build**

Run: `cd /data/projects/github.com/lupguo/linkstash && go build ./cmd/server/`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add app/handler/web_handler.go
git commit -m "feat: HandleIndex returns search_fragment for HTMX requests"
```

---

### Task 5: Rewrite `index.html` — remove Alpine, add HTMX

**Files:**
- Modify: `web/templates/index.html` (full rewrite)

- [ ] **Step 1: Replace entire `index.html` content**

```html
{{define "content"}}
<div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-4 gap-3">
        <div>
            <h1 class="text-xl font-bold">
                <span class="text-terminal-gray">$</span> ls -la /urls
            </h1>
            <p class="text-terminal-gray text-xs mt-1">
                // total: <span class="text-terminal-green">{{.Total}}</span> records
            </p>
        </div>
        <a href="/urls/new" class="terminal-btn px-4 py-2 rounded text-sm text-center">
            &gt; touch
        </a>
    </div>

    <!-- Search & Filters — HTMX-driven -->
    <div class="glass-panel p-3 rounded-lg mb-4 space-y-2">
        <form id="search-form"
              hx-get="/"
              hx-target="#search-results"
              hx-swap="innerHTML"
              hx-push-url="true"
              class="flex flex-col gap-2">
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
                        <option value="weight" {{if eq .FilterSort "weight"}}selected{{end}}>weight</option>
                        <option value="time" {{if eq .FilterSort "time"}}selected{{end}}>time</option>
                    </select>
                </div>
                <div class="flex items-center gap-2">
                    <span class="text-terminal-gray text-xs">size:</span>
                    <select name="size" class="terminal-input px-2 py-1 rounded text-xs"
                            hx-trigger="change" hx-include="closest form">
                        <option value="20" {{if eq .Size 20}}selected{{end}}>20</option>
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
    </div>

    <!-- Search Results (HTMX swap target) -->
    <div id="search-results">
        <div id="url-list" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 items-start">
            {{range .URLs}}
            {{template "url_card" .}}
            {{end}}
        </div>

        {{if not .URLs}}
        <div class="terminal-card p-8 rounded-lg text-center">
            <p class="text-terminal-gray">// no urls found</p>
            <p class="text-terminal-gray text-xs mt-2">try adding a url or adjusting filters</p>
        </div>
        {{end}}

        {{template "load_more_sentinel" .}}
    </div>
</div>
{{end}}
```

Key changes:
- Removed: `x-data="urlListPage()"`, `x-init="initScroll()"`, `@keyup.escape`, all `x-model`, `x-show`, `x-cloak`, `x-ref`, `@submit.prevent`, `@click`, `@change`, `:class` bindings
- Removed: `<script id="page-data">` JSON block
- Removed: Alpine sentinel, loading indicator, end-of-list divs
- Added: `hx-get="/"`, `hx-target="#search-results"`, `hx-swap="innerHTML"`, `hx-push-url="true"` on form
- Added: `hx-trigger="change"` + `hx-include="closest form"` on filter selects and checkbox
- Added: `#search-results` wrapper div
- Added: `{{template "load_more_sentinel" .}}` for initial sentinel
- Changed: search type from Alpine buttons to radio inputs with `has-[:checked]:` Tailwind classes
- Changed: clear search from Alpine button to `<a href="/">`
- Changed: selected states rendered server-side with `{{if eq ...}}selected{{end}}`

- [ ] **Step 2: Verify build**

Run: `cd /data/projects/github.com/lupguo/linkstash && go build ./cmd/server/`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add web/templates/index.html
git commit -m "refactor: replace Alpine bindings with HTMX in index.html"
```

---

### Task 6: Remove `urlListPage` from JS bundle

**Files:**
- Delete: `web/src/js/alpine/url-list.js`
- Modify: `web/src/js/app.js:12,21`

- [ ] **Step 1: Delete `url-list.js`**

```bash
rm web/src/js/alpine/url-list.js
```

- [ ] **Step 2: Remove imports from `app.js`**

Remove line 12 (`import { urlListPage }...`) and line 21 (`window.urlListPage = urlListPage;`) from `web/src/js/app.js`. The file should become:

```javascript
/**
 * LinkStash — Frontend Entry Point
 *
 * Vendor libraries and Alpine.js components bundled together.
 * Alpine must initialize AFTER our components are on window.
 */

// Vendor: htmx
import './vendor/htmx.min.js';

// Alpine components
import { urlCard } from './alpine/url-card.js';
import { detailPage } from './alpine/detail-page.js';
import { loginForm } from './alpine/login-form.js';

// Utilities
import { copyToClipboard } from './utils.js';

// Expose components to window for Alpine.js x-data bindings
window.urlCard = urlCard;
window.detailPage = detailPage;
window.loginForm = loginForm;
window.copyToClipboard = copyToClipboard;

// Alpine.js — import last. Its queueMicrotask(() => Alpine.start())
// runs after the current synchronous execution, but since esbuild
// hoists imports, we need to ensure our window assignments happen first.
// The solution: use dynamic import or defer Alpine's start.
// Since Alpine CDN auto-starts, we'll just import it and rely on the
// fact that all our window assignments above are synchronous and run
// before Alpine's DOMContentLoaded/microtask fires.
import './vendor/alpine.min.js';
```

- [ ] **Step 3: Rebuild frontend**

Run: `cd /data/projects/github.com/lupguo/linkstash && make frontend-js`
Expected: Build succeeds, `web/static/js/app.js` updated

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove urlListPage Alpine component from JS bundle"
```

---

### Task 7: Add HTMX config and 401 handler to layout

**Files:**
- Modify: `web/templates/layout.html`

- [ ] **Step 1: Add HTMX config meta tag in `<head>` (after the script tag, line 18)**

Add after line 18 (`<script src="/static/js/app.js?v={{.Version}}" defer></script>`):

```html
    <meta name="htmx-config" content='{"historyCacheSize": 10}'>
```

- [ ] **Step 2: Add 401 redirect handler on `<body>` tag**

Change line 20 from:

```html
<body class="bg-terminal-bg text-terminal-green font-mono min-h-screen flex flex-col" x-data="{ authenticated: document.cookie.includes('linkstash_token') }">
```

To:

```html
<body class="bg-terminal-bg text-terminal-green font-mono min-h-screen flex flex-col"
      x-data="{ authenticated: document.cookie.includes('linkstash_token') }"
      hx-on::response-error="if(event.detail.xhr.status===401) window.location='/login'">
```

- [ ] **Step 3: Verify build**

Run: `cd /data/projects/github.com/lupguo/linkstash && go build ./cmd/server/`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add web/templates/layout.html
git commit -m "feat: add HTMX history config and 401 redirect handler"
```

---

### Task 8: Full build and manual smoke test

- [ ] **Step 1: Full build**

Run: `cd /data/projects/github.com/lupguo/linkstash && make build`
Expected: Both frontend and server build without errors

- [ ] **Step 2: Start server**

Run: `cd /data/projects/github.com/lupguo/linkstash && make start`
Expected: Server starts, listening on configured port

- [ ] **Step 3: Verify initial page load**

Open browser to `http://localhost:8080/` (or configured port).
Expected:
- URL cards render correctly in grid layout
- Sentinel div visible at bottom (or end-of-list if fewer than page size)
- Search form renders with all filter controls
- No console errors

- [ ] **Step 4: Verify infinite scroll**

Scroll to bottom of page.
Expected:
- New cards load automatically when sentinel enters viewport
- "loading..." text shows briefly
- New sentinel appears for next page
- "// end of list" shows when all cards loaded
- No page refresh, URL doesn't change

- [ ] **Step 5: Verify search**

Type a search term and click "grep" or press Enter.
Expected:
- URL bar updates to `/?q=searchterm&search_type=keyword&page=1...`
- Card list replaces with search results
- "x" clear button appears
- minScore filter shows (if not already visible)
- Sentinel appears if results span multiple pages

- [ ] **Step 6: Verify filter changes**

Change category, sort, or size dropdown.
Expected:
- Results update without full page refresh
- URL bar updates with new filter params
- Sentinel resets for new result set

- [ ] **Step 7: Verify browser back button**

Navigate back after a search.
Expected: Previous view restores from HTMX history cache

- [ ] **Step 8: Verify clear search**

Click "x" button.
Expected: Full page navigates to `/`, all filters reset

- [ ] **Step 9: Stop server and commit if all passes**

```bash
make stop
```

No code commit here — this is a verification step. If issues found, fix in the relevant task's files and recommit.
