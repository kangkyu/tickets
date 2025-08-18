# Build stage
FROM golang:1.22-alpine AS builder

# Install git, ca-certificates, and build dependencies for CGO
RUN apk add --no-cache git ca-certificates gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application natively (no cross-compilation)
# Enable CGO for Lightspark crypto dependencies
RUN CGO_ENABLED=1 go build -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates, postgresql-client, curl, and bash
RUN apk --no-cache add ca-certificates postgresql-client curl bash

# Install dbmate
RUN curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64 && \
    chmod +x /usr/local/bin/dbmate

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/main .

# Copy database migrations
COPY db/ ./db/

# Copy startup script
COPY docker-entrypoint.sh ./

# Make files executable and set ownership
RUN chmod +x main docker-entrypoint.sh && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Use entrypoint script for proper startup
ENTRYPOINT ["./docker-entrypoint.sh"]
