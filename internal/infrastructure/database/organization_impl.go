package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/types"
)

// OrganizationRepositoryImpl implements the OrganizationRepository interface
type OrganizationRepositoryImpl struct {
	*BaseRepositoryImpl[types.Organization, uuid.UUID]
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *sqlx.DB) repositories.OrganizationRepository {
	return &OrganizationRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository[types.Organization, uuid.UUID](db, "organizations"),
	}
}

// GetByName retrieves an organization by name
func (o *OrganizationRepositoryImpl) GetByName(ctx context.Context, name string) (*types.Organization, error) {
	query := "SELECT * FROM organizations WHERE name = $1"
	
	var org types.Organization
	err := o.GetDB().GetContext(ctx, &org, query, name)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, errors.NewNotFoundError("organization")
		}
		return nil, errors.NewDatabaseError("get_by_name", "failed to get organization by name").WithCause(err)
	}
	
	return &org, nil
}

// GetUserOrganizations retrieves organizations for a user
func (o *OrganizationRepositoryImpl) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*types.Organization, error) {
	query := `
		SELECT o.* 
		FROM organizations o
		INNER JOIN organization_users ou ON o.id = ou.organization_id
		WHERE ou.user_id = $1 AND ou.is_active = true
		ORDER BY o.name
	`
	
	var orgs []*types.Organization
	err := o.GetDB().SelectContext(ctx, &orgs, query, userID)
	if err != nil {
		return nil, errors.NewDatabaseError("get_user_organizations", "failed to get user organizations").WithCause(err)
	}
	
	return orgs, nil
}

// AddUserToOrganization adds a user to an organization with a role
func (o *OrganizationRepositoryImpl) AddUserToOrganization(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	// Check if user is already in organization
	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM organization_users WHERE organization_id = $1 AND user_id = $2)"
	err := o.GetDB().GetContext(ctx, &exists, checkQuery, orgID, userID)
	if err != nil {
		return errors.NewDatabaseError("add_user", "failed to check user membership").WithCause(err)
	}
	
	if exists {
		// Update existing membership
		updateQuery := `
			UPDATE organization_users 
			SET role = $3, is_active = true, updated_at = NOW()
			WHERE organization_id = $1 AND user_id = $2
		`
		_, err = o.GetDB().ExecContext(ctx, updateQuery, orgID, userID, role)
		if err != nil {
			return errors.NewDatabaseError("add_user", "failed to update user membership").WithCause(err)
		}
	} else {
		// Insert new membership
		insertQuery := `
			INSERT INTO organization_users (id, organization_id, user_id, role, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, true, NOW(), NOW())
		`
		_, err = o.GetDB().ExecContext(ctx, insertQuery, uuid.New(), orgID, userID, role)
		if err != nil {
			return errors.NewDatabaseError("add_user", "failed to add user to organization").WithCause(err)
		}
	}
	
	return nil
}

// RemoveUserFromOrganization removes a user from an organization
func (o *OrganizationRepositoryImpl) RemoveUserFromOrganization(ctx context.Context, orgID, userID uuid.UUID) error {
	query := `
		UPDATE organization_users 
		SET is_active = false, updated_at = NOW()
		WHERE organization_id = $1 AND user_id = $2
	`
	
	result, err := o.GetDB().ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return errors.NewDatabaseError("remove_user", "failed to remove user from organization").WithCause(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError("remove_user", "failed to get rows affected").WithCause(err)
	}
	
	if rowsAffected == 0 {
		return errors.NewNotFoundError("organization membership")
	}
	
	return nil
}

// GetOrganizationUsers retrieves users in an organization
func (o *OrganizationRepositoryImpl) GetOrganizationUsers(ctx context.Context, orgID uuid.UUID) ([]*types.User, error) {
	query := `
		SELECT u.* 
		FROM users u
		INNER JOIN organization_users ou ON u.id = ou.user_id
		WHERE ou.organization_id = $1 AND ou.is_active = true
		ORDER BY u.email
	`
	
	var users []*types.User
	err := o.GetDB().SelectContext(ctx, &users, query, orgID)
	if err != nil {
		return nil, errors.NewDatabaseError("get_organization_users", "failed to get organization users").WithCause(err)
	}
	
	return users, nil
}

// UpdateUserRole updates a user's role in an organization
func (o *OrganizationRepositoryImpl) UpdateUserRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	query := `
		UPDATE organization_users 
		SET role = $3, updated_at = NOW()
		WHERE organization_id = $1 AND user_id = $2 AND is_active = true
	`
	
	result, err := o.GetDB().ExecContext(ctx, query, orgID, userID, role)
	if err != nil {
		return errors.NewDatabaseError("update_user_role", "failed to update user role").WithCause(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError("update_user_role", "failed to get rows affected").WithCause(err)
	}
	
	if rowsAffected == 0 {
		return errors.NewNotFoundError("organization membership")
	}
	
	return nil
}