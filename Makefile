.PHONY: help build test clean docker-up docker-down migrate-up migrate-down

# Default target
help:
	@echo "AgentScan Development Commands"
	@echo ""
	@echo "Build targets:"
	@echo "  build            Build all binaries"
	@echo "  build-api        Build API server"
	@echo "  build-orchestrator Build orchestrator"
	@echo "  build-cli        Build CLI tool"
	@echo ""
	@echo "Test targets:"
	@echo "  test             Run unit and integration tests"
	@echo "  test-unit        Run unit tests only"
	@echo "  test-integration Run integration tests"
	@echo "  test-e2e         Run end-to-end tests"
	@echo "  test-all         Run all test suites"
	@echo "  test-cover       Run tests with coverage report"
	@echo "  test-benchmark   Run benchmark tests"
	@echo "  test-race        Run race condition tests"
	@echo "  test-security    Run security tests"
	@echo "  test-performance Run performance tests"
	@echo "  test-clean       Clean test cache"
	@echo "  test-setup       Setup test dependencies"
	@echo "  test-teardown    Cleanup test dependencies"
	@echo "  test-ci          Run tests in CI mode"
	@echo ""
	@echo "Development targets:"
	@echo "  docker-up        Start development environment"
	@echo "  docker-down      Stop development environment"
	@echo "  migrate-up       Run database migrations"
	@echo "  migrate-down     Rollback database migrations"
	@echo "  lint             Run linters"
	@echo "  fmt              Format code"
	@echo "  clean            Clean build artifacts"

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
test: test-unit test-integration

# Unit tests - fast tests that don't require external dependencies
test-unit:
	@echo "Running unit tests..."
	@go test -v -race -short -timeout=30s ./internal/... ./pkg/...

# Integration tests - tests that require database, Redis, etc.
test-integration:
	@echo "Running integration tests..."
	@go test -v -race -tags=integration -timeout=2m ./tests/integration/...

# End-to-end tests - full system tests
test-e2e:
	@echo "Running E2E tests..."
	@RUN_E2E_TESTS=1 go test -v -race -tags=e2e -timeout=5m ./tests/e2e/...

# Run all tests
test-all: test-unit test-integration test-e2e

# Test coverage with detailed reporting
test-cover:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

# Benchmark tests
test-benchmark:
	@echo "Running benchmark tests..."
	@go test -v -bench=. -benchmem -run=^$$ ./...

# Race condition detection
test-race:
	@echo "Running race condition tests..."
	@go test -v -race -timeout=1m ./...

# Test with verbose output and no cache
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v -count=1 ./...

# Clean test cache
test-clean:
	@echo "Cleaning test cache..."
	@go clean -testcache

# Test specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-pkg PKG=./internal/config"; exit 1; fi
	@go test -v -race $(PKG)

# Generate test mocks
test-mocks:
	@echo "Generating test mocks..."
	@go generate ./...

# Test database setup for integration tests
test-db-setup:
	@echo "Setting up test database..."
	@docker run -d --name agentscan-test-db \
		-e POSTGRES_DB=agentscan_test \
		-e POSTGRES_USER=test \
		-e POSTGRES_PASSWORD=test \
		-p 5433:5432 \
		postgres:15-alpine || true

# Test database cleanup
test-db-cleanup:
	@echo "Cleaning up test database..."
	@docker stop agentscan-test-db || true
	@docker rm agentscan-test-db || true

# Test Redis setup for integration tests
test-redis-setup:
	@echo "Setting up test Redis..."
	@docker run -d --name agentscan-test-redis \
		-p 6380:6379 \
		redis:7-alpine || true

# Test Redis cleanup
test-redis-cleanup:
	@echo "Cleaning up test Redis..."
	@docker stop agentscan-test-redis || true
	@docker rm agentscan-test-redis || true

# Setup all test dependencies
test-setup: test-db-setup test-redis-setup
	@echo "Test environment setup complete"

# Cleanup all test dependencies
test-teardown: test-db-cleanup test-redis-cleanup
	@echo "Test environment cleanup complete"

# Run tests in CI environment
test-ci: test-clean test-mocks test-setup
	@echo "Running tests in CI mode..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout=5m ./...
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@$(MAKE) test-teardown

# Security tests
test-security:
	@echo "Running security tests..."
	@go test -v -tags=security ./tests/security/...

# Performance tests
test-performance:
	@echo "Running performance tests..."
	@go test -v -tags=performance -timeout=10m ./tests/performance/...

# Test report generation
test-report:
	@echo "Generating test report..."
	@go test -v -json ./... > test-report.json
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o test-coverage.html

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