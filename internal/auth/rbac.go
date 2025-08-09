package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/types"
)

// Permission represents a specific permission in the system
type Permission string

const (
	// Organization permissions
	PermissionOrgRead   Permission = "org:read"
	PermissionOrgWrite  Permission = "org:write"
	PermissionOrgDelete Permission = "org:delete"
	PermissionOrgAdmin  Permission = "org:admin"

	// Repository permissions
	PermissionRepoRead   Permission = "repo:read"
	PermissionRepoWrite  Permission = "repo:write"
	PermissionRepoDelete Permission = "repo:delete"
	PermissionRepoAdmin  Permission = "repo:admin"

	// Scan permissions
	PermissionScanCreate Permission = "scan:create"
	PermissionScanRead   Permission = "scan:read"
	PermissionScanCancel Permission = "scan:cancel"
	PermissionScanRetry  Permission = "scan:retry"

	// Finding permissions
	PermissionFindingRead   Permission = "finding:read"
	PermissionFindingWrite  Permission = "finding:write"
	PermissionFindingExport Permission = "finding:export"

	// User management permissions
	PermissionUserRead   Permission = "user:read"
	PermissionUserWrite  Permission = "user:write"
	PermissionUserDelete Permission = "user:delete"
	PermissionUserInvite Permission = "user:invite"

	// System permissions
	PermissionSystemAdmin Permission = "system:admin"
	PermissionAuditRead   Permission = "audit:read"
)

// RolePermissions defines the permissions for each role
var RolePermissions = map[string][]Permission{
	types.RoleOwner: {
		// Organization permissions
		PermissionOrgRead, PermissionOrgWrite, PermissionOrgDelete, PermissionOrgAdmin,
		// Repository permissions
		PermissionRepoRead, PermissionRepoWrite, PermissionRepoDelete, PermissionRepoAdmin,
		// Scan permissions
		PermissionScanCreate, PermissionScanRead, PermissionScanCancel, PermissionScanRetry,
		// Finding permissions
		PermissionFindingRead, PermissionFindingWrite, PermissionFindingExport,
		// User management permissions
		PermissionUserRead, PermissionUserWrite, PermissionUserDelete, PermissionUserInvite,
		// Audit permissions
		PermissionAuditRead,
	},
	types.RoleAdmin: {
		// Organization permissions (read/write but not delete)
		PermissionOrgRead, PermissionOrgWrite,
		// Repository permissions
		PermissionRepoRead, PermissionRepoWrite, PermissionRepoAdmin,
		// Scan permissions
		PermissionScanCreate, PermissionScanRead, PermissionScanCancel, PermissionScanRetry,
		// Finding permissions
		PermissionFindingRead, PermissionFindingWrite, PermissionFindingExport,
		// User management permissions (limited)
		PermissionUserRead, PermissionUserInvite,
		// Audit permissions
		PermissionAuditRead,
	},
	types.RoleMember: {
		// Organization permissions (read only)
		PermissionOrgRead,
		// Repository permissions (read/write but not admin)
		PermissionRepoRead, PermissionRepoWrite,
		// Scan permissions
		PermissionScanCreate, PermissionScanRead, PermissionScanCancel,
		// Finding permissions
		PermissionFindingRead, PermissionFindingWrite,
		// User permissions (read only)
		PermissionUserRead,
	},
}

// RBACService provides role-based access control functionality
type RBACService struct {
	repos *database.Repositories
}

// NewRBACService creates a new RBAC service
func NewRBACService(repos *database.Repositories) *RBACService {
	return &RBACService{
		repos: repos,
	}
}

// CheckPermission checks if a user has a specific permission for a resource
func (s *RBACService) CheckPermission(ctx context.Context, userID uuid.UUID, permission Permission, resourceType string, resourceID *uuid.UUID) (bool, error) {
	switch resourceType {
	case "organization":
		if resourceID == nil {
			return false, fmt.Errorf("organization ID is required")
		}
		return s.checkOrganizationPermission(ctx, userID, *resourceID, permission)
	case "repository":
		if resourceID == nil {
			return false, fmt.Errorf("repository ID is required")
		}
		return s.checkRepositoryPermission(ctx, userID, *resourceID, permission)
	case "system":
		return s.checkSystemPermission(ctx, userID, permission)
	default:
		return false, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// checkOrganizationPermission checks if a user has permission for an organization
func (s *RBACService) checkOrganizationPermission(ctx context.Context, userID, orgID uuid.UUID, permission Permission) (bool, error) {
	// Get user's role in the organization
	member, err := s.getOrganizationMember(ctx, userID, orgID)
	if err != nil {
		return false, err
	}
	if member == nil {
		return false, nil // User is not a member of the organization
	}

	// Check if the role has the required permission
	return s.roleHasPermission(member.Role, permission), nil
}

// checkRepositoryPermission checks if a user has permission for a repository
func (s *RBACService) checkRepositoryPermission(ctx context.Context, userID, repoID uuid.UUID, permission Permission) (bool, error) {
	// Get repository to find its organization
	repo, err := s.getRepository(ctx, repoID)
	if err != nil {
		return false, err
	}

	// Check organization-level permission first
	hasOrgPermission, err := s.checkOrganizationPermission(ctx, userID, repo.OrganizationID, permission)
	if err != nil {
		return false, err
	}
	if hasOrgPermission {
		return true, nil
	}

	// TODO: Check repository-specific permissions if needed
	// For now, repository permissions are inherited from organization

	return false, nil
}

// checkSystemPermission checks if a user has system-level permissions
func (s *RBACService) checkSystemPermission(ctx context.Context, userID uuid.UUID, permission Permission) (bool, error) {
	// TODO: Implement system-level permissions
	// For now, only system admins have system permissions
	// This could be stored in a separate system_admins table or user flags

	// Check if user is a system admin (placeholder implementation)
	user, err := s.repos.Users.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}

	// For now, we'll use email domain or a specific flag
	// In a real implementation, this would be stored in the database
	if strings.HasSuffix(user.Email, "@agentscan.com") {
		return true, nil
	}

	return false, nil
}

// roleHasPermission checks if a role has a specific permission
func (s *RBACService) roleHasPermission(role string, permission Permission) bool {
	permissions, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetUserOrganizations returns all organizations a user is a member of
func (s *RBACService) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*types.Organization, error) {
	// TODO: Implement this method
	// This would query the organization_members table and join with organizations
	return []*types.Organization{}, nil
}

// GetOrganizationMembers returns all members of an organization
func (s *RBACService) GetOrganizationMembers(ctx context.Context, orgID uuid.UUID) ([]*types.OrganizationMember, error) {
	// TODO: Implement this method
	// This would query the organization_members table
	return []*types.OrganizationMember{}, nil
}

// AddOrganizationMember adds a user to an organization with a specific role
func (s *RBACService) AddOrganizationMember(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	// Validate role
	if !s.isValidRole(role) {
		return fmt.Errorf("invalid role: %s", role)
	}

	// TODO: Implement this method
	// This would insert into the organization_members table
	return nil
}

// RemoveOrganizationMember removes a user from an organization
func (s *RBACService) RemoveOrganizationMember(ctx context.Context, orgID, userID uuid.UUID) error {
	// TODO: Implement this method
	// This would delete from the organization_members table
	return nil
}

// UpdateOrganizationMemberRole updates a user's role in an organization
func (s *RBACService) UpdateOrganizationMemberRole(ctx context.Context, orgID, userID uuid.UUID, newRole string) error {
	// Validate role
	if !s.isValidRole(newRole) {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	// TODO: Implement this method
	// This would update the organization_members table
	return nil
}

// isValidRole checks if a role is valid
func (s *RBACService) isValidRole(role string) bool {
	validRoles := []string{types.RoleOwner, types.RoleAdmin, types.RoleMember}
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

// getOrganizationMember gets a user's membership in an organization
func (s *RBACService) getOrganizationMember(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
	// TODO: Implement this method
	// This would query the organization_members table
	// For now, return nil to indicate no membership
	return nil, nil
}

// getRepository gets a repository by ID
func (s *RBACService) getRepository(ctx context.Context, repoID uuid.UUID) (*types.Repository, error) {
	// TODO: Implement this method using the repository service
	// For now, return a mock repository
	return &types.Repository{
		ID:             repoID,
		OrganizationID: uuid.New(),
		Name:           "mock-repo",
	}, nil
}

// PermissionContext represents the context for permission checking
type PermissionContext struct {
	UserID       uuid.UUID
	ResourceType string
	ResourceID   *uuid.UUID
	Permission   Permission
}

// RequirePermission is a helper function for middleware to check permissions
func (s *RBACService) RequirePermission(ctx context.Context, permCtx PermissionContext) error {
	hasPermission, err := s.CheckPermission(ctx, permCtx.UserID, permCtx.Permission, permCtx.ResourceType, permCtx.ResourceID)
	if err != nil {
		return fmt.Errorf("failed to check permission: %w", err)
	}

	if !hasPermission {
		return fmt.Errorf("insufficient permissions: user %s does not have %s permission for %s", 
			permCtx.UserID, permCtx.Permission, permCtx.ResourceType)
	}

	return nil
}

// GetUserPermissions returns all permissions a user has for a specific resource
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uuid.UUID, resourceType string, resourceID *uuid.UUID) ([]Permission, error) {
	var permissions []Permission

	// Get user's role for the resource
	var role string
	switch resourceType {
	case "organization":
		if resourceID == nil {
			return nil, fmt.Errorf("organization ID is required")
		}
		member, err := s.getOrganizationMember(ctx, userID, *resourceID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			return permissions, nil // No permissions if not a member
		}
		role = member.Role
	case "repository":
		if resourceID == nil {
			return nil, fmt.Errorf("repository ID is required")
		}
		repo, err := s.getRepository(ctx, *resourceID)
		if err != nil {
			return nil, err
		}
		member, err := s.getOrganizationMember(ctx, userID, repo.OrganizationID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			return permissions, nil // No permissions if not a member
		}
		role = member.Role
	case "system":
		// Check if user has system permissions
		hasSystemPermission, err := s.checkSystemPermission(ctx, userID, PermissionSystemAdmin)
		if err != nil {
			return nil, err
		}
		if hasSystemPermission {
			permissions = append(permissions, PermissionSystemAdmin)
		}
		return permissions, nil
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	// Get permissions for the role
	rolePermissions, exists := RolePermissions[role]
	if exists {
		permissions = append(permissions, rolePermissions...)
	}

	return permissions, nil
}