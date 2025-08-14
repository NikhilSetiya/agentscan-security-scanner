package zap

import (
	"context"
	"testing"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	zapAgent := NewAgent()
	
	assert.NotNil(t, zapAgent)
	assert.Equal(t, DefaultImage, zapAgent.config.DockerImage)
	assert.Equal(t, 1024, zapAgent.config.MaxMemoryMB)
	assert.Equal(t, 1.0, zapAgent.config.MaxCPUCores)
	assert.Equal(t, 10*time.Minute, zapAgent.config.DefaultTimeout)
	assert.Equal(t, "baseline", zapAgent.config.ScanType)
}

func TestNewAgentWithConfig(t *testing.T) {
	customConfig := AgentConfig{
		DockerImage:    "custom/zap:latest",
		MaxMemoryMB:    2048,
		MaxCPUCores:    2.0,
		DefaultTimeout: 15 * time.Minute,
		ScanType:       "full",
		MaxDepth:       10,
	}
	
	zapAgent := NewAgentWithConfig(customConfig)
	
	assert.NotNil(t, zapAgent)
	assert.Equal(t, customConfig.DockerImage, zapAgent.config.DockerImage)
	assert.Equal(t, customConfig.MaxMemoryMB, zapAgent.config.MaxMemoryMB)
	assert.Equal(t, customConfig.MaxCPUCores, zapAgent.config.MaxCPUCores)
	assert.Equal(t, customConfig.DefaultTimeout, zapAgent.config.DefaultTimeout)
	assert.Equal(t, customConfig.ScanType, zapAgent.config.ScanType)
}

func TestGetConfig(t *testing.T) {
	zapAgent := NewAgent()
	config := zapAgent.GetConfig()
	
	assert.Equal(t, AgentName, config.Name)
	assert.Equal(t, AgentVersion, config.Version)
	assert.True(t, config.RequiresDocker)
	assert.Contains(t, config.SupportedLangs, "javascript")
	assert.Contains(t, config.SupportedLangs, "python")
	assert.Contains(t, config.Categories, agent.CategoryXSS)
	assert.Contains(t, config.Categories, agent.CategorySQLInjection)
}

func TestGetVersion(t *testing.T) {
	zapAgent := NewAgent()
	version := zapAgent.GetVersion()
	
	assert.Equal(t, AgentVersion, version.AgentVersion)
	assert.NotEmpty(t, version.BuildDate)
}

func TestMapZAPSeverity(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		riskCode string
		expected agent.Severity
	}{
		{"3", agent.SeverityHigh},
		{"2", agent.SeverityMedium},
		{"1", agent.SeverityLow},
		{"0", agent.SeverityInfo},
		{"unknown", agent.SeverityMedium},
	}
	
	for _, tt := range tests {
		t.Run(tt.riskCode, func(t *testing.T) {
			result := zapAgent.mapZAPSeverity(tt.riskCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapZAPCategory(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		cweID     string
		alertName string
		expected  agent.VulnCategory
	}{
		{"79", "Cross-site Scripting", agent.CategoryXSS},
		{"89", "SQL Injection", agent.CategorySQLInjection},
		{"352", "Cross-Site Request Forgery", agent.CategoryCSRF},
		{"22", "Path Traversal", agent.CategoryPathTraversal},
		{"77", "Command Injection", agent.CategoryCommandInjection},
		{"", "XSS Vulnerability", agent.CategoryXSS},
		{"", "SQL Injection Attack", agent.CategorySQLInjection},
		{"", "Unknown Vulnerability", agent.CategoryOther},
	}
	
	for _, tt := range tests {
		t.Run(tt.alertName, func(t *testing.T) {
			result := zapAgent.mapZAPCategory(tt.cweID, tt.alertName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapZAPConfidence(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		confidence string
		expected   float64
	}{
		{"3", 0.9},
		{"high", 0.9},
		{"2", 0.7},
		{"medium", 0.7},
		{"1", 0.5},
		{"low", 0.5},
		{"0", 0.1},
		{"false positive", 0.1},
		{"unknown", 0.7},
	}
	
	for _, tt := range tests {
		t.Run(tt.confidence, func(t *testing.T) {
			result := zapAgent.mapZAPConfidence(tt.confidence)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseZAPReferences(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		name      string
		reference string
		expected  []string
	}{
		{
			name:      "empty reference",
			reference: "",
			expected:  nil,
		},
		{
			name:      "single URL",
			reference: "https://owasp.org/www-community/attacks/xss/",
			expected:  []string{"https://owasp.org/www-community/attacks/xss/"},
		},
		{
			name:      "multiple references with HTML",
			reference: "https://cwe.mitre.org/data/definitions/79.html<br>https://owasp.org/www-community/attacks/xss/",
			expected:  []string{"https://cwe.mitre.org/data/definitions/79.html", "https://owasp.org/www-community/attacks/xss/"},
		},
		{
			name:      "CWE reference",
			reference: "CWE-79: Cross-site Scripting",
			expected:  []string{"CWE-79: Cross-site Scripting"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := zapAgent.parseZAPReferences(tt.reference)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateZAPFindingID(t *testing.T) {
	tests := []struct {
		pluginID string
		uri      string
		method   string
		expected string
	}{
		{
			pluginID: "40012",
			uri:      "/api/users",
			method:   "GET",
			expected: "zap-40012-GET--api-users",
		},
		{
			pluginID: "40014",
			uri:      "/login",
			method:   "POST",
			expected: "zap-40014-POST--login",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := generateZAPFindingID(tt.pluginID, tt.uri, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test that requires Docker (skip if Docker not available)
func TestScanIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	zapAgent := NewAgent()
	
	// Test health check first
	ctx := context.Background()
	err := zapAgent.HealthCheck(ctx)
	if err != nil {
		t.Skipf("Docker or ZAP not available: %v", err)
	}
	
	// Test scan with a non-web repository (should skip DAST)
	config := agent.ScanConfig{
		RepoURL:   "https://github.com/octocat/Hello-World.git",
		Branch:    "master",
		Languages: []string{"markdown"},
		Timeout:   2 * time.Minute,
	}
	
	result, err := zapAgent.Scan(ctx, config)
	require.NoError(t, err)
	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	assert.Equal(t, AgentName, result.AgentID)
	assert.Equal(t, "dast", result.Metadata.ScanType)
	// Should have no findings since it's not a web app
	assert.Empty(t, result.Findings)
}