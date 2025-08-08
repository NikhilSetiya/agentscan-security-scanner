package govulncheck

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

// executeScan runs the actual govulncheck scan
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "govulncheck-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone or copy repository to temp directory
	repoPath := filepath.Join(tempDir, "repo")
	if err := a.prepareRepository(ctx, config, repoPath); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Check for go.mod
	goModPath := filepath.Join(repoPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		// No go.mod found, return empty results
		return []agent.Finding{}, agent.Metadata{
			ToolVersion:  a.getToolVersion(),
			RulesVersion: "latest",
			ScanType:     "sca",
			FilesScanned: 0,
			LinesScanned: 0,
			ExitCode:     0,
			CommandLine:  "govulncheck -json ./...",
			Environment: map[string]string{
				"reason": "no go.mod found",
			},
		}, nil
	}

	// Build govulncheck command
	cmd := a.buildGovulncheckCommand(ctx, config, repoPath, tempDir)

	// Execute govulncheck
	output, err := cmd.Output()
	if err != nil {
		// govulncheck returns non-zero exit code when vulnerabilities are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 3 means vulnerabilities found, which is not an error
			if exitErr.ExitCode() == 3 {
				output = exitErr.Stderr
				if len(output) == 0 {
					// Sometimes output goes to stdout even with errors
					output, _ = cmd.Output()
				}
			} else {
				return nil, agent.Metadata{}, fmt.Errorf("govulncheck execution failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
			}
		} else {
			return nil, agent.Metadata{}, fmt.Errorf("govulncheck execution failed: %w", err)
		}
	}

	// Parse JSON output
	findings, metadata, err := a.parseGovulncheckOutput(output, config)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse govulncheck output: %w", err)
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

// buildGovulncheckCommand constructs the Docker command to run govulncheck
func (a *Agent) buildGovulncheckCommand(ctx context.Context, config agent.ScanConfig, repoPath, tempDir string) *exec.Cmd {
	// Install govulncheck and run scan in a single command
	vulncheckScript := `#!/bin/sh
set -e

# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Change to the repository directory
cd /src

# Download dependencies
go mod download

# Run govulncheck with JSON output
govulncheck -json`

	if a.config.ShowTraces {
		vulncheckScript += " -show=traces"
	}

	if a.config.TestPackages {
		vulncheckScript += " -test"
	}

	if len(a.config.Tags) > 0 {
		vulncheckScript += " -tags=" + strings.Join(a.config.Tags, ",")
	}

	if a.config.VulnDBURL != "" {
		vulncheckScript += " -db=" + a.config.VulnDBURL
	}

	vulncheckScript += ` ./... || true  # Don't fail on vulnerabilities found

echo "Vulnerability check completed"
`

	scriptPath := filepath.Join(tempDir, "run-govulncheck.sh")
	os.WriteFile(scriptPath, []byte(vulncheckScript), 0755)

	args := []string{
		"run", "--rm",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		"-v", fmt.Sprintf("%s:/src:ro", repoPath),
		"-v", fmt.Sprintf("%s:/run-govulncheck.sh:ro", scriptPath),
		"-w", "/src",
	}

	// Add environment variables for Go configuration
	args = append(args, "-e", "GOPROXY=https://proxy.golang.org,direct")
	args = append(args, "-e", "GOSUMDB=sum.golang.org")

	if a.config.VulnDBURL != "" {
		args = append(args, "-e", fmt.Sprintf("GOVULNDB=%s", a.config.VulnDBURL))
	}

	args = append(args, a.config.DockerImage, "sh", "/run-govulncheck.sh")

	return exec.CommandContext(ctx, "docker", args...)
}

// GovulncheckResult represents the structure of govulncheck JSON output
type GovulncheckResult struct {
	Vulns []GovulnVulnerability `json:"Vulns"`
}

type GovulnVulnerability struct {
	OSV      OSVEntry `json:"OSV"`
	Modules  []Module `json:"Modules"`
	CallStacks []CallStack `json:"CallStacks,omitempty"`
}

type OSVEntry struct {
	ID       string    `json:"id"`
	Summary  string    `json:"summary"`
	Details  string    `json:"details"`
	Severity []Severity `json:"severity,omitempty"`
	Affected []Affected `json:"affected"`
}

type Severity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type Affected struct {
	Package Package `json:"package"`
	Ranges  []Range `json:"ranges"`
}

type Package struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type Range struct {
	Type   string  `json:"type"`
	Events []Event `json:"events"`
}

type Event struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}

type Module struct {
	Path         string `json:"Path"`
	FoundVersion string `json:"FoundVersion"`
	FixedVersion string `json:"FixedVersion"`
}

type CallStack struct {
	Symbol string `json:"Symbol"`
	PkgPath string `json:"PkgPath"`
	RecvType string `json:"RecvType,omitempty"`
	Pos     Position `json:"Pos"`
}

type Position struct {
	Filename string `json:"Filename"`
	Offset   int    `json:"Offset"`
	Line     int    `json:"Line"`
	Column   int    `json:"Column"`
}

// parseGovulncheckOutput parses the JSON output from govulncheck and converts it to our Finding format
func (a *Agent) parseGovulncheckOutput(output []byte, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// govulncheck outputs one JSON object per line
	lines := strings.Split(string(output), "\n")
	var findings []agent.Finding
	var totalVulns int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var result GovulnVulnerability
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			// Skip lines that aren't valid JSON vulnerability reports
			continue
		}

		// Skip if this vulnerability should be excluded
		if a.isVulnExcluded(result.OSV.ID) {
			continue
		}

		for _, module := range result.Modules {
			finding := agent.Finding{
				ID:          generateFindingID(module.Path, result.OSV.ID),
				Tool:        AgentName,
				RuleID:      result.OSV.ID,
				Severity:    a.mapSeverity(result.OSV.Severity),
				Category:    agent.CategoryDependencyVuln,
				Title:       fmt.Sprintf("Vulnerable dependency: %s", module.Path),
				Description: result.OSV.Summary,
				File:        "go.mod",
				Line:        0, // Dependencies don't have line numbers
				Column:      0,
				Code:        fmt.Sprintf("%s %s", module.Path, module.FoundVersion),
				Confidence:  0.9, // govulncheck has high confidence
				References:  a.getReferences(result.OSV.ID),
			}

			// Add more detailed description if available
			if result.OSV.Details != "" {
				finding.Description = result.OSV.Details
			}

			// Add fix suggestion if available
			if module.FixedVersion != "" {
				finding.Fix = &agent.FixSuggestion{
					Description: fmt.Sprintf("Update %s to version %s or later", module.Path, module.FixedVersion),
					Suggestion:  fmt.Sprintf("go get %s@%s", module.Path, module.FixedVersion),
				}
			}

			// Add call stack information if available
			if len(result.CallStacks) > 0 {
				callStack := result.CallStacks[0] // Use first call stack
				if callStack.Pos.Filename != "" {
					finding.File = callStack.Pos.Filename
					finding.Line = callStack.Pos.Line
					finding.Column = callStack.Pos.Column
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
		FilesScanned: 1, // go.mod
		LinesScanned: 0, // Not applicable for dependency scanning
		ExitCode:     0,
		CommandLine:  "govulncheck -json ./...",
		Environment: map[string]string{
			"vulnerabilities_found": fmt.Sprintf("%d", totalVulns),
			"go_version": a.config.GoVersion,
		},
	}

	return findings, metadata, nil
}

// generateFindingID creates a unique ID for a finding
func generateFindingID(modulePath, vulnID string) string {
	return fmt.Sprintf("govulncheck-%s-%s", strings.ReplaceAll(modulePath, "/", "-"), vulnID)
}

// mapSeverity converts OSV severity to our standard severity levels
func (a *Agent) mapSeverity(severities []Severity) agent.Severity {
	if len(severities) == 0 {
		return agent.SeverityMedium // Default severity
	}

	// Look for CVSS scores first
	for _, sev := range severities {
		if sev.Type == "CVSS_V3" {
			score := sev.Score
			// Parse CVSS score (format: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
			if strings.Contains(score, "C:H") || strings.Contains(score, "I:H") || strings.Contains(score, "A:H") {
				return agent.SeverityHigh
			} else if strings.Contains(score, "C:M") || strings.Contains(score, "I:M") || strings.Contains(score, "A:M") {
				return agent.SeverityMedium
			} else {
				return agent.SeverityLow
			}
		}
	}

	// Default to medium severity
	return agent.SeverityMedium
}

// getReferences returns documentation references for the vulnerability
func (a *Agent) getReferences(vulnID string) []string {
	references := []string{}
	
	// Add OSV database reference
	references = append(references, fmt.Sprintf("https://osv.dev/vulnerability/%s", vulnID))
	
	// Add CVE reference if it's a CVE
	if strings.HasPrefix(vulnID, "CVE-") {
		references = append(references, fmt.Sprintf("https://cve.mitre.org/cgi-bin/cvename.cgi?name=%s", vulnID))
	}
	
	// Add Go vulnerability database reference
	if strings.HasPrefix(vulnID, "GO-") {
		references = append(references, fmt.Sprintf("https://pkg.go.dev/vuln/%s", vulnID))
	}
	
	return references
}

// isVulnExcluded checks if a vulnerability should be excluded
func (a *Agent) isVulnExcluded(vulnID string) bool {
	for _, pattern := range a.config.ExcludePatterns {
		if strings.Contains(vulnID, pattern) {
			return true
		}
	}
	return false
}