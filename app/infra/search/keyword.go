package search

import (
	"strings"

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

// buildFTS5Query converts a user query into FTS5 MATCH syntax with prefix matching.
// e.g. "woa admin" → "woa* admin*" so partial words match.
// Special FTS5 characters are escaped to prevent syntax errors.
func buildFTS5Query(query string) string {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		// Strip FTS5 special chars that could break the query
		t = strings.TrimRight(t, "*")
		t = strings.NewReplacer(
			`"`, "",
			`(`, "",
			`)`, "",
			`{`, "",
			`}`, "",
		).Replace(t)
		if t == "" {
			continue
		}
		// Add prefix matching wildcard
		parts = append(parts, t+"*")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

// Search queries the FTS5 index and returns matching URL IDs with their BM25 rank scores.
func (ks *KeywordSearch) Search(query string, limit int) ([]KeywordResult, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	ftsQuery := buildFTS5Query(query)
	if ftsQuery == "" {
		return nil, nil
	}

	var results []KeywordResult
	// FTS5 MATCH query with BM25 ranking (negative rank = higher relevance)
	err := ks.db.Raw(`
		SELECT t_urls_fts.rowid AS id, rank, title
		FROM t_urls_fts
		WHERE t_urls_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit).Scan(&results).Error

	if err != nil {
		return nil, err
	}
	return results, nil
}
