// Command server is the entry point for the Ticket Management API.
//
// It loads configuration from environment variables, initialises the
// database, wires together the router, and starts the HTTP server.
package main

import (
	"log"
	"net/http"
	"os"

	"ticket-system/database"
	"ticket-system/internal/auth"
	"ticket-system/internal/response"
	"ticket-system/internal/ticket"
	"ticket-system/internal/user"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file if it exists.
	// In production (Docker/cloud), variables are injected directly into
	// the environment and .env will not be present — that is expected.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Read configuration with safe defaults.
	port := envOrDefault("PORT", "8080")
	dbPath := envOrDefault("DATABASE_PATH", "./database/tickets.db")

	// Fail fast if the JWT secret is missing — the server cannot operate
	// without it because all token operations would be insecure.
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable must be set")
	}

	// Open (or create) the SQLite database and run the schema migrations.
	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// --- Dependency wiring ---
	// Each layer receives only the dependencies it needs.
	userRepo := user.NewRepository(db)

	authService := auth.NewService(userRepo)
	authHandler := auth.NewHandler(authService)

	ticketRepo := ticket.NewRepository(db)
	ticketService := ticket.NewService(ticketRepo)
	ticketHandler := ticket.NewHandler(ticketService)

	// --- Router setup ---
	r := chi.NewRouter()

	// Global middleware applied to every request.
	r.Use(middleware.Logger)    // logs each request with status and duration
	r.Use(middleware.Recoverer) // recovers from panics and returns 500

	// Public endpoints — no authentication required.
	r.Get("/health", healthHandler)
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)

	// Protected endpoints — the auth middleware validates the JWT before
	// the request reaches any handler inside this group.
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)

		r.Post("/tickets", ticketHandler.Create)
		r.Get("/tickets", ticketHandler.List)
		r.Get("/tickets/{id}", ticketHandler.Get)
		r.Patch("/tickets/{id}/status", ticketHandler.UpdateStatus)
	})

	log.Printf("Ticket Management API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// healthHandler responds to GET /health.
// This endpoint is public and should always return 200 OK when the
// server is running, making it suitable for load-balancer health checks.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// envOrDefault returns the value of the named environment variable,
// or the supplied default if the variable is not set.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
