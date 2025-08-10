# AgentScan CI/CD Integrations

This directory contains integrations for various CI/CD platforms, enabling seamless security scanning in your development workflows.

## 🚀 Quick Start

### GitHub Actions

Add the following to your `.github/workflows/security.yml`:

```yaml
name: Security Scan

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  security:
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
          api-token: ${{ secrets.AGENTSCAN_TOKEN }}
          fail-on-severity: high
```

### GitLab CI

Add the following to your `.gitlab-ci.yml`:

```yaml
include:
  - remote: 'https://raw.githubusercontent.com/agentscan/agentscan/main/integrations/gitlab/.gitlab-ci.yml'

variables:
  AGENTSCAN_API_TOKEN: $AGENTSCAN_TOKEN
  AGENTSCAN_FAIL_ON_SEVERITY: "high"
```

### Jenkins

Install the AgentScan Jenkins plugin and add to your pipeline:

```groovy
pipeline {
    agent any
    stages {
        stage('Security Scan') {
            steps {
                agentScan(
                    apiUrl: 'https://api.agentscan.dev',
                    credentialsId: 'agentscan-token',
                    failOnSeverity: 'high'
                )
            }
        }
    }
}
```

## 📋 Platform Support

| Platform | Status | Features |
|----------|--------|----------|
| GitHub Actions | ✅ Complete | Workflow, Action, PR comments, Status checks |
| GitLab CI | ✅ Complete | Pipeline, SAST reports, MR comments |
| Jenkins | ✅ Complete | Plugin, Pipeline step, HTML reports |
| Azure DevOps | 🔄 Planned | Extension, Pipeline task |
| CircleCI | 🔄 Planned | Orb, Workflow integration |
| Bitbucket Pipelines | 🔄 Planned | Pipe, PR integration |

## 🔧 Configuration Options

### Common Options

All integrations support these configuration options:

| Option | Description | Default | Example |
|--------|-------------|---------|---------|
| `api-url` | AgentScan API URL | `https://api.agentscan.dev` | `https://api.company.com` |
| `api-token` | API authentication token | - | `${{ secrets.AGENTSCAN_TOKEN }}` |
| `fail-on-severity` | Fail build on findings | `high` | `medium`, `low`, `never` |
| `exclude-paths` | Paths to exclude from scanning | - | `node_modules/**,vendor/**` |
| `include-paths` | Paths to include in scanning | - | `src/**,lib/**` |
| `output-format` | Output format | `json,sarif` | `json`, `sarif`, `pdf` |
| `timeout` | Scan timeout | `30m` | `15m`, `1h` |

### Platform-Specific Options

#### GitHub Actions

| Option | Description | Default |
|--------|-------------|---------|
| `upload-sarif` | Upload SARIF to Security tab | `true` |
| `create-check-run` | Create detailed check run | `true` |
| `comment-pr` | Comment on pull requests | `true` |

#### GitLab CI

| Option | Description | Default |
|--------|-------------|---------|
| `generate-sast-report` | Generate GitLab SAST report | `true` |
| `comment-mr` | Comment on merge requests | `true` |
| `upload-artifacts` | Upload scan artifacts | `true` |

#### Jenkins

| Option | Description | Default |
|--------|-------------|---------|
| `generate-html-report` | Generate HTML report | `true` |
| `archive-results` | Archive scan results | `true` |
| `update-build-status` | Update build description | `true` |

## 🏗️ Installation

### CLI Installation

The AgentScan CLI can be installed on any CI/CD system:

#### Unix/Linux/macOS
```bash
curl -sSL https://install.agentscan.dev | sh
```

#### Windows
```powershell
Invoke-WebRequest -Uri https://install.agentscan.dev/windows -OutFile install.ps1; .\install.ps1
```

#### Manual Installation
Download the appropriate binary from [GitHub Releases](https://github.com/agentscan/agentscan/releases).

### Platform-Specific Installation

#### GitHub Actions
No installation required - use the pre-built action:
```yaml
- uses: agentscan/agentscan-action@v1
```

#### GitLab CI
Include the template in your `.gitlab-ci.yml`:
```yaml
include:
  - remote: 'https://raw.githubusercontent.com/agentscan/agentscan/main/integrations/gitlab/.gitlab-ci.yml'
```

#### Jenkins
1. Install the AgentScan plugin from the Jenkins Plugin Manager
2. Configure global settings in Jenkins → Manage Jenkins → Configure System
3. Add AgentScan build step to your jobs

## 🔐 Authentication

### API Token Setup

1. **Get API Token**: Sign up at [AgentScan](https://agentscan.dev) and generate an API token
2. **Store Securely**: Add the token to your CI/CD platform's secret management:
   - **GitHub**: Repository Settings → Secrets → Actions → `AGENTSCAN_TOKEN`
   - **GitLab**: Project Settings → CI/CD → Variables → `AGENTSCAN_TOKEN`
   - **Jenkins**: Manage Jenkins → Credentials → Add Secret Text → `agentscan-token`

### Self-Hosted Installations

For on-premises AgentScan installations:

```yaml
# GitHub Actions
- uses: agentscan/agentscan-action@v1
  with:
    api-url: https://agentscan.company.com
    api-token: ${{ secrets.AGENTSCAN_TOKEN }}

# GitLab CI
variables:
  AGENTSCAN_API_URL: "https://agentscan.company.com"
  AGENTSCAN_API_TOKEN: $AGENTSCAN_TOKEN

# Jenkins
agentScan(
    apiUrl: 'https://agentscan.company.com',
    credentialsId: 'agentscan-token'
)
```

## 📊 Reports and Outputs

### GitHub Actions Outputs

The GitHub Action sets the following outputs:

```yaml
- name: Check scan results
  run: |
    echo "Total findings: ${{ steps.agentscan.outputs.findings-count }}"
    echo "High severity: ${{ steps.agentscan.outputs.high-severity-count }}"
    echo "Results file: ${{ steps.agentscan.outputs.results-file }}"
```

### GitLab CI Artifacts

GitLab CI jobs produce these artifacts:

- `agentscan-results.json` - Detailed scan results
- `agentscan-results.sarif` - SARIF format for security tools
- `gl-sast-report.json` - GitLab SAST report format

### Jenkins Reports

Jenkins plugin generates:

- HTML security report (published to job page)
- Archived JSON and SARIF results
- Build status updates with finding counts

## 🔄 Workflow Examples

### Basic Security Gate

Fail the build if high-severity issues are found:

```yaml
# GitHub Actions
- uses: agentscan/agentscan-action@v1
  with:
    api-token: ${{ secrets.AGENTSCAN_TOKEN }}
    fail-on-severity: high

# GitLab CI
variables:
  AGENTSCAN_FAIL_ON_SEVERITY: "high"

# Jenkins
agentScan(
    credentialsId: 'agentscan-token',
    failOnSeverity: 'high'
)
```

### Comprehensive Scanning

Scan with multiple output formats and custom exclusions:

```yaml
# GitHub Actions
- uses: agentscan/agentscan-action@v1
  with:
    api-token: ${{ secrets.AGENTSCAN_TOKEN }}
    fail-on-severity: medium
    exclude-paths: |
      node_modules/**
      vendor/**
      *.min.js
      test/**
    output-format: json,sarif,pdf
    upload-sarif: true

# GitLab CI
variables:
  AGENTSCAN_FAIL_ON_SEVERITY: "medium"
  AGENTSCAN_EXCLUDE_PATHS: "node_modules/**,vendor/**,*.min.js,test/**"
  AGENTSCAN_OUTPUT_FORMAT: "json,sarif,pdf"

# Jenkins
agentScan(
    credentialsId: 'agentscan-token',
    failOnSeverity: 'medium',
    excludePaths: 'node_modules/**,vendor/**,*.min.js,test/**',
    outputFormat: 'json,sarif,pdf',
    generateReport: true
)
```

### Scheduled Security Scans

Run comprehensive security scans on a schedule:

```yaml
# GitHub Actions
name: Scheduled Security Scan
on:
  schedule:
    - cron: '0 2 * * 1'  # Weekly on Monday at 2 AM

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: agentscan/agentscan-action@v1
        with:
          api-token: ${{ secrets.AGENTSCAN_TOKEN }}
          fail-on-severity: low  # More strict for scheduled scans
          timeout: 60m

# GitLab CI
agentscan_scheduled:
  extends: agentscan_security_scan
  variables:
    AGENTSCAN_FAIL_ON_SEVERITY: "low"
    AGENTSCAN_TIMEOUT: "60m"
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"

# Jenkins
pipeline {
    triggers {
        cron('0 2 * * 1')  // Weekly on Monday at 2 AM
    }
    stages {
        stage('Scheduled Security Scan') {
            steps {
                agentScan(
                    credentialsId: 'agentscan-token',
                    failOnSeverity: 'low',
                    timeoutMinutes: 60
                )
            }
        }
    }
}
```

## 🐛 Troubleshooting

### Common Issues

#### Authentication Errors
```
Error: API request failed: 401 Unauthorized
```
**Solution**: Verify your API token is correctly set in your CI/CD platform's secrets.

#### Timeout Issues
```
Error: Scan timed out after 30 minutes
```
**Solution**: Increase the timeout value or exclude large directories:
```yaml
timeout: 60m
exclude-paths: node_modules/**,vendor/**
```

#### Network Issues
```
Error: Failed to connect to AgentScan API
```
**Solution**: Check network connectivity and firewall rules. For self-hosted installations, verify the API URL.

### Debug Mode

Enable verbose logging for troubleshooting:

```yaml
# GitHub Actions
- uses: agentscan/agentscan-action@v1
  with:
    verbose: true

# GitLab CI
variables:
  AGENTSCAN_VERBOSE: "true"

# Jenkins
agentScan(
    verbose: true
)
```

### Support

- 📖 **Documentation**: [docs.agentscan.dev](https://docs.agentscan.dev)
- 🐛 **Issues**: [GitHub Issues](https://github.com/agentscan/agentscan/issues)
- 💬 **Community**: [Discord](https://discord.gg/agentscan)
- 📧 **Support**: support@agentscan.dev

## 🤝 Contributing

We welcome contributions to improve our CI/CD integrations!

### Adding New Platforms

1. Create a new directory under `integrations/`
2. Add platform-specific configuration files
3. Create tests in `integrations/tests/`
4. Update this README with usage instructions
5. Submit a pull request

### Testing

Run the integration tests:

```bash
cd integrations/tests
go test -v ./...
```

### Development

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.