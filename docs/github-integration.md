# GitHub Integration

AgentScan provides comprehensive GitHub integration through a GitHub App that enables automatic security scanning on pull requests and pushes, with intelligent status checks and PR comments.

## Features

- **Automatic PR Scanning**: Scans are triggered automatically when PRs are opened, updated, or reopened
- **Push Event Scanning**: Scans default branch pushes for continuous security monitoring
- **Status Checks**: Creates GitHub status checks that integrate with branch protection rules
- **PR Comments**: Posts detailed security findings directly in PR comments
- **Check Runs**: Provides rich check run annotations with line-by-line security feedback
- **Repository Access Control**: Respects GitHub permissions and organization settings

## Setup

### 1. Create a GitHub App

1. Go to your GitHub organization settings
2. Navigate to "Developer settings" > "GitHub Apps"
3. Click "New GitHub App"
4. Fill in the required information:
   - **App name**: `AgentScan Security Scanner`
   - **Homepage URL**: `https://agentscan.dev`
   - **Webhook URL**: `https://your-domain.com/api/v1/internal/webhooks/github`
   - **Webhook secret**: Generate a secure random string

### 2. Configure Permissions

Grant the following permissions to your GitHub App:

#### Repository Permissions
- **Contents**: Read (to access repository code)
- **Metadata**: Read (to access repository information)
- **Pull requests**: Write (to post comments and create status checks)
- **Checks**: Write (to create check runs)
- **Statuses**: Write (to create status checks)

#### Organization Permissions
- **Members**: Read (for organization access control)

### 3. Subscribe to Events

Subscribe to the following webhook events:
- `pull_request` (opened, synchronize, reopened)
- `push` (for default branch monitoring)
- `installation` (for app lifecycle management)

### 4. Generate Private Key

1. In your GitHub App settings, scroll down to "Private keys"
2. Click "Generate a private key"
3. Download the `.pem` file and store it securely

### 5. Install the App

1. Go to the "Install App" tab in your GitHub App settings
2. Install the app on your organization or specific repositories
3. Note the installation ID from the URL after installation

### 6. Configure AgentScan

Set the following environment variables:

```bash
# GitHub App Configuration
GITHUB_APP_ID=123456
GITHUB_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"

# Webhook Configuration (optional, for signature verification)
GITHUB_WEBHOOK_SECRET=your_webhook_secret_here
```

## GitHub Actions Integration

AgentScan provides a GitHub Action for CI/CD integration:

### Basic Usage

Create `.github/workflows/agentscan.yml`:

```yaml
name: AgentScan Security

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
      pull-requests: write
      checks: write

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Run AgentScan
        uses: agentscan/agentscan-action@v1
        with:
          api-url: https://api.agentscan.dev
          api-token: ${{ secrets.AGENTSCAN_TOKEN }}
          fail-on-severity: high

      - name: Upload SARIF results
        uses: github/codeql-action/upload-sarif@v2
        if: always()
        with:
          sarif_file: agentscan-results.sarif
```

### Advanced Configuration

```yaml
      - name: Run AgentScan
        uses: agentscan/agentscan-action@v1
        with:
          api-url: https://api.agentscan.dev
          api-token: ${{ secrets.AGENTSCAN_TOKEN }}
          fail-on-severity: medium
          exclude-paths: |
            node_modules/**
            vendor/**
            *.min.js
            test/**
          include-tools: semgrep,eslint-security,bandit
          output-format: json,sarif,pdf
```

### Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `api-url` | AgentScan API URL | No | `https://api.agentscan.dev` |
| `api-token` | API authentication token | No | - |
| `fail-on-severity` | Fail on findings of this severity or higher | No | `high` |
| `exclude-paths` | Paths to exclude (newline separated) | No | - |
| `include-tools` | Comma-separated list of tools to include | No | - |
| `exclude-tools` | Comma-separated list of tools to exclude | No | - |
| `output-format` | Output format (json, sarif, pdf) | No | `json,sarif` |

### Action Outputs

| Output | Description |
|--------|-------------|
| `results-file` | Path to the results file |
| `findings-count` | Total number of findings |
| `high-severity-count` | Number of high severity findings |
| `medium-severity-count` | Number of medium severity findings |
| `low-severity-count` | Number of low severity findings |

## CLI Integration

Use the AgentScan CLI in any CI/CD system:

```bash
# Install CLI
curl -sSL https://install.agentscan.dev | sh

# Run scan
agentscan-cli scan \
  --api-url=https://api.agentscan.dev \
  --api-token=$AGENTSCAN_TOKEN \
  --fail-on-severity=high \
  --output-format=json,sarif
```

## Webhook Events

### Pull Request Events

When a pull request is opened, updated, or reopened:

1. **Webhook Received**: AgentScan receives the `pull_request` webhook
2. **Scan Triggered**: A high-priority scan is queued for the PR branch
3. **Status Check Created**: Initial "pending" status check is posted
4. **Scan Execution**: Multiple security agents analyze the code changes
5. **Results Processing**: Findings are deduplicated and scored using consensus
6. **Status Update**: Status check is updated with scan results
7. **PR Comment**: Detailed findings are posted as a PR comment

### Push Events

When code is pushed to the default branch:

1. **Webhook Received**: AgentScan receives the `push` webhook
2. **Incremental Scan**: Only changed files are scanned for efficiency
3. **Results Storage**: Findings are stored for trend analysis
4. **Notifications**: Team notifications are sent if configured

## Status Checks

AgentScan creates status checks that integrate with GitHub's branch protection:

- **Context**: `agentscan/security`
- **States**:
  - `pending`: Scan in progress
  - `success`: No high-severity issues found
  - `failure`: High-severity security issues detected
  - `error`: Scan failed due to technical issues

## PR Comments

AgentScan posts structured comments on pull requests:

```markdown
## 🔒 AgentScan Security Report

### Summary
🔴 **2 High** severity issues
🟡 **3 Medium** severity issues

### 🔴 High Severity Issues

**SQL Injection** in `src/database.js:42`
- Potential SQL injection vulnerability detected
- Detected by: semgrep

**Cross-Site Scripting (XSS)** in `src/render.js:15`
- Unescaped user input may lead to XSS
- Detected by: eslint-security

---
*Powered by [AgentScan](https://agentscan.dev) - Multi-agent security scanning*
```

## Repository Access Control

AgentScan respects GitHub's permission model:

- Users can only access scan results for repositories they have read access to
- Organization members see results based on their team memberships
- Private repository scans are only visible to authorized users
- Installation permissions control which repositories are scanned

## Troubleshooting

### Common Issues

1. **Webhook not received**
   - Check webhook URL configuration
   - Verify webhook secret matches
   - Check firewall and network settings

2. **Permission denied errors**
   - Verify GitHub App permissions
   - Check installation scope
   - Ensure user has repository access

3. **Scan timeouts**
   - Check repository size and complexity
   - Verify agent configuration
   - Monitor system resources

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
export GITHUB_DEBUG=true
```

### Webhook Testing

Test webhook delivery using GitHub's webhook testing tools or tools like ngrok for local development.

## Security Considerations

- **Private Key Security**: Store GitHub App private keys securely using environment variables or secret management systems
- **Webhook Signatures**: Always verify webhook signatures to prevent unauthorized requests
- **Access Control**: Implement proper RBAC to control access to scan results
- **Data Encryption**: Encrypt sensitive data at rest and in transit
- **Audit Logging**: Log all authentication and authorization events

## API Reference

### Webhook Endpoints

- `POST /api/v1/internal/webhooks/github` - GitHub webhook handler

### GitHub API Usage

AgentScan uses the following GitHub APIs:

- **Check Runs API**: For creating detailed check runs with annotations
- **Statuses API**: For creating simple status checks
- **Issues API**: For posting PR comments
- **Repositories API**: For accessing repository metadata
- **Installations API**: For managing app installations

## Rate Limits

AgentScan respects GitHub's API rate limits:

- **Primary rate limit**: 5,000 requests per hour per installation
- **Secondary rate limits**: Monitored and handled gracefully
- **Abuse detection**: Implements exponential backoff and retry logic

## Support

For GitHub integration support:

- Check the [troubleshooting guide](troubleshooting.md)
- Review [API documentation](api.md)
- Contact support at support@agentscan.dev