package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// UserRepositoryImpl implements the UserRepository interface
type UserRepositoryImpl struct {
	db *sqlx.DB
}

// NewUserRepository creates a new user repository implementation
func NewUserRepository(db *sqlx.DB) repositories.UserRepository {
	return &UserRepositoryImpl{
		db: db,
	}
}

// Create creates a new user
func (r *UserRepositoryImpl) Create(ctx context.Context, user *entities.User) error {
	query := `
		INSERT INTO users (id, email, name, avatar_url, supabase_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.AvatarURL,
		user.SupabaseID,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)
	
	return err
}

// GetByID retrieves a user by ID
func (r *UserRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	query := `
		SELECT id, email, name, avatar_url, supabase_id, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	
	var user entities.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("user not found")
		}
		return nil, err
	}
	
	return &user, nil
}

// Update updates an existing user
func (r *UserRepositoryImpl) Update(ctx context.Context, user *entities.User) error {
	query := `
		UPDATE users
		SET email = $2, name = $3, avatar_url = $4, supabase_id = $5, role = $6, updated_at = $7
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.AvatarURL,
		user.SupabaseID,
		user.Role,
		user.UpdatedAt,
	)
	
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("user not found")
	}
	
	return nil
}

// Delete deletes a user by ID
func (r *UserRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("user not found")
	}
	
	return nil
}

// List retrieves users with filtering and pagination
func (r *UserRepositoryImpl) List(ctx context.Context, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.User, int64, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1
	
	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argIndex, argIndex+1)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}
	
	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
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
		SELECT id, email, name, avatar_url, supabase_id, role, created_at, updated_at
		FROM users
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIndex, argIndex+1)
	
	args = append(args, pagination.PageSize, offset)
	
	var users []*entities.User
	err = r.db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		return nil, 0, err
	}
	
	return users, total, nil
}

// Exists checks if a user exists by ID
func (r *UserRepositoryImpl) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id)
	return exists, err
}

// Count returns the total count of users matching the filter
func (r *UserRepositoryImpl) Count(ctx context.Context, filter repositories.Filter) (int64, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1
	
	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argIndex, argIndex+1)
		searchPattern := "%" + filter.Search + "%"
		args = append(args, searchPattern, searchPattern)
	}
	
	query := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	
	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	return count, err
}

// GetByEmail retrieves a user by email address
func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	query := `
		SELECT id, email, name, avatar_url, supabase_id, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	
	var user entities.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("user not found")
		}
		return nil, err
	}
	
	return &user, nil
}

// GetBySupabaseID retrieves a user by Supabase ID
func (r *UserRepositoryImpl) GetBySupabaseID(ctx context.Context, supabaseID string) (*entities.User, error) {
	query := `
		SELECT id, email, name, avatar_url, supabase_id, role, created_at, updated_at
		FROM users
		WHERE supabase_id = $1
	`
	
	var user entities.User
	err := r.db.GetContext(ctx, &user, query, supabaseID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.NewNotFoundError("user not found")
		}
		return nil, err
	}
	
	return &user, nil
}

// UpdateProfile updates user profile information
func (r *UserRepositoryImpl) UpdateProfile(ctx context.Context, userID uuid.UUID, name, avatarURL string) error {
	query := `
		UPDATE users
		SET name = $2, avatar_url = $3, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, userID, name, avatarURL)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("user not found")
	}
	
	return nil
}

// UpdateRole updates user role
func (r *UserRepositoryImpl) UpdateRole(ctx context.Context, userID uuid.UUID, role entities.UserRole) error {
	query := `
		UPDATE users
		SET role = $2, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, userID, role)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("user not found")
	}
	
	return nil
}

// ListByOrganization retrieves users by organization
func (r *UserRepositoryImpl) ListByOrganization(ctx context.Context, orgID uuid.UUID, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.User, int64, error) {
	// For now, we'll return all users since organization membership is not implemented yet
	// In a real implementation, you'd join with an organization_users table
	return r.List(ctx, filter, pagination)
}

// DeactivateUser deactivates a user account
func (r *UserRepositoryImpl) DeactivateUser(ctx context.Context, userID uuid.UUID) error {
	// For now, we'll use a soft delete approach by setting a deactivated_at timestamp
	// In a real implementation, you'd add a deactivated_at column to the users table
	query := `
		UPDATE users
		SET updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("user not found")
	}
	
	return nil
}

// ActivateUser activates a user account
func (r *UserRepositoryImpl) ActivateUser(ctx context.Context, userID uuid.UUID) error {
	// For now, we'll just update the timestamp
	// In a real implementation, you'd clear the deactivated_at timestamp
	query := `
		UPDATE users
		SET updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return entities.NewNotFoundError("user not found")
	}
	
	return nil
}