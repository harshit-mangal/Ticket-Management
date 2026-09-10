// Package user contains the data-access layer for user accounts.
package user

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a user lookup finds no matching row.
var ErrNotFound = errors.New("user not found")

// Repository handles all database operations for users.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a Repository backed by the given database connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser inserts a new user into the database.
//
// The passwordHash argument must already be a bcrypt hash produced by
// the auth service — this layer never hashes passwords itself.
func (r *Repository) CreateUser(email, passwordHash string) error {
	query := `INSERT INTO users (email, password_hash) VALUES (?, ?)`
	_, err := r.db.Exec(query, email, passwordHash)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetUserByEmail looks up a user by their email address.
// Returns ErrNotFound if no user with that email exists.
func (r *Repository) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, email, password_hash, created_at FROM users WHERE email = ?`

	var u User
	err := r.db.QueryRow(query, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	return &u, nil
}
