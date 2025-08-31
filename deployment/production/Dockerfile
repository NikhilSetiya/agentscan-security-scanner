# Production Multi-Service Dockerfile for AgentScan Security Scanner
# Supports building API server, orchestrator, and CLI tools
# Usage: docker build --target api -t agentscan-api .
#        docker build --target orchestrator -t agentscan-orchestrator .
#        docker build --target cli -t agentscan-cli .

# =============================================================================
# Base builder stage with common dependencies
# =============================================================================
FROM golang:1.23.9-alpine AS base-builder

# Install build dependencies in single layer for better caching
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    upx \
    make \
    && rm -rf /var/cache/apk/*

# Set working directory
WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download and verify dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# =============================================================================
# API server builder
# =============================================================================
FROM base-builder AS api-builder

# Build API server with aggressive optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags \"-static\" -X main.version=$(git describe --tags --always --dirty) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)' \
    -a -installsuffix cgo \
    -trimpath \
    -o api ./cmd/api

# Compress binary with UPX for smaller image size
RUN upx --best --lzma api

# =============================================================================
# Orchestrator builder
# =============================================================================
FROM base-builder AS orchestrator-builder

# Build orchestrator with aggressive optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags \"-static\" -X main.version=$(git describe --tags --always --dirty) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)' \
    -a -installsuffix cgo \
    -trimpath \
    -o orchestrator ./cmd/orchestrator

# Compress binary with UPX
RUN upx --best --lzma orchestrator

# =============================================================================
# CLI builder
# =============================================================================
FROM base-builder AS cli-builder

# Build CLI tools
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags \"-static\" -X main.version=$(git describe --tags --always --dirty) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)' \
    -a -installsuffix cgo \
    -trimpath \
    -o cli ./cmd/cli

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags \"-static\"' \
    -a -installsuffix cgo \
    -trimpath \
    -o migrate ./cmd/migrate

# Compress binaries\nRUN upx --best --lzma cli migrate\n\n# =============================================================================\n# Production base image with security hardening\n# =============================================================================\nFROM gcr.io/distroless/static:nonroot AS production-base\n\n# Copy CA certificates and timezone data from builder\nCOPY --from=base-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/\nCOPY --from=base-builder /usr/share/zoneinfo /usr/share/zoneinfo\n\n# Set timezone\nENV TZ=UTC\n\n# =============================================================================\n# API server production image\n# =============================================================================\nFROM production-base AS api\n\n# Metadata labels\nLABEL maintainer=\"AgentScan Team\"\nLABEL version=\"1.0.0\"\nLABEL description=\"AgentScan Security Scanner API Server\"\nLABEL com.agentscan.service=\"api\"\nLABEL com.agentscan.resource.memory=\"512Mi\"\nLABEL com.agentscan.resource.cpu=\"500m\"\n\n# Copy the API binary\nCOPY --from=api-builder /app/api /usr/local/bin/api\n\n# Health check using the binary itself\nHEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \\\n    CMD [\"/usr/local/bin/api\", \"health\"]\n\n# Expose port\nEXPOSE 8080\n\n# Run as non-root user (distroless nonroot user)\nUSER 65532:65532\n\n# Set entrypoint\nENTRYPOINT [\"/usr/local/bin/api\"]\nCMD [\"serve\"]\n\n# =============================================================================\n# Orchestrator production image (needs Docker CLI)\n# =============================================================================\nFROM alpine:3.19 AS orchestrator\n\n# Metadata labels\nLABEL maintainer=\"AgentScan Team\"\nLABEL version=\"1.0.0\"\nLABEL description=\"AgentScan Security Scanner Orchestrator\"\nLABEL com.agentscan.service=\"orchestrator\"\nLABEL com.agentscan.resource.memory=\"1Gi\"\nLABEL com.agentscan.resource.cpu=\"1000m\"\n\n# Install runtime dependencies in single layer\nRUN apk --no-cache add \\\n    ca-certificates \\\n    tzdata \\\n    docker-cli \\\n    curl \\\n    && rm -rf /var/cache/apk/*\n\n# Create optimized user and group setup\nRUN addgroup -g 1001 -S agentscan && \\\n    adduser -u 1001 -S agentscan -G agentscan && \\\n    addgroup agentscan docker\n\n# Create necessary directories with proper permissions\nRUN mkdir -p /tmp/agentscan /var/log/agentscan && \\\n    chown -R agentscan:agentscan /tmp/agentscan /var/log/agentscan\n\n# Copy CA certificates and timezone data\nCOPY --from=base-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/\nCOPY --from=base-builder /usr/share/zoneinfo /usr/share/zoneinfo\n\n# Copy the orchestrator binary\nCOPY --from=orchestrator-builder /app/orchestrator /usr/local/bin/orchestrator\n\n# Set ownership and permissions\nRUN chown agentscan:agentscan /usr/local/bin/orchestrator && \\\n    chmod +x /usr/local/bin/orchestrator\n\n# Switch to non-root user\nUSER agentscan:agentscan\n\n# Set working directory\nWORKDIR /home/agentscan\n\n# Health check\nHEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \\\n    CMD [\"/usr/local/bin/orchestrator\", \"health\"]\n\n# Set timezone\nENV TZ=UTC\n\n# Set entrypoint\nENTRYPOINT [\"/usr/local/bin/orchestrator\"]\nCMD [\"run\"]\n\n# =============================================================================\n# CLI tools image\n# =============================================================================\nFROM production-base AS cli\n\n# Metadata labels\nLABEL maintainer=\"AgentScan Team\"\nLABEL version=\"1.0.0\"\nLABEL description=\"AgentScan Security Scanner CLI Tools\"\nLABEL com.agentscan.service=\"cli\"\n\n# Copy CLI binaries\nCOPY --from=cli-builder /app/cli /usr/local/bin/agentscan\nCOPY --from=cli-builder /app/migrate /usr/local/bin/agentscan-migrate\n\n# Health check for CLI (just version check)\nHEALTHCHECK --interval=60s --timeout=5s --start-period=5s --retries=2 \\\n    CMD [\"/usr/local/bin/agentscan\", \"version\"]\n\n# Run as non-root user\nUSER 65532:65532\n\n# Default entrypoint\nENTRYPOINT [\"/usr/local/bin/agentscan\"]\nCMD [\"--help\"]\n\n# =============================================================================\n# Development image with all tools\n# =============================================================================\nFROM golang:1.23.9-alpine AS development\n\n# Install development dependencies\nRUN apk add --no-cache \\\n    git \\\n    ca-certificates \\\n    tzdata \\\n    make \\\n    curl \\\n    docker-cli \\\n    bash \\\n    vim \\\n    && rm -rf /var/cache/apk/*\n\n# Set working directory\nWORKDIR /app\n\n# Copy source code\nCOPY . .\n\n# Install Go tools for development\nRUN go install github.com/air-verse/air@latest && \\\n    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest\n\n# Expose ports for development\nEXPOSE 8080 8081\n\n# Default command for development\nCMD [\"air\", \"-c\", \".air.toml\"]"