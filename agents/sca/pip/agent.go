package pip

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

const (
	AgentName    = "pip-audit"
	AgentVersion = "1.0.0"
	DefaultImage = "python:3.11-slim"
)

// Agent implements the SecurityAgent interface for pip-audit
type Agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for the pip-audit agent
type AgentConfig struct {
	DockerImage     string        `json:"docker_image"`
	MaxMemoryMB     int           `json:"max_memory_mb"`
	MaxCPUCores     float64       `json:"max_cpu_cores"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	RequirementsFiles []string    `json:"requirements_files"` // Files to scan
	Format          string        `json:"format"`             // Output format (json, cyclonedx, etc.)
	IndexURL        string        `json:"index_url"`          // Custom PyPI index
	ExtraIndexURLs  []string      `json:"extra_index_urls"`   // Additional PyPI indexes
	IgnoreVulns     []string      `json:"ignore_vulns"`       // Vulnerability IDs to ignore
	LocalPackages   bool          `json:"local_packages"`     // Include local packages
	PythonVersions  []string      `json:"python_versions"`    // Python versions to support
}

// NewAgent creates a new pip-audit agent with default configuration
func NewAgent() *Agent {
	return &Agent{
		config: AgentConfig{
			DockerImage:    DefaultImage,
			MaxMemoryMB:    512,
			MaxCPUCores:    1.0,
			DefaultTimeout: 5 * time.Minute,
			RequirementsFiles: []string{
				"requirements.txt",
				"requirements-dev.txt",
				"requirements-test.txt",
				"dev-requirements.txt",
				"test-requirements.txt",
			},
			Format:         "json",
			IndexURL:       "",           // Use default PyPI
			ExtraIndexURLs: []string{},   // No extra indexes
			IgnoreVulns:    []string{},   // No ignored vulnerabilities
			LocalPackages:  false,        // Don't include local packages
			PythonVersions: []string{"3.8", "3.9", "3.10", "3.11", "3.12"},
		},
	}
}

// NewAgentWithConfig creates a new pip-audit agent with custom configuration
func NewAgentWithConfig(config AgentConfig) *Agent {
	return &Agent{config: config}
}

// Scan executes the dependency analysis using pip-audit
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

	// Check if this is a Python project
	if !a.isPythonProject(config.Languages) {
		result.Status = agent.ScanStatusCompleted
		result.Duration = time.Since(startTime)
		return result, nil // No findings for non-Python projects
	}

	// Execute pip-audit scan
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

// HealthCheck verifies that pip-audit is operational
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
			agent.CategoryDependencyVuln,
			agent.CategoryOutdatedDeps,
			agent.CategorySupplyChain,
			agent.CategoryLicenseIssue,
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

// getToolVersion retrieves the pip-audit version
func (a *Agent) getToolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", a.config.DockerImage, "sh", "-c", "pip install pip-audit && pip-audit --version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse version from output (e.g., "pip-audit 2.6.1")
	version := strings.TrimSpace(string(output))
	if strings.HasPrefix(version, "pip-audit ") {
		return strings.TrimPrefix(version, "pip-audit ")
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