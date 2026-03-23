package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lupguo/linkstash/app/application"
)

// defaultIcons is the emoji pool for auto-assigning icons to new URLs.
var defaultIcons = []string{
	"🔗", "📚", "💻", "🌐", "📝", "🔧", "🎯", "📦", "🚀", "⭐",
	"📌", "🔍", "💡", "📊", "🎨", "🔒", "📡", "🧩", "🗂️", "✨",
}

// validThemes is the whitelist of allowed theme color keys.
var validThemes = map[string]bool{
	"": true, "green": true, "red": true, "cyan": true,
	"yellow": true, "purple": true, "orange": true, "blue": true,
}

// URLHandler provides HTTP handlers for URL CRUD operations.
type URLHandler struct {
	usecase         *application.URLUsecase
	analysisUsecase *application.AnalysisUsecase
}

// NewURLHandler creates a new URLHandler with the given use case.
func NewURLHandler(uc *application.URLUsecase) *URLHandler {
	return &URLHandler{usecase: uc}
}

// SetAnalysisUsecase sets the analysis usecase for async LLM processing.
func (h *URLHandler) SetAnalysisUsecase(au *application.AnalysisUsecase) {
	h.analysisUsecase = au
}

// HandleCreate handles POST /api/urls - create a new URL.
func (h *URLHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Link string `json:"link"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	url, err := h.usecase.AddURL(req.Link)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, "CONFLICT", "该链接已存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	// Auto-assign random emoji icon if not set
	if url.Icon == "" {
		url.Icon = defaultIcons[rand.Intn(len(defaultIcons))]
		_ = h.usecase.UpdateURL(url)
	}

	// Async fetch favicon — if found, clear emoji icon to prefer favicon
	go func(urlID uint, link string) {
		favicon := fetchFavicon(link)
		if favicon != "" {
			u, _ := h.usecase.GetURL(urlID)
			if u != nil {
				u.Favicon = favicon
				u.Icon = "" // clear emoji icon, prefer favicon
				h.usecase.UpdateURL(u)
			}
		}
	}(url.ID, url.Link)

	// Trigger async LLM analysis
	if h.analysisUsecase != nil {
		h.analysisUsecase.EnqueueAnalysis(url.ID)
	}

	writeJSON(w, http.StatusCreated, url)
}

// HandleList handles GET /api/urls - list URLs with pagination and filtering.
func (h *URLHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	size, _ := strconv.Atoi(q.Get("size"))
	if size < 1 {
		size = 20
	}

	sort := q.Get("sort")
	if sort == "" {
		sort = "time"
	}

	category := q.Get("category")
	tags := q.Get("tags")

	urls, total, err := h.usecase.ListURLs(page, size, sort, category, tags, false)
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

// HandleGet handles GET /api/urls/:id - get URL by ID.
func (h *URLHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
		return
	}

	url, err := h.usecase.GetURL(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, url)
}

// HandleUpdate handles PUT /api/urls/:id - partial update URL.
func (h *URLHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
		return
	}

	// Fetch existing record first
	existing, err := h.usecase.GetURL(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	// Decode partial update into a map to know which fields were sent
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	// Apply updates to existing record
	if v, ok := updates["title"]; ok {
		existing.Title = v.(string)
	}
	if v, ok := updates["keywords"]; ok {
		existing.Keywords = v.(string)
	}
	if v, ok := updates["description"]; ok {
		existing.Description = v.(string)
	}
	if v, ok := updates["category"]; ok {
		existing.Category = v.(string)
	}
	if v, ok := updates["tags"]; ok {
		existing.Tags = v.(string)
	}
	if v, ok := updates["manual_weight"]; ok {
		existing.ManualWeight = v.(float64)
	}
	if v, ok := updates["link"]; ok {
		existing.Link = v.(string)
	}
	if v, ok := updates["visit_count"]; ok {
		existing.VisitCount = int(v.(float64))
		existing.AutoWeight = float64(existing.VisitCount)
	}
	if v, ok := updates["color"]; ok {
		color := v.(string)
		if !validThemes[color] {
			color = ""
		}
		existing.Color = color
	}
	if v, ok := updates["icon"]; ok {
		existing.Icon = v.(string)
	}
	if v, ok := updates["short_code"]; ok {
		existing.ShortCode = v.(string)
	}
	if v, ok := updates["status"]; ok {
		existing.Status = v.(string)
	}
	if v, ok := updates["favicon"]; ok {
		existing.Favicon = v.(string)
	}
	if v, ok := updates["ttl"]; ok {
		ttlStr, _ := v.(string)
		if ttlStr == "" {
			existing.ShortExpiresAt = nil
		} else {
			ttl, err := parseTTL(ttlStr)
			if err == nil && ttl != nil {
				t := time.Now().Add(*ttl)
				existing.ShortExpiresAt = &t
			}
		}
	}

	if err := h.usecase.UpdateURL(existing); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// HandleDelete handles DELETE /api/urls/:id - soft delete URL.
func (h *URLHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
		return
	}

	if err := h.usecase.DeleteURL(id); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleVisit handles POST /api/urls/:id/visit - record a visit.
func (h *URLHandler) HandleVisit(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
		return
	}

	if err := h.usecase.RecordVisit(id); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func parseUintParam(r *http.Request, name string) (uint, error) {
	v, err := strconv.ParseUint(chi.URLParam(r, name), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// fetchFavicon fetches the favicon for a URL from DuckDuckGo icons service
// and returns a base64 data URI. Returns "" on failure.
func fetchFavicon(link string) string {
	parsed, err := url.Parse(link)
	if err != nil || parsed.Host == "" {
		return ""
	}
	domain := parsed.Host

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://icons.duckduckgo.com/ip3/%s.ico", domain))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // max 512KB
	if err != nil || len(body) == 0 {
		return ""
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/x-icon"
	}

	encoded := base64.StdEncoding.EncodeToString(body)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
}
