package database

import (
	"context"

	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// RepositoryAdapter adapts the Repositories struct to implement the Repository interface
type RepositoryAdapter struct {
	db    *DB
	repos *Repositories
}

// NewRepositoryAdapter creates a new repository adapter
func NewRepositoryAdapter(db *DB, repos *Repositories) Repository {
	return &RepositoryAdapter{
		db:    db,
		repos: repos,
	}
}

// Health checks database connectivity
func (r *RepositoryAdapter) Health(ctx context.Context) error {
	return r.db.Health(ctx)
}

// Scan job operations
func (r *RepositoryAdapter) CreateScanJob(ctx context.Context, job *types.ScanJob) error {
	return r.repos.ScanJobs.Create(ctx, job)
}

func (r *RepositoryAdapter) GetScanJob(ctx context.Context, id uuid.UUID) (*types.ScanJob, error) {
	return r.repos.ScanJobs.GetByID(ctx, id)
}

func (r *RepositoryAdapter) UpdateScanJob(ctx context.Context, job *types.ScanJob) error {
	return r.repos.ScanJobs.Update(ctx, job)
}

func (r *RepositoryAdapter) DeleteScanJob(ctx context.Context, id uuid.UUID) error {
	// TODO: Implement delete method in ScanJobRepository
	return nil
}

func (r *RepositoryAdapter) ListScanJobs(ctx context.Context, filter *ScanJobFilter, pagination *Pagination) ([]*types.ScanJob, int64, error) {
	return r.repos.ScanJobs.List(ctx, filter, pagination)
}

// Scan result operations
func (r *RepositoryAdapter) CreateScanResult(ctx context.Context, result *types.ScanResult) error {
	// TODO: Implement scan result repository
	return nil
}

func (r *RepositoryAdapter) GetScanResult(ctx context.Context, id uuid.UUID) (*types.ScanResult, error) {
	// TODO: Implement scan result repository
	return nil, nil
}

func (r *RepositoryAdapter) GetScanResults(ctx context.Context, scanJobID uuid.UUID) ([]*types.ScanResult, error) {
	// TODO: Implement scan result repository
	return nil, nil
}

// Finding operations
func (r *RepositoryAdapter) CreateFinding(ctx context.Context, finding *types.Finding) error {
	return r.repos.Findings.Create(ctx, finding)
}

func (r *RepositoryAdapter) GetFinding(ctx context.Context, id uuid.UUID) (*types.Finding, error) {
	return r.repos.Findings.GetByID(ctx, id)
}

func (r *RepositoryAdapter) GetFindings(ctx context.Context, scanJobID uuid.UUID, filter *FindingFilter) ([]*types.Finding, error) {
	return r.repos.Findings.ListByScanJob(ctx, scanJobID)
}

func (r *RepositoryAdapter) UpdateFinding(ctx context.Context, finding *types.Finding) error {
	// TODO: Implement update method in FindingRepository
	return nil
}

// User feedback operations
func (r *RepositoryAdapter) CreateUserFeedback(ctx context.Context, feedback *types.UserFeedback) error {
	// TODO: Implement user feedback repository
	return nil
}

func (r *RepositoryAdapter) GetUserFeedback(ctx context.Context, findingID uuid.UUID) ([]*types.UserFeedback, error) {
	// TODO: Implement user feedback repository
	return nil, nil
}

// Repository operations
func (r *RepositoryAdapter) CreateRepository(ctx context.Context, repo *types.Repository) error {
	// TODO: Implement repository repository
	return nil
}

func (r *RepositoryAdapter) GetRepository(ctx context.Context, id uuid.UUID) (*types.Repository, error) {
	// TODO: Implement repository repository
	return nil, nil
}

func (r *RepositoryAdapter) UpdateRepository(ctx context.Context, repo *types.Repository) error {
	// TODO: Implement repository repository
	return nil
}

func (r *RepositoryAdapter) ListRepositories(ctx context.Context, orgID uuid.UUID) ([]*types.Repository, error) {
	// TODO: Implement repository repository
	return nil, nil
}

// Organization operations
func (r *RepositoryAdapter) CreateOrganization(ctx context.Context, org *types.Organization) error {
	// TODO: Implement organization repository
	return nil
}

func (r *RepositoryAdapter) GetOrganization(ctx context.Context, id uuid.UUID) (*types.Organization, error) {
	// TODO: Implement organization repository
	return nil, nil
}

func (r *RepositoryAdapter) UpdateOrganization(ctx context.Context, org *types.Organization) error {
	// TODO: Implement organization repository
	return nil
}

// User operations
func (r *RepositoryAdapter) CreateUser(ctx context.Context, user *types.User) error {
	return r.repos.Users.Create(ctx, user)
}

func (r *RepositoryAdapter) GetUser(ctx context.Context, id uuid.UUID) (*types.User, error) {
	return r.repos.Users.GetByID(ctx, id)
}

func (r *RepositoryAdapter) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	return r.repos.Users.GetByEmail(ctx, email)
}

func (r *RepositoryAdapter) UpdateUser(ctx context.Context, user *types.User) error {
	return r.repos.Users.Update(ctx, user)
}