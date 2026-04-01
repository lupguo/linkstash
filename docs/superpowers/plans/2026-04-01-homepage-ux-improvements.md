# Homepage UX Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve homepage UX with 4 changes: default sort to latest, Analyze button on cards, mobile nav optimization, iOS search zoom fix.

**Architecture:** Pure frontend changes across 5 files. No backend modifications needed — existing `POST /api/urls/{id}/reanalyze` and `urlApi.reanalyze(id)` already exist. All changes are in Preact JSX components and CSS/HTML.

**Tech Stack:** Preact, Tailwind CSS v4, esbuild

**Spec:** `docs/superpowers/specs/2026-04-01-homepage-ux-improvements-design.md`

---

### Task 1: Change Default Sort to Latest

**Files:**
- Modify: `web/src/js/pages/IndexPage.jsx:15` (default sort state)
- Modify: `web/src/js/pages/IndexPage.jsx:137` (ESC reset sort)
- Modify: `web/src/js/components/SearchBar.jsx:25` (Clear handler sort reset)
- Modify: `web/src/js/components/SearchBar.jsx:37` (active filter count baseline)

- [ ] **Step 1: Change default sort state in IndexPage.jsx**

In `web/src/js/pages/IndexPage.jsx`, line 15, change the default `sort` state:

```jsx
// Before:
const [sort, setSort] = useState('weight');

// After:
const [sort, setSort] = useState('latest');
```

- [ ] **Step 2: Update ESC key reset to match new default**

In `web/src/js/pages/IndexPage.jsx`, line 137 inside the ESC handler, change:

```jsx
// Before:
setSort('weight');

// After:
setSort('latest');
```

- [ ] **Step 3: Update SearchBar Clear handler to match new default**

In `web/src/js/components/SearchBar.jsx`, inside `handleClear()` (line 25), change:

```jsx
// Before:
onFilterChange({
  category: '',
  sort: 'weight',
  size: 100,
  isShortURL: false,
  minScore: 0.6,
  searchType: 'keyword',
});

// After:
onFilterChange({
  category: '',
  sort: 'latest',
  size: 100,
  isShortURL: false,
  minScore: 0.6,
  searchType: 'keyword',
});
```

- [ ] **Step 4: Update active filter count baseline**

In `web/src/js/components/SearchBar.jsx`, line 37, the active filter count treats `sort !== 'weight'` as active. Change to:

```jsx
// Before:
sort !== 'weight',

// After:
sort !== 'latest',
```

- [ ] **Step 5: Build and verify**

Run: `make frontend-js`
Expected: Build succeeds.

Manual verification: Open browser, confirm URL list loads sorted by newest first (created_at DESC). Confirm Sort dropdown in Filters shows "Latest" as selected. Confirm ESC key resets sort to "Latest".

- [ ] **Step 6: Commit**

```bash
git add web/src/js/pages/IndexPage.jsx web/src/js/components/SearchBar.jsx
git commit -m "feat: change default sort order to latest (created_at DESC)"
```

---

### Task 2: Add Analyze Button to URLCard

**Files:**
- Modify: `web/src/js/components/URLCard.jsx:3` (add useState import — already imported)
- Modify: `web/src/js/components/URLCard.jsx:4` (add urlApi.reanalyze import — already imported)
- Modify: `web/src/js/components/URLCard.jsx` (add import for urlListVersion from store)
- Modify: `web/src/js/components/URLCard.jsx:33-34` (add analyzing state)
- Modify: `web/src/js/components/URLCard.jsx:56-69` (add handleAnalyze function before handleDelete)
- Modify: `web/src/js/components/URLCard.jsx:138-167` (add Analyze button between Edit and Del)

- [ ] **Step 1: Add urlListVersion import**

In `web/src/js/components/URLCard.jsx`, at the top imports, add:

```jsx
import { urlListVersion } from '../store.js';
```

- [ ] **Step 2: Add analyzing state**

In `web/src/js/components/URLCard.jsx`, after line 34 (`const [showDeleteModal, setShowDeleteModal] = useState(false);`), add:

```jsx
const [analyzing, setAnalyzing] = useState(false);
```

- [ ] **Step 2: Add handleAnalyze function**

In `web/src/js/components/URLCard.jsx`, after the `handleEdit` function (after line 54), add:

```jsx
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
```

- [ ] **Step 3: Add Analyze button to hover action bar**

In `web/src/js/components/URLCard.jsx`, in the hover action bar div, add the Analyze button between Edit and Del buttons. The full action bar becomes:

```jsx
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
```

- [ ] **Step 4: Build and verify**

Run: `make frontend-js`
Expected: Build succeeds.

Manual verification: Hover over a URL card, confirm 5 buttons appear in order: Visit / Copy / Edit / Analyze / Del. Click Analyze — confirm spinner appears, then button text returns after API call completes. Card status should change to "analyzing" badge.

- [ ] **Step 5: Commit**

```bash
git add web/src/js/components/URLCard.jsx
git commit -m "feat: add Analyze button to URLCard hover actions"
```

---

### Task 3: Optimize Mobile Navigation

**Files:**
- Modify: `web/src/js/components/Layout.jsx:25-31` (responsive nav buttons)

- [ ] **Step 1: Update nav buttons with responsive text and spacing**

In `web/src/js/components/Layout.jsx`, replace the button container div (lines 25-31):

```jsx
// Before:
<div class="flex items-center gap-2">
  {isAuthenticated.value ? (
    <>
      <a href="/" class="btn px-3.5 py-1.5 no-underline text-sm">Home</a>
      <a href="/urls/new" class="btn btn-primary px-3.5 py-1.5 no-underline text-sm">+ New Link</a>
      <a href="#" onClick={handleLogout} class="btn btn-danger px-3.5 py-1.5 no-underline text-sm">Logout</a>
    </>
  ) : (
    <a href="/login" class="btn px-3.5 py-1.5 no-underline text-sm">Login</a>
  )}
</div>

// After:
<div class="flex items-center gap-1 sm:gap-2">
  {isAuthenticated.value ? (
    <>
      <a href="/" class="btn px-2 py-1 sm:px-3.5 sm:py-1.5 no-underline text-xs sm:text-sm">Home</a>
      <a href="/urls/new" class="btn btn-primary px-2 py-1 sm:px-3.5 sm:py-1.5 no-underline text-xs sm:text-sm whitespace-nowrap">+ New<span class="hidden sm:inline"> Link</span></a>
      <a href="#" onClick={handleLogout} class="btn btn-danger px-2 py-1 sm:px-3.5 sm:py-1.5 no-underline text-xs sm:text-sm">Logout</a>
    </>
  ) : (
    <a href="/login" class="btn px-2 py-1 sm:px-3.5 sm:py-1.5 no-underline text-xs sm:text-sm">Login</a>
  )}
</div>
```

Key changes:
- Gap: `gap-2` → `gap-1 sm:gap-2`
- Button padding: `px-3.5 py-1.5` → `px-2 py-1 sm:px-3.5 sm:py-1.5`
- Font size: `text-sm` → `text-xs sm:text-sm`
- "+ New Link" text: `+ New<span class="hidden sm:inline"> Link</span>` (shows "+ New" on mobile, "+ New Link" on sm+)
- Added `whitespace-nowrap` to the New Link button

- [ ] **Step 2: Build and verify**

Run: `make frontend-js`
Expected: Build succeeds.

Manual verification: Open browser DevTools, toggle iPhone 15 Pro (393px) viewport. Confirm all 3 nav buttons fit on one line without wrapping. Button shows "+ New" on mobile, "+ New Link" on desktop. Spacing is tighter but readable.

- [ ] **Step 3: Commit**

```bash
git add web/src/js/components/Layout.jsx
git commit -m "fix: prevent mobile nav button text wrapping on small screens"
```

---

### Task 4: Fix iOS Safari Search Input Zoom

**Files:**
- Modify: `web/templates/spa.html:4` (viewport meta tag)
- Modify: `web/src/css/app.css:66-67` (add mobile font-size for .input)

- [ ] **Step 1: Update viewport meta tag**

In `web/templates/spa.html`, line 4, update the viewport meta tag:

```html
<!-- Before: -->
<meta name="viewport" content="width=device-width, initial-scale=1.0">

<!-- After: -->
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0">
```

- [ ] **Step 2: Add mobile font-size override for .input**

In `web/src/css/app.css`, after the `.input` block's closing brace (after line 78), add a mobile media query:

```css
  /* iOS Safari auto-zooms inputs with font-size < 16px */
  @media (max-width: 640px) {
    .input {
      font-size: 16px;
    }
  }
```

This should be placed inside the `@layer components` block, right after the `.input` definition.

- [ ] **Step 3: Build and verify**

Run: `make frontend`
Expected: Both CSS and JS build succeed.

Manual verification: Open Safari on iPhone (or iOS simulator). Tap the search input — page should NOT zoom in. Verify the input text is readable at 16px.

- [ ] **Step 4: Commit**

```bash
git add web/templates/spa.html web/src/css/app.css
git commit -m "fix: prevent iOS Safari auto-zoom on search input focus"
```

---

### Task 5: Final Build and Smoke Test

**Files:** None (verification only)

- [ ] **Step 1: Full build**

Run: `make build`
Expected: All builds succeed (frontend CSS + JS + server).

- [ ] **Step 2: Run tests**

Run: `make test`
Expected: All Go tests pass.

- [ ] **Step 3: Smoke test (if server environment available)**

Run: `make smoke-test`
Expected: Build → start → test → stop all succeed.

- [ ] **Step 4: Final commit (if any build artifact updates needed)**

Only if `web/static/` built files need to be committed:

```bash
git add web/static/
git commit -m "chore: update built frontend assets"
```
