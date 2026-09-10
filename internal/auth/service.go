// Package auth contains authentication business logic: user registration,
// login, JWT generation, and JWT validation.
package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"ticket-system/internal/user"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrDuplicateEmail is returned when a registration attempt uses an
// email address that is already registered.
var ErrDuplicateEmail = errors.New("email already registered")

// ErrInvalidCredentials is returned on login failure.
// We intentionally use a single error for both "user not found" and
// "wrong password" so the caller cannot distinguish them — this prevents
// user-enumeration attacks.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidInput is returned when required fields are missing or malformed.
var ErrInvalidInput = errors.New("invalid input")

// tokenTTL controls how long a JWT is valid after issue.
const tokenTTL = 24 * time.Hour

// claims is the custom JWT payload. It embeds the standard registered
// claims (exp, iat, etc.) and adds the authenticated user's ID.
type claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// Service provides authentication operations.
type Service struct {
	userRepo *user.Repository
}

// NewService creates an auth Service backed by the given user repository.
func NewService(userRepo *user.Repository) *Service {
	return &Service{userRepo: userRepo}
}

// Register validates input, hashes the password, and stores the new user.
// It returns ErrDuplicateEmail if the email is already taken.
func (s *Service) Register(email, password string) error {
	if email == "" || password == "" {
		return ErrInvalidInput
	}
	if !isValidEmail(email) {
		return fmt.Errorf("%w: invalid email format", ErrInvalidInput)
	}

	// Check for duplicate email before hashing to provide a clear error.
	if _, err := s.userRepo.GetUserByEmail(email); err == nil {
		// GetUserByEmail succeeded — the user already exists.
		return ErrDuplicateEmail
	} else if !errors.Is(err, user.ErrNotFound) {
		// An unexpected database error occurred.
		return fmt.Errorf("check existing user: %w", err)
	}

	// Hash the password with bcrypt. The cost factor adds deliberate slowness
	// to make brute-force attacks expensive. DefaultCost (10) is a safe choice.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.userRepo.CreateUser(email, string(hash))
}

// Login verifies credentials and returns a signed JWT on success.
// Returns ErrInvalidCredentials for any authentication failure — we do not
// distinguish between "user not found" and "wrong password".
func (s *Service) Login(email, password string) (string, error) {
	if email == "" || password == "" {
		return "", ErrInvalidCredentials
	}

	u, err := s.userRepo.GetUserByEmail(email)
	if errors.Is(err, user.ErrNotFound) {
		// Return a generic error so callers cannot enumerate users by email.
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", fmt.Errorf("lookup user: %w", err)
	}

	// CompareHashAndPassword returns an error if the password does not match.
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return generateToken(u.ID)
}

// generateToken creates a signed JWT containing the user's ID and an expiry.
// The signing key comes from the JWT_SECRET environment variable — it is
// never hardcoded.
func generateToken(userID int64) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable is not set")
	}

	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

// ParseToken validates a JWT string and returns the claims it carries.
// Returns an error if the token is malformed, has an invalid signature,
// or has expired.
func ParseToken(tokenString string) (*claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET environment variable is not set")
	}

	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (any, error) {
		// Explicitly verify the signing method to prevent "alg:none" attacks.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return c, nil
}

// isValidEmail performs a lightweight email format check.
// We use a simple approach: verify that exactly one "@" exists and
// both the local and domain parts are non-empty.
func isValidEmail(email string) bool {
	at := -1
	for i, ch := range email {
		if ch == '@' {
			if at != -1 {
				return false // more than one @
			}
			at = i
		}
	}
	if at <= 0 || at >= len(email)-1 {
		return false
	}
	domain := email[at+1:]
	// Domain must contain at least one dot and something after it.
	dot := -1
	for i, ch := range domain {
		if ch == '.' {
			dot = i
		}
	}
	return dot > 0 && dot < len(domain)-1
}
