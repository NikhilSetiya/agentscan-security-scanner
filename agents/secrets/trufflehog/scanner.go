package trufflehog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

// TruffleHogResult represents the JSON output from TruffleHog
type TruffleHogResult struct {
	SourceMetadata SourceMetadata `json:"SourceMetadata"`
	SourceID       int            `json:"SourceID"`
	SourceType     int            `json:"SourceType"`
	SourceName     string         `json:"SourceName"`
	DetectorType   int            `json:"DetectorType"`
	DetectorName   string         `json:"DetectorName"`
	DecoderName    string         `json:"DecoderName"`
	Verified       bool           `json:"Verified"`
	Raw            string         `json:"Raw"`
	Redacted       string         `json:"Redacted"`
	ExtraData      map[string]interface{} `json:"ExtraData"`
}

// SourceMetadata contains metadata about where the secret was found
type SourceMetadata struct {
	Data struct {
		Git struct {
			Commit     string `json:"commit"`
			File       string `json:"file"`
			Email      string `json:"email"`
			Repository string `json:"repository"`
			Timestamp  string `json:"timestamp"`
			Line       int    `json:"line"`
		} `json:"Git"`
	} `json:"Data"`
}

// executeScan runs TruffleHog and parses the results
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "trufflehog-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone the repository
	if err := a.cloneRepository(ctx, config.RepoURL, config.Branch, tempDir); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Build TruffleHog command
	args := a.buildTruffleHogArgs(tempDir, config)
	
	// Execute TruffleHog
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.Output()
	if err != nil {
		// TruffleHog returns non-zero exit code when secrets are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
			if len(output) == 0 {
				output, _ = cmd.Output()
			}
		} else {
			return nil, agent.Metadata{}, fmt.Errorf("trufflehog execution failed: %w", err)
		}
	}

	// Parse results
	findings, err := a.parseResults(output, config)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse results: %w", err)
	}

	// Filter findings based on whitelist
	filteredFindings := a.filterFindings(findings)

	metadata := agent.Metadata{
		ToolVersion:  a.getToolVersion(),
		ScanType:     "secrets",
		FilesScanned: len(config.Files),
		CommandLine:  strings.Join(args, " "),
	}

	return filteredFindings, metadata, nil
}

// cloneRepository clones the repository to a temporary directory
func (a *Agent) cloneRepository(ctx context.Context, repoURL, branch, targetDir string) error {
	args := []string{"clone", "--depth", fmt.Sprintf("%d", a.config.MaxDepth)}
	
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

// buildTruffleHogArgs constructs the command line arguments for TruffleHog
func (a *Agent) buildTruffleHogArgs(repoPath string, config agent.ScanConfig) []string {
	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/repo", repoPath),
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		a.config.DockerImage,
		"git",
		"file:///repo",
		"--json",
		"--no-update",
	}

	// Add detector filters if specified
	if len(a.config.IncludeDetectors) > 0 {
		args = append(args, "--include-detectors", strings.Join(a.config.IncludeDetectors, ","))
	}
	
	if len(a.config.ExcludeDetectors) > 0 {
		args = append(args, "--exclude-detectors", strings.Join(a.config.ExcludeDetectors, ","))
	}

	// Add specific files if this is an incremental scan
	if len(config.Files) > 0 {
		// TruffleHog doesn't support file filtering directly, but we'll filter results
		// For now, scan the entire repo and filter later
	}

	return args
}

// parseResults parses TruffleHog JSON output into findings
func (a *Agent) parseResults(output []byte, config agent.ScanConfig) ([]agent.Finding, error) {
	var findings []agent.Finding
	
	// Split output by lines as TruffleHog outputs one JSON object per line
	lines := strings.Split(string(output), "\n")
	
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		var result TruffleHogResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			// Skip malformed JSON lines
			continue
		}
		
		finding := a.convertToFinding(result, lineNum+1)
		if finding != nil {
			findings = append(findings, *finding)
		}
	}
	
	return findings, nil
}

// convertToFinding converts a TruffleHog result to an agent.Finding
func (a *Agent) convertToFinding(result TruffleHogResult, id int) *agent.Finding {
	// Determine severity based on verification status and detector type
	severity := a.determineSeverity(result)
	
	// Create finding ID
	findingID := fmt.Sprintf("trufflehog-%d", id)
	
	// Get file path relative to repo root
	filePath := result.SourceMetadata.Data.Git.File
	if filePath == "" {
		filePath = "unknown"
	}
	
	// Create description
	description := fmt.Sprintf("Secret detected by %s detector", result.DetectorName)
	if result.Verified {
		description += " (VERIFIED - This secret is valid and active)"
	} else {
		description += " (Unverified - This secret may be inactive or invalid)"
	}
	
	// Create title
	title := fmt.Sprintf("%s Secret Found", result.DetectorName)
	
	finding := &agent.Finding{
		ID:          findingID,
		Tool:        AgentName,
		RuleID:      fmt.Sprintf("trufflehog-%s", strings.ToLower(result.DetectorName)),
		Severity:    severity,
		Category:    agent.CategoryHardcodedSecrets,
		Title:       title,
		Description: description,
		File:        filePath,
		Line:        result.SourceMetadata.Data.Git.Line,
		Code:        result.Redacted, // Use redacted version for safety
		Confidence:  a.calculateConfidence(result),
		References:  []string{
			"https://github.com/trufflesecurity/trufflehog",
			"https://owasp.org/www-project-top-ten/2017/A3_2017-Sensitive_Data_Exposure",
		},
	}
	
	// Add fix suggestion
	finding.Fix = &agent.FixSuggestion{
		Description: "Remove the hardcoded secret and use environment variables or a secure secret management system",
		Suggestion:  a.generateFixSuggestion(result.DetectorName),
		References: []string{
			"https://owasp.org/www-community/vulnerabilities/Use_of_hard-coded_password",
			"https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html",
		},
	}
	
	return finding
}

// determineSeverity determines the severity based on the secret type and verification status
func (a *Agent) determineSeverity(result TruffleHogResult) agent.Severity {
	// All secrets are considered high severity by default as per requirements
	// Verified secrets are always high severity
	if result.Verified {
		return agent.SeverityHigh
	}
	
	// Certain detector types are always high severity even if unverified
	highSeverityDetectors := map[string]bool{
		"aws":           true,
		"github":        true,
		"gitlab":        true,
		"slack":         true,
		"stripe":        true,
		"twilio":        true,
		"mailgun":       true,
		"sendgrid":      true,
		"privatekey":    true,
		"jwt":           true,
		"database":      true,
		"postgresql":    true,
		"mysql":         true,
		"mongodb":       true,
	}
	
	detectorLower := strings.ToLower(result.DetectorName)
	for detector := range highSeverityDetectors {
		if strings.Contains(detectorLower, detector) {
			return agent.SeverityHigh
		}
	}
	
	// Default to high severity for secrets as per requirements
	return agent.SeverityHigh
}

// calculateConfidence calculates confidence score based on verification and detector type
func (a *Agent) calculateConfidence(result TruffleHogResult) float64 {
	// Verified secrets get highest confidence
	if result.Verified {
		return 0.95
	}
	
	// Base confidence for unverified secrets
	confidence := 0.7
	
	// Adjust based on detector reliability for unverified secrets
	reliableDetectors := map[string]float64{
		"aws":        0.9,
		"github":     0.9,
		"gitlab":     0.9,
		"slack":      0.85,
		"stripe":     0.85,
		"privatekey": 0.8,
		"jwt":        0.75,
	}
	
	detectorLower := strings.ToLower(result.DetectorName)
	for detector, boost := range reliableDetectors {
		if strings.Contains(detectorLower, detector) {
			confidence = boost
			break
		}
	}
	
	return confidence
}

// generateFixSuggestion generates a fix suggestion based on the detector type
func (a *Agent) generateFixSuggestion(detectorName string) string {
	detectorLower := strings.ToLower(detectorName)
	
	switch {
	case strings.Contains(detectorLower, "aws"):
		return "Use AWS IAM roles, AWS Secrets Manager, or environment variables instead of hardcoded AWS credentials"
	case strings.Contains(detectorLower, "github"):
		return "Use GitHub's encrypted secrets feature or environment variables instead of hardcoded GitHub tokens"
	case strings.Contains(detectorLower, "gitlab"):
		return "Use GitLab CI/CD variables or environment variables instead of hardcoded GitLab tokens"
	case strings.Contains(detectorLower, "slack"):
		return "Store Slack tokens in environment variables or a secure secret management system"
	case strings.Contains(detectorLower, "stripe"):
		return "Use environment variables for Stripe API keys and never commit them to version control"
	case strings.Contains(detectorLower, "privatekey"):
		return "Store private keys securely using a key management service and never commit them to version control"
	case strings.Contains(detectorLower, "jwt"):
		return "Use environment variables for JWT secrets and ensure they are properly rotated"
	case strings.Contains(detectorLower, "database"):
		return "Use environment variables or a secure configuration management system for database credentials"
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