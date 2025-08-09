package secrets

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agentscan/agentscan/agents/secrets/gitsecrets"
	"github.com/agentscan/agentscan/agents/secrets/trufflehog"
	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecretScanningIntegration tests both TruffleHog and git-secrets agents
// with repositories containing various types of secrets
func TestSecretScanningIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test repositories with different types of secrets
	testCases := []struct {
		name     string
		files    map[string]string
		expected []ExpectedFinding
	}{
		{
			name: "aws_secrets",
			files: map[string]string{
				"config.yaml": `
database:
  host: localhost
  port: 5432
aws:
  access_key_id: AKIA-FAKE-TEST-KEY-ONLY
  secret_access_key: FAKE-SECRET-KEY-FOR-TESTING-PURPOSES-ONLY
  region: us-west-2
`,
				"src/main.js": `
const AWS = require('aws-sdk');
const aws = new AWS.S3({
  accessKeyId: 'AKIA-FAKE-TEST-KEY-ONLY',
  secretAccessKey: 'FAKE-SECRET-KEY-FOR-TESTING-PURPOSES-ONLY'
});
`,
			},
			expected: []ExpectedFinding{
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "config.yaml",
					Contains:    []string{"aws", "access_key"},
					MinConfidence: 0.8,
				},
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "src/main.js",
					Contains:    []string{"aws", "access"},
					MinConfidence: 0.8,
				},
			},
		},
		{
			name: "github_tokens",
			files: map[string]string{
				".env": `
GITHUB_TOKEN=ghp_FAKE_TOKEN_FOR_TESTING_ONLY
DATABASE_URL=postgresql://user:pass@localhost/db
`,
				"scripts/deploy.sh": `#!/bin/bash
export GITHUB_TOKEN="ghp_FAKE_TOKEN_FOR_TESTING_ONLY"
git push origin main
`,
			},
			expected: []ExpectedFinding{
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        ".env",
					Contains:    []string{"github", "token"},
					MinConfidence: 0.8,
				},
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "scripts/deploy.sh",
					Contains:    []string{"github", "token"},
					MinConfidence: 0.8,
				},
			},
		},
		{
			name: "private_keys",
			files: map[string]string{
				"certs/private.key": `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKB
wQNneCjmrSueCiDHVWHRgWBjxWYIeNfcaHSpE4nqHyXXAcFcjdsQRSCiS5t6/qQa
SskHn6/+4O2d4Jw/EsEbK8dN23LukREoeKUpbRANgMSXQhHIlpiGlyQpkiYDfTv7
biHoUhJL4JBfcpLlAoGBALsvftwmrEOrr956hDo2A4q9iFigMXOEUI9VQQSQKlVV
-----END PRIVATE KEY-----`,
				"config/ssl.conf": `
ssl_certificate /etc/ssl/certs/server.crt
ssl_certificate_key /etc/ssl/private/server.key
ssl_private_key_content = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA7VJTUt9Us8cKBwQNneCjmrSueCiDHVWHRgWBjxWYIeNfcaH\n-----END RSA PRIVATE KEY-----"
`,
			},
			expected: []ExpectedFinding{
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "certs/private.key",
					Contains:    []string{"private", "key"},
					MinConfidence: 0.8,
				},
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "config/ssl.conf",
					Contains:    []string{"private", "key"},
					MinConfidence: 0.8,
				},
			},
		},
		{
			name: "database_credentials",
			files: map[string]string{
				"docker-compose.yml": `
version: '3.8'
services:
  db:
    image: postgres:13
    environment:
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: super_secret_password_123
      POSTGRES_DB: myapp
  redis:
    image: redis:6
    command: redis-server --requirepass redis_password_456
`,
				"src/config.py": `
import os

DATABASE_CONFIG = {
    'host': 'localhost',
    'port': 5432,
    'user': 'admin',
    'password': 'super_secret_password_123',
    'database': 'myapp'
}

REDIS_URL = 'redis://:redis_password_456@localhost:6379/0'
`,
			},
			expected: []ExpectedFinding{
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "docker-compose.yml",
					Contains:    []string{"password"},
					MinConfidence: 0.7,
				},
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "src/config.py",
					Contains:    []string{"password"},
					MinConfidence: 0.7,
				},
			},
		},
		{
			name: "api_keys_and_tokens",
			files: map[string]string{
				"src/services/stripe.js": `
const stripe = require('stripe')('sk_test_FAKE_STRIPE_KEY_FOR_TESTING');

module.exports = {
  createPayment: async (amount) => {
    return await stripe.paymentIntents.create({
      amount: amount,
      currency: 'usd',
    });
  }
};
`,
				"src/services/slack.py": `
import requests

SLACK_BOT_TOKEN = "xoxb-FAKE-TOKEN-FOR-TESTING-ONLY-NOT-REAL"
SLACK_WEBHOOK_URL = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"

def send_message(channel, text):
    headers = {
        'Authorization': f'Bearer {SLACK_BOT_TOKEN}',
        'Content-Type': 'application/json'
    }
    # ... rest of the function
`,
				"config/twilio.json": `{
  "account_sid": "AC1234567890abcdefghijklmnopqrstuvwx",
  "auth_token": "1234567890abcdefghijklmnopqrstuvwx",
  "phone_number": "+1234567890"
}`,
			},
			expected: []ExpectedFinding{
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "src/services/stripe.js",
					Contains:    []string{"stripe", "key"},
					MinConfidence: 0.8,
				},
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "src/services/slack.py",
					Contains:    []string{"slack", "token"},
					MinConfidence: 0.8,
				},
				{
					Category:    agent.CategoryHardcodedSecrets,
					Severity:    agent.SeverityHigh,
					File:        "config/twilio.json",
					Contains:    []string{"twilio", "token"},
					MinConfidence: 0.8,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary git repository
			repoDir := createTestRepository(t, tc.files)
			defer os.RemoveAll(repoDir)

			// Test TruffleHog agent
			t.Run("trufflehog", func(t *testing.T) {
				testAgentWithRepository(t, trufflehog.NewAgent(), repoDir, tc.expected)
			})

			// Test git-secrets agent
			t.Run("git-secrets", func(t *testing.T) {
				testAgentWithRepository(t, gitsecrets.NewAgent(), repoDir, tc.expected)
			})
		})
	}
}

// TestWhitelistFunctionality tests that both agents properly filter findings based on whitelist
func TestWhitelistFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	files := map[string]string{
		"src/config.js": `
const config = {
  apiKey: 'sk_test_FAKE_STRIPE_KEY_FOR_TESTING',
  secret: 'real_secret_value'
};
`,
		"test/fixtures.js": `
const testConfig = {
  apiKey: 'sk_test_example_key_for_testing',
  secret: 'test_secret_value'
};
`,
		"examples/sample.js": `
const exampleConfig = {
  apiKey: 'sk_test_example_api_key',
  secret: 'example_secret'
};
`,
	}

	repoDir := createTestRepository(t, files)
	defer os.RemoveAll(repoDir)

	// Test TruffleHog with whitelist
	t.Run("trufflehog_with_whitelist", func(t *testing.T) {
		truffleAgent := trufflehog.NewAgentWithConfig(trufflehog.AgentConfig{
			DockerImage:    trufflehog.DefaultImage,
			MaxMemoryMB:    1024,
			MaxCPUCores:    2.0,
			DefaultTimeout: 15 * time.Minute,
			ScanGitHistory: true,
			MaxDepth:       100,
			Whitelist: []string{
				"test/.*",     // Ignore test files
				"examples/.*", // Ignore example files
				".*example.*", // Ignore anything with "example" in content
			},
		})

		config := agent.ScanConfig{
			RepoURL: "file://" + repoDir,
			Branch:  "main",
			Timeout: 2 * time.Minute,
		}

		result, err := truffleAgent.Scan(context.Background(), config)
		require.NoError(t, err)
		assert.Equal(t, agent.ScanStatusCompleted, result.Status)

		// Should only find secrets in src/config.js (not filtered by whitelist)
		realSecrets := 0
		for _, finding := range result.Findings {
			if finding.File == "src/config.js" {
				realSecrets++
			}
		}
		assert.Greater(t, realSecrets, 0, "Should find at least one real secret in src/config.js")
	})

	// Test git-secrets with whitelist
	t.Run("git-secrets_with_whitelist", func(t *testing.T) {
		gitSecretsAgent := gitsecrets.NewAgentWithConfig(gitsecrets.AgentConfig{
			DockerImage:     gitsecrets.DefaultImage,
			MaxMemoryMB:     512,
			MaxCPUCores:     1.0,
			DefaultTimeout:  10 * time.Minute,
			ProviderPatterns: []string{"aws"},
			ScanCommits:     true,
			Whitelist: []string{
				"test/.*",     // Ignore test files
				"examples/.*", // Ignore example files
				".*example.*", // Ignore anything with "example" in content
			},
		})

		config := agent.ScanConfig{
			RepoURL: "file://" + repoDir,
			Branch:  "main",
			Timeout: 2 * time.Minute,
		}

		result, err := gitSecretsAgent.Scan(context.Background(), config)
		require.NoError(t, err)
		assert.Equal(t, agent.ScanStatusCompleted, result.Status)

		// Should only find secrets in src/config.js (not filtered by whitelist)
		realSecrets := 0
		for _, finding := range result.Findings {
			if finding.File == "src/config.js" {
				realSecrets++
			}
		}
		assert.Greater(t, realSecrets, 0, "Should find at least one real secret in src/config.js")
	})
}

// TestHighSeverityFlagging tests that all detected secrets are flagged as high severity
func TestHighSeverityFlagging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	files := map[string]string{
		"mixed_secrets.txt": `
# Various types of secrets with different confidence levels
aws_access_key_id = AKIA-FAKE-TEST-KEY-ONLY
github_token = ghp_FAKE_TOKEN_FOR_TESTING_ONLY
stripe_key = sk_test_FAKE_STRIPE_KEY_FOR_TESTING
database_password = my_weak_password
api_secret = some_generic_secret_value
jwt_secret = super_secret_jwt_signing_key
`,
	}

	repoDir := createTestRepository(t, files)
	defer os.RemoveAll(repoDir)

	agents := []agent.SecurityAgent{
		trufflehog.NewAgent(),
		gitsecrets.NewAgent(),
	}

	for i, testAgent := range agents {
		t.Run(fmt.Sprintf("agent_%d_high_severity", i), func(t *testing.T) {
			config := agent.ScanConfig{
				RepoURL: "file://" + repoDir,
				Branch:  "main",
				Timeout: 2 * time.Minute,
			}

			result, err := testAgent.Scan(context.Background(), config)
			require.NoError(t, err)
			assert.Equal(t, agent.ScanStatusCompleted, result.Status)

			// All findings should be high severity as per requirements
			for _, finding := range result.Findings {
				assert.Equal(t, agent.SeverityHigh, finding.Severity,
					"All secrets should be flagged as high severity, but found %s for %s",
					finding.Severity, finding.RuleID)
			}

			// Should find at least some secrets
			assert.Greater(t, len(result.Findings), 0, "Should detect at least some secrets")
		})
	}
}

// ExpectedFinding represents what we expect to find in a test
type ExpectedFinding struct {
	Category      agent.VulnCategory
	Severity      agent.Severity
	File          string
	Contains      []string // Strings that should be present in title/description
	MinConfidence float64
}

// createTestRepository creates a temporary git repository with the given files
func createTestRepository(t *testing.T, files map[string]string) string {
	tempDir, err := os.MkdirTemp("", "secret-scan-test-*")
	require.NoError(t, err)

	// Initialize git repository
	runCommand(t, tempDir, "git", "init")
	runCommand(t, tempDir, "git", "config", "user.email", "test@example.com")
	runCommand(t, tempDir, "git", "config", "user.name", "Test User")

	// Create files
	for filename, content := range files {
		filePath := filepath.Join(tempDir, filename)
		
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}

		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Add and commit files
	runCommand(t, tempDir, "git", "add", ".")
	runCommand(t, tempDir, "git", "commit", "-m", "Initial commit with secrets")

	return tempDir
}

// testAgentWithRepository tests an agent against a repository and validates expected findings
func testAgentWithRepository(t *testing.T, testAgent agent.SecurityAgent, repoDir string, expected []ExpectedFinding) {
	config := agent.ScanConfig{
		RepoURL: "file://" + repoDir,
		Branch:  "main",
		Timeout: 2 * time.Minute,
	}

	result, err := testAgent.Scan(context.Background(), config)
	require.NoError(t, err)
	assert.Equal(t, agent.ScanStatusCompleted, result.Status)

	// Validate that we found the expected types of secrets
	for _, expectedFinding := range expected {
		found := false
		for _, actualFinding := range result.Findings {
			if actualFinding.Category == expectedFinding.Category &&
				actualFinding.Severity == expectedFinding.Severity &&
				actualFinding.File == expectedFinding.File &&
				actualFinding.Confidence >= expectedFinding.MinConfidence {

				// Check if the finding contains expected keywords
				containsAll := true
				searchText := strings.ToLower(actualFinding.Title + " " + actualFinding.Description + " " + actualFinding.RuleID)
				for _, keyword := range expectedFinding.Contains {
					if !strings.Contains(searchText, strings.ToLower(keyword)) {
						containsAll = false
						break
					}
				}

				if containsAll {
					found = true
					break
				}
			}
		}

		assert.True(t, found, "Expected to find %+v but didn't", expectedFinding)
	}

	// Validate that all findings are properly structured
	for _, finding := range result.Findings {
		assert.NotEmpty(t, finding.ID, "Finding should have an ID")
		assert.NotEmpty(t, finding.Tool, "Finding should specify the tool")
		assert.NotEmpty(t, finding.RuleID, "Finding should have a rule ID")
		assert.Equal(t, agent.CategoryHardcodedSecrets, finding.Category, "All secret findings should be categorized as hardcoded secrets")
		assert.Equal(t, agent.SeverityHigh, finding.Severity, "All secrets should be high severity as per requirements")
		assert.NotEmpty(t, finding.Title, "Finding should have a title")
		assert.NotEmpty(t, finding.Description, "Finding should have a description")
		assert.NotEmpty(t, finding.File, "Finding should specify the file")
		assert.Greater(t, finding.Confidence, 0.0, "Finding should have a confidence score")
		assert.LessOrEqual(t, finding.Confidence, 1.0, "Confidence should not exceed 1.0")
		assert.NotNil(t, finding.Fix, "Finding should include fix suggestions")
		assert.NotEmpty(t, finding.Fix.Description, "Fix suggestion should have a description")
		assert.NotEmpty(t, finding.Fix.Suggestion, "Fix suggestion should have a suggestion")
	}
}

// runCommand runs a command in the specified directory
func runCommand(t *testing.T, dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	err := cmd.Run()
	require.NoError(t, err, "Command failed: %s %v", name, args)
}