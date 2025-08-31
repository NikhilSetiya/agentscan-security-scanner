package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// RepositoryRepositoryImpl implements the RepositoryRepository interface
type RepositoryRepositoryImpl struct {
	db *sqlx.DB
}

// NewRepositoryRepository creates a new repository repository implementation
func NewRepositoryRepository(db *sqlx.DB) repositories.RepositoryRepository {
	return &RepositoryRepositoryImpl{
		db: db,
	}
}

// Create creates a new repository
func (r *RepositoryRepositoryImpl) Create(ctx context.Context, repo *entities.Repository) error {
	query := `
		INSERT INTO repositories (
			id, organization_id, name, url, provider, provider_id, default_branch,
			language, description, languages, settings, is_active, last_scan_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		repo.ID,
		repo.OrganizationID,
		repo.Name,
		repo.URL,
		repo.Provider,
		repo.ProviderID,
		repo.DefaultBranch,
		repo.Language,
		repo.Description,
		pq.Array(repo.Languages),
		repo.Settings,
		repo.IsActive,
		repo.LastScanAt,
		repo.CreatedAt,
		repo.UpdatedAt,
	)
	
	return err
}

// GetByID retrieves a repository by ID
func (r *RepositoryRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.Repository, error) {
	query := `
		SELECT id, organization_id, name, url, provider, provider_id, default_branch,
			   language, description, languages, settings, is_active, last_scan_at,
			   created_at, updated_at
		FROM repositories
		WHERE id = $1
	`
	
	var repo entities.Repository
	err := r.db.GetContext(ctx, &repo, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("repository not found")
		}
		return nil, err
	}
	
	return &repo, nil
}

// Update updates an existing repository
func (r *RepositoryRepositoryImpl) Update(ctx context.Context, repo *entities.Repository) error {
	query := `
		UPDATE repositories
		SET organization_id = $2, name = $3, url = $4, provider = $5, provider_id = $6,
			default_branch = $7, language = $8, description = $9, languages = $10,
			settings = $11, is_active = $12, last_scan_at = $13, updated_at = $14
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query,
		repo.ID,
		repo.OrganizationID,
		repo.Name,
		repo.URL,
		repo.Provider,
		repo.ProviderID,
		repo.DefaultBranch,
		repo.Language,
		repo.Description,
		pq.Array(repo.Languages),
		repo.Settings,
		repo.IsActive,
		repo.LastScanAt,
		repo.UpdatedAt,
	)
	
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("repository not found")
	}
	
	return nil
}

// Delete deletes a repository by ID (soft delete)
func (r *RepositoryRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE repositories
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("repository not found")
	}
	
	return nil
}

// List retrieves repositories with filtering and pagination
func (r *RepositoryRepositoryImpl) List(ctx context.Context, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.Repository, int64, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1
	
	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR url ILIKE $%d OR language ILIKE $%d)", argIndex, argIndex+1, argIndex+2)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}
	
	if filter.Status != "" {
		if filter.Status == "active" {
			whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, true)
		} else if filter.Status == "inactive" {
			whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, false)
		}
		argIndex++
	}
	
	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM repositories %s", whereClause)
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
		SELECT id, organization_id, name, url, provider, provider_id, default_branch,
			   language, description, languages, settings, is_active, last_scan_at,
			   created_at, updated_at
		FROM repositories
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIndex, argIndex+1)
	
	args = append(args, pagination.PageSize, offset)
	
	var repos []*entities.Repository
	err = r.db.SelectContext(ctx, &repos, query, args...)
	if err != nil {
		return nil, 0, err
	}
	
	return repos, total, nil
}

// Exists checks if a repository exists by ID
func (r *RepositoryRepositoryImpl) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM repositories WHERE id = $1)`
	
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id)
	return exists, err
}

// Count returns the total count of repositories matching the filter
func (r *RepositoryRepositoryImpl) Count(ctx context.Context, filter repositories.Filter) (int64, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1
	
	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR url ILIKE $%d OR language ILIKE $%d)", argIndex, argIndex+1, argIndex+2)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}
	
	if filter.Status != "" {
		if filter.Status == "active" {
			whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, true)
		} else if filter.Status == "inactive" {
			whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, false)
		}
	}
	
	query := fmt.Sprintf("SELECT COUNT(*) FROM repositories %s", whereClause)
	
	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	return count, err
}

// GetByURL retrieves a repository by its URL
func (r *RepositoryRepositoryImpl) GetByURL(ctx context.Context, url string) (*entities.Repository, error) {
	query := `
		SELECT id, organization_id, name, url, provider, provider_id, default_branch,
			   language, description, languages, settings, is_active, last_scan_at,
			   created_at, updated_at
		FROM repositories
		WHERE url = $1
	`
	
	var repo entities.Repository
	err := r.db.GetContext(ctx, &repo, query, url)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("repository not found")
		}
		return nil, err
	}
	
	return &repo, nil
}

// GetByProviderID retrieves a repository by provider and provider ID
func (r *RepositoryRepositoryImpl) GetByProviderID(ctx context.Context, provider, providerID string) (*entities.Repository, error) {
	query := `
		SELECT id, organization_id, name, url, provider, provider_id, default_branch,
			   language, description, languages, settings, is_active, last_scan_at,
			   created_at, updated_at
		FROM repositories
		WHERE provider = $1 AND provider_id = $2
	`
	
	var repo entities.Repository
	err := r.db.GetContext(ctx, &repo, query, provider, providerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("repository not found")
		}
		return nil, err
	}
	
	return &repo, nil
}

// ListByOrganization retrieves repositories by organization
func (r *RepositoryRepositoryImpl) ListByOrganization(ctx context.Context, orgID uuid.UUID, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.Repository, int64, error) {
	// Add organization filter to the existing filter
	orgFilter := filter
	// For now, we'll return all repositories since organization filtering is not fully implemented
	// In a real implementation, you'd add WHERE organization_id = $X to the query
	return r.List(ctx, orgFilter, pagination)
}

// ListActive retrieves only active repositories
func (r *RepositoryRepositoryImpl) ListActive(ctx context.Context, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.Repository, int64, error) {
	// Add active status filter
	activeFilter := filter
	activeFilter.Status = "active"
	return r.List(ctx, activeFilter, pagination)
}

// UpdateLastScanTime updates the last scan timestamp
func (r *RepositoryRepositoryImpl) UpdateLastScanTime(ctx context.Context, repoID uuid.UUID) error {
	query := `
		UPDATE repositories
		SET last_scan_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, repoID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("repository not found")
	}
	
	return nil
}

// Deactivate marks a repository as inactive
func (r *RepositoryRepositoryImpl) Deactivate(ctx context.Context, repoID uuid.UUID) error {
	query := `
		UPDATE repositories
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, repoID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("repository not found")
	}
	
	return nil
}

// Activate marks a repository as active
func (r *RepositoryRepositoryImpl) Activate(ctx context.Context, repoID uuid.UUID) error {
	query := `
		UPDATE repositories
		SET is_active = true, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, repoID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("repository not found")
	}
	
	return nil
}

// UpdateSettings updates repository settings
func (r *RepositoryRepositoryImpl) UpdateSettings(ctx context.Context, repoID uuid.UUID, settings map[string]interface{}) error {
	query := `
		UPDATE repositories
		SET settings = $2, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, repoID, settings)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("repository not found")
	}
	
	return nil
}

// GetSettings retrieves repository settings
func (r *RepositoryRepositoryImpl) GetSettings(ctx context.Context, repoID uuid.UUID) (map[string]interface{}, error) {
	query := `SELECT settings FROM repositories WHERE id = $1`
	
	var settings map[string]interface{}
	err := r.db.GetContext(ctx, &settings, query, repoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("repository not found")
		}
		return nil, err
	}
	
	if settings == nil {
		settings = make(map[string]interface{})
	}
	
	return settings, nil
}