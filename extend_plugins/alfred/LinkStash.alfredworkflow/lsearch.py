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
