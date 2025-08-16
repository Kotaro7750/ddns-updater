# Multi-stage build Dockerfile
# Build stage
FROM golang:1.23.3-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary with CGO disabled
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ddns-updater .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy built binary
COPY --from=builder /app/ddns-updater /ddns-updater

# Set user for security (using nobody user ID)
USER 1000:1000

# Run the application
ENTRYPOINT ["/ddns-updater"]
