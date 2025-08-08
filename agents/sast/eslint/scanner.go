package eslint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/agentscan/agentscan/pkg/agent"
)

// executeScan runs the actual ESLint security scan
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "eslint-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone or copy repository to temp directory
	repoPath := filepath.Join(tempDir, "repo")
	if err := a.prepareRepository(ctx, config, repoPath); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Setup ESLint configuration
	if err := a.setupESLintConfig(repoPath); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to setup ESLint config: %w", err)
	}

	// Build ESLint command
	cmd := a.buildESLintCommand(ctx, config, repoPath, tempDir)

	// Execute ESLint
	output, err := cmd.Output()
	if err != nil {
		// ESLint returns non-zero exit code when findings are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means findings were found, which is not an error
			// Exit code 2 means configuration error
			if exitErr.ExitCode() == 1 {
				output = exitErr.Stderr
				if len(output) == 0 {
					// Sometimes output goes to stdout even with errors
					output, _ = cmd.Output()
				}
			} else {
				return nil, agent.Metadata{}, fmt.Errorf("eslint execution failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
			}
		} else {
			return nil, agent.Metadata{}, fmt.Errorf("eslint execution failed: %w", err)
		}
	}

	// Parse JSON output
	findings, metadata, err := a.parseESLintOutput(output, config)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse eslint output: %w", err)
	}

	return findings, metadata, nil
}

// prepareRepository clones or copies the repository to the specified path
func (a *Agent) prepareRepository(ctx context.Context, config agent.ScanConfig, repoPath string) error {
	var cmd *exec.Cmd
	if config.Commit != "" {
		// Clone specific commit
		cmd = exec.CommandContext(ctx, "git", "clone", "--depth", "1", config.RepoURL, repoPath)
	} else {
		// Clone specific branch or default
		branch := config.Branch
		if branch == "" {
			branch = "main"
		}
		cmd = exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, config.RepoURL, repoPath)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// If specific commit is requested, checkout that commit
	if config.Commit != "" {
		checkoutCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", config.Commit)
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("git checkout failed: %w", err)
		}
	}

	return nil
}

// setupESLintConfig creates the ESLint configuration for security scanning
func (a *Agent) setupESLintConfig(repoPath string) error {
	// Check if there's already an ESLint config
	configFiles := []string{".eslintrc.js", ".eslintrc.json", ".eslintrc.yml", ".eslintrc.yaml", "eslint.config.js"}
	hasConfig := false
	
	for _, configFile := range configFiles {
		if _, err := os.Stat(filepath.Join(repoPath, configFile)); err == nil {
			hasConfig = true
			break
		}
	}

	// Create package.json if it doesn't exist
	packageJsonPath := filepath.Join(repoPath, "package.json")
	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		packageJson := `{
  "name": "security-scan",
  "version": "1.0.0",
  "private": true
}`
		if err := os.WriteFile(packageJsonPath, []byte(packageJson), 0644); err != nil {
			return fmt.Errorf("failed to create package.json: %w", err)
		}
	}

	// If no config exists, create a security-focused one
	if !hasConfig {
		eslintConfig := a.generateESLintConfig()
		configPath := filepath.Join(repoPath, ".eslintrc.json")
		if err := os.WriteFile(configPath, []byte(eslintConfig), 0644); err != nil {
			return fmt.Errorf("failed to create ESLint config: %w", err)
		}
	}

	return nil
}

// generateESLintConfig creates a security-focused ESLint configuration
func (a *Agent) generateESLintConfig() string {
	config := map[string]interface{}{
		"env": map[string]bool{
			"browser": true,
			"node":    true,
			"es2021":  true,
		},
		"extends": []string{
			"eslint:recommended",
		},
		"plugins": []string{
			"security",
		},
		"parserOptions": map[string]interface{}{
			"ecmaVersion": "latest",
			"sourceType":  "module",
		},
		"rules": map[string]interface{}{},
	}

	// Add security rules
	rules := config["rules"].(map[string]interface{})
	for _, rule := range a.config.SecurityRules {
		rules[rule] = "error"
	}

	// Add additional security-related ESLint rules
	securityRules := map[string]string{
		"no-eval":                    "error",
		"no-implied-eval":            "error",
		"no-new-func":                "error",
		"no-script-url":              "error",
		"no-unsafe-innerhtml/no-unsafe-innerhtml": "error",
	}

	for rule, level := range securityRules {
		rules[rule] = level
	}

	configBytes, _ := json.MarshalIndent(config, "", "  ")
	return string(configBytes)
}

// buildESLintCommand constructs the Docker command to run ESLint
func (a *Agent) buildESLintCommand(ctx context.Context, config agent.ScanConfig, repoPath, tempDir string) *exec.Cmd {
	// Install dependencies and run ESLint in a single command
	installAndRunScript := `#!/bin/sh
set -e

# Install ESLint and security plugin
npm install --no-save eslint@latest eslint-plugin-security@latest eslint-plugin-no-unsafe-innerhtml@latest

# Create output directory
mkdir -p /tmp/eslint

# Run ESLint with JSON output
npx eslint . --format json --output-file /tmp/eslint/results.json --ext .js,.jsx,.ts,.tsx --no-error-on-unmatched-pattern || true

# Also output to stdout for debugging
cat /tmp/eslint/results.json 2>/dev/null || echo "[]"
`

	scriptPath := filepath.Join(tempDir, "run-eslint.sh")
	os.WriteFile(scriptPath, []byte(installAndRunScript), 0755)

	args := []string{
		"run", "--rm",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		"-v", fmt.Sprintf("%s:/app:ro", repoPath),
		"-v", fmt.Sprintf("%s:/tmp/eslint", tempDir),
		"-v", fmt.Sprintf("%s:/run-eslint.sh:ro", scriptPath),
		"-w", "/app",
		a.config.DockerImage,
		"sh", "/run-eslint.sh",
	}

	return exec.CommandContext(ctx, "docker", args...)
}

// ESLintResult represents the structure of ESLint JSON output
type ESLintResult struct {
	FilePath    string `json:"filePath"`
	Messages    []ESLintMessage `json:"messages"`
	ErrorCount  int    `json:"errorCount"`
	WarningCount int   `json:"warningCount"`
	FixableErrorCount   int `json:"fixableErrorCount"`
	FixableWarningCount int `json:"fixableWarningCount"`
	Source      string `json:"source,omitempty"`
}

type ESLintMessage struct {
	RuleID    string `json:"ruleId"`
	Severity  int    `json:"severity"` // 1 = warning, 2 = error
	Message   string `json:"message"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	NodeType  string `json:"nodeType,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
	Fix       *struct {
		Range []int  `json:"range"`
		Text  string `json:"text"`
	} `json:"fix,omitempty"`
}

// parseESLintOutput parses the JSON output from ESLint and converts it to our Finding format
func (a *Agent) parseESLintOutput(output []byte, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	var eslintResults []ESLintResult
	if err := json.Unmarshal(output, &eslintResults); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse ESLint JSON: %w", err)
	}

	var findings []agent.Finding
	var filesScanned int
	var linesScanned int
	var totalErrors int
	var totalWarnings int

	for _, result := range eslintResults {
		if len(result.Messages) == 0 {
			continue
		}

		filesScanned++
		totalErrors += result.ErrorCount
		totalWarnings += result.WarningCount

		for _, message := range result.Messages {
			// Only include security-related rules
			if !a.isSecurityRule(message.RuleID) {
				continue
			}

			finding := agent.Finding{
				ID:          generateFindingID(message.RuleID, result.FilePath, message.Line),
				Tool:        AgentName,
				RuleID:      message.RuleID,
				Severity:    a.mapSeverity(message.Severity, message.RuleID),
				Category:    a.mapCategory(message.RuleID),
				Title:       a.getRuleTitle(message.RuleID),
				Description: message.Message,
				File:        strings.TrimPrefix(result.FilePath, "/app/"),
				Line:        message.Line,
				Column:      message.Column,
				Code:        a.extractCodeSnippet(result.Source, message.Line),
				Confidence:  a.getConfidence(message.RuleID),
				References:  a.getRuleReferences(message.RuleID),
			}

			// Add fix suggestion if available
			if message.Fix != nil {
				finding.Fix = &agent.FixSuggestion{
					Description: fmt.Sprintf("Replace with: %s", message.Fix.Text),
					Code:        message.Fix.Text,
				}
			}

			findings = append(findings, finding)
			linesScanned += message.EndLine - message.Line + 1
		}
	}

	metadata := agent.Metadata{
		ToolVersion:  a.getToolVersion(),
		RulesVersion: "latest",
		ScanType:     "sast",
		FilesScanned: filesScanned,
		LinesScanned: linesScanned,
		ExitCode:     0,
		CommandLine:  "eslint --format json",
		Environment: map[string]string{
			"errors":   fmt.Sprintf("%d", totalErrors),
			"warnings": fmt.Sprintf("%d", totalWarnings),
		},
	}

	return findings, metadata, nil
}

// generateFindingID creates a unique ID for a finding
func generateFindingID(ruleID, file string, line int) string {
	return fmt.Sprintf("eslint-%s-%s-%d", ruleID, filepath.Base(file), line)
}

// isSecurityRule checks if the rule is security-related
func (a *Agent) isSecurityRule(ruleID string) bool {
	if ruleID == "" {
		return false
	}

	// Security plugin rules
	if strings.HasPrefix(ruleID, "security/") {
		return true
	}

	// Core ESLint security-related rules
	securityRules := map[string]bool{
		"no-eval":                    true,
		"no-implied-eval":            true,
		"no-new-func":                true,
		"no-script-url":              true,
		"no-unsafe-innerhtml/no-unsafe-innerhtml": true,
	}

	return securityRules[ruleID]
}

// mapSeverity converts ESLint severity to our standard severity levels
func (a *Agent) mapSeverity(eslintSeverity int, ruleID string) agent.Severity {
	// High severity rules
	highSeverityRules := map[string]bool{
		"security/detect-eval-with-expression":     true,
		"security/detect-non-literal-require":     true,
		"security/detect-object-injection":        true,
		"security/detect-unsafe-regex":            true,
		"no-eval":                                 true,
		"no-implied-eval":                         true,
		"no-new-func":                             true,
	}

	if highSeverityRules[ruleID] {
		return agent.SeverityHigh
	}

	// Map based on ESLint severity
	switch eslintSeverity {
	case 2: // error
		return agent.SeverityMedium
	case 1: // warning
		return agent.SeverityLow
	default:
		return agent.SeverityLow
	}
}

// mapCategory converts ESLint rule IDs to our standard vulnerability categories
func (a *Agent) mapCategory(ruleID string) agent.VulnCategory {
	categoryMap := map[string]agent.VulnCategory{
		"security/detect-eval-with-expression":     agent.CategoryCommandInjection,
		"security/detect-non-literal-require":     agent.CategoryCommandInjection,
		"security/detect-object-injection":        agent.CategoryCommandInjection,
		"security/detect-child-process":           agent.CategoryCommandInjection,
		"security/detect-non-literal-fs-filename": agent.CategoryPathTraversal,
		"security/detect-unsafe-regex":            agent.CategoryOther,
		"security/detect-buffer-noassert":         agent.CategoryMisconfiguration,
		"security/detect-disable-mustache-escape": agent.CategoryXSS,
		"security/detect-no-csrf-before-method-override": agent.CategoryCSRF,
		"security/detect-pseudoRandomBytes":       agent.CategoryInsecureCrypto,
		"security/detect-possible-timing-attacks": agent.CategoryInsecureCrypto,
		"no-eval":                                 agent.CategoryCommandInjection,
		"no-implied-eval":                         agent.CategoryCommandInjection,
		"no-new-func":                             agent.CategoryCommandInjection,
		"no-script-url":                           agent.CategoryXSS,
		"no-unsafe-innerhtml/no-unsafe-innerhtml": agent.CategoryXSS,
	}

	if category, exists := categoryMap[ruleID]; exists {
		return category
	}

	return agent.CategoryOther
}

// getRuleTitle returns a human-readable title for the rule
func (a *Agent) getRuleTitle(ruleID string) string {
	titleMap := map[string]string{
		"security/detect-eval-with-expression":     "Dangerous eval() usage detected",
		"security/detect-non-literal-require":     "Non-literal require() detected",
		"security/detect-object-injection":        "Potential object injection vulnerability",
		"security/detect-child-process":           "Child process usage detected",
		"security/detect-non-literal-fs-filename": "Non-literal filesystem path detected",
		"security/detect-unsafe-regex":            "Potentially unsafe regular expression",
		"security/detect-buffer-noassert":         "Buffer usage without assertion",
		"security/detect-disable-mustache-escape": "Mustache template escaping disabled",
		"security/detect-no-csrf-before-method-override": "Missing CSRF protection",
		"security/detect-pseudoRandomBytes":       "Insecure random number generation",
		"security/detect-possible-timing-attacks": "Potential timing attack vulnerability",
		"no-eval":                                 "Use of eval() is prohibited",
		"no-implied-eval":                         "Implied eval() usage detected",
		"no-new-func":                             "Function constructor usage detected",
		"no-script-url":                           "Script URL detected",
		"no-unsafe-innerhtml/no-unsafe-innerhtml": "Unsafe innerHTML usage detected",
	}

	if title, exists := titleMap[ruleID]; exists {
		return title
	}

	return fmt.Sprintf("Security issue: %s", ruleID)
}

// getConfidence returns confidence level for the rule
func (a *Agent) getConfidence(ruleID string) float64 {
	// High confidence rules
	highConfidenceRules := map[string]bool{
		"no-eval":                                 true,
		"no-implied-eval":                         true,
		"no-new-func":                             true,
		"security/detect-eval-with-expression":   true,
	}

	if highConfidenceRules[ruleID] {
		return 0.9
	}

	// Medium confidence for most security rules
	if strings.HasPrefix(ruleID, "security/") {
		return 0.7
	}

	return 0.6
}

// getRuleReferences returns documentation references for the rule
func (a *Agent) getRuleReferences(ruleID string) []string {
	baseURL := "https://github.com/eslint-community/eslint-plugin-security/blob/main/docs/rules/"
	
	if strings.HasPrefix(ruleID, "security/") {
		ruleName := strings.TrimPrefix(ruleID, "security/")
		return []string{
			fmt.Sprintf("%s%s.md", baseURL, ruleName),
		}
	}

	// Core ESLint rules
	coreRules := map[string]string{
		"no-eval":        "https://eslint.org/docs/rules/no-eval",
		"no-implied-eval": "https://eslint.org/docs/rules/no-implied-eval",
		"no-new-func":    "https://eslint.org/docs/rules/no-new-func",
		"no-script-url":  "https://eslint.org/docs/rules/no-script-url",
	}

	if url, exists := coreRules[ruleID]; exists {
		return []string{url}
	}

	return []string{}
}

// extractCodeSnippet extracts a code snippet around the specified line
func (a *Agent) extractCodeSnippet(source string, line int) string {
	if source == "" {
		return ""
	}

	lines := strings.Split(source, "\n")
	if line <= 0 || line > len(lines) {
		return ""
	}

	// Return the specific line (1-indexed)
	return strings.TrimSpace(lines[line-1])
}