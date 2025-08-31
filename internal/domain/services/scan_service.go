package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// ScanService provides business logic for scan operations
type ScanService struct {
	scanJobRepo repositories.ScanJobRepository
	repoRepo    repositories.RepositoryRepository
	userRepo    repositories.UserRepository
}

// NewScanService creates a new scan service
func NewScanService(scanJobRepo repositories.ScanJobRepository, repoRepo repositories.RepositoryRepository, userRepo repositories.UserRepository) *ScanService {
	return &ScanService{
		scanJobRepo: scanJobRepo,
		repoRepo:    repoRepo,
		userRepo:    userRepo,
	}
}

// CreateScanJob creates a new scan job with validation
func (s *ScanService) CreateScanJob(ctx context.Context, repositoryID uuid.UUID, userID *uuid.UUID, branch, commitSHA string, scanType entities.ScanType, priority entities.Priority, agents []string) (*entities.ScanJob, error) {
	// Validate repository exists
	repo, err := s.repoRepo.GetByID(ctx, repositoryID)
	if err != nil {
		return nil, entities.NewNotFoundError("repository not found")
	}

	if !repo.IsActive {
		return nil, entities.NewBusinessRuleError("cannot scan inactive repository")
	}

	// Validate user exists if provided
	if userID != nil {
		_, err := s.userRepo.GetByID(ctx, *userID)
		if err != nil {
			return nil, entities.NewNotFoundError("user not found")
		}
	}

	// Set defaults
	if branch == "" {
		branch = repo.DefaultBranch
	}
	if commitSHA == "" {
		commitSHA = "HEAD"
	}
	if len(agents) == 0 {
		agents = []string{"semgrep", "eslint-security"} // Default agents
	}

	// Create scan job
	scanJob := entities.NewScanJob(repositoryID, userID, branch, commitSHA, scanType, priority, agents)

	// Validate scan job
	if err := scanJob.Validate(); err != nil {
		return nil, err
	}

	// Save scan job
	if err := s.scanJobRepo.Create(ctx, scanJob); err != nil {
		return nil, err
	}

	return scanJob, nil
}

// GetScanJobByID retrieves a scan job by ID
func (s *ScanService) GetScanJobByID(ctx context.Context, scanJobID uuid.UUID) (*entities.ScanJob, error) {
	return s.scanJobRepo.GetByID(ctx, scanJobID)
}

// GetScanJobWithDetails retrieves a scan job with additional details
func (s *ScanService) GetScanJobWithDetails(ctx context.Context, scanJobID uuid.UUID) (*entities.ScanJobWithDetails, error) {
	return s.scanJobRepo.GetWithDetails(ctx, scanJobID)
}

// StartScanJob starts a scan job
func (s *ScanService) StartScanJob(ctx context.Context, scanJobID uuid.UUID) error {
	// Get scan job
	scanJob, err := s.scanJobRepo.GetByID(ctx, scanJobID)
	if err != nil {
		return err
	}

	// Validate can be started
	if scanJob.Status != entities.ScanJobStatusQueued {
		return entities.NewBusinessRuleError("only queued scan jobs can be started")
	}

	// Start the job
	return s.scanJobRepo.Start(ctx, scanJobID)
}

// CompleteScanJob completes a scan job
func (s *ScanService) CompleteScanJob(ctx context.Context, scanJobID uuid.UUID) error {
	// Get scan job
	scanJob, err := s.scanJobRepo.GetByID(ctx, scanJobID)
	if err != nil {
		return err
	}

	// Validate can be completed
	if scanJob.Status != entities.ScanJobStatusRunning {
		return entities.NewBusinessRuleError("only running scan jobs can be completed")
	}

	// Complete the job
	if err := s.scanJobRepo.Complete(ctx, scanJobID); err != nil {
		return err
	}

	// Update repository last scan time
	return s.repoRepo.UpdateLastScanTime(ctx, scanJob.RepositoryID)
}

// FailScanJob marks a scan job as failed
func (s *ScanService) FailScanJob(ctx context.Context, scanJobID uuid.UUID, errorMessage string) error {
	// Get scan job
	scanJob, err := s.scanJobRepo.GetByID(ctx, scanJobID)
	if err != nil {
		return err
	}

	// Validate can be failed
	if scanJob.IsCompleted() {
		return entities.NewBusinessRuleError("completed scan jobs cannot be marked as failed")
	}

	// Fail the job
	return s.scanJobRepo.Fail(ctx, scanJobID, errorMessage)
}

// CancelScanJob cancels a scan job
func (s *ScanService) CancelScanJob(ctx context.Context, scanJobID uuid.UUID, userID uuid.UUID) error {
	// Get scan job
	scanJob, err := s.scanJobRepo.GetByID(ctx, scanJobID)
	if err != nil {
		return err
	}

	// Validate user can cancel (owner or admin)
	if scanJob.UserID != nil && *scanJob.UserID != userID {
		// Check if user is admin
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return err
		}
		if !user.IsAdmin() {
			return entities.NewBusinessRuleError("only the scan owner or admin can cancel a scan")
		}
	}

	// Validate can be cancelled
	if scanJob.IsCompleted() {
		return entities.NewBusinessRuleError("completed scan jobs cannot be cancelled")
	}

	// Cancel the job
	return s.scanJobRepo.Cancel(ctx, scanJobID)
}

// RetryScanJob retries a failed scan job
func (s *ScanService) RetryScanJob(ctx context.Context, scanJobID uuid.UUID, userID uuid.UUID) (*entities.ScanJob, error) {
	// Get scan job
	scanJob, err := s.scanJobRepo.GetByID(ctx, scanJobID)
	if err != nil {
		return nil, err
	}

	// Validate user can retry (owner or admin)
	if scanJob.UserID != nil && *scanJob.UserID != userID {
		// Check if user is admin
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !user.IsAdmin() {
			return nil, entities.NewBusinessRuleError("only the scan owner or admin can retry a scan")
		}
	}

	// Validate can be retried
	if !scanJob.CanBeRetried() {
		return nil, entities.NewBusinessRuleError("only failed scan jobs can be retried")
	}

	// Reset the job
	scanJob.Reset()

	// Save changes
	if err := s.scanJobRepo.Update(ctx, scanJob); err != nil {
		return nil, err
	}

	return scanJob, nil
}

// AddCompletedAgent adds an agent to the completed list
func (s *ScanService) AddCompletedAgent(ctx context.Context, scanJobID uuid.UUID, agent string) error {
	return s.scanJobRepo.AddCompletedAgent(ctx, scanJobID, agent)
}

// UpdateScanJobMetadata updates metadata for a scan job
func (s *ScanService) UpdateScanJobMetadata(ctx context.Context, scanJobID uuid.UUID, metadata map[string]interface{}) error {
	return s.scanJobRepo.UpdateMetadata(ctx, scanJobID, metadata)
}

// ListScanJobs lists scan jobs with filtering and pagination
func (s *ScanService) ListScanJobs(ctx context.Context, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.ScanJob, int64, error) {
	return s.scanJobRepo.List(ctx, filter, pagination)
}

// ListScanJobsByRepository lists scan jobs for a specific repository
func (s *ScanService) ListScanJobsByRepository(ctx context.Context, repoID uuid.UUID, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.ScanJob, int64, error) {
	return s.scanJobRepo.ListByRepository(ctx, repoID, filter, pagination)
}

// ListScanJobsByUser lists scan jobs for a specific user
func (s *ScanService) ListScanJobsByUser(ctx context.Context, userID uuid.UUID, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.ScanJob, int64, error) {
	return s.scanJobRepo.ListByUser(ctx, userID, filter, pagination)
}

// GetQueuedJobs retrieves queued scan jobs ordered by priority
func (s *ScanService) GetQueuedJobs(ctx context.Context, limit int) ([]*entities.ScanJob, error) {
	return s.scanJobRepo.GetQueuedJobs(ctx, limit)
}

// GetRunningJobs retrieves currently running scan jobs
func (s *ScanService) GetRunningJobs(ctx context.Context) ([]*entities.ScanJob, error) {
	return s.scanJobRepo.GetRunningJobs(ctx)
}

// ValidateScanJobAccess validates if a user can access a scan job
func (s *ScanService) ValidateScanJobAccess(ctx context.Context, userID, scanJobID uuid.UUID) error {
	// Get scan job
	scanJob, err := s.scanJobRepo.GetByID(ctx, scanJobID)
	if err != nil {
		return err
	}

	// Check if user owns the scan
	if scanJob.UserID != nil && *scanJob.UserID == userID {
		return nil
	}

	// Check if user is admin
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.IsAdmin() {
		return nil
	}

	// TODO: Check organization-level access
	return entities.NewBusinessRuleError("insufficient permissions to access this scan job")
}