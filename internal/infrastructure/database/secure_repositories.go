package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/types"
)

// SecureRepositories provides secure database operations with parameterized queries
type SecureRepositories struct {
	db           *sqlx.DB
	queryBuilder *SecureQueryBuilder
}

// NewSecureRepositories creates a new secure repositories instance
func NewSecureRepositories(db *sqlx.DB) *SecureRepositories {
	return &SecureRepositories{
		db:           db,
		queryBuilder: NewSecureQueryBuilder(db),
	}
}

// Repository operations with secure parameterized queries

// CreateRepository creates a new repository with secure parameters
func (sr *SecureRepositories) CreateRepository(ctx context.Context, repo *types.Repository) error {
	query := `
		INSERT INTO repositories (id, name, url, description, language, branch, is_active, organization_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	now := time.Now()
	repo.ID = uuid.New()
	repo.CreatedAt = now
	repo.UpdatedAt = now
	
	_, err := sr.db.ExecContext(ctx, query,
		repo.ID,
		repo.Name,
		repo.URL,
		repo.Description,
		repo.Language,
		repo.Branch,
		repo.IsActive,
		repo.OrganizationID,
		repo.CreatedAt,
		repo.UpdatedAt,
	)
	
	if err != nil {
		return errors.NewInternalError("failed to create repository").WithCause(err)
	}
	
	return nil
}

// GetRepository retrieves a repository by ID with secure parameters
func (sr *SecureRepositories) GetRepository(ctx context.Context, id uuid.UUID) (*types.Repository, error) {
	query := `
		SELECT id, name, url, description, language, branch, is_active, organization_id, created_at, updated_at
		FROM repositories
		WHERE id = $1
	`
	
	var repo types.Repository
	err := sr.db.GetContext(ctx, &repo, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("repository not found")
		}
		return nil, errors.NewInternalError("failed to get repository").WithCause(err)
	}
	
	return &repo, nil
}

// ListRepositories lists repositories with secure filtering and pagination
func (sr *SecureRepositories) ListRepositories(ctx context.Context, orgID *uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.Repository, int, error) {
	// Build secure WHERE clauses
	var whereClauses []WhereClause
	
	if orgID != nil {
		whereClauses = append(whereClauses, CreateWhereClause("r.organization_id", "=", *orgID))
	}
	
	// Add search filter if provided
	if search, ok := filters["search"].(string); ok && search != "" {
		whereClauses = append(whereClauses, CreateWhereClause("r.name", "LIKE", search))
	}
	
	// Add language filter if provided
	if language, ok := filters["language"].(string); ok && language != "" {
		whereClauses = append(whereClauses, CreateWhereClause("r.language", "=", language))
	}
	
	// Add active filter if provided
	if isActive, ok := filters["is_active"].(bool); ok {
		whereClauses = append(whereClauses, CreateWhereClause("r.is_active", "=", isActive))
	}
	
	filter := &QueryFilter{
		WhereClauses: whereClauses,
		OrderBy:      "r.name ASC",
		Limit:        limit,
		Offset:       offset,
	}
	
	// Build and execute query
	query, args := sr.queryBuilder.BuildRepositoryListQuery(orgID, filter)
	
	var repositories []*types.Repository
	err := sr.queryBuilder.ExecuteSecureQuery(ctx, query, args, &repositories)
	if err != nil {
		return nil, 0, err
	}
	
	// Get total count with same filters (without limit/offset)
	countFilter := &QueryFilter{
		WhereClauses: whereClauses,
	}
	countQuery, countArgs := sr.buildRepositoryCountQuery(orgID, countFilter)
	
	var total int
	err = sr.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, errors.NewInternalError("failed to get repository count").WithCause(err)
	}
	
	return repositories, total, nil
}

// UpdateRepository updates a repository with secure parameters
func (sr *SecureRepositories) UpdateRepository(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.NewValidationError("no updates provided")
	}
	
	// Build dynamic UPDATE query with parameterized values
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1
	
	// Add updated_at automatically
	updates["updated_at"] = time.Now()
	
	for field, value := range updates {
		// Validate field names to prevent SQL injection
		if !isValidUpdateField(field) {
			return errors.NewValidationError("invalid field name").WithDetails(map[string]interface{}{
				"field": field,
			})
		}
		
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}
	
	query := fmt.Sprintf(`
		UPDATE repositories 
		SET %s 
		WHERE id = $%d
	`, fmt.Sprintf("%s", setParts), argIndex)
	args = append(args, id)
	
	result, err := sr.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.NewInternalError("failed to update repository").WithCause(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}
	
	if rowsAffected == 0 {
		return errors.NewNotFoundError("repository not found")
	}
	
	return nil
}

// DeleteRepository deletes a repository with secure parameters
func (sr *SecureRepositories) DeleteRepository(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM repositories WHERE id = $1`
	
	result, err := sr.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.NewInternalError("failed to delete repository").WithCause(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("failed to get rows affected").WithCause(err)
	}
	
	if rowsAffected == 0 {
		return errors.NewNotFoundError("repository not found")
	}
	
	return nil
}

// Scan Job operations with secure parameterized queries

// CreateScanJob creates a new scan job with secure parameters
func (sr *SecureRepositories) CreateScanJob(ctx context.Context, scanJob *types.ScanJob) error {
	query := `
		INSERT INTO scan_jobs (id, repository_id, user_id, status, agents, branch, started_at, completed_at, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	
	now := time.Now()
	scanJob.ID = uuid.New()
	scanJob.CreatedAt = now
	scanJob.UpdatedAt = now
	
	_, err := sr.db.ExecContext(ctx, query,
		scanJob.ID,
		scanJob.RepositoryID,
		scanJob.UserID,
		scanJob.Status,
		scanJob.Agents,
		scanJob.Branch,
		scanJob.StartedAt,
		scanJob.CompletedAt,
		scanJob.ErrorMessage,
		scanJob.CreatedAt,
		scanJob.UpdatedAt,
	)
	
	if err != nil {
		return errors.NewInternalError("failed to create scan job").WithCause(err)
	}
	
	return nil
}

// GetScanJob retrieves a scan job by ID with secure parameters
func (sr *SecureRepositories) GetScanJob(ctx context.Context, id uuid.UUID) (*types.ScanJob, error) {
	query := `
		SELECT sj.id, sj.repository_id, sj.user_id, sj.status, sj.agents, sj.branch,
		       sj.started_at, sj.completed_at, sj.error_message, sj.created_at, sj.updated_at,
		       r.name as repository_name, r.url as repository_url
		FROM scan_jobs sj
		JOIN repositories r ON sj.repository_id = r.id
		WHERE sj.id = $1
	`
	
	var scanJob types.ScanJob
	err := sr.db.GetContext(ctx, &scanJob, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("scan job not found")
		}
		return nil, errors.NewInternalError("failed to get scan job").WithCause(err)
	}
	
	return &scanJob, nil
}

// ListScanJobs lists scan jobs with secure filtering and pagination
func (sr *SecureRepositories) ListScanJobs(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*types.ScanJob, int, error) {
	var whereClauses []WhereClause
	
	// Add repository filter if provided
	if repoID, ok := filters["repository_id"].(uuid.UUID); ok {
		whereClauses = append(whereClauses, CreateWhereClause("sj.repository_id", "=", repoID))
	}
	
	// Add user filter if provided
	if userID, ok := filters["user_id"].(uuid.UUID); ok {
		whereClauses = append(whereClauses, CreateWhereClause("sj.user_id", "=", userID))
	}
	
	// Add status filter if provided
	if status, ok := filters["status"].(string); ok && status != "" {
		whereClauses = append(whereClauses, CreateWhereClause("sj.status", "=", status))
	}
	
	// Add agent filter if provided
	if agent, ok := filters["agent"].(string); ok && agent != "" {
		whereClauses = append(whereClauses, CreateWhereClause("sj.agents", "LIKE", agent))
	}
	
	filter := &QueryFilter{
		WhereClauses: whereClauses,
		OrderBy:      "sj.created_at DESC",
		Limit:        limit,
		Offset:       offset,
	}
	
	// Build and execute query
	query, args := sr.queryBuilder.BuildScanJobListQuery(filter)
	
	var scanJobs []*types.ScanJob
	err := sr.queryBuilder.ExecuteSecureQuery(ctx, query, args, &scanJobs)
	if err != nil {
		return nil, 0, err
	}
	
	// Get total count
	countFilter := &QueryFilter{
		WhereClauses: whereClauses,
	}
	countQuery, countArgs := sr.buildScanJobCountQuery(countFilter)
	
	var total int
	err = sr.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, errors.NewInternalError("failed to get scan job count").WithCause(err)
	}
	
	return scanJobs, total, nil
}

// Helper functions

// buildRepositoryCountQuery builds a count query for repositories
func (sr *SecureRepositories) buildRepositoryCountQuery(orgID *uuid.UUID, filter *QueryFilter) (string, []interface{}) {
	baseQuery := "SELECT COUNT(*) FROM repositories r"
	
	var args []interface{}
	var conditions []string
	argIndex := 1
	
	if orgID != nil {
		conditions = append(conditions, fmt.Sprintf("r.organization_id = $%d", argIndex))
		args = append(args, *orgID)
		argIndex++
	}
	
	for _, whereClause := range filter.WhereClauses {
		condition := whereClause.Condition
		for i, arg := range whereClause.Args {
			placeholder := fmt.Sprintf("$%d", i+1)
			actualPlaceholder := fmt.Sprintf("$%d", argIndex)
			condition = fmt.Sprintf(condition, placeholder, actualPlaceholder)
			args = append(args, arg)
			argIndex++
		}
		conditions = append(conditions, condition)
	}
	
	if len(conditions) > 0 {
		baseQuery += " WHERE " + fmt.Sprintf("%s", conditions)
	}
	
	return baseQuery, args
}

// buildScanJobCountQuery builds a count query for scan jobs
func (sr *SecureRepositories) buildScanJobCountQuery(filter *QueryFilter) (string, []interface{}) {
	baseQuery := "SELECT COUNT(*) FROM scan_jobs sj"
	
	var args []interface{}
	var conditions []string
	argIndex := 1
	
	for _, whereClause := range filter.WhereClauses {
		condition := whereClause.Condition
		for i, arg := range whereClause.Args {
			placeholder := fmt.Sprintf("$%d", i+1)
			actualPlaceholder := fmt.Sprintf("$%d", argIndex)
			condition = fmt.Sprintf(condition, placeholder, actualPlaceholder)
			args = append(args, arg)
			argIndex++
		}
		conditions = append(conditions, condition)
	}
	
	if len(conditions) > 0 {
		baseQuery += " WHERE " + fmt.Sprintf("%s", conditions)
	}
	
	return baseQuery, args
}

// isValidUpdateField validates field names for UPDATE queries to prevent SQL injection
func isValidUpdateField(field string) bool {
	validFields := map[string]bool{
		"name":            true,
		"url":             true,
		"description":     true,
		"language":        true,
		"branch":          true,
		"is_active":       true,
		"updated_at":      true,
		"status":          true,
		"started_at":      true,
		"completed_at":    true,
		"error_message":   true,
		"agents":          true,
	}
	
	return validFields[field]
}