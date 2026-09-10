// Package response provides reusable helpers for writing consistent
// JSON responses to HTTP clients.
//
// All API responses use JSON. This package ensures that every
// handler uses the same structure for both success and error cases,
// making the API easier to consume and debug.
package response

import (
	"encoding/json"
	"net/http"
)

// JSON writes a JSON-encoded payload to the response writer with the
// given HTTP status code. It always sets Content-Type to application/json.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encode the payload; if encoding fails we can't do much since
	// headers are already sent, so we simply ignore the error here.
	json.NewEncoder(w).Encode(payload) //nolint:errcheck
}

// Error writes a standardised JSON error response.
//
// Example output:
//
//	{"error": "invalid credentials"}
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}
