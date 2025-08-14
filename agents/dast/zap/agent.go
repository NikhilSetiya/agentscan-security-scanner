package zap

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
)

const (
	AgentName    = "zap"
	AgentVersion = "1.0.0"
	DefaultImage = "owasp/zap2docker-stable:latest"
)

// Agent implements the SecurityAgent interface for OWASP ZAP
type Agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for the ZAP agent
type AgentConfig struct {
	DockerImage    string        `json:"docker_image"`
	MaxMemoryMB    int           `json:"max_memory_mb"`
	MaxCPUCores    float64       `json:"max_cpu_cores"`
	DefaultTimeout time.Duration `json:"default_timeout"`
	ScanType       string        `json:"scan_type"` // baseline, full, api
	MaxDepth       int           `json:"max_depth"` // Maximum crawl depth
}

// NewAgent creates a new ZAP agent with default configuration
func NewAgent() *Agent {
	return &Agent{
		config: AgentConfig{
			DockerImage:    DefaultImage,
			MaxMemoryMB:    1024, // ZAP needs more memory than SAST tools
			MaxCPUCores:    1.0,
			DefaultTimeout: 10 * time.Minute,
			ScanType:       "baseline", // Start with baseline scan
			MaxDepth:       5,          // Reasonable crawl depth
		},
	}
}

// NewAgentWithConfig creates a new ZAP agent with custom configuration
func NewAgentWithConfig(config AgentConfig) *Agent {
	return &Agent{config: config}
}

// Scan executes the dynamic security analysis using OWASP ZAP
func (a *Agent) Scan(ctx context.Context, config agent.ScanConfig) (*agent.ScanResult, error) {
	startTime := time.Now()
	
	result := &agent.ScanResult{
		AgentID:  AgentName,
		Status:   agent.ScanStatusRunning,
		Findings: []agent.Finding{},
		Metadata: agent.Metadata{
			ScanType: "dast",
		},
	}

	// Apply timeout from config or use default
	timeout := config.Timeout
	if timeout == 0 {
		timeout = a.config.DefaultTimeout
	}
	
	scanCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Detect if this is a web application
	webAppConfig, err := a.detectWebApplication(scanCtx, config)
	if err != nil {
		result.Status = agent.ScanStatusFailed
		result.Error = fmt.Sprintf("web application detection failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	if webAppConfig == nil {
		// Not a web application, skip DAST scanning
		result.Status = agent.ScanStatusCompleted
		result.Metadata = agent.Metadata{
			ScanType:     "dast",
			FilesScanned: 0,
			ExitCode:     0,
		}
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Start the web application
	runningApp, err := a.startApplication(scanCtx, config, webAppConfig)
	if err != nil {
		result.Status = agent.ScanStatusFailed
		result.Error = fmt.Sprintf("failed to start application: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}
	defer a.cleanupApplication(runningApp)

	// Wait for application to be ready
	if err := a.waitForApplication(scanCtx, runningApp); err != nil {
		result.Status = agent.ScanStatusFailed
		result.Error = fmt.Sprintf("application failed to start: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Execute ZAP scan
	findings, metadata, err := a.executeScan(scanCtx, config, runningApp)
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

// HealthCheck verifies that ZAP is operational
func (a *Agent) HealthCheck(ctx context.Context) error {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	// Check if ZAP image is available
	cmd = exec.CommandContext(ctx, "docker", "image", "inspect", a.config.DockerImage)
	if err := cmd.Run(); err != nil {
		// Try to pull the image
		pullCmd := exec.CommandContext(ctx, "docker", "pull", a.config.DockerImage)
		if pullErr := pullCmd.Run(); pullErr != nil {
			return fmt.Errorf("zap image not available and pull failed: %w", pullErr)
		}
	}

	return nil
}

// GetConfig returns agent configuration and capabilities
func (a *Agent) GetConfig() agent.AgentConfig {
	return agent.AgentConfig{
		Name:           AgentName,
		Version:        AgentVersion,
		SupportedLangs: []string{"javascript", "typescript", "python", "java", "php", "ruby", "go", "csharp"}, // Web frameworks
		Categories: []agent.VulnCategory{
			agent.CategoryXSS,
			agent.CategorySQLInjection,
			agent.CategoryCSRF,
			agent.CategoryAuthBypass,
			agent.CategoryCommandInjection,
			agent.CategoryPathTraversal,
			agent.CategoryInsecureDeserialization,
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

// getToolVersion retrieves the ZAP version
func (a *Agent) getToolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", a.config.DockerImage, "zap-baseline.py", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse version from output
	version := strings.TrimSpace(string(output))
	return version
}