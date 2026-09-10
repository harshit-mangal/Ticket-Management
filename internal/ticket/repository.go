// Package ticket contains the data-access layer for tickets.
package ticket

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a ticket lookup finds no matching row
// (either the ticket doesn't exist or doesn't belong to the requesting user).
var ErrNotFound = errors.New("ticket not found")

// Repository handles all database operations for tickets.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a Repository backed by the given database connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateTicket inserts a new ticket and returns the full record (including
// the auto-assigned id, timestamps, and default status).
func (r *Repository) CreateTicket(userID int64, title, description string) (*Ticket, error) {
	query := `
		INSERT INTO tickets (user_id, title, description, status)
		VALUES (?, ?, ?, 'open')
	`
	result, err := r.db.Exec(query, userID, title, description)
	if err != nil {
		return nil, fmt.Errorf("insert ticket: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	// Re-fetch the full row so the caller gets accurate timestamps
	// that were set by SQLite's DEFAULT CURRENT_TIMESTAMP.
	return r.GetTicketByIDAndUserID(id, userID)
}

// GetTicketsByUserID returns all tickets owned by the given user.
// An empty slice (not nil) is returned when the user has no tickets.
func (r *Repository) GetTicketsByUserID(userID int64) ([]Ticket, error) {
	query := `
		SELECT id, user_id, title, description, status, created_at, updated_at
		FROM tickets
		WHERE user_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("query tickets by user: %w", err)
	}
	defer rows.Close()

	tickets := []Ticket{} // initialise as empty slice so JSON encodes as [] not null
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan ticket row: %w", err)
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

// GetTicketByIDAndUserID fetches a single ticket, but only if it belongs
// to the specified user. This is the ownership-aware lookup required by
// the assignment — it never returns a ticket belonging to another user.
//
// Returns ErrNotFound if the ticket doesn't exist OR belongs to someone else.
// Callers can safely treat this as a 404 without leaking ownership information.
func (r *Repository) GetTicketByIDAndUserID(id, userID int64) (*Ticket, error) {
	query := `
		SELECT id, user_id, title, description, status, created_at, updated_at
		FROM tickets
		WHERE id = ? AND user_id = ?
	`
	var t Ticket
	err := r.db.QueryRow(query, id, userID).Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query ticket by id and user: %w", err)
	}
	return &t, nil
}

// UpdateTicketStatus updates the status of a ticket and refreshes updated_at.
// Ownership is enforced by requiring both id and userID to match.
//
// Returns ErrNotFound if the ticket doesn't exist or belongs to another user.
func (r *Repository) UpdateTicketStatus(id, userID int64, status string) (*Ticket, error) {
	query := `
		UPDATE tickets
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`
	result, err := r.db.Exec(query, status, id, userID)
	if err != nil {
		return nil, fmt.Errorf("update ticket status: %w", err)
	}

	// RowsAffected == 0 means either the ticket doesn't exist or belongs
	// to a different user. We return 404 in either case to avoid leaking
	// whether the ticket exists at all.
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, ErrNotFound
	}

	return r.GetTicketByIDAndUserID(id, userID)
}
