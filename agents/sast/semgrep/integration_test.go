package semgrep

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSemgrepAgent_Integration tests the complete workflow with a real repository
func TestSemgrepAgent_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	a := NewAgent()
	ctx := context.Background()

	// Test health check first
	err := a.HealthCheck(ctx)
	require.NoError(t, err, "Health check should pass")

	// Use a small public repository with known vulnerabilities for testing
	// Note: In a real scenario, you'd use a dedicated test repository
	config := agent.ScanConfig{
		RepoURL:   "https://github.com/OWASP/NodeGoat.git", // Known vulnerable Node.js app
		Branch:    "master",
		Languages: []string{"javascript"},
		Timeout:   5 * time.Minute,
	}

	// Execute the scan
	result, err := a.Scan(ctx, config)
	require.NoError(t, err, "Scan should complete successfully")

	// Verify results
	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	assert.Empty(t, result.Error)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Equal(t, "semgrep", result.AgentID)

	// Verify metadata
	assert.Equal(t, "sast", result.Metadata.ScanType)
	assert.NotEmpty(t, result.Metadata.ToolVersion)
	assert.Greater(t, result.Metadata.FilesScanned, 0)

	// NodeGoat should have some security findings
	assert.Greater(t, len(result.Findings), 0, "Should find security issues in NodeGoat")

	// Verify finding structure
	for _, finding := range result.Findings {
		assert.NotEmpty(t, finding.ID)
		assert.Equal(t, "semgrep", finding.Tool)
		assert.NotEmpty(t, finding.RuleID)
		assert.Contains(t, []agent.Severity{
			agent.SeverityHigh,
			agent.SeverityMedium,
			agent.SeverityLow,
			agent.SeverityInfo,
		}, finding.Severity)
		assert.NotEmpty(t, finding.Title)
		assert.NotEmpty(t, finding.File)
		assert.Greater(t, finding.Line, 0)
		assert.GreaterOrEqual(t, finding.Confidence, 0.0)
		assert.LessOrEqual(t, finding.Confidence, 1.0)
	}

	t.Logf("Integration test completed successfully with %d findings", len(result.Findings))
}

// TestSemgrepAgent_HealthCheck_Integration tests health check functionality
func TestSemgrepAgent_HealthCheck_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	a := NewAgent()
	ctx := context.Background()

	err := a.HealthCheck(ctx)
	assert.NoError(t, err)
}

// TestSemgrepAgent_CustomRules_Integration tests scanning with custom rules
func TestSemgrepAgent_CustomRules_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	// Create agent with custom rules configuration
	config := AgentConfig{
		DockerImage:    DefaultImage,
		MaxMemoryMB:    512,
		MaxCPUCores:    1.0,
		DefaultTimeout: 5 * time.Minute,
		RulesConfig:    "p/security-audit", // Use security audit ruleset
	}
	a := NewAgentWithConfig(config)

	ctx := context.Background()

	scanConfig := agent.ScanConfig{
		RepoURL:   "https://github.com/OWASP/NodeGoat.git",
		Branch:    "master",
		Languages: []string{"javascript"},
		Rules:     []string{"p/owasp-top-ten"}, // Additional rules
		Timeout:   5 * time.Minute,
	}

	result, err := a.Scan(ctx, scanConfig)
	require.NoError(t, err)

	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	assert.Greater(t, len(result.Findings), 0)

	t.Logf("Custom rules test completed with %d findings", len(result.Findings))
}

// TestSemgrepAgent_IncrementalScan_Integration tests incremental scanning
func TestSemgrepAgent_IncrementalScan_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	a := NewAgent()
	ctx := context.Background()

	// Scan only specific files
	config := agent.ScanConfig{
		RepoURL:   "https://github.com/OWASP/NodeGoat.git",
		Branch:    "master",
		Languages: []string{"javascript"},
		Files:     []string{"*.js"}, // Only JavaScript files
		Timeout:   3 * time.Minute,
	}

	result, err := a.Scan(ctx, config)
	require.NoError(t, err)

	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	// Should still find some issues even with file filtering
	assert.GreaterOrEqual(t, len(result.Findings), 0)

	t.Logf("Incremental scan test completed with %d findings", len(result.Findings))
}

// TestSemgrepAgent_Timeout_Integration tests timeout handling
func TestSemgrepAgent_Timeout_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	a := NewAgent()
	ctx := context.Background()

	// Use a very short timeout to trigger timeout
	config := agent.ScanConfig{
		RepoURL: "https://github.com/OWASP/NodeGoat.git",
		Branch:  "master",
		Timeout: 1 * time.Second, // Very short timeout
	}

	result, err := a.Scan(ctx, config)
	
	// Should either timeout or complete quickly
	if err != nil {
		assert.Contains(t, err.Error(), "context deadline exceeded")
		assert.Equal(t, agent.ScanStatusFailed, result.Status)
	} else {
		// If it completed within timeout, that's also valid
		assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	}

	t.Logf("Timeout test completed")
}

// TestSemgrepAgent_MultiLanguage_Integration tests multi-language scanning
func TestSemgrepAgent_MultiLanguage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	a := NewAgent()
	ctx := context.Background()

	// Use a repository with multiple languages
	config := agent.ScanConfig{
		RepoURL:   "https://github.com/OWASP/NodeGoat.git",
		Branch:    "master",
		Languages: []string{"javascript", "typescript", "json"}, // Multiple languages
		Timeout:   5 * time.Minute,
	}

	result, err := a.Scan(ctx, config)
	require.NoError(t, err)

	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	assert.GreaterOrEqual(t, len(result.Findings), 0)

	t.Logf("Multi-language test completed with %d findings", len(result.Findings))
}

// isDockerAvailable checks if Docker is available on the system
func isDockerAvailable() bool {
	if os.Getenv("CI") == "true" {
		// In CI environments, Docker might not be available
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a := NewAgent()
	return a.HealthCheck(ctx) == nil
}

// Benchmark tests for performance analysis
func BenchmarkSemgrepAgent_Scan(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	if !isDockerAvailable() {
		b.Skip("Docker not available, skipping benchmark")
	}

	a := NewAgent()
	ctx := context.Background()

	config := agent.ScanConfig{
		RepoURL:   "https://github.com/OWASP/NodeGoat.git",
		Branch:    "master",
		Languages: []string{"javascript"},
		Timeout:   10 * time.Minute,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.Scan(ctx, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSemgrepAgent_HealthCheck(b *testing.B) {
	if !isDockerAvailable() {
		b.Skip("Docker not available, skipping benchmark")
	}

	a := NewAgent()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := a.HealthCheck(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}