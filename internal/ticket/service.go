// Package ticket contains the business-logic layer for ticket operations.
package ticket

import (
	"errors"
	"fmt"
)

// ErrInvalidStatus is returned when a requested status value is not one of
// the three allowed values: open, in_progress, closed.
var ErrInvalidStatus = errors.New("invalid status: must be one of open, in_progress, closed")

// ErrInvalidTransition is returned when moving between statuses violates
// the required flow: open → in_progress → closed.
var ErrInvalidTransition = errors.New("invalid status transition")

// ErrTicketClosed is returned specifically when someone tries to update
// a ticket that has already been closed. Closed tickets are final.
var ErrTicketClosed = errors.New("ticket is closed and cannot be reopened")

// validStatuses is the set of all allowed ticket statuses.
var validStatuses = map[string]bool{
	StatusOpen:       true,
	StatusInProgress: true,
	StatusClosed:     true,
}

// allowedTransitions maps each status to the single status it can advance to.
// The required flow is strictly linear: open → in_progress → closed.
// There is no going backwards.
var allowedTransitions = map[string]string{
	StatusOpen:       StatusInProgress,
	StatusInProgress: StatusClosed,
	// StatusClosed has no outgoing transition — it is a terminal state.
}

// ValidateStatusTransition checks whether moving a ticket from currentStatus
// to nextStatus is permitted by the business rules.
//
// Rules:
//   - nextStatus must be a known valid status.
//   - A closed ticket cannot move to any other status.
//   - The only allowed moves are open→in_progress and in_progress→closed.
func ValidateStatusTransition(current, next string) error {
	// First ensure the requested new status is a recognised value.
	if !validStatuses[next] {
		return ErrInvalidStatus
	}

	// A closed ticket represents a completed ticket.
	// Once closed, it cannot move back to an earlier state.
	if current == StatusClosed {
		return ErrTicketClosed
	}

	// Check that the transition follows the allowed linear flow.
	allowed, ok := allowedTransitions[current]
	if !ok || allowed != next {
		return fmt.Errorf("%w: cannot move from %q to %q", ErrInvalidTransition, current, next)
	}

	return nil
}

// Service encapsulates ticket business logic and orchestrates
// calls to the repository.
type Service struct {
	repo *Repository
}

// NewService creates a Service that uses the given repository.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateTicket creates a new ticket owned by userID.
//
// The userID is taken from the authenticated JWT — the client cannot
// choose who owns the ticket.
func (s *Service) CreateTicket(userID int64, title, description string) (*Ticket, error) {
	return s.repo.CreateTicket(userID, title, description)
}

// ListTickets returns all tickets owned by userID.
func (s *Service) ListTickets(userID int64) ([]Ticket, error) {
	return s.repo.GetTicketsByUserID(userID)
}

// GetTicket returns a single ticket only if it belongs to userID.
// Returns ErrNotFound (mapped to 404) if the ticket doesn't exist or
// belongs to someone else — this deliberately avoids leaking whether
// another user's ticket exists at the given ID.
func (s *Service) GetTicket(id, userID int64) (*Ticket, error) {
	return s.repo.GetTicketByIDAndUserID(id, userID)
}

// UpdateStatus changes the status of a ticket owned by userID.
// The transition rules are validated before touching the database.
func (s *Service) UpdateStatus(id, userID int64, newStatus string) (*Ticket, error) {
	// Fetch the current ticket to read its existing status.
	// This also validates ownership — GetTicketByIDAndUserID requires
	// both id and userID to match.
	current, err := s.repo.GetTicketByIDAndUserID(id, userID)
	if err != nil {
		return nil, err
	}

	// Validate the transition according to business rules.
	if err := ValidateStatusTransition(current.Status, newStatus); err != nil {
		return nil, err
	}

	return s.repo.UpdateTicketStatus(id, userID, newStatus)
}
