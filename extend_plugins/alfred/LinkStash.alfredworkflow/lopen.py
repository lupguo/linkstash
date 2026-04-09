#!/usr/bin/env python3
"""LinkStash Alfred Workflow — Open URL and record visit asynchronously."""

import json
import os
import subprocess
import sys
import threading
import urllib.request
import urllib.error
from pathlib import Path

# ---------------------------------------------------------------------------
# Configuration (shared with lsearch.py)
# ---------------------------------------------------------------------------

SERVER_URL = os.environ.get("LINKSTASH_SERVER", "").rstrip("/")
SECRET_KEY = os.environ.get("LINKSTASH_SECRET_KEY", "")
TOKEN_PATH = Path.home() / ".linkstash" / "token"
TIMEOUT_SECONDS = 3


# ---------------------------------------------------------------------------
# Token management (mirrors lsearch.py)
# ---------------------------------------------------------------------------

def read_cached_token():
    try:
        token = TOKEN_PATH.read_text().strip()
        return token if token else None
    except (FileNotFoundError, OSError):
        return None


def save_token(token):
    TOKEN_PATH.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    TOKEN_PATH.write_text(token)
    TOKEN_PATH.chmod(0o600)


def exchange_token():
    if not SERVER_URL or not SECRET_KEY:
        return None
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
            if token:
                save_token(token)
            return token
    except (urllib.error.HTTPError, urllib.error.URLError):
        return None


def get_token():
    token = read_cached_token()
    if token:
        return token
    return exchange_token()


# ---------------------------------------------------------------------------
# Visit recording (fire-and-forget)
# ---------------------------------------------------------------------------

def record_visit(url_id, token):
    """POST /api/urls/{id}/visit to increment visit count."""
    url = f"{SERVER_URL}/api/urls/{url_id}/visit"
    req = urllib.request.Request(
        url,
        data=b"",
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT_SECONDS) as resp:
            resp.read()
    except urllib.error.HTTPError as e:
        if e.code == 401:
            # Token expired — refresh and retry once
            new_token = exchange_token()
            if new_token:
                req.remove_header("Authorization")
                req.add_header("Authorization", f"Bearer {new_token}")
                try:
                    with urllib.request.urlopen(req, timeout=TIMEOUT_SECONDS) as resp:
                        resp.read()
                except (urllib.error.HTTPError, urllib.error.URLError):
                    pass  # Best-effort, don't block user
    except (urllib.error.URLError, OSError):
        pass  # Best-effort


def record_visit_async(url_id):
    """Record visit in a background thread so it doesn't delay URL opening."""
    token = get_token()
    if not token or not url_id:
        return
    t = threading.Thread(target=record_visit, args=(url_id, token), daemon=True)
    t.start()
    # Wait briefly so the request has time to fire before process exits
    t.join(timeout=2.0)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    # Alfred passes the selected item's arg (the URL) as {query}
    link = sys.argv[1] if len(sys.argv) > 1 else ""
    url_id = os.environ.get("url_id", "")

    if not link:
        sys.exit(1)

    # 1. Open URL immediately via macOS `open` command
    subprocess.Popen(["open", link])

    # 2. Record visit asynchronously (best-effort, non-blocking)
    if url_id and SERVER_URL:
        record_visit_async(url_id)


if __name__ == "__main__":
    main()
