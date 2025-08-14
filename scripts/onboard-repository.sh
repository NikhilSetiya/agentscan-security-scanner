#!/bin/bash

# AgentScan Repository Onboarding Script
# This script automates the setup of AgentScan for a GitHub repository

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
AGENTSCAN_API_URL="${AGENTSCAN_API_URL:-https://api.agentscan.dev}"
GITHUB_API_URL="https://api.github.com"

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

usage() {
    echo "Usage: $0 <github-repo> [options]"
    echo ""
    echo "Arguments:"
    echo "  github-repo    GitHub repository in format 'owner/repo'"
    echo ""
    echo "Options:"
    echo "  --token TOKEN        GitHub personal access token"
    echo "  --api-key KEY        AgentScan API key"
    echo "  --branch BRANCH      Default branch (default: main)"
    echo "  --webhook-url URL    Custom webhook URL"
    echo "  --auto-scan          Enable automatic scanning on push"
    echo "  --pr-comments        Enable PR comments"
    echo "  --status-checks      Enable GitHub status checks"
    echo "  --help               Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  GITHUB_TOKEN         GitHub personal access token"
    echo "  AGENTSCAN_API_KEY    AgentScan API key"
    echo "  AGENTSCAN_API_URL    AgentScan API URL (default: https://api.agentscan.dev)"
    echo ""
    echo "Examples:"
    echo "  $0 myorg/myrepo --token ghp_xxx --api-key ask_xxx"
    echo "  $0 myorg/myrepo --auto-scan --pr-comments"
}

parse_arguments() {
    if [[ $# -lt 1 ]]; then
        usage
        exit 1
    fi
    
    REPO="$1"
    shift
    
    # Default values
    GITHUB_TOKEN="${GITHUB_TOKEN:-}"
    AGENTSCAN_API_KEY="${AGENTSCAN_API_KEY:-}"
    BRANCH="main"
    WEBHOOK_URL="$AGENTSCAN_API_URL/webhooks/github"
    AUTO_SCAN=false
    PR_COMMENTS=false
    STATUS_CHECKS=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --token)
                GITHUB_TOKEN="$2"
                shift 2
                ;;
            --api-key)
                AGENTSCAN_API_KEY="$2"
                shift 2
                ;;
            --branch)
                BRANCH="$2"
                shift 2
                ;;
            --webhook-url)
                WEBHOOK_URL="$2"
                shift 2
                ;;
            --auto-scan)
                AUTO_SCAN=true
                shift
                ;;
            --pr-comments)
                PR_COMMENTS=true
                shift
                ;;
            --status-checks)
                STATUS_CHECKS=true
                shift
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    # Validate required parameters
    if [[ -z "$GITHUB_TOKEN" ]]; then
        log_error "GitHub token is required. Use --token or set GITHUB_TOKEN environment variable."
        exit 1
    fi
    
    if [[ -z "$AGENTSCAN_API_KEY" ]]; then
        log_error "AgentScan API key is required. Use --api-key or set AGENTSCAN_API_KEY environment variable."
        exit 1
    fi
    
    if [[ ! "$REPO" =~ ^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$ ]]; then
        log_error "Invalid repository format. Use 'owner/repo'."
        exit 1
    fi
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed."
        exit 1
    fi
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed. Please install it:"
        echo "  # On macOS: brew install jq"
        echo "  # On Ubuntu: sudo apt-get install jq"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

validate_github_access() {
    log_info "Validating GitHub access..."
    
    # Check if token is valid
    response=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
                   -H "Accept: application/vnd.github.v3+json" \
                   "$GITHUB_API_URL/user")
    
    if ! echo "$response" | jq -e '.login' > /dev/null 2>&1; then
        log_error "Invalid GitHub token or API error"
        echo "Response: $response"
        exit 1
    fi
    
    username=$(echo "$response" | jq -r '.login')
    log_success "Authenticated as GitHub user: $username"
    
    # Check repository access
    log_info "Checking repository access..."
    repo_response=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
                         -H "Accept: application/vnd.github.v3+json" \
                         "$GITHUB_API_URL/repos/$REPO")
    
    if ! echo "$repo_response" | jq -e '.id' > /dev/null 2>&1; then
        log_error "Cannot access repository $REPO"
        echo "Response: $repo_response"
        exit 1
    fi
    
    repo_name=$(echo "$repo_response" | jq -r '.full_name')
    log_success "Repository access confirmed: $repo_name"
}

validate_agentscan_access() {
    log_info "Validating AgentScan access..."
    
    # Check if API key is valid
    response=$(curl -s -H "Authorization: Bearer $AGENTSCAN_API_KEY" \
                   -H "Accept: application/json" \
                   "$AGENTSCAN_API_URL/api/v1/user")
    
    if ! echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        log_error "Invalid AgentScan API key or API error"
        echo "Response: $response"
        exit 1
    fi
    
    user_email=$(echo "$response" | jq -r '.email // "unknown"')
    log_success "Authenticated with AgentScan as: $user_email"
}

register_repository() {
    log_info "Registering repository with AgentScan..."
    
    # Register the repository
    payload=$(jq -n \
        --arg repo "$REPO" \
        --arg branch "$BRANCH" \
        --argjson auto_scan "$AUTO_SCAN" \
        --argjson pr_comments "$PR_COMMENTS" \
        --argjson status_checks "$STATUS_CHECKS" \
        '{
            repository_url: ("https://github.com/" + $repo),
            default_branch: $branch,
            settings: {
                auto_scan_enabled: $auto_scan,
                pr_comments_enabled: $pr_comments,
                status_checks_enabled: $status_checks,
                scan_on_push: $auto_scan,
                scan_on_pr: true
            }
        }')
    
    response=$(curl -s -X POST \
                   -H "Authorization: Bearer $AGENTSCAN_API_KEY" \
                   -H "Content-Type: application/json" \
                   -H "Accept: application/json" \
                   -d "$payload" \
                   "$AGENTSCAN_API_URL/api/v1/repositories")
    
    if ! echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        log_error "Failed to register repository with AgentScan"
        echo "Response: $response"
        exit 1
    fi
    
    repo_id=$(echo "$response" | jq -r '.id')
    log_success "Repository registered with AgentScan (ID: $repo_id)"
}

setup_webhook() {
    log_info "Setting up GitHub webhook..."
    
    # Create webhook payload
    webhook_payload=$(jq -n \
        --arg url "$WEBHOOK_URL" \
        '{
            name: "web",
            active: true,
            events: ["push", "pull_request", "pull_request_review"],
            config: {
                url: $url,
                content_type: "json",
                insecure_ssl: "0"
            }
        }')
    
    # Create the webhook
    response=$(curl -s -X POST \
                   -H "Authorization: token $GITHUB_TOKEN" \
                   -H "Accept: application/vnd.github.v3+json" \
                   -H "Content-Type: application/json" \
                   -d "$webhook_payload" \
                   "$GITHUB_API_URL/repos/$REPO/hooks")
    
    if ! echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        log_error "Failed to create GitHub webhook"
        echo "Response: $response"
        exit 1
    fi
    
    webhook_id=$(echo "$response" | jq -r '.id')
    log_success "GitHub webhook created (ID: $webhook_id)"
}

create_workflow_file() {
    log_info "Creating GitHub Actions workflow..."
    
    # Create .github/workflows directory
    mkdir -p .github/workflows
    
    # Create AgentScan workflow file
    cat > .github/workflows/agentscan.yml << EOF
name: AgentScan Security

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    name: AgentScan Security Scan
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      with:
        fetch-depth: 0
    
    - name: Run AgentScan
      uses: agentscan/agentscan-action@v1
      with:
        api-key: \${{ secrets.AGENTSCAN_API_KEY }}
        api-url: $AGENTSCAN_API_URL
        fail-on-high: true
        fail-on-medium: false
        comment-pr: $PR_COMMENTS
        
    - name: Upload results
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: agentscan-results
        path: agentscan-results.json
EOF
    
    log_success "GitHub Actions workflow created at .github/workflows/agentscan.yml"
}

setup_repository_secrets() {
    log_info "Setting up repository secrets..."
    
    # Note: GitHub CLI or API with proper permissions is needed to set secrets
    log_warning "Please set the following repository secret manually:"
    echo "  AGENTSCAN_API_KEY = $AGENTSCAN_API_KEY"
    echo ""
    echo "You can set this secret at:"
    echo "  https://github.com/$REPO/settings/secrets/actions"
    echo ""
    echo "Or use GitHub CLI:"
    echo "  gh secret set AGENTSCAN_API_KEY --body \"$AGENTSCAN_API_KEY\" --repo $REPO"
}

create_agentscan_config() {
    log_info "Creating AgentScan configuration file..."
    
    # Create .agentscan directory
    mkdir -p .agentscan
    
    # Create configuration file
    cat > .agentscan/config.yml << EOF
# AgentScan Configuration
# This file configures how AgentScan scans your repository

# Scan settings
scan:
  # Languages to scan (auto-detected if not specified)
  languages:
    - javascript
    - typescript
    - python
    - go
    - java
  
  # Paths to include in scanning
  include:
    - "src/**"
    - "lib/**"
    - "app/**"
  
  # Paths to exclude from scanning
  exclude:
    - "node_modules/**"
    - "vendor/**"
    - "*.min.js"
    - "dist/**"
    - "build/**"
    - "coverage/**"
    - "test/**"
    - "tests/**"
    - "spec/**"

# Agent configuration
agents:
  # SAST (Static Analysis Security Testing)
  sast:
    enabled: true
    tools:
      - semgrep
      - eslint-security
      - bandit
      - gosec
      - spotbugs
  
  # SCA (Software Composition Analysis)
  sca:
    enabled: true
    tools:
      - npm-audit
      - pip-audit
      - go-mod-audit
  
  # Secrets scanning
  secrets:
    enabled: true
    tools:
      - truffhog
      - detect-secrets
  
  # DAST (Dynamic Analysis Security Testing)
  dast:
    enabled: false  # Enable for web applications
    timeout: 10m

# Reporting settings
reporting:
  # Minimum severity to report
  min_severity: medium
  
  # Output formats
  formats:
    - json
    - sarif
  
  # GitHub integration
  github:
    pr_comments: $PR_COMMENTS
    status_checks: $STATUS_CHECKS
    fail_on_high: true
    fail_on_medium: false

# Suppression rules
suppressions:
  # Example: Suppress specific rules
  # - rule_id: "javascript.express.security.audit.xss.mustache.var-in-href"
  #   reason: "False positive - user input is sanitized"
  #   expires: "2024-12-31"
EOF
    
    log_success "AgentScan configuration created at .agentscan/config.yml"
}

run_initial_scan() {
    log_info "Triggering initial scan..."
    
    # Trigger an initial scan
    payload=$(jq -n \
        --arg repo "https://github.com/$REPO" \
        --arg branch "$BRANCH" \
        '{
            repository_url: $repo,
            branch: $branch,
            scan_type: "full",
            priority: 1
        }')
    
    response=$(curl -s -X POST \
                   -H "Authorization: Bearer $AGENTSCAN_API_KEY" \
                   -H "Content-Type: application/json" \
                   -H "Accept: application/json" \
                   -d "$payload" \
                   "$AGENTSCAN_API_URL/api/v1/scans")
    
    if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        scan_id=$(echo "$response" | jq -r '.id')
        log_success "Initial scan triggered (ID: $scan_id)"
        echo "You can monitor the scan at: $AGENTSCAN_API_URL/scans/$scan_id"
    else
        log_warning "Could not trigger initial scan"
        echo "Response: $response"
    fi
}

print_onboarding_summary() {
    log_success "Repository onboarding completed!"
    echo ""
    echo "=== Onboarding Summary ==="
    echo "Repository: $REPO"
    echo "Default branch: $BRANCH"
    echo "Auto scan: $AUTO_SCAN"
    echo "PR comments: $PR_COMMENTS"
    echo "Status checks: $STATUS_CHECKS"
    echo ""
    echo "=== Files Created ==="
    echo "- .github/workflows/agentscan.yml"
    echo "- .agentscan/config.yml"
    echo ""
    echo "=== Next Steps ==="
    echo "1. Set the AGENTSCAN_API_KEY repository secret"
    echo "2. Commit and push the configuration files"
    echo "3. Create a pull request to test the integration"
    echo "4. Review and customize the .agentscan/config.yml file"
    echo "5. Monitor your first scan results"
    echo ""
    echo "=== Useful Links ==="
    echo "Repository settings: https://github.com/$REPO/settings"
    echo "Actions secrets: https://github.com/$REPO/settings/secrets/actions"
    echo "AgentScan dashboard: $AGENTSCAN_API_URL/dashboard"
    echo ""
}

# Main execution
main() {
    log_info "Starting AgentScan repository onboarding for $REPO..."
    
    check_prerequisites
    validate_github_access
    validate_agentscan_access
    register_repository
    setup_webhook
    create_workflow_file
    setup_repository_secrets
    create_agentscan_config
    run_initial_scan
    print_onboarding_summary
    
    log_success "Repository onboarding completed successfully!"
}

# Parse arguments and run
parse_arguments "$@"
main