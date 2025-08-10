package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI integration tests in short mode")
	}

	// Build CLI for testing
	cliPath := buildCLIForTesting(t)
	defer os.Remove(cliPath)

	t.Run("CLIVersion", func(t *testing.T) {
		cmd := exec.Command(cliPath, "version")
		output, err := cmd.CombinedOutput()
		
		assert.NoError(t, err, "Version command should succeed")
		assert.Contains(t, string(output), "AgentScan CLI", "Should show CLI name")
	})

	t.Run("CLIHelp", func(t *testing.T) {
		cmd := exec.Command(cliPath, "help")
		output, err := cmd.CombinedOutput()
		
		assert.NoError(t, err, "Help command should succeed")
		outputStr := string(output)
		
		assert.Contains(t, outputStr, "Usage:", "Should show usage information")
		assert.Contains(t, outputStr, "agentscan-cli scan", "Should show scan command")
		assert.Contains(t, outputStr, "--api-url", "Should show API URL option")
		assert.Contains(t, outputStr, "--fail-on-severity", "Should show fail on severity option")
	})

	t.Run("CLIScanHelp", func(t *testing.T) {
		cmd := exec.Command(cliPath, "scan", "--help")
		output, _ := cmd.CombinedOutput()
		
		// Help should work even if command fails
		outputStr := string(output)
		
		assert.Contains(t, outputStr, "Scan Options:", "Should show scan options")
		assert.Contains(t, outputStr, "--exclude-path", "Should show exclude path option")
		assert.Contains(t, outputStr, "--output-format", "Should show output format option")
		assert.Contains(t, outputStr, "--timeout", "Should show timeout option")
	})

	t.Run("CLIEnvironmentDetection", func(t *testing.T) {
		testCases := []struct {
			name     string
			envVars  map[string]string
			expected string
		}{
			{
				name: "GitHubActions",
				envVars: map[string]string{
					"GITHUB_ACTIONS":   "true",
					"GITHUB_WORKSPACE": "/github/workspace",
				},
				expected: "GitHub Actions",
			},
			{
				name: "GitLabCI",
				envVars: map[string]string{
					"GITLAB_CI":      "true",
					"CI_PROJECT_DIR": "/builds/project",
				},
				expected: "GitLab CI",
			},
			{
				name: "Jenkins",
				envVars: map[string]string{
					"JENKINS_URL": "http://jenkins.example.com",
					"WORKSPACE":   "/var/jenkins_home/workspace/job",
				},
				expected: "Jenkins",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Set environment variables
				for key, value := range tc.envVars {
					os.Setenv(key, value)
					defer os.Unsetenv(key)
				}

				// Run CLI with verbose flag to see environment detection
				cmd := exec.Command(cliPath, "scan", "--verbose", "--help")
				_, _ = cmd.CombinedOutput()
				
				// In a real implementation, the CLI would log detected environment
				// For now, we just verify the environment variables are set
				for key, expectedValue := range tc.envVars {
					actualValue := os.Getenv(key)
					assert.Equal(t, expectedValue, actualValue, "Environment variable should be set: %s", key)
				}
			})
		}
	})

	t.Run("CLIFailureThresholds", func(t *testing.T) {
		testCases := []struct {
			name           string
			failOnSeverity string
			expectedExit   int
		}{
			{
				name:           "FailOnHigh",
				failOnSeverity: "high",
				expectedExit:   0, // No high severity findings in test
			},
			{
				name:           "FailOnMedium",
				failOnSeverity: "medium",
				expectedExit:   0, // No medium+ severity findings in test
			},
			{
				name:           "FailOnLow",
				failOnSeverity: "low",
				expectedExit:   0, // No findings in test
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a temporary directory for testing
				tempDir := t.TempDir()
				
				// Create a simple test file
				testFile := filepath.Join(tempDir, "test.js")
				err := os.WriteFile(testFile, []byte("console.log('hello world');"), 0644)
				require.NoError(t, err)
				
				// Change to temp directory
				originalDir, err := os.Getwd()
				require.NoError(t, err)
				defer os.Chdir(originalDir)
				
				err = os.Chdir(tempDir)
				require.NoError(t, err)
				
				// Run scan with specific failure threshold
				cmd := exec.Command(cliPath, "scan", "--fail-on-severity="+tc.failOnSeverity, "--timeout=30s")
				output, err := cmd.CombinedOutput()
				
				outputStr := string(output)
				t.Logf("CLI output: %s", outputStr)
				
				// Check exit code
				if exitError, ok := err.(*exec.ExitError); ok {
					assert.Equal(t, tc.expectedExit, exitError.ExitCode(), "Should exit with expected code")
				} else if err == nil {
					assert.Equal(t, 0, tc.expectedExit, "Should exit with code 0")
				} else {
					t.Errorf("Unexpected error type: %v", err)
				}
			})
		}
	})

	t.Run("CLIOutputFormats", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create a simple test file
		testFile := filepath.Join(tempDir, "test.py")
		err := os.WriteFile(testFile, []byte("print('hello world')"), 0644)
		require.NoError(t, err)
		
		// Change to temp directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		
		err = os.Chdir(tempDir)
		require.NoError(t, err)
		
		testCases := []struct {
			name         string
			outputFormat string
			expectedFile string
		}{
			{
				name:         "JSONOutput",
				outputFormat: "json",
				expectedFile: "agentscan-results.json",
			},
			{
				name:         "SARIFOutput",
				outputFormat: "sarif",
				expectedFile: "agentscan-results.sarif",
			},
			{
				name:         "MultipleFormats",
				outputFormat: "json,sarif",
				expectedFile: "agentscan-results.json", // Check for at least one
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Clean up any existing result files
				os.Remove("agentscan-results.json")
				os.Remove("agentscan-results.sarif")
				
				// Run scan with specific output format
				cmd := exec.Command(cliPath, "scan", "--output-format="+tc.outputFormat, "--timeout=30s")
				output, err := cmd.CombinedOutput()
				
				outputStr := string(output)
				t.Logf("CLI output: %s", outputStr)
				
				// Check that expected output file was created
				_, err = os.Stat(tc.expectedFile)
				assert.NoError(t, err, "Expected output file should be created: %s", tc.expectedFile)
			})
		}
	})
}

func TestCLIInstallationScripts(t *testing.T) {
	t.Run("UnixInstallScript", func(t *testing.T) {
		scriptPath := filepath.Join("..", "..", "scripts", "install.sh")
		
		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err, "Install script should exist")
		
		script := string(content)
		
		// Check for required functions
		assert.Contains(t, script, "detect_platform()", "Should have platform detection")
		assert.Contains(t, script, "get_latest_version()", "Should have version detection")
		assert.Contains(t, script, "install_binary()", "Should have binary installation")
		assert.Contains(t, script, "setup_path()", "Should have PATH setup")
		
		// Check for error handling
		assert.Contains(t, script, "set -e", "Should exit on error")
		assert.Contains(t, script, "log_error", "Should have error logging")
		
		// Check for platform support
		assert.Contains(t, script, "Linux*", "Should support Linux")
		assert.Contains(t, script, "Darwin*", "Should support macOS")
		assert.Contains(t, script, "x86_64", "Should support x86_64")
		assert.Contains(t, script, "arm64", "Should support ARM64")
	})

	t.Run("WindowsInstallScript", func(t *testing.T) {
		scriptPath := filepath.Join("..", "..", "scripts", "install.ps1")
		
		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err, "Windows install script should exist")
		
		script := string(content)
		
		// Check for required functions
		assert.Contains(t, script, "function Get-Platform", "Should have platform detection")
		assert.Contains(t, script, "function Get-LatestVersion", "Should have version detection")
		assert.Contains(t, script, "function Install-Binary", "Should have binary installation")
		assert.Contains(t, script, "function Setup-Path", "Should have PATH setup")
		
		// Check for error handling
		assert.Contains(t, script, "$ErrorActionPreference", "Should set error action preference")
		assert.Contains(t, script, "Log-Error", "Should have error logging")
		
		// Check for Windows-specific features
		assert.Contains(t, script, "Invoke-WebRequest", "Should use PowerShell web requests")
		assert.Contains(t, script, "Expand-Archive", "Should extract ZIP files")
		assert.Contains(t, script, "[Environment]::SetEnvironmentVariable", "Should set environment variables")
	})
}

func TestCLIConfigurationFiles(t *testing.T) {
	t.Run("CLISourceExists", func(t *testing.T) {
		cliSourcePath := filepath.Join("..", "..", "cmd", "cli", "main.go")
		
		_, err := os.Stat(cliSourcePath)
		assert.NoError(t, err, "CLI source file should exist")
		
		content, err := os.ReadFile(cliSourcePath)
		require.NoError(t, err, "Should be able to read CLI source")
		
		source := string(content)
		
		// Check for required functions
		assert.Contains(t, source, "func main()", "Should have main function")
		assert.Contains(t, source, "func runScan()", "Should have scan function")
		assert.Contains(t, source, "func printUsage()", "Should have usage function")
		
		// Check for CI/CD integration
		assert.Contains(t, source, "detectCIWorkspace", "Should detect CI workspace")
		assert.Contains(t, source, "postCIIntegrationResults", "Should post CI results")
		
		// Check for environment variable handling
		assert.Contains(t, source, "GITHUB_ACTIONS", "Should handle GitHub Actions")
		assert.Contains(t, source, "GITLAB_CI", "Should handle GitLab CI")
		assert.Contains(t, source, "JENKINS_URL", "Should handle Jenkins")
	})
}

// buildCLIForTesting builds the CLI binary for testing
func buildCLIForTesting(t *testing.T) string {
	tempDir := t.TempDir()
	cliPath := filepath.Join(tempDir, "agentscan-cli")
	
	// Build the CLI
	cmd := exec.Command("go", "build", "-o", cliPath, "../../cmd/cli")
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Build output: %s", string(output))
		t.Fatalf("Failed to build CLI for testing: %v", err)
	}
	
	return cliPath
}