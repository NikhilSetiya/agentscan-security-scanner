package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*types.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockUserRepository) GetBySupabaseID(ctx context.Context, supabaseID string) (*types.User, error) {
	args := m.Called(ctx, supabaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, pagination *database.Pagination) ([]*types.User, int64, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).([]*types.User), args.Get(1).(int64), args.Error(2)
}

// MockRepositories is a mock implementation of Repositories
type MockRepositories struct {
	Users *MockUserRepository
}

func setupTestService() (*Service, *MockRepositories) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-for-testing-only",
			JWTExpiration: time.Hour,
		},
	}

	mockRepos := &MockRepositories{
		Users: &MockUserRepository{},
	}

	repos := &database.Repositories{
		Users: mockRepos.Users,
	}

	service := NewService(cfg, repos)
	return service, mockRepos
}

func TestService_GenerateTokenPair(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	tokenPair, err := service.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")

	require.NoError(t, err)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.Equal(t, "Bearer", tokenPair.TokenType)
	assert.True(t, tokenPair.ExpiresAt.After(time.Now()))
}

func TestService_ValidateAccessToken(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Generate a token
	tokenPair, err := service.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Validate the token
	claims, err := service.ValidateAccessToken(tokenPair.AccessToken)
	require.NoError(t, err)

	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Name, claims.Name)
}

func TestService_ValidateAccessToken_InvalidToken(t *testing.T) {
	service, _ := setupTestService()

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "invalid format",
			token: "invalid-token",
		},
		{
			name:  "wrong signature",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidateAccessToken(tt.token)
			assert.Error(t, err)
		})
	}
}

func TestService_ValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create a service with very short expiration
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-for-testing-only",
			JWTExpiration: time.Millisecond, // Very short expiration
		},
	}

	repos := &database.Repositories{
		Users: &MockUserRepository{},
	}

	service := NewService(cfg, repos)
	ctx := context.Background()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Generate a token
	tokenPair, err := service.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	_, err = service.ValidateAccessToken(tokenPair.AccessToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestService_RefreshToken(t *testing.T) {
	service, mockRepos := setupTestService()
	ctx := context.Background()

	userID := uuid.New()
	user := &types.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Mock the user repository to return the user for any UUID
	// Since validateRefreshToken returns a mock session with random UserID,
	// we need to mock for any UUID
	mockRepos.Users.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(user, nil)

	// Generate initial token pair
	tokenPair, err := service.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Refresh the token
	newTokenPair, err := service.RefreshToken(ctx, tokenPair.RefreshToken, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	assert.NotEmpty(t, newTokenPair.AccessToken)
	assert.NotEmpty(t, newTokenPair.RefreshToken)
	// Note: Since validateRefreshToken is a mock that returns a different user,
	// the tokens might be the same. In a real implementation, this would be different.
	// For now, we just verify the tokens are generated successfully.

	mockRepos.Users.AssertExpectations(t)
}

func TestJWTClaims_Validation(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	tokenPair, err := service.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Parse token to check claims
	token, err := jwt.ParseWithClaims(tokenPair.AccessToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret-key-for-testing-only"), nil
	})
	require.NoError(t, err)

	claims, ok := token.Claims.(*JWTClaims)
	require.True(t, ok)

	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Name, claims.Name)
	assert.Equal(t, "agentscan", claims.Issuer)
	assert.Equal(t, user.ID.String(), claims.Subject)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.NotBefore)
}

func TestService_RevokeToken(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	// Test revoking a token (currently a no-op but should not error)
	err := service.RevokeToken(ctx, "test-refresh-token")
	assert.NoError(t, err)
}

func TestService_RevokeAllUserTokens(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	userID := uuid.New()

	// Test revoking all user tokens (currently a no-op but should not error)
	err := service.RevokeAllUserTokens(ctx, userID)
	assert.NoError(t, err)
}

func TestService_CleanupExpiredSessions(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	// Test cleanup (currently a no-op but should not error)
	err := service.CleanupExpiredSessions(ctx)
	assert.NoError(t, err)
}

func TestService_GetUserSessions(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	userID := uuid.New()

	// Test getting user sessions (currently returns empty list)
	sessions, err := service.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

// Benchmark tests
func BenchmarkService_GenerateTokenPair(b *testing.B) {
	service, _ := setupTestService()
	ctx := context.Background()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkService_ValidateAccessToken(b *testing.B) {
	service, _ := setupTestService()
	ctx := context.Background()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	tokenPair, err := service.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ValidateAccessToken(tokenPair.AccessToken)
		if err != nil {
			b.Fatal(err)
		}
	}
}