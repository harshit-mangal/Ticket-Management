// Package auth contains the JWT authentication middleware.
package auth

import (
	"context"
	"net/http"
	"strings"

	"ticket-system/internal/response"
)

// contextKey is an unexported type used for context keys in this package.
// Using a custom type prevents collisions with keys from other packages.
type contextKey string

const userIDKey contextKey = "userID"

// Middleware validates the JWT on every protected route.
//
// It reads the Authorization header, extracts the Bearer token, validates
// the signature and expiry, and stores the authenticated user's ID in the
// request context for downstream handlers to read.
//
// If authentication fails for any reason, the middleware immediately returns
// 401 Unauthorized and the request does not proceed further.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the Authorization header value.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Error(w, http.StatusUnauthorized, "authorization header is required")
			return
		}

		// The header value must follow the "Bearer <token>" scheme.
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(w, http.StatusUnauthorized, "authorization header must use Bearer scheme")
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			response.Error(w, http.StatusUnauthorized, "token is missing")
			return
		}

		// Parse and validate the JWT. This checks both the signature and
		// the expiration time. We never trust user-provided IDs — the
		// identity comes exclusively from the validated token.
		c, err := ParseToken(tokenString)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Store the user ID in the context so handlers can retrieve it
		// without needing to re-parse the token.
		ctx := context.WithValue(r.Context(), userIDKey, c.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext retrieves the authenticated user's ID from the request
// context. It panics if called outside of a route protected by Middleware,
// which would be a programming error rather than a runtime input error.
func UserIDFromContext(ctx context.Context) int64 {
	id, ok := ctx.Value(userIDKey).(int64)
	if !ok {
		// This should never happen if Middleware is applied correctly.
		panic("auth.UserIDFromContext: user ID not found in context — is auth middleware applied?")
	}
	return id
}
