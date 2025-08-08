# ESLint Security Agent

The ESLint Security Agent is a SAST (Static Application Security Testing) agent that uses ESLint with security-focused plugins to detect security vulnerabilities in JavaScript and TypeScript code.

## Overview

This agent wraps ESLint with the `eslint-plugin-security` plugin to identify common security issues in JavaScript/TypeScript applications. It focuses specifically on security-related rules and filters out non-security findings.

## Supported Languages

- JavaScript (.js, .jsx)
- TypeScript (.ts, .tsx)

## Detected Vulnerability Categories

- **Command Injection**: Detection of `eval()`, `require()` with non-literal arguments, and similar dangerous patterns
- **Cross-Site Scripting (XSS)**: Unsafe innerHTML usage, disabled template escaping
- **Cross-Site Request Forgery (CSRF)**: Missing CSRF protection patterns
- **Path Traversal**: Non-literal filesystem paths
- **Insecure Cryptography**: Weak random number generation, timing attacks
- **Misconfiguration**: Buffer usage without assertions, unsafe regex patterns

## Security Rules

### High Severity Rules
- `security/detect-eval-with-expression` - Dangerous eval() usage
- `security/detect-non-literal-require` - Dynamic require() calls
- `security/detect-object-injection` - Object injection vulnerabilities
- `security/detect-unsafe-regex` - ReDoS vulnerabilities
- `no-eval` - Direct eval() usage
- `no-implied-eval` - Implied eval() through setTimeout/setInterval
- `no-new-func` - Function constructor usage

### Medium Severity Rules
- `security/detect-child-process` - Child process execution
- `security/detect-non-literal-fs-filename` - Dynamic file paths
- `security/detect-buffer-noassert` - Buffer operations without validation
- `security/detect-disable-mustache-escape` - Template escaping disabled
- `security/detect-no-csrf-before-method-override` - CSRF protection issues

### Low Severity Rules
- `security/detect-pseudoRandomBytes` - Weak random generation
- `security/detect-possible-timing-attacks` - Timing attack vulnerabilities

## Configuration

The agent can be configured with custom settings:

```go
config := eslint.AgentConfig{
    DockerImage:    "node:18-alpine",
    MaxMemoryMB:    512,
    MaxCPUCores:    1.0,
    DefaultTimeout: 5 * time.Minute,
    SecurityRules:  []string{
        "security/detect-eval-with-expression",
        "security/detect-non-literal-require",
        // ... additional rules
    },
}

agent := eslint.NewAgentWithConfig(config)
```

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/agentscan/agentscan/agents/sast/eslint"
    "github.com/agentscan/agentscan/pkg/agent"
)

// Create agent
eslintAgent := eslint.NewAgent()

// Configure scan
config := agent.ScanConfig{
    RepoURL:   "https://github.com/user/repo",
    Branch:    "main",
    Languages: []string{"javascript", "typescript"},
    Timeout:   5 * time.Minute,
}

// Run scan
result, err := eslintAgent.Scan(context.Background(), config)
if err != nil {
    log.Fatal(err)
}

// Process findings
for _, finding := range result.Findings {
    fmt.Printf("Found %s in %s:%d - %s\n", 
        finding.Severity, finding.File, finding.Line, finding.Title)
}
```

### Health Check

```go
err := eslintAgent.HealthCheck(context.Background())
if err != nil {
    log.Printf("ESLint agent not healthy: %v", err)
}
```

## Dependencies

The agent requires:
- Docker (for containerized execution)
- Node.js image (default: `node:18-alpine`)
- ESLint and security plugins (installed automatically)

## Output Format

The agent returns findings in the standard AgentScan format:

```go
type Finding struct {
    ID          string         // Unique finding identifier
    Tool        string         // "eslint-security"
    RuleID      string         // ESLint rule ID (e.g., "security/detect-eval-with-expression")
    Severity    Severity       // High/Medium/Low
    Category    VulnCategory   // Vulnerability category
    Title       string         // Human-readable title
    Description string         // Detailed description
    File        string         // Relative file path
    Line        int            // Line number
    Column      int            // Column number
    Code        string         // Code snippet
    Fix         *FixSuggestion // Suggested fix (if available)
    Confidence  float64        // Confidence score (0.0-1.0)
    References  []string       // Documentation links
}
```

## Example Vulnerabilities Detected

### Dangerous eval() Usage
```javascript
// Detected by: security/detect-eval-with-expression, no-eval
function processUserInput(input) {
    eval("var result = " + input); // HIGH severity
    return result;
}
```

### Non-literal require()
```javascript
// Detected by: security/detect-non-literal-require
function loadModule(moduleName) {
    return require(moduleName); // HIGH severity
}
```

### Object Injection
```javascript
// Detected by: security/detect-object-injection
function updateObject(userInput) {
    var obj = {};
    obj[userInput.key] = userInput.value; // HIGH severity
    return obj;
}
```

### Unsafe innerHTML
```javascript
// Detected by: no-unsafe-innerhtml/no-unsafe-innerhtml
function displayContent(content) {
    document.getElementById('output').innerHTML = content; // MEDIUM severity
}
```

## Testing

Run the test suite:

```bash
# Unit tests
go test ./agents/sast/eslint

# Integration tests (requires Docker)
go test -v ./agents/sast/eslint -run Integration

# All tests
go test -v ./agents/sast/eslint/...
```

## Limitations

1. **Language Detection**: Only processes JavaScript/TypeScript files
2. **Docker Dependency**: Requires Docker for execution
3. **Network Access**: May need internet access to pull Node.js images
4. **Performance**: Processing time depends on codebase size and complexity
5. **False Positives**: Some patterns may be flagged incorrectly depending on context

## Contributing

When adding new security rules:

1. Add the rule to the default `SecurityRules` configuration
2. Update the severity mapping in `mapSeverity()`
3. Update the category mapping in `mapCategory()`
4. Add rule title in `getRuleTitle()`
5. Add documentation references in `getRuleReferences()`
6. Add test cases for the new rule

## References

- [ESLint Security Plugin](https://github.com/eslint-community/eslint-plugin-security)
- [ESLint Core Rules](https://eslint.org/docs/rules/)
- [OWASP JavaScript Security](https://owasp.org/www-project-top-ten/)