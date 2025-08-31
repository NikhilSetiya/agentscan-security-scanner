package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// ScanJobRepositoryImpl implements the ScanJobRepository interface
type ScanJobRepositoryImpl struct {
	db *sqlx.DB
}

// NewScanJobRepository creates a new scan job repository implementation
func NewScanJobRepository(db *sqlx.DB) repositories.ScanJobRepository {
	return &ScanJobRepositoryImpl{
		db: db,
	}
}

// Create creates a new scan job
func (r *ScanJobRepositoryImpl) Create(ctx context.Context, scanJob *entities.ScanJob) error {
	query := `
		INSERT INTO scan_jobs (
			id, repository_id, user_id, branch, commit_sha, scan_type, status,
			priority, agents, completed_agents, metadata, error_message,
			started_at, completed_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		scanJob.ID,
		scanJob.RepositoryID,
		scanJob.UserID,
		scanJob.Branch,
		scanJob.CommitSHA,
		scanJob.ScanType,
		scanJob.Status,
		scanJob.Priority,
		pq.Array(scanJob.Agents),
		pq.Array(scanJob.CompletedAgents),
		scanJob.Metadata,
		scanJob.ErrorMessage,
		scanJob.StartedAt,
		scanJob.CompletedAt,
		scanJob.CreatedAt,
		scanJob.UpdatedAt,
	)
	
	return err
}

// GetByID retrieves a scan job by ID
func (r *ScanJobRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.ScanJob, error) {
	query := `
		SELECT id, repository_id, user_id, branch, commit_sha, scan_type, status,
			   priority, agents, completed_agents, metadata, error_message,
			   started_at, completed_at, created_at, updated_at
		FROM scan_jobs
		WHERE id = $1
	`
	
	var scanJob entities.ScanJob
	err := r.db.GetContext(ctx, &scanJob, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("scan job not found")
		}
		return nil, err
	}
	
	return &scanJob, nil
}

// Update updates an existing scan job
func (r *ScanJobRepositoryImpl) Update(ctx context.Context, scanJob *entities.ScanJob) error {
	query := `
		UPDATE scan_jobs
		SET repository_id = $2, user_id = $3, branch = $4, commit_sha = $5,
			scan_type = $6, status = $7, priority = $8, agents = $9,
			completed_agents = $10, metadata = $11, error_message = $12,
			started_at = $13, completed_at = $14, updated_at = $15
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query,
		scanJob.ID,
		scanJob.RepositoryID,
		scanJob.UserID,
		scanJob.Branch,
		scanJob.CommitSHA,
		scanJob.ScanType,
		scanJob.Status,
		scanJob.Priority,
		pq.Array(scanJob.Agents),
		pq.Array(scanJob.CompletedAgents),
		scanJob.Metadata,
		scanJob.ErrorMessage,
		scanJob.StartedAt,
		scanJob.CompletedAt,
		scanJob.UpdatedAt,
	)
	
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("scan job not found")
	}
	
	return nil
}

// Delete deletes a scan job by ID
func (r *ScanJobRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM scan_jobs WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("scan job not found")
	}
	
	return nil
}

// List retrieves scan jobs with filtering and pagination
func (r *ScanJobRepositoryImpl) List(ctx context.Context, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.ScanJob, int64, error) {
	// Build WHERE clause
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1
	
	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(branch ILIKE $%d OR commit_sha ILIKE $%d)", argIndex, argIndex))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern)
		argIndex++
	}
	
	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}
	
	whereClause := strings.Join(whereClauses, " AND ")
	
	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scan_jobs WHERE %s", whereClause)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	
	// Build ORDER BY clause
	orderBy := "ORDER BY created_at DESC"
	if filter.SortBy != "" {
		direction := "ASC"
		if filter.SortOrder == "desc" {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", filter.SortBy, direction)
	}
	
	// Build main query with pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	query := fmt.Sprintf(`
		SELECT id, repository_id, user_id, branch, commit_sha, scan_type, status,
			   priority, agents, completed_agents, metadata, error_message,
			   started_at, completed_at, created_at, updated_at
		FROM scan_jobs
		WHERE %s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIndex, argIndex+1)
	
	args = append(args, pagination.PageSize, offset)
	
	var scanJobs []*entities.ScanJob
	err = r.db.SelectContext(ctx, &scanJobs, query, args...)
	if err != nil {
		return nil, 0, err
	}
	
	return scanJobs, total, nil
}

// Exists checks if a scan job exists by ID
func (r *ScanJobRepositoryImpl) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM scan_jobs WHERE id = $1)`
	
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id)
	return exists, err
}

// Count returns the total count of scan jobs matching the filter
func (r *ScanJobRepositoryImpl) Count(ctx context.Context, filter repositories.Filter) (int64, error) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1
	
	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(branch ILIKE $%d OR commit_sha ILIKE $%d)", argIndex, argIndex))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern)
		argIndex++
	}
	
	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filter.Status)
	}
	
	whereClause := strings.Join(whereClauses, " AND ")
	query := fmt.Sprintf("SELECT COUNT(*) FROM scan_jobs WHERE %s", whereClause)
	
	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	return count, err
}

// ListByRepository retrieves scan jobs by repository ID
func (r *ScanJobRepositoryImpl) ListByRepository(ctx context.Context, repositoryID uuid.UUID, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.ScanJob, int64, error) {
	// Add repository filter to the existing filter
	repoFilter := filter
	// Build WHERE clause with repository filter
	whereClauses := []string{"repository_id = $1"}
	args := []interface{}{repositoryID}
	argIndex := 2
	
	if repoFilter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(branch ILIKE $%d OR commit_sha ILIKE $%d)", argIndex, argIndex))
		searchPattern := "%" + repoFilter.Search + "%"
		args = append(args, searchPattern)
		argIndex++
	}
	
	if repoFilter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, repoFilter.Status)
		argIndex++
	}
	
	whereClause := strings.Join(whereClauses, " AND ")
	
	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scan_jobs WHERE %s", whereClause)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	
	// Build ORDER BY clause
	orderBy := "ORDER BY created_at DESC"
	if repoFilter.SortBy != "" {
		direction := "ASC"
		if repoFilter.SortOrder == "desc" {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", repoFilter.SortBy, direction)
	}
	
	// Build main query with pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	query := fmt.Sprintf(`
		SELECT id, repository_id, user_id, branch, commit_sha, scan_type, status,
			   priority, agents, completed_agents, metadata, error_message,
			   started_at, completed_at, created_at, updated_at
		FROM scan_jobs
		WHERE %s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIndex, argIndex+1)
	
	args = append(args, pagination.PageSize, offset)
	
	var scanJobs []*entities.ScanJob
	err = r.db.SelectContext(ctx, &scanJobs, query, args...)
	if err != nil {
		return nil, 0, err
	}
	
	return scanJobs, total, nil
}

// ListByUser retrieves scan jobs by user ID
func (r *ScanJobRepositoryImpl) ListByUser(ctx context.Context, userID uuid.UUID, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.ScanJob, int64, error) {
	// Build WHERE clause with user filter
	whereClauses := []string{"user_id = $1"}
	args := []interface{}{userID}
	argIndex := 2
	
	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(branch ILIKE $%d OR commit_sha ILIKE $%d)", argIndex, argIndex))
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern)
		argIndex++
	}
	
	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}
	
	whereClause := strings.Join(whereClauses, " AND ")
	
	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scan_jobs WHERE %s", whereClause)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	
	// Build ORDER BY clause
	orderBy := "ORDER BY created_at DESC"
	if filter.SortBy != "" {
		direction := "ASC"
		if filter.SortOrder == "desc" {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", filter.SortBy, direction)
	}
	
	// Build main query with pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	query := fmt.Sprintf(`
		SELECT id, repository_id, user_id, branch, commit_sha, scan_type, status,
			   priority, agents, completed_agents, metadata, error_message,
			   started_at, completed_at, created_at, updated_at
		FROM scan_jobs
		WHERE %s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIndex, argIndex+1)
	
	args = append(args, pagination.PageSize, offset)
	
	var scanJobs []*entities.ScanJob
	err = r.db.SelectContext(ctx, &scanJobs, query, args...)
	if err != nil {
		return nil, 0, err
	}
	
	return scanJobs, total, nil
}

// GetQueuedJobs retrieves queued scan jobs
func (r *ScanJobRepositoryImpl) GetQueuedJobs(ctx context.Context, limit int) ([]*entities.ScanJob, error) {
	query := `
		SELECT id, repository_id, user_id, branch, commit_sha, scan_type, status,
			   priority, agents, completed_agents, metadata, error_message,
			   started_at, completed_at, created_at, updated_at
		FROM scan_jobs
		WHERE status = 'queued'
		ORDER BY priority DESC, created_at ASC
		LIMIT $1
	`
	
	var scanJobs []*entities.ScanJob
	err := r.db.SelectContext(ctx, &scanJobs, query, limit)
	if err != nil {
		return nil, err
	}
	
	return scanJobs, nil
}

// GetRunningJobs retrieves currently running scan jobs
func (r *ScanJobRepositoryImpl) GetRunningJobs(ctx context.Context) ([]*entities.ScanJob, error) {
	query := `
		SELECT id, repository_id, user_id, branch, commit_sha, scan_type, status,
			   priority, agents, completed_agents, metadata, error_message,
			   started_at, completed_at, created_at, updated_at
		FROM scan_jobs
		WHERE status = 'running'
		ORDER BY started_at ASC
	`
	
	var scanJobs []*entities.ScanJob
	err := r.db.SelectContext(ctx, &scanJobs, query)
	if err != nil {
		return nil, err
	}
	
	return scanJobs, nil
}

// UpdateStatus updates the status of a scan job
func (r *ScanJobRepositoryImpl) UpdateStatus(ctx context.Context, scanJobID uuid.UUID, status entities.ScanStatus) error {
	query := `
		UPDATE scan_jobs
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, scanJobID, status)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("scan job not found")
	}
	
	return nil
}

// AddCompletedAgent adds an agent to the completed agents list
func (r *ScanJobRepositoryImpl) AddCompletedAgent(ctx context.Context, scanJobID uuid.UUID, agent string) error {
	query := `
		UPDATE scan_jobs
		SET completed_agents = array_append(completed_agents, $2), updated_at = NOW()
		WHERE id = $1 AND NOT ($2 = ANY(completed_agents))
	`
	
	result, err := r.db.ExecContext(ctx, query, scanJobID, agent)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("scan job not found or agent already completed")
	}
	
	return nil
}

// UpdateMetadata updates the metadata of a scan job
func (r *ScanJobRepositoryImpl) UpdateMetadata(ctx context.Context, scanJobID uuid.UUID, metadata map[string]interface{}) error {
	query := `
		UPDATE scan_jobs
		SET metadata = $2, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, scanJobID, metadata)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("scan job not found")
	}
	
	return nil
}

// GetWithDetails retrieves a scan job with related details (repository, user)
func (r *ScanJobRepositoryImpl) GetWithDetails(ctx context.Context, scanJobID uuid.UUID) (*entities.ScanJobWithDetails, error) {
	query := `
		SELECT 
			sj.id, sj.repository_id, sj.user_id, sj.branch, sj.commit_sha, 
			sj.scan_type, sj.status, sj.priority, sj.agents, sj.completed_agents,
			sj.metadata, sj.error_message, sj.started_at, sj.completed_at,
			sj.created_at, sj.updated_at,
			r.name as repo_name, r.url as repo_url,
			u.name as user_name, u.email as user_email
		FROM scan_jobs sj
		LEFT JOIN repositories r ON sj.repository_id = r.id
		LEFT JOIN users u ON sj.user_id = u.id
		WHERE sj.id = $1
	`
	
	var details entities.ScanJobWithDetails
	err := r.db.QueryRowContext(ctx, query, scanJobID).Scan(
		&details.ScanJob.ID,
		&details.ScanJob.RepositoryID,
		&details.ScanJob.UserID,
		&details.ScanJob.Branch,
		&details.ScanJob.CommitSHA,
		&details.ScanJob.ScanType,
		&details.ScanJob.Status,
		&details.ScanJob.Priority,
		pq.Array(&details.ScanJob.Agents),
		pq.Array(&details.ScanJob.CompletedAgents),
		&details.ScanJob.Metadata,
		&details.ScanJob.ErrorMessage,
		&details.ScanJob.StartedAt,
		&details.ScanJob.CompletedAt,
		&details.ScanJob.CreatedAt,
		&details.ScanJob.UpdatedAt,
		&details.RepositoryName,
		&details.RepositoryURL,
		&details.UserName,
		&details.UserEmail,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("scan job not found")
		}
		return nil, err
	}
	
	return &details, nil
}