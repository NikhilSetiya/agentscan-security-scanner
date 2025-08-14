# AgentScan Security Scanner - GitHub Action

🛡️ **Multi-agent security scanning with 80% fewer false positives**

AgentScan is a comprehensive security scanner that uses multiple security tools in parallel and applies consensus scoring to dramatically reduce false positives while maintaining comprehensive vulnerability coverage.

## 🚀 Quick Start

Add this workflow to your repository at `.github/workflows/agentscan.yml`:

```yaml
name: AgentScan Security

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: agentscan/agentscan-action@v1
      with:
        api-key: ${{ secrets.AGENTSCAN_API_KEY }}
```

## 📋 Prerequisites

1. **Get your API key**: Sign up at [agentscan.dev](https://agentscan.dev/signup)
2. **Add repository secret**: Go to Settings → Secrets → Actions and add `AGENTSCAN_API_KEY`
3. **That's it!** The action will run automatically on pushes and PRs

## ⚙️ Configuration

### Required Inputs

| Input | Description |
|-------|-------------|
| `api-key` | Your AgentScan API key (store as repository secret) |

### Optional Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `api-url` | `https://api.agentscan.dev` | AgentScan API URL |
| `fail-on-high` | `true` | Fail build on high severity findings |
| `fail-on-medium` | `false` | Fail build on medium severity findings |
| `fail-on-low` | `false` | Fail build on low severity findings |
| `comment-pr` | `true` | Add scan results as PR comment |
| `update-status` | `true` | Update GitHub status checks |
| `output-format` | `json` | Output format (`json`, `sarif`, `junit`) |
| `timeout` | `15` | Scan timeout in minutes |
| `agents` | `sast,sca,secrets` | Agents to run (`sast`, `sca`, `secrets`, `dast`) |

### Example with Custom Configuration

```yaml
- uses: agentscan/agentscan-action@v1
  with:
    api-key: ${{ secrets.AGENTSCAN_API_KEY }}
    fail-on-high: true
    fail-on-medium: true
    comment-pr: true
    agents: 'sast,sca,secrets'
    timeout: 20
```##
 📊 Outputs

| Output | Description |
|--------|-------------|
| `scan-id` | Unique identifier for the scan |
| `findings-count` | Total number of findings |
| `high-count` | Number of high severity findings |
| `medium-count` | Number of medium severity findings |
| `low-count` | Number of low severity findings |
| `scan-status` | Final scan status (`passed`, `failed`, `error`) |
| `results-url` | URL to view detailed results |
| `sarif-file` | Path to SARIF results file |

### Using Outputs

```yaml
- name: Run AgentScan
  id: agentscan
  uses: agentscan/agentscan-action@v1
  with:
    api-key: ${{ secrets.AGENTSCAN_API_KEY }}

- name: Check results
  run: |
    echo "Scan ID: ${{ steps.agentscan.outputs.scan-id }}"
    echo "Findings: ${{ steps.agentscan.outputs.findings-count }}"
    echo "Status: ${{ steps.agentscan.outputs.scan-status }}"
```

## 🔍 Supported Languages

- **JavaScript/TypeScript** - ESLint Security, Semgrep
- **Python** - Bandit, Semgrep, pip-audit
- **Go** - Gosec, Semgrep, go-mod-audit
- **Java** - SpotBugs, Semgrep
- **C#** - Security Code Scan, Semgrep
- **Ruby** - Brakeman, Semgrep
- **PHP** - PHPCS Security, Semgrep
- **Rust** - Cargo Audit, Semgrep

## 🛡️ Security Features

### Multi-Agent Consensus
- Runs multiple security tools in parallel
- Uses consensus scoring to reduce false positives by 80%
- Provides confidence scores for each finding

### Comprehensive Coverage
- **SAST**: Static Application Security Testing
- **SCA**: Software Composition Analysis (dependency scanning)
- **Secrets**: Hardcoded secrets and credentials detection
- **DAST**: Dynamic Application Security Testing (optional)

### Integration Features
- GitHub Security tab integration (SARIF output)
- Pull request comments with detailed findings
- Status checks to prevent merging vulnerable code
- Artifact uploads for detailed analysis

## 🚨 Troubleshooting

### Common Issues

#### 1. Authentication Failed
```
Error: AGENTSCAN_API_KEY is required
```

**Solution:**
1. Get your API key from [agentscan.dev](https://agentscan.dev/signup)
2. Add it as a repository secret named `AGENTSCAN_API_KEY`
3. Ensure the secret name matches exactly in your workflow

#### 2. API Connection Issues
```
Error: Failed to connect to AgentScan API
```

**Solutions:**
- Check if `api-url` is correct (default: `https://api.agentscan.dev`)
- Verify your API key is valid and not expired
- Check if there are network restrictions in your organization

#### 3. Scan Timeout
```
Error: Scan timeout after 15 minutes
```

**Solutions:**
- Increase timeout: `timeout: 30`
- Exclude large files or directories
- Contact support if scans consistently timeout

#### 4. No Findings Expected
```
Warning: No security findings detected
```

This is actually good news! Your code appears secure. You can:
- Verify the scan covered your intended files
- Check the scan logs for any skipped files
- Consider adding the "Secured by AgentScan" badge to your README

#### 5. Too Many Findings
```
Error: Build failed due to high severity findings
```

**Solutions:**
- Review and fix high severity issues first
- Temporarily set `fail-on-high: false` while addressing issues
- Use `fail-on-medium: false` for gradual improvement
- Suppress false positives through the AgentScan dashboard

### Debug Mode

Enable debug logging by setting the `ACTIONS_STEP_DEBUG` secret to `true`:

```yaml
env:
  ACTIONS_STEP_DEBUG: true
```

### Getting Help

1. **Check the logs**: Review the action logs for detailed error messages
2. **Documentation**: Visit [docs.agentscan.dev](https://docs.agentscan.dev)
3. **GitHub Issues**: Report issues at [github.com/agentscan/agentscan-action](https://github.com/agentscan/agentscan-action/issues)
4. **Support**: Email support@agentscan.dev
5. **Community**: Join our [Slack community](https://agentscan.dev/slack)

## 📈 Advanced Usage

### Matrix Builds

Scan multiple versions or configurations:

```yaml
strategy:
  matrix:
    node-version: [16, 18, 20]
    
steps:
- uses: actions/setup-node@v3
  with:
    node-version: ${{ matrix.node-version }}
- uses: agentscan/agentscan-action@v1
  with:
    api-key: ${{ secrets.AGENTSCAN_API_KEY }}
```

### Conditional Scanning

Only scan on specific conditions:

```yaml
- uses: agentscan/agentscan-action@v1
  if: github.event_name == 'pull_request'
  with:
    api-key: ${{ secrets.AGENTSCAN_API_KEY }}
```

### Custom File Filtering

```yaml
- uses: agentscan/agentscan-action@v1
  with:
    api-key: ${{ secrets.AGENTSCAN_API_KEY }}
    include-paths: 'src/**,lib/**'
    exclude-paths: 'test/**,docs/**'
```

### Integration with Other Actions

```yaml
- name: Run tests
  run: npm test

- name: Security scan
  uses: agentscan/agentscan-action@v1
  with:
    api-key: ${{ secrets.AGENTSCAN_API_KEY }}

- name: Deploy
  if: success()
  run: npm run deploy
```

## 🎯 Best Practices

### 1. Fail Fast on High Severity
```yaml
fail-on-high: true
fail-on-medium: false  # Gradually increase strictness
```

### 2. Use Branch Protection
Configure branch protection rules to require AgentScan status checks.

### 3. Regular Scanning
Run scans on:
- Every push to main/master
- All pull requests
- Scheduled weekly scans for comprehensive coverage

### 4. Team Notifications
Set up Slack or email notifications for security findings:

```yaml
- name: Notify team
  if: failure()
  uses: 8398a7/action-slack@v3
  with:
    status: failure
    text: 'Security scan failed! Check the results.'
```

### 5. Artifact Management
Always upload scan results for later analysis:

```yaml
- uses: actions/upload-artifact@v3
  if: always()
  with:
    name: security-results
    path: |
      agentscan-results.json
      agentscan-results.sarif
```

## 🔒 Security Considerations

- **API Key Security**: Never expose your API key in logs or code
- **Private Repositories**: Ensure your plan supports private repo scanning
- **Data Privacy**: AgentScan only analyzes code structure, not content
- **Compliance**: SARIF output integrates with GitHub Security tab

## 📊 Pricing

- **Free**: Unlimited public repository scanning
- **Pro**: $9/month for private repositories and advanced features
- **Team**: $29/month for team management and analytics
- **Enterprise**: Custom pricing for large organizations

[View detailed pricing](https://agentscan.dev/pricing)

## 🤝 Contributing

We welcome contributions! Please see our [contributing guide](https://github.com/agentscan/agentscan-action/blob/main/CONTRIBUTING.md).

## 📄 License

This action is licensed under the [MIT License](https://github.com/agentscan/agentscan-action/blob/main/LICENSE).

---

**Questions?** Check our [FAQ](https://docs.agentscan.dev/faq) or [contact support](mailto:support@agentscan.dev).