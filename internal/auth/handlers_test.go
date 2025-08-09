package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/internal/api"
	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/types"
)



func setupHandlers() (*Handlers, *MockRepositories) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:      "test-secret-key-for-testing-only",
			JWTExpiration:  time.Hour,
			GitHubClientID: "test-github-client-id",
			GitHubSecret:   "test-github-secret",
			GitLabClientID: "test-gitlab-client-id",
			GitLabSecret:   "test-gitlab-secret",
		},
	}

	mockRepos := &MockRepositories{
		Users: &MockUserRepository{},
	}

	repos := &database.Repositories{
		Users: mockRepos.Users,
	}

	// Create a real audit logger for testing
	auditLogger := api.NewAuditLogger(repos)

	handlers := NewHandlers(cfg, repos, auditLogger)
	return handlers, mockRepos
}

func TestHandlers_GetGitHubAuthURL(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/auth/github/url", handlers.GetGitHubAuthURL)

	req := httptest.NewRequest("GET", "/auth/github/url", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["auth_url"].(string), "github.com")
	assert.NotEmpty(t, data["state"].(string))
}

func TestHandlers_GetGitLabAuthURL(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/auth/gitlab/url", handlers.GetGitLabAuthURL)

	req := httptest.NewRequest("GET", "/auth/gitlab/url", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["auth_url"].(string), "gitlab.com")
	assert.NotEmpty(t, data["state"].(string))
}

func TestHandlers_HandleGitHubCallback_MissingCode(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/github/callback", handlers.HandleGitHubCallback)

	req := httptest.NewRequest("POST", "/auth/github/callback", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "bad_request", errorData["code"])
}

func TestHandlers_HandleGitLabCallback_MissingCode(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/gitlab/callback", handlers.HandleGitLabCallback)

	req := httptest.NewRequest("POST", "/auth/gitlab/callback", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "bad_request", errorData["code"])
}

func TestHandlers_RefreshToken_InvalidRequest(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/refresh", handlers.RefreshToken)

	// Send request with invalid JSON
	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlers_RefreshToken_MissingRefreshToken(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/refresh", handlers.RefreshToken)

	// Send request without refresh token
	reqBody := map[string]string{}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlers_GetCurrentUser_NoUser(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/user/me", handlers.GetCurrentUser)

	req := httptest.NewRequest("GET", "/user/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlers_GetCurrentUser_WithUser(t *testing.T) {
	handlers, mockRepos := setupHandlers()

	user := &types.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://example.com/avatar.jpg",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Mock user profile retrieval
	mockRepos.Users.On("GetByID", mock.Anything, user.ID).Return(user, nil)

	// Setup Gin with user in context
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	})
	router.GET("/user/me", handlers.GetCurrentUser)

	req := httptest.NewRequest("GET", "/user/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, user.Email, data["email"])
	assert.Equal(t, user.Name, data["name"])

	mockRepos.Users.AssertExpectations(t)
}

func TestHandlers_UpdateProfile_NoUser(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/user/profile", handlers.UpdateProfile)

	reqBody := UpdateProfileRequest{
		Name:      "New Name",
		AvatarURL: "https://new-avatar.com/avatar.jpg",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/user/profile", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlers_UpdateProfile_InvalidRequest(t *testing.T) {
	handlers, _ := setupHandlers()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Setup Gin with user in context
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	})
	router.PUT("/user/profile", handlers.UpdateProfile)

	// Send request with invalid data (empty name)
	reqBody := UpdateProfileRequest{
		Name:      "", // Invalid: empty name
		AvatarURL: "https://new-avatar.com/avatar.jpg",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/user/profile", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlers_ChangeEmail_NoUser(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/user/email", handlers.ChangeEmail)

	reqBody := ChangeEmailRequest{
		NewEmail: "new@example.com",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/user/email", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlers_ChangeEmail_InvalidEmail(t *testing.T) {
	handlers, _ := setupHandlers()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Setup Gin with user in context
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	})
	router.PUT("/user/email", handlers.ChangeEmail)

	// Send request with invalid email
	reqBody := ChangeEmailRequest{
		NewEmail: "invalid-email", // Invalid email format
	}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/user/email", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlers_Logout_WithUser(t *testing.T) {
	handlers, _ := setupHandlers()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Setup Gin with user in context
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	})
	router.POST("/auth/logout", handlers.Logout)

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Logged out successfully", data["message"])
}

func TestHandlers_Logout_WithoutUser(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin without user in context
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/logout", handlers.Logout)

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Logged out successfully", data["message"])
}

func TestHandlers_RevokeToken_InvalidRequest(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/revoke", handlers.RevokeToken)

	// Send request without refresh token
	reqBody := map[string]string{}
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/revoke", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlers_RevokeAllTokens_NoUser(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/revoke-all", handlers.RevokeAllTokens)

	req := httptest.NewRequest("POST", "/auth/revoke-all", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlers_GetSessions_NoUser(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/user/sessions", handlers.GetSessions)

	req := httptest.NewRequest("GET", "/user/sessions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlers_GetSessions_WithUser(t *testing.T) {
	handlers, _ := setupHandlers()

	user := &types.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Setup Gin with user in context
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	})
	router.GET("/user/sessions", handlers.GetSessions)

	req := httptest.NewRequest("GET", "/user/sessions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	// Should return empty array since sessions are not implemented
	data := response["data"].([]interface{})
	assert.Empty(t, data)
}

func TestHandlers_DeleteAccount_NoUser(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/user/account", handlers.DeleteAccount)

	req := httptest.NewRequest("DELETE", "/user/account", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandlers_HealthCheck(t *testing.T) {
	handlers, _ := setupHandlers()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/auth/health", handlers.HealthCheck)

	req := httptest.NewRequest("GET", "/auth/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "healthy", data["status"])
	assert.NotNil(t, data["timestamp"])
	
	services := data["services"].(map[string]interface{})
	assert.Equal(t, "ok", services["auth"])
	assert.Equal(t, "ok", services["oauth"])
}

// Test helper functions
func TestGenerateState(t *testing.T) {
	handlers, _ := setupHandlers()

	state1, err1 := handlers.generateState()
	require.NoError(t, err1)
	assert.NotEmpty(t, state1)

	state2, err2 := handlers.generateState()
	require.NoError(t, err2)
	assert.NotEmpty(t, state2)

	// States should be different
	assert.NotEqual(t, state1, state2)

	// States should be valid base64
	_, err := base64.URLEncoding.DecodeString(state1)
	assert.NoError(t, err)
}