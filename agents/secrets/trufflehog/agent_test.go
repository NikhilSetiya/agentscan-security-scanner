package trufflehog

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
	assert.Equal(t, 1024, a.config.MaxMemoryMB)
	assert.Equal(t, 2.0, a.config.MaxCPUCores)
	assert.Equal(t, 15*time.Minute, a.config.DefaultTimeout)
	assert.True(t, a.config.ScanGitHistory)
	assert.Equal(t, 100, a.config.MaxDepth)
}

func TestNewAgentWithConfig(t *testing.T) {
	config := AgentConfig{
		DockerImage:    "custom/trufflehog:test",
		MaxMemoryMB:    512,
		MaxCPUCores:    1.0,
		DefaultTimeout: 5 * time.Minute,
		ScanGitHistory: false,
		MaxDepth:       50,
		Whitelist:      []string{"test.*", "example.*"},
	}
	
	a := NewAgentWithConfig(config)
	
	assert.NotNil(t, a)
	assert.Equal(t, config.DockerImage, a.config.DockerImage)
	assert.Equal(t, config.MaxMemoryMB, a.config.MaxMemoryMB)
	assert.Equal(t, config.MaxCPUCores, a.config.MaxCPUCores)
	assert.Equal(t, config.DefaultTimeout, a.config.DefaultTimeout)
	assert.Equal(t, config.ScanGitHistory, a.config.ScanGitHistory)
	assert.Equal(t, config.MaxDepth, a.config.MaxDepth)
	assert.Equal(t, config.Whitelist, a.config.Whitelist)
}

func TestGetConfig(t *testing.T) {
	a := NewAgent()
	config := a.GetConfig()
	
	assert.Equal(t, AgentName, config.Name)
	assert.Equal(t, AgentVersion, config.Version)
	assert.Contains(t, config.SupportedLangs, "*")
	assert.Contains(t, config.Categories, agent.CategoryHardcodedSecrets)
	assert.True(t, config.RequiresDocker)
	assert.Equal(t, a.config.DefaultTimeout, config.DefaultTimeout)
	assert.Equal(t, a.config.MaxMemoryMB, config.MaxMemoryMB)
	assert.Equal(t, a.config.MaxCPUCores, config.MaxCPUCores)
}

func TestGetVersion(t *testing.T) {
	a := NewAgent()
	version := a.GetVersion()
	
	assert.Equal(t, AgentVersion, version.AgentVersion)
	assert.NotEmpty(t, version.BuildDate)
	assert.Equal(t, "unknown", version.GitCommit)
}

func TestDetermineSeverity(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		result   TruffleHogResult
		expected agent.Severity
	}{
		{
			name: "verified secret",
			result: TruffleHogResult{
				DetectorName: "github",
				Verified:     true,
			},
			expected: agent.SeverityHigh,
		},
		{
			name: "aws unverified",
			result: TruffleHogResult{
				DetectorName: "aws",
				Verified:     false,
			},
			expected: agent.SeverityHigh,
		},
		{
			name: "github unverified",
			result: TruffleHogResult{
				DetectorName: "github",
				Verified:     false,
			},
			expected: agent.SeverityHigh,
		},
		{
			name: "private key",
			result: TruffleHogResult{
				DetectorName: "privatekey",
				Verified:     false,
			},
			expected: agent.SeverityHigh,
		},
		{
			name: "unknown detector",
			result: TruffleHogResult{
				DetectorName: "unknown",
				Verified:     false,
			},
			expected: agent.SeverityHigh, // Default to high as per requirements
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := a.determineSeverity(tt.result)
			assert.Equal(t, tt.expected, severity)
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		result   TruffleHogResult
		expected float64
	}{
		{
			name: "verified aws secret",
			result: TruffleHogResult{
				DetectorName: "aws",
				Verified:     true,
			},
			expected: 0.95,
		},
		{
			name: "unverified aws secret",
			result: TruffleHogResult{
				DetectorName: "aws",
				Verified:     false,
			},
			expected: 0.9,
		},
		{
			name: "verified github secret",
			result: TruffleHogResult{
				DetectorName: "github",
				Verified:     true,
			},
			expected: 0.95,
		},
		{
			name: "unverified github secret",
			result: TruffleHogResult{
				DetectorName: "github",
				Verified:     false,
			},
			expected: 0.9,
		},
		{
			name: "unknown detector",
			result: TruffleHogResult{
				DetectorName: "unknown",
				Verified:     false,
			},
			expected: 0.7,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := a.calculateConfidence(tt.result)
			assert.Equal(t, tt.expected, confidence)
		})
	}
}

func TestGenerateFixSuggestion(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name         string
		detectorName string
		expected     string
	}{
		{
			name:         "aws detector",
			detectorName: "aws",
			expected:     "Use AWS IAM roles, AWS Secrets Manager, or environment variables instead of hardcoded AWS credentials",
		},
		{
			name:         "github detector",
			detectorName: "github",
			expected:     "Use GitHub's encrypted secrets feature or environment variables instead of hardcoded GitHub tokens",
		},
		{
			name:         "gitlab detector",
			detectorName: "gitlab",
			expected:     "Use GitLab CI/CD variables or environment variables instead of hardcoded GitLab tokens",
		},
		{
			name:         "slack detector",
			detectorName: "slack",
			expected:     "Store Slack tokens in environment variables or a secure secret management system",
		},
		{
			name:         "private key detector",
			detectorName: "privatekey",
			expected:     "Store private keys securely using a key management service and never commit them to version control",
		},
		{
			name:         "unknown detector",
			detectorName: "unknown",
			expected:     "Remove the hardcoded secret and use environment variables or a secure secret management system",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := a.generateFixSuggestion(tt.detectorName)
			assert.Equal(t, tt.expected, suggestion)
		})
	}
}

func TestConvertToFinding(t *testing.T) {
	a := NewAgent()
	
	result := TruffleHogResult{
		SourceMetadata: SourceMetadata{
			Data: struct {
				Git struct {
					Commit     string `json:"commit"`
					File       string `json:"file"`
					Email      string `json:"email"`
					Repository string `json:"repository"`
					Timestamp  string `json:"timestamp"`
					Line       int    `json:"line"`
				} `json:"Git"`
			}{
				Git: struct {
					Commit     string `json:"commit"`
					File       string `json:"file"`
					Email      string `json:"email"`
					Repository string `json:"repository"`
					Timestamp  string `json:"timestamp"`
					Line       int    `json:"line"`
				}{
					File: "config/secrets.yaml",
					Line: 42,
				},
			},
		},
		DetectorName: "aws",
		Verified:     true,
		Redacted:     "AKIA****FAKE",
	}
	
	finding := a.convertToFinding(result, 1)
	
	require.NotNil(t, finding)
	assert.Equal(t, "trufflehog-1", finding.ID)
	assert.Equal(t, AgentName, finding.Tool)
	assert.Equal(t, "trufflehog-aws", finding.RuleID)
	assert.Equal(t, agent.SeverityHigh, finding.Severity)
	assert.Equal(t, agent.CategoryHardcodedSecrets, finding.Category)
	assert.Equal(t, "aws Secret Found", finding.Title)
	assert.Contains(t, finding.Description, "Secret detected by aws detector")
	assert.Contains(t, finding.Description, "VERIFIED")
	assert.Equal(t, "config/secrets.yaml", finding.File)
	assert.Equal(t, 42, finding.Line)
	assert.Equal(t, "AKIA****FAKE", finding.Code)
	assert.Equal(t, 0.95, finding.Confidence)
	assert.NotNil(t, finding.Fix)
	assert.Contains(t, finding.Fix.Suggestion, "AWS IAM roles")
}

func TestFilterFindings(t *testing.T) {
	a := NewAgentWithConfig(AgentConfig{
		Whitelist: []string{
			"test.*",
			".*example.*",
		},
	})
	
	findings := []agent.Finding{
		{
			ID:   "1",
			File: "src/config.js",
			Code: "secret123",
		},
		{
			ID:   "2",
			File: "test/config.js",
			Code: "secret456",
		},
		{
			ID:   "3",
			File: "src/example.js",
			Code: "secret789",
		},
		{
			ID:   "4",
			File: "src/main.js",
			Code: "example_secret",
		},
	}
	
	filtered := a.filterFindings(findings)
	
	// Should only include the first finding (src/config.js)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "1", filtered[0].ID)
	assert.Equal(t, "src/config.js", filtered[0].File)
}

func TestScan_InvalidConfig(t *testing.T) {
	a := NewAgent()
	
	// Test with empty repo URL
	config := agent.ScanConfig{
		RepoURL: "",
		Branch:  "main",
		Timeout: 1 * time.Minute,
	}
	
	result, err := a.Scan(context.Background(), config)
	
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, agent.ScanStatusFailed, result.Status)
	assert.NotEmpty(t, result.Error)
}