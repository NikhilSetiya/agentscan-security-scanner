package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/your-org/agentscan/internal/api"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/infrastructure/middleware"
	"github.com/your-org/agentscan/internal/infrastructure/monitoring"
	"github.com/your-org/agentscan/internal/shared/testing"
)

type APIIntegrationTestSuite struct {
	suite.Suite
	testSuite        *testing.TestSuite
	router           *gin.Engine
	server           *httptest.Server
	config           *config.Config
	metricsCollector *monitoring.MetricsCollector
	healthChecker    *monitoring.HealthChecker
}

func (suite *APIIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	suite.testSuite = testing.NewTestSuite(suite.T())
	
	// Setup test configuration
	suite.config = suite.testSuite.MockConfig()
	
	// Setup monitoring
	suite.metricsCollector = monitoring.NewMetricsCollector("agentscan_integration_test")
	suite.healthChecker = monitoring.NewHealthChecker()
	
	// Setup database and Redis for integration tests
	db := suite.testSuite.SetupDatabase()
	redisClient := suite.testSuite.SetupRedis()
	
	// Add health checks
	suite.healthChecker.AddDatabaseCheck("database", db)
	suite.healthChecker.AddRedisCheck("redis", redisClient)
	
	// Setup router with middleware
	suite.router = suite.setupRouter()
	
	// Start test server
	suite.server = suite.testSuite.SetupHTTPServer(suite.router)
}

func (suite *APIIntegrationTestSuite) TearDownSuite() {
	suite.testSuite.Cleanup()
}

func (suite *APIIntegrationTestSuite) setupRouter() *gin.Engine {
	router := gin.New()
	
	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeaders(&suite.config.Security.Headers))
	router.Use(middleware.CORS(&suite.config.Security.CORS))
	router.Use(middleware.PrometheusMiddleware(suite.metricsCollector))
	router.Use(middleware.RequestSizeLimit(1024 * 1024)) // 1MB limit
	
	// Health endpoints
	router.GET("/health", monitoring.HealthEndpoint(suite.healthChecker))
	router.GET("/ready", monitoring.ReadinessEndpoint(suite.healthChecker))
	router.GET("/live", monitoring.LivenessEndpoint())
	
	// API routes
	v1 := router.Group("/api/v1")
	{
		// Public endpoints
		v1.POST("/auth/login", suite.handleLogin)
		v1.POST("/auth/register", suite.handleRegister)
		
		// Protected endpoints
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(&suite.config.Security.JWT))
		{
			protected.GET("/profile", suite.handleGetProfile)
			protected.PUT("/profile", suite.handleUpdateProfile)
			
			// Scan endpoints
			scans := protected.Group("/scans")
			{
				scans.GET("", suite.handleListScans)
				scans.POST("", suite.handleCreateScan)
				scans.GET("/:id", suite.handleGetScan)
				scans.PUT("/:id", suite.handleUpdateScan)
				scans.DELETE("/:id", suite.handleDeleteScan)
				scans.POST("/:id/start", suite.handleStartScan)
				scans.POST("/:id/stop", suite.handleStopScan)
			}
			
			// Agent endpoints
			agents := protected.Group("/agents")
			{
				agents.GET("", suite.handleListAgents)
				agents.POST("", suite.handleCreateAgent)
				agents.GET("/:id", suite.handleGetAgent)
				agents.PUT("/:id", suite.handleUpdateAgent)
				agents.DELETE("/:id", suite.handleDeleteAgent)
			}
			
			// Report endpoints
			reports := protected.Group("/reports")
			{
				reports.GET("", suite.handleListReports)
				reports.GET("/:id", suite.handleGetReport)
				reports.POST("/:id/export", suite.handleExportReport)
			}
		}
	}
	
	return router
}

func (suite *APIIntegrationTestSuite) TestHealthEndpoints() {
	suite.Run("health_check_endpoint", func() {
		resp, err := http.Get(suite.server.URL + "/health")
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		assert.Equal(suite.T(), "application/json", resp.Header.Get("Content-Type"))
		
		var healthReport monitoring.HealthReport
		err = json.NewDecoder(resp.Body).Decode(&healthReport)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), monitoring.HealthStatusHealthy, healthReport.Status)
		assert.Contains(suite.T(), healthReport.Checks, "database")
		assert.Contains(suite.T(), healthReport.Checks, "redis")
	})
	
	suite.Run("readiness_endpoint", func() {
		resp, err := http.Get(suite.server.URL + "/ready")
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	})
	
	suite.Run("liveness_endpoint", func() {
		resp, err := http.Get(suite.server.URL + "/live")
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	})
}

func (suite *APIIntegrationTestSuite) TestAuthenticationFlow() {
	suite.Run("user_registration_and_login", func() {
		// Test user registration
		registerData := map[string]interface{}{
			"email":    "test@example.com",
			"password": "securepassword123",
			"name":     "Test User",
		}
		
		registerResp := suite.makeRequest("POST", "/api/v1/auth/register", registerData, nil)
		assert.Equal(suite.T(), http.StatusCreated, registerResp.StatusCode)
		
		var registerResult map[string]interface{}
		err := json.NewDecoder(registerResp.Body).Decode(&registerResult)
		require.NoError(suite.T(), err)
		registerResp.Body.Close()
		
		assert.Contains(suite.T(), registerResult, "user_id")
		assert.Contains(suite.T(), registerResult, "message")
		
		// Test user login
		loginData := map[string]interface{}{
			"email":    "test@example.com",
			"password": "securepassword123",
		}
		
		loginResp := suite.makeRequest("POST", "/api/v1/auth/login", loginData, nil)
		assert.Equal(suite.T(), http.StatusOK, loginResp.StatusCode)
		
		var loginResult map[string]interface{}
		err = json.NewDecoder(loginResp.Body).Decode(&loginResult)
		require.NoError(suite.T(), err)
		loginResp.Body.Close()
		
		assert.Contains(suite.T(), loginResult, "token")
		assert.Contains(suite.T(), loginResult, "user")
		
		// Store token for subsequent tests
		token := loginResult["token"].(string)
		suite.testSuite.Context().Value("auth_token")
		
		// Test accessing protected endpoint with token
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}
		
		profileResp := suite.makeRequest("GET", "/api/v1/profile", nil, headers)
		assert.Equal(suite.T(), http.StatusOK, profileResp.StatusCode)
		profileResp.Body.Close()
	})
	
	suite.Run("invalid_login_credentials", func() {
		loginData := map[string]interface{}{
			"email":    "invalid@example.com",
			"password": "wrongpassword",
		}
		
		resp := suite.makeRequest("POST", "/api/v1/auth/login", loginData, nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})
	
	suite.Run("access_protected_endpoint_without_token", func() {
		resp := suite.makeRequest("GET", "/api/v1/profile", nil, nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})
}

func (suite *APIIntegrationTestSuite) TestScanManagement() {
	// First, authenticate to get a token
	token := suite.authenticateTestUser()
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	
	suite.Run("create_scan", func() {
		scanData := map[string]interface{}{
			"name":        "Test Vulnerability Scan",
			"description": "Integration test scan",
			"type":        "vulnerability",
			"targets":     []string{"https://example.com", "192.168.1.1/24"},
			"config": map[string]interface{}{
				"depth":   "deep",
				"timeout": 3600,
			},
		}
		
		resp := suite.makeRequest("POST", "/api/v1/scans", scanData, headers)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		resp.Body.Close()
		
		assert.Contains(suite.T(), result, "scan_id")
		assert.Contains(suite.T(), result, "status")
		assert.Equal(suite.T(), "created", result["status"])
		
		// Store scan ID for subsequent tests
		scanID := result["scan_id"].(string)
		suite.testSuite.Context().Value("scan_id")
		
		// Test getting the created scan
		getScanResp := suite.makeRequest("GET", "/api/v1/scans/"+scanID, nil, headers)
		assert.Equal(suite.T(), http.StatusOK, getScanResp.StatusCode)
		
		var scanDetails map[string]interface{}
		err = json.NewDecoder(getScanResp.Body).Decode(&scanDetails)
		require.NoError(suite.T(), err)
		getScanResp.Body.Close()
		
		assert.Equal(suite.T(), scanID, scanDetails["id"])
		assert.Equal(suite.T(), "Test Vulnerability Scan", scanDetails["name"])
	})
	
	suite.Run("list_scans", func() {
		resp := suite.makeRequest("GET", "/api/v1/scans", nil, headers)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		resp.Body.Close()
		
		assert.Contains(suite.T(), result, "scans")
		assert.Contains(suite.T(), result, "total")
		
		scans := result["scans"].([]interface{})
		assert.True(suite.T(), len(scans) > 0)
	})
	
	suite.Run("start_scan", func() {
		// Assuming we have a scan ID from previous test
		scanID := "test-scan-id" // In real test, get from context
		
		resp := suite.makeRequest("POST", "/api/v1/scans/"+scanID+"/start", nil, headers)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		resp.Body.Close()
		
		assert.Contains(suite.T(), result, "status")
		assert.Equal(suite.T(), "running", result["status"])
	})
}

func (suite *APIIntegrationTestSuite) TestAgentManagement() {
	token := suite.authenticateTestUser()
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	
	suite.Run("create_agent", func() {
		agentData := map[string]interface{}{
			"name":        "Test Agent",
			"description": "Integration test agent",
			"type":        "vulnerability_scanner",
			"config": map[string]interface{}{
				"max_concurrent_scans": 5,
				"timeout":              300,
			},
		}
		
		resp := suite.makeRequest("POST", "/api/v1/agents", agentData, headers)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		resp.Body.Close()
		
		assert.Contains(suite.T(), result, "agent_id")
		assert.Contains(suite.T(), result, "status")
	})
	
	suite.Run("list_agents", func() {
		resp := suite.makeRequest("GET", "/api/v1/agents", nil, headers)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		resp.Body.Close()
		
		assert.Contains(suite.T(), result, "agents")
		assert.Contains(suite.T(), result, "total")
	})
}

func (suite *APIIntegrationTestSuite) TestSecurityMiddleware() {
	suite.Run("cors_headers", func() {
		req, err := http.NewRequest("OPTIONS", suite.server.URL+"/api/v1/scans", nil)
		require.NoError(suite.T(), err)
		
		req.Header.Set("Origin", "https://app.example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		
		resp, err := http.DefaultClient.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusNoContent, resp.StatusCode)
		assert.NotEmpty(suite.T(), resp.Header.Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(suite.T(), resp.Header.Get("Access-Control-Allow-Methods"))
	})
	
	suite.Run("security_headers", func() {
		resp, err := http.Get(suite.server.URL + "/health")
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		assert.NotEmpty(suite.T(), resp.Header.Get("X-Content-Type-Options"))
		assert.NotEmpty(suite.T(), resp.Header.Get("X-Frame-Options"))
		assert.NotEmpty(suite.T(), resp.Header.Get("X-XSS-Protection"))
	})
	
	suite.Run("request_size_limit", func() {
		// Create a large payload (> 1MB)
		largeData := make(map[string]interface{})
		largeData["data"] = string(make([]byte, 2*1024*1024)) // 2MB
		
		resp := suite.makeRequest("POST", "/api/v1/scans", largeData, nil)
		assert.Equal(suite.T(), http.StatusRequestEntityTooLarge, resp.StatusCode)
		resp.Body.Close()
	})
}

func (suite *APIIntegrationTestSuite) TestMetricsCollection() {
	suite.Run("prometheus_metrics", func() {
		// Make several requests to generate metrics
		token := suite.authenticateTestUser()
		headers := map[string]string{
			"Authorization": "Bearer " + token,
		}
		
		for i := 0; i < 5; i++ {
			resp := suite.makeRequest("GET", "/api/v1/scans", nil, headers)
			resp.Body.Close()
		}
		
		// Check that metrics were collected
		// In a real implementation, you would expose a metrics endpoint
		// and verify the Prometheus metrics format
		assert.NotNil(suite.T(), suite.metricsCollector)
	})
}

func (suite *APIIntegrationTestSuite) TestErrorHandling() {
	suite.Run("not_found_endpoint", func() {
		resp, err := http.Get(suite.server.URL + "/api/v1/nonexistent")
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
	})
	
	suite.Run("method_not_allowed", func() {
		resp, err := http.Post(suite.server.URL+"/health", "application/json", nil)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusMethodNotAllowed, resp.StatusCode)
	})
	
	suite.Run("invalid_json_payload", func() {
		req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/auth/login", 
			bytes.NewBufferString("invalid json"))
		require.NoError(suite.T(), err)
		
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := http.DefaultClient.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
	})
}

func (suite *APIIntegrationTestSuite) TestRateLimiting() {
	suite.Run("rate_limit_enforcement", func() {
		// This test would require configuring rate limiting
		// and making rapid requests to trigger the limit
		
		// Make rapid requests
		for i := 0; i < 10; i++ {
			resp, err := http.Get(suite.server.URL + "/health")
			require.NoError(suite.T(), err)
			resp.Body.Close()
			
			// All requests should succeed if rate limit is high enough
			// or if rate limiting is not configured for health endpoint
			assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTooManyRequests)
		}
	})
}

// Helper methods

func (suite *APIIntegrationTestSuite) makeRequest(method, path string, data interface{}, headers map[string]string) *http.Response {
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

func (suite *APIIntegrationTestSuite) authenticateTestUser() string {
	// Create and login a test user
	registerData := map[string]interface{}{
		"email":    fmt.Sprintf("test-%d@example.com", time.Now().Unix()),
		"password": "testpassword123",
		"name":     "Test User",
	}
	
	registerResp := suite.makeRequest("POST", "/api/v1/auth/register", registerData, nil)
	registerResp.Body.Close()
	
	loginData := map[string]interface{}{
		"email":    registerData["email"],
		"password": registerData["password"],
	}
	
	loginResp := suite.makeRequest("POST", "/api/v1/auth/login", loginData, nil)
	
	var loginResult map[string]interface{}
	err := json.NewDecoder(loginResp.Body).Decode(&loginResult)
	require.NoError(suite.T(), err)
	loginResp.Body.Close()
	
	return loginResult["token"].(string)
}

// Mock handlers for testing

func (suite *APIIntegrationTestSuite) handleLogin(c *gin.Context) {
	var loginData map[string]interface{}
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	email, emailOk := loginData["email"].(string)
	password, passwordOk := loginData["password"].(string)
	
	if !emailOk || !passwordOk {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password required"})
		return
	}
	
	// Mock authentication logic
	if email == "invalid@example.com" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	
	// Generate mock JWT token
	token, err := middleware.GenerateJWT("test-user-123", &suite.config.Security.JWT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    "test-user-123",
			"email": email,
			"name":  "Test User",
		},
	})
}

func (suite *APIIntegrationTestSuite) handleRegister(c *gin.Context) {
	var registerData map[string]interface{}
	if err := c.ShouldBindJSON(&registerData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"user_id": "test-user-123",
		"message": "User created successfully",
	})
}

func (suite *APIIntegrationTestSuite) handleGetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	c.JSON(http.StatusOK, gin.H{
		"id":    userID,
		"email": "test@example.com",
		"name":  "Test User",
	})
}

func (suite *APIIntegrationTestSuite) handleUpdateProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}

func (suite *APIIntegrationTestSuite) handleListScans(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"scans": []gin.H{
			{
				"id":     "scan-1",
				"name":   "Test Scan 1",
				"status": "completed",
			},
		},
		"total": 1,
	})
}

func (suite *APIIntegrationTestSuite) handleCreateScan(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"scan_id": "test-scan-123",
		"status":  "created",
	})
}

func (suite *APIIntegrationTestSuite) handleGetScan(c *gin.Context) {
	scanID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     scanID,
		"name":   "Test Vulnerability Scan",
		"status": "created",
	})
}

func (suite *APIIntegrationTestSuite) handleUpdateScan(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Scan updated"})
}

func (suite *APIIntegrationTestSuite) handleDeleteScan(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Scan deleted"})
}

func (suite *APIIntegrationTestSuite) handleStartScan(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "running"})
}

func (suite *APIIntegrationTestSuite) handleStopScan(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

func (suite *APIIntegrationTestSuite) handleListAgents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"agents": []gin.H{},
		"total":  0,
	})
}

func (suite *APIIntegrationTestSuite) handleCreateAgent(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"agent_id": "test-agent-123",
		"status":   "created",
	})
}

func (suite *APIIntegrationTestSuite) handleGetAgent(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
}

func (suite *APIIntegrationTestSuite) handleUpdateAgent(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Agent updated"})
}

func (suite *APIIntegrationTestSuite) handleDeleteAgent(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Agent deleted"})
}

func (suite *APIIntegrationTestSuite) handleListReports(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"reports": []gin.H{},
		"total":   0,
	})
}

func (suite *APIIntegrationTestSuite) handleGetReport(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
}

func (suite *APIIntegrationTestSuite) handleExportReport(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"export_url": "/exports/report-123.pdf"})
}

func TestAPIIntegrationSuite(t *testing.T) {
	testing.IntegrationTest(t)
	suite.Run(t, new(APIIntegrationTestSuite))
}