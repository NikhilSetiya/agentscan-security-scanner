package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/internal/orchestrator"
	"github.com/agentscan/agentscan/pkg/agent"
)

func TestRegisterAgents(t *testing.T) {
	// Create agent manager
	agentManager := orchestrator.NewAgentManager()

	// Register all agents
	err := registerAgents(agentManager)
	require.NoError(t, err, "Failed to register agents")

	// Verify all expected agents are registered
	expectedAgents := []string{
		"semgrep",
		"bandit", 
		"eslint-security",
		"npm-audit",
		"pip-audit",
		"govulncheck",
		"trufflehog",
		"git-secrets",
	}

	for _, agentName := range expectedAgents {
		t.Run("agent_"+agentName, func(t *testing.T) {
			// Check if agent is registered
			agentInstance, err := agentManager.GetAgent(agentName)
			assert.NoError(t, err, "Agent %s should be registered", agentName)
			assert.NotNil(t, agentInstance, "Agent %s instance should not be nil", agentName)

			// Verify agent implements the SecurityAgent interface
			_, ok := agentInstance.(agent.SecurityAgent)
			assert.True(t, ok, "Agent %s should implement SecurityAgent interface", agentName)

			// Test agent configuration
			config := agentInstance.GetConfig()
			assert.NotEmpty(t, config.Name, "Agent %s should have a name", agentName)
			assert.NotEmpty(t, config.Version, "Agent %s should have a version", agentName)
		})
	}
}

func TestESLintAgentIntegration(t *testing.T) {
	// Create agent manager and register agents
	agentManager := orchestrator.NewAgentManager()
	err := registerAgents(agentManager)
	require.NoError(t, err)

	// Get ESLint agent
	eslintAgent, err := agentManager.GetAgent("eslint-security")
	require.NoError(t, err, "ESLint agent should be registered")
	require.NotNil(t, eslintAgent, "ESLint agent should not be nil")

	// Test agent configuration
	config := eslintAgent.GetConfig()
	assert.Equal(t, "eslint-security", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Contains(t, config.SupportedLangs, "javascript")
	assert.Contains(t, config.SupportedLangs, "typescript")

	// Test health check
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = eslintAgent.HealthCheck(ctx)
	// Note: This might fail in CI if Docker is not available, so we'll just log the result
	if err != nil {
		t.Logf("ESLint agent health check failed (expected in CI without Docker): %v", err)
	} else {
		t.Log("ESLint agent health check passed")
	}
}

func TestAgentManagerOperations(t *testing.T) {
	agentManager := orchestrator.NewAgentManager()
	
	// Test with empty manager
	agents := agentManager.ListAgents()
	assert.Empty(t, agents, "Agent manager should start empty")

	// Register agents
	err := registerAgents(agentManager)
	require.NoError(t, err)

	// Test listing agents
	agents = agentManager.ListAgents()
	assert.Len(t, agents, 8, "Should have 8 registered agents")

	// Test getting non-existent agent
	nonExistentAgent, err := agentManager.GetAgent("non-existent")
	assert.Error(t, err, "Non-existent agent should return error")
	assert.Nil(t, nonExistentAgent, "Non-existent agent should be nil")

	// Test health check all
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = agentManager.HealthCheckAll(ctx)
	// Health check might fail in CI without Docker, so we'll just verify it doesn't panic
	if err != nil {
		t.Logf("Health check all failed (expected in CI): %v", err)
	}
}

func TestESLintAgentSpecificFunctionality(t *testing.T) {
	agentManager := orchestrator.NewAgentManager()
	err := registerAgents(agentManager)
	require.NoError(t, err)

	eslintAgent, err := agentManager.GetAgent("eslint-security")
	require.NoError(t, err)
	require.NotNil(t, eslintAgent)

	// Test that ESLint agent supports JavaScript and TypeScript
	config := eslintAgent.GetConfig()
	supportedLangs := config.SupportedLangs
	
	assert.Contains(t, supportedLangs, "javascript", "ESLint should support JavaScript")
	assert.Contains(t, supportedLangs, "typescript", "ESLint should support TypeScript")
	assert.Contains(t, supportedLangs, "jsx", "ESLint should support JSX")
	assert.Contains(t, supportedLangs, "tsx", "ESLint should support TSX")

	// Test agent vulnerability categories
	expectedCategories := []agent.VulnCategory{
		agent.CategoryXSS,
		agent.CategoryCommandInjection,
		agent.CategoryPathTraversal,
		agent.CategoryInsecureCrypto,
		agent.CategoryCSRF,
		agent.CategoryInsecureDeserialization,
		agent.CategoryMisconfiguration,
		agent.CategoryOther,
	}

	for _, category := range expectedCategories {
		assert.Contains(t, config.Categories, category, "ESLint should support category %s", category)
	}

	// Test that agent requires Docker
	assert.True(t, config.RequiresDocker, "ESLint agent should require Docker")

	// Test agent version info
	versionInfo := eslintAgent.GetVersion()
	assert.Equal(t, "1.0.0", versionInfo.AgentVersion)
	assert.NotEmpty(t, versionInfo.BuildDate)
}

func TestSecretScanningAgentsIntegration(t *testing.T) {
	// Create agent manager and register agents
	agentManager := orchestrator.NewAgentManager()
	err := registerAgents(agentManager)
	require.NoError(t, err)

	// Test TruffleHog agent
	t.Run("TruffleHog", func(t *testing.T) {
		trufflehogAgent, err := agentManager.GetAgent("trufflehog")
		require.NoError(t, err, "TruffleHog agent should be registered")
		require.NotNil(t, trufflehogAgent, "TruffleHog agent should not be nil")

		// Test agent configuration
		config := trufflehogAgent.GetConfig()
		assert.Equal(t, "trufflehog", config.Name)
		assert.Equal(t, "1.0.0", config.Version)
		assert.Contains(t, config.SupportedLangs, "*") // Language agnostic
		assert.Contains(t, config.Categories, agent.CategoryHardcodedSecrets)
		assert.True(t, config.RequiresDocker, "TruffleHog agent should require Docker")

		// Test health check
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = trufflehogAgent.HealthCheck(ctx)
		if err != nil {
			t.Logf("TruffleHog agent health check failed (expected in CI without Docker): %v", err)
		} else {
			t.Log("TruffleHog agent health check passed")
		}

		// Test agent version info
		versionInfo := trufflehogAgent.GetVersion()
		assert.Equal(t, "1.0.0", versionInfo.AgentVersion)
		assert.NotEmpty(t, versionInfo.BuildDate)
	})

	// Test git-secrets agent
	t.Run("GitSecrets", func(t *testing.T) {
		gitSecretsAgent, err := agentManager.GetAgent("git-secrets")
		require.NoError(t, err, "git-secrets agent should be registered")
		require.NotNil(t, gitSecretsAgent, "git-secrets agent should not be nil")

		// Test agent configuration
		config := gitSecretsAgent.GetConfig()
		assert.Equal(t, "git-secrets", config.Name)
		assert.Equal(t, "1.0.0", config.Version)
		assert.Contains(t, config.SupportedLangs, "*") // Language agnostic
		assert.Contains(t, config.Categories, agent.CategoryHardcodedSecrets)
		assert.True(t, config.RequiresDocker, "git-secrets agent should require Docker")

		// Test health check
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = gitSecretsAgent.HealthCheck(ctx)
		if err != nil {
			t.Logf("git-secrets agent health check failed (expected in CI without Docker): %v", err)
		} else {
			t.Log("git-secrets agent health check passed")
		}

		// Test agent version info
		versionInfo := gitSecretsAgent.GetVersion()
		assert.Equal(t, "1.0.0", versionInfo.AgentVersion)
		assert.NotEmpty(t, versionInfo.BuildDate)
	})
}