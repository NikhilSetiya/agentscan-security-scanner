package entities

import (
	"time"

	"github.com/google/uuid"
)

// ScanJobStatus represents the status of a scan job
type ScanJobStatus string

const (
	ScanJobStatusQueued    ScanJobStatus = "queued"
	ScanJobStatusRunning   ScanJobStatus = "running"
	ScanJobStatusCompleted ScanJobStatus = "completed"
	ScanJobStatusFailed    ScanJobStatus = "failed"
	ScanJobStatusCancelled ScanJobStatus = "cancelled"
)

// ScanType represents the type of scan
type ScanType string

const (
	ScanTypeFull        ScanType = "full"
	ScanTypeIncremental ScanType = "incremental"
	ScanTypeIDE         ScanType = "ide"
)

// Priority represents scan priority levels
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 5
	PriorityHigh   Priority = 8
	PriorityCritical Priority = 10
)

// ScanJob represents a security scan job entity in the domain
type ScanJob struct {
	ID               uuid.UUID              `json:"id"`
	RepositoryID     uuid.UUID              `json:"repository_id"`
	UserID           *uuid.UUID             `json:"user_id,omitempty"`
	Branch           string                 `json:"branch" validate:"required"`
	CommitSHA        string                 `json:"commit_sha" validate:"required"`
	ScanType         ScanType               `json:"scan_type" validate:"required"`
	Priority         Priority               `json:"priority" validate:"min=1,max=10"`
	Status           ScanJobStatus          `json:"status" validate:"required"`
	AgentsRequested  []string               `json:"agents_requested"`
	AgentsCompleted  []string               `json:"agents_completed"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// ScanJobWithDetails represents a scan job with additional details
type ScanJobWithDetails struct {
	*ScanJob
	Repository    *Repository `json:"repository"`
	User          *User       `json:"user,omitempty"`
	FindingsCount int         `json:"findings_count"`
	Duration      *time.Duration `json:"duration,omitempty"`
}

// NewScanJob creates a new scan job entity
func NewScanJob(repositoryID uuid.UUID, userID *uuid.UUID, branch, commitSHA string, scanType ScanType, priority Priority, agents []string) *ScanJob {
	return &ScanJob{
		ID:               uuid.New(),
		RepositoryID:     repositoryID,
		UserID:           userID,
		Branch:           branch,
		CommitSHA:        commitSHA,
		ScanType:         scanType,
		Priority:         priority,
		Status:           ScanJobStatusQueued,
		AgentsRequested:  agents,
		AgentsCompleted:  []string{},
		Metadata:         make(map[string]interface{}),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// Start marks the scan job as started
func (s *ScanJob) Start() {
	s.Status = ScanJobStatusRunning
	now := time.Now()
	s.StartedAt = &now
	s.UpdatedAt = now
}

// Complete marks the scan job as completed
func (s *ScanJob) Complete() {
	s.Status = ScanJobStatusCompleted
	now := time.Now()
	s.CompletedAt = &now
	s.UpdatedAt = now
}

// Fail marks the scan job as failed with an error message
func (s *ScanJob) Fail(errorMessage string) {
	s.Status = ScanJobStatusFailed
	s.ErrorMessage = errorMessage
	now := time.Now()
	s.CompletedAt = &now
	s.UpdatedAt = now
}

// Cancel marks the scan job as cancelled
func (s *ScanJob) Cancel() {
	s.Status = ScanJobStatusCancelled
	now := time.Now()
	s.CompletedAt = &now
	s.UpdatedAt = now
}

// AddCompletedAgent adds an agent to the completed list
func (s *ScanJob) AddCompletedAgent(agent string) {
	for _, completed := range s.AgentsCompleted {
		if completed == agent {
			return // Already completed
		}
	}
	s.AgentsCompleted = append(s.AgentsCompleted, agent)
	s.UpdatedAt = time.Now()
}

// IsCompleted checks if the scan job is completed (successfully or with failure)
func (s *ScanJob) IsCompleted() bool {
	return s.Status == ScanJobStatusCompleted || s.Status == ScanJobStatusFailed || s.Status == ScanJobStatusCancelled
}

// IsRunning checks if the scan job is currently running
func (s *ScanJob) IsRunning() bool {
	return s.Status == ScanJobStatusRunning
}

// CanBeRetried checks if the scan job can be retried
func (s *ScanJob) CanBeRetried() bool {
	return s.Status == ScanJobStatusFailed
}

// Reset resets the scan job for retry
func (s *ScanJob) Reset() {
	s.Status = ScanJobStatusQueued
	s.ErrorMessage = ""
	s.StartedAt = nil
	s.CompletedAt = nil
	s.AgentsCompleted = []string{}
	s.UpdatedAt = time.Now()
}

// GetDuration calculates the duration of the scan job
func (s *ScanJob) GetDuration() *time.Duration {
	if s.StartedAt == nil {
		return nil
	}
	
	endTime := time.Now()
	if s.CompletedAt != nil {
		endTime = *s.CompletedAt
	}
	
	duration := endTime.Sub(*s.StartedAt)
	return &duration
}

// UpdateMetadata updates metadata for the scan job
func (s *ScanJob) UpdateMetadata(key string, value interface{}) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]interface{})
	}
	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
}

// GetMetadata retrieves metadata from the scan job
func (s *ScanJob) GetMetadata(key string) (interface{}, bool) {
	if s.Metadata == nil {
		return nil, false
	}
	value, exists := s.Metadata[key]
	return value, exists
}

// Validate validates the scan job entity
func (s *ScanJob) Validate() error {
	if s.RepositoryID == uuid.Nil {
		return NewValidationError("repository_id is required")
	}
	if s.Branch == "" {
		return NewValidationError("branch is required")
	}
	if s.CommitSHA == "" {
		return NewValidationError("commit_sha is required")
	}
	if s.ScanType == "" {
		return NewValidationError("scan_type is required")
	}
	if s.Priority < 1 || s.Priority > 10 {
		return NewValidationError("priority must be between 1 and 10")
	}
	return nil
}