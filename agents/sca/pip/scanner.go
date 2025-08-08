package pip

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

// executeScan runs the actual pip-audit scan
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "pip-audit-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone or copy repository to temp directory
	repoPath := filepath.Join(tempDir, "repo")
	if err := a.prepareRepository(ctx, config, repoPath); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Find requirements files
	requirementsFiles := a.findRequirementsFiles(repoPath)
	if len(requirementsFiles) == 0 {
		// No requirements files found, return empty results
		return []agent.Finding{}, agent.Metadata{
			ToolVersion:  a.getToolVersion(),
			RulesVersion: "latest",
			ScanType:     "sca",
			FilesScanned: 0,
			LinesScanned: 0,
			ExitCode:     0,
			CommandLine:  "pip-audit --format=json",
			Environment: map[string]string{
				"reason": "no requirements files found",
			},
		}, nil
	}

	// Build pip-audit command
	cmd := a.buildPipAuditCommand(ctx, config, repoPath, tempDir, requirementsFiles)

	// Execute pip-audit
	output, err := cmd.Output()
	if err != nil {
		// pip-audit returns non-zero exit code when vulnerabilities are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means vulnerabilities found, which is not an error
			if exitErr.ExitCode() == 1 {
				output = exitErr.Stderr
				if len(output) == 0 {
					// Sometimes output goes to stdout even with errors
					output, _ = cmd.Output()
				}
			} else {
				return nil, agent.Metadata{}, fmt.Errorf("pip-audit execution failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
			}
		} else {
			return nil, agent.Metadata{}, fmt.Errorf("pip-audit execution failed: %w", err)
		}
	}

	// Parse JSON output
	findings, metadata, err := a.parsePipAuditOutput(output, config, requirementsFiles)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse pip-audit output: %w", err)
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

// findRequirementsFiles finds all requirements files in the repository
func (a *Agent) findRequirementsFiles(repoPath string) []string {
	var foundFiles []string
	
	for _, reqFile := range a.config.RequirementsFiles {
		filePath := filepath.Join(repoPath, reqFile)
		if _, err := os.Stat(filePath); err == nil {
			foundFiles = append(foundFiles, reqFile)
		}
	}
	
	return foundFiles
}

// buildPipAuditCommand constructs the Docker command to run pip-audit
func (a *Agent) buildPipAuditCommand(ctx context.Context, config agent.ScanConfig, repoPath, tempDir string, requirementsFiles []string) *exec.Cmd {
	// Install pip-audit and run scan in a single command
	auditScript := `#!/bin/sh
set -e

# Install pip-audit
pip install --no-cache-dir pip-audit

# Change to the repository directory
cd /src

# Run pip-audit on each requirements file
echo "["
first=true
`

	for _, reqFile := range requirementsFiles {
		auditScript += fmt.Sprintf(`
if [ "$first" = true ]; then
    first=false
else
    echo ","
fi

echo "Scanning %s..." >&2
pip-audit --format=json --requirement=%s --no-deps`, reqFile, reqFile)

		if a.config.IndexURL != "" {
			auditScript += " --index-url=" + a.config.IndexURL
		}

		for _, extraIndex := range a.config.ExtraIndexURLs {
			auditScript += " --extra-index-url=" + extraIndex
		}

		for _, ignoreVuln := range a.config.IgnoreVulns {
			auditScript += " --ignore-vuln=" + ignoreVuln
		}

		if a.config.LocalPackages {
			auditScript += " --local"
		}

		auditScript += ` || echo '{"vulnerabilities": []}'`
	}

	auditScript += `
echo "]"
`

	scriptPath := filepath.Join(tempDir, "run-pip-audit.sh")
	os.WriteFile(scriptPath, []byte(auditScript), 0755)

	args := []string{
		"run", "--rm",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		"-v", fmt.Sprintf("%s:/src:ro", repoPath),
		"-v", fmt.Sprintf("%s:/run-pip-audit.sh:ro", scriptPath),
		"-w", "/src",
	}

	// Add environment variables for configuration
	if a.config.IndexURL != "" {
		args = append(args, "-e", fmt.Sprintf("PIP_INDEX_URL=%s", a.config.IndexURL))
	}

	args = append(args, a.config.DockerImage, "sh", "/run-pip-audit.sh")

	return exec.CommandContext(ctx, "docker", args...)
}

// PipAuditResult represents the structure of pip-audit JSON output
type PipAuditResult struct {
	Vulnerabilities []PipVulnerability `json:"vulnerabilities"`
}

type PipVulnerability struct {
	Package     string   `json:"package"`
	Version     string   `json:"version"`
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Fix         PipFix   `json:"fix"`
	Aliases     []string `json:"aliases"`
}

type PipFix struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Versions []string `json:"versions"`
}

// parsePipAuditOutput parses the JSON output from pip-audit and converts it to our Finding format
func (a *Agent) parsePipAuditOutput(output []byte, config agent.ScanConfig, requirementsFiles []string) ([]agent.Finding, agent.Metadata, error) {
	// pip-audit output might be an array of results (one per requirements file)
	var results []PipAuditResult
	
	// Try to parse as array first
	if err := json.Unmarshal(output, &results); err != nil {
		// If that fails, try to parse as single result
		var singleResult PipAuditResult
		if err := json.Unmarshal(output, &singleResult); err != nil {
			return nil, agent.Metadata{}, fmt.Errorf("failed to parse pip-audit JSON: %w", err)
		}
		results = []PipAuditResult{singleResult}
	}

	var findings []agent.Finding
	var totalVulns int
	var filesScanned int = len(requirementsFiles)

	for i, result := range results {
		reqFile := "requirements.txt"
		if i < len(requirementsFiles) {
			reqFile = requirementsFiles[i]
		}

		for _, vuln := range result.Vulnerabilities {
			// Skip if package is in ignore list
			if a.isVulnIgnored(vuln.ID) {
				continue
			}

			finding := agent.Finding{
				ID:          generateFindingID(vuln.Package, vuln.ID),
				Tool:        AgentName,
				RuleID:      vuln.ID,
				Severity:    a.mapSeverity(vuln.ID),
				Category:    a.mapCategory(vuln.ID),
				Title:       fmt.Sprintf("Vulnerable dependency: %s", vuln.Package),
				Description: vuln.Description,
				File:        reqFile,
				Line:        0, // Dependencies don't have line numbers
				Column:      0,
				Code:        fmt.Sprintf("%s==%s", vuln.Package, vuln.Version),
				Confidence:  a.calculateConfidence(vuln.ID),
				References:  a.getReferences(vuln.ID, vuln.Aliases),
			}

			// Add fix suggestion if available
			if vuln.Fix.Version != "" {
				finding.Fix = &agent.FixSuggestion{
					Description: fmt.Sprintf("Update %s to version %s or later", vuln.Package, vuln.Fix.Version),
					Suggestion:  fmt.Sprintf("pip install %s>=%s", vuln.Package, vuln.Fix.Version),
				}
			}

			findings = append(findings, finding)
			totalVulns++
		}
	}

	metadata := agent.Metadata{
		ToolVersion:  a.getToolVersion(),
		RulesVersion: "latest",
		ScanType:     "sca",
		FilesScanned: filesScanned,
		LinesScanned: 0, // Not applicable for dependency scanning
		ExitCode:     0,
		CommandLine:  "pip-audit --format=json",
		Environment: map[string]string{
			"requirements_files": strings.Join(requirementsFiles, ","),
			"vulnerabilities_found": fmt.Sprintf("%d", totalVulns),
		},
	}

	return findings, metadata, nil
}

// generateFindingID creates a unique ID for a finding
func generateFindingID(packageName, vulnID string) string {
	return fmt.Sprintf("pip-audit-%s-%s", packageName, vulnID)
}

// mapSeverity converts vulnerability ID to severity (pip-audit doesn't provide severity directly)
func (a *Agent) mapSeverity(vulnID string) agent.Severity {
	// pip-audit doesn't provide severity information directly
	// We use heuristics based on vulnerability ID patterns
	vulnID = strings.ToUpper(vulnID)
	
	// CVE-based severity mapping (simplified)
	if strings.HasPrefix(vulnID, "CVE-") {
		// For CVEs, we default to medium and could enhance with CVE database lookup
		return agent.SeverityMedium
	}
	
	// GHSA (GitHub Security Advisory) IDs
	if strings.HasPrefix(vulnID, "GHSA-") {
		return agent.SeverityMedium
	}
	
	// PYSEC (Python Security) IDs
	if strings.HasPrefix(vulnID, "PYSEC-") {
		return agent.SeverityMedium
	}
	
	// Default to medium severity
	return agent.SeverityMedium
}

// mapCategory converts vulnerability ID to category
func (a *Agent) mapCategory(vulnID string) agent.VulnCategory {
	// Most pip-audit findings are dependency vulnerabilities
	// We could enhance this with more specific categorization based on vulnerability descriptions
	return agent.CategoryDependencyVuln
}

// calculateConfidence calculates confidence based on vulnerability source
func (a *Agent) calculateConfidence(vulnID string) float64 {
	vulnID = strings.ToUpper(vulnID)
	
	// CVE IDs have high confidence
	if strings.HasPrefix(vulnID, "CVE-") {
		return 0.9
	}
	
	// GHSA IDs have high confidence (GitHub Security Advisories)
	if strings.HasPrefix(vulnID, "GHSA-") {
		return 0.9
	}
	
	// PYSEC IDs have high confidence (Python Security)
	if strings.HasPrefix(vulnID, "PYSEC-") {
		return 0.85
	}
	
	// Default confidence
	return 0.8
}

// getReferences returns documentation references for the vulnerability
func (a *Agent) getReferences(vulnID string, aliases []string) []string {
	references := []string{}
	
	vulnID = strings.ToUpper(vulnID)
	
	// Add primary vulnerability reference
	if strings.HasPrefix(vulnID, "CVE-") {
		references = append(references, fmt.Sprintf("https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s", vulnID))
	} else if strings.HasPrefix(vulnID, "GHSA-") {
		references = append(references, fmt.Sprintf("https://github.com/advisories/%s", vulnID))
	} else if strings.HasPrefix(vulnID, "PYSEC-") {
		references = append(references, fmt.Sprintf("https://osv.dev/vulnerability/%s", vulnID))
	}
	
	// Add alias references
	for _, alias := range aliases {
		alias = strings.ToUpper(alias)
		if strings.HasPrefix(alias, "CVE-") {
			references = append(references, fmt.Sprintf("https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s", alias))
		} else if strings.HasPrefix(alias, "GHSA-") {
			references = append(references, fmt.Sprintf("https://github.com/advisories/%s", alias))
		}
	}
	
	return references
}

// isVulnIgnored checks if a vulnerability should be ignored
func (a *Agent) isVulnIgnored(vulnID string) bool {
	for _, ignored := range a.config.IgnoreVulns {
		if ignored == vulnID {
			return true
		}
	}
	return false
}