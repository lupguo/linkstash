package handler

import (
	"html/template"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lupguo/linkstash/app/application"
	"github.com/lupguo/linkstash/app/infra/config"
)

// WebHandler serves the HTML web UI pages.
type WebHandler struct {
	tmpl          *template.Template
	urlUsecase    *application.URLUsecase
	searchUsecase *application.SearchUsecase
	authCfg       *config.AuthConfig
}

// NewWebHandler creates a new WebHandler, parsing all templates from
// web/templates/*.html and web/components/*.html.
func NewWebHandler(
	urlUsecase *application.URLUsecase,
	searchUsecase *application.SearchUsecase,
	authCfg *config.AuthConfig,
	templateDir string,
) *WebHandler {
	// Parse all templates and components together
	patterns := []string{
		filepath.Join(templateDir, "templates", "*.html"),
		filepath.Join(templateDir, "components", "*.html"),
	}

	tmpl := template.New("")
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			log.Fatalf("web_handler: glob pattern error %s: %v", pattern, err)
		}
		if len(files) > 0 {
			tmpl, err = tmpl.ParseFiles(files...)
			if err != nil {
				log.Fatalf("web_handler: parse templates error: %v", err)
			}
		}
	}

	return &WebHandler{
		tmpl:          tmpl,
		urlUsecase:    urlUsecase,
		searchUsecase: searchUsecase,
		authCfg:       authCfg,
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

// renderTemplate executes the "layout" template with the given data.
func (h *WebHandler) renderTemplate(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("web_handler: render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleIndex serves GET / - the URL list page.
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
		sort = "time"
	}
	category := r.URL.Query().Get("category")
	tags := r.URL.Query().Get("tags")

	urls, total, err := h.urlUsecase.ListURLs(page, size, sort, category, tags)
	if err != nil {
		log.Printf("web_handler: list urls error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pd := newPageData(page, size, total, category, sort)

	data := struct {
		pageData
		URLs interface{}
	}{
		pageData: pd,
		URLs:     urls,
	}

	h.renderTemplate(w, data)
}

// HandleLogin serves GET /login - the login page.
func (h *WebHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to index
	if h.isAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	h.renderTemplate(w, nil)
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
		URL     interface{}
		TagList []string
	}{
		URL:     urlEntity,
		TagList: tagList,
	}

	h.renderTemplate(w, data)
}

// HandleSearch serves GET /search - the search page.
func (h *WebHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	h.renderTemplate(w, nil)
}

// HandleShort serves GET /short - the short links management page.
func (h *WebHandler) HandleShort(w http.ResponseWriter, r *http.Request) {
	if !h.isAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	// Short links data will be loaded via API calls from the frontend.
	// For server-rendered data, you would load short links here.
	h.renderTemplate(w, struct {
		ShortLinks interface{}
	}{
		ShortLinks: nil,
	})
}
