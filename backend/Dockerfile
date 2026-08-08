# ============= Build Stage =============
FROM golang:1.24 AS builder

# Set working directory
WORKDIR /app

# Copy dependency files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build Go binary
RUN go build -o goapp .

# ============= Runtime Stage ============
FROM debian:bookworm-slim

# Set working directory
WORKDIR /app

# Copy goapp only from Build Stage
COPY --from=builder /app/goapp .

# Expose application port(Only for documentation)
EXPOSE 8080

# Run the Go Application
CMD ["./goapp"]
