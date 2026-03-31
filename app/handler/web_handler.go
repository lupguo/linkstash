package handler

import (
	"net/http"
)

// WebHandler serves the SPA entry point.
type WebHandler struct {
	spaHTML []byte
	version string
}

// NewWebHandler creates a WebHandler from pre-loaded SPA HTML bytes.
func NewWebHandler(spaHTML []byte, version string) *WebHandler {
	return &WebHandler{spaHTML: spaHTML, version: version}
}

// HandleSPA serves the SPA entry point for all non-API, non-static routes.
func (h *WebHandler) HandleSPA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(h.spaHTML)
}
