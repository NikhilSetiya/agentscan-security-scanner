#!/bin/bash

# Production Smoke Test Runner
# This script runs comprehensive smoke tests against the production environment

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
API_URL="${API_URL:-https://agentscan-prod.fly.dev}"
FRONTEND_URL="${FRONTEND_URL:-https://agentscan.vercel.app}"
SMOKE_TEST_EMAIL="${SMOKE_TEST_EMAIL:-smoketest@example.com}"
SMOKE_TEST_PASSWORD="${SMOKE_TEST_PASSWORD:-SmokeTest123!}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
TEST_RESULTS=()

# Function to run a test category
run_test_category() {
    local category="$1"
    local description="$2"
    
    log_info "Running $description..."
    
    if cd "$PROJECT_ROOT" && go test -v -timeout=10m -tags=smoke "./tests/smoke" -run="Test.*$category"; then
        log_success "✓ $description passed"
        ((TESTS_PASSED++))
        TEST_RESULTS+=("✓ $description")
        return 0
    else
        log_error "✗ $description failed"
        ((TESTS_FAILED++))
        TEST_RESULTS+=("✗ $description")
        return 1
    fi
}

# Function to run all smoke tests
run_all_smoke_tests() {
    log_info "Running all production smoke tests..."
    
    export SMOKE_TEST_ENV=production
    export API_URL="$API_URL"
    export FRONTEND_URL="$FRONTEND_URL"
    export SMOKE_TEST_EMAIL="$SMOKE_TEST_EMAIL"
    export SMOKE_TEST_PASSWORD="$SMOKE_TEST_PASSWORD"
    
    if cd "$PROJECT_ROOT" && go test -v -timeout=15m -tags=smoke "./tests/smoke"; then
        log_success "✓ All smoke tests passed"
        return 0
    else
        log_error "✗ Some smoke tests failed"
        return 1
    fi
}

# Function to run infrastructure verification
run_infrastructure_tests() {
    log_info "Running infrastructure verification tests..."
    
    # Run the verification script
    if "$SCRIPT_DIR/verify-production.sh" --api-url "$API_URL" --frontend-url "$FRONTEND_URL"; then
        log_success "✓ Infrastructure verification passed"
        ((TESTS_PASSED++))
        TEST_RESULTS+=("✓ Infrastructure Verification")
        return 0
    else
        log_error "✗ Infrastructure verification failed"
        ((TESTS_FAILED++))
        TEST_RESULTS+=("✗ Infrastructure Verification")
        return 1
    fi
}

# Function to test critical user journeys
test_critical_user_journeys() {
    log_info "Testing critical user journeys..."
    
    local temp_dir=$(mktemp -d)
    local test_script="$temp_dir/user_journey_test.sh"
    
    # Create user journey test script
    cat > "$test_script" << 'EOF'
#!/bin/bash
set -euo pipefail

API_URL="$1"
FRONTEND_URL="$2"

# Test 1: User Registration and Login Journey
echo "Testing user registration and login journey..."

# Generate unique test user
TIMESTAMP=$(date +%s)
TEST_EMAIL="journey-test-$TIMESTAMP@example.com"
TEST_PASSWORD="JourneyTest123!"

# Register user
REGISTER_RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/register_response \
    -X POST \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\",\"name\":\"Journey Test User\"}" \
    "$API_URL/auth/register")

REGISTER_CODE="${REGISTER_RESPONSE: -3}"
if [ "$REGISTER_CODE" != "201" ]; then
    echo "Registration failed with code $REGISTER_CODE"
    exit 1
fi

# Login user
LOGIN_RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/login_response \
    -X POST \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" \
    "$API_URL/auth/login")

LOGIN_CODE="${LOGIN_RESPONSE: -3}"
if [ "$LOGIN_CODE" != "200" ]; then
    echo "Login failed with code $LOGIN_CODE"
    exit 1
fi

# Extract token
TOKEN=$(jq -r '.token' /tmp/login_response)
if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "Failed to extract token from login response"
    exit 1
fi

# Test 2: Scan Creation and Management Journey
echo "Testing scan creation and management journey..."

# Create scan
CREATE_SCAN_RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/create_scan_response \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"name":"Journey Test Scan","type":"vulnerability","targets":["https://httpbin.org"],"config":{"timeout":300}}' \
    "$API_URL/api/v1/scans")

CREATE_SCAN_CODE="${CREATE_SCAN_RESPONSE: -3}"
if [ "$CREATE_SCAN_CODE" != "201" ]; then
    echo "Scan creation failed with code $CREATE_SCAN_CODE"
    exit 1
fi

# Extract scan ID
SCAN_ID=$(jq -r '.scan_id' /tmp/create_scan_response)
if [ "$SCAN_ID" = "null" ] || [ -z "$SCAN_ID" ]; then
    echo "Failed to extract scan ID from create response"
    exit 1
fi

# Get scan details
GET_SCAN_RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/get_scan_response \
    -H "Authorization: Bearer $TOKEN" \
    "$API_URL/api/v1/scans/$SCAN_ID")

GET_SCAN_CODE="${GET_SCAN_RESPONSE: -3}"
if [ "$GET_SCAN_CODE" != "200" ]; then
    echo "Get scan failed with code $GET_SCAN_CODE"
    exit 1
fi

# List scans
LIST_SCANS_RESPONSE=$(curl -s -w "%{http_code}" -o /tmp/list_scans_response \
    -H "Authorization: Bearer $TOKEN" \
    "$API_URL/api/v1/scans")

LIST_SCANS_CODE="${LIST_SCANS_RESPONSE: -3}"
if [ "$LIST_SCANS_CODE" != "200" ]; then
    echo "List scans failed with code $LIST_SCANS_CODE"
    exit 1
fi

# Clean up - delete test scan
DELETE_SCAN_RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null \
    -X DELETE \
    -H "Authorization: Bearer $TOKEN" \
    "$API_URL/api/v1/scans/$SCAN_ID")

DELETE_SCAN_CODE="${DELETE_SCAN_RESPONSE: -3}"
if [ "$DELETE_SCAN_CODE" != "200" ]; then
    echo "Delete scan failed with code $DELETE_SCAN_CODE"
    # Don't exit here, it's cleanup
fi

# Test 3: Frontend Integration
echo "Testing frontend integration..."

# Check if frontend loads
FRONTEND_RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null "$FRONTEND_URL")
FRONTEND_CODE="${FRONTEND_RESPONSE: -3}"
if [ "$FRONTEND_CODE" != "200" ]; then
    echo "Frontend failed to load with code $FRONTEND_CODE"
    exit 1
fi

echo "All user journey tests passed!"
EOF

    chmod +x "$test_script"
    
    if "$test_script" "$API_URL" "$FRONTEND_URL"; then
        log_success "✓ Critical user journeys passed"
        ((TESTS_PASSED++))
        TEST_RESULTS+=("✓ Critical User Journeys")
        rm -rf "$temp_dir"
        return 0
    else
        log_error "✗ Critical user journeys failed"
        ((TESTS_FAILED++))
        TEST_RESULTS+=("✗ Critical User Journeys")
        rm -rf "$temp_dir"
        return 1
    fi
}

# Function to test performance benchmarks
test_performance_benchmarks() {
    log_info "Testing performance benchmarks..."
    
    local temp_script=$(mktemp)
    
    cat > "$temp_script" << EOF
#!/bin/bash
set -euo pipefail

API_URL="$API_URL"

# Test response times for critical endpoints
ENDPOINTS=("/health" "/ready" "/metrics")
MAX_RESPONSE_TIME=5000  # 5 seconds in milliseconds

for endpoint in "\${ENDPOINTS[@]}"; do
    echo "Testing \$endpoint response time..."
    
    # Measure response time using curl
    RESPONSE_TIME=\$(curl -w "%{time_total}" -s -o /dev/null "\$API_URL\$endpoint" | awk '{print \$1 * 1000}')
    
    if (( \$(echo "\$RESPONSE_TIME > \$MAX_RESPONSE_TIME" | bc -l) )); then
        echo "Response time for \$endpoint too slow: \${RESPONSE_TIME}ms"
        exit 1
    else
        echo "Response time for \$endpoint: \${RESPONSE_TIME}ms ✓"
    fi
done

# Test concurrent requests
echo "Testing concurrent request handling..."
CONCURRENT_REQUESTS=10
TOTAL_REQUESTS=50

# Use GNU parallel if available, otherwise use background processes
if command -v parallel >/dev/null 2>&1; then
    seq 1 \$TOTAL_REQUESTS | parallel -j\$CONCURRENT_REQUESTS "curl -s -o /dev/null -w '%{http_code}' '\$API_URL/health'" > /tmp/concurrent_results
else
    # Fallback to background processes
    for i in \$(seq 1 \$TOTAL_REQUESTS); do
        (curl -s -o /dev/null -w '%{http_code}' "\$API_URL/health" >> /tmp/concurrent_results) &
        if (( i % CONCURRENT_REQUESTS == 0 )); then
            wait
        fi
    done
    wait
fi

# Check results
SUCCESS_COUNT=\$(grep -c "200" /tmp/concurrent_results || true)
SUCCESS_RATE=\$(echo "scale=2; \$SUCCESS_COUNT / \$TOTAL_REQUESTS * 100" | bc)

if (( \$(echo "\$SUCCESS_RATE < 95" | bc -l) )); then
    echo "Concurrent request success rate too low: \${SUCCESS_RATE}%"
    exit 1
else
    echo "Concurrent request success rate: \${SUCCESS_RATE}% ✓"
fi

rm -f /tmp/concurrent_results
echo "Performance benchmarks passed!"
EOF

    chmod +x "$temp_script"
    
    if bash "$temp_script"; then
        log_success "✓ Performance benchmarks passed"
        ((TESTS_PASSED++))
        TEST_RESULTS+=("✓ Performance Benchmarks")
        rm -f "$temp_script"
        return 0
    else
        log_error "✗ Performance benchmarks failed"
        ((TESTS_FAILED++))
        TEST_RESULTS+=("✗ Performance Benchmarks")
        rm -f "$temp_script"
        return 1
    fi
}

# Function to generate smoke test report
generate_smoke_test_report() {
    local report_file="smoke_test_report_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$report_file" << EOF
# Production Smoke Test Report

**Date:** $(date)
**API URL:** $API_URL
**Frontend URL:** $FRONTEND_URL

## Test Summary

- **Total Tests:** $((TESTS_PASSED + TESTS_FAILED))
- **Passed:** $TESTS_PASSED
- **Failed:** $TESTS_FAILED
- **Success Rate:** $(echo "scale=2; $TESTS_PASSED / ($TESTS_PASSED + $TESTS_FAILED) * 100" | bc)%

## Test Results

EOF

    for result in "${TEST_RESULTS[@]}"; do
        echo "- $result" >> "$report_file"
    done

    cat >> "$report_file" << EOF

## Environment Information

- **Go Version:** $(go version)
- **Test Environment:** production
- **Timestamp:** $(date -u +"%Y-%m-%dT%H:%M:%SZ")

## Recommendations

EOF

    if [ $TESTS_FAILED -gt 0 ]; then
        cat >> "$report_file" << EOF
⚠️ **Action Required:** Some tests failed. Please review the failed tests and address any issues before considering the deployment fully successful.

### Failed Tests
EOF
        for result in "${TEST_RESULTS[@]}"; do
            if [[ $result == ✗* ]]; then
                echo "- $result" >> "$report_file"
            fi
        done
    else
        cat >> "$report_file" << EOF
✅ **All tests passed!** The production deployment appears to be working correctly.
EOF
    fi

    log_info "Smoke test report generated: $report_file"
}

# Main function
main() {
    local run_all=true
    local run_infrastructure=false
    local run_journeys=false
    local run_performance=false
    local run_go_tests=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --infrastructure)
                run_all=false
                run_infrastructure=true
                shift
                ;;
            --journeys)
                run_all=false
                run_journeys=true
                shift
                ;;
            --performance)
                run_all=false
                run_performance=true
                shift
                ;;
            --go-tests)
                run_all=false
                run_go_tests=true
                shift
                ;;
            --api-url)
                API_URL="$2"
                shift 2
                ;;
            --frontend-url)
                FRONTEND_URL="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    log_info "Starting production smoke tests..."
    log_info "API URL: $API_URL"
    log_info "Frontend URL: $FRONTEND_URL"
    echo
    
    # Check dependencies
    if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    if ! command -v bc >/dev/null 2>&1; then
        log_error "bc is required but not installed"
        exit 1
    fi
    
    # Run tests based on options
    if [ "$run_all" = true ]; then
        run_infrastructure_tests
        test_critical_user_journeys
        test_performance_benchmarks
        run_all_smoke_tests
    else
        if [ "$run_infrastructure" = true ]; then
            run_infrastructure_tests
        fi
        
        if [ "$run_journeys" = true ]; then
            test_critical_user_journeys
        fi
        
        if [ "$run_performance" = true ]; then
            test_performance_benchmarks
        fi
        
        if [ "$run_go_tests" = true ]; then
            run_all_smoke_tests
        fi
    fi
    
    # Generate report
    generate_smoke_test_report
    
    # Print summary
    echo
    log_info "Smoke Test Summary:"
    log_success "Tests Passed: $TESTS_PASSED"
    
    if [ $TESTS_FAILED -gt 0 ]; then
        log_error "Tests Failed: $TESTS_FAILED"
        echo
        log_error "Production smoke tests FAILED"
        exit 1
    else
        echo
        log_success "All smoke tests PASSED! Production deployment verified ✅"
        exit 0
    fi
}

# Show help
show_help() {
    cat << EOF
Production Smoke Test Runner

This script runs comprehensive smoke tests against the production environment.

Usage: $0 [OPTIONS]

Options:
    --infrastructure    Run only infrastructure tests
    --journeys         Run only user journey tests
    --performance      Run only performance tests
    --go-tests         Run only Go-based smoke tests
    --api-url URL      API base URL (default: https://agentscan-prod.fly.dev)
    --frontend-url URL Frontend URL (default: https://agentscan.vercel.app)
    --help             Show this help message

Environment Variables:
    API_URL                API base URL
    FRONTEND_URL           Frontend URL
    SMOKE_TEST_EMAIL       Test user email
    SMOKE_TEST_PASSWORD    Test user password

Examples:
    $0                                    # Run all smoke tests
    $0 --infrastructure                   # Run only infrastructure tests
    $0 --api-url https://api.example.com  # Custom API URL

EOF
}

# Run main function
main "$@"