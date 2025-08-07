package types

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	AvatarURL string     `json:"avatar_url" db:"avatar_url"`
	GitHubID  *int       `json:"github_id,omitempty" db:"github_id"`
	GitLabID  *int       `json:"gitlab_id,omitempty" db:"gitlab_id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// Organization represents an organization/team
type Organization struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	Name      string                 `json:"name" db:"name"`
	Slug      string                 `json:"slug" db:"slug"`
	Settings  map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// OrganizationMember represents a user's membership in an organization
type OrganizationMember struct {
	ID             uuid.UUID `json:"id" db:"id"`
	OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Role           string    `json:"role" db:"role"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// Repository represents a code repository
type Repository struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	Name           string                 `json:"name" db:"name"`
	URL            string                 `json:"url" db:"url"`
	Provider       string                 `json:"provider" db:"provider"`
	ProviderID     string                 `json:"provider_id" db:"provider_id"`
	DefaultBranch  string                 `json:"default_branch" db:"default_branch"`
	Languages      []string               `json:"languages" db:"languages"`
	Settings       map[string]interface{} `json:"settings" db:"settings"`
	LastScanAt     *time.Time             `json:"last_scan_at,omitempty" db:"last_scan_at"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// ScanJob represents a security scan job
type ScanJob struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	RepositoryID     uuid.UUID              `json:"repository_id" db:"repository_id"`
	UserID           *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	Branch           string                 `json:"branch" db:"branch"`
	CommitSHA        string                 `json:"commit_sha" db:"commit_sha"`
	ScanType         string                 `json:"scan_type" db:"scan_type"`
	Priority         int                    `json:"priority" db:"priority"`
	Status           string                 `json:"status" db:"status"`
	AgentsRequested  []string               `json:"agents_requested" db:"agents_requested"`
	AgentsCompleted  []string               `json:"agents_completed" db:"agents_completed"`
	StartedAt        *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage     string                 `json:"error_message,omitempty" db:"error_message"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

// ScanResult represents the result from a single agent
type ScanResult struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	ScanJobID     uuid.UUID              `json:"scan_job_id" db:"scan_job_id"`
	AgentName     string                 `json:"agent_name" db:"agent_name"`
	Status        string                 `json:"status" db:"status"`
	FindingsCount int                    `json:"findings_count" db:"findings_count"`
	DurationMS    int                    `json:"duration_ms" db:"duration_ms"`
	ErrorMessage  string                 `json:"error_message,omitempty" db:"error_message"`
	RawOutput     map[string]interface{} `json:"raw_output" db:"raw_output"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// Finding represents a security finding
type Finding struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	ScanResultID   uuid.UUID              `json:"scan_result_id" db:"scan_result_id"`
	ScanJobID      uuid.UUID              `json:"scan_job_id" db:"scan_job_id"`
	Tool           string                 `json:"tool" db:"tool"`
	RuleID         string                 `json:"rule_id" db:"rule_id"`
	Severity       string                 `json:"severity" db:"severity"`
	Category       string                 `json:"category" db:"category"`
	Title          string                 `json:"title" db:"title"`
	Description    string                 `json:"description" db:"description"`
	FilePath       string                 `json:"file_path" db:"file_path"`
	LineNumber     int                    `json:"line_number" db:"line_number"`
	ColumnNumber   int                    `json:"column_number,omitempty" db:"column_number"`
	CodeSnippet    string                 `json:"code_snippet,omitempty" db:"code_snippet"`
	Confidence     float64                `json:"confidence" db:"confidence"`
	ConsensusScore *float64               `json:"consensus_score,omitempty" db:"consensus_score"`
	Status         string                 `json:"status" db:"status"`
	FixSuggestion  map[string]interface{} `json:"fix_suggestion,omitempty" db:"fix_suggestion"`
	References     []string               `json:"references" db:"references"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// UserFeedback represents user feedback on findings
type UserFeedback struct {
	ID        uuid.UUID `json:"id" db:"id"`
	FindingID uuid.UUID `json:"finding_id" db:"finding_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Action    string    `json:"action" db:"action"`
	Comment   string    `json:"comment,omitempty" db:"comment"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Priority levels for scan jobs
const (
	PriorityLow    = 1
	PriorityMedium = 5
	PriorityHigh   = 10
)

// Scan types
const (
	ScanTypeFull        = "full"
	ScanTypeIncremental = "incremental"
	ScanTypeIDE         = "ide"
)

// Scan job statuses
const (
	ScanJobStatusQueued     = "queued"
	ScanJobStatusRunning    = "running"
	ScanJobStatusCompleted  = "completed"
	ScanJobStatusFailed     = "failed"
	ScanJobStatusCancelled  = "cancelled"
)

// Finding statuses
const (
	FindingStatusOpen         = "open"
	FindingStatusFixed        = "fixed"
	FindingStatusIgnored      = "ignored"
	FindingStatusFalsePositive = "false_positive"
)

// User feedback actions
const (
	FeedbackActionFixed        = "fixed"
	FeedbackActionIgnored      = "ignored"
	FeedbackActionFalsePositive = "false_positive"
	FeedbackActionConfirmed    = "confirmed"
)

// Organization member roles
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)