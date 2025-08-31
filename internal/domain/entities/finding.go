package entities

import (
	"time"

	"github.com/google/uuid"
)

// FindingSeverity represents the severity level of a finding
type FindingSeverity string

const (
	FindingSeverityCritical FindingSeverity = "critical"
	FindingSeverityHigh     FindingSeverity = "high"
	FindingSeverityMedium   FindingSeverity = "medium"
	FindingSeverityLow      FindingSeverity = "low"
	FindingSeverityInfo     FindingSeverity = "info"
)

// FindingStatus represents the status of a finding
type FindingStatus string

const (
	FindingStatusOpen          FindingStatus = "open"
	FindingStatusFixed         FindingStatus = "fixed"
	FindingStatusIgnored       FindingStatus = "ignored"
	FindingStatusFalsePositive FindingStatus = "false_positive"
)

// Finding represents a security finding entity in the domain
type Finding struct {
	ID             uuid.UUID              `json:"id"`
	ScanJobID      uuid.UUID              `json:"scan_job_id"`
	Tool           string                 `json:"tool" validate:"required"`
	RuleID         string                 `json:"rule_id" validate:"required"`
	Severity       FindingSeverity        `json:"severity" validate:"required"`
	Category       string                 `json:"category" validate:"required"`
	Title          string                 `json:"title" validate:"required"`
	Description    string                 `json:"description" validate:"required"`
	FilePath       string                 `json:"file_path" validate:"required"`
	LineNumber     int                    `json:"line_number" validate:"min=1"`
	ColumnNumber   int                    `json:"column_number,omitempty"`
	CodeSnippet    string                 `json:"code_snippet,omitempty"`
	Confidence     float64                `json:"confidence" validate:"min=0,max=1"`
	ConsensusScore *float64               `json:"consensus_score,omitempty"`
	Status         FindingStatus          `json:"status" validate:"required"`
	FixSuggestion  map[string]interface{} `json:"fix_suggestion,omitempty"`
	References     []string               `json:"references,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// NewFinding creates a new finding entity
func NewFinding(scanJobID uuid.UUID, tool, ruleID string, severity FindingSeverity, category, title, description, filePath string, lineNumber int, confidence float64) *Finding {
	return &Finding{
		ID:           uuid.New(),
		ScanJobID:    scanJobID,
		Tool:         tool,
		RuleID:       ruleID,
		Severity:     severity,
		Category:     category,
		Title:        title,
		Description:  description,
		FilePath:     filePath,
		LineNumber:   lineNumber,
		Confidence:   confidence,
		Status:       FindingStatusOpen,
		References:   []string{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// SetCodeSnippet sets the code snippet for the finding
func (f *Finding) SetCodeSnippet(snippet string) {
	f.CodeSnippet = snippet
	f.UpdatedAt = time.Now()
}

// SetColumnNumber sets the column number for the finding
func (f *Finding) SetColumnNumber(column int) {
	f.ColumnNumber = column
	f.UpdatedAt = time.Now()
}

// SetConsensusScore sets the consensus score from multiple tools
func (f *Finding) SetConsensusScore(score float64) {
	f.ConsensusScore = &score
	f.UpdatedAt = time.Now()
}

// MarkAsFixed marks the finding as fixed
func (f *Finding) MarkAsFixed() {
	f.Status = FindingStatusFixed
	f.UpdatedAt = time.Now()
}

// MarkAsIgnored marks the finding as ignored
func (f *Finding) MarkAsIgnored() {
	f.Status = FindingStatusIgnored
	f.UpdatedAt = time.Now()
}

// MarkAsFalsePositive marks the finding as a false positive
func (f *Finding) MarkAsFalsePositive() {
	f.Status = FindingStatusFalsePositive
	f.UpdatedAt = time.Now()
}

// Reopen reopens a closed finding
func (f *Finding) Reopen() {
	f.Status = FindingStatusOpen
	f.UpdatedAt = time.Now()
}

// AddFixSuggestion adds a fix suggestion to the finding
func (f *Finding) AddFixSuggestion(suggestion map[string]interface{}) {
	f.FixSuggestion = suggestion
	f.UpdatedAt = time.Now()
}

// AddReference adds a reference URL to the finding
func (f *Finding) AddReference(reference string) {
	if f.References == nil {
		f.References = []string{}
	}
	
	// Check if reference already exists
	for _, ref := range f.References {
		if ref == reference {
			return
		}
	}
	
	f.References = append(f.References, reference)
	f.UpdatedAt = time.Now()
}

// GetSeverityScore returns a numeric score for the severity level
func (f *Finding) GetSeverityScore() int {
	switch f.Severity {
	case FindingSeverityCritical:
		return 5
	case FindingSeverityHigh:
		return 4
	case FindingSeverityMedium:
		return 3
	case FindingSeverityLow:
		return 2
	case FindingSeverityInfo:
		return 1
	default:
		return 0
	}
}

// IsOpen checks if the finding is in open status
func (f *Finding) IsOpen() bool {
	return f.Status == FindingStatusOpen
}

// IsClosed checks if the finding is closed (fixed, ignored, or false positive)
func (f *Finding) IsClosed() bool {
	return f.Status == FindingStatusFixed || f.Status == FindingStatusIgnored || f.Status == FindingStatusFalsePositive
}

// IsHighPriority checks if the finding is high priority (critical or high severity)
func (f *Finding) IsHighPriority() bool {
	return f.Severity == FindingSeverityCritical || f.Severity == FindingSeverityHigh
}

// Validate validates the finding entity
func (f *Finding) Validate() error {
	if f.ScanJobID == uuid.Nil {
		return NewValidationError("scan_job_id is required")
	}
	if f.Tool == "" {
		return NewValidationError("tool is required")
	}
	if f.RuleID == "" {
		return NewValidationError("rule_id is required")
	}
	if f.Severity == "" {
		return NewValidationError("severity is required")
	}
	if f.Category == "" {
		return NewValidationError("category is required")
	}
	if f.Title == "" {
		return NewValidationError("title is required")
	}
	if f.Description == "" {
		return NewValidationError("description is required")
	}
	if f.FilePath == "" {
		return NewValidationError("file_path is required")
	}
	if f.LineNumber < 1 {
		return NewValidationError("line_number must be greater than 0")
	}
	if f.Confidence < 0 || f.Confidence > 1 {
		return NewValidationError("confidence must be between 0 and 1")
	}
	return nil
}