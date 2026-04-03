# Alfred Workflow for LinkStash Search — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an Alfred Workflow with two trigger modes — native API search (`lsearch`) and web search redirect (`linkstash`) — for fast bookmark retrieval on macOS.

**Architecture:** Python Script Filter calls LinkStash `/api/search` API, outputs Alfred JSON format. A second trigger opens the browser with `?q=` query param. Frontend IndexPage reads URL params on mount to auto-fill search.

**Tech Stack:** Python 3 (stdlib only: urllib, json, os, pathlib), Alfred Workflow XML (info.plist), Preact (IndexPage.jsx change)

**Spec:** `docs/superpowers/specs/2026-04-03-alfred-workflow-design.md`

---

## File Structure

```
extend_plugins/
├── alfred/
│   └── LinkStash.alfredworkflow/
│       ├── info.plist       # Alfred Workflow definition (triggers, connections, env vars)
│       ├── lsearch.py       # Script Filter: config + auth + search + Alfred JSON output
│       └── icon.png         # Workflow icon (simple placeholder)
web/src/js/pages/
└── IndexPage.jsx            # Modify: read ?q= URL param on mount
```

Note: `popclip/` already exists at repo root. The spec mentioned `extend_plugins/popclip/` as reserved — we will NOT move the existing popclip directory; only create the new `extend_plugins/alfred/` path.

---

### Task 1: Create config and auth module (lsearch.py foundation)

**Files:**
- Create: `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py`

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p extend_plugins/alfred/LinkStash.alfredworkflow
```

- [ ] **Step 2: Write lsearch.py with config reading and token management**

Create `extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py`:

```python
#!/usr/bin/env python3
"""LinkStash Alfred Workflow — Script Filter for bookmark search."""

import json
import os
import sys
import urllib.request
import urllib.error
import urllib.parse
from pathlib import Path

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

SERVER_URL = os.environ.get("LINKSTASH_SERVER", "").rstrip("/")
SECRET_KEY = os.environ.get("LINKSTASH_SECRET_KEY", "")
TOKEN_PATH = Path.home() / ".linkstash" / "token"
SEARCH_SIZE = 10
TIMEOUT_SECONDS = 3


def alfred_error(title, subtitle=""):
    """Output a single Alfred item indicating an error."""
    print(json.dumps({
        "items": [{
            "title": title,
            "subtitle": subtitle,
            "valid": False,
            "icon": {"path": "icon.png"},
        }]
    }))
    sys.exit(0)


# ---------------------------------------------------------------------------
# Token management
# ---------------------------------------------------------------------------

def read_cached_token():
    """Read JWT token from cache file. Returns None if missing or empty."""
    try:
        token = TOKEN_PATH.read_text().strip()
        return token if token else None
    except (FileNotFoundError, OSError):
        return None


def save_token(token):
    """Save JWT token to cache file."""
    TOKEN_PATH.parent.mkdir(parents=True, exist_ok=True)
    TOKEN_PATH.write_text(token)


def delete_cached_token():
    """Remove cached token file."""
    try:
        TOKEN_PATH.unlink()
    except FileNotFoundError:
        pass


def exchange_token():
    """Exchange secret_key for a JWT via /api/auth/token."""
    if not SERVER_URL or not SECRET_KEY:
        alfred_error(
            "Configuration missing",
            "Set LINKSTASH_SERVER and LINKSTASH_SECRET_KEY in workflow settings",
        )

    url = f"{SERVER_URL}/api/auth/token"
    payload = json.dumps({"secret_key": SECRET_KEY}).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT_SECONDS) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            token = data.get("token", "")
            if not token:
                alfred_error("Auth failed", "No token in server response")
            save_token(token)
            return token
    except urllib.error.HTTPError as e:
        alfred_error("Auth failed", f"HTTP {e.code} — check secret key in workflow settings")
    except urllib.error.URLError:
        alfred_error("Connection failed", f"Cannot reach {SERVER_URL}")


def get_token():
    """Get a valid JWT token, using cache when available."""
    token = read_cached_token()
    if token:
        return token
    return exchange_token()


# ---------------------------------------------------------------------------
# Search API
# ---------------------------------------------------------------------------

def search(query, token):
    """Call /api/search and return the parsed JSON response dict."""
    params = urllib.parse.urlencode({
        "q": query,
        "type": "keyword",
        "size": SEARCH_SIZE,
    })
    url = f"{SERVER_URL}/api/search?{params}"
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {token}"})

    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT_SECONDS) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        if e.code == 401:
            return None  # Signal to retry with fresh token
        alfred_error("Server error", f"HTTP {e.code}")
    except urllib.error.URLError:
        alfred_error("Connection timeout", f"Cannot reach {SERVER_URL}")


def search_with_retry(query):
    """Search with automatic token refresh on 401."""
    token = get_token()
    result = search(query, token)

    if result is None:
        # 401 — token expired, refresh and retry once
        delete_cached_token()
        token = exchange_token()
        result = search(query, token)
        if result is None:
            alfred_error("Auth failed", "Token refresh failed — check secret key")

    return result


# ---------------------------------------------------------------------------
# Alfred output
# ---------------------------------------------------------------------------

def format_alfred_items(result, query):
    """Convert search API response to Alfred Script Filter JSON."""
    data = result.get("data", [])

    if not data:
        return {
            "items": [{
                "title": f'No results for "{query}"',
                "subtitle": "Try a different search term",
                "valid": False,
                "icon": {"path": "icon.png"},
            }]
        }

    items = []
    for entry in data:
        url_data = entry.get("url", {})
        score = entry.get("score", 0)

        uid = str(url_data.get("id", ""))
        title = url_data.get("title", "") or url_data.get("link", "")
        link = url_data.get("link", "")
        category = url_data.get("category", "")
        network_type = url_data.get("network_type", "")

        subtitle_parts = [link]
        if category:
            subtitle_parts.append(f"[{category}]")
        if network_type:
            subtitle_parts.append(f"({network_type})")
        if score:
            subtitle_parts.append(f"score: {score:.2f}")

        items.append({
            "uid": uid,
            "title": title,
            "subtitle": "  ".join(subtitle_parts),
            "arg": link,
            "icon": {"path": "icon.png"},
            "mods": {
                "cmd": {
                    "arg": link,
                    "subtitle": f"Copy to clipboard: {link}",
                },
                "alt": {
                    "arg": f"{SERVER_URL}/urls/{uid}",
                    "subtitle": "Open in LinkStash",
                },
            },
        })

    return {"items": items}


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    if not SERVER_URL or not SECRET_KEY:
        alfred_error(
            "Configuration missing",
            "Set LINKSTASH_SERVER and LINKSTASH_SECRET_KEY in workflow settings",
        )

    query = " ".join(sys.argv[1:]).strip()
    if not query:
        print(json.dumps({"items": [{
            "title": "Type to search LinkStash...",
            "subtitle": "Enter a keyword to search your bookmarks",
            "valid": False,
            "icon": {"path": "icon.png"},
        }]}))
        return

    result = search_with_retry(query)
    output = format_alfred_items(result, query)
    print(json.dumps(output))


if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Test the script manually from command line**

Run (replace with your actual server/key):
```bash
cd extend_plugins/alfred/LinkStash.alfredworkflow
LINKSTASH_SERVER="http://localhost:8888" LINKSTASH_SECRET_KEY="your-key" python3 lsearch.py claude
```

Expected: JSON output with `"items"` array. Either search results or a "No results" item.

Verify empty query:
```bash
LINKSTASH_SERVER="http://localhost:8888" LINKSTASH_SECRET_KEY="your-key" python3 lsearch.py
```

Expected: `{"items": [{"title": "Type to search LinkStash..."}]}`

- [ ] **Step 4: Commit**

```bash
git add extend_plugins/alfred/LinkStash.alfredworkflow/lsearch.py
git commit -m "feat(alfred): add Script Filter search script with auth and token caching"
```

---

### Task 2: Create Alfred Workflow info.plist

**Files:**
- Create: `extend_plugins/alfred/LinkStash.alfredworkflow/info.plist`

The `info.plist` defines the two trigger keywords, their connections, and workflow environment variables.

- [ ] **Step 1: Write info.plist**

Create `extend_plugins/alfred/LinkStash.alfredworkflow/info.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>bundleid</key>
	<string>com.linkstash.alfred</string>
	<key>category</key>
	<string>Productivity</string>
	<key>connections</key>
	<dict>
		<!-- lsearch Script Filter → Open URL action -->
		<key>lsearch-scriptfilter</key>
		<array>
			<dict>
				<key>destinationuid</key>
				<string>open-url-action</string>
				<key>modifiers</key>
				<integer>0</integer>
				<key>modifiersubtext</key>
				<string></string>
				<key>vitowards</key>
				<string></string>
			</dict>
			<dict>
				<key>destinationuid</key>
				<string>copy-action</string>
				<key>modifiers</key>
				<integer>1048576</integer>
				<key>modifiersubtext</key>
				<string>Copy URL to clipboard</string>
				<key>vitowards</key>
				<string></string>
			</dict>
			<dict>
				<key>destinationuid</key>
				<string>open-linkstash-action</string>
				<key>modifiers</key>
				<integer>524288</integer>
				<key>modifiersubtext</key>
				<string>Open in LinkStash</string>
				<key>vitowards</key>
				<string></string>
			</dict>
		</array>
		<!-- linkstash keyword → Open URL action -->
		<key>linkstash-keyword</key>
		<array>
			<dict>
				<key>destinationuid</key>
				<string>open-web-search</string>
				<key>modifiers</key>
				<integer>0</integer>
				<key>modifiersubtext</key>
				<string></string>
				<key>vitowards</key>
				<string></string>
			</dict>
		</array>
	</dict>
	<key>createdby</key>
	<string>LinkStash</string>
	<key>description</key>
	<string>Search your LinkStash bookmarks from Alfred</string>
	<key>disabled</key>
	<false/>
	<key>name</key>
	<string>LinkStash Search</string>
	<key>objects</key>
	<array>
		<!-- Object 0: lsearch Script Filter -->
		<dict>
			<key>config</key>
			<dict>
				<key>alfredfiltersresults</key>
				<false/>
				<key>argumenttype</key>
				<integer>0</integer>
				<key>escaping</key>
				<integer>102</integer>
				<key>keyword</key>
				<string>lsearch</string>
				<key>queuedelaycustom</key>
				<integer>3</integer>
				<key>queuedelayimmediatelyalinitially</key>
				<true/>
				<key>queuedelaymode</key>
				<integer>0</integer>
				<key>queuemode</key>
				<integer>1</integer>
				<key>runningsubtext</key>
				<string>Searching LinkStash...</string>
				<key>script</key>
				<string>python3 lsearch.py "{query}"</string>
				<key>scriptargtype</key>
				<integer>0</integer>
				<key>scriptfile</key>
				<string></string>
				<key>subtext</key>
				<string>Search your bookmarks</string>
				<key>title</key>
				<string>LinkStash Search</string>
				<key>type</key>
				<integer>0</integer>
				<key>withspace</key>
				<true/>
			</dict>
			<key>type</key>
			<string>alfred.workflow.input.scriptfilter</string>
			<key>uid</key>
			<string>lsearch-scriptfilter</string>
			<key>version</key>
			<integer>3</integer>
		</dict>
		<!-- Object 1: Open URL (Enter on lsearch result) -->
		<dict>
			<key>config</key>
			<dict>
				<key>browser</key>
				<string></string>
				<key>spaces</key>
				<string></string>
				<key>url</key>
				<string>{query}</string>
			</dict>
			<key>type</key>
			<string>alfred.workflow.action.openurl</string>
			<key>uid</key>
			<string>open-url-action</string>
			<key>version</key>
			<integer>1</integer>
		</dict>
		<!-- Object 2: Copy to clipboard (Cmd+Enter) -->
		<dict>
			<key>config</key>
			<dict>
				<key>clipboardtext</key>
				<string>{query}</string>
				<key>transient</key>
				<false/>
			</dict>
			<key>type</key>
			<string>alfred.workflow.output.clipboard</string>
			<key>uid</key>
			<string>copy-action</string>
			<key>version</key>
			<integer>3</integer>
		</dict>
		<!-- Object 3: Open in LinkStash (Alt+Enter) -->
		<dict>
			<key>config</key>
			<dict>
				<key>browser</key>
				<string></string>
				<key>spaces</key>
				<string></string>
				<key>url</key>
				<string>{query}</string>
			</dict>
			<key>type</key>
			<string>alfred.workflow.action.openurl</string>
			<key>uid</key>
			<string>open-linkstash-action</string>
			<key>version</key>
			<integer>1</integer>
		</dict>
		<!-- Object 4: linkstash keyword trigger -->
		<dict>
			<key>config</key>
			<dict>
				<key>argumenttype</key>
				<integer>0</integer>
				<key>keyword</key>
				<string>linkstash</string>
				<key>subtext</key>
				<string>Open LinkStash web search</string>
				<key>text</key>
				<string>LinkStash Web Search</string>
				<key>withspace</key>
				<true/>
			</dict>
			<key>type</key>
			<string>alfred.workflow.input.keyword</string>
			<key>uid</key>
			<string>linkstash-keyword</string>
			<key>version</key>
			<integer>1</integer>
		</dict>
		<!-- Object 5: Open web search URL -->
		<dict>
			<key>config</key>
			<dict>
				<key>browser</key>
				<string></string>
				<key>spaces</key>
				<string></string>
				<key>url</key>
				<string>{var:LINKSTASH_SERVER}/?q={query}</string>
			</dict>
			<key>type</key>
			<string>alfred.workflow.action.openurl</string>
			<key>uid</key>
			<string>open-web-search</string>
			<key>version</key>
			<integer>1</integer>
		</dict>
	</array>
	<key>readme</key>
	<string>## LinkStash Search for Alfred

**lsearch {query}** — Search bookmarks, results shown in Alfred
- Enter: Open URL in browser
- Cmd+Enter: Copy URL to clipboard
- Alt+Enter: Open in LinkStash web UI

**linkstash {query}** — Open LinkStash web search in browser

### Setup
Set these Workflow Environment Variables:
- LINKSTASH_SERVER: Your server URL (e.g. https://linkstash.example.com)
- LINKSTASH_SECRET_KEY: Your authentication secret key</string>
	<key>uidata</key>
	<dict>
		<key>lsearch-scriptfilter</key>
		<dict>
			<key>xpos</key>
			<integer>100</integer>
			<key>ypos</key>
			<integer>100</integer>
		</dict>
		<key>open-url-action</key>
		<dict>
			<key>xpos</key>
			<integer>400</integer>
			<key>ypos</key>
			<integer>100</integer>
		</dict>
		<key>copy-action</key>
		<dict>
			<key>xpos</key>
			<integer>400</integer>
			<key>ypos</key>
			<integer>200</integer>
		</dict>
		<key>open-linkstash-action</key>
		<dict>
			<key>xpos</key>
			<integer>400</integer>
			<key>ypos</key>
			<integer>300</integer>
		</dict>
		<key>linkstash-keyword</key>
		<dict>
			<key>xpos</key>
			<integer>100</integer>
			<key>ypos</key>
			<integer>400</integer>
		</dict>
		<key>open-web-search</key>
		<dict>
			<key>xpos</key>
			<integer>400</integer>
			<key>ypos</key>
			<integer>400</integer>
		</dict>
	</dict>
	<key>userconfigurationconfig</key>
	<array>
		<dict>
			<key>config</key>
			<dict>
				<key>default</key>
				<string></string>
				<key>placeholder</key>
				<string>https://linkstash.example.com</string>
				<key>required</key>
				<true/>
				<key>trim</key>
				<true/>
			</dict>
			<key>description</key>
			<string>Your LinkStash server URL</string>
			<key>label</key>
			<string>Server URL</string>
			<key>type</key>
			<string>textfield</string>
			<key>variable</key>
			<string>LINKSTASH_SERVER</string>
		</dict>
		<dict>
			<key>config</key>
			<dict>
				<key>default</key>
				<string></string>
				<key>placeholder</key>
				<string>your-secret-key</string>
				<key>required</key>
				<true/>
				<key>trim</key>
				<true/>
			</dict>
			<key>description</key>
			<string>Authentication secret key</string>
			<key>label</key>
			<string>Secret Key</string>
			<key>type</key>
			<string>textfield</string>
			<key>variable</key>
			<string>LINKSTASH_SECRET_KEY</string>
		</dict>
	</array>
	<key>version</key>
	<string>1.0.0</string>
	<key>webaddress</key>
	<string></string>
</dict>
</plist>
```

- [ ] **Step 2: Commit**

```bash
git add extend_plugins/alfred/LinkStash.alfredworkflow/info.plist
git commit -m "feat(alfred): add workflow info.plist with dual triggers and env var config"
```

---

### Task 3: Create placeholder icon

**Files:**
- Create: `extend_plugins/alfred/LinkStash.alfredworkflow/icon.png`

- [ ] **Step 1: Generate a simple placeholder icon**

Create a 256x256 PNG icon. Use Python to generate a simple colored square with "LS" text:

```bash
cd extend_plugins/alfred/LinkStash.alfredworkflow
python3 -c "
import struct, zlib

# 256x256 sky-blue square PNG (minimal valid PNG)
width, height = 256, 256
# Sky blue RGB: 56, 189, 248 (#38bdf8) matching LinkStash accent color
row = b'\x00' + b'\x38\xbd\xf8' * width  # filter byte + RGB pixels
raw = row * height
compressed = zlib.compress(raw)

def chunk(ctype, data):
    c = ctype + data
    return struct.pack('>I', len(data)) + c + struct.pack('>I', zlib.crc32(c) & 0xffffffff)

with open('icon.png', 'wb') as f:
    f.write(b'\x89PNG\r\n\x1a\n')
    f.write(chunk(b'IHDR', struct.pack('>IIBBBBB', width, height, 8, 2, 0, 0, 0)))
    f.write(chunk(b'IDAT', compressed))
    f.write(chunk(b'IEND', b''))
print('icon.png created')
"
```

- [ ] **Step 2: Commit**

```bash
git add extend_plugins/alfred/LinkStash.alfredworkflow/icon.png
git commit -m "feat(alfred): add placeholder workflow icon"
```

---

### Task 4: Frontend — support ?q= URL parameter in IndexPage

**Files:**
- Modify: `web/src/js/pages/IndexPage.jsx` (lines 9-23 state init area)
- Modify: `web/src/js/components/SearchBar.jsx` (line 5 sync logic)

- [ ] **Step 1: Modify IndexPage.jsx to read ?q= on mount**

In `web/src/js/pages/IndexPage.jsx`, add a new `useEffect` after the auth guard (after line 33). This reads the URL `?q=` parameter on initial mount and sets the search query:

```javascript
  // Read ?q= URL parameter on initial mount (for Alfred "linkstash" trigger)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const q = params.get('q');
    if (q) {
      setQuery(q);
      setSearchType('keyword');
    }
  }, []);
```

Insert this block between the existing auth guard `useEffect` (lines 29-33) and the "Fetch categories" `useEffect` (lines 36-43).

- [ ] **Step 2: Update SearchBar.jsx sync to handle initial query from URL param**

In `web/src/js/components/SearchBar.jsx`, the existing sync effect (lines 9-13) only resets `localQuery` when parent `query` becomes empty. It also needs to sync when query is set from a URL param (non-empty initial value). Replace lines 8-13:

Current code:
```javascript
  // Sync local query when parent clears it (e.g. ESC key)
  useEffect(() => {
    if (query === '' && localQuery !== '') {
      setLocalQuery('');
    }
  }, [query]);
```

New code:
```javascript
  // Sync local query with parent (handles ESC clear and URL param init)
  useEffect(() => {
    setLocalQuery(query || '');
  }, [query]);
```

This ensures when `IndexPage` sets `query` from the URL param, `SearchBar`'s local input also updates.

- [ ] **Step 3: Build and verify**

```bash
make frontend-js
```

Expected: esbuild bundles successfully with no errors.

- [ ] **Step 4: Manual test**

Start the server and open in browser:
```bash
make start
```

Test URL param search by navigating to: `http://localhost:8888/?q=claude`

Expected: Page loads with "claude" pre-filled in search bar and results displayed.

Test normal flow still works: Navigate to `http://localhost:8888/`, type a search manually.

Expected: Behaves exactly as before.

- [ ] **Step 5: Commit**

```bash
git add web/src/js/pages/IndexPage.jsx web/src/js/components/SearchBar.jsx
git commit -m "feat(frontend): support ?q= URL parameter for external search triggers"
```

---

### Task 5: End-to-end manual testing and README

**Files:**
- Create: `extend_plugins/alfred/README.md`

- [ ] **Step 1: Test lsearch mode end-to-end**

With the server running (`make start`), test the Script Filter from command line:

```bash
cd extend_plugins/alfred/LinkStash.alfredworkflow
LINKSTASH_SERVER="http://localhost:8888" LINKSTASH_SECRET_KEY="your-actual-key" python3 lsearch.py claude
```

Verify output:
1. Valid JSON with `"items"` array
2. Each item has `title`, `subtitle`, `arg` (URL), `uid`
3. Each item has `mods.cmd.arg` (same URL for copy) and `mods.alt.arg` (LinkStash detail URL)

Test error cases:

```bash
# Missing config
python3 lsearch.py claude
# Expected: {"items": [{"title": "Configuration missing", ...}]}

# Wrong secret key
LINKSTASH_SERVER="http://localhost:8888" LINKSTASH_SECRET_KEY="wrong" python3 lsearch.py claude
# Expected: {"items": [{"title": "Auth failed", ...}]}

# Empty query
LINKSTASH_SERVER="http://localhost:8888" LINKSTASH_SECRET_KEY="your-key" python3 lsearch.py
# Expected: {"items": [{"title": "Type to search LinkStash...", ...}]}
```

- [ ] **Step 2: Test linkstash mode (browser redirect)**

Open in browser: `http://localhost:8888/?q=test`

Verify:
1. Search bar shows "test" pre-filled
2. Results are displayed
3. Clear button works to reset

- [ ] **Step 3: Test token caching**

```bash
# First call creates token cache
LINKSTASH_SERVER="http://localhost:8888" LINKSTASH_SECRET_KEY="your-key" python3 lsearch.py claude

# Verify token cached
cat ~/.linkstash/token
# Expected: JWT string

# Second call uses cached token (faster, no auth request)
LINKSTASH_SERVER="http://localhost:8888" LINKSTASH_SECRET_KEY="your-key" python3 lsearch.py test
```

- [ ] **Step 4: Write installation README**

Create `extend_plugins/alfred/README.md`:

```markdown
# LinkStash Alfred Workflow

Search your LinkStash bookmarks directly from Alfred.

## Installation

1. Double-click `LinkStash.alfredworkflow/` directory or import it via Alfred Preferences
2. In Alfred Preferences → Workflows → LinkStash Search, configure:
   - **Server URL**: Your LinkStash server (e.g. `http://localhost:8888`)
   - **Secret Key**: Your authentication secret key

## Usage

### `lsearch {query}` — Native Alfred Search

Search bookmarks with results displayed in Alfred:

- **Enter** — Open URL in browser
- **⌘+Enter** — Copy URL to clipboard
- **⌥+Enter** — Open bookmark in LinkStash web UI

### `linkstash {query}` — Web Search

Opens LinkStash web UI in your browser with the search query pre-filled.
Useful for advanced filtering (category, network type, search type).

## Configuration

| Environment Variable | Description |
|---------------------|-------------|
| `LINKSTASH_SERVER` | Server URL (e.g. `http://localhost:8888`) |
| `LINKSTASH_SECRET_KEY` | Authentication secret key |

Token is cached at `~/.linkstash/token` and auto-refreshes on expiry.

## Troubleshooting

Test from terminal:
```bash
cd /path/to/LinkStash.alfredworkflow
LINKSTASH_SERVER="http://localhost:8888" LINKSTASH_SECRET_KEY="your-key" python3 lsearch.py test
```

If results don't appear, check:
1. Server is running (`make start` in linkstash root)
2. Environment variables are set in Alfred workflow settings
3. Token file: `cat ~/.linkstash/token` (delete to force re-auth)
```

- [ ] **Step 5: Commit**

```bash
git add extend_plugins/alfred/README.md
git commit -m "docs(alfred): add installation and usage README"
```
