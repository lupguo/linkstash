# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
make build          # Frontend (CSS+JS) + server + CLI
make build-server   # Server only (faster iteration)
make frontend       # CSS + JS only
make frontend-js    # JS only (esbuild bundle)
make dev-frontend   # Watch mode for CSS + JS

make start          # Build + start server in background (port 8080)
make stop           # Stop background server
make restart        # Stop + start

make test           # go test -v -race ./...
make smoke-test     # Full smoke test (build→start→test→stop)
make lint           # golangci-lint
make fmt            # gofmt -s -w .
make wire           # Regenerate Wire DI code (after changing providers)
```

Server requires `~/.my.env` with `OPENROUTER_API_KEY` for LLM features. Config: `conf/app_dev.yaml`. Auth secret: `clark`.

## Architecture

**DDD + Clean Architecture** with Wire DI.

```
cmd/server/main.go          → chi router, routes, graceful shutdown
app/di/                      → Wire DI (wire.go → wire_gen.go), App struct
app/handler/                 → HTTP handlers (REST API + web pages)
app/application/             → Usecases (url, search, analysis)
app/domain/entity/           → GORM models (URL, ShortLink)
app/domain/repos/            → Repository interfaces
app/domain/services/         → Domain services (visit tracking)
app/infra/                   → Infrastructure (db/SQLite, llm/OpenRouter, search/bleve, browser/rod, config)
app/middleware/               → JWT auth middleware
```

**Request flow**: `chi router → handler → usecase → repo/service → infra`

## Web UI (Go templates + HTMX + Alpine.js)

```
web/templates/layout.html    → Base layout (nav, footer, script/css includes)
web/templates/{page}.html    → Page templates (index, detail, login) — {{define "content"}}
web/components/*.html        → Shared partials (url_card, card_fragment, load_more_sentinel)
web/src/js/app.js            → esbuild entry: htmx + Alpine components
web/src/js/alpine/           → Alpine components (url-card, detail-page, login-form)
web/src/css/app.css          → Tailwind entry
web/static/                  → Built assets (served at /static/)
```

**Template loading**: `NewWebHandler` parses each page template with layout + all components via `filepath.Glob`. Components define `{{define "block_name"}}` blocks usable from any page. Access via `h.tmplMap["index"]` → `t.ExecuteTemplate(w, "block_name", data)`.

**HTMX patterns** (index page):
- Infinite scroll: sentinel with `hx-trigger="revealed"` + `hx-swap="outerHTML"`, server returns OOB cards (`hx-swap-oob="beforeend:#url-list"`) + new sentinel
- Search/filters: form `hx-get="/"` + `hx-push-url="true"`, server detects `HX-Request` header → returns `search_fragment` (inline, no OOB)
- Filter auto-submit: `onchange="this.form.requestSubmit()"` triggers HTMX form submission

## API Routes

- `POST /api/auth/token` — get JWT (body: `{"secret_key":"..."}`)
- `GET/POST/PUT/DELETE /api/urls[/{id}]` — CRUD (JWT required)
- `GET /api/search?q=...&type=keyword|semantic|hybrid` — search
- `POST /api/short-links` — create short URL
- `GET /s/{code}` — short URL redirect
- `GET /cards?page=N&size=N&sort=...` — HTMX scroll fragment

## Key Conventions

- Go: standard `slog` for logging, `chi` for routing, `gorm` for ORM (SQLite)
- Frontend: esbuild bundles `web/src/js/app.js` → `web/static/js/app.js`; Tailwind builds `web/src/css/app.css` → `web/static/css/app.css`
- Config: YAML in `conf/`, env vars interpolated with `${VAR}` syntax
- DI: Google Wire — edit `app/di/wire.go`, run `make wire` to regenerate
- Search: Bleve full-text index + OpenRouter embedding for semantic search
- URL analysis: async worker fetches page via headless Chrome (rod), sends to LLM for title/keywords/description/category extraction
