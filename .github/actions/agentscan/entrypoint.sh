#!/bin/bash

# AgentScan GitHub Action Entrypoint
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
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

# Set default values
REPOSITORY_URL="${REPOSITORY_URL:-https://github.com/${GITHUB_REPOSITORY}}"
BRANCH="${BRANCH:-${GITHUB_REF#refs/heads/}}"
SCAN_TYPE="${SCAN_TYPE:-incremental}"
FAIL_ON_HIGH="${FAIL_ON_HIGH:-true}"
FAIL_ON_MEDIUM="${FAIL_ON_MEDIUM:-false}"
FAIL_ON_LOW="${FAIL_ON_LOW:-false}"
COMMENT_PR="${COMMENT_PR:-true}"
UPDATE_STATUS="${UPDATE_STATUS:-true}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-json}"
OUTPUT_FILE="${OUTPUT_FILE:-agentscan-results.json}"
CONFIG_FILE="${CONFIG_FILE:-.agentscan/config.yml}"
TIMEOUT="${TIMEOUT:-15}"
AGENTS="${AGENTS:-sast,sca,secrets}"
SEVERITY_THRESHOLD="${SEVERITY_THRESHOLD:-medium}"
WORKING_DIRECTORY="${WORKING_DIRECTORY:-.}"

# Validate required inputs
if [[ -z "${AGENTSCAN_API_KEY:-}" ]]; then
    log_error "AGENTSCAN_API_KEY is required"
    exit 1
fi

if [[ -z "${AGENTSCAN_API_URL:-}" ]]; then
    log_error "AGENTSCAN_API_URL is required"
    exit 1
fi

# Change to working directory
cd "$WORKING_DIRECTORY"

log_info "Starting AgentScan security scan..."
log_info "Repository: $REPOSITORY_URL"
log_info "Branch: $BRANCH"
log_info "Scan type: $SCAN_TYPE"
log_info "Agents: $AGENTS"

# Determine scan type based on GitHub event
if [[ "$GITHUB_EVENT_NAME" == "pull_request" ]]; then
    SCAN_TYPE="pr"
    log_info "Detected pull request event, using PR scan mode"
fi

# Build scan request
SCAN_REQUEST=$(jq -n \
    --arg repo "$REPOSITORY_URL" \
    --arg branch "$BRANCH" \
    --arg scan_type "$SCAN_TYPE" \
    --arg agents "$AGENTS" \
    --arg severity "$SEVERITY_THRESHOLD" \
    --arg include_paths "${INCLUDE_PATHS:-}" \
    --arg exclude_paths "${EXCLUDE_PATHS:-}" \
    --arg github_run_id "$GITHUB_RUN_ID" \
    --arg github_sha "$GITHUB_SHA" \
    '{
        repository_url: $repo,
        branch: $branch,
        scan_type: $scan_type,
        agents_requested: ($agents | split(",")),
        settings: {
            severity_threshold: $severity,
            include_paths: (if $include_paths != "" then ($include_paths | split(",")) else [] end),
            exclude_paths: (if $exclude_paths != "" then ($exclude_paths | split(",")) else [] end)
        },
        metadata: {
            github_run_id: $github_run_id,
            github_sha: $github_sha,
            github_event: env.GITHUB_EVENT_NAME
        }
    }')

# Submit scan request
log_info "Submitting scan request..."
SCAN_RESPONSE=$(curl -s -X POST \
    -H "Authorization: Bearer $AGENTSCAN_API_KEY" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -H "User-Agent: AgentScan-GitHub-Action/1.0" \
    -d "$SCAN_REQUEST" \
    "$AGENTSCAN_API_URL/api/v1/scans" || {
        log_error "Failed to submit scan request"
        exit 1
    })

# Extract scan ID
SCAN_ID=$(echo "$SCAN_RESPONSE" | jq -r '.id // empty')
if [[ -z "$SCAN_ID" ]]; then
    log_error "Failed to get scan ID from response"
    echo "Response: $SCAN_RESPONSE"
    exit 1
fi

log_success "Scan submitted with ID: $SCAN_ID"

# Set output
echo "scan-id=$SCAN_ID" >> "$GITHUB_OUTPUT"

# Update GitHub status if enabled
if [[ "$UPDATE_STATUS" == "true" && "$GITHUB_EVENT_NAME" == "pull_request" ]]; then
    log_info "Updating GitHub status check..."
    
    STATUS_PAYLOAD=$(jq -n \
        --arg state "pending" \
        --arg description "AgentScan security scan in progress..." \
        --arg context "AgentScan Security" \
        --arg target_url "$AGENTSCAN_API_URL/scans/$SCAN_ID" \
        '{
            state: $state,
            description: $description,
            context: $context,
            target_url: $target_url
        }')
    
    curl -s -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Content-Type: application/json" \
        -d "$STATUS_PAYLOAD" \
        "https://api.github.com/repos/$GITHUB_REPOSITORY/statuses/$GITHUB_SHA" > /dev/null || {
            log_warning "Failed to update GitHub status"
        }
fi

# Poll for scan completion
log_info "Waiting for scan to complete (timeout: ${TIMEOUT}m)..."
TIMEOUT_SECONDS=$((TIMEOUT * 60))
START_TIME=$(date +%s)

while true; do
    # Check timeout
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    if [[ $ELAPSED -gt $TIMEOUT_SECONDS ]]; then
        log_error "Scan timeout after ${TIMEOUT} minutes"
        exit 1
    fi
    
    # Get scan status
    STATUS_RESPONSE=$(curl -s \
        -H "Authorization: Bearer $AGENTSCAN_API_KEY" \
        -H "Accept: application/json" \
        "$AGENTSCAN_API_URL/api/v1/scans/$SCAN_ID" || {
            log_warning "Failed to get scan status, retrying..."
            sleep 10
            continue
        })
    
    STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.status // "unknown"')
    
    case "$STATUS" in
        "completed")
            log_success "Scan completed successfully"
            break
            ;;
        "failed")
            ERROR_MSG=$(echo "$STATUS_RESPONSE" | jq -r '.error_message // "Unknown error"')
            log_error "Scan failed: $ERROR_MSG"
            exit 1
            ;;
        "cancelled")
            log_error "Scan was cancelled"
            exit 1
            ;;
        "running"|"queued")
            PROGRESS=$(echo "$STATUS_RESPONSE" | jq -r '.progress // 0')
            log_info "Scan in progress... ${PROGRESS}%"
            sleep 15
            ;;
        *)
            log_warning "Unknown scan status: $STATUS"
            sleep 10
            ;;
    esac
done

# Get scan results
log_info "Retrieving scan results..."
RESULTS_RESPONSE=$(curl -s \
    -H "Authorization: Bearer $AGENTSCAN_API_KEY" \
    -H "Accept: application/json" \
    "$AGENTSCAN_API_URL/api/v1/scans/$SCAN_ID/results" || {
        log_error "Failed to retrieve scan results"
        exit 1
    })

# Save results to file
echo "$RESULTS_RESPONSE" > "$OUTPUT_FILE"
log_success "Results saved to $OUTPUT_FILE"

# Parse results
FINDINGS=$(echo "$RESULTS_RESPONSE" | jq '.findings // []')
TOTAL_COUNT=$(echo "$FINDINGS" | jq 'length')
HIGH_COUNT=$(echo "$FINDINGS" | jq '[.[] | select(.severity == "high")] | length')
MEDIUM_COUNT=$(echo "$FINDINGS" | jq '[.[] | select(.severity == "medium")] | length')
LOW_COUNT=$(echo "$FINDINGS" | jq '[.[] | select(.severity == "low")] | length')

# Set outputs
echo "findings-count=$TOTAL_COUNT" >> "$GITHUB_OUTPUT"
echo "high-count=$HIGH_COUNT" >> "$GITHUB_OUTPUT"
echo "medium-count=$MEDIUM_COUNT" >> "$GITHUB_OUTPUT"
echo "low-count=$LOW_COUNT" >> "$GITHUB_OUTPUT"
echo "results-url=$AGENTSCAN_API_URL/scans/$SCAN_ID" >> "$GITHUB_OUTPUT"

# Generate SARIF output if requested
if [[ "$OUTPUT_FORMAT" == "sarif" || "$OUTPUT_FORMAT" == "json,sarif" ]]; then
    SARIF_FILE="${OUTPUT_FILE%.json}.sarif"
    
    # Convert to SARIF format
    agentscan-action convert-sarif \
        --input "$OUTPUT_FILE" \
        --output "$SARIF_FILE" \
        --repository "$REPOSITORY_URL" \
        --commit "$GITHUB_SHA"
    
    echo "sarif-file=$SARIF_FILE" >> "$GITHUB_OUTPUT"
    log_success "SARIF results saved to $SARIF_FILE"
fi

# Print summary
log_info "Scan Summary:"
echo "  Total findings: $TOTAL_COUNT"
echo "  High severity: $HIGH_COUNT"
echo "  Medium severity: $MEDIUM_COUNT"
echo "  Low severity: $LOW_COUNT"

# Determine final status
SCAN_STATUS="passed"
if [[ "$FAIL_ON_HIGH" == "true" && $HIGH_COUNT -gt 0 ]]; then
    SCAN_STATUS="failed"
elif [[ "$FAIL_ON_MEDIUM" == "true" && $MEDIUM_COUNT -gt 0 ]]; then
    SCAN_STATUS="failed"
elif [[ "$FAIL_ON_LOW" == "true" && $LOW_COUNT -gt 0 ]]; then
    SCAN_STATUS="failed"
fi

echo "scan-status=$SCAN_STATUS" >> "$GITHUB_OUTPUT"

# Update GitHub status with final result
if [[ "$UPDATE_STATUS" == "true" && "$GITHUB_EVENT_NAME" == "pull_request" ]]; then
    if [[ "$SCAN_STATUS" == "passed" ]]; then
        STATUS_STATE="success"
        STATUS_DESC="AgentScan security scan passed ($TOTAL_COUNT findings)"
    else
        STATUS_STATE="failure"
        STATUS_DESC="AgentScan security scan failed (High: $HIGH_COUNT, Medium: $MEDIUM_COUNT)"
    fi
    
    STATUS_PAYLOAD=$(jq -n \
        --arg state "$STATUS_STATE" \
        --arg description "$STATUS_DESC" \
        --arg context "AgentScan Security" \
        --arg target_url "$AGENTSCAN_API_URL/scans/$SCAN_ID" \
        '{
            state: $state,
            description: $description,
            context: $context,
            target_url: $target_url
        }')
    
    curl -s -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Content-Type: application/json" \
        -d "$STATUS_PAYLOAD" \
        "https://api.github.com/repos/$GITHUB_REPOSITORY/statuses/$GITHUB_SHA" > /dev/null || {
            log_warning "Failed to update final GitHub status"
        }
fi

# Add PR comment if enabled and this is a PR
if [[ "$COMMENT_PR" == "true" && "$GITHUB_EVENT_NAME" == "pull_request" ]]; then
    log_info "Adding PR comment..."
    
    # Get PR number
    PR_NUMBER=$(jq -r '.number' "$GITHUB_EVENT_PATH")
    
    # Generate comment body
    COMMENT_BODY="## 🛡️ AgentScan Security Report

**Scan ID:** \`$SCAN_ID\`
**Status:** $(if [[ "$SCAN_STATUS" == "passed" ]]; then echo "✅ Passed"; else echo "❌ Failed"; fi)

### 📊 Summary
- **Total Findings:** $TOTAL_COUNT
- **High Severity:** $HIGH_COUNT 🔴
- **Medium Severity:** $MEDIUM_COUNT 🟡  
- **Low Severity:** $LOW_COUNT 🔵

### 🔗 Links
- [View Detailed Results]($AGENTSCAN_API_URL/scans/$SCAN_ID)
- [Download Results](https://github.com/$GITHUB_REPOSITORY/actions/runs/$GITHUB_RUN_ID)

---
*Secured by [AgentScan](https://agentscan.dev) - Multi-agent security scanning*"

    # Post comment
    COMMENT_PAYLOAD=$(jq -n --arg body "$COMMENT_BODY" '{body: $body}')
    
    curl -s -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Content-Type: application/json" \
        -d "$COMMENT_PAYLOAD" \
        "https://api.github.com/repos/$GITHUB_REPOSITORY/issues/$PR_NUMBER/comments" > /dev/null || {
            log_warning "Failed to add PR comment"
        }
fi

# Final result
if [[ "$SCAN_STATUS" == "passed" ]]; then
    log_success "AgentScan security scan completed successfully!"
    exit 0
else
    log_error "AgentScan security scan failed due to security findings"
    exit 1
fi