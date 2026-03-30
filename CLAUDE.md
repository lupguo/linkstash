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

Server requires project root `.env` with `AUTH_SECRET_KEY`, `JWT_SECRET`, `MYSQL_PASSWORD`, `OPENROUTER_API_KEY` (see `.env.example`). Config: `conf/app_dev.yaml` (uses `${VAR}` env interpolation).

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

## Web UI (Preact SPA)

```
web/templates/spa.html           → Single HTML shell (serves all routes)
web/src/js/app.jsx               → Preact entry point (Router + Layout)
web/src/js/api.js                → JSON API client (fetch wrapper with JWT auth)
web/src/js/store.js              → Shared state (@preact/signals)
web/src/js/utils.js              → Utilities (getCookie, copyToClipboard)
web/src/js/pages/                → Page components (LoginPage, IndexPage, DetailPage)
web/src/js/components/           → Shared components (Layout, URLCard, SearchBar, ScoreFilter, ColorPicker)
web/src/css/app.css              → Tailwind entry
web/static/                      → Built assets (served at /static/)
```

**SPA architecture**: Go server serves `spa.html` for all non-API, non-static routes via `r.NotFound()`. Preact handles client-side routing with `preact-router`. All data flows through JSON APIs (`/api/*`). Auth uses JWT stored in `linkstash_token` cookie.

**Key frontend patterns**:
- Infinite scroll: IntersectionObserver on sentinel div, increments page state
- Search: fetches `/api/search` with query params, renders client-side
- ESC key: clears search query, resets filters, returns to default URL list
- Score filter: client-side min_score slider for hybrid search results
- State: `@preact/signals` for auth, `useState`/`useEffect` for component state

## API Routes

- `POST /api/auth/token` — get JWT (body: `{"secret_key":"..."}`)
- `GET/POST/PUT/DELETE /api/urls[/{id}]` — CRUD (JWT required)
- `GET /api/search?q=...&type=keyword|semantic|hybrid` — search
- `POST /api/short-links` — create short URL
- `GET /s/{code}` — short URL redirect
- `GET /* (non-API)` — SPA catch-all (serves spa.html)

## Key Conventions

- Go: standard `slog` for logging, `chi` for routing, `gorm` for ORM (SQLite/MySQL)
- Frontend: Preact SPA with preact-router and @preact/signals; esbuild bundles `web/src/js/app.jsx` → `web/static/js/app.js` (with `--jsx=automatic --jsx-import-source=preact`); Tailwind builds `web/src/css/app.css` → `web/static/css/app.css`
- npm: `package.json` manages preact deps; `node_modules/` is gitignored; run `npm install` after clone
- Config: YAML in `conf/`, env vars interpolated with `${VAR}` syntax
- DI: Google Wire — edit `app/di/wire.go`, run `make wire` to regenerate
- Search: Bleve full-text index + OpenRouter embedding for semantic search
- URL analysis: async worker fetches page via headless Chrome (rod), sends to LLM for title/keywords/description/category extraction
