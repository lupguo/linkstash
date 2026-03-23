package browser

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/lupguo/linkstash/app/infra/config"
)

// BrowserService manages Rod headless browser instances for page fetching.
// It lazily initializes two browser instances: one without proxy (Level 2)
// and one with proxy (Level 3).
type BrowserService struct {
	cfg config.BrowserConfig

	// proxyURL from config for Level 3
	proxyURL string

	mu          sync.Mutex
	directBr    *rod.Browser // Level 2: no proxy
	proxyBr     *rod.Browser // Level 3: with proxy
	directReady bool
	proxyReady  bool
}

// NewBrowserService creates a new BrowserService with the given configuration.
func NewBrowserService(browserCfg config.BrowserConfig, proxyURL string) *BrowserService {
	return &BrowserService{
		cfg:      browserCfg,
		proxyURL: proxyURL,
	}
}

// Enabled returns whether the browser fallback is enabled.
func (s *BrowserService) Enabled() bool {
	return s.cfg.Enabled
}

// PageTimeout returns the configured page timeout duration.
func (s *BrowserService) PageTimeout() time.Duration {
	return time.Duration(s.cfg.TimeoutSec) * time.Second
}

// HasProxy returns whether a proxy URL is configured for Level 3.
func (s *BrowserService) HasProxy() bool {
	return s.proxyURL != ""
}

// getDirectBrowser returns the non-proxy browser, launching it if needed.
func (s *BrowserService) getDirectBrowser() (*rod.Browser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.directReady {
		return s.directBr, nil
	}

	br, err := s.launchBrowser("")
	if err != nil {
		return nil, fmt.Errorf("launch direct browser: %w", err)
	}

	s.directBr = br
	s.directReady = true
	slog.Info("direct browser launched", "component", "browser")
	return br, nil
}

// getProxyBrowser returns the proxy browser, launching it if needed.
func (s *BrowserService) getProxyBrowser() (*rod.Browser, error) {
	if s.proxyURL == "" {
		return nil, fmt.Errorf("no proxy configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.proxyReady {
		return s.proxyBr, nil
	}

	br, err := s.launchBrowser(s.proxyURL)
	if err != nil {
		return nil, fmt.Errorf("launch proxy browser: %w", err)
	}

	s.proxyBr = br
	s.proxyReady = true
	slog.Info("proxy browser launched", "component", "browser", "proxy", s.proxyURL)
	return br, nil
}

// launchBrowser starts a Chromium instance with the given optional proxy URL.
func (s *BrowserService) launchBrowser(proxyURL string) (*rod.Browser, error) {
	l := launcher.New()

	if s.cfg.BinPath != "" {
		l = l.Bin(s.cfg.BinPath)
	}

	l = l.Headless(s.cfg.IsHeadless()).
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage")

	if proxyURL != "" {
		l = l.Proxy(proxyURL)
	}

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launcher: %w", err)
	}

	br := rod.New().ControlURL(u)
	if err := br.Connect(); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return br, nil
}

// Close shuts down all managed browser instances.
func (s *BrowserService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.directReady && s.directBr != nil {
		if err := s.directBr.Close(); err != nil {
			slog.Warn("failed to close direct browser", "component", "browser", "error", err)
		}
		s.directReady = false
		slog.Info("direct browser closed", "component", "browser")
	}

	if s.proxyReady && s.proxyBr != nil {
		if err := s.proxyBr.Close(); err != nil {
			slog.Warn("failed to close proxy browser", "component", "browser", "error", err)
		}
		s.proxyReady = false
		slog.Info("proxy browser closed", "component", "browser")
	}
}
