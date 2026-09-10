// Package ticket defines the Ticket data model.
package ticket

import "time"

// Status values represent the lifecycle of a ticket.
// Only these three values are valid; any other value is rejected.
const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusClosed     = "closed"
)

// Ticket represents a support ticket created by a user.
type Ticket struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
