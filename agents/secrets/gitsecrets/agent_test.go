package gitsecrets

import (
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
	assert.True(t, a.config.ScanCommits)
	assert.Contains(t, a.config.ProviderPatterns, "aws")
	assert.Contains(t, a.config.ProviderPatterns, "azure")
	assert.Contains(t, a.config.ProviderPatterns, "gcp")
}

func TestNewAgentWithConfig(t *testing.T) {
	config := AgentConfig{
		DockerImage:     "custom/git-secrets:test",
		MaxMemoryMB:     256,
		MaxCPUCores:     0.5,
		DefaultTimeout:  5 * time.Minute,
		CustomPatterns:  []string{"custom-pattern-1", "custom-pattern-2"},
		ProviderPatterns: []string{"aws"},
		Whitelist:       []string{"test.*", "example.*"},
		ScanCommits:     false,
	}
	
	a := NewAgentWithConfig(config)
	
	assert.NotNil(t, a)
	assert.Equal(t, config.DockerImage, a.config.DockerImage)
	assert.Equal(t, config.MaxMemoryMB, a.config.MaxMemoryMB)
	assert.Equal(t, config.MaxCPUCores, a.config.MaxCPUCores)
	assert.Equal(t, config.DefaultTimeout, a.config.DefaultTimeout)
	assert.Equal(t, config.CustomPatterns, a.config.CustomPatterns)
	assert.Equal(t, config.ProviderPatterns, a.config.ProviderPatterns)
	assert.Equal(t, config.Whitelist, a.config.Whitelist)
	assert.Equal(t, config.ScanCommits, a.config.ScanCommits)
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

func TestDetectSecretType(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "aws access key",
			content:  "aws_access_key_id = AKIA-FAKE-TEST-KEY",
			expected: "aws-access-key",
		},
		{
			name:     "aws secret key",
			content:  "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			expected: "aws-secret-key",
		},
		{
			name:     "private key",
			content:  "-----BEGIN PRIVATE KEY-----",
			expected: "private-key",
		},
		{
			name:     "password",
			content:  "password = mysecretpassword",
			expected: "password",
		},
		{
			name:     "token",
			content:  "github_token = ghp_FAKE_TOKEN_TEST",
			expected: "token",
		},
		{
			name:     "api key",
			content:  "api_key = sk-1234567890abcdef",
			expected: "api-key",
		},
		{
			name:     "generic secret",
			content:  "secret = mysecret",
			expected: "secret",
		},
		{
			name:     "unknown",
			content:  "some random text",
			expected: "unknown",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.detectSecretType(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateDescription(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		result   GitSecretsResult
		expected string
	}{
		{
			name: "working directory secret",
			result: GitSecretsResult{
				Content:     "aws_access_key_id = AKIA-FAKE-TEST-KEY",
				PatternType: "working-directory",
			},
			expected: "git-secrets detected a potential aws-access-key in working-directory. This secret is present in the current working directory.",
		},
		{
			name: "commit history secret",
			result: GitSecretsResult{
				Content:     "password = mysecret",
				PatternType: "commit-history",
			},
			expected: "git-secrets detected a potential password in commit-history. This secret was found in the git commit history and may have been exposed in previous commits.",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := a.generateDescription(tt.result)
			assert.Equal(t, tt.expected, description)
		})
	}
}

func TestRedactSecret(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "short secret",
			content:  "secret",
			expected: "******",
		},
		{
			name:     "medium secret",
			content:  "mysecret",
			expected: "********",
		},
		{
			name:     "long secret",
			content:  "AKIA-FAKE-TEST-KEY-ONLY",
			expected: "AKIA****************ONLY",
		},
		{
			name:     "very long secret",
			content:  "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			expected: "wJal********************************EKEY",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacted := a.redactSecret(tt.content)
			assert.Equal(t, tt.expected, redacted)
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		result   GitSecretsResult
		expected float64
	}{
		{
			name: "aws access key",
			result: GitSecretsResult{
				Content: "aws_access_key_id = AKIAIOSFODNN7REAL",
			},
			expected: 0.9,
		},
		{
			name: "private key",
			result: GitSecretsResult{
				Content: "-----BEGIN PRIVATE KEY-----",
			},
			expected: 0.9,
		},
		{
			name: "github token",
			result: GitSecretsResult{
				Content: "ghp_FAKE_TOKEN_TEST",
			},
			expected: 0.9,
		},
		{
			name: "gitlab token",
			result: GitSecretsResult{
				Content: "glpat-1234567890abcdef",
			},
			expected: 0.9,
		},
		{
			name: "example secret",
			result: GitSecretsResult{
				Content: "example_secret_key",
			},
			expected: 0.6,
		},
		{
			name: "test secret",
			result: GitSecretsResult{
				Content: "test_password",
			},
			expected: 0.6,
		},
		{
			name: "generic secret",
			result: GitSecretsResult{
				Content: "some_secret_value",
			},
			expected: 0.8,
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
		name     string
		result   GitSecretsResult
		expected string
	}{
		{
			name: "aws secret",
			result: GitSecretsResult{
				Content: "aws_access_key_id = AKIA-FAKE-TEST-KEY",
			},
			expected: "Use AWS IAM roles, AWS Secrets Manager, or environment variables instead of hardcoded AWS credentials",
		},
		{
			name: "private key",
			result: GitSecretsResult{
				Content: "-----BEGIN PRIVATE KEY-----",
			},
			expected: "Store private keys securely using a key management service and never commit them to version control",
		},
		{
			name: "api key",
			result: GitSecretsResult{
				Content: "api_key = sk-1234567890abcdef",
			},
			expected: "Store API keys in environment variables or a secure secret management system",
		},
		{
			name: "token",
			result: GitSecretsResult{
				Content: "github_token = ghp_FAKE_TOKEN_TEST",
			},
			expected: "Use environment variables for tokens and ensure they are properly rotated",
		},
		{
			name: "password",
			result: GitSecretsResult{
				Content: "password = mysecret",
			},
			expected: "Use environment variables or a secure configuration management system for passwords",
		},
		{
			name: "unknown",
			result: GitSecretsResult{
				Content: "some_secret",
			},
			expected: "Remove the hardcoded secret and use environment variables or a secure secret management system",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := a.generateFixSuggestion(tt.result)
			assert.Equal(t, tt.expected, suggestion)
		})
	}
}

func TestParseGitSecretsOutput(t *testing.T) {
	a := NewAgent()
	
	output := `config/secrets.yaml:10:5:aws_access_key_id = AKIA-FAKE-TEST-KEY
src/main.js:25:12:const token = "ghp_FAKE_TOKEN_TEST"
README.md:50:1:password = mysecret`
	
	results, err := a.parseGitSecretsOutput(output, "working-directory")
	
	require.NoError(t, err)
	assert.Len(t, results, 3)
	
	// Check first result
	assert.Equal(t, "config/secrets.yaml", results[0].File)
	assert.Equal(t, 10, results[0].Line)
	assert.Equal(t, 5, results[0].Column)
	assert.Equal(t, "aws_access_key_id = AKIA-FAKE-TEST-KEY", results[0].Content)
	assert.Equal(t, "working-directory", results[0].PatternType)
	
	// Check second result
	assert.Equal(t, "src/main.js", results[1].File)
	assert.Equal(t, 25, results[1].Line)
	assert.Equal(t, 12, results[1].Column)
	assert.Equal(t, `const token = "ghp_FAKE_TOKEN_TEST"`, results[1].Content)
	
	// Check third result
	assert.Equal(t, "README.md", results[2].File)
	assert.Equal(t, 50, results[2].Line)
	assert.Equal(t, 1, results[2].Column)
	assert.Equal(t, "password = mysecret", results[2].Content)
}

func TestConvertResultsToFindings(t *testing.T) {
	a := NewAgent()
	
	results := []GitSecretsResult{
		{
			File:        "config/secrets.yaml",
			Line:        10,
			Column:      5,
			Content:     "aws_access_key_id = AKIAIOSFODNN7REAL",
			PatternType: "working-directory",
		},
		{
			File:        "src/main.js",
			Line:        25,
			Column:      12,
			Content:     "const token = \"ghp_FAKE_TOKEN_TEST\"",
			PatternType: "commit-history",
		},
	}
	
	config := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo",
		Branch:  "main",
	}
	
	findings := a.convertResultsToFindings(results, config)
	
	require.Len(t, findings, 2)
	
	// Check first finding
	finding1 := findings[0]
	assert.Equal(t, "git-secrets-1", finding1.ID)
	assert.Equal(t, AgentName, finding1.Tool)
	assert.Equal(t, "git-secrets-aws-access-key", finding1.RuleID)
	assert.Equal(t, agent.SeverityHigh, finding1.Severity)
	assert.Equal(t, agent.CategoryHardcodedSecrets, finding1.Category)
	assert.Equal(t, "Secret Pattern Detected (working-directory)", finding1.Title)
	assert.Contains(t, finding1.Description, "aws-access-key")
	assert.Equal(t, "config/secrets.yaml", finding1.File)
	assert.Equal(t, 10, finding1.Line)
	assert.Equal(t, 5, finding1.Column)
	assert.Equal(t, "aws_*******************KEY", finding1.Code)
	assert.Equal(t, 0.9, finding1.Confidence)
	assert.NotNil(t, finding1.Fix)
	
	// Check second finding
	finding2 := findings[1]
	assert.Equal(t, "git-secrets-2", finding2.ID)
	assert.Equal(t, "git-secrets-token", finding2.RuleID)
	assert.Equal(t, "Secret Pattern Detected (commit-history)", finding2.Title)
	assert.Contains(t, finding2.Description, "commit history")
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