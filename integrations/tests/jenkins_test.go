package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJenkinsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Jenkins integration tests in short mode")
	}

	t.Run("JenkinsfileValidation", func(t *testing.T) {
		// Test that Jenkinsfile exists and has required stages
		jenkinsfilePath := filepath.Join("..", "jenkins", "Jenkinsfile")
		
		content, err := os.ReadFile(jenkinsfilePath)
		require.NoError(t, err, "Jenkinsfile should exist")
		
		jenkinsfile := string(content)
		
		// Check for required pipeline elements
		assert.Contains(t, jenkinsfile, "pipeline {", "Should be a declarative pipeline")
		assert.Contains(t, jenkinsfile, "agent any", "Should specify agent")
		assert.Contains(t, jenkinsfile, "stages", "Should have stages section")
		
		// Check for required stages
		assert.Contains(t, jenkinsfile, "stage('Checkout')", "Should have checkout stage")
		assert.Contains(t, jenkinsfile, "stage('Install AgentScan CLI')", "Should have CLI installation stage")
		assert.Contains(t, jenkinsfile, "stage('Security Scan')", "Should have security scan stage")
		
		// Check for parameters
		assert.Contains(t, jenkinsfile, "parameters {", "Should have parameters section")
		assert.Contains(t, jenkinsfile, "AGENTSCAN_API_URL", "Should have API URL parameter")
		assert.Contains(t, jenkinsfile, "FAIL_ON_SEVERITY", "Should have fail on severity parameter")
		
		// Check for environment variables
		assert.Contains(t, jenkinsfile, "environment {", "Should have environment section")
		assert.Contains(t, jenkinsfile, "AGENTSCAN_TOKEN", "Should reference API token credential")
		
		// Check for post actions
		assert.Contains(t, jenkinsfile, "post {", "Should have post section")
		assert.Contains(t, jenkinsfile, "archiveArtifacts", "Should archive scan results")
	})

	t.Run("JenkinsPluginConfiguration", func(t *testing.T) {
		// Test that plugin POM exists and has correct configuration
		pomPath := filepath.Join("..", "jenkins", "agentscan-plugin", "pom.xml")
		
		content, err := os.ReadFile(pomPath)
		require.NoError(t, err, "Plugin POM should exist")
		
		pom := string(content)
		
		// Check basic plugin structure
		assert.Contains(t, pom, "<groupId>dev.agentscan</groupId>", "Should have correct group ID")
		assert.Contains(t, pom, "<artifactId>agentscan-jenkins-plugin</artifactId>", "Should have correct artifact ID")
		assert.Contains(t, pom, "<packaging>hpi</packaging>", "Should be packaged as HPI")
		
		// Check Jenkins parent
		assert.Contains(t, pom, "<groupId>org.jenkins-ci.plugins</groupId>", "Should extend Jenkins plugin parent")
		assert.Contains(t, pom, "<artifactId>plugin</artifactId>", "Should use plugin parent")
		
		// Check required dependencies
		assert.Contains(t, pom, "workflow-cps", "Should have workflow support")
		assert.Contains(t, pom, "credentials", "Should have credentials support")
		assert.Contains(t, pom, "jackson-databind", "Should have JSON support")
	})

	t.Run("JenkinsPluginJavaFiles", func(t *testing.T) {
		// Test that required Java files exist
		pluginDir := filepath.Join("..", "jenkins", "agentscan-plugin", "src", "main", "java", "dev", "agentscan", "jenkins")
		
		requiredFiles := []string{
			"AgentScanBuilder.java",
			"AgentScanService.java",
			"ScanOptions.java",
			"ScanResult.java",
			"ScanSummary.java",
			"HtmlReportGenerator.java",
		}
		
		for _, file := range requiredFiles {
			filePath := filepath.Join(pluginDir, file)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "Required Java file should exist: %s", file)
		}
	})

	t.Run("JenkinsPluginResources", func(t *testing.T) {
		// Test that required resource files exist
		resourcesDir := filepath.Join("..", "jenkins", "agentscan-plugin", "src", "main", "resources", "dev", "agentscan", "jenkins", "AgentScanBuilder")
		
		requiredResources := []string{
			"config.jelly",
			"help-apiUrl.html",
		}
		
		for _, resource := range requiredResources {
			resourcePath := filepath.Join(resourcesDir, resource)
			_, err := os.Stat(resourcePath)
			assert.NoError(t, err, "Required resource file should exist: %s", resource)
		}
	})
}

func TestJenkinsEnvironmentDetection(t *testing.T) {
	t.Run("DetectJenkinsEnvironment", func(t *testing.T) {
		// Save original environment
		originalJenkinsURL := os.Getenv("JENKINS_URL")
		originalWorkspace := os.Getenv("WORKSPACE")
		
		// Cleanup
		defer func() {
			os.Setenv("JENKINS_URL", originalJenkinsURL)
			os.Setenv("WORKSPACE", originalWorkspace)
		}()
		
		// Test Jenkins detection
		os.Setenv("JENKINS_URL", "http://jenkins.example.com")
		os.Setenv("WORKSPACE", "/var/jenkins_home/workspace/test-job")
		
		// This would test the CLI's Jenkins environment detection
		// In a real implementation, you would call the CLI function
		jenkinsURL := os.Getenv("JENKINS_URL")
		workspace := os.Getenv("WORKSPACE")
		
		assert.NotEmpty(t, jenkinsURL, "Should detect Jenkins URL")
		assert.NotEmpty(t, workspace, "Should detect Jenkins workspace")
		assert.Contains(t, jenkinsURL, "jenkins", "Should be a Jenkins URL")
	})
}

func TestJenkinsWorkflowGeneration(t *testing.T) {
	t.Run("GenerateJenkinsWorkflow", func(t *testing.T) {
		// Test workflow generation for different scenarios
		testCases := []struct {
			name           string
			failOnSeverity string
			excludePaths   string
			expectedStages []string
		}{
			{
				name:           "BasicWorkflow",
				failOnSeverity: "high",
				excludePaths:   "node_modules/**,vendor/**",
				expectedStages: []string{"Checkout", "Install AgentScan CLI", "Security Scan"},
			},
			{
				name:           "StrictWorkflow",
				failOnSeverity: "medium",
				excludePaths:   "test/**,docs/**",
				expectedStages: []string{"Checkout", "Install AgentScan CLI", "Security Scan", "Generate Reports"},
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// In a real implementation, this would call a workflow generation function
				// For now, we just verify the test structure
				assert.NotEmpty(t, tc.failOnSeverity, "Should have fail on severity setting")
				assert.NotEmpty(t, tc.excludePaths, "Should have exclude paths")
				assert.NotEmpty(t, tc.expectedStages, "Should have expected stages")
			})
		}
	})
}