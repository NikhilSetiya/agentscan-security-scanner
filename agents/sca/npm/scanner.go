package npm

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

// executeScan runs the actual npm audit scan
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "npm-audit-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone or copy repository to temp directory
	repoPath := filepath.Join(tempDir, "repo")
	if err := a.prepareRepository(ctx, config, repoPath); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Check for package.json
	packageJsonPath := filepath.Join(repoPath, "package.json")
	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		// No package.json found, return empty results
		return []agent.Finding{}, agent.Metadata{
			ToolVersion:  a.getToolVersion(),
			RulesVersion: "latest",
			ScanType:     "sca",
			FilesScanned: 0,
			LinesScanned: 0,
			ExitCode:     0,
			CommandLine:  "npm audit --json",
			Environment: map[string]string{
				"reason": "no package.json found",
			},
		}, nil
	}

	// Build npm audit command
	cmd := a.buildNpmAuditCommand(ctx, config, repoPath, tempDir)

	// Execute npm audit
	output, err := cmd.Output()
	if err != nil {
		// npm audit returns non-zero exit code when vulnerabilities are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit codes 1-6 indicate vulnerabilities found, which is not an error
			if exitErr.ExitCode() >= 1 && exitErr.ExitCode() <= 6 {
				output = exitErr.Stderr
				if len(output) == 0 {
					// Sometimes output goes to stdout even with errors
					output, _ = cmd.Output()
				}
			} else {
				return nil, agent.Metadata{}, fmt.Errorf("npm audit execution failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
			}
		} else {
			return nil, agent.Metadata{}, fmt.Errorf("npm audit execution failed: %w", err)
		}
	}

	// Parse JSON output
	findings, metadata, err := a.parseNpmAuditOutput(output, config)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse npm audit output: %w", err)
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

// buildNpmAuditCommand constructs the Docker command to run npm audit
func (a *Agent) buildNpmAuditCommand(ctx context.Context, config agent.ScanConfig, repoPath, tempDir string) *exec.Cmd {
	// Install dependencies and run audit in a single command
	auditScript := `#!/bin/sh
set -e

# Change to the repository directory
cd /src

# Check if package-lock.json exists, if not create it
if [ ! -f package-lock.json ] && [ ! -f yarn.lock ]; then
    echo "No lock file found, running npm install to generate one..."
    npm install --package-lock-only --no-audit
fi

# Run npm audit with JSON output
npm audit --json --audit-level=` + a.config.AuditLevel

	if a.config.ProductionOnly {
		auditScript += " --production"
	}

	if !a.config.IncludeDevDeps {
		auditScript += " --omit=dev"
	}

	if a.config.RegistryURL != "" {
		auditScript += " --registry=" + a.config.RegistryURL
	}

	auditScript += ` || true  # Don't fail on vulnerabilities found

echo "Audit completed"
`

	scriptPath := filepath.Join(tempDir, "run-npm-audit.sh")
	os.WriteFile(scriptPath, []byte(auditScript), 0755)

	args := []string{
		"run", "--rm",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		"-v", fmt.Sprintf("%s:/src:ro", repoPath),
		"-v", fmt.Sprintf("%s:/run-npm-audit.sh:ro", scriptPath),
		"-w", "/src",
	}

	// Add environment variables for configuration
	if a.config.RegistryURL != "" {
		args = append(args, "-e", fmt.Sprintf("NPM_CONFIG_REGISTRY=%s", a.config.RegistryURL))
	}

	args = append(args, a.config.DockerImage, "sh", "/run-npm-audit.sh")

	return exec.CommandContext(ctx, "docker", args...)
}

// NpmAuditResult represents the structure of npm audit JSON output
type NpmAuditResult struct {
	AuditReportVersion int                    `json:"auditReportVersion"`
	Vulnerabilities    map[string]Vulnerability `json:"vulnerabilities"`
	Metadata           NpmMetadata            `json:"metadata"`
}

type Vulnerability struct {
	Name     string   `json:"name"`
	Severity string   `json:"severity"`
	Via      []Via    `json:"via"`
	Effects  []string `json:"effects"`
	Range    string   `json:"range"`
	Nodes    []string `json:"nodes"`
	FixAvailable interface{} `json:"fixAvailable"`
}

type Via struct {
	Source int    `json:"source,omitempty"`
	Name   string `json:"name,omitempty"`
	Dependency string `json:"dependency,omitempty"`
	Title  string `json:"title,omitempty"`
	URL    string `json:"url,omitempty"`
	Severity string `json:"severity,omitempty"`
	CWE    []string `json:"cwe,omitempty"`
	CVSS   CVSS   `json:"cvss,omitempty"`
	Range  string `json:"range,omitempty"`
}

type CVSS struct {
	Score  float64 `json:"score"`
	Vector string  `json:"vectorString"`
}

type NpmMetadata struct {
	Vulnerabilities VulnCounts `json:"vulnerabilities"`
	Dependencies    int        `json:"dependencies"`
	DevDependencies int        `json:"devDependencies"`
	OptionalDependencies int   `json:"optionalDependencies"`
	TotalDependencies int      `json:"totalDependencies"`
}

type VulnCounts struct {
	Info     int `json:"info"`
	Low      int `json:"low"`
	Moderate int `json:"moderate"`
	High     int `json:"high"`
	Critical int `json:"critical"`
	Total    int `json:"total"`
}

// parseNpmAuditOutput parses the JSON output from npm audit and converts it to our Finding format
func (a *Agent) parseNpmAuditOutput(output []byte, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	var npmResult NpmAuditResult
	if err := json.Unmarshal(output, &npmResult); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse npm audit JSON: %w", err)
	}

	var findings []agent.Finding
	var filesScanned int = 1 // package.json

	for packageName, vuln := range npmResult.Vulnerabilities {
		// Skip if package is in exclude list
		if a.isPackageExcluded(packageName) {
			continue
		}

		for _, via := range vuln.Via {
			if via.Source == 0 { // Direct vulnerability (not transitive)
				finding := agent.Finding{
					ID:          generateFindingID(packageName, via.Title),
					Tool:        AgentName,
					RuleID:      fmt.Sprintf("npm-audit-%s", packageName),
					Severity:    a.mapSeverity(vuln.Severity),
					Category:    a.mapCategory(via.CWE),
					Title:       fmt.Sprintf("Vulnerable dependency: %s", packageName),
					Description: via.Title,
					File:        "package.json",
					Line:        0, // Dependencies don't have line numbers
					Column:      0,
					Code:        fmt.Sprintf(`"%s": "%s"`, packageName, vuln.Range),
					Confidence:  a.calculateConfidence(vuln.Severity, via.CVSS.Score),
					References:  a.getReferences(via.URL, via.CWE),
				}

				// Add fix suggestion if available
				if vuln.FixAvailable != nil && vuln.FixAvailable != false {
					finding.Fix = &agent.FixSuggestion{
						Description: fmt.Sprintf("Update %s to a secure version", packageName),
						Suggestion:  fmt.Sprintf("npm update %s", packageName),
					}
				}

				findings = append(findings, finding)
			}
		}
	}

	metadata := agent.Metadata{
		ToolVersion:  a.getToolVersion(),
		RulesVersion: "latest",
		ScanType:     "sca",
		FilesScanned: filesScanned,
		LinesScanned: 0, // Not applicable for dependency scanning
		ExitCode:     0,
		CommandLine:  "npm audit --json",
		Environment: map[string]string{
			"total_dependencies": fmt.Sprintf("%d", npmResult.Metadata.TotalDependencies),
			"vulnerabilities_found": fmt.Sprintf("%d", npmResult.Metadata.Vulnerabilities.Total),
			"critical": fmt.Sprintf("%d", npmResult.Metadata.Vulnerabilities.Critical),
			"high":     fmt.Sprintf("%d", npmResult.Metadata.Vulnerabilities.High),
			"moderate": fmt.Sprintf("%d", npmResult.Metadata.Vulnerabilities.Moderate),
			"low":      fmt.Sprintf("%d", npmResult.Metadata.Vulnerabilities.Low),
		},
	}

	return findings, metadata, nil
}

// generateFindingID creates a unique ID for a finding
func generateFindingID(packageName, title string) string {
	// Create a simple hash-like ID
	return fmt.Sprintf("npm-audit-%s-%d", packageName, len(title))
}

// mapSeverity converts npm audit severity to our standard severity levels
func (a *Agent) mapSeverity(npmSeverity string) agent.Severity {
	switch strings.ToLower(npmSeverity) {
	case "critical":
		return agent.SeverityHigh
	case "high":
		return agent.SeverityHigh
	case "moderate":
		return agent.SeverityMedium
	case "low":
		return agent.SeverityLow
	case "info":
		return agent.SeverityLow
	default:
		return agent.SeverityMedium
	}
}

// mapCategory converts CWE IDs to our standard vulnerability categories
func (a *Agent) mapCategory(cwes []string) agent.VulnCategory {
	if len(cwes) == 0 {
		return agent.CategoryDependencyVuln
	}

	// Map common CWEs to categories
	for _, cwe := range cwes {
		switch cwe {
		case "CWE-79": // Cross-site Scripting
			return agent.CategoryXSS
		case "CWE-89": // SQL Injection
			return agent.CategorySQLInjection
		case "CWE-78": // Command Injection
			return agent.CategoryCommandInjection
		case "CWE-22": // Path Traversal
			return agent.CategoryPathTraversal
		case "CWE-327", "CWE-328": // Cryptographic Issues
			return agent.CategoryInsecureCrypto
		case "CWE-502": // Deserialization
			return agent.CategoryInsecureDeserialization
		case "CWE-1104": // Supply Chain
			return agent.CategorySupplyChain
		}
	}

	return agent.CategoryDependencyVuln
}

// calculateConfidence calculates confidence based on severity and CVSS score
func (a *Agent) calculateConfidence(severity string, cvssScore float64) float64 {
	baseConfidence := 0.7 // Default confidence for npm audit

	// Adjust based on severity
	switch strings.ToLower(severity) {
	case "critical":
		baseConfidence = 0.95
	case "high":
		baseConfidence = 0.9
	case "moderate":
		baseConfidence = 0.8
	case "low":
		baseConfidence = 0.6
	case "info":
		baseConfidence = 0.5
	}

	// Adjust based on CVSS score if available
	if cvssScore > 0 {
		cvssConfidence := cvssScore / 10.0 // Normalize CVSS score to 0-1
		baseConfidence = (baseConfidence + cvssConfidence) / 2
	}

	return baseConfidence
}

// getReferences returns documentation references for the vulnerability
func (a *Agent) getReferences(url string, cwes []string) []string {
	references := []string{}
	
	if url != "" {
		references = append(references, url)
	}
	
	// Add CWE references
	for _, cwe := range cwes {
		if cwe != "" {
			references = append(references, fmt.Sprintf("https://cwe.mitre.org/data/definitions/%s.html", strings.TrimPrefix(cwe, "CWE-")))
		}
	}
	
	return references
}

// isPackageExcluded checks if a package should be excluded from scanning
func (a *Agent) isPackageExcluded(packageName string) bool {
	for _, excluded := range a.config.ExcludePackages {
		if excluded == packageName {
			return true
		}
	}
	return false
}