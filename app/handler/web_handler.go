package handler

import (
	"context"
	"encoding/json"
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
	version       string
}

// NewWebHandler creates a new WebHandler, parsing per-page template sets.
func NewWebHandler(
	urlUsecase *application.URLUsecase,
	searchUsecase *application.SearchUsecase,
	authCfg *config.AuthConfig,
	shortCfg *config.ShortConfig,
	categories []string,
	templateDir string,
	version string,
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
		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("{}")
			}
			return template.JS(b)
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
		version:       version,
	}
}

// pageData holds common pagination and filter data for templates.
type pageData struct {
	Page           int
	Size           int
	Total          int64
	TotalPages     int
	FilterCategory string
	FilterSort     string
	Categories     []string
	Version        string
}

func (h *WebHandler) newPageData(page, size int, total int64, category, sort string) pageData {
	totalPages := int(math.Ceil(float64(total) / float64(size)))
	if totalPages < 1 {
		totalPages = 1
	}

	return pageData{
		Page:           page,
		Size:           size,
		Total:          total,
		TotalPages:     totalPages,
		FilterCategory: category,
		FilterSort:     sort,
		Version:        h.version,
	}
}

// listParams holds parsed request parameters for list/search endpoints.
type listParams struct {
	Page       int
	Size       int
	Sort       string
	Category   string
	Tags       string
	IsShortURL bool
	Query      string
	SearchType string
	MinScore   float64
}

// parseListParams extracts common list/search query parameters from the request.
func parseListParams(r *http.Request) listParams {
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
	searchType := r.URL.Query().Get("search_type")
	if searchType == "" {
		searchType = "keyword"
	}
	minScore, _ := strconv.ParseFloat(r.URL.Query().Get("min_score"), 64)
	if minScore <= 0 {
		minScore = 0.6
	}

	return listParams{
		Page:       page,
		Size:       size,
		Sort:       sort,
		Category:   r.URL.Query().Get("category"),
		Tags:       r.URL.Query().Get("tags"),
		IsShortURL: r.URL.Query().Get("is_shorturl") == "1",
		Query:      r.URL.Query().Get("q"),
		SearchType: searchType,
		MinScore:   minScore,
	}
}

// fragmentData holds data for HTMX card fragment templates.
type fragmentData struct {
	URLs          []indexURL
	HasMore       bool
	NextPageQuery string
}

// buildNextPageQuery returns the current query params with page incremented.
func buildNextPageQuery(r *http.Request, currentPage int) string {
	nextParams := r.URL.Query()
	nextParams.Set("page", strconv.Itoa(currentPage+1))
	return nextParams.Encode()
}

type indexURL struct {
	*entity.URL
	Score       float64
	HasScore    bool
	TotalWeight float64
}

// fetchURLs retrieves URLs based on list parameters (search or list mode).
func (h *WebHandler) fetchURLs(params listParams) ([]indexURL, int64, bool, error) {
	var (
		displayURLs []indexURL
		total       int64
		isSearch    bool
	)

	if params.Query != "" {
		isSearch = true
		ctx := context.Background()
		results, _, err := h.searchUsecase.Search(ctx, params.Query, params.SearchType, params.Page, params.Size)
		if err != nil {
			return nil, 0, false, err
		}
		for _, item := range results {
			if item.Score >= params.MinScore {
				tw := item.URL.AutoWeight + item.URL.ManualWeight
				displayURLs = append(displayURLs, indexURL{URL: item.URL, Score: item.Score, HasScore: true, TotalWeight: tw})
			}
		}
		total = int64(len(displayURLs))
	} else {
		urls, listTotal, err := h.urlUsecase.ListURLs(params.Page, params.Size, params.Sort, params.Category, params.Tags, params.IsShortURL)
		if err != nil {
			return nil, 0, false, err
		}
		total = listTotal
		for _, u := range urls {
			displayURLs = append(displayURLs, indexURL{URL: u, TotalWeight: u.AutoWeight + u.ManualWeight})
		}
	}

	return displayURLs, total, isSearch, nil
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

	params := parseListParams(r)
	displayURLs, total, isSearch, err := h.fetchURLs(params)
	if err != nil {
		slog.Error("fetch urls error", "component", "web_handler", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pd := h.newPageData(params.Page, params.Size, total, params.Category, params.Sort)
	pd.Categories = h.categories

	// PageData for the JSON block consumed by Alpine.js
	pageJSON := map[string]interface{}{
		"Query":          params.Query,
		"SearchType":     params.SearchType,
		"IsSearch":       isSearch,
		"FilterCategory": params.Category,
		"FilterSort":     params.Sort,
		"Size":           params.Size,
		"IsShortURL":     params.IsShortURL,
		"MinScore":       params.MinScore,
		"Page":           params.Page,
		"TotalPages":     pd.TotalPages,
	}

	data := struct {
		pageData
		URLs       []indexURL
		Query      string
		SearchType string
		IsSearch   bool
		IsShortURL bool
		MinScore   float64
		PageData   map[string]interface{}
	}{
		pageData:   pd,
		URLs:       displayURLs,
		Query:      params.Query,
		SearchType: params.SearchType,
		IsSearch:   isSearch,
		IsShortURL: params.IsShortURL,
		MinScore:   params.MinScore,
		PageData:   pageJSON,
	}

	h.renderTemplate(w, "index", data)
}

// HandleIndexCards serves GET /cards - returns HTMX scroll fragment (OOB cards + sentinel).
func (h *WebHandler) HandleIndexCards(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	params := parseListParams(r)
	displayURLs, _, _, err := h.fetchURLs(params)
	if err != nil {
		slog.Error("fetch urls error", "component", "web_handler", "error", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div id="load-more-sentinel" class="text-center py-4">
			<span class="text-red-400 text-sm">load failed</span>
			<button hx-get="/cards?%s" hx-target="#load-more-sentinel"
					hx-swap="outerHTML" class="text-terminal-green text-sm ml-2 underline">retry</button>
		</div>`, r.URL.RawQuery)
		return
	}

	hasMore := len(displayURLs) == params.Size

	t, ok := h.tmplMap["index"]
	if !ok {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "scroll_fragment", fragmentData{
		URLs:          displayURLs,
		HasMore:       hasMore,
		NextPageQuery: buildNextPageQuery(r, params.Page),
	}); err != nil {
		slog.Error("render scroll fragment error", "component", "web_handler", "error", err)
	}
}

// HandleLogin serves GET /login - the login page.
func (h *WebHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to index
	if h.isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	data := struct {
		Version string
	}{
		Version: h.version,
	}
	h.renderTemplate(w, "login", data)
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

	// PageData for JSON block consumed by detailPage()
	pageJSON := map[string]interface{}{
		"IsNew": false,
		"URL": map[string]interface{}{
			"ID":            urlEntity.ID,
			"Link":          urlEntity.Link,
			"Title":         urlEntity.Title,
			"Description":   urlEntity.Description,
			"Keywords":      urlEntity.Keywords,
			"Category":      urlEntity.Category,
			"Tags":          urlEntity.Tags,
			"ManualWeight":  urlEntity.ManualWeight,
			"AutoWeight":    urlEntity.AutoWeight,
			"VisitCount":    urlEntity.VisitCount,
			"ShortCode":     urlEntity.ShortCode,
			"Color":         urlEntity.Color,
			"Icon":          urlEntity.Icon,
			"Status":        urlEntity.Status,
		},
	}

	data := struct {
		URL        interface{}
		TagList    []string
		IsNew      bool
		TTLOptions []config.ShortTTLOption
		Categories []string
		Version    string
		PageData   map[string]interface{}
	}{
		URL:        urlEntity,
		TagList:    tagList,
		IsNew:      false,
		TTLOptions: h.shortCfg.TTLOptions,
		Categories: h.categories,
		Version:    h.version,
		PageData:   pageJSON,
	}

	h.renderTemplate(w, "detail", data)
}

// HandleNew serves GET /urls/new - the create URL page.
func (h *WebHandler) HandleNew(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// PageData for JSON block consumed by detailPage()
	pageJSON := map[string]interface{}{
		"IsNew": true,
		"URL":   map[string]interface{}{},
	}

	data := struct {
		URL        interface{}
		TagList    []string
		IsNew      bool
		TTLOptions []config.ShortTTLOption
		Categories []string
		Version    string
		PageData   map[string]interface{}
	}{
		URL:        &entity.URL{},
		TagList:    nil,
		IsNew:      true,
		TTLOptions: h.shortCfg.TTLOptions,
		Categories: h.categories,
		Version:    h.version,
		PageData:   pageJSON,
	}

	h.renderTemplate(w, "detail", data)
}
