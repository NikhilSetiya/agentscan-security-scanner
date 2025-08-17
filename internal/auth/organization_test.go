package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// MockRBACService is a mock implementation of RBACService
type MockRBACService struct {
	mock.Mock
}

func (m *MockRBACService) CheckPermission(ctx context.Context, userID uuid.UUID, permission Permission, resourceType string, resourceID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, userID, permission, resourceType, resourceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRBACService) RequirePermission(ctx context.Context, permCtx PermissionContext) error {
	args := m.Called(ctx, permCtx)
	return args.Error(0)
}

func (m *MockRBACService) GetUserPermissions(ctx context.Context, userID uuid.UUID, resourceType string, resourceID *uuid.UUID) ([]Permission, error) {
	args := m.Called(ctx, userID, resourceType, resourceID)
	return args.Get(0).([]Permission), args.Error(1)
}

// MockOrganizationRepository is a mock implementation of OrganizationRepository
type MockOrganizationRepository struct {
	mock.Mock
}

func (m *MockOrganizationRepository) CreateOrganization(ctx context.Context, org *types.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationRepository) GetOrganization(ctx context.Context, orgID uuid.UUID) (*types.Organization, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) UpdateOrganization(ctx context.Context, org *types.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrganizationRepository) DeleteOrganization(ctx context.Context, orgID uuid.UUID) error {
	args := m.Called(ctx, orgID)
	return args.Error(0)
}

func (m *MockOrganizationRepository) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*types.Organization, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*types.Organization), args.Error(1)
}

func (m *MockOrganizationRepository) GetOrganizationMembers(ctx context.Context, orgID uuid.UUID) ([]*types.OrganizationMember, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationRepository) GetOrganizationMember(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationRepository) AddOrganizationMember(ctx context.Context, member *types.OrganizationMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockOrganizationRepository) RemoveOrganizationMember(ctx context.Context, orgID, userID uuid.UUID) error {
	args := m.Called(ctx, orgID, userID)
	return args.Error(0)
}

func (m *MockOrganizationRepository) UpdateOrganizationMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	args := m.Called(ctx, orgID, userID, role)
	return args.Error(0)
}

func (m *MockOrganizationRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

func setupOrganizationService() (*OrganizationService, *MockRepositories, *MockRBACService, *MockOrganizationRepository) {
	mockRepos := &MockRepositories{
		Users: &MockUserRepository{},
	}

	repos := &database.Repositories{
		Users: mockRepos.Users,
	}

	mockRBAC := &MockRBACService{}
	mockOrgRepo := &MockOrganizationRepository{}
	service := NewOrganizationService(repos, mockRBAC, mockOrgRepo)
	return service, mockRepos, mockRBAC, mockOrgRepo
}

func TestOrganizationService_CreateOrganization(t *testing.T) {
	service, _, _, mockOrgRepo := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	req := &CreateOrganizationRequest{
		Name: "Test Organization",
		Slug: "test-org",
	}

	// Mock slug availability check
	mockOrgRepo.On("SlugExists", ctx, req.Slug).Return(false, nil)

	// Mock organization creation
	mockOrgRepo.On("CreateOrganization", ctx, mock.AnythingOfType("*types.Organization")).Return(nil)

	// Mock member addition
	mockOrgRepo.On("AddOrganizationMember", ctx, mock.AnythingOfType("*types.OrganizationMember")).Return(nil)

	org, err := service.CreateOrganization(ctx, userID, req)

	require.NoError(t, err)
	assert.Equal(t, req.Name, org.Name)
	assert.Equal(t, req.Slug, org.Slug)
	assert.NotEqual(t, uuid.Nil, org.ID)

	mockOrgRepo.AssertExpectations(t)
}

func TestOrganizationService_CreateOrganization_InvalidSlug(t *testing.T) {
	service, _, _ := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()

	tests := []struct {
		name string
		slug string
	}{
		{
			name: "slug with uppercase",
			slug: "Test-Org",
		},
		{
			name: "slug with spaces",
			slug: "test org",
		},
		{
			name: "slug starting with hyphen",
			slug: "-test-org",
		},
		{
			name: "slug ending with hyphen",
			slug: "test-org-",
		},
		{
			name: "slug with special characters",
			slug: "test@org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CreateOrganizationRequest{
				Name: "Test Organization",
				Slug: tt.slug,
			}

			org, err := service.CreateOrganization(ctx, userID, req)
			assert.Error(t, err)
			assert.Nil(t, org)
			assert.Contains(t, err.Error(), "invalid slug format")
		})
	}
}

func TestOrganizationService_CreateOrganization_SlugTaken(t *testing.T) {
	service, _, _ := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	req := &CreateOrganizationRequest{
		Name: "Test Organization",
		Slug: "test-org",
	}

	// Mock slug already taken
	service.slugExists = func(ctx context.Context, slug string) (bool, error) {
		return true, nil // Slug is taken
	}

	org, err := service.CreateOrganization(ctx, userID, req)
	assert.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "already taken")
}

func TestOrganizationService_GetOrganization(t *testing.T) {
	service, _, mockRBAC := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock permission check
	mockRBAC.On("CheckPermission", ctx, userID, PermissionOrgRead, "organization", &orgID).Return(true, nil)

	// Mock organization retrieval
	service.getOrganization = func(ctx context.Context, orgID uuid.UUID) (*types.Organization, error) {
		return &types.Organization{
			ID:        orgID,
			Name:      "Test Organization",
			Slug:      "test-org",
			Settings:  make(map[string]interface{}),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}

	org, err := service.GetOrganization(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Equal(t, orgID, org.ID)
	assert.Equal(t, "Test Organization", org.Name)
	assert.Equal(t, "test-org", org.Slug)

	mockRBAC.AssertExpectations(t)
}

func TestOrganizationService_GetOrganization_InsufficientPermissions(t *testing.T) {
	service, _, mockRBAC := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock permission check - no permission
	mockRBAC.On("CheckPermission", ctx, userID, PermissionOrgRead, "organization", &orgID).Return(false, nil)

	org, err := service.GetOrganization(ctx, userID, orgID)

	assert.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "insufficient permissions")

	mockRBAC.AssertExpectations(t)
}

func TestOrganizationService_UpdateOrganization(t *testing.T) {
	service, _, mockRBAC := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	req := &UpdateOrganizationRequest{
		Name: "Updated Organization",
	}

	// Mock permission check
	mockRBAC.On("CheckPermission", ctx, userID, PermissionOrgWrite, "organization", &orgID).Return(true, nil)

	// Mock organization retrieval
	service.getOrganization = func(ctx context.Context, orgID uuid.UUID) (*types.Organization, error) {
		return &types.Organization{
			ID:        orgID,
			Name:      "Old Name",
			Slug:      "test-org",
			Settings:  make(map[string]interface{}),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}

	// Mock organization update
	service.updateOrganization = func(ctx context.Context, org *types.Organization) error {
		return nil
	}

	org, err := service.UpdateOrganization(ctx, userID, orgID, req)

	require.NoError(t, err)
	assert.Equal(t, req.Name, org.Name)

	mockRBAC.AssertExpectations(t)
}

func TestOrganizationService_DeleteOrganization(t *testing.T) {
	service, _, mockRBAC := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// Mock permission check
	mockRBAC.On("CheckPermission", ctx, userID, PermissionOrgDelete, "organization", &orgID).Return(true, nil)

	// Mock organization deletion
	service.deleteOrganization = func(ctx context.Context, orgID uuid.UUID) error {
		return nil
	}

	err := service.DeleteOrganization(ctx, userID, orgID)

	require.NoError(t, err)

	mockRBAC.AssertExpectations(t)
}

func TestOrganizationService_ListUserOrganizations(t *testing.T) {
	service, _, _ := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()

	// Mock user organizations retrieval
	service.getUserOrganizations = func(ctx context.Context, userID uuid.UUID) ([]*types.Organization, error) {
		return []*types.Organization{
			{
				ID:        uuid.New(),
				Name:      "Org 1",
				Slug:      "org-1",
				Settings:  make(map[string]interface{}),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        uuid.New(),
				Name:      "Org 2",
				Slug:      "org-2",
				Settings:  make(map[string]interface{}),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}, nil
	}

	orgs, err := service.ListUserOrganizations(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, orgs, 2)
	assert.Equal(t, "Org 1", orgs[0].Name)
	assert.Equal(t, "Org 2", orgs[1].Name)
}

func TestOrganizationService_InviteMember(t *testing.T) {
	service, mockRepos, mockRBAC := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	invitedUserID := uuid.New()
	req := &InviteMemberRequest{
		Email: "invited@example.com",
		Role:  types.RoleAdmin,
	}

	// Mock permission check
	mockRBAC.On("CheckPermission", ctx, userID, PermissionUserInvite, "organization", &orgID).Return(true, nil)

	// Mock user lookup
	invitedUser := &types.User{
		ID:    invitedUserID,
		Email: req.Email,
		Name:  "Invited User",
	}
	mockRepos.Users.On("GetByEmail", ctx, req.Email).Return(invitedUser, nil)

	// Mock existing membership check
	service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
		return nil, nil // No existing membership
	}

	// Mock member addition
	service.addOrganizationMember = func(ctx context.Context, member *types.OrganizationMember) error {
		return nil
	}

	err := service.InviteMember(ctx, userID, orgID, req)

	require.NoError(t, err)

	mockRBAC.AssertExpectations(t)
	mockRepos.Users.AssertExpectations(t)
}

func TestOrganizationService_InviteMember_UserNotFound(t *testing.T) {
	service, mockRepos, mockRBAC := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	req := &InviteMemberRequest{
		Email: "nonexistent@example.com",
		Role:  types.RoleAdmin,
	}

	// Mock permission check
	mockRBAC.On("CheckPermission", ctx, userID, PermissionUserInvite, "organization", &orgID).Return(true, nil)

	// Mock user not found
	mockRepos.Users.On("GetByEmail", ctx, req.Email).Return(nil, assert.AnError)

	err := service.InviteMember(ctx, userID, orgID, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	mockRBAC.AssertExpectations(t)
	mockRepos.Users.AssertExpectations(t)
}

func TestOrganizationService_InviteMember_AlreadyMember(t *testing.T) {
	service, mockRepos, mockRBAC := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	invitedUserID := uuid.New()
	req := &InviteMemberRequest{
		Email: "invited@example.com",
		Role:  types.RoleAdmin,
	}

	// Mock permission check
	mockRBAC.On("CheckPermission", ctx, userID, PermissionUserInvite, "organization", &orgID).Return(true, nil)

	// Mock user lookup
	invitedUser := &types.User{
		ID:    invitedUserID,
		Email: req.Email,
		Name:  "Invited User",
	}
	mockRepos.Users.On("GetByEmail", ctx, req.Email).Return(invitedUser, nil)

	// Mock existing membership
	service.getOrganizationMember = func(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error) {
		return &types.OrganizationMember{
			ID:             uuid.New(),
			OrganizationID: orgID,
			UserID:         userID,
			Role:           types.RoleMember,
		}, nil
	}

	err := service.InviteMember(ctx, userID, orgID, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already a member")

	mockRBAC.AssertExpectations(t)
	mockRepos.Users.AssertExpectations(t)
}

func TestIsValidSlug(t *testing.T) {
	tests := []struct {
		name     string
		slug     string
		expected bool
	}{
		{
			name:     "valid simple slug",
			slug:     "test",
			expected: true,
		},
		{
			name:     "valid slug with hyphens",
			slug:     "test-org-name",
			expected: true,
		},
		{
			name:     "valid slug with numbers",
			slug:     "test123",
			expected: true,
		},
		{
			name:     "invalid slug with uppercase",
			slug:     "Test",
			expected: false,
		},
		{
			name:     "invalid slug starting with hyphen",
			slug:     "-test",
			expected: false,
		},
		{
			name:     "invalid slug ending with hyphen",
			slug:     "test-",
			expected: false,
		},
		{
			name:     "invalid slug with special characters",
			slug:     "test@org",
			expected: false,
		},
		{
			name:     "invalid empty slug",
			slug:     "",
			expected: false,
		},
		{
			name:     "invalid slug with spaces",
			slug:     "test org",
			expected: false,
		},
		{
			name:     "invalid slug with consecutive hyphens",
			slug:     "test--org",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSlug(tt.slug)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToOrganizationDTO(t *testing.T) {
	org := &types.Organization{
		ID:        uuid.New(),
		Name:      "Test Organization",
		Slug:      "test-org",
		Settings:  map[string]interface{}{"key": "value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	dto := toOrganizationDTO(org)

	assert.Equal(t, org.ID, dto.ID)
	assert.Equal(t, org.Name, dto.Name)
	assert.Equal(t, org.Slug, dto.Slug)
	assert.Equal(t, org.Settings, dto.Settings)
	assert.Equal(t, org.CreatedAt, dto.CreatedAt)
	assert.Equal(t, org.UpdatedAt, dto.UpdatedAt)
}

// Benchmark tests
func BenchmarkOrganizationService_CreateOrganization(b *testing.B) {
	service, _, _ := setupOrganizationService()
	ctx := context.Background()

	userID := uuid.New()
	req := &CreateOrganizationRequest{
		Name: "Test Organization",
		Slug: "test-org",
	}

	// Mock functions
	service.slugExists = func(ctx context.Context, slug string) (bool, error) {
		return false, nil
	}
	service.createOrganization = func(ctx context.Context, org *types.Organization) error {
		return nil
	}
	service.addOrganizationMember = func(ctx context.Context, member *types.OrganizationMember) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.CreateOrganization(ctx, userID, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIsValidSlug(b *testing.B) {
	slug := "test-org-name-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isValidSlug(slug)
	}
}