package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lupguo/linkstash/app/infra/config"
)

// ChatResponse holds the result of a chat completion request.
type ChatResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	LatencyMs    int64
}

// EmbeddingResponse holds the result of an embedding request.
type EmbeddingResponse struct {
	Vector      []float32
	InputTokens int
	TotalTokens int
	LatencyMs   int64
}

// LLMClient is an OpenAI-compatible HTTP client for chat and embedding endpoints.
type LLMClient struct {
	chatCfg      config.LLMEndpointConfig
	embeddingCfg config.LLMEndpointConfig
	httpClient   *http.Client
}

// NewLLMClient creates a new LLMClient with the given chat and embedding configurations.
// If httpClient is nil, a default client with 30s timeout is used.
func NewLLMClient(chatCfg, embeddingCfg config.LLMEndpointConfig, httpClient *http.Client) *LLMClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &LLMClient{
		chatCfg:      chatCfg,
		embeddingCfg: embeddingCfg,
		httpClient:   httpClient,
	}
}

// ChatModel returns the configured chat model name.
func (c *LLMClient) ChatModel() string { return c.chatCfg.Model }

// EmbeddingModel returns the configured embedding model name.
func (c *LLMClient) EmbeddingModel() string { return c.embeddingCfg.Model }

// ChatProvider returns the configured chat provider name.
func (c *LLMClient) ChatProvider() string { return c.chatCfg.Provider }

// EmbeddingProvider returns the configured embedding provider name.
func (c *LLMClient) EmbeddingProvider() string { return c.embeddingCfg.Provider }

// chatRequest is the request body for the chat completion API.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatAPIResponse is the response body from the chat completion API.
type chatAPIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// embeddingRequest is the request body for the embedding API.
type embeddingRequest struct {
	Model      string `json:"model"`
	Input      string `json:"input"`
	Dimensions int    `json:"dimensions"`
}

// embeddingAPIResponse is the response body from the embedding API.
type embeddingAPIResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// ChatCompletion sends a chat completion request and returns the parsed response.
func (c *LLMClient) ChatCompletion(ctx context.Context, systemPrompt, userContent string) (*ChatResponse, error) {
	reqBody := chatRequest{
		Model: c.chatCfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Temperature: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.chatCfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm: create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.chatCfg.APIKey)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: chat request failed: %w", err)
	}
	defer resp.Body.Close()
	latencyMs := time.Since(start).Milliseconds()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm: read chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: chat API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp chatAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("llm: unmarshal chat response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("llm: chat API returned no choices")
	}

	return &ChatResponse{
		Content:      apiResp.Choices[0].Message.Content,
		InputTokens:  apiResp.Usage.PromptTokens,
		OutputTokens: apiResp.Usage.CompletionTokens,
		TotalTokens:  apiResp.Usage.TotalTokens,
		LatencyMs:    latencyMs,
	}, nil
}

// GenerateEmbedding sends an embedding request and returns the parsed response.
func (c *LLMClient) GenerateEmbedding(ctx context.Context, input string) (*EmbeddingResponse, error) {
	reqBody := embeddingRequest{
		Model:      c.embeddingCfg.Model,
		Input:      input,
		Dimensions: c.embeddingCfg.Dimensions,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.embeddingCfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm: create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.embeddingCfg.APIKey)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: embedding request failed: %w", err)
	}
	defer resp.Body.Close()
	latencyMs := time.Since(start).Milliseconds()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm: read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: embedding API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp embeddingAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("llm: unmarshal embedding response: %w", err)
	}

	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("llm: embedding API returned no data")
	}

	return &EmbeddingResponse{
		Vector:      apiResp.Data[0].Embedding,
		InputTokens: apiResp.Usage.PromptTokens,
		TotalTokens: apiResp.Usage.TotalTokens,
		LatencyMs:   latencyMs,
	}, nil
}
