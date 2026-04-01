package fetcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/lupguo/linkstash/app/infra/config"
)

type ChainFetcher struct {
	fetchers []Fetcher
}

func NewChainFetcher(cfg *config.Config) *ChainFetcher {
	var fetchers []Fetcher

	for _, strategy := range cfg.Fetcher.Strategies {
		switch strings.TrimSpace(strings.ToLower(strategy)) {
		case "http":
			fetchers = append(fetchers, NewHTTPFetcher(cfg.Fetcher.HTTP))
			slog.Info("fetcher registered", "strategy", "http")

		case "browser":
			if !cfg.Fetcher.Browser.Enabled {
				slog.Warn("fetcher strategy 'browser' configured but browser.enabled=false, skipping")
				continue
			}
			fetchers = append(fetchers, NewBrowserFetcher(cfg.Fetcher.Browser, ""))
			slog.Info("fetcher registered", "strategy", "browser")

		case "browser-proxy":
			if !cfg.Fetcher.Browser.Enabled {
				slog.Warn("fetcher strategy 'browser-proxy' configured but browser.enabled=false, skipping")
				continue
			}
			if cfg.Proxy.HTTPProxy == "" {
				slog.Warn("fetcher strategy 'browser-proxy' configured but no proxy set, skipping")
				continue
			}
			fetchers = append(fetchers, NewBrowserProxyFetcher(cfg.Fetcher.Browser, cfg.Proxy.HTTPProxy))
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

func (c *ChainFetcher) Name() string {
	names := make([]string, len(c.fetchers))
	for i, f := range c.fetchers {
		names[i] = f.Name()
	}
	return "chain[" + strings.Join(names, "→") + "]"
}
