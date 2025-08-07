# Semgrep SAST Agent

This package implements a Semgrep security scanning agent that follows the AgentScan SecurityAgent interface.

## Overview

The Semgrep agent provides static application security testing (SAST) capabilities by wrapping the Semgrep tool in a Docker container. It supports multiple programming languages and provides comprehensive security analysis with standardized output.

## Features

- **Multi-language Support**: JavaScript, TypeScript, Python, Go, Java, C, C++, Ruby, PHP, Scala, Kotlin, Rust
- **Docker Execution**: Runs Semgrep in isolated Docker containers with resource limits
- **SARIF Output Parsing**: Converts Semgrep's SARIF output to standardized Finding format
- **Configurable Rules**: Supports custom rule configurations and language-specific scanning
- **Error Handling**: Comprehensive error handling with timeout management
- **Resource Management**: Configurable memory and CPU limits

## Supported Vulnerability Categories

- SQL Injection
- Cross-Site Scripting (XSS)
- Command Injection
- Path Traversal
- Insecure Cryptography
- Hardcoded Secrets
- Insecure Deserialization
- Authentication Bypass
- Cross-Site Request Forgery (CSRF)
- Misconfiguration

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/agentscan/agentscan/agents/sast/semgrep"
    "github.com/agentscan/agentscan/pkg/agent"
)

func main() {
    // Create a new Semgrep agent
    semgrepAgent := semgrep.NewAgent()

    // Configure the scan
    config := agent.ScanConfig{
        RepoURL:   "https://github.com/example/vulnerable-app.git",
        Branch:    "main",
        Languages: []string{"javascript", "python"},
        Timeout:   10 * time.Minute,
    }

    // Execute the scan
    result, err := semgrepAgent.Scan(context.Background(), config)
    if err != nil {
        fmt.Printf("Scan failed: %v\n", err)
        return
    }

    // Process results
    fmt.Printf("Scan completed with %d findings\n", len(result.Findings))
    for _, finding := range result.Findings {
        fmt.Printf("- %s: %s (%s)\n", finding.Severity, finding.Title, finding.File)
    }
}
```

### Custom Configuration

```go
// Create agent with custom configuration
config := semgrep.AgentConfig{
    DockerImage:    "returntocorp/semgrep:1.45.0",
    MaxMemoryMB:    1024,
    MaxCPUCores:    2.0,
    DefaultTimeout: 15 * time.Minute,
    RulesConfig:    "p/security-audit",
}

agent := semgrep.NewAgentWithConfig(config)
```

### Incremental Scanning

```go
// Scan only specific files (for incremental scans)
config := agent.ScanConfig{
    RepoURL: "https://github.com/example/app.git",
    Branch:  "main",
    Files:   []string{"src/auth.js", "src/database.py"},
    Timeout: 5 * time.Minute,
}
```

## Configuration Options

### AgentConfig

- `DockerImage`: Semgrep Docker image to use (default: "returntocorp/semgrep:latest")
- `MaxMemoryMB`: Maximum memory limit in MB (default: 512)
- `MaxCPUCores`: Maximum CPU cores (default: 1.0)
- `DefaultTimeout`: Default scan timeout (default: 10 minutes)
- `RulesConfig`: Semgrep rules configuration (default: "auto")

### ScanConfig

- `RepoURL`: Git repository URL to scan
- `Branch`: Git branch to scan (default: "main")
- `Commit`: Specific commit SHA to scan (optional)
- `Languages`: List of languages to scan (optional, auto-detected if empty)
- `Files`: List of specific files to scan (optional, for incremental scans)
- `Rules`: Custom rule configurations (optional)
- `Options`: Additional agent-specific options (optional)
- `Timeout`: Maximum scan duration (optional, uses agent default if not set)

## Requirements

- Docker must be installed and accessible
- Git must be installed for repository cloning
- Network access to clone repositories and pull Docker images

## Error Handling

The agent implements comprehensive error handling for common scenarios:

- **Docker Unavailable**: Returns error if Docker is not installed or accessible
- **Image Pull Failures**: Attempts to pull Semgrep image if not available locally
- **Repository Clone Failures**: Handles invalid repository URLs or access issues
- **Scan Timeouts**: Respects timeout configurations and cancels long-running scans
- **Resource Exhaustion**: Enforces memory and CPU limits through Docker

## Testing

Run the test suite:

```bash
# Run all tests
go test -v ./agents/sast/semgrep/...

# Run tests excluding integration tests
go test -short -v ./agents/sast/semgrep/...

# Run with coverage
go test -cover ./agents/sast/semgrep/...
```

## Performance Considerations

- **Memory Usage**: Default 512MB limit, increase for large repositories
- **CPU Usage**: Default 1.0 core, increase for faster scanning
- **Timeout**: Default 10 minutes, adjust based on repository size
- **Incremental Scanning**: Use file filters to scan only changed files

## Security Considerations

- Runs in isolated Docker containers
- Repository data is mounted read-only
- Temporary files are cleaned up after scanning
- Resource limits prevent resource exhaustion attacks

## Troubleshooting

### Common Issues

1. **Docker not found**: Ensure Docker is installed and in PATH
2. **Permission denied**: Ensure user has Docker permissions
3. **Image pull failures**: Check network connectivity and Docker Hub access
4. **Repository clone failures**: Verify repository URL and access permissions
5. **Scan timeouts**: Increase timeout for large repositories

### Debug Mode

Enable verbose logging by setting environment variables:

```bash
export DOCKER_BUILDKIT=1
export BUILDKIT_PROGRESS=plain
```

## Contributing

When contributing to the Semgrep agent:

1. Follow the existing code style and patterns
2. Add comprehensive tests for new functionality
3. Update documentation for configuration changes
4. Ensure all tests pass before submitting changes
5. Consider performance impact of changes

## License

This agent is part of the AgentScan project and follows the same licensing terms.