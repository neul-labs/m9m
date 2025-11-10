# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO is needed for sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -extldflags '-static'" \
    -o n8n-go ./cmd/n8n-go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 n8n && \
    adduser -D -u 1000 -G n8n n8n

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/n8n-go .

# Copy example workflows and templates (optional)
COPY --from=builder /build/examples ./examples
COPY --from=builder /build/test-workflows ./test-workflows

# Create directories for data persistence
RUN mkdir -p /app/data /app/logs /app/config && \
    chown -R n8n:n8n /app

# Switch to non-root user
USER n8n

# Expose ports
# 8080: Main HTTP server
# 9090: Metrics/monitoring
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Environment variables with defaults
ENV N8N_GO_PORT=8080 \
    N8N_GO_HOST=0.0.0.0 \
    N8N_GO_LOG_LEVEL=info \
    N8N_GO_METRICS_PORT=9090 \
    N8N_GO_DATA_DIR=/app/data \
    N8N_GO_LOG_DIR=/app/logs

# Volume for persistent data
VOLUME ["/app/data", "/app/logs", "/app/config"]

# Run the application
ENTRYPOINT ["/app/n8n-go"]
CMD ["serve"]
