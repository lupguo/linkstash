# Alfred Workflow for LinkStash Search

## Overview

Alfred Workflow enabling fast bookmark search from anywhere on macOS. Two trigger modes: native Alfred list results via API, and browser-based web search.

## Trigger Modes

### Mode 1: `lsearch {query}` — Alfred Native Search

- **Trigger**: `lsearch` keyword in Alfred
- **Implementation**: Python Script Filter calls `/api/search?q={query}&type=keyword&size=10`
- **Display**: Alfred native result list with title, URL, category
- **Actions**:
  - `Enter` — open original URL in browser
  - `⌘+Enter` — copy URL to clipboard
  - `⌥+Enter` — open LinkStash detail page (`/urls/{id}`)

### Mode 2: `linkstash {query}` — Web Search Redirect

- **Trigger**: `linkstash` keyword in Alfred
- **Implementation**: Open URL action → `{server}/?q={query}`
- **Behavior**: Browser opens LinkStash web UI with search pre-filled
- **Requires**: Frontend change to read `?q=` URL parameter on load

## Directory Structure

```
extend_plugins/
├── alfred/
│   └── LinkStash.alfredworkflow/
│       ├── info.plist          # Workflow definition (two triggers + connections)
│       ├── lsearch.py          # Script Filter: API search → Alfred JSON
│       ├── config.py           # Configuration management + token caching
│       └── icon.png            # Workflow icon
└── popclip/                    # Reserved for future PopClip extension
```

## Script Filter (lsearch.py)

### Core Flow

1. Read config from Alfred Workflow environment variables
2. Get JWT token (cached in `~/.linkstash/token`, refresh on 401)
3. Call `GET {server}/api/search?q={query}&type=keyword&size=10` with `Authorization: Bearer {token}`
4. Parse response `{data: [{url: {id, title, link, description, category, network_type}, score}], total}`
5. Output Alfred Script Filter JSON

### Alfred JSON Output Format

```json
{
  "items": [
    {
      "uid": "123",
      "title": "Claude AI - Anthropic",
      "subtitle": "https://claude.ai  [AI]  score: 0.95",
      "arg": "https://claude.ai",
      "icon": {"path": "icon.png"},
      "mods": {
        "cmd": {
          "arg": "https://claude.ai",
          "subtitle": "Copy URL to clipboard"
        },
        "alt": {
          "arg": "https://your-server/urls/123",
          "subtitle": "Open in LinkStash"
        }
      }
    }
  ]
}
```

### Dependencies

- Python 3 (macOS built-in) — `urllib.request`, `json`, `os`, `pathlib`
- No external packages required

## Configuration

### Alfred Workflow Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `LINKSTASH_SERVER` | Server base URL | `https://linkstash.example.com` |
| `LINKSTASH_SECRET_KEY` | Auth secret key | `your-secret-key` |

### Token Caching

- Storage: `~/.linkstash/token` (plain text file)
- On first call or 401: POST `/api/auth/token` with `{"secret_key": "..."}` to get JWT
- Cache token to file; reuse on subsequent calls
- On 401 response: delete cached token, re-authenticate, retry once

## Frontend Change (Mode 2 Support)

### IndexPage.jsx Modification

Add URL query parameter support on component mount:

```javascript
// In IndexPage component, add to existing useEffect or new useEffect:
useEffect(() => {
  const params = new URLSearchParams(window.location.search);
  const q = params.get('q');
  if (q) {
    setQuery(q);
    setSearchType('keyword');
  }
}, []);
```

This is approximately 5 lines of code in `web/src/js/pages/IndexPage.jsx`.

## Error Handling

| Scenario | Alfred Display |
|----------|---------------|
| Network timeout (>3s) | `Warning: Connection timeout - check server` |
| Auth failure (401) | `Warning: Auth failed - check secret key in workflow settings` |
| No results | `No results for "{query}"` |
| Server error (5xx) | `Warning: Server error: {status}` |
| Missing config | `Warning: Set LINKSTASH_SERVER and LINKSTASH_SECRET_KEY in workflow settings` |

All error states display as single Alfred item with appropriate icon and subtitle.

## Performance

- Script Filter response target: < 500ms (local network)
- Alfred debounce: 300ms (configured in info.plist)
- Default result limit: 10 items
- Search type: `keyword` (fastest; semantic/hybrid available but slower)

## Installation

1. Double-click `LinkStash.alfredworkflow` to import into Alfred
2. In Alfred Preferences → Workflows → LinkStash, set environment variables:
   - `LINKSTASH_SERVER`: your LinkStash server URL
   - `LINKSTASH_SECRET_KEY`: your authentication secret key
3. Test: type `lsearch test` in Alfred

## Scope

### In Scope

- Alfred Workflow with two trigger modes (lsearch + linkstash)
- Python Script Filter for API search
- Token caching and error handling
- Frontend `?q=` parameter support for web search mode

### Out of Scope

- PopClip extension (future work, directory reserved)
- Semantic/hybrid search toggle in Alfred (use web mode for advanced search)
- URL creation or editing from Alfred
- Alfred Workflow auto-update mechanism
