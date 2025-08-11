# Custom Security Agents Development Guide

This guide walks you through creating custom security agents for AgentScan, allowing you to integrate any security tool into the platform.

## Overview

AgentScan's architecture is built around the concept of security agents - modular components that wrap existing security tools and provide a standardized interface for the orchestration engine. This design allows for easy integration of new tools and technologies.

## Agent Interface

All security agents must implement the `Agent` interface defined in Go:

```go
type Agent interface {
    // Scan performs the security scan on the provided repository
    Scan(ctx context.Context, request ScanRequest) (*ScanResult, error)
    
    // HealthCheck verifies the agent is ready to perform scans
    HealthCheck(ctx context.Context) error
    
    // GetConfig returns the agent's configuration and metadata
    GetConfig() AgentConfig
}
```

### Core Types

```go
type ScanRequest struct {
    RepositoryPath string            `json:"repository_path"`
    Language       string            `json:"language"`
    Branch         string            `json:"branch"`
    Commit         string            `json:"commit"`
    Options        map[string]string `json:"options"`
}

type ScanResult struct {
    Findings []Finding `json:"findings"`
    Metadata Metadata  `json:"metadata"`
    RawOutput string   `json:"raw_output,omitempty"`
}

type Finding struct {
    ID          string            `json:"id"`
    RuleID      string            `json:"rule_id"`
    Title       string            `json:"title"`
    Description string            `json:"description"`
    Severity    Severity          `json:"severity"`
    Confidence  Confidence        `json:"confidence"`
    FilePath    string            `json:"file_path"`
    LineNumber  int               `json:"line_number"`
    ColumnNumber int              `json:"column_number,omitempty"`
    CodeSnippet string            `json:"code_snippet,omitempty"`
    Tool        string            `json:"tool"`
    Category    string            `json:"category"`
    CWE         string            `json:"cwe,omitempty"`
    OWASP       string            `json:"owasp,omitempty"`
    References  []string          `json:"references,omitempty"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}

type AgentConfig struct {
    Name         string   `json:"name"`
    Version      string   `json:"version"`
    Description  string   `json:"description"`
    Languages    []string `json:"languages"`
    Categories   []string `json:"categories"`
    RequiredTools []string `json:"required_tools"`
}
```

## Creating a Custom Agent

### Step 1: Project Structure

Create a new directory for your agent:

```
agents/
└── my-custom-agent/
    ├── agent.go
    ├── config.go
    ├── parser.go
    ├── docker/
    │   └── Dockerfile
    └── tests/
        ├── agent_test.go
        └── fixtures/
```

### Step 2: Implement the Agent Interface

```go
package mycustomagent

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "path/filepath"
    
    "github.com/agentscan/agentscan/pkg/types"
)

type MyCustomAgent struct {
    config AgentConfig
}

func New() *MyCustomAgent {
    return &MyCustomAgent{
        config: AgentConfig{
            Name:        "my-custom-agent",
            Version:     "1.0.0",
            Description: "Custom security agent for specific vulnerability detection",
            Languages:   []string{"javascript", "typescript"},
            Categories:  []string{"sast", "custom"},
            RequiredTools: []string{"my-security-tool"},
        },
    }
}

func (a *MyCustomAgent) Scan(ctx context.Context, request types.ScanRequest) (*types.ScanResult, error) {
    // Validate input
    if request.RepositoryPath == "" {
        return nil, fmt.Errorf("repository path is required")
    }

    // Prepare command
    cmd := exec.CommandContext(ctx, "my-security-tool", 
        "--format", "json",
        "--path", request.RepositoryPath,
    )

    // Execute the security tool
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to execute security tool: %w", err)
    }

    // Parse the output
    findings, err := a.parseOutput(output)
    if err != nil {
        return nil, fmt.Errorf("failed to parse tool output: %w", err)
    }

    return &types.ScanResult{
        Findings: findings,
        Metadata: types.Metadata{
            Tool:     a.config.Name,
            Version:  a.config.Version,
            Duration: time.Since(startTime),
        },
        RawOutput: string(output),
    }, nil
}

func (a *MyCustomAgent) HealthCheck(ctx context.Context) error {
    // Check if the required tool is available
    cmd := exec.CommandContext(ctx, "my-security-tool", "--version")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("security tool not available: %w", err)
    }
    return nil
}

func (a *MyCustomAgent) GetConfig() types.AgentConfig {
    return a.config
}
```

### Step 3: Implement Output Parsing

```go
func (a *MyCustomAgent) parseOutput(output []byte) ([]types.Finding, error) {
    var toolOutput struct {
        Issues []struct {
            ID          string `json:"id"`
            Rule        string `json:"rule"`
            Message     string `json:"message"`
            Severity    string `json:"severity"`
            File        string `json:"file"`
            Line        int    `json:"line"`
            Column      int    `json:"column"`
            Code        string `json:"code"`
            Category    string `json:"category"`
            CWE         string `json:"cwe"`
            References  []string `json:"references"`
        } `json:"issues"`
    }

    if err := json.Unmarshal(output, &toolOutput); err != nil {
        return nil, fmt.Errorf("failed to unmarshal tool output: %w", err)
    }

    var findings []types.Finding
    for _, issue := range toolOutput.Issues {
        finding := types.Finding{
            ID:           generateFindingID(issue.Rule, issue.File, issue.Line),
            RuleID:       issue.Rule,
            Title:        issue.Message,
            Description:  a.getDescription(issue.Rule),
            Severity:     a.mapSeverity(issue.Severity),
            Confidence:   types.ConfidenceHigh, // Adjust based on tool
            FilePath:     issue.File,
            LineNumber:   issue.Line,
            ColumnNumber: issue.Column,
            CodeSnippet:  issue.Code,
            Tool:         a.config.Name,
            Category:     issue.Category,
            CWE:          issue.CWE,
            References:   issue.References,
        }
        findings = append(findings, finding)
    }

    return findings, nil
}

func (a *MyCustomAgent) mapSeverity(toolSeverity string) types.Severity {
    switch strings.ToLower(toolSeverity) {
    case "critical", "high":
        return types.SeverityHigh
    case "medium", "moderate":
        return types.SeverityMedium
    case "low", "minor":
        return types.SeverityLow
    case "info", "informational":
        return types.SeverityInfo
    default:
        return types.SeverityMedium
    }
}
```

### Step 4: Docker Integration

Create a Dockerfile for your agent:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o agent ./cmd/agent

FROM alpine:latest

# Install your security tool
RUN apk add --no-cache my-security-tool

# Copy the agent binary
COPY --from=builder /app/agent /usr/local/bin/agent

# Create non-root user
RUN adduser -D -s /bin/sh agentscan
USER agentscan

ENTRYPOINT ["/usr/local/bin/agent"]
```

### Step 5: Configuration

Create a configuration file:

```go
package mycustomagent

type Config struct {
    ToolPath    string            `yaml:"tool_path" env:"TOOL_PATH"`
    Timeout     time.Duration     `yaml:"timeout" env:"TIMEOUT" default:"10m"`
    MaxFileSize int64             `yaml:"max_file_size" env:"MAX_FILE_SIZE" default:"10485760"`
    Rules       map[string]bool   `yaml:"rules"`
    Options     map[string]string `yaml:"options"`
}

func LoadConfig() (*Config, error) {
    config := &Config{}
    
    // Load from environment variables
    if err := env.Parse(config); err != nil {
        return nil, fmt.Errorf("failed to parse environment variables: %w", err)
    }
    
    // Load from config file if exists
    if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
        data, err := os.ReadFile(configFile)
        if err != nil {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
        
        if err := yaml.Unmarshal(data, config); err != nil {
            return nil, fmt.Errorf("failed to parse config file: %w", err)
        }
    }
    
    return config, nil
}
```

## Testing Your Agent

### Unit Tests

```go
package mycustomagent

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyCustomAgent_Scan(t *testing.T) {
    agent := New()
    
    tests := []struct {
        name        string
        request     types.ScanRequest
        expectError bool
        expectCount int
    }{
        {
            name: "successful scan",
            request: types.ScanRequest{
                RepositoryPath: "testdata/vulnerable-app",
                Language:       "javascript",
            },
            expectError: false,
            expectCount: 3,
        },
        {
            name: "empty repository path",
            request: types.ScanRequest{
                Language: "javascript",
            },
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()
            
            result, err := agent.Scan(ctx, tt.request)
            
            if tt.expectError {
                assert.Error(t, err)
                return
            }
            
            require.NoError(t, err)
            assert.Len(t, result.Findings, tt.expectCount)
            assert.Equal(t, agent.config.Name, result.Metadata.Tool)
        })
    }
}

func TestMyCustomAgent_HealthCheck(t *testing.T) {
    agent := New()
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    err := agent.HealthCheck(ctx)
    assert.NoError(t, err)
}
```

### Integration Tests

```go
func TestMyCustomAgent_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    agent := New()
    
    // Create temporary repository with vulnerable code
    tempDir := t.TempDir()
    vulnerableCode := `
        function processInput(userInput) {
            // Vulnerable to XSS
            document.innerHTML = userInput;
        }
    `
    
    err := os.WriteFile(filepath.Join(tempDir, "app.js"), []byte(vulnerableCode), 0644)
    require.NoError(t, err)
    
    // Run scan
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    
    result, err := agent.Scan(ctx, types.ScanRequest{
        RepositoryPath: tempDir,
        Language:       "javascript",
    })
    
    require.NoError(t, err)
    assert.Greater(t, len(result.Findings), 0)
    
    // Verify finding details
    finding := result.Findings[0]
    assert.Equal(t, "app.js", finding.FilePath)
    assert.Greater(t, finding.LineNumber, 0)
    assert.NotEmpty(t, finding.Title)
    assert.NotEmpty(t, finding.Description)
}
```

## Registration and Integration

### Register Your Agent

Add your agent to the orchestrator:

```go
// In internal/orchestrator/agents.go
func registerAgents(orchestrator *Orchestrator) {
    // Existing agents
    orchestrator.RegisterAgent("semgrep", semgrep.New())
    orchestrator.RegisterAgent("eslint", eslint.New())
    
    // Your custom agent
    orchestrator.RegisterAgent("my-custom-agent", mycustomagent.New())
}
```

### Configuration

Add agent configuration to the main config:

```yaml
# config/agents.yaml
agents:
  my-custom-agent:
    enabled: true
    timeout: 10m
    max_file_size: 10MB
    languages:
      - javascript
      - typescript
    rules:
      xss-detection: true
      sql-injection: true
    options:
      strict_mode: true
      include_low_severity: false
```

## Advanced Features

### Custom Rule Management

```go
type RuleManager struct {
    rules map[string]Rule
}

type Rule struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Severity    Severity `json:"severity"`
    Languages   []string `json:"languages"`
    Pattern     string   `json:"pattern"`
    Enabled     bool     `json:"enabled"`
}

func (rm *RuleManager) LoadRules(configPath string) error {
    // Load custom rules from configuration
}

func (rm *RuleManager) ApplyRules(findings []Finding) []Finding {
    // Filter and modify findings based on custom rules
}
```

### Incremental Scanning Support

```go
func (a *MyCustomAgent) SupportsIncremental() bool {
    return true
}

func (a *MyCustomAgent) ScanIncremental(ctx context.Context, request types.IncrementalScanRequest) (*types.ScanResult, error) {
    // Only scan changed files
    var findings []types.Finding
    
    for _, changedFile := range request.ChangedFiles {
        fileFindings, err := a.scanFile(ctx, changedFile)
        if err != nil {
            return nil, err
        }
        findings = append(findings, fileFindings...)
    }
    
    return &types.ScanResult{
        Findings: findings,
        Metadata: types.Metadata{
            Tool:        a.config.Name,
            ScanType:    "incremental",
            FilesScanned: len(request.ChangedFiles),
        },
    }, nil
}
```

### Caching Support

```go
type CacheableAgent struct {
    *MyCustomAgent
    cache Cache
}

func (ca *CacheableAgent) Scan(ctx context.Context, request types.ScanRequest) (*types.ScanResult, error) {
    // Generate cache key based on file content hash and tool version
    cacheKey := ca.generateCacheKey(request)
    
    // Check cache first
    if cached, found := ca.cache.Get(cacheKey); found {
        return cached.(*types.ScanResult), nil
    }
    
    // Perform scan
    result, err := ca.MyCustomAgent.Scan(ctx, request)
    if err != nil {
        return nil, err
    }
    
    // Cache result
    ca.cache.Set(cacheKey, result, time.Hour)
    
    return result, nil
}
```

## Best Practices

### Error Handling

```go
func (a *MyCustomAgent) Scan(ctx context.Context, request types.ScanRequest) (*types.ScanResult, error) {
    // Validate inputs
    if err := a.validateRequest(request); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    // Set up timeout
    ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
    defer cancel()
    
    // Handle context cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Wrap errors with context
    result, err := a.performScan(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("scan failed for %s: %w", request.RepositoryPath, err)
    }
    
    return result, nil
}
```

### Logging

```go
import "github.com/sirupsen/logrus"

func (a *MyCustomAgent) Scan(ctx context.Context, request types.ScanRequest) (*types.ScanResult, error) {
    logger := logrus.WithFields(logrus.Fields{
        "agent":      a.config.Name,
        "repository": request.RepositoryPath,
        "language":   request.Language,
    })
    
    logger.Info("Starting security scan")
    
    startTime := time.Now()
    result, err := a.performScan(ctx, request)
    duration := time.Since(startTime)
    
    if err != nil {
        logger.WithError(err).Error("Scan failed")
        return nil, err
    }
    
    logger.WithFields(logrus.Fields{
        "findings": len(result.Findings),
        "duration": duration,
    }).Info("Scan completed successfully")
    
    return result, nil
}
```

### Resource Management

```go
func (a *MyCustomAgent) Scan(ctx context.Context, request types.ScanRequest) (*types.ScanResult, error) {
    // Limit memory usage
    if err := a.checkResourceLimits(request); err != nil {
        return nil, err
    }
    
    // Create temporary working directory
    tempDir, err := os.MkdirTemp("", "agent-scan-")
    if err != nil {
        return nil, err
    }
    defer os.RemoveAll(tempDir)
    
    // Set resource limits for child processes
    cmd := exec.CommandContext(ctx, "my-security-tool")
    cmd.Dir = tempDir
    
    // Limit CPU and memory
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }
    
    return a.executeWithLimits(cmd)
}
```

## Deployment

### Docker Compose

```yaml
version: '3.8'
services:
  agentscan-orchestrator:
    image: agentscan/orchestrator:latest
    environment:
      - AGENTS_MY_CUSTOM_AGENT_ENABLED=true
      - AGENTS_MY_CUSTOM_AGENT_IMAGE=my-org/my-custom-agent:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    
  my-custom-agent:
    image: my-org/my-custom-agent:latest
    environment:
      - TOOL_PATH=/usr/local/bin/my-security-tool
      - TIMEOUT=10m
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-custom-agent
spec:
  replicas: 2
  selector:
    matchLabels:
      app: my-custom-agent
  template:
    metadata:
      labels:
        app: my-custom-agent
    spec:
      containers:
      - name: agent
        image: my-org/my-custom-agent:latest
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        env:
        - name: TOOL_PATH
          value: "/usr/local/bin/my-security-tool"
        - name: TIMEOUT
          value: "10m"
```

## Troubleshooting

### Common Issues

1. **Tool not found**: Ensure the security tool is properly installed in the Docker image
2. **Timeout errors**: Adjust the timeout configuration for large repositories
3. **Memory issues**: Implement proper resource limits and cleanup
4. **Parsing errors**: Validate tool output format and handle edge cases

### Debugging

```go
func (a *MyCustomAgent) Scan(ctx context.Context, request types.ScanRequest) (*types.ScanResult, error) {
    if os.Getenv("DEBUG") == "true" {
        // Enable verbose logging
        logrus.SetLevel(logrus.DebugLevel)
        
        // Save raw output for debugging
        defer func() {
            if result != nil && result.RawOutput != "" {
                debugFile := filepath.Join(os.TempDir(), "agent-debug.json")
                os.WriteFile(debugFile, []byte(result.RawOutput), 0644)
                logrus.Debugf("Raw output saved to %s", debugFile)
            }
        }()
    }
    
    // ... rest of scan logic
}
```

## Contributing

To contribute your custom agent to the AgentScan project:

1. Fork the repository
2. Create your agent in the `agents/` directory
3. Add comprehensive tests
4. Update documentation
5. Submit a pull request

For more information, see the [Contributing Guide](../contributing.md).

## Examples

Check out the existing agents for reference implementations:

- [Semgrep Agent](../../agents/sast/semgrep/) - SAST scanning
- [ESLint Agent](../../agents/sast/eslint/) - JavaScript security linting
- [Bandit Agent](../../agents/sast/bandit/) - Python security scanning
- [TruffleHog Agent](../../agents/secrets/trufflehog/) - Secret detection

These examples demonstrate different approaches to tool integration, output parsing, and error handling.