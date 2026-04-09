# Stage 1: Build
FROM golang:1.26-bookworm AS builder

# Install dependencies for CGO (SQLite)
RUN apt-get update && apt-get install -y gcc libc6-dev

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy static and templates first (for embedding)
COPY static ./static
COPY templates ./templates

# Copy the rest of the source code
COPY . .

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -o flow-manager main.go

# Stage 2: Final Image
FROM debian:bookworm-slim

# Install CA certificates for OIDC/HTTPS and SQLite runtime dependencies
RUN apt-get update && apt-get install -y ca-certificates libc6 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from the builder
# Note: templates and static are now embedded in the binary
COPY --from=builder /app/flow-manager .
COPY --from=builder /app/config.yaml.example ./config.yaml.example

# Create a data directory for the SQLite database
RUN mkdir -p /app/data

# Default environment variables
ENV GIN_MODE=release
ENV PORT=8080

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./flow-manager", "-port", "8080", "-config", "config.yaml"]
