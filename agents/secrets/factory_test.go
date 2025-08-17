package secrets

import (
	"testing"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/agents/secrets/gitsecrets"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/secrets/trufflehog"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)
}

func TestCreateAgent(t *testing.T) {
	factory := NewFactory()
	
	tests := []struct {
		name      string
		agentType AgentType
		expectErr bool
	}{
		{
			name:      "create trufflehog agent",
			agentType: TruffleHogAgent,
			expectErr: false,
		},
		{
			name:      "create git-secrets agent",
			agentType: GitSecretsAgent,
			expectErr: false,
		},
		{
			name:      "create unknown agent",
			agentType: AgentType("unknown"),
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := factory.CreateAgent(tt.agentType)
			
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
			}
		})
	}
}

func TestCreateTruffleHogAgent(t *testing.T) {
	factory := NewFactory()
	
	config := trufflehog.AgentConfig{
		DockerImage:    "custom/trufflehog:test",
		MaxMemoryMB:    512,
		MaxCPUCores:    1.0,
		ScanGitHistory: false,
		MaxDepth:       50,
	}
	
	agent := factory.CreateTruffleHogAgent(config)
	
	assert.NotNil(t, agent)
	agentConfig := agent.GetConfig()
	assert.Equal(t, "trufflehog", agentConfig.Name)
}

func TestCreateGitSecretsAgent(t *testing.T) {
	factory := NewFactory()
	
	config := gitsecrets.AgentConfig{
		DockerImage:     "custom/git-secrets:test",
		MaxMemoryMB:     256,
		MaxCPUCores:     0.5,
		ScanCommits:     false,
	}
	
	agent := factory.CreateGitSecretsAgent(config)
	
	assert.NotNil(t, agent)
	agentConfig := agent.GetConfig()
	assert.Equal(t, "git-secrets", agentConfig.Name)
}

func TestGetAllAgentTypes(t *testing.T) {
	factory := NewFactory()
	
	agentTypes := factory.GetAllAgentTypes()
	
	assert.Len(t, agentTypes, 2)
	assert.Contains(t, agentTypes, TruffleHogAgent)
	assert.Contains(t, agentTypes, GitSecretsAgent)
}

func TestGetAgentCapabilities(t *testing.T) {
	factory := NewFactory()
	
	tests := []struct {
		name      string
		agentType AgentType
		expectErr bool
	}{
		{
			name:      "get trufflehog capabilities",
			agentType: TruffleHogAgent,
			expectErr: false,
		},
		{
			name:      "get git-secrets capabilities",
			agentType: GitSecretsAgent,
			expectErr: false,
		},
		{
			name:      "get unknown agent capabilities",
			agentType: AgentType("unknown"),
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities, err := factory.GetAgentCapabilities(tt.agentType)
			
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, capabilities.Name)
				assert.NotEmpty(t, capabilities.Version)
				assert.NotEmpty(t, capabilities.Categories)
			}
		})
	}
}

func TestCreateAllAgents(t *testing.T) {
	factory := NewFactory()
	
	agents := factory.CreateAllAgents()
	
	assert.Len(t, agents, 2)
	
	// Verify we have both agent types
	agentNames := make(map[string]bool)
	for _, agent := range agents {
		config := agent.GetConfig()
		agentNames[config.Name] = true
	}
	
	assert.True(t, agentNames["trufflehog"])
	assert.True(t, agentNames["git-secrets"])
}

func TestDefaultTruffleHogConfig(t *testing.T) {
	config := DefaultTruffleHogConfig()
	
	assert.Equal(t, trufflehog.DefaultImage, config.DockerImage)
	assert.Equal(t, 1024, config.MaxMemoryMB)
	assert.Equal(t, 2.0, config.MaxCPUCores)
	assert.Equal(t, 15*time.Minute, config.DefaultTimeout)
	assert.True(t, config.ScanGitHistory)
	assert.Equal(t, 100, config.MaxDepth)
	assert.Empty(t, config.Whitelist)
}

func TestDefaultGitSecretsConfig(t *testing.T) {
	config := DefaultGitSecretsConfig()
	
	assert.Equal(t, gitsecrets.DefaultImage, config.DockerImage)
	assert.Equal(t, 512, config.MaxMemoryMB)
	assert.Equal(t, 1.0, config.MaxCPUCores)
	assert.Equal(t, 10*time.Minute, config.DefaultTimeout)
	assert.Contains(t, config.ProviderPatterns, "aws")
	assert.Contains(t, config.ProviderPatterns, "azure")
	assert.Contains(t, config.ProviderPatterns, "gcp")
	assert.True(t, config.ScanCommits)
	assert.Empty(t, config.Whitelist)
}

func TestSecureDefaultTruffleHogConfig(t *testing.T) {
	config := SecureDefaultTruffleHogConfig()
	
	// Should include high-value detectors
	assert.Contains(t, config.IncludeDetectors, "aws")
	assert.Contains(t, config.IncludeDetectors, "github")
	assert.Contains(t, config.IncludeDetectors, "privatekey")
	
	// Should exclude noisy detectors
	assert.Contains(t, config.ExcludeDetectors, "generic")
	assert.Contains(t, config.ExcludeDetectors, "uri")
	
	// Should have whitelist patterns
	assert.NotEmpty(t, config.Whitelist)
	assert.Contains(t, config.Whitelist, `test/.*`)
	assert.Contains(t, config.Whitelist, `examples?/.*`)
	assert.Contains(t, config.Whitelist, `.*example.*`)
}

func TestSecureDefaultGitSecretsConfig(t *testing.T) {
	config := SecureDefaultGitSecretsConfig()
	
	// Should have custom patterns
	assert.NotEmpty(t, config.CustomPatterns)
	
	// Should have custom patterns (at least one)
	assert.Greater(t, len(config.CustomPatterns), 0, "Should have custom patterns")
	
	// Should have whitelist patterns
	assert.NotEmpty(t, config.Whitelist)
	assert.Contains(t, config.Whitelist, `test/.*`)
	assert.Contains(t, config.Whitelist, `examples?/.*`)
	assert.Contains(t, config.Whitelist, `.*example.*`)
}

func TestAgentTypeConstants(t *testing.T) {
	assert.Equal(t, "trufflehog", string(TruffleHogAgent))
	assert.Equal(t, "git-secrets", string(GitSecretsAgent))
}

func TestFactoryIntegration(t *testing.T) {
	factory := NewFactory()
	
	// Test creating agents with secure defaults
	truffleConfig := SecureDefaultTruffleHogConfig()
	truffleAgent := factory.CreateTruffleHogAgent(truffleConfig)
	require.NotNil(t, truffleAgent)
	
	gitSecretsConfig := SecureDefaultGitSecretsConfig()
	gitSecretsAgent := factory.CreateGitSecretsAgent(gitSecretsConfig)
	require.NotNil(t, gitSecretsAgent)
	
	// Verify both agents implement the SecurityAgent interface
	agents := []agent.SecurityAgent{truffleAgent, gitSecretsAgent}
	for _, securityAgent := range agents {
		config := securityAgent.GetConfig()
		version := securityAgent.GetVersion()
		
		assert.NotEmpty(t, config.Name, "Agent should have a name")
		assert.NotEmpty(t, version.AgentVersion, "Agent should have a version")
	}
}

