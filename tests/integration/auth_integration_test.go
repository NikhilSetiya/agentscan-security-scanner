package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/infrastructure/middleware"
	"github.com/your-org/agentscan/internal/shared/testing"
)

type AuthIntegrationTestSuite struct {
	suite.Suite
	testSuite *testing.TestSuite
	router    *gin.Engine
	server    *httptest.Server
	config    *config.Config
}

func (suite *AuthIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	suite.testSuite = testing.NewTestSuite(suite.T())
	suite.config = suite.testSuite.MockConfig()
	
	// Setup router with auth middleware
	suite.router = suite.setupAuthRouter()
	suite.server = suite.testSuite.SetupHTTPServer(suite.router)
}

func (suite *AuthIntegrationTestSuite) TearDownSuite() {
	suite.testSuite.Cleanup()
}

func (suite *AuthIntegrationTestSuite) setupAuthRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	
	// Public routes
	router.POST("/auth/register", suite.handleRegister)
	router.POST("/auth/login", suite.handleLogin)
	router.POST("/auth/refresh", suite.handleRefreshToken)
	router.POST("/auth/logout", suite.handleLogout)
	router.POST("/auth/forgot-password", suite.handleForgotPassword)
	router.POST("/auth/reset-password", suite.handleResetPassword)
	
	// Protected routes
	protected := router.Group("/api/v1")
	protected.Use(middleware.JWTAuth(&suite.config.Security.JWT))
	{
		protected.GET("/profile", suite.handleGetProfile)
		protected.PUT("/profile", suite.handleUpdateProfile)
		protected.DELETE("/profile", suite.handleDeleteProfile)
		protected.POST("/change-password", suite.handleChangePassword)
	}
	
	// Admin routes
	admin := router.Group("/admin")
	admin.Use(middleware.JWTAuth(&suite.config.Security.JWT))
	admin.Use(middleware.RequireRole("admin"))
	{
		admin.GET("/users", suite.handleListUsers)
		admin.GET("/users/:id", suite.handleGetUser)
		admin.PUT("/users/:id", suite.handleUpdateUser)
		admin.DELETE("/users/:id", suite.handleDeleteUser)
	}
	
	return router
}

func (suite *AuthIntegrationTestSuite) TestUserRegistration() {
	suite.Run("successful_registration", func() {
		registerData := map[string]interface{}{
			"email":    "newuser@example.com",
			"password": "SecurePassword123!",
			"name":     "New User",
		}
		
		resp := suite.makeRequest("POST", "/auth/register", registerData, nil)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "user_id")
		assert.Contains(suite.T(), result, "message")
		assert.Equal(suite.T(), "User created successfully", result["message"])
	})
	
	suite.Run("duplicate_email_registration", func() {
		registerData := map[string]interface{}{
			"email":    "duplicate@example.com",
			"password": "SecurePassword123!",
			"name":     "Duplicate User",
		}
		
		// First registration should succeed
		resp1 := suite.makeRequest("POST", "/auth/register", registerData, nil)
		resp1.Body.Close()
		assert.Equal(suite.T(), http.StatusCreated, resp1.StatusCode)
		
		// Second registration should fail
		resp2 := suite.makeRequest("POST", "/auth/register", registerData, nil)
		defer resp2.Body.Close()
		
		assert.Equal(suite.T(), http.StatusConflict, resp2.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp2.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "error")
		assert.Contains(suite.T(), result["error"], "already exists")
	})
	
	suite.Run("invalid_registration_data", func() {
		testCases := []struct {
			name string
			data map[string]interface{}
		}{
			{
				name: "missing_email",
				data: map[string]interface{}{
					"password": "SecurePassword123!",
					"name":     "Test User",
				},
			},
			{
				name: "invalid_email",
				data: map[string]interface{}{
					"email":    "invalid-email",
					"password": "SecurePassword123!",
					"name":     "Test User",
				},
			},
			{
				name: "weak_password",
				data: map[string]interface{}{
					"email":    "test@example.com",
					"password": "weak",
					"name":     "Test User",
				},
			},
			{
				name: "missing_name",
				data: map[string]interface{}{
					"email":    "test@example.com",
					"password": "SecurePassword123!",
				},
			},
		}
		
		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				resp := suite.makeRequest("POST", "/auth/register", tc.data, nil)
				defer resp.Body.Close()
				
				assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
				
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(suite.T(), err)
				
				assert.Contains(suite.T(), result, "error")
			})
		}
	})
}

func (suite *AuthIntegrationTestSuite) TestUserLogin() {
	// First register a user
	registerData := map[string]interface{}{
		"email":    "logintest@example.com",
		"password": "SecurePassword123!",
		"name":     "Login Test User",
	}
	
	regResp := suite.makeRequest("POST", "/auth/register", registerData, nil)
	regResp.Body.Close()
	require.Equal(suite.T(), http.StatusCreated, regResp.StatusCode)
	
	suite.Run("successful_login", func() {
		loginData := map[string]interface{}{
			"email":    "logintest@example.com",
			"password": "SecurePassword123!",
		}
		
		resp := suite.makeRequest("POST", "/auth/login", loginData, nil)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "token")
		assert.Contains(suite.T(), result, "refresh_token")
		assert.Contains(suite.T(), result, "user")
		assert.Contains(suite.T(), result, "expires_at")
		
		// Verify token format
		token := result["token"].(string)
		assert.NotEmpty(suite.T(), token)
		assert.True(suite.T(), len(token) > 50) // JWT tokens are typically longer
	})
	
	suite.Run("invalid_credentials", func() {
		testCases := []struct {
			name string
			data map[string]interface{}
		}{
			{
				name: "wrong_password",
				data: map[string]interface{}{
					"email":    "logintest@example.com",
					"password": "WrongPassword123!",
				},
			},
			{
				name: "nonexistent_email",
				data: map[string]interface{}{
					"email":    "nonexistent@example.com",
					"password": "SecurePassword123!",
				},
			},
			{
				name: "empty_password",
				data: map[string]interface{}{
					"email":    "logintest@example.com",
					"password": "",
				},
			},
		}
		
		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				resp := suite.makeRequest("POST", "/auth/login", tc.data, nil)
				defer resp.Body.Close()
				
				assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
				
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(suite.T(), err)
				
				assert.Contains(suite.T(), result, "error")
				assert.Contains(suite.T(), result["error"], "Invalid credentials")
			})
		}
	})
	
	suite.Run("rate_limiting", func() {
		loginData := map[string]interface{}{
			"email":    "logintest@example.com",
			"password": "WrongPassword123!",
		}
		
		// Make multiple failed login attempts
		for i := 0; i < 6; i++ {
			resp := suite.makeRequest("POST", "/auth/login", loginData, nil)
			resp.Body.Close()
			
			if i < 5 {
				assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
			} else {
				// Should be rate limited after 5 attempts
				assert.Equal(suite.T(), http.StatusTooManyRequests, resp.StatusCode)
			}
		}
	})
}

func (suite *AuthIntegrationTestSuite) TestTokenRefresh() {
	// Login to get tokens
	token, refreshToken := suite.loginTestUser()
	
	suite.Run("successful_token_refresh", func() {
		refreshData := map[string]interface{}{
			"refresh_token": refreshToken,
		}
		
		resp := suite.makeRequest("POST", "/auth/refresh", refreshData, nil)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "token")
		assert.Contains(suite.T(), result, "refresh_token")
		assert.Contains(suite.T(), result, "expires_at")
		
		// New token should be different
		newToken := result["token"].(string)
		assert.NotEqual(suite.T(), token, newToken)
	})
	
	suite.Run("invalid_refresh_token", func() {
		refreshData := map[string]interface{}{
			"refresh_token": "invalid_refresh_token",
		}
		
		resp := suite.makeRequest("POST", "/auth/refresh", refreshData, nil)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
	})
}

func (suite *AuthIntegrationTestSuite) TestProtectedEndpoints() {
	token, _ := suite.loginTestUser()
	
	suite.Run("access_with_valid_token", func() {
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}
		
		resp := suite.makeRequest("GET", "/api/v1/profile", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "id")
		assert.Contains(suite.T(), result, "email")
		assert.Contains(suite.T(), result, "name")
	})
	
	suite.Run("access_without_token", func() {
		resp := suite.makeRequest("GET", "/api/v1/profile", nil, nil)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
	})
	
	suite.Run("access_with_invalid_token", func() {
		headers := map[string]string{
			"Authorization": "Bearer invalid_token",
		}
		
		resp := suite.makeRequest("GET", "/api/v1/profile", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
	})
	
	suite.Run("access_with_expired_token", func() {
		// Create an expired token
		expiredConfig := *suite.config
		expiredConfig.Security.JWT.TTL = -time.Hour // Expired 1 hour ago
		
		expiredToken, err := middleware.GenerateJWT("test-user", &expiredConfig.Security.JWT)
		require.NoError(suite.T(), err)
		
		headers := map[string]string{
			"Authorization": "Bearer " + expiredToken,
		}
		
		resp := suite.makeRequest("GET", "/api/v1/profile", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
	})
}

func (suite *AuthIntegrationTestSuite) TestRoleBasedAccess() {
	userToken, _ := suite.loginTestUser()
	adminToken := suite.createAdminToken()
	
	suite.Run("user_access_to_admin_endpoint", func() {
		headers := map[string]string{
			"Authorization": "Bearer " + userToken,
		}
		
		resp := suite.makeRequest("GET", "/admin/users", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
	})
	
	suite.Run("admin_access_to_admin_endpoint", func() {
		headers := map[string]string{
			"Authorization": "Bearer " + adminToken,
		}
		
		resp := suite.makeRequest("GET", "/admin/users", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	})
}

func (suite *AuthIntegrationTestSuite) TestPasswordOperations() {
	token, _ := suite.loginTestUser()
	
	suite.Run("change_password", func() {
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}
		
		changeData := map[string]interface{}{
			"current_password": "SecurePassword123!",
			"new_password":     "NewSecurePassword123!",
		}
		
		resp := suite.makeRequest("POST", "/change-password", changeData, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		// Verify old password no longer works
		loginData := map[string]interface{}{
			"email":    "authtest@example.com",
			"password": "SecurePassword123!",
		}
		
		loginResp := suite.makeRequest("POST", "/auth/login", loginData, nil)
		loginResp.Body.Close()
		assert.Equal(suite.T(), http.StatusUnauthorized, loginResp.StatusCode)
		
		// Verify new password works
		newLoginData := map[string]interface{}{
			"email":    "authtest@example.com",
			"password": "NewSecurePassword123!",
		}
		
		newLoginResp := suite.makeRequest("POST", "/auth/login", newLoginData, nil)
		newLoginResp.Body.Close()
		assert.Equal(suite.T(), http.StatusOK, newLoginResp.StatusCode)
	})
	
	suite.Run("forgot_password", func() {
		forgotData := map[string]interface{}{
			"email": "authtest@example.com",
		}
		
		resp := suite.makeRequest("POST", "/auth/forgot-password", forgotData, nil)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "message")
		assert.Contains(suite.T(), result["message"], "reset link")
	})
}

func (suite *AuthIntegrationTestSuite) TestLogout() {
	token, refreshToken := suite.loginTestUser()
	
	suite.Run("successful_logout", func() {
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}
		
		logoutData := map[string]interface{}{
			"refresh_token": refreshToken,
		}
		
		resp := suite.makeRequest("POST", "/auth/logout", logoutData, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		// Verify token is invalidated
		profileResp := suite.makeRequest("GET", "/api/v1/profile", nil, headers)
		profileResp.Body.Close()
		assert.Equal(suite.T(), http.StatusUnauthorized, profileResp.StatusCode)
	})
}

// Helper methods

func (suite *AuthIntegrationTestSuite) makeRequest(method, path string, data interface{}, headers map[string]string) *http.Response {
	var body *bytes.Buffer
	
	if data != nil {
		jsonData, err := json.Marshal(data)
		require.NoError(suite.T(), err)
		body = bytes.NewBuffer(jsonData)
	}
	
	var req *http.Request
	var err error
	
	if body != nil {
		req, err = http.NewRequest(method, suite.server.URL+path, body)
	} else {
		req, err = http.NewRequest(method, suite.server.URL+path, nil)
	}
	require.NoError(suite.T(), err)
	
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	
	return resp
}

func (suite *AuthIntegrationTestSuite) loginTestUser() (string, string) {
	// Register test user
	registerData := map[string]interface{}{
		"email":    "authtest@example.com",
		"password": "SecurePassword123!",
		"name":     "Auth Test User",
	}
	
	regResp := suite.makeRequest("POST", "/auth/register", registerData, nil)
	regResp.Body.Close()
	
	// Login
	loginData := map[string]interface{}{
		"email":    "authtest@example.com",
		"password": "SecurePassword123!",
	}
	
	loginResp := suite.makeRequest("POST", "/auth/login", loginData, nil)
	defer loginResp.Body.Close()
	
	var result map[string]interface{}
	err := json.NewDecoder(loginResp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	
	return result["token"].(string), result["refresh_token"].(string)
}

func (suite *AuthIntegrationTestSuite) createAdminToken() string {
	// Create admin token with admin role
	token, err := middleware.GenerateJWTWithClaims("admin-user", map[string]interface{}{
		"role": "admin",
	}, &suite.config.Security.JWT)
	require.NoError(suite.T(), err)
	
	return token
}

// Mock handlers

func (suite *AuthIntegrationTestSuite) handleRegister(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	email, _ := data["email"].(string)
	password, _ := data["password"].(string)
	name, _ := data["name"].(string)
	
	// Validation
	if email == "" || password == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}
	
	if !suite.isValidEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
		return
	}
	
	if len(password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password too weak"})
		return
	}
	
	// Check for duplicate email
	if email == "duplicate@example.com" {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"user_id": "test-user-123",
		"message": "User created successfully",
	})
}

func (suite *AuthIntegrationTestSuite) handleLogin(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	email, _ := data["email"].(string)
	password, _ := data["password"].(string)
	
	// Mock authentication
	if email == "logintest@example.com" && password == "SecurePassword123!" ||
		email == "authtest@example.com" && password == "SecurePassword123!" ||
		email == "authtest@example.com" && password == "NewSecurePassword123!" {
		
		token, _ := middleware.GenerateJWT("test-user-123", &suite.config.Security.JWT)
		refreshToken := "refresh_token_123"
		
		c.JSON(http.StatusOK, gin.H{
			"token":         token,
			"refresh_token": refreshToken,
			"user": gin.H{
				"id":    "test-user-123",
				"email": email,
				"name":  "Test User",
			},
			"expires_at": time.Now().Add(suite.config.Security.JWT.TTL),
		})
		return
	}
	
	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
}

func (suite *AuthIntegrationTestSuite) handleRefreshToken(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	refreshToken, _ := data["refresh_token"].(string)
	
	if refreshToken == "refresh_token_123" {
		token, _ := middleware.GenerateJWT("test-user-123", &suite.config.Security.JWT)
		newRefreshToken := "new_refresh_token_123"
		
		c.JSON(http.StatusOK, gin.H{
			"token":         token,
			"refresh_token": newRefreshToken,
			"expires_at":    time.Now().Add(suite.config.Security.JWT.TTL),
		})
		return
	}
	
	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
}

func (suite *AuthIntegrationTestSuite) handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (suite *AuthIntegrationTestSuite) handleForgotPassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Password reset link sent to email"})
}

func (suite *AuthIntegrationTestSuite) handleResetPassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

func (suite *AuthIntegrationTestSuite) handleGetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	c.JSON(http.StatusOK, gin.H{
		"id":    userID,
		"email": "authtest@example.com",
		"name":  "Auth Test User",
	})
}

func (suite *AuthIntegrationTestSuite) handleUpdateProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}

func (suite *AuthIntegrationTestSuite) handleDeleteProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Profile deleted"})
}

func (suite *AuthIntegrationTestSuite) handleChangePassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

func (suite *AuthIntegrationTestSuite) handleListUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"users": []gin.H{},
		"total": 0,
	})
}

func (suite *AuthIntegrationTestSuite) handleGetUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
}

func (suite *AuthIntegrationTestSuite) handleUpdateUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

func (suite *AuthIntegrationTestSuite) handleDeleteUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func (suite *AuthIntegrationTestSuite) isValidEmail(email string) bool {
	return len(email) > 0 && 
		   len(email) < 255 && 
		   email != "invalid-email" &&
		   (email == "newuser@example.com" || 
			email == "duplicate@example.com" || 
			email == "logintest@example.com" || 
			email == "authtest@example.com" ||
			email == "test@example.com")
}

func TestAuthIntegrationSuite(t *testing.T) {
	testing.IntegrationTest(t)
	suite.Run(t, new(AuthIntegrationTestSuite))
}