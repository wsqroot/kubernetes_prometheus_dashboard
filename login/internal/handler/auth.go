package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"login/internal/service"
)

type loginService interface {
	Login(ctx context.Context, username, password string) (int64, string, error)
}

type AuthHandler struct {
	service loginService
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewAuthHandler(service loginService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, loginResponse{
		Success: true,
		Message: "ok",
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, loginResponse{
			Success: false,
			Message: "method not allowed",
		})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, loginResponse{
			Success: false,
			Message: "invalid request body",
		})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, loginResponse{
			Success: false,
			Message: "username and password are required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id, username, err := h.service.Login(ctx, req.Username, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		message := "login failed"
		if errors.Is(err, service.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
			message = "invalid username or password"
		}
		writeJSON(w, status, loginResponse{
			Success: false,
			Message: message,
		})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Success: true,
		Message: "login success",
		Data: map[string]interface{}{
			"id":       id,
			"username": username,
		},
	})
}

func WithCORS(next http.Handler, allowedOrigins []string) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAnyOrigin := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAnyOrigin = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowAnyOrigin && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		} else if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload loginResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
