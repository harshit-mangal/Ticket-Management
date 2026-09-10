// Package auth contains the HTTP handlers for authentication endpoints.
package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"ticket-system/internal/response"
)

// Handler holds the auth service and exposes HTTP handler methods.
type Handler struct {
	service *Service
}

// NewHandler creates a Handler backed by the given service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register handles POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.Email == "" || body.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	err := h.service.Register(body.Email, body.Password)
	if errors.Is(err, ErrDuplicateEmail) {
		response.Error(w, http.StatusConflict, "email already registered")
		return
	}
	if errors.Is(err, ErrInvalidInput) {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "registration failed")
		return
	}

	response.JSON(w, http.StatusCreated, map[string]string{"message": "user registered successfully"})
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.Email == "" || body.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	token, err := h.service.Login(body.Email, body.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		// Return a single generic message — do not tell the client whether
		// the email or the password was wrong.
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "login failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"token": token})
}
