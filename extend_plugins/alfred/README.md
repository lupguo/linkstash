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
