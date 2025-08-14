#!/bin/bash

# AgentScan Demo Environment Setup Script
# This script creates a self-service demo environment with vulnerable code samples

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEMO_REPO_NAME="agentscan-demo-vulnerable-app"
DEMO_DESCRIPTION="Vulnerable application for AgentScan security scanning demonstration"
AGENTSCAN_API_URL="${AGENTSCAN_API_URL:-https://api.agentscan.dev}"

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
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --github-token TOKEN    GitHub personal access token"
    echo "  --api-key KEY          AgentScan API key"
    echo "  --org ORG              GitHub organization (default: agentscan-demo)"
    echo "  --public               Make repository public (default: private)"
    echo "  --help                 Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  GITHUB_TOKEN           GitHub personal access token"
    echo "  AGENTSCAN_API_KEY      AgentScan API key"
    echo "  AGENTSCAN_API_URL      AgentScan API URL"
}

parse_arguments() {
    # Default values
    GITHUB_TOKEN="${GITHUB_TOKEN:-}"
    AGENTSCAN_API_KEY="${AGENTSCAN_API_KEY:-}"
    GITHUB_ORG="agentscan-demo"
    REPO_PRIVATE=true
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --github-token)
                GITHUB_TOKEN="$2"
                shift 2
                ;;
            --api-key)
                AGENTSCAN_API_KEY="$2"
                shift 2
                ;;
            --org)
                GITHUB_ORG="$2"
                shift 2
                ;;
            --public)
                REPO_PRIVATE=false
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
        log_error "GitHub token is required. Use --github-token or set GITHUB_TOKEN environment variable."
        exit 1
    fi
    
    if [[ -z "$AGENTSCAN_API_KEY" ]]; then
        log_error "AgentScan API key is required. Use --api-key or set AGENTSCAN_API_KEY environment variable."
        exit 1
    fi
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if required tools are installed
    local required_tools=("curl" "jq" "git")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is required but not installed."
            exit 1
        fi
    done
    
    log_success "Prerequisites check passed"
}

create_demo_repository() {
    log_info "Creating demo repository..."
    
    # Create repository payload
    local repo_payload=$(jq -n \
        --arg name "$DEMO_REPO_NAME" \
        --arg description "$DEMO_DESCRIPTION" \
        --argjson private "$REPO_PRIVATE" \
        '{
            name: $name,
            description: $description,
            private: $private,
            has_issues: true,
            has_projects: false,
            has_wiki: false,
            auto_init: true,
            gitignore_template: "Node",
            license_template: "mit"
        }')
    
    # Create the repository
    local response=$(curl -s -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Content-Type: application/json" \
        -d "$repo_payload" \
        "https://api.github.com/orgs/$GITHUB_ORG/repos" 2>/dev/null || \
        curl -s -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Content-Type: application/json" \
        -d "$repo_payload" \
        "https://api.github.com/user/repos")
    
    if ! echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        log_error "Failed to create repository"
        echo "Response: $response"
        exit 1
    fi
    
    REPO_URL=$(echo "$response" | jq -r '.clone_url')
    REPO_FULL_NAME=$(echo "$response" | jq -r '.full_name')
    
    log_success "Repository created: $REPO_FULL_NAME"
}

setup_demo_content() {
    log_info "Setting up demo content..."
    
    # Create temporary directory
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    # Clone the repository
    git clone "$REPO_URL" demo-repo
    cd demo-repo
    
    # Create directory structure
    mkdir -p {src,lib,config,scripts,docs}
    
    # Copy vulnerable samples
    cp -r "$(dirname "$0")/vulnerable-samples/"* src/
    
    # Create package.json for JavaScript project
    cat > package.json << 'EOF'
{
  "name": "agentscan-demo-vulnerable-app",
  "version": "1.0.0",
  "description": "Vulnerable web application for AgentScan security scanning demonstration",
  "main": "src/javascript/xss-example.js",
  "scripts": {
    "start": "node src/javascript/xss-example.js",
    "test": "echo \"Error: no test specified\" && exit 1"
  },
  "dependencies": {
    "express": "^4.18.2",
    "mysql": "^2.18.1",
    "xml2js": "^0.6.2",
    "ldapjs": "^3.0.7"
  },
  "devDependencies": {
    "eslint": "^8.0.0",
    "eslint-plugin-security": "^1.7.1"
  },
  "keywords": [
    "demo",
    "security",
    "vulnerable",
    "agentscan"
  ],
  "author": "AgentScan Team",
  "license": "MIT"
}
EOF
    
    # Create requirements.txt for Python project
    cat > requirements.txt << 'EOF'
Flask==2.3.3
PyMySQL==1.1.0
python-ldap==3.4.3
PyYAML==6.0.1
pymongo==4.5.0
lxml==4.9.3
EOF
    
    # Create go.mod for Go project
    cat > go.mod << 'EOF'
module github.com/agentscan-demo/vulnerable-app

go 1.21

require (
    github.com/go-sql-driver/mysql v1.7.1
)
EOF
    
    # Create README with demo instructions
    cat > README.md << 'EOF'
# AgentScan Demo - Vulnerable Application

This repository contains intentionally vulnerable code samples for demonstrating AgentScan's security scanning capabilities.

⚠️ **WARNING**: This code contains serious security vulnerabilities and should NEVER be used in production!

## What's Inside

This demo application showcases various types of security vulnerabilities that AgentScan can detect:

### 🔍 Vulnerability Types Demonstrated

- **SQL Injection** - Direct query construction with user input
- **Cross-Site Scripting (XSS)** - Reflected, stored, and DOM-based XSS
- **Command Injection** - OS command execution with user input
- **Path Traversal** - Unrestricted file access
- **Insecure Deserialization** - Unsafe object deserialization
- **Weak Cryptography** - MD5 hashing, weak random numbers
- **Hardcoded Secrets** - API keys and passwords in source code
- **XML External Entity (XXE)** - XML parsing vulnerabilities
- **Server-Side Template Injection** - Template engines with user input
- **Race Conditions** - Concurrent access without proper locking
- **Information Disclosure** - Exposing sensitive debug information

### 📁 Project Structure

```
src/
├── javascript/     # Node.js/Express vulnerabilities
├── python/         # Flask/Django vulnerabilities  
└── go/            # Go web application vulnerabilities
```

### 🚀 Getting Started with AgentScan

1. **Install AgentScan VS Code Extension**
   ```bash
   code --install-extension agentscan-security
   ```

2. **Configure AgentScan**
   - Set your API key in VS Code settings
   - Configure server URL: `https://api.agentscan.dev`

3. **Scan This Repository**
   - Open any file in VS Code
   - Save the file to trigger automatic scanning
   - Or use `Ctrl+Shift+P` → "AgentScan: Scan Workspace"

4. **Explore the Results**
   - View findings in the Problems panel
   - Hover over highlighted code for details
   - Use F8/Shift+F8 to navigate between findings

### 🎯 Expected Scan Results

AgentScan should detect approximately:
- **25+ High Severity** vulnerabilities
- **15+ Medium Severity** issues  
- **10+ Low Severity** findings

### 📊 Multi-Agent Consensus

Watch how AgentScan's multiple security tools work together:
- **Semgrep** - Pattern-based static analysis
- **ESLint Security** - JavaScript security rules
- **Bandit** - Python security linter
- **Gosec** - Go security analyzer
- **TruffleHog** - Secret detection

### 🔧 Try These Features

1. **Real-time Scanning** - Edit files and see instant feedback
2. **Rich Tooltips** - Hover over findings for detailed information
3. **Quick Actions** - Suppress false positives, mark as fixed
4. **Security Health** - View overall security posture
5. **GitHub Integration** - Create PRs to see scan results in comments

### 🛡️ Learning Resources

- [AgentScan Documentation](https://docs.agentscan.dev)
- [Security Best Practices](https://docs.agentscan.dev/best-practices)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)

### ⚠️ Disclaimer

This application is for educational and demonstration purposes only. The vulnerabilities are intentional and should not be deployed in any production environment.

---

**Secured by AgentScan** - Multi-agent security scanning platform
EOF
    
    # Create AgentScan configuration
    mkdir -p .agentscan
    cat > .agentscan/config.yml << 'EOF'
# AgentScan Demo Configuration
scan:
  languages:
    - javascript
    - typescript
    - python
    - go
  
  include:
    - "src/**"
    - "lib/**"
    - "*.js"
    - "*.py"
    - "*.go"
  
  exclude:
    - "node_modules/**"
    - "vendor/**"
    - "*.min.js"

agents:
  sast:
    enabled: true
    tools:
      - semgrep
      - eslint-security
      - bandit
      - gosec
  
  sca:
    enabled: true
    tools:
      - npm-audit
      - pip-audit
      - go-mod-audit
  
  secrets:
    enabled: true
    tools:
      - truffhog
      - detect-secrets

reporting:
  min_severity: low
  formats:
    - json
    - sarif
  
  github:
    pr_comments: true
    status_checks: true
    fail_on_high: true
    fail_on_medium: false
EOF
    
    # Create GitHub Actions workflow
    mkdir -p .github/workflows
    cat > .github/workflows/agentscan-demo.yml << 'EOF'
name: AgentScan Security Demo

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
        api-key: ${{ secrets.AGENTSCAN_API_KEY }}
        fail-on-high: false  # Don't fail for demo purposes
        fail-on-medium: false
        comment-pr: true
        
    - name: Upload results
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: agentscan-results
        path: agentscan-results.json
EOF
    
    # Commit and push changes
    git add .
    git commit -m "feat: Add vulnerable code samples for AgentScan demo

This commit adds intentionally vulnerable code samples to demonstrate
AgentScan's security scanning capabilities across multiple languages:

- JavaScript/Node.js vulnerabilities (XSS, SQL injection, command injection)
- Python/Flask vulnerabilities (SSTI, deserialization, weak crypto)
- Go web application vulnerabilities (path traversal, race conditions)

⚠️ WARNING: This code contains serious security vulnerabilities and 
should NEVER be used in production environments!

Features demonstrated:
- Multi-agent consensus scanning
- Real-time VS Code integration
- GitHub Actions workflow
- Comprehensive vulnerability coverage
- Educational security examples"
    
    git push origin main
    
    # Cleanup
    cd /
    rm -rf "$temp_dir"
    
    log_success "Demo content setup completed"
}

register_demo_repository() {
    log_info "Registering demo repository with AgentScan..."
    
    # Register the repository
    local payload=$(jq -n \
        --arg repo "https://github.com/$REPO_FULL_NAME" \
        --arg branch "main" \
        '{
            repository_url: $repo,
            default_branch: $branch,
            settings: {
                auto_scan_enabled: true,
                pr_comments_enabled: true,
                status_checks_enabled: true,
                scan_on_push: true,
                scan_on_pr: true
            },
            tags: ["demo", "vulnerable", "educational"]
        }')
    
    local response=$(curl -s -X POST \
        -H "Authorization: Bearer $AGENTSCAN_API_KEY" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        -d "$payload" \
        "$AGENTSCAN_API_URL/api/v1/repositories")
    
    if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        local repo_id=$(echo "$response" | jq -r '.id')
        log_success "Repository registered with AgentScan (ID: $repo_id)"
    else
        log_warning "Could not register repository with AgentScan"
        echo "Response: $response"
    fi
}

trigger_initial_scan() {
    log_info "Triggering initial security scan..."
    
    # Trigger an initial scan
    local payload=$(jq -n \
        --arg repo "https://github.com/$REPO_FULL_NAME" \
        --arg branch "main" \
        '{
            repository_url: $repo,
            branch: $branch,
            scan_type: "full",
            priority: 1
        }')
    
    local response=$(curl -s -X POST \
        -H "Authorization: Bearer $AGENTSCAN_API_KEY" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        -d "$payload" \
        "$AGENTSCAN_API_URL/api/v1/scans")
    
    if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        local scan_id=$(echo "$response" | jq -r '.id')
        log_success "Initial scan triggered (ID: $scan_id)"
        echo "Monitor scan progress at: $AGENTSCAN_API_URL/scans/$scan_id"
    else
        log_warning "Could not trigger initial scan"
        echo "Response: $response"
    fi
}

print_demo_summary() {
    log_success "Demo environment setup completed!"
    echo ""
    echo "=== Demo Repository Information ==="
    echo "Repository: https://github.com/$REPO_FULL_NAME"
    echo "Clone URL: $REPO_URL"
    echo "Visibility: $(if $REPO_PRIVATE; then echo "Private"; else echo "Public"; fi)"
    echo ""
    echo "=== Getting Started ==="
    echo "1. Clone the repository:"
    echo "   git clone $REPO_URL"
    echo ""
    echo "2. Install AgentScan VS Code extension:"
    echo "   code --install-extension agentscan-security"
    echo ""
    echo "3. Configure AgentScan in VS Code settings:"
    echo "   - Server URL: $AGENTSCAN_API_URL"
    echo "   - API Key: [Your AgentScan API key]"
    echo ""
    echo "4. Open the repository in VS Code and start scanning!"
    echo ""
    echo "=== Expected Results ==="
    echo "- 25+ High severity vulnerabilities"
    echo "- 15+ Medium severity issues"
    echo "- 10+ Low severity findings"
    echo ""
    echo "=== Demo Features ==="
    echo "- Real-time scanning with sub-2-second response times"
    echo "- Rich hover tooltips with vulnerability details"
    echo "- Code actions for quick fixes and suppression"
    echo "- Multi-agent consensus across security tools"
    echo "- GitHub Actions integration"
    echo ""
    echo "=== Useful Links ==="
    echo "Repository: https://github.com/$REPO_FULL_NAME"
    echo "AgentScan Dashboard: $AGENTSCAN_API_URL/dashboard"
    echo "Documentation: https://docs.agentscan.dev"
    echo ""
}

# Main execution
main() {
    log_info "Setting up AgentScan demo environment..."
    
    check_prerequisites
    create_demo_repository
    setup_demo_content
    register_demo_repository
    trigger_initial_scan
    print_demo_summary
    
    log_success "Demo environment setup completed successfully!"
}

# Parse arguments and run
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_arguments "$@"
    main
fi