# Quick Reference: Alfred Workflow & Visit Tracking

## 🎯 Quick Index

| Component | Location | Type | Purpose |
|-----------|----------|------|---------|
| Workflow Config | `extend_plugins/alfred/LinkStash.alfredworkflow/info.plist` | XML | Alfred workflow definition |
| Script Filter | `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py` | Python 3 | Bookmark search handler |
| Setup Guide | `extend_plugins/alfred/README.md` | Markdown | Installation & usage |
| Visit Record | `app/domain/entity/visit_record.go` | Go | Data model |
| Visit Service | `app/domain/services/visit_service.go` | Go | Business logic |
| URL Service | `app/domain/services/url_service.go` | Go | Domain service |
| Visit Repo | `app/infra/db/visit_repo_impl.go` | Go | Database layer |
| Short Link Handler | `app/handler/shorturl_handler.go` | Go | `/s/:code` endpoint |
| URL Handler | `app/handler/url_handler.go` | Go | `/api/urls/:id/visit` endpoint |
| Usecase | `app/application/url_usecase.go` | Go | Application orchestration |

---

## 🔄 Alfred Workflow Flow

```
User types: "ls bookmark"
        ↓
Alfred executes: python3 lsearch.py "bookmark"
        ↓
Script reads environment variables:
  • LINKSTASH_SERVER
  • LINKSTASH_SECRET_KEY
        ↓
Get/refresh JWT token:
  1. Check cache (~/.linkstash/token)
  2. If missing/expired: POST /api/auth/token
        ↓
Search bookmarks:
  GET /api/search?q=bookmark&type=keyword&size=10
  Header: Authorization: Bearer {jwt}
        ↓
Parse response and format as Alfred items:
  {
    "items": [
      {
        "uid": "123",
        "title": "bookmark title",
        "subtitle": "url [category] (network_type) score: 0.95",
        "arg": "https://...",
        "mods": {
          "cmd": { "arg": "url", "subtitle": "Copy" },
          "alt": { "arg": "/urls/123", "subtitle": "Open in LinkStash" }
        }
      }
    ]
  }
        ↓
Display results in Alfred
        ↓
User selects item and presses:
  • Enter → Opens URL in browser
  • ⌘+Enter → Copies URL to clipboard
  • ⌥+Enter → Opens in LinkStash web UI
```

---

## 📊 Visit Tracking Flow - Short Link

```
User clicks short link: https://example.com/s/abc123
        ↓
Request: GET /s/abc123
        ↓
ShortURLHandler.HandleRedirect()
        ↓
┌─ Resolve code → Find URL record (ID=42)
│
├─ [ASYNC] Record visit in goroutine:
│   └─ URLUsecase.RecordVisit(42)
│       ├─ URLService.RecordVisit(42)
│       │   └─ URLRepo.IncrementVisit(42)
│       │       └─ UPDATE t_urls SET visit_count += 1 WHERE id = 42
│       │
│       └─ VisitService.RecordShortVisit(42, ip, ua)
│           └─ INSERT INTO t_visit_records (short_id, ip, user_agent, ...)
│
└─ Immediately redirect: HTTP 302 → long URL
```

**Key Point:** Visit recording is non-blocking (goroutine), so redirect happens instantly.

---

## 📊 Visit Tracking Flow - API Endpoint

```
Authenticated request: POST /api/urls/:id/visit
        ↓
URLHandler.HandleVisit(id)
        ↓
URLUsecase.RecordVisit(id)
        ↓
URLService.RecordVisit(id)
        ↓
URLRepo.IncrementVisit(id)
        ↓
UPDATE t_urls SET visit_count = visit_count + 1 WHERE id = ?
        ↓
Return: {"status": "ok"}
```

**Key Point:** This endpoint doesn't create VisitRecord entries (only increments counter).

---

## 🗄️ Database Schema (Visit Tracking)

### t_visit_records Table
```sql
CREATE TABLE t_visit_records (
    id              INT PRIMARY KEY AUTO_INCREMENT,
    url_id          INT NOT NULL INDEXED,      -- Foreign key to t_urls
    short_id        INT INDEXED,                -- Optional: short link reference
    ip              VARCHAR(45) NOT NULL,       -- IPv4 or IPv6
    user_agent      VARCHAR(1000),              -- Browser user agent
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    deleted_at      TIMESTAMP NULLABLE INDEXED -- Soft delete flag
);
```

### t_urls Table (relevant fields)
```sql
CREATE TABLE t_urls (
    id              INT PRIMARY KEY AUTO_INCREMENT,
    link            VARCHAR(2048) NOT NULL UNIQUE,
    short_code      VARCHAR(50) UNIQUE NULLABLE,
    visit_count     INT DEFAULT 0,              -- Incremented on each visit
    short_expires_at TIMESTAMP NULLABLE,
    ...other fields...
    created_at      TIMESTAMP,
    updated_at      TIMESTAMP,
    deleted_at      TIMESTAMP NULLABLE         -- Soft delete
);
```

---

## 🔐 Security Architecture

### Alfred Workflow
```
Environment Variables (user-configured)
    ↓
LINKSTASH_SERVER (e.g., http://localhost:8888)
LINKSTASH_SECRET_KEY (e.g., "super-secret-key-123")
    ↓
Script exchanges secret_key for JWT token
    ↓
JWT cached at ~/.linkstash/token (chmod 600)
    ↓
JWT sent in Authorization header: "Bearer {jwt}"
    ↓
[Cache Management]
    ├─ Token refresh on 401 (unauthorized)
    ├─ Auto-delete stale cache
    └─ Delete manually to force re-auth
```

### Visit Recording
```
Public endpoint (no auth):
    GET /s/:code → Records visit in background
    
Protected endpoint (auth required):
    POST /api/urls/:id/visit → Records visit synchronously
    
Visit data captured:
    ├─ IP address (from HTTP request)
    ├─ User-Agent (from HTTP header)
    ├─ Timestamp (auto-generated by GORM)
    └─ Associated URL/Short ID
```

---

## 🚀 API Contract

### Authentication
```http
POST /api/auth/token
Content-Type: application/json

{"secret_key": "your-secret-key"}

Response (200 OK):
{"token": "eyJhbGciOiJIUzI1NiIs..."}
```

### Search (Alfred uses this)
```http
GET /api/search?q=python&type=keyword&size=10
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

Response (200 OK):
{
  "data": [
    {
      "id": 123,
      "url": {
        "id": 123,
        "title": "Python Docs",
        "link": "https://docs.python.org",
        "category": "Programming",
        "network_type": "public",
        ...
      },
      "score": 0.95
    }
  ]
}
```

### Record Visit (Protected)
```http
POST /api/urls/:id/visit
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

Response (200 OK):
{"status": "ok"}
```

### Resolve Short Link (Public)
```http
GET /s/abc123

Response (302 Found):
Location: https://original-long-url.com
```

---

## 📝 Configuration Guide

### Step 1: Install Alfred Workflow
1. Double-click `LinkStash.alfredworkflow/` directory
2. Or import via Alfred Preferences → Workflows

### Step 2: Configure Environment Variables
In Alfred Preferences → Workflows → LinkStash Search:
```
Server URL:      http://localhost:8888 (or your server)
Secret Key:      your-authentication-secret-key
```

### Step 3: Test
```bash
# From terminal
cd extend_plugins/alfred/LinkStash.alfredworkflow
LINKSTASH_SERVER="http://localhost:8888" \
LINKSTASH_SECRET_KEY="your-key" \
python3 lsearch.py test
```

---

## 🐛 Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| "Configuration missing" | Env vars not set | Set LINKSTASH_SERVER and LINKSTASH_SECRET_KEY in Alfred workflow settings |
| "Connection failed" | Server unreachable | Verify server URL and network connectivity |
| "Auth failed" | Invalid secret key | Check secret key in workflow settings |
| "No results" | Token expired | Delete `~/.linkstash/token` to force re-auth |
| Slow search | Network latency | Check server performance or increase timeout |

---

## 🔗 Code Entry Points

### Alfred Integration
- **Entry:** `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py` (line 206)
- **Main function** processes command-line arguments and outputs JSON

### Visit Recording - Short Links
- **Entry:** `app/handler/shorturl_handler.go` (line 162, `HandleRedirect`)
- **Async visit recording** in goroutine (line 180-182)

### Visit Recording - API
- **Entry:** `app/handler/url_handler.go` (line 260, `HandleVisit`)
- **Synchronous visit recording** (line 267)

### Domain Logic
- **URL Service:** `app/domain/services/url_service.go` (line 81)
- **Visit Service:** `app/domain/services/visit_service.go` (line 18)

---

## 📈 Performance Notes

### Alfred Workflow
- **Search timeout:** 3 seconds
- **Max results:** 10 per query
- **Token cache:** Eliminates redundant auth calls
- **HTTP client timeout:** 3 seconds

### Visit Recording
- **Short link visits:** Non-blocking (background goroutine)
- **API visits:** Blocking (waits for database insert)
- **Database indexes:** On `url_id`, `short_id`, `deleted_at`

---

## 🎓 Key Design Patterns

1. **Clean Architecture**
   - Entity → Service → Usecase → Handler
   - Separation of concerns (domain, infra, application layers)

2. **Repository Pattern**
   - Database access abstracted via interfaces
   - Easy to swap implementations or add caching

3. **Service Layer**
   - Business logic encapsulated in domain services
   - Reusable across multiple handlers

4. **Async Operations**
   - Non-blocking visit recording for short links
   - Improves user experience (instant redirect)

5. **Token Caching**
   - JWT cached locally to reduce auth calls
   - Auto-refresh on 401 (unauthorized)

