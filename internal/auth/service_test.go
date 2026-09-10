package auth

import (
	"testing"
)

// TestIsValidEmail tests the lightweight email format validator.
func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"user+tag@sub.domain.org", true},
		{"a@b.co", true},
		{"", false},
		{"notanemail", false},
		{"@nodomain.com", false},
		{"noatsign.com", false},
		{"double@@at.com", false},
		{"trailing@dot.", false},
		{"user@", false},
		{"user@nodot", false},
	}

	for _, tc := range tests {
		t.Run(tc.email, func(t *testing.T) {
			got := isValidEmail(tc.email)
			if got != tc.valid {
				t.Errorf("isValidEmail(%q) = %v, want %v", tc.email, got, tc.valid)
			}
		})
	}
}
