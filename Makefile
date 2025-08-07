.PHONY: help build test clean docker-up docker-down migrate-up migrate-down

# Default target
help:
	@echo "AgentScan Development Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build all binaries"
	@echo "  test         Run all tests"
	@echo "  test-cover   Run tests with coverage"
	@echo "  clean        Clean build artifacts"
	@echo "  docker-up    Start development environment"
	@echo "  docker-down  Stop development environment"
	@echo "  migrate-up   Run database migrations"
	@echo "  migrate-down Rollback database migrations"
	@echo "  lint         Run linters"
	@echo "  fmt          Format code"

# Build targets
build: build-api build-orchestrator build-cli build-migrate

build-api:
	@echo "Building API server..."
	@go build -o bin/api ./cmd/api

build-orchestrator:
	@echo "Building orchestrator..."
	@go build -o bin/orchestrator ./cmd/orchestrator

build-cli:
	@echo "Building CLI..."
	@go build -o bin/agentscan ./cmd/cli

# Test targets
test:
	@echo "Running tests..."
	@go test ./...

test-cover:
	@echo "Running tests with coverage..."
	@go test -cover ./...
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

test-integration:
	@echo "Running integration tests..."
	@INTEGRATION_TESTS=1 go test -tags=integration ./internal/database ./internal/queue

# Development environment
docker-up:
	@echo "Starting development environment..."
	@docker-compose up -d

docker-down:
	@echo "Stopping development environment..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

# Database migrations
migrate-up: build-migrate
	@echo "Running database migrations..."
	@./bin/migrate up

migrate-down: build-migrate
	@echo "Rolling back database migrations..."
	@./bin/migrate down

migrate-version: build-migrate
	@./bin/migrate version

migrate-force: build-migrate
	@echo "Usage: make migrate-force VERSION=<version>"
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required"; exit 1; fi
	@./bin/migrate force $(VERSION)

build-migrate:
	@echo "Building migration tool..."
	@go build -o bin/migrate ./cmd/migrate

# Code quality
lint:
	@echo "Running linters..."
	@golangci-lint run

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# Clean up
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf tmp/
	@rm -f coverage.out coverage.html

# Install development dependencies
install-deps:
	@echo "Installing development dependencies..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Docker build targets
docker-build: docker-build-api docker-build-orchestrator

docker-build-api:
	@echo "Building API Docker image..."
	@docker build -f Dockerfile.api -t agentscan/api:latest .

docker-build-orchestrator:
	@echo "Building orchestrator Docker image..."
	@docker build -f Dockerfile.orchestrator -t agentscan/orchestrator:latest .

# Development shortcuts
dev-api: build-api
	@echo "Starting API server in development mode..."
	@./bin/api

dev-orchestrator: build-orchestrator
	@echo "Starting orchestrator in development mode..."
	@./bin/orchestrator

# Generate code (TODO: implement code generation)
generate:
	@echo "Generating code..."
	@go generate ./...