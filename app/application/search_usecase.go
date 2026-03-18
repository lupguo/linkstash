package application

import (
	"context"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"github.com/lupguo/linkstash/app/domain/services"
)

type SearchUsecase struct {
	searchService *services.SearchService
	urlRepo       repos.URLRepo
}

func NewSearchUsecase(ss *services.SearchService, urlRepo repos.URLRepo) *SearchUsecase {
	return &SearchUsecase{searchService: ss, urlRepo: urlRepo}
}

type SearchResultItem struct {
	URL   *entity.URL `json:"url"`
	Score float64     `json:"score"`
}

// Search performs search and enriches results with URL details.
func (uc *SearchUsecase) Search(ctx context.Context, query string, searchType string, page, size int) ([]SearchResultItem, int, error) {
	limit := 100 // fetch more for pagination
	var results []services.SearchResult
	var err error

	switch searchType {
	case "keyword":
		results, err = uc.searchService.KeywordSearch(query, limit)
	case "semantic":
		results, err = uc.searchService.SemanticSearch(ctx, query, limit)
	default: // hybrid
		results, err = uc.searchService.HybridSearch(ctx, query, limit)
	}
	if err != nil {
		return nil, 0, err
	}

	total := len(results)

	// Paginate
	start := (page - 1) * size
	if start >= len(results) {
		return nil, total, nil
	}
	end := start + size
	if end > len(results) {
		end = len(results)
	}
	pageResults := results[start:end]

	// Enrich with URL details
	items := make([]SearchResultItem, 0, len(pageResults))
	for _, r := range pageResults {
		url, err := uc.urlRepo.GetByID(r.URLID)
		if err != nil {
			continue // skip if URL not found (e.g., deleted)
		}
		items = append(items, SearchResultItem{URL: url, Score: r.Score})
	}

	return items, total, nil
}
