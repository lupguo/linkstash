package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

// WebHandler serves the SPA entry point.
type WebHandler struct {
	spaHTML []byte
	version string
}

// NewWebHandler loads the SPA HTML template once.
func NewWebHandler(templateDir string, version string) *WebHandler {
	spaPath := filepath.Join(templateDir, "templates", "spa.html")
	data, err := os.ReadFile(spaPath)
	if err != nil {
		slog.Error("failed to read spa.html", "error", err)
		fmt.Fprintf(os.Stderr, "web_handler: failed to read %s: %v\n", spaPath, err)
		os.Exit(1)
	}
	return &WebHandler{spaHTML: data, version: version}
}

// HandleSPA serves the SPA entry point for all non-API, non-static routes.
func (h *WebHandler) HandleSPA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(h.spaHTML)
}
