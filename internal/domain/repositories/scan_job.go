package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
)

// ScanJobRepository defines the interface for scan job data operations
type ScanJobRepository interface {
	BaseRepository[*entities.ScanJob, uuid.UUID]
	
	// ListByRepository retrieves scan jobs for a specific repository
	ListByRepository(ctx context.Context, repoID uuid.UUID, filter Filter, pagination Pagination) ([]*entities.ScanJob, int64, error)
	
	// ListByUser retrieves scan jobs for a specific user
	ListByUser(ctx context.Context, userID uuid.UUID, filter Filter, pagination Pagination) ([]*entities.ScanJob, int64, error)
	
	// ListByStatus retrieves scan jobs by status
	ListByStatus(ctx context.Context, status entities.ScanJobStatus, filter Filter, pagination Pagination) ([]*entities.ScanJob, int64, error)
	
	// GetWithDetails retrieves a scan job with additional details
	GetWithDetails(ctx context.Context, id uuid.UUID) (*entities.ScanJobWithDetails, error)
	
	// UpdateStatus updates the status of a scan job
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.ScanJobStatus) error
	
	// Start marks a scan job as started
	Start(ctx context.Context, id uuid.UUID) error
	
	// Complete marks a scan job as completed
	Complete(ctx context.Context, id uuid.UUID) error
	
	// Fail marks a scan job as failed with an error message
	Fail(ctx context.Context, id uuid.UUID, errorMessage string) error
	
	// Cancel marks a scan job as cancelled
	Cancel(ctx context.Context, id uuid.UUID) error
	
	// AddCompletedAgent adds an agent to the completed list
	AddCompletedAgent(ctx context.Context, id uuid.UUID, agent string) error
	
	// UpdateMetadata updates metadata for a scan job
	UpdateMetadata(ctx context.Context, id uuid.UUID, metadata map[string]interface{}) error
	
	// GetQueuedJobs retrieves queued scan jobs ordered by priority
	GetQueuedJobs(ctx context.Context, limit int) ([]*entities.ScanJob, error)
	
	// GetRunningJobs retrieves currently running scan jobs
	GetRunningJobs(ctx context.Context) ([]*entities.ScanJob, error)
	
	// GetJobsForCleanup retrieves old completed/failed jobs for cleanup
	GetJobsForCleanup(ctx context.Context, olderThanDays int) ([]*entities.ScanJob, error)
}