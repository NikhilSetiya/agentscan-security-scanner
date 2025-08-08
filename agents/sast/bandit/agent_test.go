package bandit

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
	assert.Equal(t, 10*time.Minute, a.config.DefaultTimeout)
	assert.Equal(t, "low", a.config.Severity)
	assert.Equal(t, "low", a.config.Confidence)
	assert.NotEmpty(t, a.config.ExcludePaths)
	assert.NotEmpty(t, a.config.PythonVersions)
}

func TestNewAgentWithConfig(t *testing.T) {
	config := AgentConfig{
		DockerImage:    "custom:latest",
		MaxMemoryMB:    1024,
		MaxCPUCores:    2.0,
		DefaultTimeout: 15 * time.Minute,
		Severity:       "high",
		Confidence:     "medium",
		SkipTests:      []string{"B101", "B102"},
	}
	
	a := NewAgentWithConfig(config)
	
	assert.NotNil(t, a)
	assert.Equal(t, config.DockerImage, a.config.DockerImage)
	assert.Equal(t, config.MaxMemoryMB, a.config.MaxMemoryMB)
	assert.Equal(t, config.MaxCPUCores, a.config.MaxCPUCores)
	assert.Equal(t, config.DefaultTimeout, a.config.DefaultTimeout)
	assert.Equal(t, config.Severity, a.config.Severity)
	assert.Equal(t, config.Confidence, a.config.Confidence)
	assert.Equal(t, config.SkipTests, a.config.SkipTests)
}

func TestAgent_GetConfig(t *testing.T) {
	a := NewAgent()
	config := a.GetConfig()
	
	assert.Equal(t, AgentName, config.Name)
	assert.Equal(t, AgentVersion, config.Version)
	assert.Contains(t, config.SupportedLangs, "python")
	assert.Contains(t, config.SupportedLangs, "py")
	assert.Contains(t, config.Categories, agent.CategorySQLInjection)
	assert.Contains(t, config.Categories, agent.CategoryCommandInjection)
	assert.Contains(t, config.Categories, agent.CategoryInsecureCrypto)
	assert.True(t, config.RequiresDocker)
}

func TestAgent_GetVersion(t *testing.T) {
	a := NewAgent()
	version := a.GetVersion()
	
	assert.Equal(t, AgentVersion, version.AgentVersion)
	assert.NotEmpty(t, version.BuildDate)
}

func TestAgent_isPythonProject(t *testing.T) {
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
			name:      "python language",
			languages: []string{"python"},
			expected:  true,
		},
		{
			name:      "py language",
			languages: []string{"py"},
			expected:  true,
		},
		{
			name:      "python3 language",
			languages: []string{"python3"},
			expected:  true,
		},
		{
			name:      "mixed languages with python",
			languages: []string{"javascript", "python", "go"},
			expected:  true,
		},
		{
			name:      "no python languages",
			languages: []string{"javascript", "go", "java"},
			expected:  false,
		},
		{
			name:      "case insensitive",
			languages: []string{"Python", "PY"},
			expected:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.isPythonProject(tt.languages)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_Scan_NonPythonProject(t *testing.T) {
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL:   "https://github.com/test/java-repo",
		Branch:    "main",
		Languages: []string{"java", "javascript"},
	}
	
	result, err := a.Scan(context.Background(), config)
	
	require.NoError(t, err)
	assert.Equal(t, agent.ScanStatusCompleted, result.Status)
	assert.Empty(t, result.Findings)
	assert.Equal(t, AgentName, result.AgentID)
}

func TestAgent_mapSeverity(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name            string
		banditSeverity  string
		expected        agent.Severity
	}{
		{
			name:           "high severity",
			banditSeverity: "HIGH",
			expected:       agent.SeverityHigh,
		},
		{
			name:           "medium severity",
			banditSeverity: "MEDIUM",
			expected:       agent.SeverityMedium,
		},
		{
			name:           "low severity",
			banditSeverity: "LOW",
			expected:       agent.SeverityLow,
		},
		{
			name:           "unknown severity defaults to medium",
			banditSeverity: "UNKNOWN",
			expected:       agent.SeverityMedium,
		},
		{
			name:           "case insensitive",
			banditSeverity: "high",
			expected:       agent.SeverityHigh,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.mapSeverity(tt.banditSeverity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_mapCategory(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		testID   string
		testName string
		expected agent.VulnCategory
	}{
		{
			name:     "hardcoded password",
			testID:   "B105",
			testName: "hardcoded_password_string",
			expected: agent.CategoryHardcodedSecrets,
		},
		{
			name:     "SQL injection",
			testID:   "B608",
			testName: "hardcoded_sql_expressions",
			expected: agent.CategorySQLInjection,
		},
		{
			name:     "command injection",
			testID:   "B602",
			testName: "subprocess_popen_with_shell_equals_true",
			expected: agent.CategoryCommandInjection,
		},
		{
			name:     "insecure crypto",
			testID:   "B303",
			testName: "md5_insecure_hash",
			expected: agent.CategoryInsecureCrypto,
		},
		{
			name:     "XSS vulnerability",
			testID:   "B701",
			testName: "jinja2_autoescape_false",
			expected: agent.CategoryXSS,
		},
		{
			name:     "insecure deserialization",
			testID:   "B301",
			testName: "pickle_usage",
			expected: agent.CategoryInsecureDeserialization,
		},
		{
			name:     "path traversal",
			testID:   "B108",
			testName: "hardcoded_tmp_directory",
			expected: agent.CategoryPathTraversal,
		},
		{
			name:     "misconfiguration",
			testID:   "B201",
			testName: "flask_debug_true",
			expected: agent.CategoryMisconfiguration,
		},
		{
			name:     "unknown rule",
			testID:   "B999",
			testName: "unknown_test",
			expected: agent.CategoryOther,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.mapCategory(tt.testID, tt.testName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_mapConfidence(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name              string
		banditConfidence  string
		expected          float64
	}{
		{
			name:             "high confidence",
			banditConfidence: "HIGH",
			expected:         0.9,
		},
		{
			name:             "medium confidence",
			banditConfidence: "MEDIUM",
			expected:         0.7,
		},
		{
			name:             "low confidence",
			banditConfidence: "LOW",
			expected:         0.5,
		},
		{
			name:             "unknown confidence defaults to medium",
			banditConfidence: "UNKNOWN",
			expected:         0.7,
		},
		{
			name:             "case insensitive",
			banditConfidence: "high",
			expected:         0.9,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.mapConfidence(tt.banditConfidence)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_getRuleTitle(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		testID   string
		testName string
		expected string
	}{
		{
			name:     "with test name",
			testID:   "B105",
			testName: "hardcoded_password_string",
			expected: "Hardcoded Password String",
		},
		{
			name:     "without test name",
			testID:   "B105",
			testName: "",
			expected: "Security issue: B105",
		},
		{
			name:     "complex test name",
			testID:   "B602",
			testName: "subprocess_popen_with_shell_equals_true",
			expected: "Subprocess Popen With Shell Equals True",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.getRuleTitle(tt.testID, tt.testName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgent_getRuleReferences(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		testID   string
		moreInfo string
		expected int // number of references
	}{
		{
			name:     "with more info",
			testID:   "B105",
			moreInfo: "https://example.com/security-info",
			expected: 2, // moreInfo + bandit docs
		},
		{
			name:     "without more info",
			testID:   "B105",
			moreInfo: "",
			expected: 1, // just bandit docs
		},
		{
			name:     "empty test ID",
			testID:   "",
			moreInfo: "https://example.com/info",
			expected: 1, // just moreInfo
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.getRuleReferences(tt.testID, tt.moreInfo)
			assert.Len(t, result, tt.expected)
			if tt.expected > 0 {
				for _, ref := range result {
					assert.NotEmpty(t, ref)
				}
			}
		})
	}
}

func TestGenerateFindingID(t *testing.T) {
	tests := []struct {
		name     string
		testID   string
		file     string
		line     int
		expected string
	}{
		{
			name:     "basic finding ID",
			testID:   "B105",
			file:     "/src/app/main.py",
			line:     42,
			expected: "bandit-B105-main.py-42",
		},
		{
			name:     "nested file path",
			testID:   "B602",
			file:     "/src/utils/security.py",
			line:     15,
			expected: "bandit-B602-security.py-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFindingID(tt.testID, tt.file, tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}