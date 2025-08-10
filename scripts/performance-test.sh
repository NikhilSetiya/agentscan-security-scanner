#!/bin/bash

# AgentScan Performance Testing Script
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
CONCURRENT_REQUESTS="${CONCURRENT_REQUESTS:-10}"
TEST_DURATION="${TEST_DURATION:-60}"
TEST_REPO="${TEST_REPO:-https://github.com/OWASP/WebGoat}"

echo -e "${GREEN}Starting AgentScan Performance Tests${NC}"
echo "API URL: $API_URL"
echo "Concurrent Requests: $CONCURRENT_REQUESTS"
echo "Test Duration: ${TEST_DURATION}s"
echo "Test Repository: $TEST_REPO"
echo ""

# Function to check if service is ready
wait_for_service() {
    local url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=1

    echo -e "${YELLOW}Waiting for $service_name to be ready...${NC}"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}$service_name is ready!${NC}"
            return 0
        fi
        
        echo "Attempt $attempt/$max_attempts failed, waiting 2 seconds..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo -e "${RED}$service_name failed to become ready after $max_attempts attempts${NC}"
    return 1
}

# Function to run load test
run_load_test() {
    local test_name=$1
    local concurrent=$2
    local duration=$3
    
    echo -e "${YELLOW}Running $test_name (${concurrent} concurrent, ${duration}s)...${NC}"
    
    # Create test payload
    local payload=$(cat <<EOF
{
    "concurrency": $concurrent,
    "test_repository": "$TEST_REPO",
    "incremental_scans": false,
    "wait_for_completion": false
}
EOF
)
    
    # Submit load test
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$API_URL/api/v1/benchmark/load-test")
    
    local test_id=$(echo "$response" | jq -r '.test_id // empty')
    
    if [ -z "$test_id" ]; then
        echo -e "${RED}Failed to start load test: $response${NC}"
        return 1
    fi
    
    echo "Test ID: $test_id"
    
    # Wait for test completion
    local status="running"
    while [ "$status" = "running" ]; do
        sleep 5
        local status_response=$(curl -s "$API_URL/api/v1/benchmark/load-test/$test_id")
        status=$(echo "$status_response" | jq -r '.status // "unknown"')
        echo "Status: $status"
    done
    
    # Get final results
    local results=$(curl -s "$API_URL/api/v1/benchmark/load-test/$test_id")
    
    echo -e "${GREEN}$test_name Results:${NC}"
    echo "$results" | jq '.metrics'
    echo ""
}

# Function to run benchmark test
run_benchmark_test() {
    local operation=$1
    local iterations=$2
    
    echo -e "${YELLOW}Running $operation benchmark ($iterations iterations)...${NC}"
    
    local payload=$(cat <<EOF
{
    "operation": "$operation",
    "iterations": $iterations,
    "test_repository": "$TEST_REPO"
}
EOF
)
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$API_URL/api/v1/benchmark/benchmark")
    
    local test_id=$(echo "$response" | jq -r '.test_id // empty')
    
    if [ -z "$test_id" ]; then
        echo -e "${RED}Failed to start benchmark: $response${NC}"
        return 1
    fi
    
    echo "Benchmark ID: $test_id"
    
    # Wait for benchmark completion
    local status="running"
    while [ "$status" = "running" ]; do
        sleep 2
        local status_response=$(curl -s "$API_URL/api/v1/benchmark/benchmark/$test_id")
        status=$(echo "$status_response" | jq -r '.status // "unknown"')
        echo "Status: $status"
    done
    
    # Get final results
    local results=$(curl -s "$API_URL/api/v1/benchmark/benchmark/$test_id")
    
    echo -e "${GREEN}$operation Benchmark Results:${NC}"
    echo "$results" | jq '.metrics'
    echo ""
}

# Function to test database performance
test_database_performance() {
    echo -e "${YELLOW}Testing Database Performance...${NC}"
    
    # Test connection pool
    local db_stats=$(curl -s "$API_URL/api/v1/system/metrics" | jq '.database_connections')
    echo "Database Connections: $db_stats"
    
    # Test query performance
    local start_time=$(date +%s%N)
    curl -s "$API_URL/api/v1/scans?limit=100" > /dev/null
    local end_time=$(date +%s%N)
    local query_time=$(( (end_time - start_time) / 1000000 ))
    
    echo "Query Response Time: ${query_time}ms"
    echo ""
}

# Function to test cache performance
test_cache_performance() {
    echo -e "${YELLOW}Testing Cache Performance...${NC}"
    
    # Test Redis connection
    local redis_stats=$(curl -s "$API_URL/api/v1/system/metrics" | jq '.redis_connections')
    echo "Redis Connections: $redis_stats"
    
    # Test cache hit/miss ratio
    local cache_stats=$(curl -s "$API_URL/api/v1/system/cache-stats")
    echo "Cache Stats: $cache_stats"
    echo ""
}

# Function to generate performance report
generate_report() {
    echo -e "${GREEN}Generating Performance Report...${NC}"
    
    local report_file="performance-report-$(date +%Y%m%d-%H%M%S).json"
    
    # Collect system metrics
    local system_metrics=$(curl -s "$API_URL/api/v1/system/metrics")
    
    # Create report
    cat > "$report_file" <<EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "test_configuration": {
        "api_url": "$API_URL",
        "concurrent_requests": $CONCURRENT_REQUESTS,
        "test_duration": $TEST_DURATION,
        "test_repository": "$TEST_REPO"
    },
    "system_metrics": $system_metrics
}
EOF
    
    echo "Performance report saved to: $report_file"
}

# Main execution
main() {
    # Check dependencies
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}curl is required but not installed${NC}"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        echo -e "${RED}jq is required but not installed${NC}"
        exit 1
    fi
    
    # Wait for services to be ready
    wait_for_service "$API_URL/health" "API Server"
    
    # Run performance tests
    echo -e "${GREEN}Starting Performance Test Suite${NC}"
    echo "=================================="
    
    # Database performance tests
    test_database_performance
    
    # Cache performance tests
    test_cache_performance
    
    # Load tests
    run_load_test "Light Load Test" 5 30
    run_load_test "Medium Load Test" $CONCURRENT_REQUESTS $TEST_DURATION
    run_load_test "Heavy Load Test" $((CONCURRENT_REQUESTS * 2)) $((TEST_DURATION / 2))
    
    # Benchmark tests
    run_benchmark_test "scan" 10
    run_benchmark_test "query" 50
    run_benchmark_test "cache" 100
    
    # Generate final report
    generate_report
    
    echo -e "${GREEN}Performance testing completed!${NC}"
}

# Run main function
main "$@"