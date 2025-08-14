# OWASP ZAP DAST Agent

This agent provides Dynamic Application Security Testing (DAST) capabilities using OWASP ZAP (Zed Attack Proxy).

## Overview

The ZAP agent automatically detects web applications in repositories, starts them in isolated Docker containers, and performs security scans against the running applications. This complements static analysis by finding runtime vulnerabilities that can only be detected when the application is running.

## Features

### Web Application Detection

The agent automatically detects web applications by analyzing:

- **Node.js/JavaScript**: `package.json` with web framework dependencies (Express, Next.js, React, etc.)
- **Python**: `requirements.txt` with web frameworks (Django, Flask, FastAPI, etc.)
- **Java**: `pom.xml` or `build.gradle` with Spring Boot or web dependencies
- **PHP**: `composer.json` with Laravel, Symfony, or other PHP frameworks
- **Ruby**: `Gemfile` with Rails, Sinatra, or Rack
- **Go**: `go.mod` with web framework dependencies (Gin, Echo, Fiber, etc.)
- **Docker**: `Dockerfile` with web application indicators

### Supported Scan Types

- **Baseline Scan** (default): Quick passive scan with minimal false positives
- **Full Scan**: Comprehensive active scan including spider and active attacks
- **API Scan**: Specialized scanning for REST APIs

### Vulnerability Categories

The agent detects common web application vulnerabilities:

- Cross-Site Scripting (XSS)
- SQL Injection
- Cross-Site Request Forgery (CSRF)
- Authentication Bypass
- Command Injection
- Path Traversal
- Insecure Deserialization
- Security Misconfigurations

## Configuration

### Default Configuration

```go
config := AgentConfig{
    DockerImage:    "owasp/zap2docker-stable:latest",
    MaxMemoryMB:    1024,
    MaxCPUCores:    1.0,
    DefaultTimeout: 10 * time.Minute,
    ScanType:       "baseline",
    MaxDepth:       5,
}
```

### Custom Configuration

```go
customConfig := AgentConfig{
    DockerImage:    "owasp/zap2docker-weekly:latest",
    MaxMemoryMB:    2048,
    MaxCPUCores:    2.0,
    DefaultTimeout: 15 * time.Minute,
    ScanType:       "full",
    MaxDepth:       10,
}

agent := NewAgentWithConfig(customConfig)
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/agentscan/agentscan/agents/dast/zap"
    "github.com/agentscan/agentscan/pkg/agent"
)

func main() {
    zapAgent := zap.NewAgent()
    
    config := agent.ScanConfig{
        RepoURL:   "https://github.com/example/webapp.git",
        Branch:    "main",
        Languages: []string{"javascript", "python"},
        Timeout:   10 * time.Minute,
    }
    
    result, err := zapAgent.Scan(context.Background(), config)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Scan completed with %d findings\n", len(result.Findings))
    for _, finding := range result.Findings {
        fmt.Printf("- %s: %s (Severity: %s)\n", 
            finding.RuleID, finding.Title, finding.Severity)
    }
}
```

### Health Check

```go
ctx := context.Background()
if err := zapAgent.HealthCheck(ctx); err != nil {
    fmt.Printf("ZAP agent not ready: %v\n", err)
} else {
    fmt.Println("ZAP agent is ready")
}
```

## How It Works

### 1. Web Application Detection

The agent analyzes the repository structure to determine if it contains a web application:

```go
// Detects Node.js web apps
if fileExists("package.json") {
    config := analyzeNodeJS("package.json")
    if config != nil {
        // Found web application
    }
}
```

### 2. Application Startup

If a web application is detected, the agent:

1. Clones the repository to a temporary directory
2. Creates a Dockerfile if one doesn't exist
3. Builds a Docker image
4. Starts the application in an isolated container
5. Waits for the application to be ready (health check)

### 3. Security Scanning

Once the application is running, ZAP performs:

1. **Passive Scanning**: Analyzes HTTP traffic without sending attacks
2. **Active Scanning**: Sends test payloads to find vulnerabilities
3. **Spider/Crawling**: Discovers application endpoints
4. **Authentication**: Handles login forms and sessions (if configured)

### 4. Result Processing

The agent parses ZAP's JSON output and converts it to standardized findings:

```go
type Finding struct {
    ID          string
    Tool        string         // "zap"
    RuleID      string         // ZAP plugin ID
    Severity    Severity       // High/Medium/Low/Info
    Category    VulnCategory   // XSS, SQLi, etc.
    Title       string
    Description string
    File        string         // URL that was tested
    Confidence  float64
    References  []string
}
```

## Framework-Specific Detection

### Node.js Applications

Detects frameworks by analyzing `package.json`:

- **Express.js**: `express` dependency
- **Next.js**: `next` dependency
- **React**: `react-scripts` dependency
- **Fastify**: `fastify` dependency
- **Koa**: `koa` dependency

Default start command: `npm start` or `npm run dev`

### Python Applications

Detects frameworks by analyzing `requirements.txt`:

- **Django**: `Django` package
- **Flask**: `Flask` package  
- **FastAPI**: `fastapi` package
- **Tornado**: `tornado` package

### Java Applications

Detects frameworks by analyzing `pom.xml` or `build.gradle`:

- **Spring Boot**: `spring-boot-starter-web`
- **Jersey**: `jersey-server`
- **Servlet API**: `servlet-api`

## Limitations

### Current Limitations

1. **Application Startup**: Some applications may require additional configuration (databases, environment variables)
2. **Authentication**: Basic authentication detection only
3. **Complex Deployments**: Multi-service applications not fully supported
4. **Resource Usage**: DAST scans are resource-intensive and time-consuming

### Graceful Degradation

The agent gracefully handles failures:

- If web application detection fails → Skip DAST, return empty results
- If application startup fails → Skip DAST, return empty results  
- If ZAP scan fails → Return partial results with error information
- If scan times out → Return available results

## Testing

### Unit Tests

```bash
go test -v ./agents/dast/zap
```

### Integration Tests

```bash
# Requires Docker
go test -v ./agents/dast/zap -tags=integration
```

### Test Coverage

The test suite covers:

- Web application detection for all supported frameworks
- ZAP output parsing and finding conversion
- Error handling and graceful degradation
- Configuration validation
- Dockerfile generation

## Security Considerations

### Container Isolation

- Applications run in isolated Docker containers
- Containers are automatically cleaned up after scanning
- Network access is limited to localhost
- Resource limits prevent resource exhaustion

### Sensitive Data

- Temporary directories are cleaned up automatically
- No sensitive data is logged or persisted
- Application containers are ephemeral

## Performance

### Typical Scan Times

- **Baseline Scan**: 2-5 minutes
- **Full Scan**: 10-30 minutes (depending on application size)
- **API Scan**: 5-15 minutes

### Resource Requirements

- **Memory**: 1-2 GB (configurable)
- **CPU**: 1-2 cores (configurable)  
- **Disk**: 500 MB - 2 GB (temporary files)
- **Network**: Internet access for image pulls

## Troubleshooting

### Common Issues

1. **Docker not available**
   ```
   Error: docker not available
   Solution: Install Docker and ensure it's running
   ```

2. **Application won't start**
   ```
   Error: application failed to start within timeout
   Solution: Check application dependencies and configuration
   ```

3. **ZAP scan timeout**
   ```
   Error: zap execution failed with timeout
   Solution: Increase timeout or use baseline scan instead of full scan
   ```

### Debug Mode

Enable debug output by setting environment variable:

```bash
export ZAP_DEBUG=true
```

This will show detailed ZAP command execution and output.

## Contributing

### Adding New Framework Support

To add support for a new web framework:

1. Add detection logic in `scanner.go`
2. Add framework-specific Dockerfile template in `createDockerfile()`
3. Add test cases in `scanner_test.go`
4. Update this README

### Example: Adding Ruby on Rails Support

```go
func (a *Agent) analyzeRuby(repoPath string) *WebAppConfig {
    gemfilePath := filepath.Join(repoPath, "Gemfile")
    // ... detection logic
    
    return &WebAppConfig{
        Framework:    "rails",
        StartCommand: "bundle exec rails server",
        Port:         3000,
        HealthCheck:  "/",
        Timeout:      60 * time.Second,
    }
}
```