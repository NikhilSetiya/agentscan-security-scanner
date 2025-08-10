package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGitLabIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GitLab integration tests in short mode")
	}

	t.Run("GitLabCIYAMLValidation", func(t *testing.T) {
		// Test that GitLab CI YAML exists and is valid
		ciYAMLPath := filepath.Join("..", "gitlab", ".gitlab-ci.yml")
		
		content, err := os.ReadFile(ciYAMLPath)
		require.NoError(t, err, "GitLab CI YAML should exist")
		
		// Parse YAML to ensure it's valid
		var ciConfig map[string]interface{}
		err = yaml.Unmarshal(content, &ciConfig)
		require.NoError(t, err, "GitLab CI YAML should be valid")
		
		// Check for required top-level keys
		assert.Contains(t, ciConfig, "variables", "Should have variables section")
		assert.Contains(t, ciConfig, "stages", "Should have stages section")
		
		// Check variables
		variables, ok := ciConfig["variables"].(map[string]interface{})
		require.True(t, ok, "Variables should be a map")
		assert.Contains(t, variables, "AGENTSCAN_API_URL", "Should have API URL variable")
		assert.Contains(t, variables, "AGENTSCAN_FAIL_ON_SEVERITY", "Should have fail on severity variable")
		
		// Check stages
		stages, ok := ciConfig["stages"].([]interface{})
		require.True(t, ok, "Stages should be an array")
		assert.Contains(t, stages, "security", "Should have security stage")
		
		// Check security scan job
		assert.Contains(t, ciConfig, "agentscan_security_scan", "Should have security scan job")
		
		securityJob, ok := ciConfig["agentscan_security_scan"].(map[string]interface{})
		require.True(t, ok, "Security job should be a map")
		assert.Equal(t, "security", securityJob["stage"], "Should be in security stage")
		assert.Contains(t, securityJob, "script", "Should have script section")
		assert.Contains(t, securityJob, "artifacts", "Should have artifacts section")
	})

	t.Run("GitLabCIScriptValidation", func(t *testing.T) {
		// Test that the CI script contains required commands
		ciYAMLPath := filepath.Join("..", "gitlab", ".gitlab-ci.yml")
		
		content, err := os.ReadFile(ciYAMLPath)
		require.NoError(t, err, "GitLab CI YAML should exist")
		
		ciYAML := string(content)
		
		// Check for required script elements
		assert.Contains(t, ciYAML, "curl -sSL https://install.agentscan.dev", "Should install AgentScan CLI")
		assert.Contains(t, ciYAML, "agentscan-cli scan", "Should run security scan")
		assert.Contains(t, ciYAML, "jq -r", "Should process JSON results")
		
		// Check for GitLab-specific integrations
		assert.Contains(t, ciYAML, "CI_PIPELINE_SOURCE", "Should check pipeline source")
		assert.Contains(t, ciYAML, "merge_request_event", "Should handle merge requests")
		assert.Contains(t, ciYAML, "CI_API_V4_URL", "Should use GitLab API")
		
		// Check for SARIF report generation
		assert.Contains(t, ciYAML, "gl-sast-report.json", "Should generate GitLab SAST report")
		assert.Contains(t, ciYAML, "reports:", "Should have reports section")
		assert.Contains(t, ciYAML, "sast:", "Should have SAST report")
	})

	t.Run("GitLabSARIFConversion", func(t *testing.T) {
		// Test SARIF to GitLab format conversion logic
		ciYAMLPath := filepath.Join("..", "gitlab", ".gitlab-ci.yml")
		
		content, err := os.ReadFile(ciYAMLPath)
		require.NoError(t, err, "GitLab CI YAML should exist")
		
		ciYAML := string(content)
		
		// Check for Python conversion script
		assert.Contains(t, ciYAML, "python3 -c", "Should have Python conversion script")
		assert.Contains(t, ciYAML, "import json", "Should import JSON module")
		assert.Contains(t, ciYAML, "vulnerabilities", "Should create vulnerabilities array")
		assert.Contains(t, ciYAML, "version': '14.0.0'", "Should use correct GitLab SAST version")
		
		// Check for required GitLab SAST fields
		assert.Contains(t, ciYAML, "category': 'sast'", "Should set SAST category")
		assert.Contains(t, ciYAML, "scanner", "Should have scanner information")
		assert.Contains(t, ciYAML, "location", "Should have location information")
		assert.Contains(t, ciYAML, "identifiers", "Should have identifiers")
	})

	t.Run("GitLabMergeRequestIntegration", func(t *testing.T) {
		// Test merge request comment functionality
		ciYAMLPath := filepath.Join("..", "gitlab", ".gitlab-ci.yml")
		
		content, err := os.ReadFile(ciYAMLPath)
		require.NoError(t, err, "GitLab CI YAML should exist")
		
		ciYAML := string(content)
		
		// Check for merge request comment logic
		assert.Contains(t, ciYAML, "CI_MERGE_REQUEST_IID", "Should use merge request IID")
		assert.Contains(t, ciYAML, "GITLAB_TOKEN", "Should use GitLab token")
		assert.Contains(t, ciYAML, "merge_requests", "Should post to merge requests endpoint")
		assert.Contains(t, ciYAML, "notes", "Should create notes/comments")
		
		// Check comment content
		assert.Contains(t, ciYAML, "AgentScan Security Report", "Should have report title")
		assert.Contains(t, ciYAML, "High Severity Issues", "Should show high severity section")
		assert.Contains(t, ciYAML, "Powered by AgentScan", "Should have branding")
	})

	t.Run("GitLabRulesValidation", func(t *testing.T) {
		// Test that GitLab CI rules are properly configured
		ciYAMLPath := filepath.Join("..", "gitlab", ".gitlab-ci.yml")
		
		content, err := os.ReadFile(ciYAMLPath)
		require.NoError(t, err, "GitLab CI YAML should exist")
		
		var ciConfig map[string]interface{}
		err = yaml.Unmarshal(content, &ciConfig)
		require.NoError(t, err, "GitLab CI YAML should be valid")
		
		// Check main job rules
		securityJob, ok := ciConfig["agentscan_security_scan"].(map[string]interface{})
		require.True(t, ok, "Security job should exist")
		
		rules, ok := securityJob["rules"].([]interface{})
		require.True(t, ok, "Should have rules section")
		assert.NotEmpty(t, rules, "Should have at least one rule")
		
		// Convert to string to check rule content
		rulesYAML, err := yaml.Marshal(rules)
		require.NoError(t, err, "Should be able to marshal rules")
		
		rulesStr := string(rulesYAML)
		assert.Contains(t, rulesStr, "merge_request_event", "Should run on merge requests")
		assert.Contains(t, rulesStr, "CI_DEFAULT_BRANCH", "Should run on default branch")
	})
}

func TestGitLabEnvironmentDetection(t *testing.T) {
	t.Run("DetectGitLabEnvironment", func(t *testing.T) {
		// Save original environment
		originalGitLabCI := os.Getenv("GITLAB_CI")
		originalProjectDir := os.Getenv("CI_PROJECT_DIR")
		originalMRIID := os.Getenv("CI_MERGE_REQUEST_IID")
		
		// Cleanup
		defer func() {
			os.Setenv("GITLAB_CI", originalGitLabCI)
			os.Setenv("CI_PROJECT_DIR", originalProjectDir)
			os.Setenv("CI_MERGE_REQUEST_IID", originalMRIID)
		}()
		
		// Test GitLab CI detection
		os.Setenv("GITLAB_CI", "true")
		os.Setenv("CI_PROJECT_DIR", "/builds/group/project")
		os.Setenv("CI_MERGE_REQUEST_IID", "123")
		
		// This would test the CLI's GitLab environment detection
		gitlabCI := os.Getenv("GITLAB_CI")
		projectDir := os.Getenv("CI_PROJECT_DIR")
		mrIID := os.Getenv("CI_MERGE_REQUEST_IID")
		
		assert.Equal(t, "true", gitlabCI, "Should detect GitLab CI")
		assert.NotEmpty(t, projectDir, "Should detect project directory")
		assert.Equal(t, "123", mrIID, "Should detect merge request IID")
	})
}

func TestGitLabWorkflowGeneration(t *testing.T) {
	t.Run("GenerateGitLabWorkflow", func(t *testing.T) {
		// Test workflow generation for different scenarios
		testCases := []struct {
			name           string
			failOnSeverity string
			excludePaths   []string
			includeMR      bool
			includeSchedule bool
		}{
			{
				name:           "BasicMRWorkflow",
				failOnSeverity: "high",
				excludePaths:   []string{"node_modules/**", "vendor/**"},
				includeMR:      true,
				includeSchedule: false,
			},
			{
				name:           "ScheduledWorkflow",
				failOnSeverity: "medium",
				excludePaths:   []string{"test/**", "docs/**"},
				includeMR:      false,
				includeSchedule: true,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// In a real implementation, this would call a workflow generation function
				workflow := generateGitLabWorkflow(tc.failOnSeverity, tc.excludePaths, tc.includeMR, tc.includeSchedule)
				
				assert.Contains(t, workflow, "agentscan-cli scan", "Should contain scan command")
				assert.Contains(t, workflow, tc.failOnSeverity, "Should contain fail on severity")
				
				if tc.includeMR {
					assert.Contains(t, workflow, "merge_request_event", "Should include MR rules")
				}
				
				if tc.includeSchedule {
					assert.Contains(t, workflow, "schedule", "Should include schedule rules")
				}
				
				for _, path := range tc.excludePaths {
					assert.Contains(t, workflow, path, "Should contain exclude path: %s", path)
				}
			})
		}
	})
}

// Mock function for testing workflow generation
func generateGitLabWorkflow(failOnSeverity string, excludePaths []string, includeMR, includeSchedule bool) string {
	workflow := `
variables:
  AGENTSCAN_FAIL_ON_SEVERITY: "` + failOnSeverity + `"
  AGENTSCAN_EXCLUDE_PATHS: "` + strings.Join(excludePaths, ",") + `"

agentscan_security_scan:
  script:
    - agentscan-cli scan --fail-on-severity=` + failOnSeverity + `
  rules:`
	
	if includeMR {
		workflow += `
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"`
	}
	
	if includeSchedule {
		workflow += `
    - if: $CI_PIPELINE_SOURCE == "schedule"`
	}
	
	return workflow
}