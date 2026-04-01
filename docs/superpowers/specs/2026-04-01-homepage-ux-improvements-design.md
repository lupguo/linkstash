# Homepage UX Improvements Design

**Date:** 2026-04-01
**Scope:** 4 frontend changes — default sort, Analyze button, mobile nav, iOS zoom fix

## Changes

### 1. Default Sort Order → Latest

**Current:** `useState('weight')` — URLs sorted by combined weight (auto_weight + manual_weight) descending.
**New:** `useState('time')` — URLs sorted by `created_at` descending (newest first).

**Files affected:**
- `web/src/js/pages/IndexPage.jsx` — change default `sort` state from `'weight'` to `'time'`
- `web/src/js/components/SearchBar.jsx` — sync Sort dropdown default to match

**Backend:** No change needed. The `GET /api/urls` endpoint already supports `sort=time` (maps to `ORDER BY created_at DESC`).

### 2. Add Analyze Button to URLCard Hover Actions

**Current action bar order:** Visit → Copy → Edit → Del
**New action bar order:** Visit → Copy → Edit → **Analyze** → Del

**Button spec:**
- Label: "Analyze" (7 chars, consistent with other button lengths)
- Style: Same as Visit/Copy/Edit — `bg-slate-700 text-slate-200`, no accent color
- Click behavior: Triggers `POST /api/urls/{id}/reanalyze` (re-analyze regardless of prior state)
- Loading state: Replace button text with spinner while analysis runs
- Completion: Refresh card data to show updated title/description/category/keywords

**Files affected:**
- `web/src/js/components/URLCard.jsx` — add Analyze button between Edit and Del
- `web/src/js/api.js` — already has `reanalyze(id)` method, no changes needed

**Backend:** Existing endpoint `POST /api/urls/{id}/reanalyze` (HandleReanalyze) resets status, clears LLM fields, and re-enqueues for analysis. No backend changes needed.

### 3. Mobile Navigation Optimization

**Problem:** On iPhone 15 Pro (393px width), "+ New Link" button text wraps to two lines due to insufficient horizontal space.

**Fix (sm breakpoint, < 640px):**
- Shorten button text: "+ New Link" → "+ New"
- Reduce button padding: `px-3.5 py-1.5` → `px-2 py-1`
- Reduce button gap: `gap-2` → `gap-1`

**Implementation approach:** Use responsive Tailwind classes or a conditional render based on viewport. Preferred: use `<span class="hidden sm:inline">Link</span>` pattern so the word "Link" hides on small screens without JS.

**Files affected:**
- `web/src/js/components/Layout.jsx` — responsive button text and spacing

### 4. iOS Safari Search Input Zoom Fix

**Problem:** On iOS Safari, tapping the search input triggers automatic page zoom because the input font-size is below 16px. The zoom persists after defocusing.

**Fix (two-pronged):**
1. **CSS:** Set `.input` font-size to `16px` on mobile (`@media (max-width: 640px)`)
2. **Viewport meta:** Add `maximum-scale=1` to the viewport meta tag in `spa.html`

**Files affected:**
- `web/templates/spa.html` — update `<meta name="viewport">` tag
- `web/src/css/app.css` — add mobile font-size override for `.input` class

## Files Summary

| File | Changes |
|------|---------|
| `web/src/js/pages/IndexPage.jsx` | Default sort `'weight'` → `'time'` |
| `web/src/js/components/SearchBar.jsx` | Sort dropdown default sync |
| `web/src/js/components/URLCard.jsx` | Add Analyze button (same style, spinner loading) |
| `web/src/js/api.js` | No changes needed (already has `reanalyze(id)`) |
| `web/src/js/components/Layout.jsx` | Mobile-responsive nav text and spacing |
| `web/templates/spa.html` | Viewport `maximum-scale=1` |
| `web/src/css/app.css` | Mobile `.input` font-size 16px |

## Out of Scope

- Backend API changes (all needed endpoints exist)
- Desktop layout changes (only mobile affected for items 3 & 4)
- Analysis result display redesign (existing card refresh is sufficient)
