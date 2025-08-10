package findings

import (
	"time"

	"github.com/google/uuid"
)

// FindingStatus represents the current status of a finding
type FindingStatus string

const (
	FindingStatusOpen          FindingStatus = "open"
	FindingStatusFixed         FindingStatus = "fixed"
	FindingStatusIgnored       FindingStatus = "ignored"
	FindingStatusFalsePositive FindingStatus = "false_positive"
)

// Finding represents a security finding from a scan
type Finding struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	ScanResultID   uuid.UUID     `json:"scan_result_id" db:"scan_result_id"`
	ScanJobID      uuid.UUID     `json:"scan_job_id" db:"scan_job_id"`
	Tool           string        `json:"tool" db:"tool"`
	RuleID         string        `json:"rule_id" db:"rule_id"`
	Severity       string        `json:"severity" db:"severity"`
	Category       string        `json:"category" db:"category"`
	Title          string        `json:"title" db:"title"`
	Description    string        `json:"description" db:"description"`
	FilePath       string        `json:"file_path" db:"file_path"`
	LineNumber     *int          `json:"line_number" db:"line_number"`
	ColumnNumber   *int          `json:"column_number" db:"column_number"`
	CodeSnippet    *string       `json:"code_snippet" db:"code_snippet"`
	Confidence     float64       `json:"confidence" db:"confidence"`
	ConsensusScore *float64      `json:"consensus_score" db:"consensus_score"`
	Status         FindingStatus `json:"status" db:"status"`
	FixSuggestion  *string       `json:"fix_suggestion" db:"fix_suggestion"`
	References     []string      `json:"references" db:"references"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
}

// UserFeedback represents user feedback on a finding for ML training
type UserFeedback struct {
	ID        uuid.UUID     `json:"id" db:"id"`
	FindingID uuid.UUID     `json:"finding_id" db:"finding_id"`
	UserID    uuid.UUID     `json:"user_id" db:"user_id"`
	Action    FindingStatus `json:"action" db:"action"`
	Comment   *string       `json:"comment" db:"comment"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
}

// FindingFilter represents filters for querying findings
type FindingFilter struct {
	ScanJobID    *uuid.UUID      `json:"scan_job_id,omitempty"`
	Severity     []string        `json:"severity,omitempty"`
	Status       []FindingStatus `json:"status,omitempty"`
	Tool         []string        `json:"tool,omitempty"`
	FilePath     *string         `json:"file_path,omitempty"`
	MinConfidence *float64       `json:"min_confidence,omitempty"`
	Search       *string         `json:"search,omitempty"`
}

// FindingUpdate represents an update to a finding
type FindingUpdate struct {
	Status        *FindingStatus `json:"status,omitempty"`
	Comment       *string        `json:"comment,omitempty"`
	FixSuggestion *string        `json:"fix_suggestion,omitempty"`
}

// FindingSuppression represents a suppression rule for findings
type FindingSuppression struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	RuleID      string    `json:"rule_id" db:"rule_id"`
	FilePath    *string   `json:"file_path" db:"file_path"`
	LineNumber  *int      `json:"line_number" db:"line_number"`
	Reason      string    `json:"reason" db:"reason"`
	ExpiresAt   *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// FindingStats represents statistics about findings
type FindingStats struct {
	Total          int `json:"total"`
	High           int `json:"high"`
	Medium         int `json:"medium"`
	Low            int `json:"low"`
	Open           int `json:"open"`
	Fixed          int `json:"fixed"`
	Ignored        int `json:"ignored"`
	FalsePositives int `json:"false_positives"`
}

// ExportFormat represents the format for exporting findings
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatPDF  ExportFormat = "pdf"
	ExportFormatCSV  ExportFormat = "csv"
)

// ExportRequest represents a request to export findings
type ExportRequest struct {
	ScanJobID uuid.UUID     `json:"scan_job_id"`
	Format    ExportFormat  `json:"format"`
	Filter    FindingFilter `json:"filter"`
}

// ExportResult represents the result of an export operation
type ExportResult struct {
	ID          uuid.UUID `json:"id"`
	Format      ExportFormat `json:"format"`
	Filename    string    `json:"filename"`
	URL         string    `json:"url"`
	Size        int64     `json:"size"`
	GeneratedAt time.Time `json:"generated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}