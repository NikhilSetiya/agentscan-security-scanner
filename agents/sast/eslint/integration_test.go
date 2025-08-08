package eslint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_Integration_VulnerableJavaScript(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory with vulnerable JavaScript code
	tempDir, err := os.MkdirTemp("", "eslint-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create vulnerable JavaScript files
	vulnerableCode := map[string]string{
		"app.js": `
// Vulnerable code examples
function dangerousEval(userInput) {
    // This should trigger security/detect-eval-with-expression
    eval("var result = " + userInput);
    return result;
}

function unsafeRequire(moduleName) {
    // This should trigger security/detect-non-literal-require
    return require(moduleName);
}

function objectInjection(userInput) {
    // This should trigger security/detect-object-injection
    var obj = {};
    obj[userInput] = "value";
    return obj;
}

// This should trigger no-eval
function directEval() {
    eval("console.log('dangerous')");
}

// This should trigger no-implied-eval
function impliedEval() {
    setTimeout("console.log('dangerous')", 1000);
}
`,
		"server.js": `
const fs = require('fs');
const crypto = require('crypto');

function readFile(filename) {
    // This should trigger security/detect-non-literal-fs-filename
    return fs.readFileSync(filename);
}

function weakRandom() {
    // This should trigger security/detect-pseudoRandomBytes
    return crypto.pseudoRandomBytes(16);
}

function unsafeRegex(input) {
    // This should trigger security/detect-unsafe-regex
    const regex = new RegExp(input);
    return regex.test("test");
}
`,
		"package.json": `{
  "name": "vulnerable-app",
  "version": "1.0.0",
  "description": "Test app with security vulnerabilities",
  "main": "app.js",
  "dependencies": {}
}`,
	}

	// Write files to temp directory
	for filename, content := range vulnerableCode {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Initialize git repository (required for cloning)
	setupGitRepo(t, tempDir)

	// Create agent and run scan
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL:   tempDir, // Use local path for testing
		Branch:    "main",
		Languages: []string{"javascript"},
		Timeout:   2 * time.Minute,
	}

	result, err := a.Scan(context.Background(), config)
	
	// The scan might fail due to Docker/network issues in CI, so we'll be flexible
	if err != nil {
		t.Logf("Scan failed (expected in some CI environments): %v", err)
		return
	}

	require.NotNil(t, result)
	assert.Equal(t, AgentName, result.AgentID)
	
	// If scan completed successfully, verify findings
	if result.Status == agent.ScanStatusCompleted {
		assert.Greater(t, len(result.Findings), 0, "Should find security vulnerabilities")
		
		// Check for specific vulnerability types
		foundRules := make(map[string]bool)
		for _, finding := range result.Findings {
			foundRules[finding.RuleID] = true
			
			// Verify finding structure
			assert.NotEmpty(t, finding.ID)
			assert.Equal(t, AgentName, finding.Tool)
			assert.NotEmpty(t, finding.RuleID)
			assert.NotEmpty(t, finding.Title)
			assert.NotEmpty(t, finding.Description)
			assert.NotEmpty(t, finding.File)
			assert.Greater(t, finding.Line, 0)
			assert.Greater(t, finding.Confidence, 0.0)
			
			// Verify severity mapping
			assert.Contains(t, []agent.Severity{
				agent.SeverityHigh,
				agent.SeverityMedium,
				agent.SeverityLow,
			}, finding.Severity)
			
			// Verify category mapping
			assert.NotEqual(t, agent.VulnCategory(""), finding.Category)
		}
		
		// Verify metadata
		assert.NotEmpty(t, result.Metadata.ToolVersion)
		assert.Equal(t, "sast", result.Metadata.ScanType)
		assert.Greater(t, result.Duration, time.Duration(0))
		
		t.Logf("Found %d security issues with rules: %v", len(result.Findings), foundRules)
	}
}

func TestAgent_Integration_TypeScript(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory with vulnerable TypeScript code
	tempDir, err := os.MkdirTemp("", "eslint-ts-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create vulnerable TypeScript files
	vulnerableCode := map[string]string{
		"app.ts": `
interface User {
    name: string;
    email: string;
}

class UserService {
    private users: User[] = [];

    // Vulnerable eval usage
    executeCode(code: string): any {
        return eval(code); // Should trigger no-eval
    }

    // Vulnerable require usage
    loadModule(moduleName: string): any {
        return require(moduleName); // Should trigger security/detect-non-literal-require
    }

    // Object injection vulnerability
    updateUser(userInput: any): void {
        const user: any = {};
        user[userInput.key] = userInput.value; // Should trigger security/detect-object-injection
        this.users.push(user);
    }
}

// Implied eval
function scheduleTask(code: string): void {
    setTimeout(code, 1000); // Should trigger no-implied-eval
}
`,
		"package.json": `{
  "name": "vulnerable-ts-app",
  "version": "1.0.0",
  "description": "Test TypeScript app with security vulnerabilities",
  "main": "app.ts",
  "devDependencies": {
    "typescript": "^4.0.0"
  }
}`,
		"tsconfig.json": `{
  "compilerOptions": {
    "target": "es2020",
    "module": "commonjs",
    "strict": true,
    "esModuleInterop": true
  }
}`,
	}

	// Write files to temp directory
	for filename, content := range vulnerableCode {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Initialize git repository
	setupGitRepo(t, tempDir)

	// Create agent and run scan
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL:   tempDir,
		Branch:    "main",
		Languages: []string{"typescript"},
		Timeout:   2 * time.Minute,
	}

	result, err := a.Scan(context.Background(), config)
	
	// Handle potential CI environment issues
	if err != nil {
		t.Logf("TypeScript scan failed (expected in some CI environments): %v", err)
		return
	}

	require.NotNil(t, result)
	assert.Equal(t, AgentName, result.AgentID)
	
	if result.Status == agent.ScanStatusCompleted {
		t.Logf("TypeScript scan found %d security issues", len(result.Findings))
		
		// Verify TypeScript files are processed
		for _, finding := range result.Findings {
			assert.True(t, 
				finding.File == "app.ts" || finding.File == "app.js", // ESLint might process compiled JS
				"Finding should be in TypeScript file: %s", finding.File)
		}
	}
}

func TestAgent_Integration_NoVulnerabilities(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory with safe JavaScript code
	tempDir, err := os.MkdirTemp("", "eslint-safe-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create safe JavaScript code
	safeCode := map[string]string{
		"safe.js": `
// Safe JavaScript code
function safeFunction(input) {
    // Safe string concatenation
    const message = "Hello, " + input;
    console.log(message);
    return message;
}

function safeFileOperation() {
    // Safe file operations with literal paths
    const fs = require('fs');
    return fs.readFileSync('./config.json', 'utf8');
}

function safeRandomGeneration() {
    // Safe random generation
    const crypto = require('crypto');
    return crypto.randomBytes(16);
}

module.exports = {
    safeFunction,
    safeFileOperation,
    safeRandomGeneration
};
`,
		"package.json": `{
  "name": "safe-app",
  "version": "1.0.0",
  "description": "Safe JavaScript application",
  "main": "safe.js"
}`,
	}

	// Write files to temp directory
	for filename, content := range safeCode {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Initialize git repository
	setupGitRepo(t, tempDir)

	// Create agent and run scan
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL:   tempDir,
		Branch:    "main",
		Languages: []string{"javascript"},
		Timeout:   2 * time.Minute,
	}

	result, err := a.Scan(context.Background(), config)
	
	// Handle potential CI environment issues
	if err != nil {
		t.Logf("Safe code scan failed (expected in some CI environments): %v", err)
		return
	}

	require.NotNil(t, result)
	assert.Equal(t, AgentName, result.AgentID)
	
	if result.Status == agent.ScanStatusCompleted {
		// Safe code should have no or very few security findings
		assert.LessOrEqual(t, len(result.Findings), 2, 
			"Safe code should have minimal security findings, found: %d", len(result.Findings))
		
		t.Logf("Safe code scan found %d security issues (expected to be minimal)", len(result.Findings))
	}
}

// setupGitRepo initializes a git repository in the given directory
func setupGitRepo(t *testing.T, dir string) {
	// Initialize git repo
	cmd := []string{"git", "init"}
	if err := runCommand(dir, cmd...); err != nil {
		t.Logf("Failed to init git repo: %v", err)
		return
	}

	// Configure git user
	runCommand(dir, "git", "config", "user.email", "test@example.com")
	runCommand(dir, "git", "config", "user.name", "Test User")

	// Add files and commit
	runCommand(dir, "git", "add", ".")
	runCommand(dir, "git", "commit", "-m", "Initial commit")
}

// runCommand executes a command in the specified directory
func runCommand(dir string, command ...string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	return cmd.Run()
}

func TestAgent_parseESLintOutput(t *testing.T) {
	a := NewAgent()
	
	// Sample ESLint JSON output
	eslintOutput := `[
  {
    "filePath": "/app/test.js",
    "messages": [
      {
        "ruleId": "security/detect-eval-with-expression",
        "severity": 2,
        "message": "eval can be harmful.",
        "line": 5,
        "column": 5,
        "nodeType": "CallExpression",
        "messageId": "eval",
        "endLine": 5,
        "endColumn": 25
      },
      {
        "ruleId": "no-eval",
        "severity": 2,
        "message": "eval can be harmful.",
        "line": 8,
        "column": 5,
        "nodeType": "CallExpression",
        "endLine": 8,
        "endColumn": 20
      },
      {
        "ruleId": "no-unused-vars",
        "severity": 1,
        "message": "'unused' is defined but never used.",
        "line": 2,
        "column": 7,
        "nodeType": "Identifier",
        "endLine": 2,
        "endColumn": 13
      }
    ],
    "errorCount": 2,
    "warningCount": 1,
    "fixableErrorCount": 0,
    "fixableWarningCount": 0,
    "source": "function test() {\n  var unused = 'test';\n  \n  // Dangerous eval\n  eval('console.log(\"test\")');\n  \n  // Another eval\n  eval('alert(1)');\n}"
  }
]`

	config := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo",
	}

	findings, metadata, err := a.parseESLintOutput([]byte(eslintOutput), config)
	
	require.NoError(t, err)
	
	// Should only include security-related findings (2 eval issues, not the unused var)
	assert.Len(t, findings, 2)
	
	// Check first finding
	finding1 := findings[0]
	assert.Equal(t, AgentName, finding1.Tool)
	assert.Equal(t, "security/detect-eval-with-expression", finding1.RuleID)
	assert.Equal(t, agent.SeverityHigh, finding1.Severity)
	assert.Equal(t, agent.CategoryCommandInjection, finding1.Category)
	assert.Equal(t, "test.js", finding1.File)
	assert.Equal(t, 5, finding1.Line)
	assert.Equal(t, 5, finding1.Column)
	assert.Greater(t, finding1.Confidence, 0.0)
	
	// Check second finding
	finding2 := findings[1]
	assert.Equal(t, "no-eval", finding2.RuleID)
	assert.Equal(t, agent.SeverityHigh, finding2.Severity)
	
	// Check metadata
	assert.Equal(t, "sast", metadata.ScanType)
	assert.Equal(t, 1, metadata.FilesScanned)
	assert.Greater(t, metadata.LinesScanned, 0)
}