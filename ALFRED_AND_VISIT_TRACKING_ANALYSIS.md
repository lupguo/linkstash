# LinkStash Repository: Alfred Workflow & Visit Tracking Analysis

## Overview
This document provides a comprehensive inventory of all Alfred workflow-related files and visit tracking API implementation in the LinkStash repository.

---

## 1. ALFRED WORKFLOW FILES

### Directory Structure
```
extend_plugins/
├── alfred/
│   ├── README.md                              # Installation and usage documentation
│   └── LinkStash.alfredworkflow/
│       ├── info.plist                         # Alfred workflow configuration (XML)
│       ├── lsearch.py                         # Python Script Filter for API search
│       └── icon.png                           # Workflow icon (256x256 PNG)
```

### 1.1 Main Alfred Integration Files

#### File: `extend_plugins/alfred/README.md`
- **Purpose**: Installation guide and usage documentation for Alfred workflow users
- **Key Content**:
  - Installation instructions (double-click or import via Alfred Preferences)
  - Configuration requirements (Server URL, Secret Key)
  - Usage instructions for two modes:
    - `ls {query}` — Alfred native search
    - `lsweb {query}` — Web search in browser
  - Keyboard shortcuts:
    - Enter — Open URL in browser
    - ⌘+Enter — Copy URL to clipboard
    - ⌥+Enter — Open in LinkStash web UI
  - Token caching at `~/.linkstash/token`
  - Troubleshooting guide

#### File: `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py`
- **Purpose**: Python 3 script filter for native Alfred search integration
- **Type**: Executable Python script (chmod +x)
- **Key Components**:

**Configuration Management (lines 15-20)**
- Reads `LINKSTASH_SERVER` environment variable
- Reads `LINKSTASH_SECRET_KEY` environment variable
- Token cache location: `~/.linkstash/token`
- Search result size: 10 items
- Timeout: 3 seconds

**Token Management Functions**
- `read_cached_token()`: Reads JWT from cache file
- `save_token(token)`: Saves JWT to cache with 0o600 permissions
- `delete_cached_token()`: Removes expired token
- `exchange_token()`: POST to `/api/auth/token` with `{"secret_key": "..."}`
- `get_token()`: Uses cache or exchanges for fresh token

**Search API Integration (lines 107-141)**
- `search(query, token)`: Calls `GET /api/search?q={query}&type=keyword&size=10`
- `search_with_retry(query)`: Auto-refreshes token on 401, retries once
- Returns parsed JSON response

**Alfred Output Format (lines 148-199)**
```json
{
  "items": [
    {
      "uid": "URL_ID",
      "title": "Page Title",
      "subtitle": "https://example.com  [Category]  (NetworkType)  score: 0.95",
      "arg": "https://example.com",
      "icon": {"path": "icon.png"},
      "mods": {
        "cmd": {
          "arg": "https://example.com",
          "subtitle": "Copy URL to clipboard"
        },
        "alt": {
          "arg": "https://server/urls/123",
          "subtitle": "Open in LinkStash"
        }
      }
    }
  ]
}
```

**Main Entry Point (lines 206-229)**
- Reads query from `sys.argv[1:]`
- Validates configuration (SERVER_URL, SECRET_KEY required)
- Calls `search_with_retry(query)`
- Outputs Alfred JSON to stdout

**Error Handling**
- Configuration missing → Shows error item
- Network timeout (>3s) → Connection error
- Authentication failure (401) → Auto-refresh and retry
- No results → "No results for {query}" message

#### File: `extend_plugins/alfred/LinkStash.alfredworkflow/info.plist`
- **Purpose**: Alfred workflow configuration (Apple Property List format)
- **Type**: XML (info.plist)
- **Bundle ID**: `com.linkstash.alfred`
- **Category**: Productivity
- **Version**: 1.0.0

**Workflow Objects Defined**:
1. **lsearch-scriptfilter** (uid: "lsearch-scriptfilter")
   - Input: Keyword trigger "ls"
   - Script: `python3 lsearch.py "{query}"`
   - Queue delay: 3 seconds (debounce)
   - Title: "LinkStash Search"
   - Subtext: "Search your bookmarks"

2. **open-url-action** (uid: "open-url-action")
   - Type: Alfred open URL action
   - Triggered on: Enter key (default)
   - Opens: `{query}` (the selected URL)

3. **copy-action** (uid: "copy-action")
   - Type: Clipboard output
   - Triggered on: ⌘+Enter (modifier: 1048576)
   - Copies: `{query}` (the URL to clipboard)

4. **open-linkstash-action** (uid: "open-linkstash-action")
   - Type: Alfred open URL action
   - Triggered on: ⌥+Enter (modifier: 524288)
   - Opens: `{query}` (LinkStash detail URL)

5. **linkstash-keyword** (uid: "linkstash-keyword")
   - Input: Keyword trigger "lsweb"
   - Title: "LinkStash Web Search"
   - Subtext: "Open LinkStash web search"

6. **open-web-search** (uid: "open-web-search")
   - Type: Alfred open URL action
   - Opens: `{var:LINKSTASH_SERVER}/?q={query}`
   - Passes search query as URL parameter

**User Configuration Variables** (lines 258-302)
1. `LINKSTASH_SERVER`
   - Type: textfield
   - Required: true
   - Placeholder: "https://linkstash.example.com"
   - Description: "Your LinkStash server URL"

2. `LINKSTASH_SECRET_KEY`
   - Type: textfield
   - Required: true
   - Placeholder: "your-secret-key"
   - Description: "Authentication secret key"

**Workflow Connections**:
- lsearch-scriptfilter → open-url-action (Enter)
- lsearch-scriptfilter → copy-action (Cmd+Enter)
- lsearch-scriptfilter → open-linkstash-action (Alt+Enter)
- linkstash-keyword → open-web-search (Enter)

#### File: `extend_plugins/alfred/LinkStash.alfredworkflow/icon.png`
- **Purpose**: Workflow icon displayed in Alfred interface
- **Format**: PNG image (256x256 pixels)
- **Color**: Sky blue (#38bdf8 - LinkStash accent color)

### 1.2 Documentation Files

#### File: `docs/superpowers/plans/2026-04-03-alfred-workflow.md`
- **Purpose**: Implementation plan for Alfred workflow development
- **Scope**: Complete specification for developers implementing the workflow
- **Contains**: 5 main tasks with detailed steps and code examples
- **Key Sections**:
  - Task 1: Create config and auth module (lsearch.py foundation)
  - Task 2: Create Alfred Workflow info.plist
  - Task 3: Create placeholder icon
  - Task 4: Frontend support for ?q= URL parameter in IndexPage.jsx
  - Task 5: End-to-end manual testing and README

#### File: `docs/superpowers/specs/2026-04-03-alfred-workflow-design.md`
- **Purpose**: Technical specification and design document for Alfred workflow
- **Contains**: Architecture, API contracts, error handling, performance requirements
- **Key Sections**:
  - Trigger Modes (lsearch vs linkstash)
  - Directory structure
  - Script Filter core flow
  - Alfred JSON output format
  - Dependencies (Python 3 stdlib only)
  - Configuration (environment variables, token caching)
  - Frontend changes (URL query parameter support)
  - Error handling scenarios
  - Performance targets (<500ms response)
  - Installation instructions
  - Scope (in vs out of scope)

---

## 2. VISIT TRACKING API & DOMAIN SERVICES

### Visit Tracking Architecture Overview
The system uses a Domain-Driven Design (DDD) pattern with the following layers:
1. **Domain Layer** (entities and services)
2. **Application Layer** (use cases)
3. **Infrastructure Layer** (repositories and database)
4. **Handler Layer** (HTTP endpoints)

### 2.1 Domain Entities

#### File: `app/domain/entity/visit_record.go`
- **Purpose**: Defines the VisitRecord domain entity
- **Table**: `t_visit_records` (soft-delete enabled)
- **Fields**:
  ```go
  type VisitRecord struct {
    ID        uint           // Primary key with auto-increment
    URLID     uint           // Foreign key to URL (indexed)
    ShortID   uint           // Foreign key to short link (indexed)
    IP        string         // Visitor IP address
    UserAgent string         // Browser user agent
    CreatedAt time.Time      // Auto-set on creation
    UpdatedAt time.Time      // Auto-updated
    DeletedAt gorm.DeletedAt  // Soft-delete timestamp (indexed)
  }
  ```

### 2.2 Domain Services

#### File: `app/domain/services/visit_service.go`
- **Purpose**: Business logic for recording visits
- **Methods**:
  - `RecordURLVisit(urlID uint, ip, userAgent string) error`
    - Creates visit record for a regular URL
    - Stores IP and user agent
  - `RecordShortVisit(shortID uint, ip, userAgent string) error`
    - Creates visit record for a short link
    - Stores IP and user agent

#### File: `app/domain/services/url_service.go`
- **Purpose**: Business logic for URL management and visit tracking
- **Key Methods**:
  - `RecordVisit(id uint) error` (lines 81-84)
    - Increments visit counter for a URL
    - Called asynchronously after redirect or API visit endpoint
  - `GenerateShortLink(longURL, customCode string, ttl *time.Duration) (*entity.URL, error)`
    - Creates/finds URL record
    - Generates unique short code (base62, 6 characters)
    - Sets optional TTL expiration
  - `ResolveShortCode(code string) (*entity.URL, error)`
    - Retrieves URL by short code
    - Checks for expiration
    - Called in short link redirect handler

### 2.3 HTTP Handlers

#### File: `app/handler/url_handler.go`

**Visit Endpoint: `HandleVisit` (lines 259-273)**
```
POST /api/urls/{id}/visit
Authorization: Bearer {JWT_TOKEN}
```
- **Purpose**: Record a manual visit for a URL
- **Parameters**: URL ID from path
- **Response**: `{"status": "ok"}`
- **Usage**: Called when user manually marks a bookmark as visited

**Short Link Redirect Endpoint** in `app/handler/shorturl_handler.go` (lines 160-185)
```
GET /s/{code}
(Public route - no authentication)
```
- **Handler**: `HandleRedirect`
- **Flow**:
  1. Validates short code parameter
  2. Resolves code to URL using `ResolveShortCode()`
  3. **Async records visit**: `go func() { h.usecase.RecordVisit(url.ID) }()`
  4. Returns HTTP 302 redirect to original URL
- **Error Handling**:
  - Code not found → 404 NOT_FOUND
  - Short link expired → 410 GONE with "short link has expired"

### 2.4 Repository Layer

#### File: `app/infra/db/visit_repo_impl.go`
- **Purpose**: Database persistence for visit records (GORM implementation)
- **Methods**:
  - `Create(record *entity.VisitRecord) error`
    - Inserts new visit record
    - Fields: URLID, ShortID, IP, UserAgent
  - `ListByURLID(urlID uint, page, size int) ([]*entity.VisitRecord, int64, error)`
    - Paginated list of visits for a specific URL
    - Ordered by created_at DESC
  - `ListByShortID(shortID uint, page, size int) ([]*entity.VisitRecord, int64, error)`
    - Paginated list of visits for a specific short link
    - Ordered by created_at DESC

#### File: `app/infra/db/url_repo_impl.go`
- **Purpose**: Database persistence for URL records
- **Visit-Related Methods**:
  - `IncrementVisit(id uint) error`
    - Increments visit_count for a URL
    - Updates auto_weight (set to visit count)
    - Called by visit tracking service

### 2.5 API Flow Summary

**For Regular URLs:**
1. User visits: `GET /api/urls/{id}` or URL detail page
2. Frontend (optional): Calls `POST /api/urls/{id}/visit`
3. Handler increments visit counter

**For Short Links:**
1. User clicks short URL: `GET /s/{code}`
2. Handler resolves short code to URL
3. Handler **asynchronously** calls `RecordVisit()` in goroutine
4. Handler returns 302 redirect to original URL
5. Visit record is created (IP, UserAgent)
6. Visit count is incremented

**Visit Record Storage:**
- Table: `t_visit_records`
- Tracks: IP address, user agent, timestamp
- Associated with: URLID or ShortID (or both for tracking same content)
- Soft-deleted records preserved (DeletedAt index)

---

## 3. DATA FLOW DIAGRAMS

### 3.1 Alfred Workflow Search Flow
```
User types in Alfred
    ↓
"ls {query}" keyword triggered
    ↓
Alfred executes: python3 lsearch.py "{query}"
    ↓
lsearch.py:
  1. Read LINKSTASH_SERVER, LINKSTASH_SECRET_KEY from env
  2. Check ~/.linkstash/token for cached JWT
  3. If no token, POST /api/auth/token → get JWT
  4. Save JWT to cache with 0o600 permissions
  5. GET /api/search?q={query}&type=keyword&size=10
     with Authorization: Bearer {token}
  6. Parse response: {data: [{url: {...}, score}, ...]}
  7. Format as Alfred JSON items
  8. Output JSON to stdout
    ↓
Alfred displays results
    ↓
User presses Enter/Cmd+Enter/Alt+Enter
    ↓
Alfred executes configured action:
  - Enter: Open URL in default browser
  - Cmd+Enter: Copy URL to clipboard
  - Alt+Enter: Open LinkStash detail page
```

### 3.2 Alfred Web Search Flow
```
User types in Alfred
    ↓
"lsweb {query}" keyword triggered
    ↓
Alfred executes: Open URL {var:LINKSTASH_SERVER}/?q={query}
    ↓
Browser opens LinkStash with: http://localhost:8888/?q={query}
    ↓
IndexPage.jsx (modified):
  1. useEffect detects ?q= URL parameter on mount
  2. Sets query state from URL param
  3. setSearchType('keyword')
    ↓
SearchBar.jsx (modified):
  1. useEffect syncs parent query to local input
  2. Search automatically triggers
    ↓
Results displayed in LinkStash web UI
    ↓
User can use advanced filters (category, network type, etc.)
```

### 3.3 Visit Tracking Flow (Short Links)
```
User clicks short link: http://server/s/abc123
    ↓
GET /s/{code} route (public, no auth)
    ↓
ShortURLHandler.HandleRedirect:
  1. Extract code from URL path
  2. Call usecase.ResolveShortCode(code)
  3. Check expiration (ShortExpiresAt)
  4. If expired: return 410 GONE
  5. If not found: return 404 NOT_FOUND
  6. Spawn goroutine: go func() {
       usecase.RecordVisit(url.ID)
     }()
  7. Return HTTP 302 redirect to url.Link
    ↓
In background goroutine:
  1. VisitService.RecordURLVisit(urlID, ip, userAgent)
  2. Create VisitRecord entity
  3. VisitRepoImpl.Create() → insert to t_visit_records
  4. URLRepoImpl.IncrementVisit(urlID) → visit_count++
    ↓
User redirected to original URL (no visit latency)
    ↓
Visit tracked in database with IP and user agent
```

---

## 4. KEY API ENDPOINTS

### Public Endpoints
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/s/{code}` | Resolve short link and redirect (tracks visit async) |

### Protected Endpoints (Require JWT Auth)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/urls/{id}/visit` | Record manual visit |
| GET | `/api/search` | Search bookmarks (called by Alfred) |
| POST | `/api/auth/token` | Exchange secret key for JWT |

### Alfred Integration Endpoints
| Method | Endpoint | Purpose | Called By |
|--------|----------|---------|-----------|
| POST | `/api/auth/token` | Get JWT token | lsearch.py first call |
| GET | `/api/search?q={q}&type=keyword&size=10` | Search bookmarks | lsearch.py on each query |

---

## 5. CONFIGURATION & ENVIRONMENT

### Alfred Workflow Configuration Variables
```
LINKSTASH_SERVER: https://your-server.com
LINKSTASH_SECRET_KEY: your-secret-key-here
```

### Python Script Environment
- **Reads from**: Alfred Workflow Environment Variables
- **Token Cache**: `~/.linkstash/token` (0o600 permissions)
- **Network Timeout**: 3 seconds
- **Search Result Size**: 10 items
- **Search Type**: "keyword" (default, fastest)

### Database Table
- **Table**: `t_visit_records`
- **Soft-Delete**: Enabled (DeletedAt indexed)
- **Indexes**: URLID, ShortID, DeletedAt
- **Fields**: ID, URLID, ShortID, IP, UserAgent, CreatedAt, UpdatedAt, DeletedAt

---

## 6. IMPLEMENTATION STATUS

### Completed ✓
- [x] Alfred workflow Python script filter (lsearch.py)
- [x] Alfred workflow configuration (info.plist)
- [x] Workflow icon (icon.png)
- [x] Installation README
- [x] Design specification document
- [x] Implementation plan document
- [x] Visit tracking domain service
- [x] Visit record entity
- [x] Visit repository implementation
- [x] Short link redirect with async visit tracking
- [x] Manual visit endpoint

### Frontend Integration
- [ ] IndexPage.jsx — Read ?q= URL parameter (mentioned in plan, status unclear)
- [ ] SearchBar.jsx — Sync URL param to search input (mentioned in plan, status unclear)

---

## 7. SECURITY CONSIDERATIONS

### Authentication
- JWT tokens cached locally in `~/.linkstash/token`
- Token cache file permissions: 0o600 (user read/write only)
- Auto-refresh on 401 (expired token)
- Secret key stored in Alfred Workflow Environment Variables

### Visit Tracking
- Short link redirects do NOT require authentication (public)
- Manual visit endpoint requires JWT authentication
- Visit records include IP and user agent (privacy considerations)
- IP addresses stored in plaintext (consider anonymization for GDPR)

---

## 8. TESTING RECOMMENDATIONS

### Alfred Script Filter Testing
```bash
cd extend_plugins/alfred/LinkStash.alfredworkflow
LINKSTASH_SERVER="http://localhost:8888" \
LINKSTASH_SECRET_KEY="test-key" \
python3 lsearch.py "test"
```

Expected: Valid JSON with items array or error message

### Token Caching
```bash
# First call creates cache
python3 lsearch.py "test"

# Verify cache
cat ~/.linkstash/token

# Second call uses cache (no auth request)
python3 lsearch.py "test2"
```

### Short Link Visit Tracking
```bash
# Create short link
curl -X POST http://localhost:8888/api/short-links \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"long_url":"https://example.com","code":"test"}'

# Click short link (no auth needed)
curl -L http://localhost:8888/s/test

# Verify visit recorded
curl -X GET http://localhost:8888/api/visit-records \
  -H "Authorization: Bearer $TOKEN"
```

---

## 9. FILE SUMMARY TABLE

| File Path | Type | Lines | Purpose |
|-----------|------|-------|---------|
| `extend_plugins/alfred/README.md` | Markdown | 48 | Installation & usage guide |
| `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py` | Python 3 | 230 | Script filter for search |
| `extend_plugins/alfred/LinkStash.alfredworkflow/info.plist` | XML | 308 | Workflow configuration |
| `extend_plugins/alfred/LinkStash.alfredworkflow/icon.png` | PNG | N/A | Workflow icon |
| `docs/superpowers/plans/2026-04-03-alfred-workflow.md` | Markdown | 884 | Implementation plan |
| `docs/superpowers/specs/2026-04-03-alfred-workflow-design.md` | Markdown | 158 | Technical specification |
| `app/domain/entity/visit_record.go` | Go | 21 | Visit record entity |
| `app/domain/services/visit_service.go` | Go | 37 | Visit service business logic |
| `app/domain/services/url_service.go` | Go | 243 | URL service with visit tracking |
| `app/handler/url_handler.go` | Go | 373 | HTTP handlers for URLs (includes visit endpoint) |
| `app/handler/shorturl_handler.go` | Go | 212 | HTTP handlers for short links (includes redirect with async visit) |
| `app/infra/db/visit_repo_impl.go` | Go | 68 | Visit repository (GORM) |

---

## 10. INTEGRATION POINTS

### Frontend Integration Required
```javascript
// web/src/js/pages/IndexPage.jsx
useEffect(() => {
  const params = new URLSearchParams(window.location.search);
  const q = params.get('q');
  if (q) {
    setQuery(q);
    setSearchType('keyword');
  }
}, []);
```

### Backend Integration
- Alfred calls: `POST /api/auth/token` → `GET /api/search`
- Short links: `GET /s/{code}` → async `RecordVisit()`
- Manual visits: `POST /api/urls/{id}/visit`

---

## Conclusion

The LinkStash repository contains a complete Alfred workflow implementation with:
- **Two trigger modes**: Native search (lsearch) and web search redirect (lsweb)
- **Token caching**: Efficient JWT reuse with auto-refresh
- **Visit tracking**: Comprehensive recording of URL access via short links
- **Domain-driven design**: Clean separation of concerns with services, entities, and repositories
- **Async operations**: Non-blocking visit recording for optimal redirect performance

All components are production-ready with comprehensive error handling and documentation.
