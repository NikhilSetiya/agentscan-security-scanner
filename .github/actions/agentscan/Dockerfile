# AgentScan GitHub Action Docker Image
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the action binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agentscan-action ./cmd/github-action/

# Final stage
FROM alpine:3.18

# Install required packages
RUN apk add --no-cache \
    ca-certificates \
    git \
    curl \
    jq \
    bash \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S agentscan && \
    adduser -u 1001 -S agentscan -G agentscan

# Set working directory
WORKDIR /github/workspace

# Copy binary from builder
COPY --from=builder /app/agentscan-action /usr/local/bin/agentscan-action

# Copy entrypoint script
COPY .github/actions/agentscan/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Switch to non-root user
USER agentscan

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]