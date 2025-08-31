# Testing Guide

This document provides comprehensive information about testing in the AgentScan project.

## Table of Contents

- [Overview](#overview)
- [Test Types](#test-types)
- [Running Tests](#running-tests)
- [Test Structure](#test-structure)
- [Writing Tests](#writing-tests)
- [Test Configuration](#test-configuration)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

AgentScan uses a comprehensive testing strategy that includes:

- **Unit Tests**: Fast, isolated tests for individual components
- **Integration Tests**: Tests that verify component interactions
- **End-to-End Tests**: Full system tests simulating real user scenarios
- **Security Tests**: Tests focused on security vulnerabilities
- **Performance Tests**: Tests that measure system performance
- **Benchmark Tests**: Tests that measure code performance

## Test Types

### Unit Tests

Unit tests are fast, isolated tests that test individual functions or methods without external dependencies.

**Location**: `internal/*/` (alongside source code)
**Naming**: `*_test.go`
**Command**: `make test-unit`

```go
func TestConfigValidation(t *testing.T) {
    config := &Config{
        App: AppConfig{Name: "test"},
    }
    
    err := config.Validate()
    assert.NoError(t, err)
}
```

### Integration Tests

Integration tests verify that different components work together correctly. They may use real databases, Redis, or other external services.

**Location**: `tests/integration/`
**Naming**: `*_integration_test.go`
**Command**: `make test-integration`

```go
func TestDatabaseIntegration(t *testing.T) {
    testing.IntegrationTest(t)
    
    suite := testing.NewTestSuite(t)
    defer suite.Cleanup()
    
    db := suite.SetupDatabase()
    // Test database operations
}
```

### End-to-End Tests

E2E tests simulate real user scenarios and test the complete system.

**Location**: `tests/e2e/`
**Naming**: `*_e2e_test.go`
**Command**: `make test-e2e`

### Security Tests

Security tests focus on identifying security vulnerabilities.

**Location**: `tests/security/`
**Command**: `make test-security`

### Performance Tests

Performance tests measure system performance under various conditions.

**Location**: `tests/performance/`
**Command**: `make test-performance`

### Benchmark Tests

Benchmark tests measure code performance and help identify bottlenecks.

**Command**: `make test-benchmark`

```go
func BenchmarkConfigLoad(b *testing.B) {
    for i := 0; i < b.N; i++ {
        config, _ := LoadConfig()
        _ = config
    }
}
```

## Running Tests

### Quick Commands

```bash
# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-e2e

# Run tests with coverage
make test-cover

# Run benchmark tests
make test-benchmark

# Run security tests
make test-security
```

### Advanced Commands

```bash
# Run tests for specific package
make test-pkg PKG=./internal/config

# Run tests with race detection
make test-race

# Run tests with verbose output
make test-verbose

# Clean test cache
make test-clean

# Run tests in CI mode
make test-ci
```

### Test Environment Setup

```bash
# Setup test dependencies (PostgreSQL, Redis)
make test-setup

# Cleanup test dependencies
make test-teardown
```

## Test Structure

### Directory Structure

```
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go          # Unit tests
│   └── api/
│       ├── handlers.go
│       └── handlers_test.go        # Unit tests
├── tests/
│   ├── integration/
│   │   ├── api_integration_test.go # Integration tests
│   │   └── database_test.go
│   ├── e2e/
│   │   └── user_journey_test.go    # E2E tests
│   ├── security/
│   │   └── security_test.go        # Security tests
│   ├── performance/
│   │   └── load_test.go           # Performance tests
│   └── testdata/
│       ├── config.test.yaml       # Test configuration
│       └── fixtures/              # Test fixtures
└── pkg/
    └── utils/
        ├── utils.go
        └── utils_test.go           # Unit tests
```

### Test Utilities

The project provides comprehensive test utilities in `internal/shared/testing/`:

- `TestSuite`: Main test suite with setup/cleanup
- `TestLogger`: Logger for testing with assertion methods
- Database and Redis setup with testcontainers
- HTTP client and server utilities
- Mock configuration helpers

## Writing Tests

### Basic Test Structure

```go
package mypackage

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/your-org/agentscan/internal/shared/testing"
)

func TestMyFunction(t *testing.T) {
    // Arrange
    suite := testing.NewTestSuite(t)
    defer suite.Cleanup()
    
    // Act
    result := MyFunction("input")
    
    // Assert
    assert.Equal(t, "expected", result)
}
```

### Integration Test Structure

```go
func TestDatabaseOperations(t *testing.T) {
    testing.IntegrationTest(t) // Skip in short mode
    
    suite := testing.NewTestSuite(t)
    defer suite.Cleanup()
    
    db := suite.SetupDatabase()
    
    // Test database operations
    err := CreateUser(db, user)
    assert.NoError(t, err)
}
```

### Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectError bool
    }{
        {"valid input", "valid", false},
        {"invalid input", "invalid", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### HTTP API Tests

```go
func TestAPIEndpoint(t *testing.T) {
    suite := testing.NewTestSuite(t)
    defer suite.Cleanup()
    
    router := setupTestRouter()
    server := suite.SetupHTTPServer(router)
    
    resp := suite.HTTPRequest("GET", server.URL+"/api/users", nil, nil)
    suite.AssertHTTPStatus(resp, 200)
}
```

## Test Configuration

### Environment Variables

Tests use specific environment variables:

```bash
GO_ENV=test                    # Test environment
DATABASE_URL=postgres://...    # Test database
REDIS_URL=redis://...         # Test Redis
RUN_E2E_TESTS=1               # Enable E2E tests
```

### Test Configuration File

Tests use `tests/testdata/config.test.yaml` for configuration:

```yaml
app:
  name: "AgentScan Test"
  environment: "test"
  debug: true

database:
  url: "postgres://test:test@localhost:5432/agentscan_test"

test:
  timeouts:
    unit_test: "30s"
    integration_test: "2m"
    e2e_test: "5m"
```

### Test Fixtures

Test fixtures are stored in `tests/testdata/fixtures/`:

- `users.json`: Test user data
- `scans.json`: Test scan data
- `agents.json`: Test agent data

## CI/CD Integration

### GitHub Actions

The project uses GitHub Actions for automated testing:

- **Unit Tests**: Run on every push/PR
- **Integration Tests**: Run with PostgreSQL and Redis services
- **Security Tests**: Run gosec and custom security tests
- **Coverage**: Generate and upload coverage reports
- **Multi-Version**: Test against multiple Go versions

### Test Workflow

1. **Unit Tests**: Fast feedback on basic functionality
2. **Integration Tests**: Verify component interactions
3. **Security Tests**: Check for vulnerabilities
4. **Coverage Report**: Ensure adequate test coverage
5. **Benchmark Tests**: Monitor performance
6. **Race Detection**: Check for race conditions

### Coverage Requirements

- **Minimum Coverage**: 80%
- **Critical Packages**: 90%+ (config, security, api)
- **Coverage Report**: Generated on every CI run

## Best Practices

### General Guidelines

1. **Test Naming**: Use descriptive test names that explain what is being tested
2. **Test Structure**: Follow Arrange-Act-Assert pattern
3. **Test Independence**: Tests should not depend on each other
4. **Test Data**: Use fixtures and factories for test data
5. **Cleanup**: Always clean up resources in tests

### Unit Test Guidelines

1. **Fast**: Unit tests should run quickly (< 1s each)
2. **Isolated**: No external dependencies
3. **Deterministic**: Same input should always produce same output
4. **Focused**: Test one thing at a time

### Integration Test Guidelines

1. **Real Dependencies**: Use real databases, Redis, etc.
2. **Testcontainers**: Use testcontainers for consistent environments
3. **Cleanup**: Always clean up test data
4. **Timeouts**: Set appropriate timeouts for external services

### Security Test Guidelines

1. **Input Validation**: Test with malicious inputs
2. **Authentication**: Test authentication and authorization
3. **Injection**: Test for SQL injection, XSS, etc.
4. **Rate Limiting**: Test rate limiting mechanisms

### Performance Test Guidelines

1. **Baseline**: Establish performance baselines
2. **Load Testing**: Test under realistic load
3. **Resource Usage**: Monitor memory and CPU usage
4. **Bottlenecks**: Identify and test bottlenecks

## Troubleshooting

### Common Issues

#### Tests Fail in CI but Pass Locally

**Cause**: Environment differences
**Solution**: 
- Check environment variables
- Ensure consistent test data
- Use testcontainers for dependencies

#### Flaky Tests

**Cause**: Race conditions, timing issues, external dependencies
**Solution**:
- Use proper synchronization
- Add retries for external services
- Mock external dependencies

#### Slow Tests

**Cause**: Inefficient test setup, too many external calls
**Solution**:
- Optimize test setup
- Use mocks for external services
- Parallelize tests where possible

#### Memory Leaks in Tests

**Cause**: Not cleaning up resources
**Solution**:
- Use defer for cleanup
- Close database connections
- Stop test servers

### Debugging Tests

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestSpecificFunction ./internal/config

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Profile tests
go test -cpuprofile cpu.prof -memprofile mem.prof ./...
```

### Test Environment Issues

#### Database Connection Issues

```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Check connection
psql -h localhost -p 5432 -U test -d agentscan_test

# Reset test database
make test-db-cleanup
make test-db-setup
```

#### Redis Connection Issues

```bash
# Check if Redis is running
docker ps | grep redis

# Check connection
redis-cli -h localhost -p 6379 ping

# Reset test Redis
make test-redis-cleanup
make test-redis-setup
```

## Test Metrics

### Coverage Targets

- **Overall**: 80%
- **Critical Packages**: 90%
- **New Code**: 90%

### Performance Targets

- **Unit Tests**: < 30 seconds total
- **Integration Tests**: < 2 minutes total
- **E2E Tests**: < 5 minutes total

### Quality Metrics

- **Test Reliability**: > 99% (flaky test rate < 1%)
- **Test Maintenance**: Tests should be updated with code changes
- **Documentation**: All test utilities should be documented

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Testcontainers Go](https://golang.testcontainers.org/)
- [Go Race Detector](https://golang.org/doc/articles/race_detector.html)
- [Benchmark Tests](https://pkg.go.dev/testing#hdr-Benchmarks)