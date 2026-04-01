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
	content, err := f.doFetch(ctx, rawURL)
	if err == nil {
		return content, nil
	}

	slog.Info("http fetch failed, trying root domain", "url", rawURL, "error", err)

	parsed, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", err
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
