package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lupguo/linkstash/app/application"
)

// ShortURLHandler provides HTTP handlers for short link operations.
type ShortURLHandler struct {
	usecase         *application.URLUsecase
	analysisUsecase *application.AnalysisUsecase
}

// NewShortURLHandler creates a new ShortURLHandler with the given use case.
func NewShortURLHandler(uc *application.URLUsecase) *ShortURLHandler {
	return &ShortURLHandler{usecase: uc}
}

// SetAnalysisUsecase sets the analysis usecase for async LLM processing.
func (h *ShortURLHandler) SetAnalysisUsecase(au *application.AnalysisUsecase) {
	h.analysisUsecase = au
}

// HandleCreate handles POST /api/short-links - create a new short link.
func (h *ShortURLHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LongURL string `json:"long_url"`
		Code    string `json:"code"`
		TTL     string `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if req.LongURL == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "long_url is required")
		return
	}

	ttl, err := parseTTL(req.TTL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	url, err := h.usecase.GenerateShortLink(req.LongURL, strings.TrimSpace(req.Code), ttl)
	if err != nil {
		if strings.Contains(err.Error(), "already in use") {
			writeError(w, http.StatusConflict, "CONFLICT", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	// Trigger async LLM analysis if URL is newly created (pending)
	if url.Status == "pending" && h.analysisUsecase != nil {
		h.analysisUsecase.EnqueueAnalysis(url.ID)
	}

	// Return compatible JSON format
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":               url.ID,
		"code":             url.ShortCode,
		"long_url":         url.Link,
		"short_code":       url.ShortCode,
		"link":             url.Link,
		"short_expires_at": url.ShortExpiresAt,
		"visit_count":      url.VisitCount,
		"created_at":       url.CreatedAt,
	})
}

// HandleList handles GET /api/short-links - list short links with pagination.
func (h *ShortURLHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	size, _ := strconv.Atoi(q.Get("size"))
	if size < 1 {
		size = 20
	}

	urls, total, err := h.usecase.ListShortLinks(page, size)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  urls,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// HandleDelete handles DELETE /api/short-links/:id - clear short code (keep URL record).
func (h *ShortURLHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
		return
	}

	if err := h.usecase.ClearShortCode(id); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleUpdate handles PUT /api/short-links/:id - update a short link's code or long URL.
func (h *ShortURLHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
		return
	}

	var req struct {
		Code    string `json:"code"`
		LongURL string `json:"long_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	url, err := h.usecase.UpdateShortLink(id, req.Code, req.LongURL)
	if err != nil {
		if strings.Contains(err.Error(), "already in use") {
			writeError(w, http.StatusConflict, "CONFLICT", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, url)
}

// HandleRedirect handles GET /s/:code - resolve code and redirect.
// This is a PUBLIC route (no auth needed).
func (h *ShortURLHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "code is required")
		return
	}

	url, err := h.usecase.ResolveShortCode(code)
	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			writeError(w, http.StatusGone, "EXPIRED", "short link has expired")
			return
		}
		writeError(w, http.StatusNotFound, "NOT_FOUND", "short link not found")
		return
	}

	// Async record visit
	go func() {
		_ = h.usecase.RecordVisit(url.ID)
	}()

	http.Redirect(w, r, url.Link, http.StatusFound)
}

// parseTTL parses a TTL string like "1d", "7d", "30d" into a time.Duration.
// Returns nil if the input is empty (no expiry).
func parseTTL(ttl string) (*time.Duration, error) {
	if ttl == "" {
		return nil, nil
	}

	ttl = strings.TrimSpace(ttl)
	if strings.HasSuffix(ttl, "d") {
		daysStr := strings.TrimSuffix(ttl, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TTL format: %s", ttl)
		}
		d := time.Duration(days) * 24 * time.Hour
		return &d, nil
	}

	// Try parsing as standard Go duration
	d, err := time.ParseDuration(ttl)
	if err != nil {
		return nil, fmt.Errorf("invalid TTL format: %s", ttl)
	}
	return &d, nil
}
