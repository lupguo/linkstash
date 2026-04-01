package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"github.com/lupguo/linkstash/app/infra/llm"
)

// ContentFetcher fetches page content from a URL using a configured strategy chain.
type ContentFetcher interface {
	Fetch(ctx context.Context, url string) (string, error)
	Name() string
}

type WorkerService struct {
	queue         chan uint
	urlRepo       repos.URLRepo
	llmLogRepo    repos.LLMLogRepo
	embeddingRepo repos.EmbeddingRepo
	llmClient     *llm.LLMClient
	prompts       map[string]string
	fetcher       ContentFetcher
	done          chan struct{}
}

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
		prompts:       prompts,
		fetcher:       fetcher,
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

// fetchPageContent fetches page content for LLM analysis using the configured fetcher chain.
func (w *WorkerService) fetchPageContent(link string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return w.fetcher.Fetch(ctx, link)
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
