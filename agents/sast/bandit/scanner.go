package bandit

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

// executeScan runs the actual Bandit scan
func (a *Agent) executeScan(ctx context.Context, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "bandit-scan-*")
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone or copy repository to temp directory
	repoPath := filepath.Join(tempDir, "repo")
	if err := a.prepareRepository(ctx, config, repoPath); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Detect Python environment and setup
	if err := a.setupPythonEnvironment(ctx, repoPath); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to setup Python environment: %w", err)
	}

	// Build Bandit command
	cmd := a.buildBanditCommand(ctx, config, repoPath, tempDir)

	// Execute Bandit
	output, err := cmd.Output()
	if err != nil {
		// Bandit returns non-zero exit code when findings are found
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
				return nil, agent.Metadata{}, fmt.Errorf("bandit execution failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
			}
		} else {
			return nil, agent.Metadata{}, fmt.Errorf("bandit execution failed: %w", err)
		}
	}

	// Parse JSON output
	findings, metadata, err := a.parseBanditOutput(output, config)
	if err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse bandit output: %w", err)
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

// setupPythonEnvironment detects and sets up the Python environment
func (a *Agent) setupPythonEnvironment(ctx context.Context, repoPath string) error {
	// Check for common Python environment files
	envFiles := []string{
		"requirements.txt",
		"Pipfile",
		"pyproject.toml",
		"setup.py",
		"environment.yml",
		".python-version",
	}

	pythonEnvInfo := make(map[string]interface{})
	
	for _, envFile := range envFiles {
		envFilePath := filepath.Join(repoPath, envFile)
		if _, err := os.Stat(envFilePath); err == nil {
			pythonEnvInfo[envFile] = true
		}
	}

	// Check for virtual environment directories
	venvDirs := []string{"venv", ".venv", "env", ".env"}
	for _, venvDir := range venvDirs {
		venvPath := filepath.Join(repoPath, venvDir)
		if stat, err := os.Stat(venvPath); err == nil && stat.IsDir() {
			pythonEnvInfo["virtual_env"] = venvDir
			break
		}
	}

	// Store environment info for metadata
	// This could be used later for more sophisticated environment setup
	return nil
}

// buildBanditCommand constructs the Docker command to run Bandit
func (a *Agent) buildBanditCommand(ctx context.Context, config agent.ScanConfig, repoPath, tempDir string) *exec.Cmd {
	// Install Bandit and run scan in a single command
	installAndRunScript := `#!/bin/sh
set -e

# Install Bandit
pip install --no-cache-dir bandit[toml]

# Create output directory
mkdir -p /tmp/bandit

# Build Bandit command arguments
BANDIT_ARGS="-f json -o /tmp/bandit/results.json"

# Add severity filter
if [ -n "$BANDIT_SEVERITY" ]; then
    BANDIT_ARGS="$BANDIT_ARGS -ll"
fi

# Add confidence filter
if [ -n "$BANDIT_CONFIDENCE" ]; then
    BANDIT_ARGS="$BANDIT_ARGS -i"
fi

# Add skip tests
if [ -n "$BANDIT_SKIP_TESTS" ]; then
    BANDIT_ARGS="$BANDIT_ARGS -s $BANDIT_SKIP_TESTS"
fi

# Add exclude paths
if [ -n "$BANDIT_EXCLUDE_PATHS" ]; then
    BANDIT_ARGS="$BANDIT_ARGS -x $BANDIT_EXCLUDE_PATHS"
fi

# Run Bandit scan
bandit -r /src $BANDIT_ARGS || true

# Also output to stdout for debugging
cat /tmp/bandit/results.json 2>/dev/null || echo '{"results": [], "metrics": {}}'
`

	scriptPath := filepath.Join(tempDir, "run-bandit.sh")
	os.WriteFile(scriptPath, []byte(installAndRunScript), 0755)

	args := []string{
		"run", "--rm",
		"--memory", fmt.Sprintf("%dm", a.config.MaxMemoryMB),
		"--cpus", fmt.Sprintf("%.1f", a.config.MaxCPUCores),
		"-v", fmt.Sprintf("%s:/src:ro", repoPath),
		"-v", fmt.Sprintf("%s:/tmp/bandit", tempDir),
		"-v", fmt.Sprintf("%s:/run-bandit.sh:ro", scriptPath),
		"-w", "/src",
	}

	// Add environment variables for configuration
	if a.config.Severity != "" {
		args = append(args, "-e", fmt.Sprintf("BANDIT_SEVERITY=%s", a.config.Severity))
	}
	if a.config.Confidence != "" {
		args = append(args, "-e", fmt.Sprintf("BANDIT_CONFIDENCE=%s", a.config.Confidence))
	}
	if len(a.config.SkipTests) > 0 {
		args = append(args, "-e", fmt.Sprintf("BANDIT_SKIP_TESTS=%s", strings.Join(a.config.SkipTests, ",")))
	}
	if len(a.config.ExcludePaths) > 0 {
		args = append(args, "-e", fmt.Sprintf("BANDIT_EXCLUDE_PATHS=%s", strings.Join(a.config.ExcludePaths, ",")))
	}

	args = append(args, a.config.DockerImage, "sh", "/run-bandit.sh")

	return exec.CommandContext(ctx, "docker", args...)
}

// BanditResult represents the structure of Bandit JSON output
type BanditResult struct {
	Results []BanditFinding `json:"results"`
	Metrics BanditMetrics   `json:"metrics"`
}

type BanditFinding struct {
	Code           string  `json:"code"`
	ColNumber      int     `json:"col_number"`
	Filename       string  `json:"filename"`
	IssueConfidence string  `json:"issue_confidence"`
	IssueCwe       struct {
		ID   int    `json:"id"`
		Link string `json:"link"`
	} `json:"issue_cwe"`
	IssueSeverity string `json:"issue_severity"`
	IssueText     string `json:"issue_text"`
	LineNumber    int    `json:"line_number"`
	LineRange     []int  `json:"line_range"`
	MoreInfo      string `json:"more_info"`
	TestID        string `json:"test_id"`
	TestName      string `json:"test_name"`
}

type BanditMetrics struct {
	FilesSkipped     int                    `json:"_totals.files_skipped"`
	LinesOfCode      int                    `json:"_totals.loc"`
	NoSec            int                    `json:"_totals.nosec"`
	SkippedTests     int                    `json:"_totals.skipped_tests"`
	ConfidenceLevels map[string]int         `json:"_totals.confidence"`
	SeverityLevels   map[string]int         `json:"_totals.severity"`
	RankingsByTest   map[string]interface{} `json:"_totals.ranking"`
}

// parseBanditOutput parses the JSON output from Bandit and converts it to our Finding format
func (a *Agent) parseBanditOutput(output []byte, config agent.ScanConfig) ([]agent.Finding, agent.Metadata, error) {
	var banditResult BanditResult
	if err := json.Unmarshal(output, &banditResult); err != nil {
		return nil, agent.Metadata{}, fmt.Errorf("failed to parse Bandit JSON: %w", err)
	}

	var findings []agent.Finding
	var filesScanned int
	var linesScanned int

	for _, result := range banditResult.Results {
		finding := agent.Finding{
			ID:          generateFindingID(result.TestID, result.Filename, result.LineNumber),
			Tool:        AgentName,
			RuleID:      result.TestID,
			Severity:    a.mapSeverity(result.IssueSeverity),
			Category:    a.mapCategory(result.TestID, result.TestName),
			Title:       a.getRuleTitle(result.TestID, result.TestName),
			Description: result.IssueText,
			File:        strings.TrimPrefix(result.Filename, "/src/"),
			Line:        result.LineNumber,
			Column:      result.ColNumber,
			Code:        result.Code,
			Confidence:  a.mapConfidence(result.IssueConfidence),
			References:  a.getRuleReferences(result.TestID, result.MoreInfo),
		}

		findings = append(findings, finding)
		filesScanned++
		if len(result.LineRange) >= 2 {
			linesScanned += result.LineRange[1] - result.LineRange[0] + 1
		} else {
			linesScanned++
		}
	}

	// Remove duplicates based on unique files
	uniqueFiles := make(map[string]bool)
	for _, result := range banditResult.Results {
		uniqueFiles[result.Filename] = true
	}
	filesScanned = len(uniqueFiles)

	metadata := agent.Metadata{
		ToolVersion:  a.getToolVersion(),
		RulesVersion: "latest",
		ScanType:     "sast",
		FilesScanned: filesScanned,
		LinesScanned: banditResult.Metrics.LinesOfCode,
		ExitCode:     0,
		CommandLine:  "bandit -r /src -f json",
		Environment: map[string]string{
			"files_skipped":  fmt.Sprintf("%d", banditResult.Metrics.FilesSkipped),
			"nosec_comments": fmt.Sprintf("%d", banditResult.Metrics.NoSec),
			"skipped_tests":  fmt.Sprintf("%d", banditResult.Metrics.SkippedTests),
		},
	}

	return findings, metadata, nil
}

// generateFindingID creates a unique ID for a finding
func generateFindingID(testID, file string, line int) string {
	return fmt.Sprintf("bandit-%s-%s-%d", testID, filepath.Base(file), line)
}

// mapSeverity converts Bandit severity to our standard severity levels
func (a *Agent) mapSeverity(banditSeverity string) agent.Severity {
	switch strings.ToLower(banditSeverity) {
	case "high":
		return agent.SeverityHigh
	case "medium":
		return agent.SeverityMedium
	case "low":
		return agent.SeverityLow
	default:
		return agent.SeverityMedium
	}
}

// mapCategory converts Bandit test IDs to our standard vulnerability categories
func (a *Agent) mapCategory(testID, testName string) agent.VulnCategory {
	// Map based on test ID patterns
	switch {
	case strings.Contains(testID, "B101"), strings.Contains(testName, "assert"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B102"), strings.Contains(testName, "exec"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B103"), strings.Contains(testName, "set_bad_file_permissions"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B104"), strings.Contains(testName, "hardcoded_bind_all_interfaces"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B105"), strings.Contains(testName, "hardcoded_password"):
		return agent.CategoryHardcodedSecrets
	case strings.Contains(testID, "B106"), strings.Contains(testName, "hardcoded_password"):
		return agent.CategoryHardcodedSecrets
	case strings.Contains(testID, "B107"), strings.Contains(testName, "hardcoded_password"):
		return agent.CategoryHardcodedSecrets
	case strings.Contains(testID, "B108"), strings.Contains(testName, "hardcoded_tmp_directory"):
		return agent.CategoryPathTraversal
	case strings.Contains(testID, "B110"), strings.Contains(testName, "try_except_pass"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B112"), strings.Contains(testName, "try_except_continue"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B201"), strings.Contains(testName, "flask_debug_true"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B301"), strings.Contains(testName, "pickle"):
		return agent.CategoryInsecureDeserialization
	case strings.Contains(testID, "B302"), strings.Contains(testName, "marshal"):
		return agent.CategoryInsecureDeserialization
	case strings.Contains(testID, "B303"), strings.Contains(testName, "md5"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B304"), strings.Contains(testName, "md5"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B305"), strings.Contains(testName, "cipher"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B306"), strings.Contains(testName, "mktemp_q"):
		return agent.CategoryPathTraversal
	case strings.Contains(testID, "B307"), strings.Contains(testName, "eval"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B308"), strings.Contains(testName, "mark_safe"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B309"), strings.Contains(testName, "httpsconnection"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B310"), strings.Contains(testName, "urllib"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B311"), strings.Contains(testName, "random"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B312"), strings.Contains(testName, "telnetlib"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B313"), strings.Contains(testName, "xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B314"), strings.Contains(testName, "xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B315"), strings.Contains(testName, "xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B316"), strings.Contains(testName, "xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B317"), strings.Contains(testName, "xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B318"), strings.Contains(testName, "xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B319"), strings.Contains(testName, "xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B320"), strings.Contains(testName, "xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B321"), strings.Contains(testName, "ftp"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B322"), strings.Contains(testName, "input"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B323"), strings.Contains(testName, "unverified_context"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B324"), strings.Contains(testName, "hashlib"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B325"), strings.Contains(testName, "tempfile"):
		return agent.CategoryPathTraversal
	case strings.Contains(testID, "B401"), strings.Contains(testName, "import_telnetlib"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B402"), strings.Contains(testName, "import_ftplib"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B403"), strings.Contains(testName, "import_pickle"):
		return agent.CategoryInsecureDeserialization
	case strings.Contains(testID, "B404"), strings.Contains(testName, "import_subprocess"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B405"), strings.Contains(testName, "import_xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B406"), strings.Contains(testName, "import_xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B407"), strings.Contains(testName, "import_xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B408"), strings.Contains(testName, "import_xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B409"), strings.Contains(testName, "import_xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B410"), strings.Contains(testName, "import_xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B411"), strings.Contains(testName, "import_xml"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B501"), strings.Contains(testName, "request_with_no_cert_validation"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B502"), strings.Contains(testName, "ssl_with_bad_version"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B503"), strings.Contains(testName, "ssl_with_bad_defaults"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B504"), strings.Contains(testName, "ssl_with_no_version"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B505"), strings.Contains(testName, "weak_cryptographic_key"):
		return agent.CategoryInsecureCrypto
	case strings.Contains(testID, "B506"), strings.Contains(testName, "yaml_load"):
		return agent.CategoryInsecureDeserialization
	case strings.Contains(testID, "B507"), strings.Contains(testName, "ssh_no_host_key_verification"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B601"), strings.Contains(testName, "paramiko_calls"):
		return agent.CategoryMisconfiguration
	case strings.Contains(testID, "B602"), strings.Contains(testName, "subprocess_popen_with_shell_equals_true"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B603"), strings.Contains(testName, "subprocess_without_shell_equals_true"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B604"), strings.Contains(testName, "any_other_function_with_shell_equals_true"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B605"), strings.Contains(testName, "start_process_with_a_shell"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B606"), strings.Contains(testName, "start_process_with_no_shell"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B607"), strings.Contains(testName, "start_process_with_partial_path"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B608"), strings.Contains(testName, "hardcoded_sql_expressions"):
		return agent.CategorySQLInjection
	case strings.Contains(testID, "B609"), strings.Contains(testName, "linux_commands_wildcard_injection"):
		return agent.CategoryCommandInjection
	case strings.Contains(testID, "B610"), strings.Contains(testName, "django_extra_used"):
		return agent.CategorySQLInjection
	case strings.Contains(testID, "B611"), strings.Contains(testName, "django_rawsql_used"):
		return agent.CategorySQLInjection
	case strings.Contains(testID, "B701"), strings.Contains(testName, "jinja2_autoescape_false"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B702"), strings.Contains(testName, "use_of_mako_templates"):
		return agent.CategoryXSS
	case strings.Contains(testID, "B703"), strings.Contains(testName, "django_mark_safe"):
		return agent.CategoryXSS
	default:
		return agent.CategoryOther
	}
}

// mapConfidence converts Bandit confidence to a numeric value
func (a *Agent) mapConfidence(banditConfidence string) float64 {
	switch strings.ToLower(banditConfidence) {
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

// getRuleTitle returns a human-readable title for the rule
func (a *Agent) getRuleTitle(testID, testName string) string {
	if testName != "" {
		// Convert test name to title case
		return strings.Title(strings.ReplaceAll(testName, "_", " "))
	}
	return fmt.Sprintf("Security issue: %s", testID)
}

// getRuleReferences returns documentation references for the rule
func (a *Agent) getRuleReferences(testID, moreInfo string) []string {
	references := []string{}
	
	if moreInfo != "" {
		references = append(references, moreInfo)
	}
	
	// Add Bandit documentation link
	if testID != "" {
		references = append(references, fmt.Sprintf("https://bandit.readthedocs.io/en/latest/plugins/%s.html", strings.ToLower(testID)))
	}
	
	return references
}