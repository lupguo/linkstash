package search

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// KeywordResult holds a single keyword search result.
type KeywordResult struct {
	ID    uint
	Rank  float64
	Title string
}

// KeywordSearcher is the interface for keyword-based search.
// SQLite uses FTS5, MySQL uses LIKE-based queries.
type KeywordSearcher interface {
	Search(query string, limit int) ([]KeywordResult, error)
}

// --- SQLite FTS5 implementation ---

// FTS5KeywordSearch performs FTS5 full-text search on t_urls_fts (SQLite only).
type FTS5KeywordSearch struct {
	db *gorm.DB
}

func NewFTS5KeywordSearch(db *gorm.DB) *FTS5KeywordSearch {
	return &FTS5KeywordSearch{db: db}
}

// NewKeywordSearch creates the default keyword search (FTS5 for backward compatibility).
// Deprecated: use NewFTS5KeywordSearch or NewLikeKeywordSearch explicitly.
func NewKeywordSearch(db *gorm.DB) *FTS5KeywordSearch {
	return NewFTS5KeywordSearch(db)
}

// buildFTS5Query converts a user query into FTS5 MATCH syntax with prefix matching.
func buildFTS5Query(query string) string {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
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
		parts = append(parts, t+"*")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

// Search queries the FTS5 index and returns matching URL IDs with their BM25 rank scores.
func (ks *FTS5KeywordSearch) Search(query string, limit int) ([]KeywordResult, error) {
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

// --- MySQL LIKE-based implementation ---

// LikeKeywordSearch performs LIKE-based keyword search on t_urls (MySQL mode).
// This is simpler than FULLTEXT but sufficient for personal-scale data (<10K URLs).
type LikeKeywordSearch struct {
	db *gorm.DB
}

func NewLikeKeywordSearch(db *gorm.DB) *LikeKeywordSearch {
	return &LikeKeywordSearch{db: db}
}

// Search queries t_urls using LIKE patterns across searchable columns.
// Results are ranked by number of matching columns (simple relevance scoring).
func (ks *LikeKeywordSearch) Search(query string, limit int) ([]KeywordResult, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	// Build LIKE conditions: each token must match at least one searchable column
	searchCols := []string{"link", "title", "keywords", "description", "category", "tags", "short_code"}
	var whereClauses []string
	var args []interface{}

	for _, token := range tokens {
		var colClauses []string
		pattern := "%" + token + "%"
		for _, col := range searchCols {
			colClauses = append(colClauses, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, pattern)
		}
		whereClauses = append(whereClauses, "("+strings.Join(colClauses, " OR ")+")")
	}

	// Build a relevance score: count of matching columns (more matches = higher relevance)
	var scoreParts []string
	var scoreArgs []interface{}
	for _, token := range tokens {
		pattern := "%" + token + "%"
		for _, col := range searchCols {
			scoreParts = append(scoreParts, fmt.Sprintf("(CASE WHEN %s LIKE ? THEN 1 ELSE 0 END)", col))
			scoreArgs = append(scoreArgs, pattern)
		}
	}
	scoreExpr := strings.Join(scoreParts, " + ")

	where := strings.Join(whereClauses, " AND ")

	// Combine all args: score args first, then where args
	allArgs := append(scoreArgs, args...)
	allArgs = append(allArgs, limit)

	sql := fmt.Sprintf(`
		SELECT id, (%s) AS `+"`rank`"+`, title
		FROM t_urls
		WHERE deleted_at IS NULL AND %s
		ORDER BY `+"`rank`"+` DESC
		LIMIT ?
	`, scoreExpr, where)

	var results []KeywordResult
	if err := ks.db.Raw(sql, allArgs...).Scan(&results).Error; err != nil {
		return nil, err
	}

	// Negate ranks to match FTS5 convention (more negative = better) for consistent normalization
	for i := range results {
		results[i].Rank = -results[i].Rank
	}

	return results, nil
}
