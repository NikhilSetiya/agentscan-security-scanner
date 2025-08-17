package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

func setupMiddleware() (*Middleware, *Service, *UserService, *MockRepositories) {
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

	authService := NewService(cfg, repos)
	userService := NewUserService(repos)
	rbacService := NewRBACService(repos)
	middleware := NewMiddleware(authService, userService, rbacService, repos, cfg)

	return middleware, authService, userService, mockRepos
}

func TestMiddleware_AuthRequired_ValidToken(t *testing.T) {
	middleware, authService, _, mockRepos := setupMiddleware()

	// Create a test user and token
	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := context.Background()
	tokenPair, err := authService.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Mock user lookup
	mockRepos.Users.On("GetByID", mock.Anything, user.ID).Return(user, nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		currentUser, exists := GetCurrentUser(c)
		assert.True(t, exists)
		assert.Equal(t, user.ID, currentUser.ID)
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make request with valid token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestMiddleware_AuthRequired_NoToken(t *testing.T) {
	middleware, _, _, _ := setupMiddleware()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make request without token
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_AuthRequired_InvalidToken(t *testing.T) {
	middleware, _, _, _ := setupMiddleware()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make request with invalid token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_AuthRequired_ExpiredToken(t *testing.T) {
	// Create middleware with very short token expiration
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-key-for-testing-only",
			JWTExpiration: time.Millisecond, // Very short expiration
		},
	}

	mockRepos := &MockRepositories{
		Users: &MockUserRepository{},
	}

	repos := &database.Repositories{
		Users: mockRepos.Users,
	}

	authService := NewService(cfg, repos)
	userService := NewUserService(repos)
	middleware := NewMiddleware(authService, userService, repos, cfg)

	// Create a test user and token
	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := context.Background()
	tokenPair, err := authService.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make request with expired token
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_AuthRequired_UserNotFound(t *testing.T) {
	middleware, authService, _, mockRepos := setupMiddleware()

	// Create a test user and token
	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := context.Background()
	tokenPair, err := authService.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Mock user not found
	mockRepos.Users.On("GetByID", mock.Anything, user.ID).Return(nil, assert.AnError)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthRequired())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make request with valid token but user doesn't exist
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestMiddleware_OptionalAuth_WithValidToken(t *testing.T) {
	middleware, authService, _, mockRepos := setupMiddleware()

	// Create a test user and token
	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := context.Background()
	tokenPair, err := authService.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Mock user lookup
	mockRepos.Users.On("GetByID", mock.Anything, user.ID).Return(user, nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.OptionalAuth())
	router.GET("/optional", func(c *gin.Context) {
		currentUser, exists := GetCurrentUser(c)
		if exists {
			c.JSON(200, gin.H{"user": currentUser.Email})
		} else {
			c.JSON(200, gin.H{"user": "anonymous"})
		}
	})

	// Make request with valid token
	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), user.Email)
	mockRepos.Users.AssertExpectations(t)
}

func TestMiddleware_OptionalAuth_WithoutToken(t *testing.T) {
	middleware, _, _, _ := setupMiddleware()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.OptionalAuth())
	router.GET("/optional", func(c *gin.Context) {
		currentUser, exists := GetCurrentUser(c)
		if exists {
			c.JSON(200, gin.H{"user": currentUser.Email})
		} else {
			c.JSON(200, gin.H{"user": "anonymous"})
		}
	})

	// Make request without token
	req := httptest.NewRequest("GET", "/optional", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "anonymous")
}

func TestMiddleware_OptionalAuth_WithInvalidToken(t *testing.T) {
	middleware, _, _, _ := setupMiddleware()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.OptionalAuth())
	router.GET("/optional", func(c *gin.Context) {
		currentUser, exists := GetCurrentUser(c)
		if exists {
			c.JSON(200, gin.H{"user": currentUser.Email})
		} else {
			c.JSON(200, gin.H{"user": "anonymous"})
		}
	})

	// Make request with invalid token
	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "anonymous")
}

func TestMiddleware_RequireRole_Admin(t *testing.T) {
	middleware, authService, _, mockRepos := setupMiddleware()

	// Create a test user and token
	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := context.Background()
	tokenPair, err := authService.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Mock user lookup
	mockRepos.Users.On("GetByID", mock.Anything, user.ID).Return(user, nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthRequired())
	router.Use(middleware.RequireRole("admin"))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "admin access"})
	})

	// Make request with valid token but non-admin user
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should be forbidden since user is not admin
	assert.Equal(t, http.StatusForbidden, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestMiddleware_RequireRole_NoAuth(t *testing.T) {
	middleware, _, _, _ := setupMiddleware()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequireRole("admin"))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "admin access"})
	})

	// Make request without authentication
	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_RequireOrganizationAccess_InvalidOrgID(t *testing.T) {
	middleware, authService, _, mockRepos := setupMiddleware()

	// Create a test user and token
	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := context.Background()
	tokenPair, err := authService.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Mock user lookup
	mockRepos.Users.On("GetByID", mock.Anything, user.ID).Return(user, nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthRequired())
	router.Use(middleware.RequireOrganizationAccess("orgId"))
	router.GET("/org/:orgId/data", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "org data"})
	})

	// Make request with invalid org ID
	req := httptest.NewRequest("GET", "/org/invalid-uuid/data", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestMiddleware_RequireRepositoryAccess_InvalidRepoID(t *testing.T) {
	middleware, authService, _, mockRepos := setupMiddleware()

	// Create a test user and token
	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx := context.Background()
	tokenPair, err := authService.GenerateTokenPair(ctx, user, "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Mock user lookup
	mockRepos.Users.On("GetByID", mock.Anything, user.ID).Return(user, nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthRequired())
	router.Use(middleware.RequireRepositoryAccess("repoId"))
	router.GET("/repo/:repoId/data", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "repo data"})
	})

	// Make request with invalid repo ID
	req := httptest.NewRequest("GET", "/repo/invalid-uuid/data", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepos.Users.AssertExpectations(t)
}

func TestMiddleware_RateLimit(t *testing.T) {
	middleware, _, _, _ := setupMiddleware()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RateLimit(60)) // 60 requests per minute
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make request (should pass through since rate limiting is not implemented)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetCurrentUser(t *testing.T) {
	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Setup Gin context
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user", user)

	// Test getting current user
	currentUser, exists := GetCurrentUser(c)
	assert.True(t, exists)
	assert.Equal(t, user, currentUser)

	// Test with no user in context
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	currentUser2, exists2 := GetCurrentUser(c2)
	assert.False(t, exists2)
	assert.Nil(t, currentUser2)
}

func TestGetCurrentUserID(t *testing.T) {
	userID := uuid.New()

	// Setup Gin context
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", userID)

	// Test getting current user ID
	currentUserID, exists := GetCurrentUserID(c)
	assert.True(t, exists)
	assert.Equal(t, userID, currentUserID)

	// Test with no user ID in context
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	currentUserID2, exists2 := GetCurrentUserID(c2)
	assert.False(t, exists2)
	assert.Equal(t, uuid.Nil, currentUserID2)
}

func TestGetOrganizationID(t *testing.T) {
	orgID := uuid.New()

	// Setup Gin context
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("organization_id", orgID)

	// Test getting organization ID
	currentOrgID, exists := GetOrganizationID(c)
	assert.True(t, exists)
	assert.Equal(t, orgID, currentOrgID)

	// Test with no organization ID in context
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	currentOrgID2, exists2 := GetOrganizationID(c2)
	assert.False(t, exists2)
	assert.Equal(t, uuid.Nil, currentOrgID2)
}

func TestGetRepositoryID(t *testing.T) {
	repoID := uuid.New()

	// Setup Gin context
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("repository_id", repoID)

	// Test getting repository ID
	currentRepoID, exists := GetRepositoryID(c)
	assert.True(t, exists)
	assert.Equal(t, repoID, currentRepoID)

	// Test with no repository ID in context
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	currentRepoID2, exists2 := GetRepositoryID(c2)
	assert.False(t, exists2)
	assert.Equal(t, uuid.Nil, currentRepoID2)
}