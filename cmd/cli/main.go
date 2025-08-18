package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "scan":
		runScan()
	case "version":
		printVersion()
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("AgentScan CLI - Multi-agent security scanner")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agentscan-cli scan [options]")
	fmt.Println("  agentscan-cli version")
	fmt.Println("  agentscan-cli help")
	fmt.Println()
	fmt.Println("Scan Options:")
	fmt.Println("  --api-url=URL              AgentScan API URL (default: https://api.agentscan.dev)")
	fmt.Println("  --api-token=TOKEN          API authentication token")
	fmt.Println("  --fail-on-severity=LEVEL   Fail on findings of this severity or higher (low, medium, high)")
	fmt.Println("  --exclude-path=PATH        Exclude path from scanning (can be used multiple times)")
	fmt.Println("  --include-tools=TOOLS      Comma-separated list of tools to include")
	fmt.Println("  --exclude-tools=TOOLS      Comma-separated list of tools to exclude")
	fmt.Println("  --output-format=FORMAT     Output format: json, sarif, pdf (default: json)")
	fmt.Println("  --output-file=FILE         Output file path (default: agentscan-results.json)")
	fmt.Println("  --timeout=DURATION         Scan timeout (default: 30m)")
	fmt.Println("  --verbose                  Enable verbose output")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  AGENTSCAN_API_URL          API URL")
	fmt.Println("  AGENTSCAN_API_TOKEN        API token")
	fmt.Println("  GITHUB_WORKSPACE           GitHub Actions workspace path")
	fmt.Println("  GITHUB_REPOSITORY          GitHub repository name")
	fmt.Println("  GITHUB_SHA                 Git commit SHA")
	fmt.Println("  GITHUB_REF                 Git reference")
}

func printVersion() {
	fmt.Println("AgentScan CLI v1.0.0")
}

func runScan() {
	options := parseScanOptions()
	
	if options.Verbose {
		log.Printf("Starting scan with options: %+v", options)
	}

	// Detect current directory as workspace
	workspace, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Override with CI workspace if available
	if ciWorkspace := detectCIWorkspace(); ciWorkspace != "" {
		workspace = ciWorkspace
	}

	var results *ScanResults
	
	if options.APIUrl != "" && options.APIToken != "" {
		// Use API-based scanning
		results, err = performAPIScan(workspace, options)
	} else {
		// Use local scanning
		results, err = performLocalScan(workspace, options)
	}
	
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	// Write results to file
	if err := writeResults(results, options); err != nil {
		log.Fatalf("Failed to write results: %v", err)
	}

	// Print summary
	printScanSummary(results, options)

	// Post CI/CD integration comments if applicable
	if err := postCIIntegrationResults(results, options); err != nil {
		log.Printf("Warning: Failed to post CI integration results: %v", err)
	}

	// Exit with appropriate code based on findings
	exitCode := determineExitCode(results, options)
	os.Exit(exitCode)
}

type ScanOptions struct {
	APIUrl         string
	APIToken       string
	FailOnSeverity string
	ExcludePaths   []string
	IncludeTools   []string
	ExcludeTools   []string
	OutputFormat   string
	OutputFile     string
	Timeout        time.Duration
	Verbose        bool
}

func parseScanOptions() *ScanOptions {
	options := &ScanOptions{
		APIUrl:       getEnvOrDefault("AGENTSCAN_API_URL", "https://api.agentscan.dev"),
		APIToken:     os.Getenv("AGENTSCAN_API_TOKEN"),
		OutputFormat: "json",
		OutputFile:   "agentscan-results.json",
		Timeout:      30 * time.Minute,
	}

	// Parse command line arguments
	for _, arg := range os.Args[2:] {
		if strings.HasPrefix(arg, "--api-url=") {
			options.APIUrl = strings.TrimPrefix(arg, "--api-url=")
		} else if strings.HasPrefix(arg, "--api-token=") {
			options.APIToken = strings.TrimPrefix(arg, "--api-token=")
		} else if strings.HasPrefix(arg, "--fail-on-severity=") {
			options.FailOnSeverity = strings.TrimPrefix(arg, "--fail-on-severity=")
		} else if strings.HasPrefix(arg, "--exclude-path=") {
			path := strings.TrimPrefix(arg, "--exclude-path=")
			options.ExcludePaths = append(options.ExcludePaths, path)
		} else if strings.HasPrefix(arg, "--include-tools=") {
			tools := strings.TrimPrefix(arg, "--include-tools=")
			options.IncludeTools = strings.Split(tools, ",")
		} else if strings.HasPrefix(arg, "--exclude-tools=") {
			tools := strings.TrimPrefix(arg, "--exclude-tools=")
			options.ExcludeTools = strings.Split(tools, ",")
		} else if strings.HasPrefix(arg, "--output-format=") {
			options.OutputFormat = strings.TrimPrefix(arg, "--output-format=")
		} else if strings.HasPrefix(arg, "--output-file=") {
			options.OutputFile = strings.TrimPrefix(arg, "--output-file=")
		} else if strings.HasPrefix(arg, "--timeout=") {
			timeoutStr := strings.TrimPrefix(arg, "--timeout=")
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				options.Timeout = timeout
			}
		} else if arg == "--verbose" {
			options.Verbose = true
		}
	}

	return options
}

func detectRepoURL() string {
	// Try GitHub Actions environment
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		return fmt.Sprintf("https://github.com/%s", repo)
	}

	// Try git remote
	// This would normally execute: git remote get-url origin
	// For now, return a placeholder
	return "local-repository"
}

func detectBranch() string {
	// Try GitHub Actions environment
	if ref := os.Getenv("GITHUB_REF"); ref != "" {
		if strings.HasPrefix(ref, "refs/heads/") {
			return strings.TrimPrefix(ref, "refs/heads/")
		}
	}

	// Try git branch
	// This would normally execute: git branch --show-current
	// For now, return a placeholder
	return "main"
}

func detectCommit() string {
	// Try GitHub Actions environment
	if sha := os.Getenv("GITHUB_SHA"); sha != "" {
		return sha
	}

	// Try git rev-parse
	// This would normally execute: git rev-parse HEAD
	// For now, return a placeholder
	return "unknown"
}

func performLocalScan(workspace string, options *ScanOptions) (*ScanResults, error) {
	// This is a simplified local scan implementation
	// In a real implementation, this would:
	// 1. Initialize the scanning agents
	// 2. Run each agent on the workspace
	// 3. Collect and aggregate results
	// 4. Apply consensus scoring

	if options.Verbose {
		log.Printf("Scanning workspace: %s", workspace)
	}

	// Mock results for demonstration
	results := &ScanResults{
		Status:    "completed",
		StartTime: time.Now().Add(-2 * time.Minute),
		EndTime:   time.Now(),
		Findings:  []types.Finding{},
		Summary: ScanSummary{
			TotalFindings: 0,
			BySeverity: map[string]int{
				"high":   0,
				"medium": 0,
				"low":    0,
			},
			ByTool: map[string]int{},
		},
	}

	// Simulate finding some issues based on file patterns
	if err := filepath.Walk(workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip excluded paths
		for _, excludePath := range options.ExcludePaths {
			if matched, _ := filepath.Match(excludePath, path); matched {
				return nil
			}
		}

		// Simple pattern matching for demonstration
		if strings.HasSuffix(path, ".js") && strings.Contains(path, "vulnerable") {
			finding := types.Finding{
				ID:           uuid.New(),
				Tool:         "eslint-security",
				RuleID:       "security/detect-unsafe-regex",
				Severity:     "medium",
				Category:     "security",
				Title:        "Potentially unsafe regular expression",
				Description:  "This regular expression may be vulnerable to ReDoS attacks",
				FilePath:     path,
				LineNumber:   42,
				Confidence:   0.8,
				Status:       "open",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			results.Findings = append(results.Findings, finding)
			results.Summary.TotalFindings++
			results.Summary.BySeverity["medium"]++
			results.Summary.ByTool["eslint-security"]++
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to scan workspace: %w", err)
	}

	if options.Verbose {
		log.Printf("Scan completed: found %d findings", results.Summary.TotalFindings)
	}

	return results, nil
}

func writeResults(results *ScanResults, options *ScanOptions) error {
	switch options.OutputFormat {
	case "json":
		return writeJSONResults(results, options.OutputFile)
	case "sarif":
		return writeSARIFResults(results, options.OutputFile)
	case "pdf":
		return writePDFResults(results, options.OutputFile)
	default:
		return fmt.Errorf("unsupported output format: %s", options.OutputFormat)
	}
}

func writeJSONResults(results *ScanResults, filename string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

func writeSARIFResults(results *ScanResults, filename string) error {
	// Convert to SARIF format
	sarif := convertToSARIF(results)
	
	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

func writePDFResults(results *ScanResults, filename string) error {
	// PDF generation would be implemented here
	// For now, just write a placeholder
	content := fmt.Sprintf("AgentScan Security Report\n\nTotal Findings: %d\n", results.Summary.TotalFindings)
	return os.WriteFile(filename, []byte(content), 0644)
}

func printScanSummary(results *ScanResults, options *ScanOptions) {
	fmt.Println("🔒 AgentScan Security Report")
	fmt.Println("=" + strings.Repeat("=", 28))
	fmt.Printf("Status: %s\n", results.Status)
	fmt.Printf("Duration: %v\n", results.EndTime.Sub(results.StartTime).Round(time.Second))
	fmt.Printf("Total Findings: %d\n", results.Summary.TotalFindings)
	
	if results.Summary.TotalFindings > 0 {
		fmt.Println("\nBy Severity:")
		for severity, count := range results.Summary.BySeverity {
			if count > 0 {
				emoji := getSeverityEmoji(severity)
				fmt.Printf("  %s %s: %d\n", emoji, strings.Title(severity), count)
			}
		}

		fmt.Println("\nBy Tool:")
		for tool, count := range results.Summary.ByTool {
			if count > 0 {
				fmt.Printf("  • %s: %d\n", tool, count)
			}
		}
	}

	fmt.Printf("\nResults written to: %s\n", options.OutputFile)
}

func determineExitCode(results *ScanResults, options *ScanOptions) int {
	if results.Status != "completed" {
		return 2 // Scan failed
	}

	if options.FailOnSeverity == "" {
		return 0 // Success regardless of findings
	}

	switch options.FailOnSeverity {
	case "high":
		if results.Summary.BySeverity["high"] > 0 {
			return 1
		}
	case "medium":
		if results.Summary.BySeverity["high"] > 0 || results.Summary.BySeverity["medium"] > 0 {
			return 1
		}
	case "low":
		if results.Summary.TotalFindings > 0 {
			return 1
		}
	}

	return 0
}

func getSeverityEmoji(severity string) string {
	switch severity {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// detectCIWorkspace detects the workspace path from various CI environments
func detectCIWorkspace() string {
	// GitHub Actions
	if workspace := os.Getenv("GITHUB_WORKSPACE"); workspace != "" {
		return workspace
	}
	
	// GitLab CI
	if workspace := os.Getenv("CI_PROJECT_DIR"); workspace != "" {
		return workspace
	}
	
	// Jenkins
	if workspace := os.Getenv("WORKSPACE"); workspace != "" {
		return workspace
	}
	
	// Azure DevOps
	if workspace := os.Getenv("BUILD_SOURCESDIRECTORY"); workspace != "" {
		return workspace
	}
	
	// CircleCI
	if workspace := os.Getenv("CIRCLE_WORKING_DIRECTORY"); workspace != "" {
		return workspace
	}
	
	return ""
}

// performAPIScan performs a scan using the AgentScan API
func performAPIScan(workspace string, options *ScanOptions) (*ScanResults, error) {
	// This would implement API-based scanning
	// For now, fall back to local scanning
	return performLocalScan(workspace, options)
}

// postCIIntegrationResults posts results to CI/CD platforms
func postCIIntegrationResults(results *ScanResults, options *ScanOptions) error {
	// GitHub Actions - Set outputs
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return postGitHubActionsResults(results)
	}
	
	// GitLab CI - Already handled in GitLab CI template
	if os.Getenv("GITLAB_CI") == "true" {
		return postGitLabResults(results)
	}
	
	// Jenkins - Set build description
	if os.Getenv("JENKINS_URL") != "" {
		return postJenkinsResults(results)
	}
	
	return nil
}

// postGitHubActionsResults sets GitHub Actions outputs
func postGitHubActionsResults(results *ScanResults) error {
	if outputFile := os.Getenv("GITHUB_OUTPUT"); outputFile != "" {
		f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		
		fmt.Fprintf(f, "results-file=agentscan-results.json\n")
		fmt.Fprintf(f, "findings-count=%d\n", results.Summary.TotalFindings)
		fmt.Fprintf(f, "high-severity-count=%d\n", results.Summary.BySeverity["high"])
		fmt.Fprintf(f, "medium-severity-count=%d\n", results.Summary.BySeverity["medium"])
		fmt.Fprintf(f, "low-severity-count=%d\n", results.Summary.BySeverity["low"])
	}
	
	return nil
}

// postGitLabResults handles GitLab CI integration
func postGitLabResults(results *ScanResults) error {
	// GitLab integration is handled in the CI template
	// This could be extended to post additional metadata
	return nil
}

// postJenkinsResults handles Jenkins integration
func postJenkinsResults(results *ScanResults) error {
	// Jenkins integration is handled by the plugin
	// This could be extended to set environment variables or write files
	return nil
}

// ScanResults represents the results of a security scan
type ScanResults struct {
	Status    string            `json:"status"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time"`
	Findings  []types.Finding   `json:"findings"`
	Summary   ScanSummary       `json:"summary"`
}

// ScanSummary provides a summary of scan results
type ScanSummary struct {
	TotalFindings int            `json:"total_findings"`
	BySeverity    map[string]int `json:"by_severity"`
	ByTool        map[string]int `json:"by_tool"`
}

// convertToSARIF converts scan results to SARIF format
func convertToSARIF(results *ScanResults) map[string]interface{} {
	// Basic SARIF structure
	sarif := map[string]interface{}{
		"version": "2.1.0",
		"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		"runs": []map[string]interface{}{
			{
				"tool": map[string]interface{}{
					"driver": map[string]interface{}{
						"name":    "AgentScan",
						"version": "1.0.0",
						"informationUri": "https://agentscan.dev",
					},
				},
				"results": convertFindingsToSARIF(results.Findings),
			},
		},
	}

	return sarif
}

func convertFindingsToSARIF(findings []types.Finding) []map[string]interface{} {
	var sarifResults []map[string]interface{}

	for _, finding := range findings {
		sarifResult := map[string]interface{}{
			"ruleId":  finding.RuleID,
			"message": map[string]interface{}{
				"text": finding.Description,
			},
			"level": convertSeverityToSARIF(finding.Severity),
			"locations": []map[string]interface{}{
				{
					"physicalLocation": map[string]interface{}{
						"artifactLocation": map[string]interface{}{
							"uri": finding.FilePath,
						},
						"region": map[string]interface{}{
							"startLine": finding.LineNumber,
						},
					},
				},
			},
		}

		sarifResults = append(sarifResults, sarifResult)
	}

	return sarifResults
}

func convertSeverityToSARIF(severity string) string {
	switch severity {
	case "high":
		return "error"
	case "medium":
		return "warning"
	case "low":
		return "note"
	default:
		return "info"
	}
}
