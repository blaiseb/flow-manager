# Stage 1: Build
FROM golang:1.23-bookworm AS builder

# Install dependencies for CGO (SQLite)
RUN apt-get update && apt-get install -y gcc libc6-dev

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -o flow-manager main.go

# Stage 2: Final Image
FROM debian:bookworm-slim

# Install CA certificates for OIDC/HTTPS and SQLite runtime dependencies
RUN apt-get update && apt-get install -y ca-certificates libc6 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/flow-manager .

# Copy templates and default config
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/config.yaml.example ./config.yaml.example

# Create a data directory for the SQLite database
RUN mkdir -p /app/data

# Default environment variables
ENV GIN_MODE=release
ENV PORT=8080

# Expose the application port
EXPOSE 8080

# Run the application
# We use a command that points to the config in the app root
CMD ["./flow-manager", "-port", "8080", "-config", "config.yaml"]
