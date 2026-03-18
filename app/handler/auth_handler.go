package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lupguo/linkstash/app/infra/config"
	"github.com/lupguo/linkstash/app/middleware"
)

// AuthHandler handles authentication-related requests.
type AuthHandler struct {
	cfg *config.AuthConfig
}

// NewAuthHandler creates a new AuthHandler with the given auth configuration.
func NewAuthHandler(cfg *config.AuthConfig) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

// HandleToken authenticates via a shared secret and returns a JWT.
func (h *AuthHandler) HandleToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SecretKey string `json:"secret_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if req.SecretKey != h.cfg.SecretKey {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid secret key")
		return
	}

	token, err := middleware.GenerateToken(h.cfg.JWTSecret, h.cfg.JWTExpireHours)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"expires_in": h.cfg.JWTExpireHours * 3600,
	})
}
