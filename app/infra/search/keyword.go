package search

import (
	"gorm.io/gorm"
)

// KeywordResult holds a single FTS5 search result.
type KeywordResult struct {
	ID    uint
	Rank  float64
	Title string
}

// KeywordSearch performs FTS5 full-text search on t_urls_fts.
type KeywordSearch struct {
	db *gorm.DB
}

func NewKeywordSearch(db *gorm.DB) *KeywordSearch {
	return &KeywordSearch{db: db}
}

// Search queries the FTS5 index and returns matching URL IDs with their BM25 rank scores.
func (ks *KeywordSearch) Search(query string, limit int) ([]KeywordResult, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	var results []KeywordResult
	// FTS5 MATCH query with BM25 ranking (negative rank = higher relevance)
	err := ks.db.Raw(`
		SELECT t_urls_fts.rowid AS id, rank, title
		FROM t_urls_fts
		WHERE t_urls_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit).Scan(&results).Error

	if err != nil {
		return nil, err
	}
	return results, nil
}
