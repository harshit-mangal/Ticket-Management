// Package user defines the User data model.
package user

import "time"

// User represents an account in the system.
// The PasswordHash field stores a bcrypt hash — the plaintext password
// is never stored or logged anywhere.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // "-" ensures the hash is never included in JSON responses
	CreatedAt    time.Time `json:"created_at"`
}
