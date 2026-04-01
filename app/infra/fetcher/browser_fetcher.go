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

type BrowserFetcher struct {
	binPath    string
	proxyURL   string
	timeoutSec int
	maxContent int
	headless   bool
}

func NewBrowserFetcher(cfg config.BrowserConfig, proxyURL string) *BrowserFetcher {
	return &BrowserFetcher{
		binPath:    cfg.BinPath,
		proxyURL:   proxyURL,
		timeoutSec: cfg.TimeoutSec,
		maxContent: cfg.MaxContent,
		headless:   cfg.IsHeadless(),
	}
}

func (f *BrowserFetcher) Name() string { return "browser" }

func (f *BrowserFetcher) Fetch(ctx context.Context, url string) (string, error) {
	return f.fetchWithBrowser(ctx, url, "")
}

type BrowserProxyFetcher struct {
	*BrowserFetcher
}

func NewBrowserProxyFetcher(cfg config.BrowserConfig, proxyURL string) *BrowserProxyFetcher {
	return &BrowserProxyFetcher{
		BrowserFetcher: NewBrowserFetcher(cfg, proxyURL),
	}
}

func (f *BrowserProxyFetcher) Name() string { return "browser-proxy" }

func (f *BrowserProxyFetcher) Fetch(ctx context.Context, url string) (string, error) {
	if f.proxyURL == "" {
		return "", fmt.Errorf("browser-proxy: no proxy configured")
	}
	return f.fetchWithBrowser(ctx, url, f.proxyURL)
}

func (f *BrowserFetcher) fetchWithBrowser(ctx context.Context, pageURL string, proxyURL string) (string, error) {
	br, err := f.launchBrowser(proxyURL)
	if err != nil {
		return "", fmt.Errorf("launch browser: %w", err)
	}
	defer func() {
		if closeErr := br.Close(); closeErr != nil {
			slog.Warn("failed to close browser", "error", closeErr)
		}
	}()

	page, err := stealth.Page(br)
	if err != nil {
		return "", fmt.Errorf("create page: %w", err)
	}
	defer page.Close()

	timeout := time.Duration(f.timeoutSec) * time.Second
	page = page.Timeout(timeout)

	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	_ = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: ua})

	if err := page.Navigate(pageURL); err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		slog.Warn("page load wait failed, continuing", "url", pageURL, "error", err)
	}

	time.Sleep(2 * time.Second)

	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("get html: %w", err)
	}

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
