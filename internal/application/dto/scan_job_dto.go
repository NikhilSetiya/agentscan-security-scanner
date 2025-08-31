package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
)

// CreateScanJobRequest represents a request to create a scan job
type CreateScanJobRequest struct {
	RepositoryID    uuid.UUID            `json:"repository_id" validate:"required"`
	Branch          string               `json:"branch" validate:"omitempty,min=1,max=100"`
	CommitSHA       string               `json:"commit_sha" validate:"omitempty,min=1,max=100"`
	ScanType        entities.ScanType    `json:"scan_type" validate:"required"`
	Priority        entities.Priority    `json:"priority" validate:"min=1,max=10"`
	AgentsRequested []string             `json:"agents_requested" validate:"omitempty"`
}

// UpdateScanJobStatusRequest represents a request to update scan job status
type UpdateScanJobStatusRequest struct {
	Status entities.ScanJobStatus `json:"status" validate:"required"`
}

// ScanJobResponse represents a scan job in API responses
type ScanJobResponse struct {
	ID               uuid.UUID              `json:"id"`
	RepositoryID     uuid.UUID              `json:"repository_id"`
	UserID           *uuid.UUID             `json:"user_id,omitempty"`
	Branch           string                 `json:"branch"`
	CommitSHA        string                 `json:"commit_sha"`
	ScanType         entities.ScanType      `json:"scan_type"`
	Priority         entities.Priority      `json:"priority"`
	Status           entities.ScanJobStatus `json:"status"`
	AgentsRequested  []string               `json:"agents_requested"`
	AgentsCompleted  []string               `json:"agents_completed"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// ScanJobWithDetailsResponse represents a scan job with additional details
type ScanJobWithDetailsResponse struct {
	ScanJobResponse
	Repository    *RepositoryResponse `json:"repository,omitempty"`
	User          *UserResponse       `json:"user,omitempty"`
	FindingsCount int                 `json:"findings_count"`
	Duration      *time.Duration      `json:"duration,omitempty"`
}

// ScanJobListResponse represents a paginated list of scan jobs
type ScanJobListResponse struct {
	ScanJobs   []ScanJobResponse `json:"scan_jobs"`
	Pagination Pagination        `json:"pagination"`
}

// ToScanJobResponse converts a domain scan job entity to response DTO
func ToScanJobResponse(scanJob *entities.ScanJob) ScanJobResponse {
	return ScanJobResponse{
		ID:               scanJob.ID,
		RepositoryID:     scanJob.RepositoryID,
		UserID:           scanJob.UserID,
		Branch:           scanJob.Branch,
		CommitSHA:        scanJob.CommitSHA,
		ScanType:         scanJob.ScanType,
		Priority:         scanJob.Priority,
		Status:           scanJob.Status,
		AgentsRequested:  scanJob.AgentsRequested,
		AgentsCompleted:  scanJob.AgentsCompleted,
		StartedAt:        scanJob.StartedAt,
		CompletedAt:      scanJob.CompletedAt,
		ErrorMessage:     scanJob.ErrorMessage,
		Metadata:         scanJob.Metadata,
		CreatedAt:        scanJob.CreatedAt,
		UpdatedAt:        scanJob.UpdatedAt,
	}
}

// ToScanJobWithDetailsResponse converts a domain scan job with details to response DTO
func ToScanJobWithDetailsResponse(scanJobDetails *entities.ScanJobWithDetails) ScanJobWithDetailsResponse {
	response := ScanJobWithDetailsResponse{
		ScanJobResponse: ToScanJobResponse(scanJobDetails.ScanJob),
		FindingsCount:   scanJobDetails.FindingsCount,
		Duration:        scanJobDetails.Duration,
	}
	
	if scanJobDetails.Repository != nil {
		repoResponse := ToRepositoryResponse(scanJobDetails.Repository)
		response.Repository = &repoResponse
	}
	
	if scanJobDetails.User != nil {
		userResponse := ToUserResponse(scanJobDetails.User)
		response.User = &userResponse
	}
	
	return response
}

// ToScanJobListResponse converts a list of domain scan job entities to response DTO
func ToScanJobListResponse(scanJobs []*entities.ScanJob, pagination Pagination) ScanJobListResponse {
	scanJobResponses := make([]ScanJobResponse, len(scanJobs))
	for i, scanJob := range scanJobs {
		scanJobResponses[i] = ToScanJobResponse(scanJob)
	}
	
	return ScanJobListResponse{
		ScanJobs:   scanJobResponses,
		Pagination: pagination,
	}
}