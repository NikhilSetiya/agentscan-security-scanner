package bandit

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
)

const (
	AgentName    = "bandit"
	AgentVersion = "1.0.0"
	DefaultImage = "python:3.11-slim"
)

// Agent implements the SecurityAgent interface for Bandit
type Agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for the Bandit agent
type AgentConfig struct {
	DockerImage     string        `json:"docker_image"`
	MaxMemoryMB     int           `json:"max_memory_mb"`
	MaxCPUCores     float64       `json:"max_cpu_cores"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	BanditConfig    string        `json:"bandit_config"`    // Path to custom Bandit config
	SkipTests       []string      `json:"skip_tests"`       // Test IDs to skip
	Severity        string        `json:"severity"`         // Minimum severity level (low, medium, high)
	Confidence      string        `json:"confidence"`       // Minimum confidence level (low, medium, high)
	ExcludePaths    []string      `json:"exclude_paths"`    // Paths to exclude from scanning
	PythonVersions  []string      `json:"python_versions"`  // Python versions to support
}

// NewAgent creates a new Bandit agent with default configuration
func NewAgent() *Agent {
	return &Agent{
		config: AgentConfig{
			DockerImage:    DefaultImage,
			MaxMemoryMB:    512,
			MaxCPUCores:    1.0,
			DefaultTimeout: 10 * time.Minute,
			BanditConfig:   "", // Use default configuration
			SkipTests:      []string{},
			Severity:       "low",    // Report all severity levels
			Confidence:     "low",    // Report all confidence levels
			ExcludePaths:   []string{"*/tests/*", "*/test/*", "*/.venv/*", "*/venv/*"},
			PythonVersions: []string{"3.8", "3.9", "3.10", "3.11", "3.12"},
		},
	}
}

// NewAgentWithConfig creates a new Bandit agent with custom configuration
func NewAgentWithConfig(config AgentConfig) *Agent {
	return &Agent{config: config}
}

// Scan executes the security analysis using Bandit
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

	// Check if this is a Python project
	if !a.isPythonProject(config.Languages) {
		result.Status = agent.ScanStatusCompleted
		result.Duration = time.Since(startTime)
		return result, nil // No findings for non-Python projects
	}

	// Execute Bandit scan
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

// HealthCheck verifies that Bandit is operational
func (a *Agent) HealthCheck(ctx context.Context) error {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	// Check if Python image is available
	cmd = exec.CommandContext(ctx, "docker", "image", "inspect", a.config.DockerImage)
	if err := cmd.Run(); err != nil {
		// Try to pull the image
		pullCmd := exec.CommandContext(ctx, "docker", "pull", a.config.DockerImage)
		if pullErr := pullCmd.Run(); pullErr != nil {
			return fmt.Errorf("python image not available and pull failed: %w", pullErr)
		}
	}

	return nil
}

// GetConfig returns agent configuration and capabilities
func (a *Agent) GetConfig() agent.AgentConfig {
	return agent.AgentConfig{
		Name:           AgentName,
		Version:        AgentVersion,
		SupportedLangs: []string{"python", "py"},
		Categories: []agent.VulnCategory{
			agent.CategorySQLInjection,
			agent.CategoryXSS,
			agent.CategoryCommandInjection,
			agent.CategoryPathTraversal,
			agent.CategoryInsecureCrypto,
			agent.CategoryHardcodedSecrets,
			agent.CategoryInsecureDeserialization,
			agent.CategoryMisconfiguration,
			agent.CategoryOther,
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

// getToolVersion retrieves the Bandit version
func (a *Agent) getToolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", a.config.DockerImage, "sh", "-c", "pip show bandit | grep Version || echo 'unknown'")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse version from output (e.g., "Version: 1.7.5")
	version := strings.TrimSpace(string(output))
	if strings.HasPrefix(version, "Version: ") {
		return strings.TrimPrefix(version, "Version: ")
	}
	return "unknown"
}

// isPythonProject checks if the project contains Python files
func (a *Agent) isPythonProject(languages []string) bool {
	if len(languages) == 0 {
		return true // Assume it might be Python if no languages specified
	}
	
	pythonLanguages := map[string]bool{
		"python": true,
		"py":     true,
		"python3": true,
	}
	
	for _, lang := range languages {
		if pythonLanguages[strings.ToLower(lang)] {
			return true
		}
	}
	
	return false
}