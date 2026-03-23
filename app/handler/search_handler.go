package handler

import (
	"net/http"
	"strconv"

	"github.com/lupguo/linkstash/app/application"
)

type SearchHandler struct {
	usecase *application.SearchUsecase
}

func NewSearchHandler(uc *application.SearchUsecase) *SearchHandler {
	return &SearchHandler{usecase: uc}
}

// HandleSearch handles GET /api/search?q=&type=keyword|semantic|hybrid&page=1&size=20
func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := q.Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "query parameter 'q' is required")
		return
	}

	searchType := q.Get("type")
	if searchType == "" {
		searchType = "hybrid"
	}

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	size, _ := strconv.Atoi(q.Get("size"))
	if size < 1 {
		size = 20
	}

	items, total, err := h.usecase.Search(r.Context(), query, searchType, page, size)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	// Filter by min_score
	minScore, _ := strconv.ParseFloat(q.Get("min_score"), 64)
	if minScore > 0 {
		filtered := items[:0]
		for _, item := range items {
			if item.Score >= minScore {
				filtered = append(filtered, item)
			}
		}
		items = filtered
		total = len(items)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  items,
		"total": total,
		"page":  page,
		"size":  size,
		"type":  searchType,
	})
}
