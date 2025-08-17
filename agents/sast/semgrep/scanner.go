package semgrep

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

// executeScan runs the actual Semgrep scan
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "semgrep-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone or copy repository to temp directory
	repoPath := filepath.Join(tempDir, "repo")
	if err := a.prepareRepository(ctx, config, repoPath); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Build Semgrep command
	cmd := a.buildSemgrepCommand(ctx, config, repoPath, tempDir)

	// Execute Semgrep
	output, err := cmd.Output()
	if err != nil {
		// Semgrep returns non-zero exit code when findings are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means findings were found, which is not an error
			if exitErr.ExitCode() == 1 {
				output = exitErr.Stderr
			} else {
				return nil, agent.Metadata{}, fmt.Errorf("semgrep execution failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
			}
		} else {
			return nil, agent.Metadata{}, fmt.Errorf("semgrep execution failed: %w", err)
		}
	}

	// Parse SARIF output
	findings, metadata, err := a.parseSARIFOutput(output, config)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse semgrep output: %w", err)
	}

	return findings, metadata, nil
}

// prepareRepository clones or copies the repository to the specified path
func (a *Agent) prepareRepository(ctx context.Context, config agent.ScanConfig, repoPath string) error {
	// For now, we'll use git clone. In a production environment, you might want to
	// handle different scenarios (local path, archive, etc.)
	
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

// buildSemgrepCommand constructs the Docker command to run Semgrep
func (a *Agent) buildSemgrepCommand(ctx context.Context, config agent.ScanConfig, repoPath, tempDir string) *exec.Cmd {
	args := []string{
		"run", "--rm",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		"-v", fmt.Sprintf("%s:/src:ro", repoPath),
		"-v", fmt.Sprintf("%s:/tmp/semgrep", tempDir),
		a.config.DockerImage,
		"--config", a.config.RulesConfig,
		"--sarif",
		"--output", "/tmp/semgrep/results.sarif",
	}

	// Add language-specific configurations
	if len(config.Languages) > 0 {
		for _, lang := range config.Languages {
			args = append(args, "--lang", lang)
		}
	}

	// Add file filters for incremental scans
	if len(config.Files) > 0 {
		for _, file := range config.Files {
			args = append(args, "--include", file)
		}
	}

	// Add custom rules if specified
	if len(config.Rules) > 0 {
		for _, rule := range config.Rules {
			args = append(args, "--config", rule)
		}
	}

	// Add scan target
	args = append(args, "/src")

	return exec.CommandContext(ctx, "docker", args...)
}

// SARIFReport represents the structure of SARIF output from Semgrep
type SARIFReport struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []struct {
		Tool struct {
			Driver struct {
				Name           string `json:"name"`
				Version        string `json:"version"`
				InformationURI string `json:"informationUri"`
				Rules          []struct {
					ID               string `json:"id"`
					Name             string `json:"name"`
					ShortDescription struct {
						Text string `json:"text"`
					} `json:"shortDescription"`
					FullDescription struct {
						Text string `json:"text"`
					} `json:"fullDescription"`
					DefaultConfiguration struct {
						Level string `json:"level"`
					} `json:"defaultConfiguration"`
					Properties struct {
						Tags []string `json:"tags"`
					} `json:"properties"`
					HelpURI string `json:"helpUri"`
				} `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
		Results []struct {
			RuleID    string `json:"ruleId"`
			RuleIndex int    `json:"ruleIndex"`
			Level     string `json:"level"`
			Message   struct {
				Text string `json:"text"`
			} `json:"message"`
			Locations []struct {
				PhysicalLocation struct {
					ArtifactLocation struct {
						URI string `json:"uri"`
					} `json:"artifactLocation"`
					Region struct {
						StartLine   int `json:"startLine"`
						StartColumn int `json:"startColumn"`
						EndLine     int `json:"endLine"`
						EndColumn   int `json:"endColumn"`
						Snippet     struct {
							Text string `json:"text"`
						} `json:"snippet"`
					} `json:"region"`
				} `json:"physicalLocation"`
			} `json:"locations"`
			Properties struct {
				Extra struct {
					Severity string   `json:"severity"`
					Metadata struct {
						Category   string   `json:"category"`
						Confidence string   `json:"confidence"`
						References []string `json:"references"`
					} `json:"metadata"`
				} `json:"extra"`
			} `json:"properties"`
		} `json:"results"`
	} `json:"runs"`
}

// parseSARIFOutput parses the SARIF output from Semgrep and converts it to our Finding format
func (a *Agent) parseSARIFOutput(output []byte, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	var sarif SARIFReport
	if err := json.Unmarshal(output, &sarif); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse SARIF JSON: %w", err)
	}

	var findings []agent.Finding
	var toolVersion string
	var rulesVersion string
	var filesScanned int
	var linesScanned int

	if len(sarif.Runs) > 0 {
		run := sarif.Runs[0]
		toolVersion = run.Tool.Driver.Version
		rulesVersion = run.Tool.Driver.Version // Semgrep uses same version for rules
		
		// Create rule lookup map
		ruleMap := make(map[string]interface{})
		for _, rule := range run.Tool.Driver.Rules {
			ruleMap[rule.ID] = rule
		}

		for _, result := range run.Results {
			if len(result.Locations) == 0 {
				continue
			}

			location := result.Locations[0].PhysicalLocation
			finding := agent.Finding{
				ID:          generateFindingID(result.RuleID, location.ArtifactLocation.URI, location.Region.StartLine),
				Tool:        AgentName,
				RuleID:      result.RuleID,
				Severity:    a.mapSeverity(result.Level, result.Properties.Extra.Severity),
				Category:    a.mapCategory(result.Properties.Extra.Metadata.Category),
				Title:       result.Message.Text,
				Description: result.Message.Text,
				File:        strings.TrimPrefix(location.ArtifactLocation.URI, "/src/"),
				Line:        location.Region.StartLine,
				Column:      location.Region.StartColumn,
				Code:        location.Region.Snippet.Text,
				Confidence:  a.mapConfidence(result.Properties.Extra.Metadata.Confidence),
				References:  result.Properties.Extra.Metadata.References,
			}

			findings = append(findings, finding)
			filesScanned++
			linesScanned += location.Region.EndLine - location.Region.StartLine + 1
		}
	}

	metadata := agent.Metadata{
		ToolVersion:  toolVersion,
		RulesVersion: rulesVersion,
		ScanType:     "sast",
		FilesScanned: filesScanned,
		LinesScanned: linesScanned,
		ExitCode:     0,
		CommandLine:  "semgrep --config auto --sarif",
	}

	return findings, metadata, nil
}

// generateFindingID creates a unique ID for a finding
func generateFindingID(ruleID, file string, line int) string {
	return fmt.Sprintf("semgrep-%s-%s-%d", ruleID, filepath.Base(file), line)
}

// mapSeverity converts Semgrep severity to our standard severity levels
func (a *Agent) mapSeverity(level, severity string) agent.Severity {
	// Semgrep uses both 'level' (SARIF standard) and custom 'severity'
	switch strings.ToLower(severity) {
	case "error", "high":
		return agent.SeverityHigh
	case "warning", "medium":
		return agent.SeverityMedium
	case "info", "low":
		return agent.SeverityLow
	default:
		// Fallback to SARIF level
		switch strings.ToLower(level) {
		case "error":
			return agent.SeverityHigh
		case "warning":
			return agent.SeverityMedium
		case "note", "info":
			return agent.SeverityLow
		default:
			return agent.SeverityMedium
		}
	}
}

// mapCategory converts Semgrep categories to our standard vulnerability categories
func (a *Agent) mapCategory(category string) agent.VulnCategory {
	switch strings.ToLower(category) {
	case "security", "sql-injection", "sqli":
		return agent.CategorySQLInjection
	case "xss", "cross-site-scripting":
		return agent.CategoryXSS
	case "command-injection", "code-injection":
		return agent.CategoryCommandInjection
	case "path-traversal", "directory-traversal":
		return agent.CategoryPathTraversal
	case "crypto", "cryptography", "insecure-crypto":
		return agent.CategoryInsecureCrypto
	case "secrets", "hardcoded-secrets":
		return agent.CategoryHardcodedSecrets
	case "deserialization", "insecure-deserialization":
		return agent.CategoryInsecureDeserialization
	case "auth", "authentication", "authorization":
		return agent.CategoryAuthBypass
	case "csrf", "cross-site-request-forgery":
		return agent.CategoryCSRF
	case "misconfiguration", "config":
		return agent.CategoryMisconfiguration
	default:
		return agent.CategoryOther
	}
}

// mapConfidence converts Semgrep confidence to a numeric value
func (a *Agent) mapConfidence(confidence string) float64 {
	switch strings.ToLower(confidence) {
	case "high":
		return 0.9
	case "medium":
		return 0.7
	case "low":
		return 0.5
	default:
		return 0.7 // Default to medium confidence
	}
}