package agent

import (
	"context"
	"time"
)

// SecurityAgent defines the interface that all security scanning agents must implement
type SecurityAgent interface {
	// Scan executes the security analysis
	Scan(ctx context.Context, config ScanConfig) (*ScanResult, error)

	// HealthCheck verifies agent is operational
	HealthCheck(ctx context.Context) error

	// GetConfig returns agent configuration and capabilities
	GetConfig() AgentConfig

	// GetVersion returns agent and tool version information
	GetVersion() VersionInfo
}

// ScanConfig contains the configuration for a security scan
type ScanConfig struct {
	RepoURL   string            `json:"repo_url"`
	Branch    string            `json:"branch"`
	Commit    string            `json:"commit"`
	Languages []string          `json:"languages"`
	Files     []string          `json:"files,omitempty"`     // For incremental scans
	Rules     []string          `json:"rules,omitempty"`     // Custom rules to apply
	Options   map[string]string `json:"options,omitempty"`   // Agent-specific options
	Timeout   time.Duration     `json:"timeout"`             // Maximum execution time
}

// ScanResult contains the results of a security scan
type ScanResult struct {
	AgentID  string        `json:"agent_id"`
	Status   ScanStatus    `json:"status"`
	Findings []Finding     `json:"findings"`
	Metadata Metadata      `json:"metadata"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// ScanStatus represents the status of a scan
type ScanStatus string

const (
	ScanStatusQueued     ScanStatus = "queued"
	ScanStatusRunning    ScanStatus = "running"
	ScanStatusCompleted  ScanStatus = "completed"
	ScanStatusFailed     ScanStatus = "failed"
	ScanStatusCancelled  ScanStatus = "cancelled"
	ScanStatusTimedOut   ScanStatus = "timed_out"
)

// Finding represents a security vulnerability or issue found by an agent
type Finding struct {
	ID          string         `json:"id"`
	Tool        string         `json:"tool"`
	RuleID      string         `json:"rule_id"`
	Severity    Severity       `json:"severity"`
	Category    VulnCategory   `json:"category"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	File        string         `json:"file"`
	Line        int            `json:"line"`
	Column      int            `json:"column,omitempty"`
	Code        string         `json:"code,omitempty"`
	Fix         *FixSuggestion `json:"fix,omitempty"`
	Confidence  float64        `json:"confidence"`
	References  []string       `json:"references,omitempty"`
}

// Severity represents the severity level of a finding
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
	SeverityInfo   Severity = "info"
)

// VulnCategory represents the category of vulnerability
type VulnCategory string

const (
	CategorySQLInjection      VulnCategory = "sql_injection"
	CategoryXSS               VulnCategory = "xss"
	CategoryCSRF              VulnCategory = "csrf"
	CategoryAuthBypass        VulnCategory = "auth_bypass"
	CategoryInsecureCrypto    VulnCategory = "insecure_crypto"
	CategoryHardcodedSecrets  VulnCategory = "hardcoded_secrets"
	CategoryPathTraversal     VulnCategory = "path_traversal"
	CategoryCommandInjection  VulnCategory = "command_injection"
	CategoryInsecureDeserialization VulnCategory = "insecure_deserialization"
	CategoryVulnerableDependency    VulnCategory = "vulnerable_dependency"
	CategoryMisconfiguration  VulnCategory = "misconfiguration"
	CategoryOther             VulnCategory = "other"
)

// FixSuggestion contains suggested fixes for a finding
type FixSuggestion struct {
	Description string `json:"description"`
	Code        string `json:"code,omitempty"`
	References  []string `json:"references,omitempty"`
}

// Metadata contains additional information about the scan
type Metadata struct {
	ToolVersion   string            `json:"tool_version"`
	RulesVersion  string            `json:"rules_version"`
	ScanType      string            `json:"scan_type"`
	FilesScanned  int               `json:"files_scanned"`
	LinesScanned  int               `json:"lines_scanned"`
	ExitCode      int               `json:"exit_code"`
	CommandLine   string            `json:"command_line,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
}

// AgentConfig contains configuration and capabilities of an agent
type AgentConfig struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	SupportedLangs  []string `json:"supported_languages"`
	Categories      []VulnCategory `json:"categories"`
	RequiresDocker  bool     `json:"requires_docker"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	MaxMemoryMB     int      `json:"max_memory_mb"`
	MaxCPUCores     float64  `json:"max_cpu_cores"`
}

// VersionInfo contains version information for the agent and underlying tool
type VersionInfo struct {
	AgentVersion string `json:"agent_version"`
	ToolVersion  string `json:"tool_version"`
	BuildDate    string `json:"build_date"`
	GitCommit    string `json:"git_commit"`
}