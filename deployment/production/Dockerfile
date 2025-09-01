# Multi-stage production Dockerfile for AgentScan
# Stage 1: Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binaries with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o bin/api ./cmd/api

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o bin/orchestrator ./cmd/orchestrator

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o bin/migrate ./cmd/migrate

# Stage 2: Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 -S agentscan && \
    adduser -u 1001 -S agentscan -G agentscan

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/api /app/bin/api
COPY --from=builder /app/bin/orchestrator /app/bin/orchestrator
COPY --from=builder /app/bin/migrate /app/bin/migrate

# Copy configuration files
COPY --from=builder /app/configs/ /app/configs/
COPY --from=builder /app/migrations/ /app/migrations/

# Create data directory
RUN mkdir -p /data && chown -R agentscan:agentscan /data

# Set permissions
RUN chmod +x /app/bin/api /app/bin/orchestrator /app/bin/migrate
RUN chown -R agentscan:agentscan /app

# Switch to non-root user
USER agentscan

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose ports
EXPOSE 8080 9090

# Default command
CMD ["/app/bin/api"]