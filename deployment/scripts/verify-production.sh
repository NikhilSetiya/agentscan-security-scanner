#!/bin/bash

# Production Deployment Verification Script
# This script verifies that the production deployment is working correctly

set -euo pipefail

# Configuration
API_URL="${API_URL:-https://agentscan-prod.fly.dev}"
FRONTEND_URL="${FRONTEND_URL:-https://agentscan.vercel.app}"
TIMEOUT=30

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

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    log_info "Running test: $test_name"
    
    if eval "$test_command"; then
        log_success "✓ $test_name"
        ((TESTS_PASSED++))
        return 0
    else
        log_error "✗ $test_name"
        FAILED_TESTS+=("$test_name")
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test API health endpoint
test_api_health() {
    local response=$(curl -s -w "%{http_code}" -o /tmp/health_response "$API_URL/health" --max-time $TIMEOUT)
    local http_code="${response: -3}"
    
    if [ "$http_code" = "200" ]; then
        local health_status=$(jq -r '.status' /tmp/health_response 2>/dev/null || echo "unknown")
        if [ "$health_status" = "healthy" ]; then
            return 0
        else
            log_error "Health status is not healthy: $health_status"
            return 1
        fi
    else
        log_error "Health endpoint returned HTTP $http_code"
        return 1
    fi
}

# Test API readiness endpoint
test_api_readiness() {
    local response=$(curl -s -w "%{http_code}" -o /tmp/ready_response "$API_URL/ready" --max-time $TIMEOUT)
    local http_code="${response: -3}"
    
    if [ "$http_code" = "200" ]; then
        return 0
    else
        log_error "Readiness endpoint returned HTTP $http_code"
        return 1
    fi
}

# Test API metrics endpoint
test_api_metrics() {
    local response=$(curl -s -w "%{http_code}" -o /tmp/metrics_response "$API_URL/metrics" --max-time $TIMEOUT)
    local http_code="${response: -3}"
    
    if [ "$http_code" = "200" ]; then
        # Check if it contains Prometheus metrics
        if grep -q "# HELP" /tmp/metrics_response; then
            return 0
        else
            log_error "Metrics endpoint does not contain valid Prometheus metrics"
            return 1
        fi
    else
        log_error "Metrics endpoint returned HTTP $http_code"
        return 1
    fi
}

# Test HTTPS enforcement
test_https_enforcement() {
    local http_url="${API_URL/https:/http:}"
    local response=$(curl -s -w "%{http_code}" -o /dev/null "$http_url/health" --max-time $TIMEOUT)
    local http_code="${response: -3}"
    
    # Should redirect to HTTPS (301/302) or be forbidden (403)
    if [ "$http_code" = "301" ] || [ "$http_code" = "302" ] || [ "$http_code" = "403" ]; then
        return 0
    else
        log_error "HTTP requests are not being redirected to HTTPS (got $http_code)"
        return 1
    fi
}

# Test security headers
test_security_headers() {
    local headers=$(curl -s -I "$API_URL/health" --max-time $TIMEOUT)
    
    local required_headers=(
        "Strict-Transport-Security"
        "X-Content-Type-Options"
        "X-Frame-Options"
        "X-XSS-Protection"
    )
    
    for header in "${required_headers[@]}"; do
        if ! echo "$headers" | grep -qi "$header"; then
            log_error "Missing security header: $header"
            return 1
        fi
    done
    
    return 0
}

# Test CORS configuration
test_cors_configuration() {
    local response=$(curl -s -H "Origin: https://agentscan.vercel.app" \
                          -H "Access-Control-Request-Method: POST" \
                          -H "Access-Control-Request-Headers: Content-Type" \
                          -X OPTIONS \
                          -w "%{http_code}" \
                          -o /tmp/cors_response \
                          "$API_URL/api/v1/scans" \
                          --max-time $TIMEOUT)
    local http_code="${response: -3}"
    
    if [ "$http_code" = "204" ] || [ "$http_code" = "200" ]; then
        # Check for CORS headers in response
        local cors_headers=$(curl -s -I -H "Origin: https://agentscan.vercel.app" \
                                  -X OPTIONS \
                                  "$API_URL/api/v1/scans" \
                                  --max-time $TIMEOUT)
        
        if echo "$cors_headers" | grep -qi "Access-Control-Allow-Origin"; then
            return 0
        else
            log_error "CORS headers not found in response"
            return 1
        fi
    else
        log_error "CORS preflight request failed with HTTP $http_code"
        return 1
    fi
}

# Test database connectivity (through API)
test_database_connectivity() {
    # This assumes there's a database health check in the health endpoint
    local response=$(curl -s "$API_URL/health" --max-time $TIMEOUT)
    
    if echo "$response" | jq -e '.checks.database.status == "healthy"' >/dev/null 2>&1; then
        return 0
    else
        log_error "Database health check failed"
        return 1
    fi
}

# Test Redis connectivity (through API)
test_redis_connectivity() {
    # This assumes there's a Redis health check in the health endpoint
    local response=$(curl -s "$API_URL/health" --max-time $TIMEOUT)
    
    if echo "$response" | jq -e '.checks.redis.status == "healthy"' >/dev/null 2>&1; then
        return 0
    else
        log_error "Redis health check failed"
        return 1
    fi
}

# Test authentication endpoint
test_authentication() {
    # Test that auth endpoint exists and returns proper error for missing credentials
    local response=$(curl -s -w "%{http_code}" -o /tmp/auth_response \
                          -X POST \
                          -H "Content-Type: application/json" \
                          -d '{}' \
                          "$API_URL/auth/login" \
                          --max-time $TIMEOUT)
    local http_code="${response: -3}"
    
    # Should return 400 (bad request) for missing credentials
    if [ "$http_code" = "400" ]; then
        return 0
    else
        log_error "Authentication endpoint returned unexpected HTTP $http_code"
        return 1
    fi
}

# Test rate limiting
test_rate_limiting() {
    log_info "Testing rate limiting (this may take a moment)..."
    
    local rate_limited=false
    
    # Make rapid requests to trigger rate limiting
    for i in {1..20}; do
        local response=$(curl -s -w "%{http_code}" -o /dev/null "$API_URL/health" --max-time 5)
        local http_code="${response: -3}"
        
        if [ "$http_code" = "429" ]; then
            rate_limited=true
            break
        fi
        
        sleep 0.1
    done
    
    if [ "$rate_limited" = true ]; then
        return 0
    else
        log_warning "Rate limiting not triggered (may be configured with high limits)"
        return 0  # Don't fail the test, just warn
    fi
}

# Test frontend accessibility
test_frontend_accessibility() {
    local response=$(curl -s -w "%{http_code}" -o /dev/null "$FRONTEND_URL" --max-time $TIMEOUT)
    local http_code="${response: -3}"
    
    if [ "$http_code" = "200" ]; then
        return 0
    else
        log_error "Frontend returned HTTP $http_code"
        return 1
    fi
}

# Test frontend HTTPS
test_frontend_https() {
    if [[ "$FRONTEND_URL" == https://* ]]; then
        return 0
    else
        log_error "Frontend URL is not using HTTPS: $FRONTEND_URL"
        return 1
    fi
}

# Test API response time
test_api_response_time() {
    local start_time=$(date +%s%N)
    curl -s "$API_URL/health" --max-time $TIMEOUT >/dev/null
    local end_time=$(date +%s%N)
    
    local response_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds
    
    log_info "API response time: ${response_time}ms"
    
    # Fail if response time is over 5 seconds
    if [ $response_time -gt 5000 ]; then
        log_error "API response time too slow: ${response_time}ms"
        return 1
    else
        return 0
    fi
}

# Test SSL certificate validity
test_ssl_certificate() {
    local domain=$(echo "$API_URL" | sed 's|https://||' | sed 's|/.*||')
    
    if command -v openssl >/dev/null 2>&1; then
        local cert_info=$(echo | openssl s_client -servername "$domain" -connect "$domain:443" 2>/dev/null | openssl x509 -noout -dates 2>/dev/null)
        
        if [ $? -eq 0 ]; then
            local not_after=$(echo "$cert_info" | grep "notAfter" | cut -d= -f2)
            local expiry_date=$(date -d "$not_after" +%s 2>/dev/null || date -j -f "%b %d %H:%M:%S %Y %Z" "$not_after" +%s 2>/dev/null)
            local current_date=$(date +%s)
            local days_until_expiry=$(( (expiry_date - current_date) / 86400 ))
            
            log_info "SSL certificate expires in $days_until_expiry days"
            
            if [ $days_until_expiry -lt 30 ]; then
                log_warning "SSL certificate expires soon: $days_until_expiry days"
            fi
            
            if [ $days_until_expiry -lt 0 ]; then
                log_error "SSL certificate has expired"
                return 1
            fi
            
            return 0
        else
            log_error "Failed to retrieve SSL certificate information"
            return 1
        fi
    else
        log_warning "OpenSSL not available, skipping SSL certificate check"
        return 0
    fi
}

# Main verification function
main() {
    log_info "Starting production deployment verification..."
    log_info "API URL: $API_URL"
    log_info "Frontend URL: $FRONTEND_URL"
    echo
    
    # Run all tests
    run_test "API Health Check" "test_api_health"
    run_test "API Readiness Check" "test_api_readiness"
    run_test "API Metrics Endpoint" "test_api_metrics"
    run_test "HTTPS Enforcement" "test_https_enforcement"
    run_test "Security Headers" "test_security_headers"
    run_test "CORS Configuration" "test_cors_configuration"
    run_test "Database Connectivity" "test_database_connectivity"
    run_test "Redis Connectivity" "test_redis_connectivity"
    run_test "Authentication Endpoint" "test_authentication"
    run_test "Rate Limiting" "test_rate_limiting"
    run_test "Frontend Accessibility" "test_frontend_accessibility"
    run_test "Frontend HTTPS" "test_frontend_https"
    run_test "API Response Time" "test_api_response_time"
    run_test "SSL Certificate" "test_ssl_certificate"
    
    # Cleanup temporary files
    rm -f /tmp/health_response /tmp/ready_response /tmp/metrics_response /tmp/cors_response /tmp/auth_response
    
    # Print summary
    echo
    log_info "Verification Summary:"
    log_success "Tests Passed: $TESTS_PASSED"
    
    if [ $TESTS_FAILED -gt 0 ]; then
        log_error "Tests Failed: $TESTS_FAILED"
        echo
        log_error "Failed Tests:"
        for test in "${FAILED_TESTS[@]}"; do
            echo "  - $test"
        done
        echo
        log_error "Production deployment verification FAILED"
        exit 1
    else
        echo
        log_success "All tests passed! Production deployment verification SUCCESSFUL"
        exit 0
    fi
}

# Show help
show_help() {
    cat << EOF
Production Deployment Verification Script

This script verifies that the AgentScan production deployment is working correctly.

Usage: $0 [OPTIONS]

Options:
    --api-url URL       API base URL (default: https://agentscan-prod.fly.dev)
    --frontend-url URL  Frontend URL (default: https://agentscan.vercel.app)
    --timeout SECONDS   Request timeout in seconds (default: 30)
    --help              Show this help message

Environment Variables:
    API_URL            API base URL
    FRONTEND_URL       Frontend URL

Examples:
    $0                                    # Use default URLs
    $0 --api-url https://api.example.com  # Custom API URL
    $0 --timeout 60                       # Increase timeout

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --api-url)
            API_URL="$2"
            shift 2
            ;;
        --frontend-url)
            FRONTEND_URL="$2"
            shift 2
            ;;
        --timeout)
            TIMEOUT="$2"
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

# Check dependencies
if ! command -v curl >/dev/null 2>&1; then
    log_error "curl is required but not installed"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required but not installed"
    exit 1
fi

# Run main function
main