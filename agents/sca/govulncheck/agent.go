package govulncheck

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

const (
	AgentName    = "govulncheck"
	AgentVersion = "1.0.0"
	DefaultImage = "golang:1.21-alpine"
)

// Agent implements the SecurityAgent interface for govulncheck
type Agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for the govulncheck agent
type AgentConfig struct {
	DockerImage     string        `json:"docker_image"`
	MaxMemoryMB     int           `json:"max_memory_mb"`
	MaxCPUCores     float64       `json:"max_cpu_cores"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	GoVersion       string        `json:"go_version"`       // Go version to use
	VulnDBURL       string        `json:"vulndb_url"`       // Custom vulnerability database URL
	ShowTraces      bool          `json:"show_traces"`      // Show call traces
	TestPackages    bool          `json:"test_packages"`    // Include test packages
	Tags            []string      `json:"tags"`             // Build tags
	ExcludePatterns []string      `json:"exclude_patterns"` // Patterns to exclude
}

// NewAgent creates a new govulncheck agent with default configuration
func NewAgent() *Agent {
	return &Agent{
		config: AgentConfig{
			DockerImage:     DefaultImage,
			MaxMemoryMB:     512,
			MaxCPUCores:     1.0,
			DefaultTimeout:  5 * time.Minute,
			GoVersion:       "1.21",
			VulnDBURL:       "",           // Use default vulnerability database
			ShowTraces:      true,         // Show call traces for better context
			TestPackages:    false,        // Don't include test packages by default
			Tags:            []string{},   // No build tags by default
			ExcludePatterns: []string{},   // No exclusions by default
		},
	}
}

// NewAgentWithConfig creates a new govulncheck agent with custom configuration
func NewAgentWithConfig(config AgentConfig) *Agent {
	return &Agent{config: config}
}

// Scan executes the dependency analysis using govulncheck
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

	// Check if this is a Go project
	if !a.isGoProject(config.Languages) {
		result.Status = agent.ScanStatusCompleted
		result.Duration = time.Since(startTime)
		return result, nil // No findings for non-Go projects
	}

	// Execute govulncheck scan
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

// HealthCheck verifies that govulncheck is operational
func (a *Agent) HealthCheck(ctx context.Context) error {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	// Check if Go image is available
	cmd = exec.CommandContext(ctx, "docker", "image", "inspect", a.config.DockerImage)
	if err := cmd.Run(); err != nil {
		// Try to pull the image
		pullCmd := exec.CommandContext(ctx, "docker", "pull", a.config.DockerImage)
		if pullErr := pullCmd.Run(); pullErr != nil {
			return fmt.Errorf("go image not available and pull failed: %w", pullErr)
		}
	}

	return nil
}

// GetConfig returns agent configuration and capabilities
func (a *Agent) GetConfig() agent.AgentConfig {
	return agent.AgentConfig{
		Name:           AgentName,
		Version:        AgentVersion,
		SupportedLangs: []string{"go", "golang"},
		Categories: []agent.VulnCategory{
			agent.CategoryDependencyVuln,
			agent.CategoryOutdatedDeps,
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

// getToolVersion retrieves the govulncheck version
func (a *Agent) getToolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", a.config.DockerImage, "sh", "-c", "go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck -version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse version from output
	version := strings.TrimSpace(string(output))
	return version
}

// isGoProject checks if the project contains Go files
func (a *Agent) isGoProject(languages []string) bool {
	if len(languages) == 0 {
		return true // Assume it might be Go if no languages specified
	}
	
	goLanguages := map[string]bool{
		"go":     true,
		"golang": true,
	}
	
	for _, lang := range languages {
		if goLanguages[strings.ToLower(lang)] {
			return true
		}
	}
	
	return false
}