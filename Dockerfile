# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o log-trimmer \
    ./cmd/log-trimmer

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/log-trimmer .

# Create log directory
RUN mkdir -p /var/log/log-trimmer && \
    chown -R appuser:appgroup /var/log/log-trimmer

# Change ownership of the app directory
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose no ports (this is a CLI tool)

# Set entrypoint
ENTRYPOINT ["./log-trimmer"]

# Default command shows help
CMD ["--help"]

# Labels
LABEL maintainer="your-email@company.com" \
      version="1.0.0" \
      description="Elasticsearch Log Trimmer - Clean up old log indexes" \
      org.opencontainers.image.source="https://github.com/company/log-trimmer" \
      org.opencontainers.image.documentation="https://github.com/company/log-trimmer/README.md" \
      org.opencontainers.image.licenses="MIT"
