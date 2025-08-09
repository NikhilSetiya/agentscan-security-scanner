package gitsecrets

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/agentscan/agentscan/pkg/agent"
)

// GitSecretsResult represents a finding from git-secrets
type GitSecretsResult struct {
	File        string
	Line        int
	Column      int
	Content     string
	Pattern     string
	PatternType string
}

// executeScan runs git-secrets and parses the results
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "git-secrets-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone the repository
	if err := a.cloneRepository(ctx, config.RepoURL, config.Branch, tempDir); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Initialize git-secrets in the repository
	if err := a.initializeGitSecrets(ctx, tempDir); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to initialize git-secrets: %w", err)
	}

	// Run git-secrets scan
	results, err := a.runGitSecretsScans(ctx, tempDir, config)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("git-secrets scan failed: %w", err)
	}

	// Convert results to findings
	findings := a.convertResultsToFindings(results, config)

	// Filter findings based on whitelist
	filteredFindings := a.filterFindings(findings)

	metadata := agent.Metadata{
		ToolVersion:  a.getToolVersion(),
		ScanType:     "secrets",
		FilesScanned: len(config.Files),
		CommandLine:  "git secrets --scan",
	}

	return filteredFindings, metadata, nil
}

// cloneRepository clones the repository to a temporary directory
func (a *Agent) cloneRepository(ctx context.Context, repoURL, branch, targetDir string) error {
	args := []string{"clone"}
	
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	
	args = append(args, repoURL, targetDir)
	
	cmd := exec.CommandContext(ctx, "git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	
	return nil
}

// initializeGitSecrets initializes git-secrets in the repository
func (a *Agent) initializeGitSecrets(ctx context.Context, repoPath string) error {
	// Build Docker command to initialize git-secrets
	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/repo", repoPath),
		"-w", "/repo",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		a.config.DockerImage,
		"sh", "-c",
	}

	// Initialize git-secrets and install patterns
	initScript := `
		git secrets --install --force &&
		git secrets --register-aws &&
		git secrets --install-hooks --force
	`

	// Add provider patterns
	for _, provider := range a.config.ProviderPatterns {
		switch provider {
		case "aws":
			initScript += " && git secrets --register-aws"
		case "azure":
			initScript += " && git secrets --add 'DefaultEndpointsProtocol=https;AccountName=[^;]+;AccountKey=[^;]+'"
			initScript += " && git secrets --add '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'"
		case "gcp":
			initScript += " && git secrets --add '\"type\": \"service_account\"'"
			initScript += " && git secrets --add '\"private_key_id\":'"
		}
	}

	// Add custom patterns
	for _, pattern := range a.config.CustomPatterns {
		initScript += fmt.Sprintf(" && git secrets --add '%s'", pattern)
	}

	args = append(args, initScript)

	cmd := exec.CommandContext(ctx, "docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git-secrets initialization failed: %w", err)
	}

	return nil
}

// runGitSecretsScans runs various git-secrets scan commands
func (a *Agent) runGitSecretsScans(ctx context.Context, repoPath string, config agent.ScanConfig) ([]GitSecretsResult, error) {
	var allResults []GitSecretsResult

	// Scan working directory
	workingDirResults, err := a.scanWorkingDirectory(ctx, repoPath, config)
	if err != nil {
		// Don't fail completely if working directory scan fails
		fmt.Printf("Warning: working directory scan failed: %v\n", err)
	} else {
		allResults = append(allResults, workingDirResults...)
	}

	// Scan commit history if enabled
	if a.config.ScanCommits {
		commitResults, err := a.scanCommitHistory(ctx, repoPath)
		if err != nil {
			// Don't fail completely if commit history scan fails
			fmt.Printf("Warning: commit history scan failed: %v\n", err)
		} else {
			allResults = append(allResults, commitResults...)
		}
	}

	return allResults, nil
}

// scanWorkingDirectory scans the current working directory
func (a *Agent) scanWorkingDirectory(ctx context.Context, repoPath string, config agent.ScanConfig) ([]GitSecretsResult, error) {
	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/repo", repoPath),
		"-w", "/repo",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		a.config.DockerImage,
		"git", "secrets", "--scan",
	}

	// Add specific files if this is an incremental scan
	if len(config.Files) > 0 {
		for _, file := range config.Files {
			args = append(args, file)
		}
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	
	// git-secrets returns non-zero exit code when secrets are found
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 means secrets were found, which is expected
		} else {
			return nil, fmt.Errorf("git-secrets scan failed: %w", err)
		}
	}

	return a.parseGitSecretsOutput(string(output), "working-directory")
}

// scanCommitHistory scans the git commit history
func (a *Agent) scanCommitHistory(ctx context.Context, repoPath string) ([]GitSecretsResult, error) {
	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/repo", repoPath),
		"-w", "/repo",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		a.config.DockerImage,
		"git", "secrets", "--scan-history",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	
	// git-secrets returns non-zero exit code when secrets are found
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 means secrets were found, which is expected
		} else {
			return nil, fmt.Errorf("git-secrets history scan failed: %w", err)
		}
	}

	return a.parseGitSecretsOutput(string(output), "commit-history")
}

// parseGitSecretsOutput parses the output from git-secrets
func (a *Agent) parseGitSecretsOutput(output, scanType string) ([]GitSecretsResult, error) {
	var results []GitSecretsResult
	
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse git-secrets output format: filename:line:column:content
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}

		lineNum, err := strconv.Atoi(parts[1])
		if err != nil {
			lineNum = 0
		}

		colNum, err := strconv.Atoi(parts[2])
		if err != nil {
			colNum = 0
		}

		result := GitSecretsResult{
			File:        parts[0],
			Line:        lineNum,
			Column:      colNum,
			Content:     parts[3],
			PatternType: scanType,
		}

		results = append(results, result)
	}

	return results, nil
}

// convertResultsToFindings converts git-secrets results to agent findings
func (a *Agent) convertResultsToFindings(results []GitSecretsResult, config agent.ScanConfig) []agent.Finding {
	var findings []agent.Finding

	for i, result := range results {
		finding := agent.Finding{
			ID:          fmt.Sprintf("git-secrets-%d", i+1),
			Tool:        AgentName,
			RuleID:      fmt.Sprintf("git-secrets-%s", a.detectSecretType(result.Content)),
			Severity:    agent.SeverityHigh, // All secrets are high severity as per requirements
			Category:    agent.CategoryHardcodedSecrets,
			Title:       fmt.Sprintf("Secret Pattern Detected (%s)", result.PatternType),
			Description: a.generateDescription(result),
			File:        result.File,
			Line:        result.Line,
			Column:      result.Column,
			Code:        a.redactSecret(result.Content),
			Confidence:  a.calculateConfidence(result),
			References: []string{
				"https://github.com/awslabs/git-secrets",
				"https://owasp.org/www-project-top-ten/2017/A3_2017-Sensitive_Data_Exposure",
			},
		}

		// Add fix suggestion
		finding.Fix = &agent.FixSuggestion{
			Description: "Remove the hardcoded secret and use environment variables or a secure secret management system",
			Suggestion:  a.generateFixSuggestion(result),
			References: []string{
				"https://owasp.org/www-community/vulnerabilities/Use_of_hard-coded_password",
				"https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html",
			},
		}

		findings = append(findings, finding)
	}

	return findings
}

// detectSecretType attempts to detect the type of secret based on content
func (a *Agent) detectSecretType(content string) string {
	content = strings.ToLower(content)
	
	switch {
	case strings.Contains(content, "aws_access_key_id") || strings.Contains(content, "akia"):
		return "aws-access-key"
	case strings.Contains(content, "aws_secret_access_key"):
		return "aws-secret-key"
	case strings.Contains(content, "-----begin private key-----") || strings.Contains(content, "private_key"):
		return "private-key"
	case strings.Contains(content, "password"):
		return "password"
	case strings.Contains(content, "token"):
		return "token"
	case strings.Contains(content, "api_key") || strings.Contains(content, "apikey"):
		return "api-key"
	case strings.Contains(content, "secret"):
		return "secret"
	default:
		return "unknown"
	}
}

// generateDescription generates a description for the finding
func (a *Agent) generateDescription(result GitSecretsResult) string {
	secretType := a.detectSecretType(result.Content)
	
	description := fmt.Sprintf("git-secrets detected a potential %s in %s", secretType, result.PatternType)
	
	if result.PatternType == "commit-history" {
		description += ". This secret was found in the git commit history and may have been exposed in previous commits."
	} else {
		description += ". This secret is present in the current working directory."
	}
	
	return description
}

// redactSecret redacts sensitive parts of the secret for safe display
func (a *Agent) redactSecret(content string) string {
	// Simple redaction - replace middle characters with asterisks
	if len(content) <= 8 {
		return strings.Repeat("*", len(content))
	}
	
	start := content[:4]
	end := content[len(content)-4:]
	middle := strings.Repeat("*", len(content)-8)
	
	return start + middle + end
}

// calculateConfidence calculates confidence score for the finding
func (a *Agent) calculateConfidence(result GitSecretsResult) float64 {
	// Base confidence for git-secrets findings
	confidence := 0.8
	
	// Increase confidence for certain patterns
	content := strings.ToLower(result.Content)
	
	// Decrease confidence for generic patterns first
	if strings.Contains(content, "example") ||
	   strings.Contains(content, "test") ||
	   strings.Contains(content, "dummy") ||
	   strings.Contains(content, "placeholder") {
		return 0.6
	}
	
	// Increase confidence for high-confidence patterns
	if strings.Contains(content, "akia") || // AWS access key pattern
	   strings.Contains(content, "-----begin") || // Private key pattern
	   strings.Contains(content, "ghp_") || // GitHub personal access token
	   strings.Contains(content, "glpat-") || // GitLab personal access token
	   (len(content) >= 20 && strings.HasPrefix(content, "akia")) { // AWS access key starting with AKIA
		confidence = 0.9
	}
	
	return confidence
}

// generateFixSuggestion generates a fix suggestion based on the secret type
func (a *Agent) generateFixSuggestion(result GitSecretsResult) string {
	secretType := a.detectSecretType(result.Content)
	
	switch secretType {
	case "aws-access-key", "aws-secret-key":
		return "Use AWS IAM roles, AWS Secrets Manager, or environment variables instead of hardcoded AWS credentials"
	case "private-key":
		return "Store private keys securely using a key management service and never commit them to version control"
	case "api-key":
		return "Store API keys in environment variables or a secure secret management system"
	case "token":
		return "Use environment variables for tokens and ensure they are properly rotated"
	case "password":
		return "Use environment variables or a secure configuration management system for passwords"
	default:
		return "Remove the hardcoded secret and use environment variables or a secure secret management system"
	}
}

// filterFindings filters out findings that match the whitelist patterns
func (a *Agent) filterFindings(findings []agent.Finding) []agent.Finding {
	if len(a.config.Whitelist) == 0 {
		return findings
	}
	
	var filtered []agent.Finding
	
	for _, finding := range findings {
		shouldInclude := true
		
		for _, pattern := range a.config.Whitelist {
			// Compile regex pattern
			regex, err := regexp.Compile(pattern)
			if err != nil {
				// Skip invalid regex patterns
				continue
			}
			
			// Check if the finding matches the whitelist pattern
			if regex.MatchString(finding.File) || 
			   regex.MatchString(finding.Code) || 
			   regex.MatchString(finding.Description) {
				shouldInclude = false
				break
			}
		}
		
		if shouldInclude {
			filtered = append(filtered, finding)
		}
	}
	
	return filtered
}