# AgentScan Testing Guide

This comprehensive guide covers all aspects of testing the AgentScan security scanning platform, from unit tests to chaos engineering.

## Overview

The AgentScan testing strategy follows a multi-layered approach with quality gates to ensure reliability, security, and performance:

1. **Unit Tests** - Individual component testing
2. **Integration Tests** - Component interaction testing  
3. **End-to-End Tests** - Complete user workflow testing
4. **Security Tests** - Vulnerability and penetration testing
5. **Performance Tests** - Load, stress, and benchmark testing
6. **Chaos Engineering** - System resilience testing
7. **Quality Gates** - Automated quality assurance

## Test Structure

```
tests/
├── unit/                    # Unit tests (Go and JavaScript)
├── integration/             # Integration tests
├── e2e/                     # End-to-end tests (Playwright)
├── security/                # Security and penetration tests
├── performance/             # Performance and load tests
├── chaos/                   # Chaos engineering tests
├── pipeline/                # CI/CD pipeline and quality gates
├── run-all-tests.sh         # Comprehensive test runner
└── docker-compose.test.yml  # Test environment setup
```

## Quick Start

### Prerequisites

```bash
# Install required tools
brew install go node docker docker-compose k6 jq bc

# Install Go dependencies
go mod download

# Install Node.js dependencies
cd web && npm ci
cd tests/e2e && npm ci
cd tests/security && npm ci
cd tests/performance && npm ci
cd tests/chaos && npm ci
```

### Running All Tests

```bash
# Run complete test suite
./tests/run-all-tests.sh

# Run specific test suites
./tests/run-all-tests.sh unit integration e2e

# Run tests in parallel
./tests/run-all-tests.sh --parallel

# Run with verbose output
./tests/run-all-tests.sh --verbose
```

## Test Suites

### 1. Unit Tests

Unit tests verify individual components in isolation.

**Go Unit Tests:**
```bash
# Run all Go unit tests
go test -v -race ./...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific package
go test -v ./internal/agents/semgrep/
```

**JavaScript Unit Tests:**
```bash
cd web
npm test                    # Interactive mode
npm test -- --coverage     # With coverage
npm test -- --watchAll=false --ci  # CI mode
```

**Coverage Requirements:**
- Go: 80% statement coverage
- JavaScript: 80% statement coverage, 75% branch coverage
- Critical paths: 95% coverage required

### 2. Integration Tests

Integration tests verify component interactions and API endpoints.

```bash
# Run integration tests
go test -v -tags=integration ./tests/integration/...

# Run API integration tests
cd tests/integration
npm test
```

**Test Categories:**
- Database integration
- Redis integration
- External API integration
- Agent orchestration
- Authentication flows

### 3. End-to-End Tests

E2E tests verify complete user workflows using Playwright.

```bash
cd tests/e2e

# Run all E2E tests
npm test

# Run specific test file
npx playwright test auth.spec.ts

# Run in headed mode (see browser)
npm run test:headed

# Debug mode
npm run test:debug

# Generate report
npm run test:report
```

**Test Scenarios:**
- User authentication and authorization
- Repository management
- Complete scanning workflows
- Dashboard functionality
- Real-time updates
- Error handling

### 4. Security Tests

Security tests include vulnerability assessments and penetration testing.

```bash
cd tests/security

# Run security test suite
npm test

# Run penetration tests
npm run pentest

# Run specific security tests
npm run test:auth
npm run test:injection
npm run test:xss
```

**Security Test Categories:**

**Authentication Security:**
- Brute force protection
- Session management
- Password policies
- OAuth security
- Multi-factor authentication

**Injection Attacks:**
- SQL injection
- NoSQL injection
- Command injection
- LDAP injection
- XPath injection
- Template injection

**Authorization:**
- Role-based access control
- Privilege escalation
- Direct object references
- Horizontal/vertical access

**Infrastructure Security:**
- Security headers
- TLS configuration
- CORS policies
- Information disclosure

### 5. Performance Tests

Performance tests ensure the system meets performance requirements under various load conditions.

```bash
cd tests/performance

# Run load tests
npm run test:load

# Run stress tests  
npm run test:stress

# Run spike tests
npm run test:spike

# Run API benchmarks
npm run benchmark
```

**Performance Test Types:**

**Load Testing (k6):**
- Normal load simulation
- Gradual load increase
- Sustained load testing
- Mixed workload patterns

**Stress Testing:**
- High load scenarios
- Breaking point identification
- System recovery testing
- Resource exhaustion

**Benchmarking:**
- API response times
- Throughput measurement
- Concurrent request handling
- Memory usage profiling

**Performance Thresholds:**
- API response time: <1s (95th percentile)
- Error rate: <1%
- Throughput: >100 RPS
- Memory usage: <80%

### 6. Chaos Engineering

Chaos tests verify system resilience under failure conditions.

```bash
cd tests/chaos

# Run chaos engineering tests
npm test

# Run specific experiments
npm run test:network
npm run test:database
npm run test:memory
```

**Chaos Experiments:**

**Network Chaos:**
- Latency injection
- Packet loss simulation
- Connection timeouts
- DNS failures

**Infrastructure Chaos:**
- Database connection exhaustion
- Memory pressure
- CPU throttling
- Disk space exhaustion

**Dependency Chaos:**
- External service failures
- API rate limiting
- Authentication failures
- Circuit breaker testing

### 7. Quality Gates

Quality gates ensure code meets standards before deployment.

```bash
cd tests/pipeline

# Run quality gate checks
node quality-gates.js

# Check specific gate
node quality-gates.js --gate security
```

**Quality Gate Criteria:**

| Gate | Weight | Threshold | Metrics |
|------|--------|-----------|---------|
| Code Quality | 20% | 70% | Lint errors, security issues, duplication |
| Test Coverage | 25% | 80% | Unit, integration, branch coverage |
| Security | 30% | 0 critical | Vulnerabilities by severity |
| Performance | 15% | <1s response | API times, throughput, errors |
| Reliability | 10% | 95% pass | E2E tests, chaos tests, uptime |

**Overall Requirements:**
- Minimum 75% overall score to pass
- Zero critical security vulnerabilities
- All E2E tests must pass
- Performance thresholds must be met

## CI/CD Pipeline

The automated testing pipeline runs on every commit and pull request.

### Pipeline Stages

1. **Code Quality** - Linting, security scanning, dependency checks
2. **Unit & Integration Tests** - Comprehensive test execution
3. **Security Tests** - Vulnerability assessment and penetration testing
4. **Performance Tests** - Load testing and benchmarking
5. **E2E Tests** - Complete workflow validation
6. **Chaos Tests** - Resilience validation (nightly)
7. **Quality Gates** - Final deployment readiness check

### Pipeline Configuration

```yaml
# .github/workflows/test-pipeline.yml
name: AgentScan Test Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 2 * * *'  # Nightly tests
```

### Quality Gate Enforcement

The pipeline enforces quality gates at each stage:

- **Blocking:** Critical security issues, test failures
- **Warning:** Performance degradation, coverage drops
- **Info:** Code quality improvements, optimization suggestions

## Test Data Management

### Test Environment Setup

```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run migrations
docker-compose -f docker-compose.test.yml exec api go run cmd/migrate/main.go up

# Seed test data
docker-compose -f docker-compose.test.yml exec api go run cmd/seed/main.go
```

### Test Data Categories

**User Data:**
- Admin users with full permissions
- Developer users with limited access
- Viewer users with read-only access

**Repository Data:**
- Vulnerable JavaScript applications
- Secure Python applications
- Mixed-language repositories
- Large repositories for performance testing

**Scan Data:**
- Completed scans with findings
- Failed scans for error handling
- In-progress scans for real-time testing

### Data Cleanup

Test data is automatically cleaned up after each test run:

```bash
# Manual cleanup
docker-compose -f docker-compose.test.yml down --volumes
rm -rf tmp/test-*
```

## Debugging Tests

### Common Issues

**Test Environment:**
```bash
# Check service health
curl http://localhost:8080/health

# View service logs
docker-compose -f docker-compose.test.yml logs api

# Reset environment
docker-compose -f docker-compose.test.yml down --volumes
docker-compose -f docker-compose.test.yml up -d
```

**Database Issues:**
```bash
# Check database connection
docker-compose -f docker-compose.test.yml exec postgres psql -U postgres -d agentscan_test -c "SELECT 1;"

# Reset database
docker-compose -f docker-compose.test.yml exec api go run cmd/migrate/main.go reset
docker-compose -f docker-compose.test.yml exec api go run cmd/migrate/main.go up
```

**Test Failures:**
```bash
# Run specific test with verbose output
go test -v -run TestSpecificFunction ./path/to/package/

# Run E2E test in headed mode
cd tests/e2e
npx playwright test --headed --debug specific.spec.ts
```

### Debug Tools

**Go Testing:**
```bash
# Race condition detection
go test -race ./...

# Memory profiling
go test -memprofile=mem.prof ./...
go tool pprof mem.prof

# CPU profiling
go test -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

**JavaScript Testing:**
```bash
# Debug Jest tests
cd web
npm test -- --detectOpenHandles --forceExit

# Debug Playwright tests
cd tests/e2e
npx playwright test --debug
```

## Performance Optimization

### Test Execution Speed

**Parallel Execution:**
```bash
# Run tests in parallel
./tests/run-all-tests.sh --parallel

# Parallel Go tests
go test -parallel 4 ./...

# Parallel Playwright tests
npx playwright test --workers=4
```

**Test Caching:**
```bash
# Cache Go modules
go mod download

# Cache Node modules
npm ci --cache ~/.npm

# Cache Docker layers
docker-compose -f docker-compose.test.yml build --parallel
```

### Resource Management

**Memory Optimization:**
- Limit concurrent test execution
- Clean up test data between tests
- Use test-specific databases
- Monitor memory usage during tests

**CPU Optimization:**
- Balance parallel execution
- Use appropriate test timeouts
- Optimize test setup/teardown
- Profile slow tests

## Monitoring and Reporting

### Test Metrics

The testing system tracks comprehensive metrics:

**Execution Metrics:**
- Test duration and performance
- Success/failure rates
- Coverage percentages
- Resource utilization

**Quality Metrics:**
- Code quality scores
- Security vulnerability counts
- Performance benchmarks
- Reliability indicators

### Reporting

**Automated Reports:**
- HTML test reports with detailed results
- JSON reports for programmatic access
- Coverage reports with visual indicators
- Performance trend analysis

**Integration:**
- Slack notifications for failures
- GitHub status checks
- SonarQube integration
- Codecov coverage tracking

## Best Practices

### Writing Tests

**Unit Tests:**
- Test one thing at a time
- Use descriptive test names
- Mock external dependencies
- Test edge cases and error conditions
- Maintain high coverage

**Integration Tests:**
- Test realistic scenarios
- Use test databases
- Clean up after tests
- Test error handling
- Verify end-to-end flows

**E2E Tests:**
- Focus on user journeys
- Use stable selectors
- Handle async operations
- Test across browsers
- Keep tests independent

### Test Maintenance

**Regular Tasks:**
- Update test dependencies
- Review and update test data
- Optimize slow tests
- Remove obsolete tests
- Update documentation

**Monitoring:**
- Track test execution times
- Monitor flaky tests
- Review coverage trends
- Analyze failure patterns
- Update performance baselines

## Troubleshooting

### Common Problems

**Flaky Tests:**
- Add proper waits for async operations
- Use stable element selectors
- Handle race conditions
- Increase timeouts for slow operations
- Isolate test data

**Performance Issues:**
- Profile slow tests
- Optimize database queries
- Reduce test data size
- Use parallel execution
- Cache expensive operations

**Environment Issues:**
- Verify service dependencies
- Check port conflicts
- Validate environment variables
- Review Docker resource limits
- Monitor disk space

### Getting Help

**Resources:**
- Internal documentation: `docs/`
- Test examples: `tests/examples/`
- Troubleshooting guides: `docs/troubleshooting/`
- Team chat: `#testing` channel

**Support:**
- Create GitHub issues for bugs
- Ask questions in team chat
- Review existing documentation
- Check CI/CD pipeline logs

## Conclusion

The AgentScan testing strategy provides comprehensive coverage across all aspects of the system, from individual components to complete user workflows. By following this guide and maintaining the testing standards, we ensure a reliable, secure, and performant security scanning platform.

The multi-layered approach with quality gates provides confidence in deployments while the automated pipeline ensures consistent quality standards. Regular monitoring and optimization of the test suite maintains its effectiveness as the system evolves.