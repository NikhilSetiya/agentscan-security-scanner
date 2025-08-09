# Secret Scanning Agents

This directory contains secret scanning agents that detect hardcoded secrets, API keys, passwords, and other sensitive information in source code and git repositories.

## Agents

### TruffleHog Agent

The TruffleHog agent uses [TruffleHog](https://github.com/trufflesecurity/trufflehog) to scan for secrets in git repositories and file systems.

**Features:**
- Scans git history for secrets
- Supports 700+ secret detectors
- Verifies secrets when possible
- High accuracy with low false positives
- Configurable depth for git history scanning

**Configuration:**
```go
config := trufflehog.AgentConfig{
    DockerImage:      "trufflesecurity/trufflehog:latest",
    MaxMemoryMB:      1024,
    MaxCPUCores:      2.0,
    DefaultTimeout:   15 * time.Minute,
    ScanGitHistory:   true,
    MaxDepth:         100,
    IncludeDetectors: []string{"aws", "github", "slack"},
    ExcludeDetectors: []string{"generic"},
    Whitelist:        []string{"test.*", "example.*"},
}
```

### git-secrets Agent

The git-secrets agent uses [git-secrets](https://github.com/awslabs/git-secrets) to prevent secrets from being committed to git repositories.

**Features:**
- Scans working directory and commit history
- Supports AWS, Azure, and GCP patterns
- Custom pattern support
- Fast scanning with regex patterns
- Configurable provider patterns

**Configuration:**
```go
config := gitsecrets.AgentConfig{
    DockerImage:      "agentscan/git-secrets:latest",
    MaxMemoryMB:      512,
    MaxCPUCores:      1.0,
    DefaultTimeout:   10 * time.Minute,
    CustomPatterns:   []string{"custom-pattern-1", "custom-pattern-2"},
    ProviderPatterns: []string{"aws", "azure", "gcp"},
    Whitelist:        []string{"test.*", "example.*"},
    ScanCommits:      true,
}
```

## Supported Secret Types

Both agents detect various types of secrets:

### Cloud Provider Secrets
- **AWS**: Access keys, secret keys, session tokens
- **Azure**: Connection strings, service principal credentials
- **Google Cloud**: Service account keys, API keys

### Version Control Secrets
- **GitHub**: Personal access tokens, OAuth tokens
- **GitLab**: Personal access tokens, deploy tokens
- **Bitbucket**: App passwords, access tokens

### API Keys and Tokens
- **Stripe**: API keys, webhook secrets
- **Slack**: Bot tokens, webhook URLs
- **Twilio**: Account SID, auth tokens
- **SendGrid**: API keys
- **Mailgun**: API keys

### Database Credentials
- **PostgreSQL**: Connection strings, passwords
- **MySQL**: Connection strings, passwords
- **MongoDB**: Connection strings, passwords
- **Redis**: Passwords, connection URLs

### Cryptographic Material
- **Private Keys**: RSA, ECDSA, Ed25519 private keys
- **JWT Secrets**: Signing keys, tokens
- **SSL/TLS**: Private keys, certificates

### Generic Secrets
- **Passwords**: Database passwords, application passwords
- **API Keys**: Generic API keys and secrets
- **Tokens**: Authentication tokens, session tokens

## High Severity Flagging

As per the requirements, **all detected secrets are flagged as high severity**. This ensures that security teams prioritize secret remediation regardless of the secret type or verification status.

The severity determination follows this logic:
1. **Verified secrets**: Always high severity
2. **Critical service secrets** (AWS, GitHub, etc.): Always high severity
3. **All other secrets**: High severity by default

## Secret Pattern Customization

Both agents support custom patterns and whitelisting:

### Custom Patterns
Add custom regex patterns to detect organization-specific secrets:

```go
customPatterns := []string{
    `MYCOMPANY_API_KEY_[A-Za-z0-9]{32}`,
    `internal_secret_[A-Za-z0-9]+`,
}
```

### Whitelisting
Exclude false positives using regex patterns:

```go
whitelist := []string{
    `test/.*`,           // Ignore test files
    `examples/.*`,       // Ignore example files
    `.*example.*`,       // Ignore content with "example"
    `.*placeholder.*`,   // Ignore placeholder values
}
```

## Usage Examples

### Basic Usage

```go
// TruffleHog agent
truffleAgent := trufflehog.NewAgent()
result, err := truffleAgent.Scan(ctx, agent.ScanConfig{
    RepoURL: "https://github.com/user/repo",
    Branch:  "main",
    Timeout: 10 * time.Minute,
})

// git-secrets agent
gitSecretsAgent := gitsecrets.NewAgent()
result, err := gitSecretsAgent.Scan(ctx, agent.ScanConfig{
    RepoURL: "https://github.com/user/repo",
    Branch:  "main",
    Timeout: 5 * time.Minute,
})
```

### Custom Configuration

```go
// TruffleHog with custom config
truffleConfig := trufflehog.AgentConfig{
    MaxDepth:         50,
    IncludeDetectors: []string{"aws", "github"},
    Whitelist:        []string{"test/.*"},
}
truffleAgent := trufflehog.NewAgentWithConfig(truffleConfig)

// git-secrets with custom config
gitSecretsConfig := gitsecrets.AgentConfig{
    CustomPatterns:   []string{`MYAPI_[A-Za-z0-9]{32}`},
    ProviderPatterns: []string{"aws"},
    ScanCommits:      false,
}
gitSecretsAgent := gitsecrets.NewAgentWithConfig(gitSecretsConfig)
```

### Incremental Scanning

```go
// Scan only specific files (incremental scan)
result, err := agent.Scan(ctx, agent.ScanConfig{
    RepoURL: "https://github.com/user/repo",
    Branch:  "main",
    Files:   []string{"src/config.js", "config/database.yml"},
    Timeout: 2 * time.Minute,
})
```

## Docker Images

### TruffleHog
Uses the official TruffleHog Docker image:
```
trufflesecurity/trufflehog:latest
```

### git-secrets
Uses a custom Docker image with git-secrets installed:
```
agentscan/git-secrets:latest
```

To build the git-secrets image:
```bash
cd agents/secrets/gitsecrets
docker build -t agentscan/git-secrets:latest .
```

## Testing

The agents include comprehensive tests:

### Unit Tests
```bash
# Test TruffleHog agent
go test ./agents/secrets/trufflehog -v

# Test git-secrets agent
go test ./agents/secrets/gitsecrets -v
```

### Integration Tests
```bash
# Test both agents with sample repositories
go test ./agents/secrets -v
```

The integration tests create temporary git repositories with various types of secrets and validate that both agents can detect them correctly.

## Performance Considerations

### TruffleHog
- **Memory**: 1GB recommended for large repositories
- **CPU**: 2 cores recommended for parallel scanning
- **Time**: ~2-5 minutes for typical repositories
- **Git History**: Configurable depth (default: 100 commits)

### git-secrets
- **Memory**: 512MB sufficient for most repositories
- **CPU**: 1 core sufficient
- **Time**: ~1-2 minutes for typical repositories
- **Patterns**: Fast regex-based scanning

## Security Considerations

1. **Secret Redaction**: Both agents redact secrets in output for safety
2. **Temporary Files**: All temporary files are cleaned up after scanning
3. **Docker Isolation**: Scans run in isolated Docker containers
4. **Memory Limits**: Configurable memory limits prevent resource exhaustion
5. **Timeout Protection**: Configurable timeouts prevent hanging scans

## Troubleshooting

### Common Issues

1. **Docker not available**: Ensure Docker is installed and running
2. **Image pull failures**: Check network connectivity and Docker Hub access
3. **Git clone failures**: Verify repository URL and access permissions
4. **Timeout errors**: Increase timeout for large repositories
5. **Memory errors**: Increase memory limits for large repositories

### Debug Mode

Enable debug logging by setting environment variables:
```bash
export AGENTSCAN_DEBUG=true
export DOCKER_DEBUG=true
```

### Health Checks

Both agents provide health check methods:
```go
err := agent.HealthCheck(ctx)
if err != nil {
    log.Printf("Agent health check failed: %v", err)
}
```

## Contributing

When adding new secret patterns or detectors:

1. Add patterns to the appropriate agent configuration
2. Update tests with sample secrets
3. Ensure proper redaction of sensitive data
4. Test with real repositories (using test secrets only)
5. Update documentation with new secret types

## References

- [TruffleHog Documentation](https://github.com/trufflesecurity/trufflehog)
- [git-secrets Documentation](https://github.com/awslabs/git-secrets)
- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [NIST Guidelines for Cryptographic Key Management](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)