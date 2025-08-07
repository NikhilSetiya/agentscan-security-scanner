package semgrep

import (
	"context"
	"testing"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/stretchr/testify/assert"
)

func TestNewAgent(t *testing.T) {
	a := NewAgent()
	
	assert.NotNil(t, a)
	assert.Equal(t, DefaultImage, a.config.DockerImage)
	assert.Equal(t, 512, a.config.MaxMemoryMB)
	assert.Equal(t, 1.0, a.config.MaxCPUCores)
	assert.Equal(t, 10*time.Minute, a.config.DefaultTimeout)
	assert.Equal(t, "auto", a.config.RulesConfig)
}

func TestNewAgentWithConfig(t *testing.T) {
	config := AgentConfig{
		DockerImage:    "custom/semgrep:v1.0.0",
		MaxMemoryMB:    1024,
		MaxCPUCores:    2.0,
		DefaultTimeout: 5 * time.Minute,
		RulesConfig:    "p/security-audit",
	}
	
	a := NewAgentWithConfig(config)
	
	assert.NotNil(t, a)
	assert.Equal(t, config.DockerImage, a.config.DockerImage)
	assert.Equal(t, config.MaxMemoryMB, a.config.MaxMemoryMB)
	assert.Equal(t, config.MaxCPUCores, a.config.MaxCPUCores)
	assert.Equal(t, config.DefaultTimeout, a.config.DefaultTimeout)
	assert.Equal(t, config.RulesConfig, a.config.RulesConfig)
}

func TestAgent_GetConfig(t *testing.T) {
	a := NewAgent()
	config := a.GetConfig()
	
	assert.Equal(t, AgentName, config.Name)
	assert.Equal(t, AgentVersion, config.Version)
	assert.True(t, config.RequiresDocker)
	assert.Contains(t, config.SupportedLangs, "javascript")
	assert.Contains(t, config.SupportedLangs, "python")
	assert.Contains(t, config.SupportedLangs, "go")
	assert.Contains(t, config.Categories, agent.CategorySQLInjection)
	assert.Contains(t, config.Categories, agent.CategoryXSS)
}

func TestAgent_GetVersion(t *testing.T) {
	a := NewAgent()
	version := a.GetVersion()
	
	assert.Equal(t, AgentVersion, version.AgentVersion)
	assert.NotEmpty(t, version.BuildDate)
	assert.Equal(t, "unknown", version.GitCommit)
}

func TestAgent_Scan_InvalidConfig(t *testing.T) {
	a := NewAgent()
	ctx := context.Background()
	
	// Test with empty repo URL
	config := agent.ScanConfig{
		RepoURL: "",
		Branch:  "main",
		Timeout: 1 * time.Minute,
	}
	
	result, err := a.Scan(ctx, config)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, agent.ScanStatusFailed, result.Status)
	assert.NotEmpty(t, result.Error)
}

func TestAgent_Scan_Timeout(t *testing.T) {
	a := NewAgent()
	ctx := context.Background()
	
	// Use a very short timeout to trigger timeout
	config := agent.ScanConfig{
		RepoURL: "https://github.com/example/repo.git",
		Branch:  "main",
		Timeout: 1 * time.Nanosecond, // Extremely short timeout
	}
	
	result, err := a.Scan(ctx, config)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, agent.ScanStatusFailed, result.Status)
	assert.Contains(t, result.Error, "context deadline exceeded")
}

func TestAgent_Scan_CancelledContext(t *testing.T) {
	a := NewAgent()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	config := agent.ScanConfig{
		RepoURL: "https://github.com/example/repo.git",
		Branch:  "main",
		Timeout: 1 * time.Minute,
	}
	
	result, err := a.Scan(ctx, config)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, agent.ScanStatusFailed, result.Status)
	assert.Contains(t, result.Error, "context canceled")
}

func TestMapSeverity(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		level    string
		severity string
		expected agent.Severity
	}{
		{"error", "high", agent.SeverityHigh},
		{"warning", "medium", agent.SeverityMedium},
		{"info", "low", agent.SeverityLow},
		{"error", "", agent.SeverityHigh},
		{"warning", "", agent.SeverityMedium},
		{"note", "", agent.SeverityLow},
		{"unknown", "unknown", agent.SeverityMedium},
	}
	
	for _, tt := range tests {
		t.Run(tt.level+"_"+tt.severity, func(t *testing.T) {
			result := a.mapSeverity(tt.level, tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapCategory(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		category string
		expected agent.VulnCategory
	}{
		{"sql-injection", agent.CategorySQLInjection},
		{"xss", agent.CategoryXSS},
		{"command-injection", agent.CategoryCommandInjection},
		{"path-traversal", agent.CategoryPathTraversal},
		{"crypto", agent.CategoryInsecureCrypto},
		{"secrets", agent.CategoryHardcodedSecrets},
		{"deserialization", agent.CategoryInsecureDeserialization},
		{"auth", agent.CategoryAuthBypass},
		{"csrf", agent.CategoryCSRF},
		{"misconfiguration", agent.CategoryMisconfiguration},
		{"unknown", agent.CategoryOther},
	}
	
	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			result := a.mapCategory(tt.category)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapConfidence(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		confidence string
		expected   float64
	}{
		{"high", 0.9},
		{"medium", 0.7},
		{"low", 0.5},
		{"unknown", 0.7},
		{"", 0.7},
	}
	
	for _, tt := range tests {
		t.Run(tt.confidence, func(t *testing.T) {
			result := a.mapConfidence(tt.confidence)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateFindingID(t *testing.T) {
	tests := []struct {
		ruleID   string
		file     string
		line     int
		expected string
	}{
		{"javascript.lang.security.audit.xss.react-dangerously-set-inner-html", "/src/app.js", 42, "semgrep-javascript.lang.security.audit.xss.react-dangerously-set-inner-html-app.js-42"},
		{"python.lang.security.audit.sqli.python-sqli", "/src/models/user.py", 15, "semgrep-python.lang.security.audit.sqli.python-sqli-user.py-15"},
	}
	
	for _, tt := range tests {
		t.Run(tt.ruleID, func(t *testing.T) {
			result := generateFindingID(tt.ruleID, tt.file, tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}