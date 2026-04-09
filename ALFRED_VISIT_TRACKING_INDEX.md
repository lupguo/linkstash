# Alfred Workflow & Visit Tracking - Complete Analysis Index

**Generated:** April 9, 2026  
**Repository:** LinkStash  
**Analysis Scope:** Alfred integration files + Visit tracking API

---

## 📋 Quick Navigation

### Files Found: 12 Total
- **Python Scripts:** 1
- **Go Source:** 5
- **Configuration:** 1
- **Images:** 1
- **Documentation:** 4

---

## 🎯 Alfred Workflow Components

### 1. Entry Point: `extend_plugins/alfred/README.md`
- **Lines:** 48
- **Purpose:** Installation and usage guide
- **Key Sections:**
  - Installation instructions
  - Usage modes (`ls`, `lsweb`)
  - Configuration requirements
  - Troubleshooting

### 2. Script Filter: `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py`
- **Lines:** 230
- **Language:** Python 3
- **Purpose:** Alfred Script Filter for bookmark search

**Core Modules:**
- Configuration management (lines 15-20)
- Token management (lines 36-100)
- Search API integration (lines 107-141)
- Alfred JSON output formatter (lines 148-199)
- Main entry point (lines 206-229)

**Key Functions:**
| Function | Lines | Purpose |
|----------|-------|---------|
| `alfred_error()` | 23-33 | Error output to Alfred |
| `read_cached_token()` | 40-46 | Read JWT from cache |
| `save_token()` | 49-52 | Save JWT with 0o600 permissions |
| `delete_cached_token()` | 56-61 | Remove expired token |
| `exchange_token()` | 64-93 | POST /api/auth/token |
| `get_token()` | 95-100 | Get token (cache or new) |
| `search()` | 107-125 | GET /api/search API call |
| `search_with_retry()` | 128-141 | Auto-refresh on 401 |
| `format_alfred_items()` | 148-199 | Convert to Alfred JSON |
| `main()` | 206-229 | Script entry point |

### 3. Workflow Definition: `extend_plugins/alfred/LinkStash.alfredworkflow/info.plist`
- **Lines:** 308
- **Format:** XML (Apple Property List)
- **Purpose:** Define workflow structure and connections

**Workflow Objects:**
1. **lsearch-scriptfilter** — Keyword "ls" → Script Filter
2. **open-url-action** — Enter key → Open URL
3. **copy-action** — Cmd+Enter → Copy to clipboard
4. **open-linkstash-action** — Alt+Enter → Open LinkStash
5. **linkstash-keyword** — Keyword "lsweb" → Web search
6. **open-web-search** — Open browser with query parameter

**Configuration Variables:**
- `LINKSTASH_SERVER` (required, textfield)
- `LINKSTASH_SECRET_KEY` (required, textfield)

### 4. Workflow Icon: `extend_plugins/alfred/LinkStash.alfredworkflow/icon.png`
- **Format:** PNG (256×256)
- **Color:** Sky blue (#38bdf8)

### 5. Implementation Plan: `docs/superpowers/plans/2026-04-03-alfred-workflow.md`
- **Lines:** 884
- **Structure:** 5 tasks with detailed steps
- **Coverage:** Complete implementation guide

### 6. Technical Specification: `docs/superpowers/specs/2026-04-03-alfred-workflow-design.md`
- **Lines:** 158
- **Coverage:** Architecture, API contracts, error handling

---

## 🔍 Visit Tracking API

### Domain Layer

#### Entity: `app/domain/entity/visit_record.go`
```go
type VisitRecord struct {
  ID        uint           // Primary key
  URLID     uint           // Link to URL (indexed)
  ShortID   uint           // Link to short link (indexed)
  IP        string         // Visitor IP
  UserAgent string         // Browser UA
  CreatedAt time.Time      // Auto-created
  UpdatedAt time.Time      // Auto-updated
  DeletedAt gorm.DeletedAt  // Soft-delete (indexed)
}
```

#### Service: `app/domain/services/visit_service.go`
- **Lines:** 37
- **Methods:**
  - `RecordURLVisit(urlID uint, ip, userAgent string) error`
  - `RecordShortVisit(shortID uint, ip, userAgent string) error`

#### Service: `app/domain/services/url_service.go`
- **Lines:** 243
- **Key Methods for Visit Tracking:**
  - `RecordVisit(id uint) error` (lines 81-84)
  - `GenerateShortLink(...)` (lines 87-166)
  - `ResolveShortCode(...)` (lines 168-180)

### Handler Layer

#### URL Handler: `app/handler/url_handler.go`
- **Lines:** 373
- **Visit Endpoint:** `HandleVisit` (lines 259-273)
  - Route: `POST /api/urls/{id}/visit`
  - Auth: Requires JWT Bearer token
  - Response: `{"status": "ok"}`

#### Short Link Handler: `app/handler/shorturl_handler.go`
- **Lines:** 212
- **Redirect Endpoint:** `HandleRedirect` (lines 160-185)
  - Route: `GET /s/{code}`
  - Auth: PUBLIC (no JWT required)
  - Flow: Resolve code → Async record visit → 302 redirect

### Repository Layer

#### Visit Repository: `app/infra/db/visit_repo_impl.go`
- **Lines:** 68
- **Methods:**
  - `Create(record *entity.VisitRecord) error`
  - `ListByURLID(urlID uint, page, size int) ([]*entity.VisitRecord, int64, error)`
  - `ListByShortID(shortID uint, page, size int) ([]*entity.VisitRecord, int64, error)`

---

## 🔄 API Flows

### Alfred Search Flow
```
User → "ls test" in Alfred
  ↓
Alfred executes: python3 lsearch.py "test"
  ↓
lsearch.py:
  1. Read: LINKSTASH_SERVER, LINKSTASH_SECRET_KEY
  2. Check: ~/.linkstash/token (cached JWT)
  3. If missing: POST /api/auth/token
  4. Call: GET /api/search?q=test&type=keyword&size=10
  5. Format: Response → Alfred JSON
  6. Output: JSON to stdout
  ↓
Alfred displays results
  ↓
User selects:
  • Enter: open-url-action
  • Cmd+Enter: copy-action
  • Alt+Enter: open-linkstash-action
```

### Short Link Visit Tracking
```
User → clicks http://server/s/abc123 [PUBLIC]
  ↓
GET /s/{code} [No JWT required]
  ↓
ShortURLHandler.HandleRedirect:
  1. Extract code from URL
  2. Call: usecase.ResolveShortCode(code)
  3. Check: ShortExpiresAt expiration
  4. Spawn: go func() { RecordVisit(url.ID) }()
  5. Return: HTTP 302 redirect
  ↓
[Background Goroutine]:
  1. Create: VisitRecord entity
  2. Insert: to t_visit_records
  3. Increment: URL.VisitCount
  ↓
User → redirected to original URL
```

---

## 🛣️ API Endpoints Summary

| Method | Path | Auth | Purpose | Handler |
|--------|------|------|---------|---------|
| **POST** | `/api/auth/token` | ❌ | Get JWT token | - |
| **GET** | `/api/search` | ✅ | Search bookmarks | SearchHandler |
| **POST** | `/api/urls/{id}/visit` | ✅ | Record visit | URLHandler.HandleVisit |
| **GET** | `/s/{code}` | ❌ | Resolve & redirect | ShortURLHandler.HandleRedirect |

---

## ⚙️ Configuration Reference

### Alfred Workflow Environment Variables
```
LINKSTASH_SERVER = https://your-server.com
LINKSTASH_SECRET_KEY = your-secret-key-here
```

### Python Script Defaults
```
Token cache: ~/.linkstash/token
Cache mode: 0o600 (user read/write)
Timeout: 3 seconds
Result size: 10 items
Search type: keyword
```

### Database Table
```
Table: t_visit_records
Soft-delete: Yes
Indexes: URLID, ShortID, DeletedAt
Primary key: ID (auto-increment)
```

---

## 🔐 Security Notes

✅ **Secure:**
- JWT tokens cached with restricted permissions (0o600)
- Auto-refresh on 401 (expired token handling)
- Short link redirects public (by design, no sensitive data)
- Manual visit endpoint protected (JWT required)

⚠️ **Considerations:**
- Visit records store IP addresses in plaintext
- Consider GDPR implications for IP storage
- Possible anonymization/hashing for privacy

---

## 📊 Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| Alfred Script Filter | ✅ Complete | lsearch.py fully implemented |
| Workflow Config | ✅ Complete | info.plist with 6 objects |
| Workflow Icon | ✅ Complete | 256×256 PNG |
| Documentation | ✅ Complete | README, plan, spec |
| Visit Service | ✅ Complete | RecordURLVisit, RecordShortVisit |
| Visit Entity | ✅ Complete | GORM with soft-delete |
| Visit Repository | ✅ Complete | GORM implementation |
| Visit Endpoint | ✅ Complete | POST /api/urls/{id}/visit |
| Short Link Redirect | ✅ Complete | Async visit tracking |
| Frontend Param Support | ⚠️ Partial | IndexPage.jsx changes mentioned in plan |

---

## 🧪 Testing Quick Reference

### Test Alfred Script Filter
```bash
cd extend_plugins/alfred/LinkStash.alfredworkflow
LINKSTASH_SERVER="http://localhost:8888" \
LINKSTASH_SECRET_KEY="test-key" \
python3 lsearch.py "test"
```

### Verify Token Caching
```bash
cat ~/.linkstash/token
```

### Test Short Link
```bash
curl -L http://localhost:8888/s/abc123
```

---

## 📈 File Statistics

| Category | Count | Languages |
|----------|-------|-----------|
| Alfred Files | 4 | Python, XML, PNG |
| Visit Tracking | 5 | Go (GORM) |
| Documentation | 4 | Markdown |
| **Total** | **12** | Multi-language |

---

## 🎓 Key Learnings

### Architecture Pattern
- **DDD (Domain-Driven Design)**: Clear separation of concerns
- **Layers**: Domain → Application → Infrastructure → Handler

### Technology Stack
- **Backend**: Go, GORM, Chi router
- **Alfred**: Python 3 (stdlib only, no external deps)
- **Auth**: JWT with token caching

### Notable Implementations
- Async visit tracking (non-blocking redirect)
- JWT token caching with auto-refresh
- Soft-delete for data retention
- Base62 short code generation
- TTL-based short link expiration

---

## 📚 Full Analysis Document

See: `ALFRED_AND_VISIT_TRACKING_ANALYSIS.md`

Contains:
- Detailed file descriptions (1000+ lines)
- Complete data flow diagrams
- Database schema documentation
- Error handling specifications
- Security considerations
- Integration points
- Testing recommendations

---

## 🔗 Cross-References

**Alfred Integration:**
- Entry → `README.md`
- Implementation → `lsearch.py`
- Configuration → `info.plist`
- Planning → `2026-04-03-alfred-workflow.md`

**Visit Tracking:**
- Data → `visit_record.go`
- Logic → `visit_service.go`
- API → `url_handler.go`
- Redirect → `shorturl_handler.go`
- Storage → `visit_repo_impl.go`

---

**Document Status:** Complete  
**Last Updated:** April 9, 2026  
**All Files Accounted For:** ✅
