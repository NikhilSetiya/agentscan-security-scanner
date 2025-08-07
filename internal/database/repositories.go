package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/types"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *types.User) error {
	query := `
		INSERT INTO users (id, email, name, avatar_url, github_id, gitlab_id)
		VALUES (:id, :email, :name, :avatar_url, :github_id, :gitlab_id)`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return errors.NewInternalError("failed to create user").WithCause(err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	var user types.User
	query := `SELECT * FROM users WHERE id = $1`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("user")
		}
		return nil, errors.NewInternalError("failed to get user by ID").WithCause(err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*types.User, error) {
	var user types.User
	query := `SELECT * FROM users WHERE email = $1`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("user")
		}
		return nil, errors.NewInternalError("failed to get user by email").WithCause(err)
	}

	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *types.User) error {
	query := `
		UPDATE users 
		SET name = :name, avatar_url = :avatar_url, updated_at = NOW()
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return errors.NewInternalError("failed to update user").WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("user")
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.NewInternalError("failed to delete user").WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("user")
	}

	return nil
}

// ScanJobRepository handles scan job database operations
type ScanJobRepository struct {
	db *DB
}

// NewScanJobRepository creates a new scan job repository
func NewScanJobRepository(db *DB) *ScanJobRepository {
	return &ScanJobRepository{db: db}
}

// Create creates a new scan job
func (r *ScanJobRepository) Create(ctx context.Context, job *types.ScanJob) error {
	query := `
		INSERT INTO scan_jobs (
			id, repository_id, user_id, branch, commit_sha, scan_type, 
			priority, status, agents_requested, metadata
		) VALUES (
			:id, :repository_id, :user_id, :branch, :commit_sha, :scan_type,
			:priority, :status, :agents_requested, :metadata
		)`

	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	_, err := r.db.NamedExecContext(ctx, query, job)
	if err != nil {
		return errors.NewInternalError("failed to create scan job").WithCause(err)
	}

	return nil
}

// GetByID retrieves a scan job by ID
func (r *ScanJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.ScanJob, error) {
	var job types.ScanJob
	query := `SELECT * FROM scan_jobs WHERE id = $1`

	err := r.db.GetContext(ctx, &job, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("scan job")
		}
		return nil, errors.NewInternalError("failed to get scan job by ID").WithCause(err)
	}

	return &job, nil
}

// UpdateStatus updates the status of a scan job
func (r *ScanJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE scan_jobs SET status = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return errors.NewInternalError("failed to update scan job status").WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("scan job")
	}

	return nil
}

// SetStarted marks a scan job as started
func (r *ScanJobRepository) SetStarted(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE scan_jobs 
		SET status = 'running', started_at = NOW(), updated_at = NOW() 
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.NewInternalError("failed to set scan job as started").WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("scan job")
	}

	return nil
}

// SetCompleted marks a scan job as completed
func (r *ScanJobRepository) SetCompleted(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE scan_jobs 
		SET status = 'completed', completed_at = NOW(), updated_at = NOW() 
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.NewInternalError("failed to set scan job as completed").WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("scan job")
	}

	return nil
}

// SetFailed marks a scan job as failed with an error message
func (r *ScanJobRepository) SetFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	query := `
		UPDATE scan_jobs 
		SET status = 'failed', error_message = $2, completed_at = NOW(), updated_at = NOW() 
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, errorMsg)
	if err != nil {
		return errors.NewInternalError("failed to set scan job as failed").WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("scan job")
	}

	return nil
}

// ListByRepository lists scan jobs for a repository
func (r *ScanJobRepository) ListByRepository(ctx context.Context, repoID uuid.UUID, limit, offset int) ([]*types.ScanJob, error) {
	var jobs []*types.ScanJob
	query := `
		SELECT * FROM scan_jobs 
		WHERE repository_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`

	err := r.db.SelectContext(ctx, &jobs, query, repoID, limit, offset)
	if err != nil {
		return nil, errors.NewInternalError("failed to list scan jobs by repository").WithCause(err)
	}

	return jobs, nil
}

// FindingRepository handles finding database operations
type FindingRepository struct {
	db *DB
}

// NewFindingRepository creates a new finding repository
func NewFindingRepository(db *DB) *FindingRepository {
	return &FindingRepository{db: db}
}

// Create creates a new finding
func (r *FindingRepository) Create(ctx context.Context, finding *types.Finding) error {
	query := `
		INSERT INTO findings (
			id, scan_result_id, scan_job_id, tool, rule_id, severity, category,
			title, description, file_path, line_number, column_number, code_snippet,
			confidence, status, fix_suggestion, references
		) VALUES (
			:id, :scan_result_id, :scan_job_id, :tool, :rule_id, :severity, :category,
			:title, :description, :file_path, :line_number, :column_number, :code_snippet,
			:confidence, :status, :fix_suggestion, :references
		)`

	if finding.ID == uuid.Nil {
		finding.ID = uuid.New()
	}
	finding.CreatedAt = time.Now()
	finding.UpdatedAt = time.Now()

	_, err := r.db.NamedExecContext(ctx, query, finding)
	if err != nil {
		return errors.NewInternalError("failed to create finding").WithCause(err)
	}

	return nil
}

// GetByID retrieves a finding by ID
func (r *FindingRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.Finding, error) {
	var finding types.Finding
	query := `SELECT * FROM findings WHERE id = $1`

	err := r.db.GetContext(ctx, &finding, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("finding")
		}
		return nil, errors.NewInternalError("failed to get finding by ID").WithCause(err)
	}

	return &finding, nil
}

// ListByScanJob lists findings for a scan job
func (r *FindingRepository) ListByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*types.Finding, error) {
	var findings []*types.Finding
	query := `
		SELECT * FROM findings 
		WHERE scan_job_id = $1 
		ORDER BY severity DESC, created_at DESC`

	err := r.db.SelectContext(ctx, &findings, query, scanJobID)
	if err != nil {
		return nil, errors.NewInternalError("failed to list findings by scan job").WithCause(err)
	}

	return findings, nil
}

// UpdateStatus updates the status of a finding
func (r *FindingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE findings SET status = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return errors.NewInternalError("failed to update finding status").WithCause(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("finding")
	}

	return nil
}

// Repositories aggregates all repository interfaces
type Repositories struct {
	Users     *UserRepository
	ScanJobs  *ScanJobRepository
	Findings  *FindingRepository
}

// NewRepositories creates a new repositories instance
func NewRepositories(db *DB) *Repositories {
	return &Repositories{
		Users:    NewUserRepository(db),
		ScanJobs: NewScanJobRepository(db),
		Findings: NewFindingRepository(db),
	}
}