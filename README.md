# Ticket System

A production-quality Ticket Management REST API built in Go.

---

## Overview

This service provides user registration, JWT-based authentication, and full ticket lifecycle management. Each user can only view and manage their own tickets. Ticket status follows a strict one-way flow: `open → in_progress → closed`.

---

## Features

- User registration with bcrypt password hashing
- JWT login with configurable expiry
- Ownership-based ticket access (users see only their own tickets)
- Ticket status validation and linear transition enforcement
- Consistent JSON responses with meaningful HTTP status codes
- SQLite database (zero external dependencies)
- Multi-stage Docker build
- Health check endpoint

---

## Architecture

```
HTTP Request
     │
     ▼
  chi Router
     │
     ├─── Public routes ──────► Handlers
     │
     └─── Auth Middleware ────► JWT Validation
                                     │
                                     ▼
                                 Handlers  (parse, validate, respond)
                                     │
                                     ▼
                                 Services  (business logic)
                                     │
                                     ▼
                               Repositories  (SQL queries)
                                     │
                                     ▼
                                  SQLite
```

---

## Technology Stack

| Concern           | Library / Tool                      |
|-------------------|-------------------------------------|
| Language          | Go 1.23                             |
| Router            | `go-chi/chi/v5`                     |
| JWT               | `golang-jwt/jwt/v5`                 |
| Password hashing  | `golang.org/x/crypto/bcrypt`        |
| Database          | SQLite via `mattn/go-sqlite3`       |
| Env loading       | `joho/godotenv`                     |
| Container         | Docker (multi-stage, Alpine)        |

---

## Project Structure

```
ticket-system/
├── cmd/
│   └── server/
│       └── main.go          # Entry point — wires dependencies, starts server
├── internal/
│   ├── auth/
│   │   ├── handler.go       # POST /auth/register, POST /auth/login
│   │   ├── middleware.go    # JWT validation middleware
│   │   └── service.go       # Register, Login, JWT generation/parsing
│   ├── ticket/
│   │   ├── handler.go       # Ticket HTTP handlers
│   │   ├── model.go         # Ticket struct and status constants
│   │   ├── repository.go    # SQL queries for tickets
│   │   ├── service.go       # Business logic, status transition rules
│   │   └── service_test.go  # Unit tests
│   ├── user/
│   │   ├── model.go         # User struct
│   │   └── repository.go    # SQL queries for users
│   └── response/
│       └── response.go      # JSON / Error response helpers
├── database/
│   └── db.go                # SQLite connection + schema init
├── .env.example
├── .gitignore
├── .dockerignore
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

---

## Authentication

All protected routes require:

```
Authorization: Bearer <JWT>
```

The JWT is obtained from `POST /auth/login`. It expires after **24 hours**.

The user ID is extracted exclusively from the validated JWT — the server never trusts a user ID supplied in the request body.

---

## API Documentation

### GET /health

Public. Returns service status.

**Response `200 OK`:**
```json
{ "status": "ok" }
```

---

### POST /auth/register

Register a new account.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Responses:**

| Status | Meaning |
|--------|---------|
| `201 Created` | Registration successful |
| `400 Bad Request` | Missing/invalid fields |
| `409 Conflict` | Email already registered |

**Success `201`:**
```json
{ "message": "user registered successfully" }
```

---

### POST /auth/login

Login and receive a JWT.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Responses:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Login successful |
| `400 Bad Request` | Missing fields |
| `401 Unauthorized` | Invalid credentials |

**Success `200`:**
```json
{ "token": "eyJ..." }
```

---

### POST /tickets

**Auth required.** Create a new ticket.

**Request:**
```json
{
  "title": "Unable to login",
  "description": "The login page returns an error."
}
```

**Success `201`:**
```json
{
  "id": 1,
  "user_id": 1,
  "title": "Unable to login",
  "description": "The login page returns an error.",
  "status": "open",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

### GET /tickets

**Auth required.** List all tickets owned by the authenticated user.

**Success `200`:**
```json
[
  { "id": 1, "user_id": 1, "title": "...", "status": "open", ... },
  { "id": 2, "user_id": 1, "title": "...", "status": "in_progress", ... }
]
```

Returns an empty array `[]` if the user has no tickets.

---

### GET /tickets/{id}

**Auth required.** Get a single ticket owned by the authenticated user.

**Responses:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Ticket found |
| `400 Bad Request` | Invalid ID format |
| `401 Unauthorized` | Missing/invalid JWT |
| `404 Not Found` | Ticket not found or belongs to another user |

---

### PATCH /tickets/{id}/status

**Auth required.** Update the status of a ticket.

**Request:**
```json
{ "status": "in_progress" }
```

**Valid statuses:** `open`, `in_progress`, `closed`

**Responses:**

| Status | Meaning |
|--------|---------|
| `200 OK` | Status updated |
| `400 Bad Request` | Invalid status or illegal transition |
| `404 Not Found` | Ticket not found or belongs to another user |

**Success `200`:** returns the full updated ticket object.

---

## Ticket Status Flow

```
  open
   │
   ▼
in_progress
   │
   ▼
 closed   ← terminal state, cannot be reopened
```

Attempting any other transition returns `400 Bad Request`.

---

## Local Setup

**Prerequisites:** Go 1.21+, gcc (for SQLite CGO)

```bash
# 1. Clone
git clone <your-repo-url>
cd ticket-system

# 2. Copy environment file and set your secret
cp .env.example .env
# Edit .env: set JWT_SECRET to a long random string

# 3. Install dependencies
go mod download

# 4. Run the server
go run ./cmd/server

# 5. Verify
curl http://localhost:8080/health
```

---

## Environment Variables

| Variable        | Required | Default                    | Description                         |
|-----------------|----------|----------------------------|-------------------------------------|
| `JWT_SECRET`    | **Yes**  | —                          | Secret key for signing JWTs         |
| `PORT`          | No       | `8080`                     | Port the server listens on          |
| `DATABASE_PATH` | No       | `./database/tickets.db`    | Path to the SQLite database file    |

---

## Docker Setup

### Option 1: Docker Compose (Recommended)

```bash
# Start container with Docker Compose
docker compose up --build -d

# Verify
curl http://localhost:8080/health
# → {"status":"ok"}

# Stop container
docker compose down
```

### Option 2: Docker CLI

```bash
# Build the image
docker build -t ticket-system .

# Run container with environment variables
docker run -p 8080:8080 -e JWT_SECRET=my-secret-key ticket-system

# Verify
curl http://localhost:8080/health
# → {"status":"ok"}
```

To persist the database using Docker CLI:

```bash
docker run -p 8080:8080 \
  -e JWT_SECRET=my-secret-key \
  -v $(pwd)/database:/app/database \
  ticket-system
```

---

## Testing

```bash
# Run all unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Run static analysis
go vet ./...
```

---

## Assumptions

- SQLite is sufficient for this assignment's workload.
- A single JWT_SECRET is used (no key rotation).
- Ticket ownership cannot be transferred.
- Passwords must be non-empty (no minimum length enforced beyond that).

---

## Author

Built as a Golang Backend Intern Assignment.
