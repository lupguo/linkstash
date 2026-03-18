package services

import (
	"context"
	"math"

	"github.com/lupguo/linkstash/app/infra/llm"
	"github.com/lupguo/linkstash/app/infra/search"
)

// SearchResult represents a unified search result.
type SearchResult struct {
	URLID uint    `json:"url_id"`
	Score float64 `json:"score"`
}

// SearchService provides keyword, semantic, and hybrid search.
type SearchService struct {
	keywordSearch *search.KeywordSearch
	vectorSearch  *search.VectorSearch
	llmClient     *llm.LLMClient
}

func NewSearchService(ks *search.KeywordSearch, vs *search.VectorSearch, llmClient *llm.LLMClient) *SearchService {
	return &SearchService{
		keywordSearch: ks,
		vectorSearch:  vs,
		llmClient:     llmClient,
	}
}

// KeywordSearch performs FTS5 keyword search.
func (s *SearchService) KeywordSearch(query string, limit int) ([]SearchResult, error) {
	results, err := s.keywordSearch.Search(query, limit)
	if err != nil {
		return nil, err
	}

	// Normalize BM25 ranks to 0-1 (rank is negative, more negative = better)
	if len(results) == 0 {
		return nil, nil
	}

	minRank := results[0].Rank
	maxRank := results[len(results)-1].Rank
	rankRange := maxRank - minRank

	out := make([]SearchResult, len(results))
	for i, r := range results {
		score := 1.0
		if rankRange != 0 {
			score = (maxRank - r.Rank) / rankRange // normalize to 0-1, higher is better
		}
		out[i] = SearchResult{URLID: r.ID, Score: score}
	}
	return out, nil
}

// SemanticSearch performs vector similarity search.
func (s *SearchService) SemanticSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// Generate embedding for query
	embResp, err := s.llmClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	results := s.vectorSearch.Search(embResp.Vector, limit)

	out := make([]SearchResult, len(results))
	for i, r := range results {
		out[i] = SearchResult{URLID: r.ID, Score: math.Max(0, r.Similarity)} // cosine sim already 0-1
	}
	return out, nil
}

// HybridSearch combines keyword and semantic search with equal weights.
func (s *SearchService) HybridSearch(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	kwResults, err := s.KeywordSearch(query, 50)
	if err != nil {
		return nil, err
	}

	semResults, err := s.SemanticSearch(ctx, query, 50)
	if err != nil {
		return nil, err
	}

	// Merge and combine scores
	scoreMap := make(map[uint]float64)
	for _, r := range kwResults {
		scoreMap[r.URLID] += 0.5 * r.Score
	}
	for _, r := range semResults {
		scoreMap[r.URLID] += 0.5 * r.Score
	}

	// Sort by combined score
	merged := make([]SearchResult, 0, len(scoreMap))
	for id, score := range scoreMap {
		merged = append(merged, SearchResult{URLID: id, Score: score})
	}

	// Sort descending
	for i := 0; i < len(merged); i++ {
		for j := i + 1; j < len(merged); j++ {
			if merged[j].Score > merged[i].Score {
				merged[i], merged[j] = merged[j], merged[i]
			}
		}
	}

	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}
