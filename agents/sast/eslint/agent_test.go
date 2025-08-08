package eslint

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
	assert.NotEmpty(t, a.config.SecurityRules)
}

func TestNewAgentWithConfig(t *testing.T) {
	config := AgentConfig{
		DockerImage:    "custom:latest",
		MaxMemoryMB:    1024,
		MaxCPUCores:    2.0,
		DefaultTimeout: 10 * time.Minute,
		SecurityRules:  []string{"security/detect-eval-with-expression"},
	}
	
	a := NewAgentWithConfig(config)
	
	assert.NotNil(t, a)
	assert.Equal(t, config.DockerImage, a.config.DockerImage)
	assert.Equal(t, config.MaxMemoryMB, a.config.MaxMemoryMB)
	assert.Equal(t, config.MaxCPUCores, a.config.MaxCPUCores)
	assert.Equal(t, config.DefaultTimeout, a.config.DefaultTimeout)
	assert.Equal(t, config.SecurityRules, a.config.SecurityRules)
}

func TestAgent_GetConfig(t *testing.T) {
	a := NewAgent()
	config := a.GetConfig()
	
	assert.Equal(t, AgentName, config.Name)
	assert.Equal(t, AgentVersion, config.Version)
	assert.Contains(t, config.SupportedLangs, "javascript")
	assert.Contains(t, config.SupportedLangs, "typescript")
	assert.Contains(t, config.Categories, agent.CategoryXSS)
	assert.Contains(t, config.Categories, agent.CategoryCommandInjection)
	assert.True(t, config.RequiresDocker)
}

func TestAgent_GetVersion(t *testing.T) {
	a := NewAgent()
	version := a.GetVersion()
	
	assert.Equal(t, AgentVersion, version.AgentVersion)
	assert.NotEmpty(t, version.BuildDate)
}

func TestAgent_isJavaScriptProject(t *testing.T) {
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
			name:      "mixed languages with javascript",
			languages: []string{"python", "javascript", "go"},
			expected:  true,
		},
		{
			name:      "no javascript languages",
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
			result := a.isJavaScriptProject(tt.languages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_Scan_NonJavaScriptProject(t *testing.T) {
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL:   "https://github.com/test/python-repo",
		Branch:    "main",
		Languages: []string{"python", "go"},
	}
	
	result, err := a.Scan(context.Background(), config)
	
	require.NoError(t, err)
	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	assert.Empty(t, result.Findings)
	assert.Equal(t, AgentName, result.AgentID)
}

func TestAgent_generateESLintConfig(t *testing.T) {
	a := NewAgent()
	config := a.generateESLintConfig()
	
	assert.NotEmpty(t, config)
	assert.Contains(t, config, "security")
	assert.Contains(t, config, "no-eval")
	assert.Contains(t, config, "security/detect-eval-with-expression")
}

func TestAgent_mapSeverity(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name           string
		eslintSeverity int
		ruleID         string
		expected       agent.Severity
	}{
		{
			name:           "high severity rule",
			eslintSeverity: 2,
			ruleID:         "security/detect-eval-with-expression",
			expected:       agent.SeverityHigh,
		},
		{
			name:           "error severity",
			eslintSeverity: 2,
			ruleID:         "security/detect-buffer-noassert",
			expected:       agent.SeverityMedium,
		},
		{
			name:           "warning severity",
			eslintSeverity: 1,
			ruleID:         "security/detect-possible-timing-attacks",
			expected:       agent.SeverityLow,
		},
		{
			name:           "no-eval rule",
			eslintSeverity: 2,
			ruleID:         "no-eval",
			expected:       agent.SeverityHigh,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.mapSeverity(tt.eslintSeverity, tt.ruleID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_mapCategory(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		ruleID   string
		expected agent.VulnCategory
	}{
		{
			name:     "eval detection",
			ruleID:   "security/detect-eval-with-expression",
			expected: agent.CategoryCommandInjection,
		},
		{
			name:     "XSS detection",
			ruleID:   "security/detect-disable-mustache-escape",
			expected: agent.CategoryXSS,
		},
		{
			name:     "CSRF detection",
			ruleID:   "security/detect-no-csrf-before-method-override",
			expected: agent.CategoryCSRF,
		},
		{
			name:     "path traversal",
			ruleID:   "security/detect-non-literal-fs-filename",
			expected: agent.CategoryPathTraversal,
		},
		{
			name:     "crypto issue",
			ruleID:   "security/detect-pseudoRandomBytes",
			expected: agent.CategoryInsecureCrypto,
		},
		{
			name:     "core eslint rule",
			ruleID:   "no-eval",
			expected: agent.CategoryCommandInjection,
		},
		{
			name:     "unknown rule",
			ruleID:   "unknown-rule",
			expected: agent.CategoryOther,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.mapCategory(tt.ruleID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_isSecurityRule(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		ruleID   string
		expected bool
	}{
		{
			name:     "security plugin rule",
			ruleID:   "security/detect-eval-with-expression",
			expected: true,
		},
		{
			name:     "core security rule",
			ruleID:   "no-eval",
			expected: true,
		},
		{
			name:     "non-security rule",
			ruleID:   "no-unused-vars",
			expected: false,
		},
		{
			name:     "empty rule",
			ruleID:   "",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.isSecurityRule(tt.ruleID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_getRuleTitle(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		ruleID   string
		expected string
	}{
		{
			name:     "known security rule",
			ruleID:   "security/detect-eval-with-expression",
			expected: "Dangerous eval() usage detected",
		},
		{
			name:     "core eslint rule",
			ruleID:   "no-eval",
			expected: "Use of eval() is prohibited",
		},
		{
			name:     "unknown rule",
			ruleID:   "unknown-rule",
			expected: "Security issue: unknown-rule",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.getRuleTitle(tt.ruleID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_getConfidence(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		ruleID   string
		expected float64
	}{
		{
			name:     "high confidence rule",
			ruleID:   "no-eval",
			expected: 0.9,
		},
		{
			name:     "security plugin rule",
			ruleID:   "security/detect-buffer-noassert",
			expected: 0.7,
		},
		{
			name:     "other rule",
			ruleID:   "some-other-rule",
			expected: 0.6,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.getConfidence(tt.ruleID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_getRuleReferences(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		ruleID   string
		expected int // number of references
	}{
		{
			name:     "security plugin rule",
			ruleID:   "security/detect-eval-with-expression",
			expected: 1,
		},
		{
			name:     "core eslint rule",
			ruleID:   "no-eval",
			expected: 1,
		},
		{
			name:     "unknown rule",
			ruleID:   "unknown-rule",
			expected: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.getRuleReferences(tt.ruleID)
			assert.Len(t, result, tt.expected)
			if tt.expected > 0 {
				assert.NotEmpty(t, result[0])
			}
		})
	}
}

func TestAgent_extractCodeSnippet(t *testing.T) {
	a := NewAgent()
	
	source := `function test() {
    eval("dangerous code");
    console.log("safe code");
}`
	
	tests := []struct {
		name     string
		source   string
		line     int
		expected string
	}{
		{
			name:     "valid line",
			source:   source,
			line:     2,
			expected: `eval("dangerous code");`,
		},
		{
			name:     "empty source",
			source:   "",
			line:     1,
			expected: "",
		},
		{
			name:     "invalid line",
			source:   source,
			line:     10,
			expected: "",
		},
		{
			name:     "line zero",
			source:   source,
			line:     0,
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.extractCodeSnippet(tt.source, tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}