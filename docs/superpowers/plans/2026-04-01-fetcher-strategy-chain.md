# Fetcher Strategy Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the rigid browser-first URL fetching with a configurable strategy chain (`http` → `browser` → `browser-proxy`) that degrades gracefully based on YAML config, with on-demand browser lifecycle to minimize memory usage.

**Architecture:** New `FetcherConfig` in config.go defines an ordered list of strategies. A `ChainFetcher` iterates through configured fetchers (HTTPFetcher, BrowserFetcher) in order. The existing `BrowserFetcher` interface in `worker_service.go` is replaced by a simpler `Fetcher` interface. Browser launches on-demand per fetch and closes immediately after.

**Tech Stack:** Go, Rod (headless Chrome), net/http, YAML config, Google Wire DI

**Spec:** `docs/superpowers/specs/2026-04-01-fetcher-strategy-chain-design.md`

---

### Task 1: Add FetcherConfig to config

**Files:**
- Modify: `app/infra/config/config.go`
- Modify: `conf/app_example.yaml`

- [ ] **Step 1: Add FetcherConfig structs to config.go**

Add after the existing `BrowserConfig` struct (around line 38):

```go
// FetcherConfig defines the URL content fetching strategy chain.
type FetcherConfig struct {
	Strategies []string      `yaml:"strategies"` // Ordered: "http", "browser", "browser-proxy"
	HTTP       HTTPFetchConfig   `yaml:"http"`
	Browser    BrowserFetchConfig `yaml:"browser"`
}

type HTTPFetchConfig struct {
	TimeoutSec int    `yaml:"timeout_sec"` // Default 15
	MaxContent int    `yaml:"max_content"` // Default 51200 (50KB)
	UserAgent  string `yaml:"user_agent"`
}

type BrowserFetchConfig struct {
	TimeoutSec int    `yaml:"timeout_sec"` // Default 30
	MaxContent int    `yaml:"max_content"` // Default 51200 (50KB)
	Lifecycle  string `yaml:"lifecycle"`   // "on-demand" or "singleton"
}
```

- [ ] **Step 2: Add `Fetcher` field to Config struct**

In the `Config` struct (line 12–22), add:

```go
type Config struct {
	Server     ServerConfig   `yaml:"server"`
	Auth       AuthConfig     `yaml:"auth"`
	Database   DatabaseConfig `yaml:"database"`
	Log        LogConfig      `yaml:"log"`
	LLM        LLMConfig      `yaml:"llm"`
	Short      ShortConfig    `yaml:"short"`
	Categories []string       `yaml:"categories"`
	Proxy      ProxyConfig    `yaml:"proxy"`
	Browser    BrowserConfig  `yaml:"browser"`
	Fetcher    FetcherConfig  `yaml:"fetcher"`  // NEW
}
```

- [ ] **Step 3: Set defaults in Load function**

In `Load()` function, after existing defaults (around line 220), add:

```go
	// Fetcher defaults
	if len(cfg.Fetcher.Strategies) == 0 {
		if cfg.Browser.Enabled {
			cfg.Fetcher.Strategies = []string{"http", "browser"}
		} else {
			cfg.Fetcher.Strategies = []string{"http"}
		}
	}
	if cfg.Fetcher.HTTP.TimeoutSec == 0 {
		cfg.Fetcher.HTTP.TimeoutSec = 15
	}
	if cfg.Fetcher.HTTP.MaxContent == 0 {
		cfg.Fetcher.HTTP.MaxContent = 51200
	}
	if cfg.Fetcher.HTTP.UserAgent == "" {
		cfg.Fetcher.HTTP.UserAgent = "LinkStash/1.0 (+https://github.com/lupguo/linkstash)"
	}
	if cfg.Fetcher.Browser.TimeoutSec == 0 {
		cfg.Fetcher.Browser.TimeoutSec = 30
	}
	if cfg.Fetcher.Browser.MaxContent == 0 {
		cfg.Fetcher.Browser.MaxContent = 51200
	}
	if cfg.Fetcher.Browser.Lifecycle == "" {
		cfg.Fetcher.Browser.Lifecycle = "on-demand"
	}
```

- [ ] **Step 4: Add fetcher config to app_example.yaml**

Add before the `browser:` section:

```yaml
# URL content fetching strategy chain
# Strategies are tried in order; first success wins, failures fall through
# Available: "http" (Go net/http), "browser" (Rod headless Chrome), "browser-proxy" (Chrome + proxy)
# Default: ["http", "browser"] if browser.enabled=true, ["http"] otherwise
fetcher:
  strategies: ["http"]    # Low-memory servers: use only HTTP
  # strategies: ["http", "browser"]  # HTTP first, browser fallback for JS-rendered pages
  http:
    timeout_sec: 15
    max_content: 51200      # 50KB
    user_agent: "LinkStash/1.0 (+https://github.com/lupguo/linkstash)"
  browser:
    timeout_sec: 30
    max_content: 51200
    lifecycle: "on-demand"  # "on-demand" = launch/close per fetch, "singleton" = persistent
```

- [ ] **Step 5: Commit**

```bash
git add app/infra/config/config.go conf/app_example.yaml
git commit -m "feat: add FetcherConfig for configurable fetch strategies"
```

---

### Task 2: Create Fetcher interface and HTTPFetcher

**Files:**
- Create: `app/infra/fetcher/fetcher.go`
- Create: `app/infra/fetcher/http_fetcher.go`

- [ ] **Step 1: Create the Fetcher interface**

Create `app/infra/fetcher/fetcher.go`:

```go
package fetcher

import "context"

// Fetcher fetches page content from a URL.
type Fetcher interface {
	Name() string
	Fetch(ctx context.Context, url string) (string, error)
}
```

- [ ] **Step 2: Create HTTPFetcher**

Create `app/infra/fetcher/http_fetcher.go`:

```go
package fetcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lupguo/linkstash/app/infra/config"
)

// HTTPFetcher fetches page content using Go net/http.
type HTTPFetcher struct {
	client     *http.Client
	maxContent int
	userAgent  string
}

func NewHTTPFetcher(cfg config.HTTPFetchConfig) *HTTPFetcher {
	return &HTTPFetcher{
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		maxContent: cfg.MaxContent,
		userAgent:  cfg.UserAgent,
	}
}

func (f *HTTPFetcher) Name() string { return "http" }

func (f *HTTPFetcher) Fetch(ctx context.Context, rawURL string) (string, error) {
	// Try the URL directly first
	content, err := f.doFetch(ctx, rawURL)
	if err == nil {
		return content, nil
	}

	slog.Info("http fetch failed, trying root domain", "url", rawURL, "error", err)

	// Fallback: try root domain (e.g. https://example.com/path → https://example.com)
	parsed, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", err // return original error
	}
	rootURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	if rootURL == rawURL {
		return "", err
	}

	content, rootErr := f.doFetch(ctx, rootURL)
	if rootErr != nil {
		return "", fmt.Errorf("direct: %w; root: %v", err, rootErr)
	}
	return content, nil
}

func (f *HTTPFetcher) doFetch(ctx context.Context, fetchURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fetchURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(f.maxContent)))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	content := string(body)

	// Detect Cloudflare challenge pages
	if isCloudflareChallenge(content) {
		return "", fmt.Errorf("cloudflare challenge detected")
	}

	return content, nil
}

func isCloudflareChallenge(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "checking your browser") ||
		strings.Contains(lower, "cf-browser-verification") ||
		strings.Contains(lower, "just a moment")
}
```

- [ ] **Step 3: Commit**

```bash
git add app/infra/fetcher/
git commit -m "feat: add Fetcher interface and HTTPFetcher implementation"
```

---

### Task 3: Create BrowserFetcher with on-demand lifecycle

**Files:**
- Create: `app/infra/fetcher/browser_fetcher.go`

- [ ] **Step 1: Create BrowserFetcher**

Create `app/infra/fetcher/browser_fetcher.go`:

```go
package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/lupguo/linkstash/app/infra/config"
)

// BrowserFetcher fetches page content using headless Chrome via Rod.
// On-demand lifecycle: launches Chrome per fetch, closes immediately after.
type BrowserFetcher struct {
	binPath    string
	proxyURL   string
	timeoutSec int
	maxContent int
	headless   bool
}

func NewBrowserFetcher(browserCfg config.BrowserConfig, fetchCfg config.BrowserFetchConfig, proxyURL string) *BrowserFetcher {
	return &BrowserFetcher{
		binPath:    browserCfg.BinPath,
		proxyURL:   proxyURL,
		timeoutSec: fetchCfg.TimeoutSec,
		maxContent: fetchCfg.MaxContent,
		headless:   browserCfg.IsHeadless(),
	}
}

func (f *BrowserFetcher) Name() string { return "browser" }

func (f *BrowserFetcher) Fetch(ctx context.Context, url string) (string, error) {
	return f.fetchWithBrowser(ctx, url, "")
}

// BrowserProxyFetcher is the same as BrowserFetcher but always uses proxy.
type BrowserProxyFetcher struct {
	*BrowserFetcher
}

func NewBrowserProxyFetcher(browserCfg config.BrowserConfig, fetchCfg config.BrowserFetchConfig, proxyURL string) *BrowserProxyFetcher {
	return &BrowserProxyFetcher{
		BrowserFetcher: NewBrowserFetcher(browserCfg, fetchCfg, proxyURL),
	}
}

func (f *BrowserProxyFetcher) Name() string { return "browser-proxy" }

func (f *BrowserProxyFetcher) Fetch(ctx context.Context, url string) (string, error) {
	if f.proxyURL == "" {
		return "", fmt.Errorf("browser-proxy: no proxy configured")
	}
	return f.fetchWithBrowser(ctx, url, f.proxyURL)
}

// fetchWithBrowser launches a browser, fetches the page, and closes the browser.
func (f *BrowserFetcher) fetchWithBrowser(ctx context.Context, pageURL string, proxyURL string) (string, error) {
	// Launch browser (on-demand)
	br, err := f.launchBrowser(proxyURL)
	if err != nil {
		return "", fmt.Errorf("launch browser: %w", err)
	}
	defer func() {
		if closeErr := br.Close(); closeErr != nil {
			slog.Warn("failed to close browser", "error", closeErr)
		}
	}()

	// Create stealth page
	page, err := stealth.Page(br)
	if err != nil {
		return "", fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	timeout := time.Duration(f.timeoutSec) * time.Second
	page = page.Timeout(timeout)

	// Override user agent
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	_ = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: ua})

	// Navigate
	if err := page.Navigate(pageURL); err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		slog.Warn("page load wait failed, continuing", "url", pageURL, "error", err)
	}

	// Wait for JS rendering
	time.Sleep(2 * time.Second)

	// Extract HTML
	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("get html: %w", err)
	}

	// Limit content size
	if len(html) > f.maxContent {
		html = html[:f.maxContent]
	}

	return html, nil
}

func (f *BrowserFetcher) launchBrowser(proxyURL string) (*rod.Browser, error) {
	l := launcher.New()

	if f.binPath != "" {
		l = l.Bin(f.binPath)
	}

	l = l.Headless(f.headless).
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-infobars").
		Set("disable-extensions").
		Set("window-size", "1920,1080").
		Set("lang", "en-US")

	if proxyURL != "" {
		l = l.Proxy(proxyURL)
	}

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch: %w", err)
	}

	br := rod.New().ControlURL(u)
	if err := br.Connect(); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return br, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add app/infra/fetcher/browser_fetcher.go
git commit -m "feat: add BrowserFetcher with on-demand lifecycle"
```

---

### Task 4: Create ChainFetcher

**Files:**
- Create: `app/infra/fetcher/chain.go`

- [ ] **Step 1: Create ChainFetcher**

Create `app/infra/fetcher/chain.go`:

```go
package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lupguo/linkstash/app/infra/config"
)

// ChainFetcher tries each fetcher in order, returning the first successful result.
type ChainFetcher struct {
	fetchers []Fetcher
}

// NewChainFetcher builds a chain from the configured strategy list.
func NewChainFetcher(cfg *config.Config) *ChainFetcher {
	var fetchers []Fetcher

	for _, strategy := range cfg.Fetcher.Strategies {
		switch strings.TrimSpace(strings.ToLower(strategy)) {
		case "http":
			fetchers = append(fetchers, NewHTTPFetcher(cfg.Fetcher.HTTP))
			slog.Info("fetcher registered", "strategy", "http")

		case "browser":
			if !cfg.Browser.Enabled {
				slog.Warn("fetcher strategy 'browser' configured but browser.enabled=false, skipping")
				continue
			}
			fetchers = append(fetchers, NewBrowserFetcher(cfg.Browser, cfg.Fetcher.Browser, ""))
			slog.Info("fetcher registered", "strategy", "browser")

		case "browser-proxy":
			if !cfg.Browser.Enabled {
				slog.Warn("fetcher strategy 'browser-proxy' configured but browser.enabled=false, skipping")
				continue
			}
			if cfg.Proxy.HTTPProxy == "" {
				slog.Warn("fetcher strategy 'browser-proxy' configured but no proxy set, skipping")
				continue
			}
			fetchers = append(fetchers, NewBrowserProxyFetcher(cfg.Browser, cfg.Fetcher.Browser, cfg.Proxy.HTTPProxy))
			slog.Info("fetcher registered", "strategy", "browser-proxy")

		default:
			slog.Warn("unknown fetcher strategy, skipping", "strategy", strategy)
		}
	}

	if len(fetchers) == 0 {
		slog.Warn("no fetchers configured, falling back to http-only")
		fetchers = append(fetchers, NewHTTPFetcher(cfg.Fetcher.HTTP))
	}

	return &ChainFetcher{fetchers: fetchers}
}

// Fetch tries each fetcher in order.
func (c *ChainFetcher) Fetch(ctx context.Context, url string) (string, error) {
	var errs []string

	for _, f := range c.fetchers {
		content, err := f.Fetch(ctx, url)
		if err != nil {
			slog.Info("fetcher failed, trying next", "fetcher", f.Name(), "url", url, "error", err)
			errs = append(errs, fmt.Sprintf("%s: %v", f.Name(), err))
			continue
		}
		slog.Info("fetcher succeeded", "fetcher", f.Name(), "url", url, "content_len", len(content))
		return content, nil
	}

	return "", fmt.Errorf("all fetchers failed for %s: %s", url, strings.Join(errs, "; "))
}

// Name returns a description of the chain.
func (c *ChainFetcher) Name() string {
	names := make([]string, len(c.fetchers))
	for i, f := range c.fetchers {
		names[i] = f.Name()
	}
	return "chain[" + strings.Join(names, "→") + "]"
}
```

- [ ] **Step 2: Commit**

```bash
git add app/infra/fetcher/chain.go
git commit -m "feat: add ChainFetcher with ordered strategy fallback"
```

---

### Task 5: Integrate ChainFetcher into WorkerService

**Files:**
- Modify: `app/domain/services/worker_service.go`

- [ ] **Step 1: Replace BrowserFetcher interface with Fetcher interface**

At lines 21–26, replace the `BrowserFetcher` interface:

```go
// ContentFetcher fetches page content from a URL using a configured strategy chain.
type ContentFetcher interface {
	Fetch(ctx context.Context, url string) (string, error)
	Name() string
}
```

- [ ] **Step 2: Update WorkerService struct**

Replace the `browserSvc`, `httpClient` fields (lines 28–38):

```go
type WorkerService struct {
	queue         chan uint
	urlRepo       repos.URLRepo
	llmLogRepo    repos.LLMLogRepo
	embeddingRepo repos.EmbeddingRepo
	llmClient     *llm.LLMClient
	fetcher       ContentFetcher  // CHANGED: was browserSvc + httpClient
	prompts       map[string]string
	done          chan struct{}
}
```

- [ ] **Step 3: Update constructor**

Replace `NewWorkerService` (lines 40–63):

```go
func NewWorkerService(
	urlRepo repos.URLRepo,
	llmLogRepo repos.LLMLogRepo,
	embeddingRepo repos.EmbeddingRepo,
	llmClient *llm.LLMClient,
	prompts map[string]string,
	fetcher ContentFetcher,
) *WorkerService {
	return &WorkerService{
		queue:         make(chan uint, 100),
		urlRepo:       urlRepo,
		llmLogRepo:    llmLogRepo,
		embeddingRepo: embeddingRepo,
		llmClient:     llmClient,
		fetcher:       fetcher,
		prompts:       prompts,
		done:          make(chan struct{}),
	}
}
```

- [ ] **Step 4: Simplify fetchPageContent**

Replace `fetchPageContent`, `fetchWithBrowser`, `fetchWithHTTP`, `doFetch` methods (lines 236–356) with a single delegation:

```go
func (w *WorkerService) fetchPageContent(link string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return w.fetcher.Fetch(ctx, link)
}
```

Delete the old `fetchWithBrowser`, `fetchWithHTTP`, `doFetch`, and `isBlockedContent` methods — all of that logic now lives in the fetcher package.

Keep the `isCloudflareBlock` check in `doProcessURL` if it exists there, otherwise it's handled by HTTPFetcher.

- [ ] **Step 5: Remove unused imports**

Remove `net/http` import from worker_service.go (the HTTP client is no longer a field).

- [ ] **Step 6: Commit**

```bash
git add app/domain/services/worker_service.go
git commit -m "refactor: replace browser+http fetching with ChainFetcher in worker"
```

---

### Task 6: Update Wire DI

**Files:**
- Modify: `app/di/wire.go`
- Regenerate: `app/di/wire_gen.go` (via `make wire`)

- [ ] **Step 1: Add ChainFetcher provider, remove old BrowserService provider**

In `wire.go`, replace `ProvideBrowserService` (lines 58-60) and update `ProvideWorkerService` (lines 78-88):

```go
// ProvideChainFetcher builds the configurable fetch strategy chain.
func ProvideChainFetcher(cfg *config.Config) *fetcher.ChainFetcher {
	return fetcher.NewChainFetcher(cfg)
}

// ProvideWorkerService creates the worker with ChainFetcher.
func ProvideWorkerService(
	urlRepo repos.URLRepo,
	llmLogRepo repos.LLMLogRepo,
	embeddingRepo repos.EmbeddingRepo,
	llmClient *llm.LLMClient,
	cfg *config.Config,
	chainFetcher *fetcher.ChainFetcher,
) *services.WorkerService {
	return services.NewWorkerService(
		urlRepo, llmLogRepo, embeddingRepo,
		llmClient, cfg.LLM.Prompts, chainFetcher,
	)
}
```

- [ ] **Step 2: Update InfraSet**

Replace `ProvideBrowserService` with `ProvideChainFetcher` in InfraSet, and remove `ProvideHTTPClient`:

```go
var InfraSet = wire.NewSet(
	ProvideConfig,
	ProvideDB,
	ProvideLLMClient,
	ProvideChainFetcher,    // CHANGED: was ProvideBrowserService + ProvideHTTPClient
	ProvideKeywordSearch,
	ProvideVectorSearch,
)
```

- [ ] **Step 3: Add import for fetcher package**

Add to imports in wire.go:

```go
"github.com/lupguo/linkstash/app/infra/fetcher"
```

Remove unused import for `browser` package if no longer referenced.

- [ ] **Step 4: Regenerate wire code**

```bash
make wire
```

Verify `wire_gen.go` compiles correctly.

- [ ] **Step 5: Remove old ProvideHTTPClient if unused**

Check if `ProvideHTTPClient` is used anywhere else. If not, remove it from `wire.go`.

- [ ] **Step 6: Commit**

```bash
git add app/di/wire.go app/di/wire_gen.go
git commit -m "refactor: wire ChainFetcher into DI, remove old browser/http providers"
```

---

### Task 7: Update graceful shutdown

**Files:**
- Modify: `cmd/server/main.go` (if it calls `browserSvc.Close()`)

- [ ] **Step 1: Check and remove browser close on shutdown**

The old code may have `browserSvc.Close()` in the shutdown handler. Since on-demand browsers close themselves after each fetch, remove any global browser close calls.

Search for `browserSvc.Close()` or `BrowserService.Close()` in `cmd/server/main.go` and the DI `App` struct. Remove them.

- [ ] **Step 2: Verify build**

```bash
make build
```

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "chore: remove browser close from shutdown (on-demand lifecycle)"
```

---

### Task 8: Update dev config and test

**Files:**
- Modify: `conf/app_dev.yaml`

- [ ] **Step 1: Add fetcher config to app_dev.yaml**

Add the fetcher block (HTTP-only for your low-memory server):

```yaml
fetcher:
  strategies: ["http"]
  http:
    timeout_sec: 15
    max_content: 51200
    user_agent: "LinkStash/1.0 (+https://github.com/lupguo/linkstash)"
```

- [ ] **Step 2: Full build and verify**

```bash
make build
make start
```

Verify in logs:
- Should see `fetcher registered strategy=http`
- Should NOT see any browser launch attempts
- Add a URL and verify analysis completes via HTTP fetcher

- [ ] **Step 3: Stop and test with browser strategy**

Temporarily change to `strategies: ["http", "browser"]` and `browser.enabled: true`, rebuild, add a URL. Verify:
- HTTP is tried first
- If HTTP fails (e.g. Cloudflare), browser is launched and closed after fetch
- Memory returns to baseline after analysis completes

- [ ] **Step 4: Commit config**

```bash
git add -f conf/app_dev.yaml
git commit -m "chore: set fetcher strategies to http-only for dev server"
```

---

### Task 9: Cleanup old browser code

**Files:**
- Remove or archive: `app/infra/browser/fetcher.go` (old FetchPage method)
- Remove or archive: `app/infra/browser/browser.go` (old singleton management)
- Keep: `app/infra/browser/` package if BrowserConfig is still referenced from config

- [ ] **Step 1: Check if old browser package is still imported anywhere**

```bash
grep -r "infra/browser" app/ --include="*.go" | grep -v "_test.go"
```

If nothing imports it (Wire no longer references it), the old `browser/` package is dead code.

- [ ] **Step 2: Remove old browser package if unused**

```bash
rm -rf app/infra/browser/
```

If `config.BrowserConfig` was in that package, it's already in `config/config.go` so this is safe.

- [ ] **Step 3: Final build verification**

```bash
make build
make test
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove old browser singleton code, replaced by fetcher package"
```

---

### Task 10: Tag release

- [ ] **Step 1: Tag and push**

```bash
git tag -a v0.5.2 -m "feat: configurable fetcher strategy chain (http/browser/browser-proxy)"
git push origin main
git push origin v0.5.2
```

- [ ] **Step 2: Verify CI builds successfully**

```bash
gh run watch $(gh run list --limit 1 --json databaseId --jq '.[0].databaseId')
```

- [ ] **Step 3: Verify release on GitHub**

```bash
gh release view v0.5.2
```
