package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"github.com/lupguo/linkstash/app/infra/llm"
)

// BrowserFetcher defines the interface for headless browser page fetching.
// Defined in the domain layer to maintain Clean Architecture (no infra dependency).
type BrowserFetcher interface {
	Enabled() bool
	HasProxy() bool
	FetchPage(ctx context.Context, url string, useProxy bool) (string, error)
	Close()
}

type WorkerService struct {
	queue         chan uint
	urlRepo       repos.URLRepo
	llmLogRepo    repos.LLMLogRepo
	embeddingRepo repos.EmbeddingRepo
	llmClient     *llm.LLMClient
	httpClient    *http.Client
	prompts       map[string]string
	browserSvc    BrowserFetcher
	done          chan struct{}
}

func NewWorkerService(
	urlRepo repos.URLRepo,
	llmLogRepo repos.LLMLogRepo,
	embeddingRepo repos.EmbeddingRepo,
	llmClient *llm.LLMClient,
	prompts map[string]string,
	httpClient *http.Client,
	browserSvc BrowserFetcher,
) *WorkerService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &WorkerService{
		queue:         make(chan uint, 100),
		urlRepo:       urlRepo,
		llmLogRepo:    llmLogRepo,
		embeddingRepo: embeddingRepo,
		llmClient:     llmClient,
		httpClient:    httpClient,
		prompts:       prompts,
		browserSvc:    browserSvc,
		done:          make(chan struct{}),
	}
}

func (w *WorkerService) Enqueue(urlID uint) {
	select {
	case w.queue <- urlID:
		slog.Debug("enqueued url for analysis", "component", "worker", "url_id", urlID)
	default:
		slog.Warn("queue full, dropping url", "component", "worker", "url_id", urlID)
	}
}

func (w *WorkerService) Start(ctx context.Context) {
	go func() {
		defer close(w.done)
		for {
			select {
			case <-ctx.Done():
				slog.Info("worker stopped", "component", "worker")
				return
			case urlID := <-w.queue:
				w.processWithRetry(ctx, urlID)
			}
		}
	}()
}

func (w *WorkerService) RecoverPending() {
	for _, status := range []string{"pending", "analyzing"} {
		urls, err := w.urlRepo.FindByStatus(status)
		if err != nil {
			slog.Error("recover error", "component", "worker", "status", status, "error", err)
			continue
		}
		for _, u := range urls {
			w.Enqueue(u.ID)
		}
		if len(urls) > 0 {
			slog.Info("recovered urls", "component", "worker", "count", len(urls), "status", status)
		}
	}
}

func (w *WorkerService) processWithRetry(ctx context.Context, urlID uint) {
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			slog.Info("retrying url processing", "component", "worker", "url_id", urlID, "attempt", attempt, "backoff", backoff)
			time.Sleep(backoff)
		}

		if err := w.doProcessURL(ctx, urlID); err != nil {
			slog.Error("url processing failed", "component", "worker", "url_id", urlID, "attempt", attempt, "error", err)
			if attempt == maxRetries {
				w.setURLFailed(urlID, err.Error())
			}
			continue
		}
		return
	}
}

func (w *WorkerService) doProcessURL(ctx context.Context, urlID uint) error {
	// 1. Get URL, set status "analyzing"
	url, err := w.urlRepo.GetByID(urlID)
	if err != nil {
		return fmt.Errorf("get url: %w", err)
	}
	url.Status = "analyzing"
	if err := w.urlRepo.Update(url); err != nil {
		return fmt.Errorf("set analyzing: %w", err)
	}

	// 2. Fetch page content (with fallback to root domain)
	pageContent, err := w.fetchPageContent(url.Link)
	if err != nil {
		return fmt.Errorf("fetch url(%s): %w", url.Link, err)
	}

	// 3. LLM chat completion for analysis
	prompt := w.prompts["url_analysis"]
	chatResp, chatErr := w.llmClient.ChatCompletion(ctx, prompt, pageContent)

	// Log chat request
	chatLog := &entity.LLMLog{
		URLID:        urlID,
		RequestType:  "chat",
		Provider:     w.llmClient.ChatProvider(),
		ModelName:    w.llmClient.ChatModel(),
		PromptKey:    "url_analysis",
		InputContent: pageContent,
	}
	if chatErr != nil {
		chatLog.Success = false
		chatLog.ErrorMessage = chatErr.Error()
		_ = w.llmLogRepo.Create(chatLog)
		return fmt.Errorf("chat completion: %w", chatErr)
	}
	chatLog.Success = true
	chatLog.OutputContent = chatResp.Content
	chatLog.InputTokens = chatResp.InputTokens
	chatLog.OutputTokens = chatResp.OutputTokens
	chatLog.TotalTokens = chatResp.TotalTokens
	chatLog.LatencyMs = chatResp.LatencyMs
	if chatResp.LatencyMs > 0 && chatResp.OutputTokens > 0 {
		chatLog.TokensPerSec = float64(chatResp.OutputTokens) / (float64(chatResp.LatencyMs) / 1000.0)
	}
	_ = w.llmLogRepo.Create(chatLog)

	// 4. Parse JSON response
	var parsed struct {
		Title       string `json:"title"`
		Keywords    string `json:"keywords"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Tags        string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(chatResp.Content), &parsed); err != nil {
		return fmt.Errorf("parse llm response: %w", err)
	}

	// 5. Update URL with parsed fields
	url.Title = parsed.Title
	url.Keywords = parsed.Keywords
	url.Description = parsed.Description
	url.Category = parsed.Category
	url.Tags = parsed.Tags
	url.Status = "ready"
	if err := w.urlRepo.Update(url); err != nil {
		return fmt.Errorf("save ready: %w", err)
	}

	// 6. Generate embedding
	embText := strings.Join([]string{parsed.Title, parsed.Keywords, parsed.Description}, " ")
	embResp, embErr := w.llmClient.GenerateEmbedding(ctx, embText)

	// Log embedding request
	embLog := &entity.LLMLog{
		URLID:        urlID,
		RequestType:  "embedding",
		Provider:     w.llmClient.EmbeddingProvider(),
		ModelName:    w.llmClient.EmbeddingModel(),
		PromptKey:    "embedding",
		InputContent: embText,
	}
	if embErr != nil {
		embLog.Success = false
		embLog.ErrorMessage = embErr.Error()
		_ = w.llmLogRepo.Create(embLog)
		return fmt.Errorf("generate embedding: %w", embErr)
	}
	embLog.Success = true
	embLog.InputTokens = embResp.InputTokens
	embLog.TotalTokens = embResp.TotalTokens
	embLog.LatencyMs = embResp.LatencyMs
	_ = w.llmLogRepo.Create(embLog)

	// 7. Save embedding
	embedding := &entity.Embedding{
		URLID:  urlID,
		Vector: llm.Float32sToBytes(embResp.Vector),
	}
	if err := w.embeddingRepo.Save(embedding); err != nil {
		return fmt.Errorf("save embedding: %w", err)
	}

	slog.Info(fmt.Sprintf("successfully processed url(%s)", url.Link), "component", "worker", "url_id", urlID, "title", parsed.Title)
	return nil
}

// fetchPageContent fetches page content for LLM analysis.
//   - Browser enabled: Rod headless browser (no proxy → with proxy)
//   - Browser disabled: HTTP GET with browser-like headers → root domain fallback
func (w *WorkerService) fetchPageContent(link string) (string, error) {
	// Browser mode: use Rod directly, no HTTP fallback
	if w.browserSvc != nil && w.browserSvc.Enabled() {
		return w.fetchWithBrowser(link)
	}

	// HTTP mode: optimized HTTP GET + root domain fallback
	return w.fetchWithHTTP(link)
}

// fetchWithBrowser fetches using Rod headless browser (no proxy first, then with proxy).
func (w *WorkerService) fetchWithBrowser(link string) (string, error) {
	ctx := context.Background()

	// Try without proxy first
	content, err := w.browserSvc.FetchPage(ctx, link, false)
	if err == nil && !isBlockedContent(content) {
		slog.Debug("browser fetch success", "component", "worker", "url", link)
		return content, nil
	}
	if err != nil {
		slog.Info("browser fetch failed", "component", "worker", "url", link, "error", err)
	} else {
		slog.Info("browser fetch returned blocked content", "component", "worker", "url", link)
	}

	// Try with proxy if configured
	if w.browserSvc.HasProxy() {
		proxyContent, proxyErr := w.browserSvc.FetchPage(ctx, link, true)
		if proxyErr == nil && !isBlockedContent(proxyContent) {
			slog.Debug("browser+proxy fetch success", "component", "worker", "url", link)
			return proxyContent, nil
		}
		if proxyErr != nil {
			slog.Info("browser+proxy fetch failed", "component", "worker", "url", link, "error", proxyErr)
		} else {
			slog.Info("browser+proxy fetch returned blocked content", "component", "worker", "url", link)
		}
	}

	// All browser attempts failed
	if err != nil {
		return "", fmt.Errorf("browser fetch: %w", err)
	}
	return "", fmt.Errorf("browser fetch: blocked content")
}

// fetchWithHTTP fetches using optimized HTTP GET with root domain fallback.
func (w *WorkerService) fetchWithHTTP(link string) (string, error) {
	content, err := w.doFetch(link)
	if err == nil && !isBlockedContent(content) {
		return content, nil
	}

	// Fallback: try root domain
	parsed, parseErr := neturl.Parse(link)
	if parseErr != nil || parsed.Host == "" {
		if err != nil {
			return "", err
		}
		return content, nil
	}

	rootURL := parsed.Scheme + "://" + parsed.Host + "/"
	if rootURL == link || rootURL+"/" == link {
		if err != nil {
			return "", err
		}
		return content, nil
	}

	slog.Info("fallback to root domain", "component", "worker", "root_url", rootURL, "url", link)
	rootContent, rootErr := w.doFetch(rootURL)
	if rootErr == nil && !isBlockedContent(rootContent) {
		return rootContent, nil
	}

	if content != "" {
		return content, nil
	}
	if err != nil {
		return "", err
	}
	return "", rootErr
}

// doFetch fetches a URL with browser-like headers and returns its body content (up to 50KB).
func (w *WorkerService) doFetch(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// Set browser-like headers to reduce blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 50*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(bodyBytes), nil
}

// isBlockedContent checks if the content looks like a Cloudflare challenge
// or an auth-wall page rather than actual content.
func isBlockedContent(content string) bool {
	if len(content) < 100 {
		return true
	}
	lower := strings.ToLower(content)
	markers := []string{
		"cf-mitigated",
		"challenge-platform",
		"cf-chl-bypass",
		"just a moment",
		"checking your browser",
		"attention required",
		"ray id",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func (w *WorkerService) setURLFailed(urlID uint, errMsg string) {
	u, err := w.urlRepo.GetByID(urlID)
	if err != nil {
		slog.Error("get url for failure", "component", "worker", "url_id", urlID, "error", err)
		return
	}
	slog.Warn(fmt.Sprintf("marking url as failed(%s)", u.Link), "component", "worker", "url_id", urlID, "error", errMsg)
	u.Status = "failed"
	if err := w.urlRepo.Update(u); err != nil {
		slog.Error("set url failed status", "component", "worker", "url_id", urlID, "error", err)
	}
}
