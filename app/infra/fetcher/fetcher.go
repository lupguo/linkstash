package fetcher

import "context"

// Fetcher fetches page content from a URL.
type Fetcher interface {
	Name() string
	Fetch(ctx context.Context, url string) (string, error)
}
