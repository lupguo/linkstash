# LinkStash: Alfred Workflow & Visit Tracking Report

## Executive Summary
This repository contains a **LinkStash** bookmark management system with:
1. **Alfred Workflow Integration** - Native macOS Alfred app integration for searching bookmarks
2. **Visit Tracking System** - Comprehensive tracking of user access patterns and analytics

---

## PART 1: ALFRED WORKFLOW FILES

### Directory Structure
```
extend_plugins/alfred/
├── LinkStash.alfredworkflow/
│   ├── info.plist                 # Workflow configuration
│   ├── lsearch.py                 # Python script filter for search
│   └── icon.png                   # Workflow icon (3092 bytes)
└── README.md                       # Setup and usage documentation
```

### File 1: README.md
**Path:** `extend_plugins/alfred/README.md`

**Purpose:** Installation and usage documentation for the Alfred workflow.

**Key Features:**
- **ls {query}** — Native Alfred search for bookmarks
  - Enter → Open URL in browser
  - ⌘+Enter → Copy URL to clipboard
  - ⌥+Enter → Open bookmark in LinkStash web UI
- **lsweb {query}** — Web search interface in browser

**Configuration Requirements:**
```
LINKSTASH_SERVER     = Server URL (e.g., http://localhost:8888)
LINKSTASH_SECRET_KEY = Authentication secret key
```

**Token Management:**
- Tokens cached at `~/.linkstash/token`
- Auto-refreshes on expiry
- Can be deleted to force re-authentication

---

### File 2: lsearch.py (Python Script Filter)
**Path:** `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py`
**Type:** Python 3 executable script (7047 bytes)

**Purpose:** Alfred Script Filter that handles bookmark search queries and outputs Alfred-compatible JSON.

**Key Components:**

#### 1. Configuration Section
```python
SERVER_URL = os.environ.get("LINKSTASH_SERVER", "").rstrip("/")
SECRET_KEY = os.environ.get("LINKSTASH_SECRET_KEY", "")
TOKEN_PATH = Path.home() / ".linkstash" / "token"
SEARCH_SIZE = 10        # Default number of results
TIMEOUT_SECONDS = 3     # API request timeout
```

#### 2. Token Management Functions
- `read_cached_token()` — Reads JWT from cache
- `save_token(token)` — Saves JWT with restricted permissions (mode 0o600)
- `delete_cached_token()` — Removes cached token
- `exchange_token()` — Exchanges secret_key for JWT via `/api/auth/token`
- `get_token()` — Gets valid JWT, using cache or requesting fresh

#### 3. Search Functions
- `search(query, token)` — Calls `/api/search` endpoint
  - Parameters: `q`, `type: keyword`, `size: 10`
  - Returns parsed JSON response
- `search_with_retry(query)` — Implements automatic token refresh on 401 errors

#### 4. Alfred Output Formatting
- `format_alfred_items(result, query)` — Converts API response to Alfred JSON format
- Returns items with:
  - `uid` — URL ID
  - `title` — Bookmark title
  - `subtitle` — Link with category, network type, and relevance score
  - `arg` — URL to open
  - `mods.cmd` — Copy to clipboard modifier
  - `mods.alt` — Open in LinkStash modifier

#### 5. Error Handling
- `alfred_error(title, subtitle)` — Returns error item to Alfred
- Handles HTTP errors (401, connection failures)
- Validates configuration on startup

#### 6. Main Execution Flow
```
1. Parse environment variables
2. Get or refresh JWT token
3. Call /api/search?q={query}&type=keyword&size=10
4. Format results as Alfred items
5. Output JSON to stdout
```

**Example Output:**
```json
{
  "items": [
    {
      "uid": "123",
      "title": "Example Bookmark",
      "subtitle": "https://example.com  [Technology]  (public)  score: 0.95",
      "arg": "https://example.com",
      "icon": {"path": "icon.png"},
      "mods": {
        "cmd": {
          "arg": "https://example.com",
          "subtitle": "Copy to clipboard: https://example.com"
        },
        "alt": {
          "arg": "http://localhost:8888/urls/123",
          "subtitle": "Open in LinkStash"
        }
      }
    }
  ]
}
```

---

### File 3: info.plist (Workflow Configuration)
**Path:** `extend_plugins/alfred/LinkStash.alfredworkflow/info.plist`
**Type:** XML Property List (7236 bytes)

**Workflow Metadata:**
```xml
<bundleid>com.linkstash.alfred</bundleid>
<category>Productivity</category>
<createdby>LinkStash</createdby>
<name>LinkStash Search</name>
<description>Search your LinkStash bookmarks from Alfred</description>
<version>1.0.0</version>
```

**Workflow Objects:**

#### 1. Input: Script Filter (ID: lsearch-scriptfilter)
- **Keyword:** `ls`
- **Command:** `python3 lsearch.py "{query}"`
- **Queue Mode:** 1 (Deferred)
- **Queue Delay:** 3 seconds custom
- **Running Subtext:** "Searching LinkStash..."
- **Escaping:** 102 (Normalize Diacritics)

#### 2. Actions (Output):
- **Open URL Action** (ID: open-url-action)
  - Default action: Opens URL in browser
  
- **Copy Action** (ID: copy-action)
  - ⌘+Enter modifier: Copy URL to clipboard
  
- **Open LinkStash Action** (ID: open-linkstash-action)
  - ⌥+Enter modifier: Opens `{LINKSTASH_SERVER}/urls/{id}` in browser

#### 3. Web Search Input (ID: linkstash-keyword)
- **Keyword:** `lsweb`
- **Subtext:** "Open LinkStash web search"
- **Action:** Opens `{var:LINKSTASH_SERVER}/?q={query}` in browser

**User Configuration:**
- `LINKSTASH_SERVER` — Server URL textfield (required)
- `LINKSTASH_SECRET_KEY` — Secret key textfield (required)

**Visual Layout:**
- Script filter at (100, 100)
- Actions positioned at (400, 100-300)
- Web search at (100, 400)

---

## PART 2: VISIT TRACKING SYSTEM

### Overview
The visit tracking system records every time a URL or short link is accessed, capturing IP address and user agent information.

### Architecture Diagram
```
HTTP Request
    ↓
Handler (shorturl_handler.go / url_handler.go)
    ↓
URLUsecase.RecordVisit(id)
    ↓
URLService.RecordVisit(id)
    ↓
URLRepo.IncrementVisit(id) + VisitService.RecordVisit()
    ↓
Database
    ├── t_urls (VisitCount++)
    └── t_visit_records (New Record)
```

### File 1: Visit Record Entity
**Path:** `app/domain/entity/visit_record.go`

**Structure:**
```go
type VisitRecord struct {
    ID        uint           // Primary key
    URLID     uint           // Associated URL ID (indexed)
    ShortID   uint           // Short link ID (indexed)
    IP        string         // Visitor IP address
    UserAgent string         // Browser user agent
    CreatedAt time.Time      // Auto-created timestamp
    UpdatedAt time.Time      // Auto-updated timestamp
    DeletedAt gorm.DeletedAt // Soft delete flag (indexed)
}

// Table name in database
TableName() → "t_visit_records"
```

### File 2: Visit Service (Domain Service)
**Path:** `app/domain/services/visit_service.go`

**Responsibility:** Business logic for recording visits.

**Methods:**
```go
// RecordURLVisit creates a visit record for a URL
func (s *VisitService) RecordURLVisit(urlID uint, ip, userAgent string) error

// RecordShortVisit creates a visit record for a short link
func (s *VisitService) RecordShortVisit(shortID uint, ip, userAgent string) error
```

### File 3: Visit Repository Implementation
**Path:** `app/infra/db/visit_repo_impl.go`

**Methods:**
```go
// Create inserts a new VisitRecord
func (r *VisitRepoImpl) Create(record *entity.VisitRecord) error

// ListByURLID returns paginated visit records for a URL
func (r *VisitRepoImpl) ListByURLID(urlID uint, page, size int) 
    ([]*entity.VisitRecord, int64, error)

// ListByShortID returns paginated visit records for a short link
func (r *VisitRepoImpl) ListByShortID(shortID uint, page, size int)
    ([]*entity.VisitRecord, int64, error)
```

### File 4: URL Service (Domain Service)
**Path:** `app/domain/services/url_service.go`

**Visit-Related Method:**
```go
// RecordVisit increments the visit counter for the URL with the given ID
func (s *URLService) RecordVisit(id uint) error {
    return s.urlRepo.IncrementVisit(id)
}
```

### File 5: Short URL Handler (API Endpoint)
**Path:** `app/handler/shorturl_handler.go` (Line 162-185)

**Endpoint:** `GET /s/:code` (PUBLIC - no auth required)

**Flow:**
```go
func (h *ShortURLHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
    code := chi.URLParam(r, "code")
    
    // 1. Resolve short code
    url, err := h.usecase.ResolveShortCode(code)
    
    // 2. Record visit ASYNCHRONOUSLY (non-blocking)
    go func() {
        _ = h.usecase.RecordVisit(url.ID)
    }()
    
    // 3. Redirect immediately
    http.Redirect(w, r, url.Link, http.StatusFound)
}
```

**Key Points:**
- Visit recording happens in background goroutine
- Non-blocking (redirect happens immediately)
- Error in visit recording doesn't affect redirect

### File 6: URL Handler (API Endpoint)
**Path:** `app/handler/url_handler.go` (Line 259-273)

**Endpoint:** `POST /api/urls/:id/visit` (PROTECTED - auth required)

```go
func (h *URLHandler) HandleVisit(w http.ResponseWriter, r *http.Request) {
    id, err := parseUintParam(r, "id")
    if err != nil {
        writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
        return
    }
    
    if err := h.usecase.RecordVisit(id); err != nil {
        writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
        return
    }
    
    writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
```

### File 7: URL Usecase (Application Layer)
**Path:** `app/application/url_usecase.go` (Line 45-48)

```go
// RecordVisit increments the visit counter for the URL
func (uc *URLUsecase) RecordVisit(id uint) error {
    return uc.urlService.RecordVisit(id)
}
```

### Visit Tracking Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│ User accesses short link: GET /s/abc123                    │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────┐
        │ ShortURLHandler.HandleRedirect │
        └────────────────┬───────────────┘
                         │
                         ├─ Resolve code: /s/abc123 → URL(ID=42)
                         │
                         ├─ [ASYNC] Record visit
                         │       │
                         │       ├─ URLUsecase.RecordVisit(42)
                         │       │
                         │       ├─ URLService.RecordVisit(42)
                         │       │
                         │       ├─ URLRepo.IncrementVisit(42)
                         │       │   └─ UPDATE t_urls SET visit_count = visit_count + 1
                         │       │
                         │       └─ VisitService.RecordShortVisit(42, ip, ua)
                         │           └─ INSERT INTO t_visit_records VALUES (...)
                         │
                         └─ Immediately redirect to long URL
```

---

## API ENDPOINTS SUMMARY

### Authentication
- **POST /api/auth/token** — Exchange secret_key for JWT
  - Request: `{"secret_key": "..."}`
  - Response: `{"token": "jwt..."}`

### Search
- **GET /api/search?q={query}&type=keyword&size=10** — Search bookmarks
  - Headers: `Authorization: Bearer {jwt}`
  - Response: `{"data": [...]}`

### URL Management
- **POST /api/urls** — Create new URL
  - Request: `{"link": "..."}`
  
- **GET /api/urls** — List URLs
  - Query: `?page=1&size=20&sort=time&category=&tags=&is_shorturl=0&network_type=`
  
- **GET /api/urls/:id** — Get URL details
  
- **PUT /api/urls/:id** — Update URL metadata
  
- **DELETE /api/urls/:id** — Soft delete URL
  
- **POST /api/urls/:id/visit** — Record visit (Protected)
  - Response: `{"status": "ok"}`

### Short Links
- **POST /api/short-links** — Create short link
  - Request: `{"long_url": "...", "code": "optional", "ttl": "7d"}`
  
- **GET /api/short-links** — List short links
  
- **GET /s/:code** — Redirect short link (PUBLIC, visit recorded)
  
- **PUT /api/short-links/:id** — Update short link
  
- **DELETE /api/short-links/:id** — Clear short code

---

## Technology Stack

### Alfred Integration
- **Language:** Python 3
- **Dependencies:** Standard library only (urllib, json, os, sys, pathlib)
- **Configuration:** Environment variables
- **Token Storage:** Local file system (~/.linkstash/token)
- **API Communication:** HTTP/HTTPS

### Visit Tracking Backend
- **Language:** Go
- **Database:** GORM (ORM)
- **Concurrency:** Goroutines for non-blocking operations
- **Data Capture:** IP address and User-Agent from HTTP requests

---

## File Summary Table

| File | Type | Size | Purpose |
|------|------|------|---------|
| `extend_plugins/alfred/README.md` | Markdown | ~1.5KB | Setup documentation |
| `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py` | Python3 | 7047B | Script filter script |
| `extend_plugins/alfred/LinkStash.alfredworkflow/info.plist` | XML | 7236B | Workflow configuration |
| `extend_plugins/alfred/LinkStash.alfredworkflow/icon.png` | PNG | 3092B | Workflow icon |
| `app/domain/entity/visit_record.go` | Go | ~600B | Visit data model |
| `app/domain/services/visit_service.go` | Go | ~900B | Visit business logic |
| `app/domain/services/url_service.go` | Go | ~8KB | URL business logic |
| `app/infra/db/visit_repo_impl.go` | Go | ~1.5KB | Visit repository |
| `app/handler/shorturl_handler.go` | Go | ~6KB | Short URL API endpoints |
| `app/handler/url_handler.go` | Go | ~10KB | URL API endpoints |
| `app/application/url_usecase.go` | Go | ~2KB | Application orchestration |

---

## Key Security Notes

1. **Alfred Workflow**
   - Secret keys stored in environment variables (workflow settings)
   - JWT tokens cached with restricted permissions (0o600)
   - Timeout protection (3 seconds)

2. **Visit Tracking**
   - Short link redirects are PUBLIC (no auth)
   - Visits recorded asynchronously (non-blocking)
   - IP and User-Agent captured automatically
   - Soft deletes preserve audit trail

3. **Database**
   - Visit records indexed on URLID and ShortID
   - Soft delete flag for data retention
   - Timestamps auto-managed by GORM

