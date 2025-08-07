package semgrep

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
)

const (
	AgentName    = "semgrep"
	AgentVersion = "1.0.0"
	DefaultImage = "returntocorp/semgrep:latest"
)

// Agent implements the SecurityAgent interface for Semgrep
type Agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for the Semgrep agent
type AgentConfig struct {
	DockerImage    string        `json:"docker_image"`
	MaxMemoryMB    int           `json:"max_memory_mb"`
	MaxCPUCores    float64       `json:"max_cpu_cores"`
	DefaultTimeout time.Duration `json:"default_timeout"`
	RulesConfig    string        `json:"rules_config"` // auto, p/security-audit, etc.
}

// NewAgent creates a new Semgrep agent with default configuration
func NewAgent() *Agent {
	return &Agent{
		config: AgentConfig{
			DockerImage:    DefaultImage,
			MaxMemoryMB:    512,
			MaxCPUCores:    1.0,
			DefaultTimeout: 10 * time.Minute,
			RulesConfig:    "auto", // Use Semgrep's auto-detection
		},
	}
}

// NewAgentWithConfig creates a new Semgrep agent with custom configuration
func NewAgentWithConfig(config AgentConfig) *Agent {
	return &Agent{config: config}
}

// Scan executes the security analysis using Semgrep
func (a *Agent) Scan(ctx context.Context, config agent.ScanConfig) (*agent.ScanResult, error) {
	startTime := time.Now()
	
	result := &agent.ScanResult{
		AgentID:  AgentName,
		Status:   agent.ScanStatusRunning,
		Findings: []agent.Finding{},
		Metadata: agent.Metadata{
			ScanType: "sast",
		},
	}

	// Apply timeout from config or use default
	timeout := config.Timeout
	if timeout == 0 {
		timeout = a.config.DefaultTimeout
	}
	
	scanCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute Semgrep scan
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

// HealthCheck verifies that Semgrep is operational
func (a *Agent) HealthCheck(ctx context.Context) error {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	// Check if Semgrep image is available
	cmd = exec.CommandContext(ctx, "docker", "image", "inspect", a.config.DockerImage)
	if err := cmd.Run(); err != nil {
		// Try to pull the image
		pullCmd := exec.CommandContext(ctx, "docker", "pull", a.config.DockerImage)
		if pullErr := pullCmd.Run(); pullErr != nil {
			return fmt.Errorf("semgrep image not available and pull failed: %w", pullErr)
		}
	}

	return nil
}

// GetConfig returns agent configuration and capabilities
func (a *Agent) GetConfig() agent.AgentConfig {
	return agent.AgentConfig{
		Name:           AgentName,
		Version:        AgentVersion,
		SupportedLangs: []string{"javascript", "typescript", "python", "go", "java", "c", "cpp", "ruby", "php", "scala", "kotlin", "rust"},
		Categories: []agent.VulnCategory{
			agent.CategorySQLInjection,
			agent.CategoryXSS,
			agent.CategoryCommandInjection,
			agent.CategoryPathTraversal,
			agent.CategoryInsecureCrypto,
			agent.CategoryHardcodedSecrets,
			agent.CategoryInsecureDeserialization,
			agent.CategoryAuthBypass,
			agent.CategoryMisconfiguration,
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

// getToolVersion retrieves the Semgrep version
func (a *Agent) getToolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", a.config.DockerImage, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse version from output (e.g., "1.45.0")
	version := strings.TrimSpace(string(output))
	return version
}