package orchestrator

import (
	"time"

	"github.com/google/uuid"
)

// ScanRequest represents a request to start a new scan
type ScanRequest struct {
	RepositoryID uuid.UUID              `json:"repository_id"`
	UserID       *uuid.UUID             `json:"user_id,omitempty"`
	RepoURL      string                 `json:"repo_url"`
	Branch       string                 `json:"branch"`
	CommitSHA    string                 `json:"commit_sha"`
	BaseSHA      string                 `json:"base_sha,omitempty"`      // For PR scans - base commit to compare against
	ChangedFiles []string               `json:"changed_files,omitempty"` // For incremental scans - specific files to scan
	ScanType     string                 `json:"scan_type"`               // full, incremental, ide
	Priority     int                    `json:"priority"`                // 1-10, higher = more priority
	Agents       []string               `json:"agents"`                  // List of agent names to run
	Options      map[string]interface{} `json:"options,omitempty"`
	Timeout      time.Duration          `json:"timeout,omitempty"`
	CallbackURL  string                 `json:"callback_url,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ScanStatus represents the current status of a scan
type ScanStatus struct {
	JobID           string        `json:"job_id"`
	Status          string        `json:"status"`
	Progress        float64       `json:"progress"` // 0-100
	StartedAt       *time.Time    `json:"started_at,omitempty"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	Duration        time.Duration `json:"duration"`
	AgentsRequested []string      `json:"agents_requested"`
	AgentsCompleted []string      `json:"agents_completed"`
	ErrorMessage    string        `json:"error_message,omitempty"`
	Results         []AgentResult `json:"results"`
}

// AgentResult represents the result from a single agent
type AgentResult struct {
	AgentName     string        `json:"agent_name"`
	Status        string        `json:"status"`
	FindingsCount int           `json:"findings_count"`
	Duration      time.Duration `json:"duration"`
	ErrorMessage  string        `json:"error_message,omitempty"`
}

// ScanResults represents the complete results of a scan
type ScanResults struct {
	JobID        string                 `json:"job_id"`
	Status       string                 `json:"status"`
	Repository   string                 `json:"repository"`
	Branch       string                 `json:"branch"`
	CommitSHA    string                 `json:"commit_sha"`
	ScanType     string                 `json:"scan_type"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Findings     []Finding              `json:"findings"`
	Summary      ResultSummary          `json:"summary"`
	AgentResults []AgentResult          `json:"agent_results"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Finding represents a security finding
type Finding struct {
	ID            string                 `json:"id"`
	Tool          string                 `json:"tool"`
	RuleID        string                 `json:"rule_id"`
	Severity      string                 `json:"severity"`
	Category      string                 `json:"category"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	File          string                 `json:"file"`
	Line          int                    `json:"line"`
	Column        int                    `json:"column,omitempty"`
	Code          string                 `json:"code,omitempty"`
	Confidence    float64                `json:"confidence"`
	Status        string                 `json:"status"`
	FixSuggestion map[string]interface{} `json:"fix_suggestion,omitempty"`
	References    []string               `json:"references,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"` // Additional metadata like is_new_in_pr
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ResultSummary provides summary statistics for scan results
type ResultSummary struct {
	TotalFindings int            `json:"total_findings"`
	BySeverity    map[string]int `json:"by_severity"`
	ByTool        map[string]int `json:"by_tool"`
	ByCategory    map[string]int `json:"by_category"`
}

// ResultFilter represents filters for scan results
type ResultFilter struct {
	Severity string `json:"severity,omitempty"`
	Tool     string `json:"tool,omitempty"`
	Category string `json:"category,omitempty"`
	Status   string `json:"status,omitempty"`
	File     string `json:"file,omitempty"`
}

// ScanFilter represents filters for listing scans
type ScanFilter struct {
	RepositoryID *uuid.UUID `json:"repository_id,omitempty"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	Status       string     `json:"status,omitempty"`
	ScanType     string     `json:"scan_type,omitempty"`
	Since        time.Time  `json:"since,omitempty"`
	Until        time.Time  `json:"until,omitempty"`
}

// Pagination represents pagination parameters
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// ScanList represents a paginated list of scans
type ScanList struct {
	Scans      []ScanSummary `json:"scans"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int64         `json:"total_pages"`
}

// ScanSummary represents a summary of a scan for listing
type ScanSummary struct {
	JobID       string        `json:"job_id"`
	Repository  string        `json:"repository"`
	Branch      string        `json:"branch"`
	CommitSHA   string        `json:"commit_sha"`
	ScanType    string        `json:"scan_type"`
	Status      string        `json:"status"`
	Priority    int           `json:"priority"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration"`
	CreatedAt   time.Time     `json:"created_at"`
}

// WorkerStats represents statistics for a worker
type WorkerStats struct {
	WorkerID      string        `json:"worker_id"`
	Status        string        `json:"status"`
	JobsProcessed int64         `json:"jobs_processed"`
	JobsFailed    int64         `json:"jobs_failed"`
	LastJobAt     *time.Time    `json:"last_job_at,omitempty"`
	Uptime        time.Duration `json:"uptime"`
}

// ServiceStats represents statistics for the orchestration service
type ServiceStats struct {
	Status         string        `json:"status"`
	WorkerCount    int           `json:"worker_count"`
	ActiveScans    int64         `json:"active_scans"`
	QueuedScans    int64         `json:"queued_scans"`
	CompletedScans int64         `json:"completed_scans"`
	FailedScans    int64         `json:"failed_scans"`
	Uptime         time.Duration `json:"uptime"`
	Workers        []WorkerStats `json:"workers"`
}