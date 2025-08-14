package zap

import (
	"context"
	"testing"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZAPAgentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	zapAgent := NewAgent()
	
	// Test agent configuration
	config := zapAgent.GetConfig()
	assert.Equal(t, "zap", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.True(t, config.RequiresDocker)
	assert.Contains(t, config.SupportedLangs, "javascript")
	assert.Contains(t, config.SupportedLangs, "python")
	assert.Contains(t, config.Categories, agent.CategoryXSS)
	assert.Contains(t, config.Categories, agent.CategorySQLInjection)

	// Test version info
	versionInfo := zapAgent.GetVersion()
	assert.Equal(t, "1.0.0", versionInfo.AgentVersion)
	assert.NotEmpty(t, versionInfo.BuildDate)

	// Test health check (may fail if Docker not available)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := zapAgent.HealthCheck(ctx)
	if err != nil {
		t.Logf("ZAP agent health check failed (expected in CI without Docker): %v", err)
		t.Skip("Docker or ZAP not available, skipping scan test")
	} else {
		t.Log("ZAP agent health check passed")
	}

	// Test scan with a non-web repository (should gracefully skip DAST)
	scanConfig := agent.ScanConfig{
		RepoURL:   "https://github.com/octocat/Hello-World.git",
		Branch:    "master",
		Languages: []string{"markdown"},
		Timeout:   2 * time.Minute,
	}

	result, err := zapAgent.Scan(ctx, scanConfig)
	require.NoError(t, err)
	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	assert.Equal(t, "zap", result.AgentID)
	assert.Equal(t, "dast", result.Metadata.ScanType)
	// Should have no findings since it's not a web app
	assert.Empty(t, result.Findings)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestZAPAgentWebAppDetection(t *testing.T) {
	zapAgent := NewAgent()

	tests := []struct {
		name        string
		repoURL     string
		shouldDetect bool
		description string
	}{
		{
			name:        "non-web repository",
			repoURL:     "https://github.com/octocat/Hello-World.git",
			shouldDetect: false,
			description: "Simple Hello World repository without web framework",
		},
		// Note: We can't easily test actual web app detection without
		// setting up test repositories, but the unit tests cover the
		// detection logic thoroughly
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			// For this test, we just verify the agent handles non-web repos gracefully
			scanConfig := agent.ScanConfig{
				RepoURL: tt.repoURL,
				Branch:  "master",
				Timeout: 1 * time.Minute,
			}

			result, err := zapAgent.Scan(ctx, scanConfig)
			
			if !tt.shouldDetect {
				// Should complete successfully but with no findings
				require.NoError(t, err)
				assert.Equal(t, agent.ScanStatusCompleted, result.Status)
				assert.Empty(t, result.Findings)
			}
		})
	}
}

func TestZAPAgentErrorHandling(t *testing.T) {
	zapAgent := NewAgent()

	tests := []struct {
		name        string
		config      agent.ScanConfig
		expectError bool
		description string
	}{
		{
			name: "invalid repository URL",
			config: agent.ScanConfig{
				RepoURL: "invalid-url",
				Timeout: 30 * time.Second,
			},
			expectError: true,
			description: "Should handle invalid repository URLs gracefully",
		},
		{
			name: "empty repository URL",
			config: agent.ScanConfig{
				RepoURL: "",
				Timeout: 30 * time.Second,
			},
			expectError: true,
			description: "Should handle empty repository URLs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()

			result, err := zapAgent.Scan(ctx, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if result != nil {
					assert.Equal(t, agent.ScanStatusFailed, result.Status)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, agent.ScanStatusCompleted, result.Status)
			}
		})
	}
}

func TestZAPAgentTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	zapAgent := NewAgent()

	// Test with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/octocat/Hello-World.git",
		Branch:  "master",
		Timeout: 1 * time.Second,
	}

	result, err := zapAgent.Scan(ctx, scanConfig)

	// Should either complete quickly (if it's not a web app) or timeout
	if err != nil {
		// Timeout is acceptable
		assert.Contains(t, err.Error(), "context deadline exceeded")
	} else {
		// Or complete successfully if detection is fast
		assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	}
}