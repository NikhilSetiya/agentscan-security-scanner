package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// Permission represents a specific permission
type Permission string

const (
	// Organization permissions
	PermissionOrgRead   Permission = "org:read"
	PermissionOrgWrite  Permission = "org:write"
	PermissionOrgAdmin  Permission = "org:admin"
	PermissionOrgDelete Permission = "org:delete"

	// Repository permissions
	PermissionRepoRead   Permission = "repo:read"
	PermissionRepoWrite  Permission = "repo:write"
	PermissionRepoAdmin  Permission = "repo:admin"
	PermissionRepoDelete Permission = "repo:delete"

	// Scan permissions
	PermissionScanCreate Permission = "scan:create"
	PermissionScanRead   Permission = "scan:read"
	PermissionScanCancel Permission = "scan:cancel"
	PermissionScanRetry  Permission = "scan:retry"

	// Finding permissions
	PermissionFindingRead   Permission = "finding:read"
	PermissionFindingWrite  Permission = "finding:write"
	PermissionFindingExport Permission = "finding:export"

	// User permissions
	PermissionUserRead  Permission = "user:read"
	PermissionUserWrite Permission = "user:write"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[string][]Permission{
	types.RoleOwner: {
		PermissionOrgRead, PermissionOrgWrite, PermissionOrgAdmin, PermissionOrgDelete,
		PermissionRepoRead, PermissionRepoWrite, PermissionRepoAdmin, PermissionRepoDelete,
		PermissionScanCreate, PermissionScanRead, PermissionScanCancel, PermissionScanRetry,
		PermissionFindingRead, PermissionFindingWrite, PermissionFindingExport,
		PermissionUserRead, PermissionUserWrite,
	},
	types.RoleAdmin: {
		PermissionOrgRead, PermissionOrgWrite,
		PermissionRepoRead, PermissionRepoWrite, PermissionRepoAdmin,
		PermissionScanCreate, PermissionScanRead, PermissionScanCancel, PermissionScanRetry,
		PermissionFindingRead, PermissionFindingWrite, PermissionFindingExport,
		PermissionUserRead,
	},
	types.RoleMember: {
		PermissionOrgRead,
		PermissionRepoRead,
		PermissionScanCreate, PermissionScanRead,
		PermissionFindingRead,
		PermissionUserRead,
	},
}

// RBACService handles role-based access control
type RBACService struct {
	repos *database.Repositories
}

// NewRBACService creates a new RBAC service
func NewRBACService(repos *database.Repositories) *RBACService {
	return &RBACService{
		repos: repos,
	}
}

// UserPermissions represents a user's permissions in an organization
type UserPermissions struct {
	UserID         uuid.UUID    `json:"user_id"`
	OrganizationID uuid.UUID    `json:"organization_id"`
	Role           string       `json:"role"`
	Permissions    []Permission `json:"permissions"`
}

// GetUserPermissions retrieves a user's permissions for an organization
func (r *RBACService) GetUserPermissions(ctx context.Context, userID, orgID uuid.UUID) (*UserPermissions, error) {
	// Get user's organization membership
	member, err := r.getUserOrgMembership(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	if member == nil {
		// User is not a member of this organization
		return &UserPermissions{
			UserID:         userID,
			OrganizationID: orgID,
			Role:           "",
			Permissions:    []Permission{},
		}, nil
	}

	permissions := RolePermissions[member.Role]
	if permissions == nil {
		permissions = []Permission{}
	}

	return &UserPermissions{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           member.Role,
		Permissions:    permissions,
	}, nil
}

// HasPermission checks if a user has a specific permission in an organization
func (r *RBACService) HasPermission(ctx context.Context, userID, orgID uuid.UUID, permission Permission) (bool, error) {
	userPerms, err := r.GetUserPermissions(ctx, userID, orgID)
	if err != nil {
		return false, err
	}

	for _, perm := range userPerms.Permissions {
		if perm == permission {
			return true, nil
		}
	}

	return false, nil
}

// HasAnyPermission checks if a user has any of the specified permissions
func (r *RBACService) HasAnyPermission(ctx context.Context, userID, orgID uuid.UUID, permissions ...Permission) (bool, error) {
	for _, permission := range permissions {
		hasPermission, err := r.HasPermission(ctx, userID, orgID, permission)
		if err != nil {
			return false, err
		}
		if hasPermission {
			return true, nil
		}
	}
	return false, nil
}

// RequirePermission middleware that requires a specific permission
func (r *RBACService) RequirePermission(permission Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			UnauthorizedResponse(c, "User authentication required")
			c.Abort()
			return
		}

		// Get organization ID from request (could be in path, query, or body)
		orgID := r.extractOrganizationID(c)
		if orgID == uuid.Nil {
			BadRequestResponse(c, "Organization ID is required")
			c.Abort()
			return
		}

		hasPermission, err := r.HasPermission(c.Request.Context(), userID, orgID, permission)
		if err != nil {
			InternalErrorResponse(c, "Failed to check permissions")
			c.Abort()
			return
		}

		if !hasPermission {
			ForbiddenResponse(c, "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission middleware that requires any of the specified permissions
func (r *RBACService) RequireAnyPermission(permissions ...Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			UnauthorizedResponse(c, "User authentication required")
			c.Abort()
			return
		}

		orgID := r.extractOrganizationID(c)
		if orgID == uuid.Nil {
			BadRequestResponse(c, "Organization ID is required")
			c.Abort()
			return
		}

		hasPermission, err := r.HasAnyPermission(c.Request.Context(), userID, orgID, permissions...)
		if err != nil {
			InternalErrorResponse(c, "Failed to check permissions")
			c.Abort()
			return
		}

		if !hasPermission {
			ForbiddenResponse(c, "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole middleware that requires a specific role
func (r *RBACService) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetCurrentUserID(c)
		if !exists {
			UnauthorizedResponse(c, "User authentication required")
			c.Abort()
			return
		}

		orgID := r.extractOrganizationID(c)
		if orgID == uuid.Nil {
			BadRequestResponse(c, "Organization ID is required")
			c.Abort()
			return
		}

		userPerms, err := r.GetUserPermissions(c.Request.Context(), userID, orgID)
		if err != nil {
			InternalErrorResponse(c, "Failed to check permissions")
			c.Abort()
			return
		}

		if userPerms.Role != role {
			ForbiddenResponse(c, "Insufficient role")
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractOrganizationID extracts organization ID from the request
func (r *RBACService) extractOrganizationID(c *gin.Context) uuid.UUID {
	// Try to get from path parameter
	if orgIDStr := c.Param("org_id"); orgIDStr != "" {
		if orgID, err := uuid.Parse(orgIDStr); err == nil {
			return orgID
		}
	}

	// Try to get from query parameter
	if orgIDStr := c.Query("org_id"); orgIDStr != "" {
		if orgID, err := uuid.Parse(orgIDStr); err == nil {
			return orgID
		}
	}

	// Try to get from header
	if orgIDStr := c.GetHeader("X-Organization-ID"); orgIDStr != "" {
		if orgID, err := uuid.Parse(orgIDStr); err == nil {
			return orgID
		}
	}

	return uuid.Nil
}

// getUserOrgMembership gets a user's organization membership
func (r *RBACService) getUserOrgMembership(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
	// TODO: Implement this method in the database layer
	// For testing, return nil to simulate no membership (should deny access)
	return nil, nil
}

// IsOwner checks if a user is an owner of an organization
func (r *RBACService) IsOwner(ctx context.Context, userID, orgID uuid.UUID) (bool, error) {
	userPerms, err := r.GetUserPermissions(ctx, userID, orgID)
	if err != nil {
		return false, err
	}
	return userPerms.Role == types.RoleOwner, nil
}

// IsAdmin checks if a user is an admin of an organization
func (r *RBACService) IsAdmin(ctx context.Context, userID, orgID uuid.UUID) (bool, error) {
	userPerms, err := r.GetUserPermissions(ctx, userID, orgID)
	if err != nil {
		return false, err
	}
	return userPerms.Role == types.RoleAdmin || userPerms.Role == types.RoleOwner, nil
}

// IsMember checks if a user is a member of an organization
func (r *RBACService) IsMember(ctx context.Context, userID, orgID uuid.UUID) (bool, error) {
	userPerms, err := r.GetUserPermissions(ctx, userID, orgID)
	if err != nil {
		return false, err
	}
	return userPerms.Role != "", nil
}

// GetUserRole gets a user's role in an organization
func (r *RBACService) GetUserRole(ctx context.Context, userID, orgID uuid.UUID) (string, error) {
	userPerms, err := r.GetUserPermissions(ctx, userID, orgID)
	if err != nil {
		return "", err
	}
	return userPerms.Role, nil
}

// PermissionToString converts a permission to a string
func (p Permission) String() string {
	return string(p)
}

// ParsePermission parses a string into a permission
func ParsePermission(s string) Permission {
	return Permission(s)
}

// HasPermissionString checks if a role has a specific permission (string version)
func HasPermissionString(role, permission string) bool {
	permissions := RolePermissions[role]
	for _, perm := range permissions {
		if perm.String() == permission {
			return true
		}
	}
	return false
}

// GetRolePermissions returns all permissions for a role
func GetRolePermissions(role string) []Permission {
	return RolePermissions[role]
}

// IsValidRole checks if a role is valid
func IsValidRole(role string) bool {
	validRoles := []string{types.RoleOwner, types.RoleAdmin, types.RoleMember}
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

// GetHigherRoles returns roles that are higher than the given role
func GetHigherRoles(role string) []string {
	switch role {
	case types.RoleMember:
		return []string{types.RoleAdmin, types.RoleOwner}
	case types.RoleAdmin:
		return []string{types.RoleOwner}
	case types.RoleOwner:
		return []string{}
	default:
		return []string{types.RoleOwner, types.RoleAdmin, types.RoleMember}
	}
}

// CanManageRole checks if a user with one role can manage another role
func CanManageRole(managerRole, targetRole string) bool {
	higherRoles := GetHigherRoles(targetRole)
	for _, higherRole := range higherRoles {
		if managerRole == higherRole {
			return true
		}
	}
	return managerRole == targetRole // Can manage same role
}