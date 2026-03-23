package handler

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lupguo/linkstash/app/application"
	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/infra/config"
)

// WebHandler serves the HTML web UI pages.
type WebHandler struct {
	tmplMap       map[string]*template.Template
	urlUsecase    *application.URLUsecase
	searchUsecase *application.SearchUsecase
	authCfg       *config.AuthConfig
	shortCfg      *config.ShortConfig
	categories    []string
}

// NewWebHandler creates a new WebHandler, parsing per-page template sets.
// Each page template is paired with layout.html and component files so that
// the "content" block is unique per page (Go's html/template keeps only the
// last definition when all files are parsed together).
func NewWebHandler(
	urlUsecase *application.URLUsecase,
	searchUsecase *application.SearchUsecase,
	authCfg *config.AuthConfig,
	shortCfg *config.ShortConfig,
	categories []string,
	templateDir string,
) *WebHandler {
	layoutFile := filepath.Join(templateDir, "templates", "layout.html")

	// Collect component files (shared partials)
	componentPattern := filepath.Join(templateDir, "components", "*.html")
	componentFiles, err := filepath.Glob(componentPattern)
	if err != nil {
		slog.Error("glob component error", "component", "web_handler", "error", err)
		fmt.Fprintf(os.Stderr, "web_handler: glob component error: %v\n", err)
		os.Exit(1)
	}

	// Custom template functions
	funcMap := template.FuncMap{
		"domain": func(link string) string {
			u, err := url.Parse(link)
			if err != nil {
				return ""
			}
			return u.Hostname()
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"splitTags": func(tags string) []string {
			if tags == "" {
				return nil
			}
			var result []string
			for _, t := range strings.Split(tags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					result = append(result, t)
				}
			}
			return result
		},
	}

	// Page templates to parse individually with the layout
	pages := []string{"login", "index", "detail"}
	tmplMap := make(map[string]*template.Template, len(pages))

	for _, page := range pages {
		pageFile := filepath.Join(templateDir, "templates", page+".html")
		files := append([]string{layoutFile, pageFile}, componentFiles...)
		t, err := template.New("").Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			slog.Error("parse template error", "component", "web_handler", "page", page, "error", err)
			fmt.Fprintf(os.Stderr, "web_handler: parse template %s error: %v\n", page, err)
			os.Exit(1)
		}
		tmplMap[page] = t
	}

	return &WebHandler{
		tmplMap:       tmplMap,
		urlUsecase:    urlUsecase,
		searchUsecase: searchUsecase,
		authCfg:       authCfg,
		shortCfg:      shortCfg,
		categories:    categories,
	}
}

// pageData holds common pagination and filter data for templates.
type pageData struct {
	Page           int
	Size           int
	Total          int64
	TotalPages     int
	PrevPage       int
	NextPage       int
	PageNumbers    []int
	FilterCategory string
	FilterSort     string
	Categories     []string
}

func newPageData(page, size int, total int64, category, sort string) pageData {
	totalPages := int(math.Ceil(float64(total) / float64(size)))
	if totalPages < 1 {
		totalPages = 1
	}

	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}

	// Generate page numbers (up to 5 around current page)
	start := page - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > totalPages {
		end = totalPages
		start = end - 4
		if start < 1 {
			start = 1
		}
	}
	pageNumbers := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		pageNumbers = append(pageNumbers, i)
	}

	return pageData{
		Page:           page,
		Size:           size,
		Total:          total,
		TotalPages:     totalPages,
		PrevPage:       prevPage,
		NextPage:       nextPage,
		PageNumbers:    pageNumbers,
		FilterCategory: category,
		FilterSort:     sort,
	}
}

// isAuthenticated checks if the request has a valid JWT token in the cookie.
func (h *WebHandler) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("linkstash_token")
	if err != nil || cookie.Value == "" {
		return false
	}

	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.authCfg.JWTSecret), nil
	})

	return err == nil && token.Valid
}

// renderTemplate executes the "layout" template for the given page with the given data.
func (h *WebHandler) renderTemplate(w http.ResponseWriter, page string, data interface{}) {
	t, ok := h.tmplMap[page]
	if !ok {
		slog.Error("unknown page template", "component", "web_handler", "page", page)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		slog.Error("render error", "component", "web_handler", "page", page, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleIndex serves GET / - the URL list page with optional search.
func (h *WebHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 20
	}
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "weight"
	}
	category := r.URL.Query().Get("category")
	tags := r.URL.Query().Get("tags")
	isShortURL := r.URL.Query().Get("is_shorturl") == "1"

	// Search parameters
	query := r.URL.Query().Get("q")
	searchType := r.URL.Query().Get("search_type")
	if searchType == "" {
		searchType = "keyword"
	}
	minScore, _ := strconv.ParseFloat(r.URL.Query().Get("min_score"), 64)
	if minScore <= 0 {
		minScore = 0.6
	}

	type indexURL struct {
		*entity.URL
		Score       float64
		HasScore    bool
		TotalWeight float64
	}

	var (
		displayURLs []indexURL
		total       int64
		isSearch    bool
	)

	if query != "" {
		// Search mode
		isSearch = true
		ctx := context.Background()
		results, searchTotal, err := h.searchUsecase.Search(ctx, query, searchType, page, size)
		if err != nil {
			slog.Error("search error", "component", "web_handler", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Filter by min_score
		for _, item := range results {
			if item.Score >= minScore {
				tw := item.URL.AutoWeight + item.URL.ManualWeight
				displayURLs = append(displayURLs, indexURL{URL: item.URL, Score: item.Score, HasScore: true, TotalWeight: tw})
			}
		}
		total = int64(len(displayURLs))
		_ = searchTotal // total is now count after filtering
	} else {
		// Normal list mode
		urls, listTotal, err := h.urlUsecase.ListURLs(page, size, sort, category, tags, isShortURL)
		if err != nil {
			slog.Error("list urls error", "component", "web_handler", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		total = listTotal
		for _, u := range urls {
			displayURLs = append(displayURLs, indexURL{URL: u, TotalWeight: u.AutoWeight + u.ManualWeight})
		}
	}

	pd := newPageData(page, size, total, category, sort)
	pd.Categories = h.categories

	data := struct {
		pageData
		URLs       []indexURL
		Query      string
		SearchType string
		IsSearch   bool
		IsShortURL bool
		MinScore   float64
	}{
		pageData:   pd,
		URLs:       displayURLs,
		Query:      query,
		SearchType: searchType,
		IsSearch:   isSearch,
		IsShortURL: isShortURL,
		MinScore:   minScore,
	}

	h.renderTemplate(w, "index", data)
}

// HandleIndexCards serves GET /cards - returns only card HTML fragments for infinite scroll.
func (h *WebHandler) HandleIndexCards(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters (same as HandleIndex)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 20
	}
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "weight"
	}
	category := r.URL.Query().Get("category")
	tags := r.URL.Query().Get("tags")
	isShortURL := r.URL.Query().Get("is_shorturl") == "1"

	// Search parameters
	query := r.URL.Query().Get("q")
	searchType := r.URL.Query().Get("search_type")
	if searchType == "" {
		searchType = "keyword"
	}
	minScore, _ := strconv.ParseFloat(r.URL.Query().Get("min_score"), 64)
	if minScore <= 0 {
		minScore = 0.6
	}

	type indexURL struct {
		*entity.URL
		Score       float64
		HasScore    bool
		TotalWeight float64
	}

	var displayURLs []indexURL

	if query != "" {
		ctx := context.Background()
		results, _, err := h.searchUsecase.Search(ctx, query, searchType, page, size)
		if err != nil {
			slog.Error("search error", "component", "web_handler", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		for _, item := range results {
			if item.Score >= minScore {
				tw := item.URL.AutoWeight + item.URL.ManualWeight
				displayURLs = append(displayURLs, indexURL{URL: item.URL, Score: item.Score, HasScore: true, TotalWeight: tw})
			}
		}
	} else {
		urls, _, err := h.urlUsecase.ListURLs(page, size, sort, category, tags, isShortURL)
		if err != nil {
			slog.Error("list urls error", "component", "web_handler", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		for _, u := range urls {
			displayURLs = append(displayURLs, indexURL{URL: u, TotalWeight: u.AutoWeight + u.ManualWeight})
		}
	}

	// Return empty body if no results
	if len(displayURLs) == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		return
	}

	// Render each card fragment using the "url_card" template
	t, ok := h.tmplMap["index"]
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, item := range displayURLs {
		if err := t.ExecuteTemplate(w, "url_card", item); err != nil {
			slog.Error("render card error", "component", "web_handler", "error", err)
			return
		}
	}
}

// HandleLogin serves GET /login - the login page.
func (h *WebHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to index
	if h.isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	h.renderTemplate(w, "login", nil)
}

// HandleDetail serves GET /urls/{id} - the URL detail page.
func (h *WebHandler) HandleDetail(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		return
	}

	urlEntity, err := h.urlUsecase.GetURL(uint(id))
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Split tags for display
	var tagList []string
	if urlEntity.Tags != "" {
		for _, t := range strings.Split(urlEntity.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagList = append(tagList, t)
			}
		}
	}

	data := struct {
		URL        interface{}
		TagList    []string
		IsNew      bool
		TTLOptions []config.ShortTTLOption
		Categories []string
	}{
		URL:        urlEntity,
		TagList:    tagList,
		IsNew:      false,
		TTLOptions: h.shortCfg.TTLOptions,
		Categories: h.categories,
	}

	h.renderTemplate(w, "detail", data)
}

// HandleNew serves GET /urls/new - the create URL page.
func (h *WebHandler) HandleNew(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	data := struct {
		URL        interface{}
		TagList    []string
		IsNew      bool
		TTLOptions []config.ShortTTLOption
		Categories []string
	}{
		URL:        &entity.URL{},
		TagList:    nil,
		IsNew:      true,
		TTLOptions: h.shortCfg.TTLOptions,
		Categories: h.categories,
	}

	h.renderTemplate(w, "detail", data)
}
