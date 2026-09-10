# ─── Stage 1: Build ────────────────────────────────────────────────────────────
# We use the Alpine-based Go image. Since we use modernc.org/sqlite (a pure-Go
# SQLite port), no C compiler (CGO) is required. This simplifies the build.
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency manifests first so Docker layer caching skips the
# module download step when only source files change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code and build the binary.
# CGO_ENABLED=0 produces a fully static binary that runs in scratch/alpine.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# ─── Stage 2: Runtime ──────────────────────────────────────────────────────────
# Use a minimal Alpine image for the final container. This keeps the
# runtime image small and reduces the attack surface.
FROM alpine:3.19

WORKDIR /app

# Copy only the compiled binary from the builder stage.
COPY --from=builder /app/server .

# Create the database directory so the SQLite file can be written there.
RUN mkdir -p /app/database

# The application listens on port 8080.
EXPOSE 8080

# Environment variable defaults — override at runtime with -e flags.
# JWT_SECRET has no default because omitting it causes a clean startup failure.
ENV PORT=8080
ENV DATABASE_PATH=/app/database/tickets.db

# Run the binary directly (not via a shell) for clean signal handling.
CMD ["/app/server"]
