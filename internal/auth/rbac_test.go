package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

func setupRBACService() (*RBACService, *MockRepositories) {
	mockRepos := &MockRepositories{
		Users: &MockUserRepository{},
	}

	repos := &database.Repositories{
		Users: mockRepos.Users,
	}

	service := NewRBACService(repos)
	return service, mockRepos
}

func TestRBACService_CheckPermission_Organization(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	tests := []struct {
		name       string
		permission Permission
		role       string
		expected   bool
	}{
		{
			name:       "owner has org read permission",
			permission: PermissionOrgRead,
			role:       types.RoleOwner,
			expected:   true,
		},
		{
			name:       "admin has org read permission",
			permission: PermissionOrgRead,
			role:       types.RoleAdmin,
			expected:   true,
		},
		{
			name:       "member has org read permission",
			permission: PermissionOrgRead,
			role:       types.RoleMember,
			expected:   true,
		},
		{
			name:       "owner has org delete permission",
			permission: PermissionOrgDelete,
			role:       types.RoleOwner,
			expected:   true,
		},
		{
			name:       "admin does not have org delete permission",
			permission: PermissionOrgDelete,
			role:       types.RoleAdmin,
			expected:   false,
		},
		{
			name:       "member does not have org delete permission",
			permission: PermissionOrgDelete,
			role:       types.RoleMember,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the organization member lookup
			service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
				return &types.OrganizationMember{
					ID:             uuid.New(),
					OrganizationID: orgID,
					UserID:         userID,
					Role:           tt.role,
				}, nil
			}

			hasPermission, err := service.CheckPermission(ctx, userID, tt.permission, "organization", &orgID)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, hasPermission)
		})
	}
}

func TestRBACService_CheckPermission_Repository(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	repoID := uuid.New()
	orgID := uuid.New()

	// Mock the repository lookup
	service.getRepository = func(ctx context.Context, repoID uuid.UUID) (*types.Repository, error) {
		return &types.Repository{
			ID:             repoID,
			OrganizationID: orgID,
			Name:           "test-repo",
		}, nil
	}

	tests := []struct {
		name       string
		permission Permission
		role       string
		expected   bool
	}{
		{
			name:       "owner has repo read permission",
			permission: PermissionRepoRead,
			role:       types.RoleOwner,
			expected:   true,
		},
		{
			name:       "admin has repo admin permission",
			permission: PermissionRepoAdmin,
			role:       types.RoleAdmin,
			expected:   true,
		},
		{
			name:       "member has repo read permission",
			permission: PermissionRepoRead,
			role:       types.RoleMember,
			expected:   true,
		},
		{
			name:       "member does not have repo delete permission",
			permission: PermissionRepoDelete,
			role:       types.RoleMember,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the organization member lookup
			service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
				return &types.OrganizationMember{
					ID:             uuid.New(),
					OrganizationID: orgID,
					UserID:         userID,
					Role:           tt.role,
				}, nil
			}

			hasPermission, err := service.CheckPermission(ctx, userID, tt.permission, "repository", &repoID)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, hasPermission)
		})
	}
}

func TestRBACService_CheckPermission_System(t *testing.T) {
	service, mockRepos := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()

	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "agentscan.com email has system permission",
			email:    "admin@agentscan.com",
			expected: true,
		},
		{
			name:     "external email does not have system permission",
			email:    "user@example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &types.User{
				ID:    userID,
				Email: tt.email,
				Name:  "Test User",
			}

			mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil).Once()

			hasPermission, err := service.CheckPermission(ctx, userID, PermissionSystemAdmin, "system", nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, hasPermission)

			mockRepos.Users.AssertExpectations(t)
		})
	}
}

func TestRBACService_CheckPermission_NoMembership(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock no organization membership
	service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
		return nil, nil // No membership
	}

	hasPermission, err := service.CheckPermission(ctx, userID, PermissionOrgRead, "organization", &orgID)
	require.NoError(t, err)
	assert.False(t, hasPermission)
}

func TestRBACService_CheckPermission_InvalidResourceType(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()

	hasPermission, err := service.CheckPermission(ctx, userID, PermissionOrgRead, "invalid", &resourceID)
	assert.Error(t, err)
	assert.False(t, hasPermission)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestRBACService_CheckPermission_MissingResourceID(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()

	hasPermission, err := service.CheckPermission(ctx, userID, PermissionOrgRead, "organization", nil)
	assert.Error(t, err)
	assert.False(t, hasPermission)
	assert.Contains(t, err.Error(), "organization ID is required")
}

func TestRBACService_roleHasPermission(t *testing.T) {
	service, _ := setupRBACService()

	tests := []struct {
		name       string
		role       string
		permission Permission
		expected   bool
	}{
		{
			name:       "owner has org delete permission",
			role:       types.RoleOwner,
			permission: PermissionOrgDelete,
			expected:   true,
		},
		{
			name:       "admin does not have org delete permission",
			role:       types.RoleAdmin,
			permission: PermissionOrgDelete,
			expected:   false,
		},
		{
			name:       "member has org read permission",
			role:       types.RoleMember,
			permission: PermissionOrgRead,
			expected:   true,
		},
		{
			name:       "invalid role has no permissions",
			role:       "invalid",
			permission: PermissionOrgRead,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasPermission := service.roleHasPermission(tt.role, tt.permission)
			assert.Equal(t, tt.expected, hasPermission)
		})
	}
}

func TestRBACService_RequirePermission(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock successful permission check
	service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
		return &types.OrganizationMember{
			ID:             uuid.New(),
			OrganizationID: orgID,
			UserID:         userID,
			Role:           types.RoleOwner,
		}, nil
	}

	permCtx := PermissionContext{
		UserID:       userID,
		ResourceType: "organization",
		ResourceID:   &orgID,
		Permission:   PermissionOrgRead,
	}

	err := service.RequirePermission(ctx, permCtx)
	assert.NoError(t, err)
}

func TestRBACService_RequirePermission_InsufficientPermissions(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock no membership (insufficient permissions)
	service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
		return nil, nil // No membership
	}

	permCtx := PermissionContext{
		UserID:       userID,
		ResourceType: "organization",
		ResourceID:   &orgID,
		Permission:   PermissionOrgRead,
	}

	err := service.RequirePermission(ctx, permCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
}

func TestRBACService_GetUserPermissions(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock organization membership
	service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
		return &types.OrganizationMember{
			ID:             uuid.New(),
			OrganizationID: orgID,
			UserID:         userID,
			Role:           types.RoleAdmin,
		}, nil
	}

	permissions, err := service.GetUserPermissions(ctx, userID, "organization", &orgID)
	require.NoError(t, err)

	// Admin should have specific permissions but not org delete
	assert.Contains(t, permissions, PermissionOrgRead)
	assert.Contains(t, permissions, PermissionOrgWrite)
	assert.NotContains(t, permissions, PermissionOrgDelete)
}

func TestRBACService_GetUserPermissions_NoMembership(t *testing.T) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock no membership
	service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
		return nil, nil // No membership
	}

	permissions, err := service.GetUserPermissions(ctx, userID, "organization", &orgID)
	require.NoError(t, err)
	assert.Empty(t, permissions)
}

// Benchmark tests
func BenchmarkRBACService_CheckPermission(b *testing.B) {
	service, _ := setupRBACService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock organization membership
	service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
		return &types.OrganizationMember{
			ID:             uuid.New(),
			OrganizationID: orgID,
			UserID:         userID,
			Role:           types.RoleAdmin,
		}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.CheckPermission(ctx, userID, PermissionOrgRead, "organization", &orgID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRBACService_roleHasPermission(b *testing.B) {
	service, _ := setupRBACService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.roleHasPermission(types.RoleAdmin, PermissionOrgRead)
	}
}