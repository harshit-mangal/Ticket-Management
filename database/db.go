// Package database handles SQLite connection setup and schema initialisation.
//
// We use a single *sql.DB connection pool that is opened once at startup
// and shared across the entire application. SQLite supports concurrent reads
// but serialises writes, which is fine for this assignment's workload.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// modernc.org/sqlite is a pure-Go SQLite driver that requires no C
	// compiler (no CGO). It registers itself with database/sql under the
	// name "sqlite".
	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite database at the given file path,
// enables foreign key enforcement, and creates the required tables if
// they do not already exist.
//
// The caller is responsible for closing the returned *sql.DB when done.
func Open(dsn string) (*sql.DB, error) {
	// Make sure the directory for the database file exists before opening.
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Verify the connection is actually usable.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// SQLite does not enforce foreign keys by default; enable it per connection.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return db, nil
}

// createSchema runs the DDL statements that create the users and tickets
// tables. Using IF NOT EXISTS makes this safe to call on every startup —
// it is a no-op when the tables already exist.
func createSchema(db *sql.DB) error {
	schema := `
	-- Users table stores account credentials.
	-- Passwords are stored as bcrypt hashes, never in plaintext.
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		email         TEXT    UNIQUE NOT NULL,
		password_hash TEXT    NOT NULL,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Tickets table stores support tickets.
	-- user_id is a foreign key to users.id, establishing ownership.
	CREATE TABLE IF NOT EXISTS tickets (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id     INTEGER NOT NULL REFERENCES users(id),
		title       TEXT    NOT NULL,
		description TEXT    NOT NULL,
		status      TEXT    NOT NULL DEFAULT 'open',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("execute schema DDL: %w", err)
	}
	return nil
}
