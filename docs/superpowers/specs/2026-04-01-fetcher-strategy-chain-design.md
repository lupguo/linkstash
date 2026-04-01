# Configurable Fetcher Strategy Chain

**Date:** 2026-04-01
**Status:** Approved
**Problem:** Browser-based URL fetching consumes 500MB+ memory on a 1.7GB server, crashing the system. Additionally, the Rod browser binary is missing on the server, causing all URL analysis to fail.

## Design

### Configuration

New `fetcher` block in YAML config:

```yaml
fetcher:
  # Ordered strategy chain — try each in order, fall through on failure
  # Available: "http", "browser", "browser-proxy"
  # Default if omitted: ["http", "browser"]
  strategies: ["http"]

  http:
    timeout: 15s          # per-request timeout
    max_content: 51200     # 50KB max page content
    user_agent: "LinkStash/1.0 (+https://github.com/lupguo/linkstash)"

  browser:
    timeout: 30s
    max_content: 51200
    lifecycle: "on-demand"  # "on-demand" (launch/close per fetch) or "singleton" (persistent)
```

When `fetcher` block is absent, default to `strategies: ["http", "browser"]` for backward compatibility.

### Fetcher Interface

```go
// Fetcher fetches page content from a URL.
type Fetcher interface {
    Name() string
    Fetch(ctx context.Context, url string) (content string, err error)
}
```

Three implementations:
- `HTTPFetcher` — Go `net/http` client, follows redirects, reads body up to max_content
- `BrowserFetcher` — Rod headless Chrome, direct connection
- `BrowserProxyFetcher` — Rod headless Chrome through configured proxy

### Strategy Chain

```go
// ChainFetcher tries each fetcher in order, returns first success.
type ChainFetcher struct {
    fetchers []Fetcher
}

func (c *ChainFetcher) Fetch(ctx context.Context, url string) (string, error) {
    var lastErr error
    for _, f := range c.fetchers {
        content, err := f.Fetch(ctx, url)
        if err != nil {
            slog.Info("fetcher failed, trying next", "fetcher", f.Name(), "error", err)
            lastErr = err
            continue
        }
        return content, nil
    }
    return "", fmt.Errorf("all fetchers failed: %w", lastErr)
}
```

### Browser Lifecycle: On-Demand

Current singleton pattern (two global `*rod.Browser` instances) replaced with on-demand:

```
Fetch request → Launch Chrome → Navigate → Extract content → Close Chrome
```

- Each `BrowserFetcher.Fetch()` call launches a fresh Chrome process
- Chrome is closed immediately after content extraction (deferred)
- Idle memory: 0 (no persistent browser process)
- Trade-off: ~2-3s launch overhead per fetch (acceptable for async background worker)

The `lifecycle` config allows future `"singleton"` mode for high-volume servers.

### HTTPFetcher Details

- Uses `net/http.Client` with configurable timeout and custom User-Agent
- Follows redirects (up to 10)
- Reads response body up to `max_content` bytes
- Returns raw HTML content
- Handles common errors: timeout, DNS failure, TLS errors, non-2xx status
- Cloudflare challenge detection: if response contains known CF challenge markers, return error to trigger fallback to browser

### Changes to Worker

Current flow in `app/application/url_usecase.go`:
```
processURL() → browserFetcher.Fetch() → llmClient.Analyze() → save
```

New flow:
```
processURL() → chainFetcher.Fetch() → llmClient.Analyze() → save
```

The worker receives a `ChainFetcher` instead of `*browser.Fetcher`. No other worker logic changes needed.

### Wire DI Changes

In `app/di/wire.go`:
- New provider: `NewChainFetcher(cfg *config.Config) *fetcher.ChainFetcher`
- This provider reads `cfg.Fetcher.Strategies` and constructs only the configured fetchers
- If `"browser"` is not in strategies, no Rod code is initialized at all
- Inject `ChainFetcher` into `URLUsecase` replacing the current `*browser.Fetcher`

### Config Struct

```go
type FetcherConfig struct {
    Strategies []string     `yaml:"strategies"`
    HTTP       HTTPConfig   `yaml:"http"`
    Browser    BrowserConfig `yaml:"browser"`
}

type HTTPConfig struct {
    Timeout    time.Duration `yaml:"timeout"`
    MaxContent int           `yaml:"max_content"`
    UserAgent  string        `yaml:"user_agent"`
}

type BrowserConfig struct {
    Timeout    time.Duration `yaml:"timeout"`
    MaxContent int           `yaml:"max_content"`
    Lifecycle  string        `yaml:"lifecycle"`
}
```

### Files to Modify

| File | Change |
|------|--------|
| `app/infra/config/config.go` | Add `FetcherConfig` struct |
| `app/infra/browser/fetcher.go` | Refactor to implement `Fetcher` interface, add on-demand lifecycle |
| `app/infra/browser/http_fetcher.go` | **New file** — HTTP-based fetcher |
| `app/infra/browser/chain.go` | **New file** — ChainFetcher |
| `app/application/url_usecase.go` | Accept `Fetcher` interface instead of `*browser.Fetcher` |
| `app/di/wire.go` | Wire ChainFetcher provider |
| `conf/app_example.yaml` | Add `fetcher` config block with comments |
| `conf/app_dev.yaml` | Set `strategies: ["http"]` for low-memory dev server |

### Default Behavior

- No `fetcher` config → `strategies: ["http", "browser"]`, HTTP timeouts 15s, browser timeout 30s
- Existing deployments continue working without config changes
- Low-memory servers: set `strategies: ["http"]` to disable browser entirely
