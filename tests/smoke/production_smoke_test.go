package smoke

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ProductionSmokeTestSuite runs critical smoke tests against production environment
type ProductionSmokeTestSuite struct {
	apiURL      string
	frontendURL string
	client      *http.Client
	authToken   string
}

// NewProductionSmokeTestSuite creates a new smoke test suite
func NewProductionSmokeTestSuite() *ProductionSmokeTestSuite {
	return &ProductionSmokeTestSuite{
		apiURL:      getEnvOrDefault("API_URL", "https://agentscan-prod.fly.dev"),
		frontendURL: getEnvOrDefault("FRONTEND_URL", "https://agentscan.vercel.app"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TestProductionSmokeTests runs all critical smoke tests
func TestProductionSmokeTests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping smoke tests in short mode")
	}

	// Only run if SMOKE_TEST_ENV is set to production
	if os.Getenv("SMOKE_TEST_ENV") != "production" {
		t.Skip("Skipping production smoke tests (set SMOKE_TEST_ENV=production to run)")
	}

	suite := NewProductionSmokeTestSuite()

	t.Run("Infrastructure", func(t *testing.T) {
		suite.testInfrastructure(t)
	})

	t.Run("Authentication", func(t *testing.T) {
		suite.testAuthentication(t)
	})

	t.Run("Core_Functionality", func(t *testing.T) {
		suite.testCoreFunctionality(t)
	})

	t.Run("Security", func(t *testing.T) {
		suite.testSecurity(t)
	})

	t.Run("Performance", func(t *testing.T) {
		suite.testPerformance(t)
	})
}

// testInfrastructure tests basic infrastructure health
func (s *ProductionSmokeTestSuite) testInfrastructure(t *testing.T) {
	t.Run("API_Health_Check", func(t *testing.T) {
		resp, err := s.client.Get(s.apiURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "healthy", health["status"])
		assert.Contains(t, health, "timestamp")
		assert.Contains(t, health, "version")

		// Check individual service health
		if checks, ok := health["checks"].(map[string]interface{}); ok {
			for service, check := range checks {
				checkMap := check.(map[string]interface{})
				assert.Equal(t, "healthy", checkMap["status"], 
					"Service %s should be healthy", service)
			}
		}
	})

	t.Run("API_Readiness_Check", func(t *testing.T) {
		resp, err := s.client.Get(s.apiURL + "/ready")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Frontend_Accessibility", func(t *testing.T) {
		resp, err := s.client.Get(s.frontendURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	})

	t.Run("HTTPS_Enforcement", func(t *testing.T) {
		// Test that HTTP redirects to HTTPS
		httpURL := strings.Replace(s.apiURL, "https://", "http://", 1)
		
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
			Timeout: 10 * time.Second,
		}

		resp, err := client.Get(httpURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should redirect to HTTPS
		assert.True(t, resp.StatusCode == http.StatusMovedPermanently || 
			resp.StatusCode == http.StatusFound ||
			resp.StatusCode == http.StatusForbidden,
			"HTTP should redirect to HTTPS or be forbidden")
	})

	t.Run("Security_Headers", func(t *testing.T) {
		resp, err := s.client.Get(s.apiURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check required security headers
		requiredHeaders := map[string]string{
			"Strict-Transport-Security": "",
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"X-XSS-Protection":          "1; mode=block",
		}

		for header, expectedValue := range requiredHeaders {
			headerValue := resp.Header.Get(header)
			assert.NotEmpty(t, headerValue, "Header %s should be present", header)
			
			if expectedValue != "" {
				assert.Contains(t, headerValue, expectedValue,
					"Header %s should contain %s", header, expectedValue)
			}
		}
	})

	t.Run("Metrics_Endpoint", func(t *testing.T) {
		resp, err := s.client.Get(s.apiURL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

		// Read response body to check for Prometheus metrics format
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		body := buf.String()

		assert.Contains(t, body, "# HELP", "Should contain Prometheus metrics")
		assert.Contains(t, body, "# TYPE", "Should contain Prometheus metrics")
	})
}

// testAuthentication tests authentication and authorization flows
func (s *ProductionSmokeTestSuite) testAuthentication(t *testing.T) {
	t.Run("Registration_Flow", func(t *testing.T) {
		// Create unique test user
		timestamp := time.Now().Unix()
		testUser := map[string]interface{}{
			"email":    fmt.Sprintf("smoketest+%d@example.com", timestamp),
			"password": "SmokeTest123!@#",
			"name":     "Smoke Test User",
		}

		// Test user registration
		resp := s.makeRequest(t, "POST", "/auth/register", testUser)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "user_id")
		assert.Contains(t, result, "message")
	})

	t.Run("Login_Flow", func(t *testing.T) {
		// Use test credentials from environment or create test user
		email := getEnvOrDefault("SMOKE_TEST_EMAIL", "smoketest@example.com")
		password := getEnvOrDefault("SMOKE_TEST_PASSWORD", "SmokeTest123!")

		loginData := map[string]interface{}{
			"email":    email,
			"password": password,
		}

		resp := s.makeRequest(t, "POST", "/auth/login", loginData)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			// If test user doesn't exist, create it first
			s.createTestUser(t, email, password)
			
			// Try login again
			resp = s.makeRequest(t, "POST", "/auth/login", loginData)
			defer resp.Body.Close()
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "token")
		assert.Contains(t, result, "user")

		// Store token for subsequent tests
		s.authToken = result["token"].(string)
		assert.NotEmpty(t, s.authToken)
	})

	t.Run("Protected_Endpoint_Access", func(t *testing.T) {
		if s.authToken == "" {
			t.Skip("No auth token available")
		}

		// Test accessing protected endpoint with valid token
		req, err := http.NewRequest("GET", s.apiURL+"/api/v1/profile", nil)
		require.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+s.authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var profile map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&profile)
		require.NoError(t, err)

		assert.Contains(t, profile, "id")
		assert.Contains(t, profile, "email")
	})

	t.Run("Unauthorized_Access", func(t *testing.T) {
		// Test accessing protected endpoint without token
		resp, err := s.client.Get(s.apiURL + "/api/v1/profile")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// testCoreFunctionality tests core application functionality
func (s *ProductionSmokeTestSuite) testCoreFunctionality(t *testing.T) {
	if s.authToken == "" {
		t.Skip("No auth token available for core functionality tests")
	}

	t.Run("Scan_Management", func(t *testing.T) {
		// Test creating a scan
		scanData := map[string]interface{}{
			"name":        "Production Smoke Test Scan",
			"description": "Automated smoke test scan",
			"type":        "vulnerability",
			"targets":     []string{"https://httpbin.org"},
			"config": map[string]interface{}{
				"timeout": 300,
				"depth":   "shallow",
			},
		}

		resp := s.makeAuthenticatedRequest(t, "POST", "/api/v1/scans", scanData)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "scan_id")
		scanID := result["scan_id"].(string)

		// Test retrieving the scan
		resp = s.makeAuthenticatedRequest(t, "GET", "/api/v1/scans/"+scanID, nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var scan map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&scan)
		require.NoError(t, err)

		assert.Equal(t, scanID, scan["id"])
		assert.Equal(t, "Production Smoke Test Scan", scan["name"])

		// Test listing scans
		resp = s.makeAuthenticatedRequest(t, "GET", "/api/v1/scans", nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var scanList map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&scanList)
		require.NoError(t, err)

		assert.Contains(t, scanList, "scans")
		assert.Contains(t, scanList, "total")

		scans := scanList["scans"].([]interface{})
		assert.True(t, len(scans) > 0)

		// Clean up - delete the test scan
		resp = s.makeAuthenticatedRequest(t, "DELETE", "/api/v1/scans/"+scanID, nil)
		resp.Body.Close()
	})

	t.Run("User_Profile_Management", func(t *testing.T) {
		// Test getting user profile
		resp := s.makeAuthenticatedRequest(t, "GET", "/api/v1/profile", nil)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var profile map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&profile)
		require.NoError(t, err)

		assert.Contains(t, profile, "id")
		assert.Contains(t, profile, "email")
		assert.Contains(t, profile, "name")

		// Test updating profile
		updateData := map[string]interface{}{
			"name": "Updated Smoke Test User",
		}

		resp = s.makeAuthenticatedRequest(t, "PUT", "/api/v1/profile", updateData)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// testSecurity tests security measures
func (s *ProductionSmokeTestSuite) testSecurity(t *testing.T) {
	t.Run("CORS_Configuration", func(t *testing.T) {
		// Test CORS preflight request
		req, err := http.NewRequest("OPTIONS", s.apiURL+"/api/v1/scans", nil)
		require.NoError(t, err)

		req.Header.Set("Origin", s.frontendURL)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")

		resp, err := s.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent)
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	})

	t.Run("Rate_Limiting", func(t *testing.T) {
		// Make rapid requests to test rate limiting
		rateLimited := false
		
		for i := 0; i < 50; i++ {
			resp, err := s.client.Get(s.apiURL + "/health")
			if err != nil {
				continue
			}
			
			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimited = true
				resp.Body.Close()
				break
			}
			resp.Body.Close()
			
			time.Sleep(100 * time.Millisecond)
		}

		// Rate limiting should eventually kick in
		t.Logf("Rate limiting triggered: %v", rateLimited)
	})

	t.Run("Input_Validation", func(t *testing.T) {
		// Test with malicious input
		maliciousData := map[string]interface{}{
			"name": "<script>alert('xss')</script>",
			"sql":  "'; DROP TABLE users; --",
		}

		resp := s.makeRequest(t, "POST", "/auth/register", maliciousData)
		defer resp.Body.Close()

		// Should return validation error, not 500
		assert.True(t, resp.StatusCode == http.StatusBadRequest || 
			resp.StatusCode == http.StatusUnprocessableEntity)
	})
}

// testPerformance tests performance requirements
func (s *ProductionSmokeTestSuite) testPerformance(t *testing.T) {
	t.Run("Response_Time", func(t *testing.T) {
		endpoints := []string{
			"/health",
			"/ready",
			"/metrics",
		}

		for _, endpoint := range endpoints {
			t.Run(endpoint, func(t *testing.T) {
				start := time.Now()
				resp, err := s.client.Get(s.apiURL + endpoint)
				duration := time.Since(start)

				require.NoError(t, err)
				resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Less(t, duration, 5*time.Second, 
					"Endpoint %s should respond within 5 seconds", endpoint)

				t.Logf("Endpoint %s response time: %v", endpoint, duration)
			})
		}
	})

	t.Run("Concurrent_Requests", func(t *testing.T) {
		concurrency := 10
		requests := 50
		
		start := time.Now()
		
		// Channel to collect results
		results := make(chan bool, requests)
		
		// Launch concurrent requests
		for i := 0; i < concurrency; i++ {
			go func() {
				for j := 0; j < requests/concurrency; j++ {
					resp, err := s.client.Get(s.apiURL + "/health")
					if err == nil && resp.StatusCode == http.StatusOK {
						results <- true
					} else {
						results <- false
					}
					if resp != nil {
						resp.Body.Close()
					}
				}
			}()
		}
		
		// Collect results
		successCount := 0
		for i := 0; i < requests; i++ {
			if <-results {
				successCount++
			}
		}
		
		duration := time.Since(start)
		successRate := float64(successCount) / float64(requests)
		
		assert.Greater(t, successRate, 0.95, "Success rate should be > 95%%")
		assert.Less(t, duration, 30*time.Second, "Concurrent requests should complete within 30 seconds")
		
		t.Logf("Concurrent test: %d/%d successful (%.2f%%) in %v", 
			successCount, requests, successRate*100, duration)
	})
}

// Helper methods

func (s *ProductionSmokeTestSuite) makeRequest(t *testing.T, method, path string, data interface{}) *http.Response {
	var body *bytes.Buffer
	
	if data != nil {
		jsonData, err := json.Marshal(data)
		require.NoError(t, err)
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, s.apiURL+path, body)
	require.NoError(t, err)

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	require.NoError(t, err)

	return resp
}

func (s *ProductionSmokeTestSuite) makeAuthenticatedRequest(t *testing.T, method, path string, data interface{}) *http.Response {
	var body *bytes.Buffer
	
	if data != nil {
		jsonData, err := json.Marshal(data)
		require.NoError(t, err)
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, s.apiURL+path, body)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+s.authToken)
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	require.NoError(t, err)

	return resp
}

func (s *ProductionSmokeTestSuite) createTestUser(t *testing.T, email, password string) {
	userData := map[string]interface{}{
		"email":    email,
		"password": password,
		"name":     "Smoke Test User",
	}

	resp := s.makeRequest(t, "POST", "/auth/register", userData)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		t.Logf("Failed to create test user: %d", resp.StatusCode)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}