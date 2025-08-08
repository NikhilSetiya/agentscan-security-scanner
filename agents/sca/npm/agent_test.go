package npm

import (
	"context"
	"testing"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgent(t *testing.T) {
	a := NewAgent()
	
	assert.NotNil(t, a)
	assert.Equal(t, DefaultImage, a.config.DockerImage)
	assert.Equal(t, 512, a.config.MaxMemoryMB)
	assert.Equal(t, 1.0, a.config.MaxCPUCores)
	assert.Equal(t, 5*time.Minute, a.config.DefaultTimeout)
	assert.Equal(t, "low", a.config.AuditLevel)
	assert.False(t, a.config.ProductionOnly)
	assert.True(t, a.config.IncludeDevDeps)
	assert.Empty(t, a.config.RegistryURL)
	assert.Empty(t, a.config.ExcludePackages)
}

func TestNewAgentWithConfig(t *testing.T) {
	config := AgentConfig{
		DockerImage:     "node:16-alpine",
		MaxMemoryMB:     1024,
		MaxCPUCores:     2.0,
		DefaultTimeout:  10 * time.Minute,
		AuditLevel:      "high",
		ProductionOnly:  true,
		IncludeDevDeps:  false,
		RegistryURL:     "https://registry.npmjs.org/",
		ExcludePackages: []string{"lodash", "moment"},
	}
	
	a := NewAgentWithConfig(config)
	
	assert.NotNil(t, a)
	assert.Equal(t, config.DockerImage, a.config.DockerImage)
	assert.Equal(t, config.MaxMemoryMB, a.config.MaxMemoryMB)
	assert.Equal(t, config.MaxCPUCores, a.config.MaxCPUCores)
	assert.Equal(t, config.DefaultTimeout, a.config.DefaultTimeout)
	assert.Equal(t, config.AuditLevel, a.config.AuditLevel)
	assert.Equal(t, config.ProductionOnly, a.config.ProductionOnly)
	assert.Equal(t, config.IncludeDevDeps, a.config.IncludeDevDeps)
	assert.Equal(t, config.RegistryURL, a.config.RegistryURL)
	assert.Equal(t, config.ExcludePackages, a.config.ExcludePackages)
}

func TestAgent_GetConfig(t *testing.T) {
	a := NewAgent()
	
	config := a.GetConfig()
	
	assert.Equal(t, AgentName, config.Name)
	assert.Equal(t, AgentVersion, config.Version)
	assert.Contains(t, config.SupportedLangs, "javascript")
	assert.Contains(t, config.SupportedLangs, "typescript")
	assert.Contains(t, config.SupportedLangs, "node")
	assert.Contains(t, config.SupportedLangs, "nodejs")
	assert.True(t, config.RequiresDocker)
	assert.Equal(t, a.config.DefaultTimeout, config.DefaultTimeout)
	assert.Equal(t, a.config.MaxMemoryMB, config.MaxMemoryMB)
	assert.Equal(t, a.config.MaxCPUCores, config.MaxCPUCores)
	
	// Check vulnerability categories
	expectedCategories := []agent.VulnCategory{
		agent.CategoryDependencyVuln,
		agent.CategoryOutdatedDeps,
		agent.CategoryLicenseIssue,
		agent.CategorySupplyChain,
	}
	
	for _, category := range expectedCategories {
		assert.Contains(t, config.Categories, category)
	}
}

func TestAgent_GetVersion(t *testing.T) {
	a := NewAgent()
	
	version := a.GetVersion()
	
	assert.Equal(t, AgentVersion, version.AgentVersion)
	assert.NotEmpty(t, version.BuildDate)
	assert.Equal(t, "unknown", version.GitCommit)
	// ToolVersion might be "unknown" if Docker is not available
}

func TestAgent_isNodeProject(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name      string
		languages []string
		expected  bool
	}{
		{
			name:      "empty languages should return true",
			languages: []string{},
			expected:  true,
		},
		{
			name:      "javascript language",
			languages: []string{"javascript"},
			expected:  true,
		},
		{
			name:      "typescript language",
			languages: []string{"typescript"},
			expected:  true,
		},
		{
			name:      "node language",
			languages: []string{"node"},
			expected:  true,
		},
		{
			name:      "nodejs language",
			languages: []string{"nodejs"},
			expected:  true,
		},
		{
			name:      "js language",
			languages: []string{"js"},
			expected:  true,
		},
		{
			name:      "ts language",
			languages: []string{"ts"},
			expected:  true,
		},
		{
			name:      "mixed languages with javascript",
			languages: []string{"python", "javascript", "go"},
			expected:  true,
		},
		{
			name:      "no node languages",
			languages: []string{"python", "go", "java"},
			expected:  false,
		},
		{
			name:      "case insensitive",
			languages: []string{"JavaScript", "TypeScript"},
			expected:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.isNodeProject(tt.languages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_Scan_NonNodeProject(t *testing.T) {
	a := NewAgent()
	ctx := context.Background()
	
	config := agent.ScanConfig{
		RepoURL:   "https://github.com/test/python-repo",
		Branch:    "main",
		Languages: []string{"python"},
	}
	
	result, err := a.Scan(ctx, config)
	
	require.NoError(t, err)
	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	assert.Empty(t, result.Findings)
	assert.Equal(t, "sca", result.Metadata.ScanType)
}

func TestAgent_mapSeverity(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name         string
		npmSeverity  string
		expected     agent.Severity
	}{
		{
			name:        "critical severity",
			npmSeverity: "critical",
			expected:    agent.SeverityHigh,
		},
		{
			name:        "high severity",
			npmSeverity: "high",
			expected:    agent.SeverityHigh,
		},
		{
			name:        "moderate severity",
			npmSeverity: "moderate",
			expected:    agent.SeverityMedium,
		},
		{
			name:        "low severity",
			npmSeverity: "low",
			expected:    agent.SeverityLow,
		},
		{
			name:        "info severity",
			npmSeverity: "info",
			expected:    agent.SeverityLow,
		},
		{
			name:        "unknown severity defaults to medium",
			npmSeverity: "unknown",
			expected:    agent.SeverityMedium,
		},
		{
			name:        "case insensitive",
			npmSeverity: "CRITICAL",
			expected:    agent.SeverityHigh,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.mapSeverity(tt.npmSeverity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_mapCategory(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		cwes     []string
		expected agent.VulnCategory
	}{
		{
			name:     "XSS vulnerability",
			cwes:     []string{"CWE-79"},
			expected: agent.CategoryXSS,
		},
		{
			name:     "SQL injection",
			cwes:     []string{"CWE-89"},
			expected: agent.CategorySQLInjection,
		},
		{
			name:     "command injection",
			cwes:     []string{"CWE-78"},
			expected: agent.CategoryCommandInjection,
		},
		{
			name:     "path traversal",
			cwes:     []string{"CWE-22"},
			expected: agent.CategoryPathTraversal,
		},
		{
			name:     "insecure crypto",
			cwes:     []string{"CWE-327"},
			expected: agent.CategoryInsecureCrypto,
		},
		{
			name:     "insecure deserialization",
			cwes:     []string{"CWE-502"},
			expected: agent.CategoryInsecureDeserialization,
		},
		{
			name:     "supply chain",
			cwes:     []string{"CWE-1104"},
			expected: agent.CategorySupplyChain,
		},
		{
			name:     "multiple CWEs - first match wins",
			cwes:     []string{"CWE-79", "CWE-89"},
			expected: agent.CategoryXSS,
		},
		{
			name:     "unknown CWE",
			cwes:     []string{"CWE-999"},
			expected: agent.CategoryDependencyVuln,
		},
		{
			name:     "empty CWEs",
			cwes:     []string{},
			expected: agent.CategoryDependencyVuln,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.mapCategory(tt.cwes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_calculateConfidence(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name      string
		severity  string
		cvssScore float64
		expected  float64
	}{
		{
			name:      "critical severity",
			severity:  "critical",
			cvssScore: 0,
			expected:  0.95,
		},
		{
			name:      "high severity",
			severity:  "high",
			cvssScore: 0,
			expected:  0.9,
		},
		{
			name:      "moderate severity",
			severity:  "moderate",
			cvssScore: 0,
			expected:  0.8,
		},
		{
			name:      "low severity",
			severity:  "low",
			cvssScore: 0,
			expected:  0.6,
		},
		{
			name:      "info severity",
			severity:  "info",
			cvssScore: 0,
			expected:  0.5,
		},
		{
			name:      "with CVSS score",
			severity:  "high",
			cvssScore: 8.0,
			expected:  0.85, // (0.9 + 0.8) / 2
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.calculateConfidence(tt.severity, tt.cvssScore)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestAgent_isPackageExcluded(t *testing.T) {
	config := AgentConfig{
		ExcludePackages: []string{"lodash", "moment", "debug"},
	}
	a := NewAgentWithConfig(config)
	
	tests := []struct {
		name        string
		packageName string
		expected    bool
	}{
		{
			name:        "excluded package",
			packageName: "lodash",
			expected:    true,
		},
		{
			name:        "not excluded package",
			packageName: "express",
			expected:    false,
		},
		{
			name:        "case sensitive",
			packageName: "Lodash",
			expected:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.isPackageExcluded(tt.packageName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateFindingID(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		title       string
		expected    string
	}{
		{
			name:        "basic finding ID",
			packageName: "lodash",
			title:       "Prototype Pollution",
			expected:    "npm-audit-lodash-19",
		},
		{
			name:        "different package",
			packageName: "express",
			title:       "XSS vulnerability",
			expected:    "npm-audit-express-17",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFindingID(tt.packageName, tt.title)
			assert.Equal(t, tt.expected, result)
		})
	}
}