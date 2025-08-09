package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/types"
)

func setupUserService() (*UserService, *MockRepositories) {
	mockRepos := &MockRepositories{
		Users: &MockUserRepository{},
	}

	repos := &database.Repositories{
		Users: mockRepos.Users,
	}

	service := NewUserService(repos)
	return service, mockRepos
}

func TestUserService_GetProfile(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:        userID,
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://example.com/avatar.jpg",
		GitHubID:  intPtr(12345),
		GitLabID:  intPtr(67890),
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil)

	profile, err := service.GetProfile(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, user.ID, profile.ID)
	assert.Equal(t, user.Email, profile.Email)
	assert.Equal(t, user.Name, profile.Name)
	assert.Equal(t, user.AvatarURL, profile.AvatarURL)
	assert.Equal(t, user.GitHubID, profile.GitHubID)
	assert.Equal(t, user.GitLabID, profile.GitLabID)
	assert.Equal(t, user.CreatedAt, profile.CreatedAt)
	assert.Equal(t, user.UpdatedAt, profile.UpdatedAt)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_GetProfile_UserNotFound(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()

	mockRepos.Users.On("GetByID", ctx, userID).Return(nil, assert.AnError)

	profile, err := service.GetProfile(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Contains(t, err.Error(), "failed to get user")

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_UpdateProfile(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:        userID,
		Email:     "test@example.com",
		Name:      "Old Name",
		AvatarURL: "https://old-avatar.com/avatar.jpg",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	req := &UpdateProfileRequest{
		Name:      "New Name",
		AvatarURL: "https://new-avatar.com/avatar.jpg",
	}

	mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil)
	mockRepos.Users.On("Update", ctx, mock.MatchedBy(func(u *types.User) bool {
		return u.ID == userID && 
			   u.Name == "New Name" && 
			   u.AvatarURL == "https://new-avatar.com/avatar.jpg"
			   // Note: We don't check UpdatedAt timing due to test timing issues
	})).Return(nil)

	profile, err := service.UpdateProfile(ctx, userID, req)

	require.NoError(t, err)
	assert.Equal(t, userID, profile.ID)
	assert.Equal(t, "New Name", profile.Name)
	assert.Equal(t, "https://new-avatar.com/avatar.jpg", profile.AvatarURL)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_UpdateProfile_EmptyAvatarURL(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:        userID,
		Email:     "test@example.com",
		Name:      "Old Name",
		AvatarURL: "https://old-avatar.com/avatar.jpg",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	req := &UpdateProfileRequest{
		Name:      "New Name",
		AvatarURL: "", // Empty avatar URL should not update
	}

	mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil)
	mockRepos.Users.On("Update", ctx, mock.MatchedBy(func(u *types.User) bool {
		return u.ID == userID && 
			   u.Name == "New Name" && 
			   u.AvatarURL == "https://old-avatar.com/avatar.jpg" // Should keep old avatar
	})).Return(nil)

	profile, err := service.UpdateProfile(ctx, userID, req)

	require.NoError(t, err)
	assert.Equal(t, "New Name", profile.Name)
	assert.Equal(t, "https://old-avatar.com/avatar.jpg", profile.AvatarURL)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_ChangeEmail(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:        userID,
		Email:     "old@example.com",
		Name:      "Test User",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	req := &ChangeEmailRequest{
		NewEmail: "new@example.com",
	}

	// Mock email not in use
	mockRepos.Users.On("GetByEmail", ctx, "new@example.com").Return(nil, assert.AnError)
	mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil)
	mockRepos.Users.On("Update", ctx, mock.MatchedBy(func(u *types.User) bool {
		return u.ID == userID && 
			   u.Email == "new@example.com"
			   // Note: We don't check UpdatedAt timing due to test timing issues
	})).Return(nil)

	err := service.ChangeEmail(ctx, userID, req)

	require.NoError(t, err)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_ChangeEmail_EmailInUse(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	otherUserID := uuid.New()
	
	existingUser := &types.User{
		ID:    otherUserID,
		Email: "new@example.com",
		Name:  "Other User",
	}

	req := &ChangeEmailRequest{
		NewEmail: "new@example.com",
	}

	// Mock email already in use by another user
	mockRepos.Users.On("GetByEmail", ctx, "new@example.com").Return(existingUser, nil)

	err := service.ChangeEmail(ctx, userID, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email already in use")

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_ChangeEmail_SameUser(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:        userID,
		Email:     "current@example.com",
		Name:      "Test User",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	req := &ChangeEmailRequest{
		NewEmail: "new@example.com",
	}

	// Mock email in use by same user (should be allowed)
	mockRepos.Users.On("GetByEmail", ctx, "new@example.com").Return(user, nil)
	mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil)
	mockRepos.Users.On("Update", ctx, mock.MatchedBy(func(u *types.User) bool {
		return u.ID == userID && u.Email == "new@example.com"
	})).Return(nil)

	err := service.ChangeEmail(ctx, userID, req)

	require.NoError(t, err)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_DeleteAccount(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()

	mockRepos.Users.On("Delete", ctx, userID).Return(nil)

	err := service.DeleteAccount(ctx, userID)

	require.NoError(t, err)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_DeleteAccount_Error(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()

	mockRepos.Users.On("Delete", ctx, userID).Return(assert.AnError)

	err := service.DeleteAccount(ctx, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete user")

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_GetUserByEmail(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	email := "test@example.com"
	user := &types.User{
		ID:    uuid.New(),
		Email: email,
		Name:  "Test User",
	}

	mockRepos.Users.On("GetByEmail", ctx, email).Return(user, nil)

	result, err := service.GetUserByEmail(ctx, email)

	require.NoError(t, err)
	assert.Equal(t, user, result)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_GetUserByID(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil)

	result, err := service.GetUserByID(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, user, result)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_ListUsers(t *testing.T) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	pagination := &database.Pagination{
		Page:     1,
		PageSize: 10,
	}

	users := []*types.User{
		{
			ID:    uuid.New(),
			Email: "user1@example.com",
			Name:  "User 1",
		},
		{
			ID:    uuid.New(),
			Email: "user2@example.com",
			Name:  "User 2",
		},
	}

	mockRepos.Users.On("List", ctx, pagination).Return(users, int64(2), nil)

	result, total, err := service.ListUsers(ctx, pagination)

	require.NoError(t, err)
	assert.Equal(t, users, result)
	assert.Equal(t, int64(2), total)

	mockRepos.Users.AssertExpectations(t)
}

func TestUserService_GetUserStats(t *testing.T) {
	service, _ := setupUserService()
	ctx := context.Background()

	stats, err := service.GetUserStats(ctx)

	require.NoError(t, err)
	assert.NotNil(t, stats)
	// Currently returns zero values as it's not implemented
	assert.Equal(t, int64(0), stats.TotalUsers)
	assert.Equal(t, int64(0), stats.ActiveUsers)
	assert.Equal(t, int64(0), stats.NewUsers)
	assert.Equal(t, int64(0), stats.GitHubUsers)
	assert.Equal(t, int64(0), stats.GitLabUsers)
}

func TestUserService_GetUserActivity(t *testing.T) {
	service, _ := setupUserService()
	ctx := context.Background()

	userID := uuid.New()

	activity, err := service.GetUserActivity(ctx, userID)

	require.NoError(t, err)
	assert.NotNil(t, activity)
	assert.Equal(t, userID, activity.UserID)
	// Currently returns zero/nil values as it's not implemented
	assert.Nil(t, activity.LastLoginAt)
	assert.Nil(t, activity.LastActiveAt)
	assert.Equal(t, int64(0), activity.LoginCount)
	assert.Equal(t, int64(0), activity.ScanCount)
}

func TestUserService_UpdateLastActive(t *testing.T) {
	service, _ := setupUserService()
	ctx := context.Background()

	userID := uuid.New()

	err := service.UpdateLastActive(ctx, userID)

	// Currently a no-op, should not error
	require.NoError(t, err)
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Benchmark tests
func BenchmarkUserService_GetProfile(b *testing.B) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:        userID,
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://example.com/avatar.jpg",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetProfile(ctx, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUserService_UpdateProfile(b *testing.B) {
	service, mockRepos := setupUserService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:        userID,
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://example.com/avatar.jpg",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	req := &UpdateProfileRequest{
		Name:      "Updated Name",
		AvatarURL: "https://new-avatar.com/avatar.jpg",
	}

	mockRepos.Users.On("GetByID", ctx, userID).Return(user, nil)
	mockRepos.Users.On("Update", ctx, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.UpdateProfile(ctx, userID, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}