#!/bin/bash

set -euo pipefail

# Comprehensive Test Runner for AgentScan
# This script runs all test suites in the correct order with proper setup and teardown

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PARALLEL_TESTS=${PARALLEL_TESTS:-false}
SKIP_SETUP=${SKIP_SETUP:-false}
SKIP_TEARDOWN=${SKIP_TEARDOWN:-false}
TEST_ENVIRONMENT=${TEST_ENVIRONMENT:-test}
VERBOSE=${VERBOSE:-false}

# Test results
RESULTS_DIR="test-results"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
REPORT_FILE="$RESULTS_DIR/test-report-$TIMESTAMP.json"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# Function to show help
show_help() {
    cat << EOF
AgentScan Comprehensive Test Runner

Usage: $0 [OPTIONS] [TEST_SUITES...]

Options:
    --parallel              Run tests in parallel where possible
    --skip-setup           Skip test environment setup
    --skip-teardown        Skip test environment teardown
    --environment ENV      Test environment (test, staging, production)
    --verbose              Enable verbose logging
    --help                 Show this help message

Test Suites:
    unit                   Unit tests (Go and JavaScript)
    integration            Integration tests
    e2e                    End-to-end tests
    security               Security tests
    performance            Performance and load tests
    chaos                  Chaos engineering tests
    all                    All test suites (default)

Examples:
    $0                     # Run all tests
    $0 unit integration    # Run only unit and integration tests
    $0 --parallel e2e      # Run E2E tests in parallel
    $0 --environment staging security  # Run security tests against staging

Environment Variables:
    PARALLEL_TESTS         Run tests in parallel (true/false)
    SKIP_SETUP            Skip setup (true/false)
    SKIP_TEARDOWN         Skip teardown (true/false)
    TEST_ENVIRONMENT      Test environment
    VERBOSE               Verbose logging (true/false)
EOF
}

# Function to setup test environment
setup_test_environment() {
    if [[ "$SKIP_SETUP" == "true" ]]; then
        log_info "Skipping test environment setup"
        return
    fi

    log_info "Setting up test environment..."
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Start test services
    log_verbose "Starting test services with Docker Compose..."
    docker-compose -f docker-compose.test.yml down --remove-orphans
    docker-compose -f docker-compose.test.yml up -d
    
    # Wait for services to be ready
    log_info "Waiting for services to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            log_success "Services are ready"
            break
        fi
        
        log_verbose "Waiting for services... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    if [[ $attempt -gt $max_attempts ]]; then
        log_error "Services failed to start within timeout"
        exit 1
    fi
    
    # Run database migrations
    log_verbose "Running database migrations..."
    docker-compose -f docker-compose.test.yml exec -T api go run cmd/migrate/main.go up
    
    # Seed test data
    log_verbose "Seeding test data..."
    docker-compose -f docker-compose.test.yml exec -T api go run cmd/seed/main.go
    
    log_success "Test environment setup completed"
}

# Function to teardown test environment
teardown_test_environment() {
    if [[ "$SKIP_TEARDOWN" == "true" ]]; then
        log_info "Skipping test environment teardown"
        return
    fi

    log_info "Tearing down test environment..."
    
    # Stop and remove containers
    docker-compose -f docker-compose.test.yml down --remove-orphans --volumes
    
    # Clean up test data
    log_verbose "Cleaning up test data..."
    rm -rf tmp/test-*
    
    log_success "Test environment teardown completed"
}

# Function to run unit tests
run_unit_tests() {
    log_info "Running unit tests..."
    
    local start_time=$(date +%s)
    local success=true
    
    # Go unit tests
    log_verbose "Running Go unit tests..."
    if ! go test -v -race -coverprofile=coverage.out ./...; then
        success=false
    fi
    
    # Generate coverage report
    go tool cover -html=coverage.out -o "$RESULTS_DIR/go-coverage.html"
    
    # JavaScript unit tests
    log_verbose "Running JavaScript unit tests..."
    cd web
    if ! npm test -- --coverage --watchAll=false --ci; then
        success=false
    fi
    cd ..
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ "$success" == "true" ]]; then
        log_success "Unit tests completed in ${duration}s"
        return 0
    else
        log_error "Unit tests failed after ${duration}s"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    
    local start_time=$(date +%s)
    local success=true
    
    # Go integration tests
    log_verbose "Running Go integration tests..."
    if ! go test -v -tags=integration ./tests/integration/...; then
        success=false
    fi
    
    # API integration tests
    log_verbose "Running API integration tests..."
    cd tests/integration
    if ! npm test; then
        success=false
    fi
    cd ../..
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ "$success" == "true" ]]; then
        log_success "Integration tests completed in ${duration}s"
        return 0
    else
        log_error "Integration tests failed after ${duration}s"
        return 1
    fi
}

# Function to run E2E tests
run_e2e_tests() {
    log_info "Running end-to-end tests..."
    
    local start_time=$(date +%s)
    local success=true
    
    cd tests/e2e
    
    # Install Playwright if not already installed
    if [[ ! -d "node_modules" ]]; then
        npm ci
        npx playwright install --with-deps
    fi
    
    # Run E2E tests
    local playwright_args=""
    if [[ "$PARALLEL_TESTS" == "true" ]]; then
        playwright_args="--workers=4"
    fi
    
    if ! npx playwright test $playwright_args; then
        success=false
    fi
    
    # Copy results
    cp -r test-results/* "../../$RESULTS_DIR/" 2>/dev/null || true
    cp -r playwright-report "../../$RESULTS_DIR/" 2>/dev/null || true
    
    cd ../..
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ "$success" == "true" ]]; then
        log_success "E2E tests completed in ${duration}s"
        return 0
    else
        log_error "E2E tests failed after ${duration}s"
        return 1
    fi
}

# Function to run security tests
run_security_tests() {
    log_info "Running security tests..."
    
    local start_time=$(date +%s)
    local success=true
    
    cd tests/security
    
    # Install dependencies if needed
    if [[ ! -d "node_modules" ]]; then
        npm ci
    fi
    
    # Run security tests
    if ! npm test; then
        success=false
    fi
    
    # Run penetration tests
    log_verbose "Running penetration tests..."
    if ! npm run pentest; then
        success=false
    fi
    
    # Copy results
    cp -r *.json "../../$RESULTS_DIR/" 2>/dev/null || true
    
    cd ../..
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ "$success" == "true" ]]; then
        log_success "Security tests completed in ${duration}s"
        return 0
    else
        log_error "Security tests failed after ${duration}s"
        return 1
    fi
}

# Function to run performance tests
run_performance_tests() {
    log_info "Running performance tests..."
    
    local start_time=$(date +%s)
    local success=true
    
    cd tests/performance
    
    # Install dependencies if needed
    if [[ ! -d "node_modules" ]]; then
        npm ci
    fi
    
    # Run load tests
    log_verbose "Running load tests..."
    if ! npm run test:load; then
        success=false
    fi
    
    # Run benchmarks
    log_verbose "Running API benchmarks..."
    if ! npm run benchmark; then
        success=false
    fi
    
    # Copy results
    cp -r *.json "../../$RESULTS_DIR/" 2>/dev/null || true
    
    cd ../..
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ "$success" == "true" ]]; then
        log_success "Performance tests completed in ${duration}s"
        return 0
    else
        log_error "Performance tests failed after ${duration}s"
        return 1
    fi
}

# Function to run chaos tests
run_chaos_tests() {
    log_info "Running chaos engineering tests..."
    
    local start_time=$(date +%s)
    local success=true
    
    cd tests/chaos
    
    # Install dependencies if needed
    if [[ ! -d "node_modules" ]]; then
        npm ci
    fi
    
    # Run chaos tests
    if ! npm test; then
        success=false
    fi
    
    # Copy results
    cp -r *.json "../../$RESULTS_DIR/" 2>/dev/null || true
    
    cd ../..
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [[ "$success" == "true" ]]; then
        log_success "Chaos tests completed in ${duration}s"
        return 0
    else
        log_error "Chaos tests failed after ${duration}s"
        return 1
    fi
}

# Function to run quality gates
run_quality_gates() {
    log_info "Running quality gates..."
    
    cd tests/pipeline
    
    # Install dependencies if needed
    if [[ ! -d "node_modules" ]]; then
        npm ci
    fi
    
    # Run quality gate checks
    if node quality-gates.js; then
        log_success "Quality gates passed"
        return 0
    else
        log_error "Quality gates failed"
        return 1
    fi
}

# Function to generate comprehensive report
generate_report() {
    log_info "Generating comprehensive test report..."
    
    local report_data="{
        \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
        \"environment\": \"$TEST_ENVIRONMENT\",
        \"duration\": $(($(date +%s) - $START_TIME)),
        \"results\": {
            \"unit\": $UNIT_RESULT,
            \"integration\": $INTEGRATION_RESULT,
            \"e2e\": $E2E_RESULT,
            \"security\": $SECURITY_RESULT,
            \"performance\": $PERFORMANCE_RESULT,
            \"chaos\": $CHAOS_RESULT,
            \"quality_gates\": $QUALITY_GATES_RESULT
        },
        \"summary\": {
            \"total_suites\": $TOTAL_SUITES,
            \"passed_suites\": $PASSED_SUITES,
            \"failed_suites\": $FAILED_SUITES,
            \"success_rate\": $(echo "scale=2; $PASSED_SUITES * 100 / $TOTAL_SUITES" | bc -l)
        }
    }"
    
    echo "$report_data" | jq '.' > "$REPORT_FILE"
    
    # Generate HTML report
    cat > "$RESULTS_DIR/test-report-$TIMESTAMP.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>AgentScan Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .success { color: #28a745; }
        .failure { color: #dc3545; }
        .suite { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin: 20px 0; }
        .metric { background: #f8f9fa; padding: 15px; border-radius: 5px; text-align: center; }
    </style>
</head>
<body>
    <div class="header">
        <h1>AgentScan Test Report</h1>
        <p><strong>Generated:</strong> $(date)</p>
        <p><strong>Environment:</strong> $TEST_ENVIRONMENT</p>
        <p><strong>Duration:</strong> $(($(date +%s) - $START_TIME))s</p>
    </div>
    
    <div class="metrics">
        <div class="metric">
            <h3>Total Suites</h3>
            <div style="font-size: 2em; font-weight: bold;">$TOTAL_SUITES</div>
        </div>
        <div class="metric">
            <h3>Passed</h3>
            <div style="font-size: 2em; font-weight: bold; color: #28a745;">$PASSED_SUITES</div>
        </div>
        <div class="metric">
            <h3>Failed</h3>
            <div style="font-size: 2em; font-weight: bold; color: #dc3545;">$FAILED_SUITES</div>
        </div>
        <div class="metric">
            <h3>Success Rate</h3>
            <div style="font-size: 2em; font-weight: bold;">$(echo "scale=1; $PASSED_SUITES * 100 / $TOTAL_SUITES" | bc -l)%</div>
        </div>
    </div>
    
    <h2>Test Suite Results</h2>
    
    <div class="suite">
        <h3 class="$([ $UNIT_RESULT -eq 0 ] && echo 'success' || echo 'failure')">Unit Tests</h3>
        <p>Status: $([ $UNIT_RESULT -eq 0 ] && echo '✅ Passed' || echo '❌ Failed')</p>
    </div>
    
    <div class="suite">
        <h3 class="$([ $INTEGRATION_RESULT -eq 0 ] && echo 'success' || echo 'failure')">Integration Tests</h3>
        <p>Status: $([ $INTEGRATION_RESULT -eq 0 ] && echo '✅ Passed' || echo '❌ Failed')</p>
    </div>
    
    <div class="suite">
        <h3 class="$([ $E2E_RESULT -eq 0 ] && echo 'success' || echo 'failure')">End-to-End Tests</h3>
        <p>Status: $([ $E2E_RESULT -eq 0 ] && echo '✅ Passed' || echo '❌ Failed')</p>
    </div>
    
    <div class="suite">
        <h3 class="$([ $SECURITY_RESULT -eq 0 ] && echo 'success' || echo 'failure')">Security Tests</h3>
        <p>Status: $([ $SECURITY_RESULT -eq 0 ] && echo '✅ Passed' || echo '❌ Failed')</p>
    </div>
    
    <div class="suite">
        <h3 class="$([ $PERFORMANCE_RESULT -eq 0 ] && echo 'success' || echo 'failure')">Performance Tests</h3>
        <p>Status: $([ $PERFORMANCE_RESULT -eq 0 ] && echo '✅ Passed' || echo '❌ Failed')</p>
    </div>
    
    <div class="suite">
        <h3 class="$([ $CHAOS_RESULT -eq 0 ] && echo 'success' || echo 'failure')">Chaos Tests</h3>
        <p>Status: $([ $CHAOS_RESULT -eq 0 ] && echo '✅ Passed' || echo '❌ Failed')</p>
    </div>
    
    <div class="suite">
        <h3 class="$([ $QUALITY_GATES_RESULT -eq 0 ] && echo 'success' || echo 'failure')">Quality Gates</h3>
        <p>Status: $([ $QUALITY_GATES_RESULT -eq 0 ] && echo '✅ Passed' || echo '❌ Failed')</p>
    </div>
</body>
</html>
EOF

    log_success "Test report generated: $REPORT_FILE"
    log_info "HTML report: $RESULTS_DIR/test-report-$TIMESTAMP.html"
}

# Parse command line arguments
TEST_SUITES=()
while [[ $# -gt 0 ]]; do
    case $1 in
        --parallel)
            PARALLEL_TESTS=true
            shift
            ;;
        --skip-setup)
            SKIP_SETUP=true
            shift
            ;;
        --skip-teardown)
            SKIP_TEARDOWN=true
            shift
            ;;
        --environment)
            TEST_ENVIRONMENT="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        unit|integration|e2e|security|performance|chaos|all)
            TEST_SUITES+=("$1")
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Default to all tests if none specified
if [[ ${#TEST_SUITES[@]} -eq 0 ]]; then
    TEST_SUITES=("all")
fi

# Expand "all" to individual test suites
if [[ " ${TEST_SUITES[*]} " =~ " all " ]]; then
    TEST_SUITES=("unit" "integration" "e2e" "security" "performance" "chaos")
fi

# Main execution
START_TIME=$(date +%s)
TOTAL_SUITES=${#TEST_SUITES[@]}
PASSED_SUITES=0
FAILED_SUITES=0

# Initialize result variables
UNIT_RESULT=0
INTEGRATION_RESULT=0
E2E_RESULT=0
SECURITY_RESULT=0
PERFORMANCE_RESULT=0
CHAOS_RESULT=0
QUALITY_GATES_RESULT=0

log_info "Starting comprehensive test suite..."
log_info "Environment: $TEST_ENVIRONMENT"
log_info "Test suites: ${TEST_SUITES[*]}"
log_info "Parallel execution: $PARALLEL_TESTS"

# Setup test environment
setup_test_environment

# Run test suites
for suite in "${TEST_SUITES[@]}"; do
    case $suite in
        unit)
            if run_unit_tests; then
                UNIT_RESULT=0
                ((PASSED_SUITES++))
            else
                UNIT_RESULT=1
                ((FAILED_SUITES++))
            fi
            ;;
        integration)
            if run_integration_tests; then
                INTEGRATION_RESULT=0
                ((PASSED_SUITES++))
            else
                INTEGRATION_RESULT=1
                ((FAILED_SUITES++))
            fi
            ;;
        e2e)
            if run_e2e_tests; then
                E2E_RESULT=0
                ((PASSED_SUITES++))
            else
                E2E_RESULT=1
                ((FAILED_SUITES++))
            fi
            ;;
        security)
            if run_security_tests; then
                SECURITY_RESULT=0
                ((PASSED_SUITES++))
            else
                SECURITY_RESULT=1
                ((FAILED_SUITES++))
            fi
            ;;
        performance)
            if run_performance_tests; then
                PERFORMANCE_RESULT=0
                ((PASSED_SUITES++))
            else
                PERFORMANCE_RESULT=1
                ((FAILED_SUITES++))
            fi
            ;;
        chaos)
            if run_chaos_tests; then
                CHAOS_RESULT=0
                ((PASSED_SUITES++))
            else
                CHAOS_RESULT=1
                ((FAILED_SUITES++))
            fi
            ;;
    esac
done

# Run quality gates
if run_quality_gates; then
    QUALITY_GATES_RESULT=0
else
    QUALITY_GATES_RESULT=1
fi

# Generate comprehensive report
generate_report

# Teardown test environment
teardown_test_environment

# Final summary
TOTAL_DURATION=$(($(date +%s) - $START_TIME))
SUCCESS_RATE=$(echo "scale=1; $PASSED_SUITES * 100 / $TOTAL_SUITES" | bc -l)

echo ""
echo "=========================================="
echo "🧪 COMPREHENSIVE TEST SUMMARY"
echo "=========================================="
echo "Total Duration: ${TOTAL_DURATION}s"
echo "Test Suites: $TOTAL_SUITES"
echo "Passed: $PASSED_SUITES"
echo "Failed: $FAILED_SUITES"
echo "Success Rate: ${SUCCESS_RATE}%"
echo "Quality Gates: $([ $QUALITY_GATES_RESULT -eq 0 ] && echo '✅ Passed' || echo '❌ Failed')"
echo "=========================================="

# Exit with appropriate code
if [[ $FAILED_SUITES -eq 0 && $QUALITY_GATES_RESULT -eq 0 ]]; then
    log_success "All tests passed! 🎉"
    exit 0
else
    log_error "Some tests failed. Check the report for details."
    exit 1
fi