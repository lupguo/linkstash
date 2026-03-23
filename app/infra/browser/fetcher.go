package browser

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/stealth"
)

// FetchPage fetches a URL using the headless browser and returns the HTML content.
// If useProxy is true, it uses the proxy-enabled browser instance (Level 3);
// otherwise, it uses the direct browser instance (Level 2).
// Content is limited to 50KB to stay consistent with HTTP fetch limits.
func (s *BrowserService) FetchPage(ctx context.Context, url string, useProxy bool) (string, error) {
	if !s.cfg.Enabled {
		return "", fmt.Errorf("browser not enabled")
	}

	var br *rod.Browser
	var err error
	if useProxy {
		br, err = s.getProxyBrowser()
	} else {
		br, err = s.getDirectBrowser()
	}
	if err != nil {
		return "", fmt.Errorf("get browser: %w", err)
	}

	level := "L2-direct"
	if useProxy {
		level = "L3-proxy"
	}
	slog.Info("browser fetching page", "component", "browser", "level", level, "url", url)

	// Create a stealth page to bypass anti-bot detection
	page, err := stealth.Page(br)
	if err != nil {
		return "", fmt.Errorf("create stealth page: %w", err)
	}
	defer func() {
		if err := page.Close(); err != nil {
			slog.Warn("failed to close page", "component", "browser", "error", err)
		}
	}()

	// Set timeout for page operations
	timeout := s.PageTimeout()
	page = page.Timeout(timeout)

	// Navigate to the URL
	if err := page.Navigate(url); err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}

	// Wait for the page to load
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("wait load: %w", err)
	}

	// Wait for page to stabilize (dynamic content, JS rendering)
	if err := page.WaitStable(2 * time.Second); err != nil {
		// WaitStable timeout is non-fatal; page may already have useful content
		slog.Debug("wait stable timeout (non-fatal)", "component", "browser", "url", url, "error", err)
	}

	// Get page HTML
	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("get html: %w", err)
	}

	// Limit content to 50KB (consistent with HTTP fetch)
	const maxSize = 50 * 1024
	if len(html) > maxSize {
		html = html[:maxSize]
	}

	slog.Info("browser fetch success", "component", "browser", "level", level, "url", url, "content_len", len(html))
	return html, nil
}
