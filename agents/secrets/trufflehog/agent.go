package trufflehog

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

const (
	AgentName    = "trufflehog"
	AgentVersion = "1.0.0"
	DefaultImage = "trufflesecurity/trufflehog:latest"
)

// Agent implements the SecurityAgent interface for TruffleHog
type Agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for the TruffleHog agent
type AgentConfig struct {
	DockerImage     string        `json:"docker_image"`
	MaxMemoryMB     int           `json:"max_memory_mb"`
	MaxCPUCores     float64       `json:"max_cpu_cores"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	ScanGitHistory  bool          `json:"scan_git_history"`
	MaxDepth        int           `json:"max_depth"`        // Git history depth
	IncludeDetectors []string     `json:"include_detectors,omitempty"`
	ExcludeDetectors []string     `json:"exclude_detectors,omitempty"`
	Whitelist       []string      `json:"whitelist,omitempty"` // Patterns to ignore
}

// NewAgent creates a new TruffleHog agent with default configuration
func NewAgent() *Agent {
	return &Agent{
		config: AgentConfig{
			DockerImage:    DefaultImage,
			MaxMemoryMB:    1024,
			MaxCPUCores:    2.0,
			DefaultTimeout: 15 * time.Minute,
			ScanGitHistory: true,
			MaxDepth:       100, // Scan last 100 commits
			Whitelist:      []string{},
		},
	}
}

// NewAgentWithConfig creates a new TruffleHog agent with custom configuration
func NewAgentWithConfig(config AgentConfig) *Agent {
	return &Agent{config: config}
}

// Scan executes the secret scanning using TruffleHog
func (a *Agent) Scan(ctx context.Context, config agent.ScanConfig) (*agent.ScanResult, error) {
	startTime := time.Now()
	
	result := &agent.ScanResult{
		AgentID:  AgentName,
		Status:   agent.ScanStatusRunning,
		Findings: []agent.Finding{},
		Metadata: agent.Metadata{
			ScanType: "secrets",
		},
	}

	// Apply timeout from config or use default
	timeout := config.Timeout
	if timeout == 0 {
		timeout = a.config.DefaultTimeout
	}
	
	scanCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute TruffleHog scan
	findings, metadata, err := a.executeScan(scanCtx, config)
	if err != nil {
		result.Status = agent.ScanStatusFailed
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.Status = agent.ScanStatusCompleted
	result.Findings = findings
	result.Metadata = metadata
	result.Duration = time.Since(startTime)

	return result, nil
}

// HealthCheck verifies that TruffleHog is operational
func (a *Agent) HealthCheck(ctx context.Context) error {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	// Check if TruffleHog image is available
	cmd = exec.CommandContext(ctx, "docker", "image", "inspect", a.config.DockerImage)
	if err := cmd.Run(); err != nil {
		// Try to pull the image
		pullCmd := exec.CommandContext(ctx, "docker", "pull", a.config.DockerImage)
		if pullErr := pullCmd.Run(); pullErr != nil {
			return fmt.Errorf("trufflehog image not available and pull failed: %w", pullErr)
		}
	}

	return nil
}

// GetConfig returns agent configuration and capabilities
func (a *Agent) GetConfig() agent.AgentConfig {
	return agent.AgentConfig{
		Name:           AgentName,
		Version:        AgentVersion,
		SupportedLangs: []string{"*"}, // Language agnostic
		Categories: []agent.VulnCategory{
			agent.CategoryHardcodedSecrets,
		},
		RequiresDocker:  true,
		DefaultTimeout:  a.config.DefaultTimeout,
		MaxMemoryMB:     a.config.MaxMemoryMB,
		MaxCPUCores:     a.config.MaxCPUCores,
	}
}

// GetVersion returns agent and tool version information
func (a *Agent) GetVersion() agent.VersionInfo {
	return agent.VersionInfo{
		AgentVersion: AgentVersion,
		ToolVersion:  a.getToolVersion(),
		BuildDate:    time.Now().Format("2006-01-02"),
		GitCommit:    "unknown", // This would be set during build
	}
}

// getToolVersion retrieves the TruffleHog version
func (a *Agent) getToolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", a.config.DockerImage, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse version from output
	version := strings.TrimSpace(string(output))
	return version
}