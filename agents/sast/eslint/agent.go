package eslint

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

const (
	AgentName    = "eslint-security"
	AgentVersion = "1.0.0"
	DefaultImage = "node:18-alpine"
)

// Agent implements the SecurityAgent interface for ESLint Security
type Agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for the ESLint Security agent
type AgentConfig struct {
	DockerImage    string        `json:"docker_image"`
	MaxMemoryMB    int           `json:"max_memory_mb"`
	MaxCPUCores    float64       `json:"max_cpu_cores"`
	DefaultTimeout time.Duration `json:"default_timeout"`
	ESLintConfig   string        `json:"eslint_config"` // Path to custom ESLint config
	SecurityRules  []string      `json:"security_rules"` // Specific security rules to enable
}

// NewAgent creates a new ESLint Security agent with default configuration
func NewAgent() *Agent {
	return &Agent{
		config: AgentConfig{
			DockerImage:    DefaultImage,
			MaxMemoryMB:    512,
			MaxCPUCores:    1.0,
			DefaultTimeout: 5 * time.Minute,
			ESLintConfig:   "", // Use default security configuration
			SecurityRules: []string{
				"security/detect-buffer-noassert",
				"security/detect-child-process",
				"security/detect-disable-mustache-escape",
				"security/detect-eval-with-expression",
				"security/detect-no-csrf-before-method-override",
				"security/detect-non-literal-fs-filename",
				"security/detect-non-literal-regexp",
				"security/detect-non-literal-require",
				"security/detect-object-injection",
				"security/detect-possible-timing-attacks",
				"security/detect-pseudoRandomBytes",
				"security/detect-unsafe-regex",
			},
		},
	}
}

// NewAgentWithConfig creates a new ESLint Security agent with custom configuration
func NewAgentWithConfig(config AgentConfig) *Agent {
	return &Agent{config: config}
}

// Scan executes the security analysis using ESLint Security
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

	// Check if this is a JavaScript/TypeScript project
	if !a.isJavaScriptProject(config.Languages) {
		result.Status = agent.ScanStatusCompleted
		result.Duration = time.Since(startTime)
		return result, nil // No findings for non-JS projects
	}

	// Execute ESLint scan
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

// HealthCheck verifies that ESLint is operational
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
		SupportedLangs: []string{"javascript", "typescript", "jsx", "tsx"},
		Categories: []agent.VulnCategory{
			agent.CategoryXSS,
			agent.CategoryCommandInjection,
			agent.CategoryPathTraversal,
			agent.CategoryInsecureCrypto,
			agent.CategoryCSRF,
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

// getToolVersion retrieves the ESLint version
func (a *Agent) getToolVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", a.config.DockerImage, "sh", "-c", "npm list -g eslint --depth=0 2>/dev/null | grep eslint || echo 'unknown'")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse version from output
	version := strings.TrimSpace(string(output))
	if strings.Contains(version, "@") {
		parts := strings.Split(version, "@")
		if len(parts) > 1 {
			return parts[1]
		}
	}
	return "unknown"
}

// isJavaScriptProject checks if the project contains JavaScript/TypeScript files
func (a *Agent) isJavaScriptProject(languages []string) bool {
	if len(languages) == 0 {
		return true // Assume it might be JS if no languages specified
	}
	
	jsLanguages := map[string]bool{
		"javascript": true,
		"typescript": true,
		"jsx":        true,
		"tsx":        true,
		"js":         true,
		"ts":         true,
	}
	
	for _, lang := range languages {
		if jsLanguages[strings.ToLower(lang)] {
			return true
		}
	}
	
	return false
}