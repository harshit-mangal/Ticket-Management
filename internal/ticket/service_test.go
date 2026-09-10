package ticket

import (
	"errors"
	"testing"
)

// TestValidateStatusTransition verifies that the status transition rules
// are enforced correctly. This is the core business-logic test.
func TestValidateStatusTransition(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		next        string
		wantErr     bool
		errContains error
	}{
		// --- Valid transitions ---
		{
			name:    "open to in_progress is allowed",
			current: StatusOpen,
			next:    StatusInProgress,
			wantErr: false,
		},
		{
			name:    "in_progress to closed is allowed",
			current: StatusInProgress,
			next:    StatusClosed,
			wantErr: false,
		},

		// --- Closed ticket cannot transition ---
		{
			name:        "closed to open is rejected",
			current:     StatusClosed,
			next:        StatusOpen,
			wantErr:     true,
			errContains: ErrTicketClosed,
		},
		{
			name:        "closed to in_progress is rejected",
			current:     StatusClosed,
			next:        StatusInProgress,
			wantErr:     true,
			errContains: ErrTicketClosed,
		},
		{
			name:        "closed to closed is rejected",
			current:     StatusClosed,
			next:        StatusClosed,
			wantErr:     true,
			errContains: ErrTicketClosed,
		},

		// --- Invalid status values ---
		{
			name:        "invalid status is rejected",
			current:     StatusOpen,
			next:        "completed",
			wantErr:     true,
			errContains: ErrInvalidStatus,
		},
		{
			name:        "pending is rejected",
			current:     StatusOpen,
			next:        "pending",
			wantErr:     true,
			errContains: ErrInvalidStatus,
		},
		{
			name:        "empty status is rejected",
			current:     StatusOpen,
			next:        "",
			wantErr:     true,
			errContains: ErrInvalidStatus,
		},

		// --- Skipping / backward transitions ---
		{
			name:        "open to closed (skip) is rejected",
			current:     StatusOpen,
			next:        StatusClosed,
			wantErr:     true,
			errContains: ErrInvalidTransition,
		},
		{
			name:        "in_progress to open (backward) is rejected",
			current:     StatusInProgress,
			next:        StatusOpen,
			wantErr:     true,
			errContains: ErrInvalidTransition,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStatusTransition(tc.current, tc.next)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if tc.errContains != nil && !errors.Is(err, tc.errContains) {
					t.Fatalf("expected error wrapping %v, got: %v", tc.errContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			}
		})
	}
}
