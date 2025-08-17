package secrets

import (
	"fmt"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/agents/secrets/gitsecrets"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/secrets/trufflehog"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

// AgentType represents the type of secret scanning agent
type AgentType string

const (
	// TruffleHogAgent represents the TruffleHog secret scanner
	TruffleHogAgent AgentType = "trufflehog"
	
	// GitSecretsAgent represents the git-secrets scanner
	GitSecretsAgent AgentType = "git-secrets"
)

// Factory provides methods to create secret scanning agents
type Factory struct{}

// NewFactory creates a new agent factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateAgent creates a secret scanning agent of the specified type
func (f *Factory) CreateAgent(agentType AgentType) (agent.SecurityAgent, error) {
	switch agentType {
	case TruffleHogAgent:
		return trufflehog.NewAgent(), nil
	case GitSecretsAgent:
		return gitsecrets.NewAgent(), nil
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}

// CreateTruffleHogAgent creates a TruffleHog agent with custom configuration
func (f *Factory) CreateTruffleHogAgent(config trufflehog.AgentConfig) agent.SecurityAgent {
	return trufflehog.NewAgentWithConfig(config)
}

// CreateGitSecretsAgent creates a git-secrets agent with custom configuration
func (f *Factory) CreateGitSecretsAgent(config gitsecrets.AgentConfig) agent.SecurityAgent {
	return gitsecrets.NewAgentWithConfig(config)
}

// GetAllAgentTypes returns all available secret scanning agent types
func (f *Factory) GetAllAgentTypes() []AgentType {
	return []AgentType{
		TruffleHogAgent,
		GitSecretsAgent,
	}
}

// GetAgentCapabilities returns the capabilities of a specific agent type
func (f *Factory) GetAgentCapabilities(agentType AgentType) (agent.AgentConfig, error) {
	securityAgent, err := f.CreateAgent(agentType)
	if err != nil {
		return agent.AgentConfig{}, err
	}
	
	return securityAgent.GetConfig(), nil
}

// CreateAllAgents creates instances of all available secret scanning agents
func (f *Factory) CreateAllAgents() []agent.SecurityAgent {
	var agents []agent.SecurityAgent
	
	for _, agentType := range f.GetAllAgentTypes() {
		if securityAgent, err := f.CreateAgent(agentType); err == nil {
			agents = append(agents, securityAgent)
		}
	}
	
	return agents
}

// DefaultTruffleHogConfig returns a default configuration for TruffleHog
func DefaultTruffleHogConfig() trufflehog.AgentConfig {
	return trufflehog.AgentConfig{
		DockerImage:    trufflehog.DefaultImage,
		MaxMemoryMB:    1024,
		MaxCPUCores:    2.0,
		DefaultTimeout: 15 * time.Minute,
		ScanGitHistory: true,
		MaxDepth:       100,
		Whitelist:      []string{},
	}
}

// DefaultGitSecretsConfig returns a default configuration for git-secrets
func DefaultGitSecretsConfig() gitsecrets.AgentConfig {
	return gitsecrets.AgentConfig{
		DockerImage:     gitsecrets.DefaultImage,
		MaxMemoryMB:     512,
		MaxCPUCores:     1.0,
		DefaultTimeout:  10 * time.Minute,
		ProviderPatterns: []string{"aws", "azure", "gcp"},
		ScanCommits:     true,
		Whitelist:       []string{},
	}
}

// SecureDefaultTruffleHogConfig returns a security-focused configuration for TruffleHog
func SecureDefaultTruffleHogConfig() trufflehog.AgentConfig {
	config := DefaultTruffleHogConfig()
	
	// Include high-value detectors
	config.IncludeDetectors = []string{
		"aws",
		"github",
		"gitlab",
		"slack",
		"stripe",
		"twilio",
		"mailgun",
		"sendgrid",
		"privatekey",
		"jwt",
	}
	
	// Exclude noisy detectors
	config.ExcludeDetectors = []string{
		"generic",
		"uri",
	}
	
	// Common whitelist patterns for test/example files
	config.Whitelist = []string{
		`test/.*`,
		`tests/.*`,
		`.*test.*`,
		`examples?/.*`,
		`.*example.*`,
		`.*placeholder.*`,
		`.*dummy.*`,
		`.*fake.*`,
		`.*mock.*`,
		`docs?/.*`,
		`.*\.md$`,
		`.*\.txt$`,
	}
	
	return config
}

// SecureDefaultGitSecretsConfig returns a security-focused configuration for git-secrets
func SecureDefaultGitSecretsConfig() gitsecrets.AgentConfig {
	config := DefaultGitSecretsConfig()
	
	// Add custom patterns for common secrets
	config.CustomPatterns = []string{
		// API Keys
		`[Aa][Pp][Ii]_?[Kk][Ee][Yy].*['|"][0-9a-zA-Z]{32,45}['|"]`,
		`[Aa][Pp][Ii][-_]?[Kk][Ee][Yy].*['|"][0-9a-zA-Z]{32,45}['|"]`,
		
		// Generic secrets
		`[Ss][Ee][Cc][Rr][Ee][Tt].*['|"][0-9a-zA-Z]{16,}['|"]`,
		`[Pp][Aa][Ss][Ss][Ww][Oo][Rr][Dd].*['|"][0-9a-zA-Z]{8,}['|"]`,
		
		// JWT tokens
		`eyJ[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*`,
		
		// Database URLs
		`[a-zA-Z][a-zA-Z0-9+.-]*://[a-zA-Z0-9._-]+:[a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+`,
	}
	
	// Common whitelist patterns for test/example files
	config.Whitelist = []string{
		`test/.*`,
		`tests/.*`,
		`.*test.*`,
		`examples?/.*`,
		`.*example.*`,
		`.*placeholder.*`,
		`.*dummy.*`,
		`.*fake.*`,
		`.*mock.*`,
		`docs?/.*`,
		`.*\.md$`,
		`.*\.txt$`,
	}
	
	return config
}