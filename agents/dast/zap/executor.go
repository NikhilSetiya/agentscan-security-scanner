package zap

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

// executeScan runs the actual ZAP scan against the running application
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig, app *RunningApp) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for ZAP output
	tempDir, err := os.MkdirTemp("", "zap-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Build ZAP command
	cmd := a.buildZAPCommand(ctx, config, app, tempDir)

	// Execute ZAP scan
	output, err := cmd.CombinedOutput()
	if err != nil {
		// ZAP may return non-zero exit code when findings are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit codes 1-3 are acceptable (findings found)
			if exitErr.ExitCode() > 3 {
				return nil, agent.Metadata{}, fmt.Errorf("zap execution failed with exit code %d: %s", exitErr.ExitCode(), string(output))
			}
		} else {
			return nil, agent.Metadata{}, fmt.Errorf("zap execution failed: %w", err)
		}
	}

	// Parse JSON output
	findings, metadata, err := a.parseZAPOutput(tempDir, config, app)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse zap output: %w", err)
	}

	return findings, metadata, nil
}

// buildZAPCommand constructs the Docker command to run ZAP
func (a *Agent) buildZAPCommand(ctx context.Context, config agent.ScanConfig, app *RunningApp, tempDir string) *exec.Cmd {
	args := []string{
		"run", "--rm",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		"--network", "host", // Use host network to access the application
		"-v", fmt.Sprintf("%s:/zap/wrk:rw", tempDir),
		a.config.DockerImage,
	}

	// Use baseline scan by default
	switch a.config.ScanType {
	case "baseline":
		args = append(args, "zap-baseline.py")
	case "full":
		args = append(args, "zap-full-scan.py")
	case "api":
		args = append(args, "zap-api-scan.py")
	default:
		args = append(args, "zap-baseline.py")
	}

	// Add target URL
	args = append(args, "-t", app.URL)

	// Add JSON output
	args = append(args, "-J", "/zap/wrk/report.json")

	// Add additional options
	args = append(args, "-d") // Include debug info

	// Set maximum scan time (ZAP timeout)
	args = append(args, "-m", "5") // 5 minutes max

	return exec.CommandContext(ctx, "docker", args...)
}

// ZAPReport represents the structure of ZAP JSON output
type ZAPReport struct {
	Site []struct {
		Name   string `json:"@name"`
		Host   string `json:"@host"`
		Port   string `json:"@port"`
		SSL    string `json:"@ssl"`
		Alerts []struct {
			PluginID    string `json:"pluginid"`
			AlertRef    string `json:"alertRef"`
			Alert       string `json:"alert"`
			Name        string `json:"name"`
			RiskCode    string `json:"riskcode"`
			Confidence  string `json:"confidence"`
			RiskDesc    string `json:"riskdesc"`
			Desc        string `json:"desc"`
			Count       string `json:"count"`
			Solution    string `json:"solution"`
			OtherInfo   string `json:"otherinfo"`
			Reference   string `json:"reference"`
			CWEId       string `json:"cweid"`
			WASCId      string `json:"wascid"`
			SourceID    string `json:"sourceid"`
			Instances   []struct {
				URI    string `json:"uri"`
				Method string `json:"method"`
				Param  string `json:"param"`
				Attack string `json:"attack"`
				Evidence string `json:"evidence"`
			} `json:"instances"`
		} `json:"alerts"`
	} `json:"site"`
}

// parseZAPOutput parses the ZAP JSON output and converts it to our Finding format
func (a *Agent) parseZAPOutput(tempDir string, config agent.ScanConfig, app *RunningApp) ([]agent.Finding, agent.Metadata, error) {
	reportPath := filepath.Join(tempDir, "report.json")
	
	// Check if report file exists
	if !fileExists(reportPath) {
		// No report file means no findings
		return []agent.Finding{}, agent.Metadata{
			ScanType:     "dast",
			FilesScanned: 1, // We scanned one web application
			ExitCode:     0,
			CommandLine:  "zap-baseline.py",
		}, nil
	}

	data, err := ioutil.ReadFile(reportPath)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to read ZAP report: %w", err)
	}

	var zapReport ZAPReport
	if err := json.Unmarshal(data, &zapReport); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse ZAP JSON: %w", err)
	}

	var findings []agent.Finding
	var totalInstances int

	for _, site := range zapReport.Site {
		for _, alert := range site.Alerts {
			for _, instance := range alert.Instances {
				finding := agent.Finding{
					ID:          generateZAPFindingID(alert.PluginID, instance.URI, instance.Method),
					Tool:        AgentName,
					RuleID:      alert.PluginID,
					Severity:    a.mapZAPSeverity(alert.RiskCode),
					Category:    a.mapZAPCategory(alert.CWEId, alert.Alert),
					Title:       alert.Alert,
					Description: alert.Desc,
					File:        instance.URI,
					Line:        0, // DAST doesn't have line numbers
					Column:      0,
					Code:        instance.Evidence,
					Confidence:  a.mapZAPConfidence(alert.Confidence),
					References:  a.parseZAPReferences(alert.Reference),
				}

				// Add DAST-specific metadata
				if finding.Fix == nil {
					finding.Fix = &agent.FixSuggestion{}
				}
				finding.Fix.Description = alert.Solution

				findings = append(findings, finding)
				totalInstances++
			}
		}
	}

	metadata := agent.Metadata{
		ToolVersion:  a.getToolVersion(),
		ScanType:     "dast",
		FilesScanned: 1, // We scanned one web application
		ExitCode:     0,
		CommandLine:  "zap-baseline.py -t " + app.URL,
		Environment: map[string]string{
			"target_url": app.URL,
			"scan_type":  a.config.ScanType,
		},
	}

	return findings, metadata, nil
}

// generateZAPFindingID creates a unique ID for a ZAP finding
func generateZAPFindingID(pluginID, uri, method string) string {
	return fmt.Sprintf("zap-%s-%s-%s", pluginID, method, strings.ReplaceAll(uri, "/", "-"))
}

// mapZAPSeverity converts ZAP risk codes to our standard severity levels
func (a *Agent) mapZAPSeverity(riskCode string) agent.Severity {
	switch riskCode {
	case "3": // High
		return agent.SeverityHigh
	case "2": // Medium
		return agent.SeverityMedium
	case "1": // Low
		return agent.SeverityLow
	case "0": // Informational
		return agent.SeverityInfo
	default:
		return agent.SeverityMedium
	}
}

// mapZAPCategory converts ZAP CWE IDs and alert names to our standard vulnerability categories
func (a *Agent) mapZAPCategory(cweID, alertName string) agent.VulnCategory {
	// Map by CWE ID first
	switch cweID {
	case "79": // Cross-site Scripting (XSS)
		return agent.CategoryXSS
	case "89": // SQL Injection
		return agent.CategorySQLInjection
	case "352": // Cross-Site Request Forgery (CSRF)
		return agent.CategoryCSRF
	case "22": // Path Traversal
		return agent.CategoryPathTraversal
	case "77", "78": // Command Injection
		return agent.CategoryCommandInjection
	case "502": // Deserialization
		return agent.CategoryInsecureDeserialization
	case "287", "306": // Authentication
		return agent.CategoryAuthBypass
	}

	// Fallback to alert name mapping
	alertLower := strings.ToLower(alertName)
	switch {
	case strings.Contains(alertLower, "xss") || strings.Contains(alertLower, "cross-site scripting"):
		return agent.CategoryXSS
	case strings.Contains(alertLower, "sql") && strings.Contains(alertLower, "injection"):
		return agent.CategorySQLInjection
	case strings.Contains(alertLower, "csrf") || strings.Contains(alertLower, "cross-site request forgery"):
		return agent.CategoryCSRF
	case strings.Contains(alertLower, "path") && strings.Contains(alertLower, "traversal"):
		return agent.CategoryPathTraversal
	case strings.Contains(alertLower, "command") && strings.Contains(alertLower, "injection"):
		return agent.CategoryCommandInjection
	case strings.Contains(alertLower, "auth"):
		return agent.CategoryAuthBypass
	case strings.Contains(alertLower, "crypto"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(alertLower, "config"):
		return agent.CategoryMisconfiguration
	default:
		return agent.CategoryOther
	}
}

// mapZAPConfidence converts ZAP confidence levels to numeric values
func (a *Agent) mapZAPConfidence(confidence string) float64 {
	switch strings.ToLower(confidence) {
	case "3", "high":
		return 0.9
	case "2", "medium":
		return 0.7
	case "1", "low":
		return 0.5
	case "0", "false positive":
		return 0.1
	default:
		return 0.7 // Default to medium confidence
	}
}

// parseZAPReferences parses ZAP reference strings into a slice
func (a *Agent) parseZAPReferences(reference string) []string {
	if reference == "" {
		return nil
	}

	// ZAP references are typically separated by newlines or HTML breaks
	// First, split by <br> tags
	reference = strings.ReplaceAll(reference, "<br>", "\n")
	reference = strings.ReplaceAll(reference, "<p>", "\n")
	reference = strings.ReplaceAll(reference, "</p>", "\n")
	
	refs := strings.Split(reference, "\n")
	var cleanRefs []string

	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		
		if ref != "" && (strings.HasPrefix(ref, "http") || strings.Contains(ref, "CWE") || strings.Contains(ref, "OWASP")) {
			cleanRefs = append(cleanRefs, ref)
		}
	}

	return cleanRefs
}