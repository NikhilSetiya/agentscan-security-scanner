package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

type ScanAPIIntegrationTestSuite struct {
	suite.Suite
	testSuite *testing.TestSuite
	router    *gin.Engine
	server    *httptest.Server
	config    *config.Config
	authToken string
	scans     map[string]map[string]interface{} // Mock scan storage
}

func (suite *ScanAPIIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	
	suite.testSuite = testing.NewTestSuite(suite.T())
	suite.config = suite.testSuite.MockConfig()
	suite.scans = make(map[string]map[string]interface{})
	
	// Setup router
	suite.router = suite.setupScanRouter()
	suite.server = suite.testSuite.SetupHTTPServer(suite.router)
	
	// Get auth token
	suite.authToken = suite.getAuthToken()
}

func (suite *ScanAPIIntegrationTestSuite) TearDownSuite() {
	suite.testSuite.Cleanup()
}

func (suite *ScanAPIIntegrationTestSuite) setupScanRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	
	// Auth endpoint
	router.POST("/auth/login", suite.handleLogin)
	
	// Scan API endpoints
	api := router.Group("/api/v1")
	api.Use(middleware.JWTAuth(&suite.config.Security.JWT))
	{
		scans := api.Group("/scans")
		{
			scans.GET("", suite.handleListScans)
			scans.POST("", suite.handleCreateScan)
			scans.GET("/:id", suite.handleGetScan)
			scans.PUT("/:id", suite.handleUpdateScan)
			scans.DELETE("/:id", suite.handleDeleteScan)
			scans.POST("/:id/start", suite.handleStartScan)
			scans.POST("/:id/stop", suite.handleStopScan)
			scans.POST("/:id/pause", suite.handlePauseScan)
			scans.POST("/:id/resume", suite.handleResumeScan)
			scans.GET("/:id/results", suite.handleGetScanResults)
			scans.GET("/:id/logs", suite.handleGetScanLogs)
			scans.POST("/:id/export", suite.handleExportScan)
		}
		
		// Scan templates
		templates := api.Group("/scan-templates")
		{
			templates.GET("", suite.handleListScanTemplates)
			templates.POST("", suite.handleCreateScanTemplate)
			templates.GET("/:id", suite.handleGetScanTemplate)
			templates.PUT("/:id", suite.handleUpdateScanTemplate)
			templates.DELETE("/:id", suite.handleDeleteScanTemplate)
		}
	}
	
	return router
}

func (suite *ScanAPIIntegrationTestSuite) TestScanCRUDOperations() {
	headers := map[string]string{
		"Authorization": "Bearer " + suite.authToken,
	}
	
	var scanID string
	
	suite.Run("create_scan", func() {
		scanData := map[string]interface{}{
			"name":        "Test Vulnerability Scan",
			"description": "Integration test scan for vulnerability assessment",
			"type":        "vulnerability",
			"targets": []string{
				"https://example.com",
				"192.168.1.0/24",
			},
			"config": map[string]interface{}{
				"depth":       "deep",
				"timeout":     3600,
				"max_threads": 10,
				"scan_types": []string{"port_scan", "vulnerability_scan", "web_scan"},
			},
			"schedule": map[string]interface{}{
				"enabled":   false,
				"frequency": "weekly",
			},
		}
		
		resp := suite.makeRequest("POST", "/api/v1/scans", scanData, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "scan_id")
		assert.Contains(suite.T(), result, "status")
		assert.Equal(suite.T(), "created", result["status"])
		
		scanID = result["scan_id"].(string)
		assert.NotEmpty(suite.T(), scanID)
	})
	
	suite.Run("get_scan", func() {
		resp := suite.makeRequest("GET", "/api/v1/scans/"+scanID, nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), scanID, result["id"])
		assert.Equal(suite.T(), "Test Vulnerability Scan", result["name"])
		assert.Equal(suite.T(), "vulnerability", result["type"])
		assert.Contains(suite.T(), result, "targets")
		assert.Contains(suite.T(), result, "config")
		assert.Contains(suite.T(), result, "created_at")
		assert.Contains(suite.T(), result, "updated_at")
	})
	
	suite.Run("update_scan", func() {
		updateData := map[string]interface{}{
			"name":        "Updated Vulnerability Scan",
			"description": "Updated description for integration test",
			"config": map[string]interface{}{
				"depth":       "medium",
				"timeout":     1800,
				"max_threads": 5,
			},
		}
		
		resp := suite.makeRequest("PUT", "/api/v1/scans/"+scanID, updateData, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), "Updated Vulnerability Scan", result["name"])
		assert.Equal(suite.T(), "Updated description for integration test", result["description"])
	})
	
	suite.Run("list_scans", func() {
		resp := suite.makeRequest("GET", "/api/v1/scans", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "scans")
		assert.Contains(suite.T(), result, "total")
		assert.Contains(suite.T(), result, "page")
		assert.Contains(suite.T(), result, "limit")
		
		scans := result["scans"].([]interface{})
		assert.True(suite.T(), len(scans) > 0)
		
		// Find our scan in the list
		found := false
		for _, scan := range scans {
			scanMap := scan.(map[string]interface{})
			if scanMap["id"] == scanID {
				found = true
				assert.Equal(suite.T(), "Updated Vulnerability Scan", scanMap["name"])
				break
			}
		}
		assert.True(suite.T(), found, "Created scan should be in the list")
	})
	
	suite.Run("delete_scan", func() {
		resp := suite.makeRequest("DELETE", "/api/v1/scans/"+scanID, nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "message")
		assert.Contains(suite.T(), result["message"], "deleted")
		
		// Verify scan is deleted
		getResp := suite.makeRequest("GET", "/api/v1/scans/"+scanID, nil, headers)
		getResp.Body.Close()
		assert.Equal(suite.T(), http.StatusNotFound, getResp.StatusCode)
	})
}

func (suite *ScanAPIIntegrationTestSuite) TestScanLifecycleOperations() {
	headers := map[string]string{
		"Authorization": "Bearer " + suite.authToken,
	}
	
	// Create a scan for lifecycle testing
	scanData := map[string]interface{}{
		"name":    "Lifecycle Test Scan",
		"type":    "vulnerability",
		"targets": []string{"https://example.com"},
		"config": map[string]interface{}{
			"timeout": 1800,
		},
	}
	
	createResp := suite.makeRequest("POST", "/api/v1/scans", scanData, headers)
	defer createResp.Body.Close()
	
	var createResult map[string]interface{}
	err := json.NewDecoder(createResp.Body).Decode(&createResult)
	require.NoError(suite.T(), err)
	
	scanID := createResult["scan_id"].(string)
	
	suite.Run("start_scan", func() {
		resp := suite.makeRequest("POST", "/api/v1/scans/"+scanID+"/start", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "status")
		assert.Equal(suite.T(), "running", result["status"])
		assert.Contains(suite.T(), result, "started_at")
	})
	
	suite.Run("pause_scan", func() {
		resp := suite.makeRequest("POST", "/api/v1/scans/"+scanID+"/pause", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), "paused", result["status"])
	})
	
	suite.Run("resume_scan", func() {
		resp := suite.makeRequest("POST", "/api/v1/scans/"+scanID+"/resume", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), "running", result["status"])
	})
	
	suite.Run("stop_scan", func() {
		resp := suite.makeRequest("POST", "/api/v1/scans/"+scanID+"/stop", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), "stopped", result["status"])
		assert.Contains(suite.T(), result, "stopped_at")
	})
}

func (suite *ScanAPIIntegrationTestSuite) TestScanResults() {
	headers := map[string]string{
		"Authorization": "Bearer " + suite.authToken,
	}
	
	// Create and complete a scan
	scanID := suite.createCompletedScan()
	
	suite.Run("get_scan_results", func() {
		resp := suite.makeRequest("GET", "/api/v1/scans/"+scanID+"/results", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "scan_id")
		assert.Contains(suite.T(), result, "status")
		assert.Contains(suite.T(), result, "summary")
		assert.Contains(suite.T(), result, "vulnerabilities")
		assert.Contains(suite.T(), result, "completed_at")
		
		summary := result["summary"].(map[string]interface{})
		assert.Contains(suite.T(), summary, "total_vulnerabilities")
		assert.Contains(suite.T(), summary, "critical")
		assert.Contains(suite.T(), summary, "high")
		assert.Contains(suite.T(), summary, "medium")
		assert.Contains(suite.T(), summary, "low")
	})
	
	suite.Run("get_scan_results_with_filters", func() {
		// Test filtering by severity
		resp := suite.makeRequest("GET", "/api/v1/scans/"+scanID+"/results?severity=critical,high", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		vulnerabilities := result["vulnerabilities"].([]interface{})
		for _, vuln := range vulnerabilities {
			vulnMap := vuln.(map[string]interface{})
			severity := vulnMap["severity"].(string)
			assert.True(suite.T(), severity == "critical" || severity == "high")
		}
	})
	
	suite.Run("get_scan_logs", func() {
		resp := suite.makeRequest("GET", "/api/v1/scans/"+scanID+"/logs", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "logs")
		assert.Contains(suite.T(), result, "total")
		
		logs := result["logs"].([]interface{})
		assert.True(suite.T(), len(logs) > 0)
		
		// Check log entry structure
		logEntry := logs[0].(map[string]interface{})
		assert.Contains(suite.T(), logEntry, "timestamp")
		assert.Contains(suite.T(), logEntry, "level")
		assert.Contains(suite.T(), logEntry, "message")
	})
}

func (suite *ScanAPIIntegrationTestSuite) TestScanExport() {
	headers := map[string]string{
		"Authorization": "Bearer " + suite.authToken,
	}
	
	scanID := suite.createCompletedScan()
	
	suite.Run("export_scan_pdf", func() {
		exportData := map[string]interface{}{
			"format": "pdf",
			"options": map[string]interface{}{
				"include_summary":        true,
				"include_vulnerabilities": true,
				"include_recommendations": true,
			},
		}
		
		resp := suite.makeRequest("POST", "/api/v1/scans/"+scanID+"/export", exportData, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "export_id")
		assert.Contains(suite.T(), result, "download_url")
		assert.Contains(suite.T(), result, "expires_at")
	})
	
	suite.Run("export_scan_json", func() {
		exportData := map[string]interface{}{
			"format": "json",
		}
		
		resp := suite.makeRequest("POST", "/api/v1/scans/"+scanID+"/export", exportData, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "export_id")
		assert.Contains(suite.T(), result, "download_url")
	})
}

func (suite *ScanAPIIntegrationTestSuite) TestScanTemplates() {
	headers := map[string]string{
		"Authorization": "Bearer " + suite.authToken,
	}
	
	var templateID string
	
	suite.Run("create_scan_template", func() {
		templateData := map[string]interface{}{
			"name":        "Web Application Security Template",
			"description": "Template for web application security scans",
			"type":        "vulnerability",
			"config": map[string]interface{}{
				"scan_types": []string{"web_scan", "vulnerability_scan"},
				"depth":      "deep",
				"timeout":    3600,
			},
			"default_targets": []string{"https://"},
		}
		
		resp := suite.makeRequest("POST", "/api/v1/scan-templates", templateData, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "template_id")
		templateID = result["template_id"].(string)
	})
	
	suite.Run("list_scan_templates", func() {
		resp := suite.makeRequest("GET", "/api/v1/scan-templates", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "templates")
		templates := result["templates"].([]interface{})
		assert.True(suite.T(), len(templates) > 0)
	})
	
	suite.Run("create_scan_from_template", func() {
		scanData := map[string]interface{}{
			"name":        "Scan from Template",
			"template_id": templateID,
			"targets":     []string{"https://example.com"},
		}
		
		resp := suite.makeRequest("POST", "/api/v1/scans", scanData, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "scan_id")
		assert.Equal(suite.T(), "vulnerability", result["type"])
	})
}

func (suite *ScanAPIIntegrationTestSuite) TestScanValidation() {
	headers := map[string]string{
		"Authorization": "Bearer " + suite.authToken,
	}
	
	suite.Run("invalid_scan_data", func() {
		testCases := []struct {
			name string
			data map[string]interface{}
		}{
			{
				name: "missing_name",
				data: map[string]interface{}{
					"type":    "vulnerability",
					"targets": []string{"https://example.com"},
				},
			},
			{
				name: "invalid_type",
				data: map[string]interface{}{
					"name":    "Test Scan",
					"type":    "invalid_type",
					"targets": []string{"https://example.com"},
				},
			},
			{
				name: "empty_targets",
				data: map[string]interface{}{
					"name":    "Test Scan",
					"type":    "vulnerability",
					"targets": []string{},
				},
			},
			{
				name: "invalid_target_format",
				data: map[string]interface{}{
					"name":    "Test Scan",
					"type":    "vulnerability",
					"targets": []string{"invalid-url"},
				},
			},
		}
		
		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				resp := suite.makeRequest("POST", "/api/v1/scans", tc.data, headers)
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

func (suite *ScanAPIIntegrationTestSuite) TestScanPagination() {
	headers := map[string]string{
		"Authorization": "Bearer " + suite.authToken,
	}
	
	// Create multiple scans for pagination testing
	for i := 0; i < 15; i++ {
		scanData := map[string]interface{}{
			"name":    fmt.Sprintf("Pagination Test Scan %d", i+1),
			"type":    "vulnerability",
			"targets": []string{"https://example.com"},
		}
		
		resp := suite.makeRequest("POST", "/api/v1/scans", scanData, headers)
		resp.Body.Close()
	}
	
	suite.Run("paginated_scan_list", func() {
		// Test first page
		resp := suite.makeRequest("GET", "/api/v1/scans?page=1&limit=10", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), result, "scans")
		assert.Contains(suite.T(), result, "total")
		assert.Contains(suite.T(), result, "page")
		assert.Contains(suite.T(), result, "limit")
		assert.Contains(suite.T(), result, "has_next")
		
		scans := result["scans"].([]interface{})
		assert.Equal(suite.T(), 10, len(scans))
		assert.Equal(suite.T(), float64(1), result["page"])
		assert.Equal(suite.T(), float64(10), result["limit"])
		assert.True(suite.T(), result["has_next"].(bool))
	})
	
	suite.Run("scan_list_sorting", func() {
		resp := suite.makeRequest("GET", "/api/v1/scans?sort=created_at&order=desc", nil, headers)
		defer resp.Body.Close()
		
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(suite.T(), err)
		
		scans := result["scans"].([]interface{})
		assert.True(suite.T(), len(scans) > 1)
		
		// Verify sorting (newest first)
		for i := 1; i < len(scans); i++ {
			scan1 := scans[i-1].(map[string]interface{})
			scan2 := scans[i].(map[string]interface{})
			
			time1, _ := time.Parse(time.RFC3339, scan1["created_at"].(string))
			time2, _ := time.Parse(time.RFC3339, scan2["created_at"].(string))
			
			assert.True(suite.T(), time1.After(time2) || time1.Equal(time2))
		}
	})
}

// Helper methods

func (suite *ScanAPIIntegrationTestSuite) makeRequest(method, path string, data interface{}, headers map[string]string) *http.Response {
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

func (suite *ScanAPIIntegrationTestSuite) getAuthToken() string {
	loginData := map[string]interface{}{
		"email":    "test@example.com",
		"password": "password123",
	}
	
	resp := suite.makeRequest("POST", "/auth/login", loginData, nil)
	defer resp.Body.Close()
	
	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	
	return result["token"].(string)
}

func (suite *ScanAPIIntegrationTestSuite) createCompletedScan() string {
	headers := map[string]string{
		"Authorization": "Bearer " + suite.authToken,
	}
	
	scanData := map[string]interface{}{
		"name":    "Completed Test Scan",
		"type":    "vulnerability",
		"targets": []string{"https://example.com"},
	}
	
	resp := suite.makeRequest("POST", "/api/v1/scans", scanData, headers)
	defer resp.Body.Close()
	
	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(suite.T(), err)
	
	scanID := result["scan_id"].(string)
	
	// Mark scan as completed with results
	suite.scans[scanID] = map[string]interface{}{
		"id":     scanID,
		"status": "completed",
		"results": map[string]interface{}{
			"summary": map[string]interface{}{
				"total_vulnerabilities": 5,
				"critical":              1,
				"high":                  2,
				"medium":                2,
				"low":                   0,
			},
			"vulnerabilities": []map[string]interface{}{
				{
					"id":          "vuln-1",
					"title":       "SQL Injection",
					"severity":    "critical",
					"description": "SQL injection vulnerability found",
				},
				{
					"id":          "vuln-2",
					"title":       "XSS Vulnerability",
					"severity":    "high",
					"description": "Cross-site scripting vulnerability",
				},
			},
		},
	}
	
	return scanID
}

// Mock handlers

func (suite *ScanAPIIntegrationTestSuite) handleLogin(c *gin.Context) {
	token, _ := middleware.GenerateJWT("test-user-123", &suite.config.Security.JWT)
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    "test-user-123",
			"email": "test@example.com",
		},
	})
}

func (suite *ScanAPIIntegrationTestSuite) handleListScans(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	// Mock scan list
	scans := []gin.H{}
	for id, scan := range suite.scans {
		scans = append(scans, gin.H{
			"id":         id,
			"name":       scan["name"],
			"type":       scan["type"],
			"status":     scan["status"],
			"created_at": time.Now().Add(-time.Hour).Format(time.RFC3339),
			"updated_at": time.Now().Format(time.RFC3339),
		})
	}
	
	// Add some default scans if none exist
	if len(scans) == 0 {
		for i := 0; i < 15; i++ {
			scans = append(scans, gin.H{
				"id":         fmt.Sprintf("scan-%d", i+1),
				"name":       fmt.Sprintf("Test Scan %d", i+1),
				"type":       "vulnerability",
				"status":     "completed",
				"created_at": time.Now().Add(-time.Duration(i)*time.Hour).Format(time.RFC3339),
				"updated_at": time.Now().Add(-time.Duration(i)*time.Minute).Format(time.RFC3339),
			})
		}
	}
	
	// Apply pagination
	start := (page - 1) * limit
	end := start + limit
	if end > len(scans) {
		end = len(scans)
	}
	
	paginatedScans := scans[start:end]
	hasNext := end < len(scans)
	
	c.JSON(http.StatusOK, gin.H{
		"scans":    paginatedScans,
		"total":    len(scans),
		"page":     page,
		"limit":    limit,
		"has_next": hasNext,
	})
}

func (suite *ScanAPIIntegrationTestSuite) handleCreateScan(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	// Validation
	name, _ := data["name"].(string)
	scanType, _ := data["type"].(string)
	targets, _ := data["targets"].([]interface{})
	
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}
	
	if scanType == "" || (scanType != "vulnerability" && scanType != "compliance" && scanType != "network") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan type"})
		return
	}
	
	if len(targets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one target is required"})
		return
	}
	
	scanID := fmt.Sprintf("scan-%d", time.Now().Unix())
	
	// Store scan
	suite.scans[scanID] = map[string]interface{}{
		"id":          scanID,
		"name":        name,
		"type":        scanType,
		"status":      "created",
		"targets":     targets,
		"config":      data["config"],
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"scan_id": scanID,
		"status":  "created",
	})
}

func (suite *ScanAPIIntegrationTestSuite) handleGetScan(c *gin.Context) {
	scanID := c.Param("id")
	
	scan, exists := suite.scans[scanID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	c.JSON(http.StatusOK, scan)
}

func (suite *ScanAPIIntegrationTestSuite) handleUpdateScan(c *gin.Context) {
	scanID := c.Param("id")
	
	scan, exists := suite.scans[scanID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	// Update scan
	for key, value := range updateData {
		scan[key] = value
	}
	scan["updated_at"] = time.Now()
	
	suite.scans[scanID] = scan
	
	c.JSON(http.StatusOK, scan)
}

func (suite *ScanAPIIntegrationTestSuite) handleDeleteScan(c *gin.Context) {
	scanID := c.Param("id")
	
	if _, exists := suite.scans[scanID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	delete(suite.scans, scanID)
	
	c.JSON(http.StatusOK, gin.H{"message": "Scan deleted successfully"})
}

func (suite *ScanAPIIntegrationTestSuite) handleStartScan(c *gin.Context) {
	scanID := c.Param("id")
	
	scan, exists := suite.scans[scanID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	scan["status"] = "running"
	scan["started_at"] = time.Now()
	suite.scans[scanID] = scan
	
	c.JSON(http.StatusOK, gin.H{
		"status":     "running",
		"started_at": scan["started_at"],
	})
}

func (suite *ScanAPIIntegrationTestSuite) handleStopScan(c *gin.Context) {
	scanID := c.Param("id")
	
	scan, exists := suite.scans[scanID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	scan["status"] = "stopped"
	scan["stopped_at"] = time.Now()
	suite.scans[scanID] = scan
	
	c.JSON(http.StatusOK, gin.H{
		"status":     "stopped",
		"stopped_at": scan["stopped_at"],
	})
}

func (suite *ScanAPIIntegrationTestSuite) handlePauseScan(c *gin.Context) {
	scanID := c.Param("id")
	
	scan, exists := suite.scans[scanID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	scan["status"] = "paused"
	suite.scans[scanID] = scan
	
	c.JSON(http.StatusOK, gin.H{"status": "paused"})
}

func (suite *ScanAPIIntegrationTestSuite) handleResumeScan(c *gin.Context) {
	scanID := c.Param("id")
	
	scan, exists := suite.scans[scanID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	scan["status"] = "running"
	suite.scans[scanID] = scan
	
	c.JSON(http.StatusOK, gin.H{"status": "running"})
}

func (suite *ScanAPIIntegrationTestSuite) handleGetScanResults(c *gin.Context) {
	scanID := c.Param("id")
	
	scan, exists := suite.scans[scanID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	results, hasResults := scan["results"]
	if !hasResults {
		c.JSON(http.StatusNotFound, gin.H{"error": "No results available"})
		return
	}
	
	// Apply severity filter if provided
	severity := c.Query("severity")
	if severity != "" {
		// Filter vulnerabilities by severity
		// This is a simplified implementation
	}
	
	response := gin.H{
		"scan_id":      scanID,
		"status":       scan["status"],
		"completed_at": time.Now(),
	}
	
	// Merge results
	if resultsMap, ok := results.(map[string]interface{}); ok {
		for key, value := range resultsMap {
			response[key] = value
		}
	}
	
	c.JSON(http.StatusOK, response)
}

func (suite *ScanAPIIntegrationTestSuite) handleGetScanLogs(c *gin.Context) {
	scanID := c.Param("id")
	
	if _, exists := suite.scans[scanID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	// Mock logs
	logs := []gin.H{
		{
			"timestamp": time.Now().Add(-time.Hour).Format(time.RFC3339),
			"level":     "info",
			"message":   "Scan started",
		},
		{
			"timestamp": time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			"level":     "info",
			"message":   "Port scan completed",
		},
		{
			"timestamp": time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
			"level":     "warning",
			"message":   "Vulnerability detected",
		},
		{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     "info",
			"message":   "Scan completed",
		},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
	})
}

func (suite *ScanAPIIntegrationTestSuite) handleExportScan(c *gin.Context) {
	scanID := c.Param("id")
	
	if _, exists := suite.scans[scanID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}
	
	var exportData map[string]interface{}
	if err := c.ShouldBindJSON(&exportData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	format, _ := exportData["format"].(string)
	if format == "" {
		format = "pdf"
	}
	
	exportID := fmt.Sprintf("export-%d", time.Now().Unix())
	downloadURL := fmt.Sprintf("/downloads/%s.%s", exportID, format)
	
	c.JSON(http.StatusOK, gin.H{
		"export_id":    exportID,
		"download_url": downloadURL,
		"expires_at":   time.Now().Add(24 * time.Hour),
	})
}

func (suite *ScanAPIIntegrationTestSuite) handleListScanTemplates(c *gin.Context) {
	templates := []gin.H{
		{
			"id":          "template-1",
			"name":        "Web Application Security Template",
			"description": "Template for web application security scans",
			"type":        "vulnerability",
		},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"total":     len(templates),
	})
}

func (suite *ScanAPIIntegrationTestSuite) handleCreateScanTemplate(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	
	templateID := fmt.Sprintf("template-%d", time.Now().Unix())
	
	c.JSON(http.StatusCreated, gin.H{
		"template_id": templateID,
		"message":     "Template created successfully",
	})
}

func (suite *ScanAPIIntegrationTestSuite) handleGetScanTemplate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
}

func (suite *ScanAPIIntegrationTestSuite) handleUpdateScanTemplate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Template updated"})
}

func (suite *ScanAPIIntegrationTestSuite) handleDeleteScanTemplate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Template deleted"})
}

func TestScanAPIIntegrationSuite(t *testing.T) {
	testing.IntegrationTest(t)
	suite.Run(t, new(ScanAPIIntegrationTestSuite))
}