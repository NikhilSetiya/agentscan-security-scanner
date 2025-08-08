package npm

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
)

const (
	AgentName    = "npm-audit"
	AgentVersion = "1.0.0"
	DefaultImage = "node:18-alpine"
)

// Agent implements the SecurityAgent interface for npm audit
type Agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for the npm audit agent
type AgentConfig struct {
	DockerImage     string        `json:"docker_image"`
	MaxMemoryMB     int           `json:"max_memory_mb"`
	MaxCPUCores     float64       `json:"max_cpu_cores"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	AuditLevel      string        `json:"audit_level"`      // low, moderate, high, critical
	ProductionOnly  bool          `json:"production_only"`  // Only audit production dependencies
	IncludeDevDeps  bool          `json:"include_dev_deps"` // Include dev dependencies
	RegistryURL     string        `json:"registry_url"`     // Custom npm registry
	ExcludePackages []string      `json:"exclude_packages"` // Packages to exclude from audit
}

// NewAgent creates a new npm audit agent with default configuration
func NewAgent() *Agent {
	return &Agent{
		config: AgentConfig{
			DockerImage:     DefaultImage,
			MaxMemoryMB:     512,
			MaxCPUCores:     1.0,
			DefaultTimeout:  5 * time.Minute,
			AuditLevel:      "low",        // Report all severity levels
			ProductionOnly:  false,        // Include all dependencies
			IncludeDevDeps:  true,         // Include dev dependencies
			RegistryURL:     "",           // Use default registry
			ExcludePackages: []string{},   // No exclusions by default
		},
	}
}

// NewAgentWithConfig creates a new npm audit agent with custom configuration
func NewAgentWithConfig(config AgentConfig) *Agent {
	return &Agent{config: config}
}

// Scan executes the dependency analysis using npm audit
func (a *Agent) Scan(ctx context.Context, config agent.ScanConfig) (*agent.ScanResult, error) {
	startTime := time.Now()
	
	result := &agent.ScanResult{
		AgentID:  AgentName,
		Status:   agent.ScanStatusRunning,
		Findings: []agent.Finding{},
		Metadata: agent.Metadata{
			ScanType: "sca",
		},
	}

	// Apply timeout from config or use default
	timeout := config.Timeout
	if timeout == 0 {
		timeout = a.config.DefaultTimeout
	}
	
	scanCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check if this is a Node.js project
	if !a.isNodeProject(config.Languages) {
		result.Status = agent.ScanStatusCompleted
		result.Duration = time.Since(startTime)
		return result, nil // No findings for non-Node.js projects
	}

	// Execute npm audit scan
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

// HealthCheck verifies that npm audit is operational
func (a *Agent) HealthCheck(ctx context.Context) error {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	// Check if Node.js image is available
	cmd = exec.CommandContext(ctx, "docker", "image", "inspect", a.config.DockerImage)
	if err := cmd.Run(); err != nil {
		// Try to pull the image
		pullCmd := exec.CommandContext(ctx, "docker", "pull", a.config.DockerImage)
		if pullErr := pullCmd.Run(); pullErr != nil {
			return fmt.Errorf("node.js image not available and pull failed: %w", pullErr)
		}
	}

	return nil
}

// GetConfig returns agent configuration and capabilities
func (a *Agent) GetConfig() agent.AgentConfig {
	return agent.AgentConfig{
		Name:           AgentName,
		Version:        AgentVersion,
		SupportedLangs: []string{"javascript", "typescript", "node", "nodejs"},
		Categories: []agent.VulnCategory{
			agent.CategoryDependencyVuln,
			agent.CategoryOutdatedDeps,
			agent.CategoryLicenseIssue,
			agent.CategorySupplyChain,
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

// getToolVersion retrieves the npm version
func (a *Agent) getToolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", a.config.DockerImage, "npm", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// isNodeProject checks if the project contains Node.js files
func (a *Agent) isNodeProject(languages []string) bool {
	if len(languages) == 0 {
		return true // Assume it might be Node.js if no languages specified
	}
	
	nodeLanguages := map[string]bool{
		"javascript":  true,
		"typescript":  true,
		"node":        true,
		"nodejs":      true,
		"js":          true,
		"ts":          true,
	}
	
	for _, lang := range languages {
		if nodeLanguages[strings.ToLower(lang)] {
			return true
		}
	}
	
	return false
}