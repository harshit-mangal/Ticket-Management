// Package ticket contains the HTTP handlers for ticket operations.
package ticket

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"ticket-system/internal/auth"
	"ticket-system/internal/response"

	"github.com/go-chi/chi/v5"
)

// Handler holds the ticket service and exposes HTTP handler methods.
type Handler struct {
	service *Service
}

// NewHandler creates a Handler backed by the given service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /tickets.
// It creates a new ticket owned by the authenticated user.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.Title == "" || body.Description == "" {
		response.Error(w, http.StatusBadRequest, "title and description are required")
		return
	}

	// The user ID is taken from the validated JWT rather than the request body.
	// This prevents a client from creating a ticket on behalf of another user.
	userID := auth.UserIDFromContext(r.Context())

	ticket, err := h.service.CreateTicket(userID, body.Title, body.Description)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create ticket")
		return
	}

	response.JSON(w, http.StatusCreated, ticket)
}

// List handles GET /tickets.
// Returns only the tickets owned by the authenticated user.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	tickets, err := h.service.ListTickets(userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list tickets")
		return
	}

	response.JSON(w, http.StatusOK, tickets)
}

// Get handles GET /tickets/{id}.
// Returns the ticket only if it belongs to the authenticated user.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	ticket, err := h.service.GetTicket(id, userID)
	if errors.Is(err, ErrNotFound) {
		// Return 404 regardless of whether the ticket exists but belongs
		// to another user — this avoids leaking ownership information.
		response.Error(w, http.StatusNotFound, "ticket not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to get ticket")
		return
	}

	response.JSON(w, http.StatusOK, ticket)
}

// UpdateStatus handles PATCH /tickets/{id}/status.
// Updates the ticket status, enforcing ownership and transition rules.
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.Status == "" {
		response.Error(w, http.StatusBadRequest, "status is required")
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	ticket, err := h.service.UpdateStatus(id, userID, body.Status)
	if errors.Is(err, ErrNotFound) {
		response.Error(w, http.StatusNotFound, "ticket not found")
		return
	}
	if errors.Is(err, ErrInvalidStatus) {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, ErrTicketClosed) {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, ErrInvalidTransition) {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to update ticket status")
		return
	}

	response.JSON(w, http.StatusOK, ticket)
}

// parseIDParam parses a URL path parameter as an int64.
func parseIDParam(r *http.Request, param string) (int64, error) {
	raw := chi.URLParam(r, param)
	return strconv.ParseInt(raw, 10, 64)
}
