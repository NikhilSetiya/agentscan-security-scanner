package testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/shared/logging"
)

// APIIntegrationTest provides comprehensive API integration testing
type APIIntegrationTest struct {
	server     *httptest.Server
	client     *http.Client
	config     *config.ProductionConfig
	logger     logging.Logger
	baseURL    string
	authToken  string
}

// APITestCase represents a single API test case
type APITestCase struct {
	Name           string
	Method         string
	Path           string
	Headers        map[string]string
	Body           interface{}
	ExpectedStatus int
	ExpectedBody   interface{}
	Validator      func(*http.Response, []byte) error
	Setup          func() error
	Cleanup        func() error
}

// APITestSuite represents a collection of related API tests
type APITestSuite struct {
	Name        string
	Description string
	Setup       func() error
	Cleanup     func() error
	Tests       []APITestCase
}

// NewAPIIntegrationTest creates a new API integration test instance
func NewAPIIntegrationTest(router *gin.Engine, config *config.ProductionConfig) *APIIntegrationTest {
	server := httptest.NewServer(router)
	
	return &APIIntegrationTest{
		server: server,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config:  config,
		logger:  logging.GetLogger(),
		baseURL: server.URL,
	}
}

// Close closes the test server
func (ait *APIIntegrationTest) Close() {
	if ait.server != nil {
		ait.server.Close()
	}
}

// RunAllAPITests runs all API integration tests
func (ait *APIIntegrationTest) RunAllAPITests(t *testing.T) {
	ait.logger.Info("Starting comprehensive API integration test suite")

	testSuites := []APITestSuite{
		ait.createHealthCheckTests(),
		ait.createAuthenticationTests(),
		ait.createRateLimitingTests(),
		ait.createSecurityTests(),
		ait.createCRUDTests(),
		ait.createErrorHandlingTests(),
		ait.createPerformanceTests(),
	}

	for _, suite := range testSuites {
		t.Run(suite.Name, func(t *testing.T) {
			ait.runTestSuite(t, suite)
		})
	}

	ait.logger.Info("API integration test suite completed successfully")
}

// runTestSuite runs a single test suite
func (ait *APIIntegrationTest) runTestSuite(t *testing.T, suite APITestSuite) {
	ait.logger.Info("Running API test suite", "suite", suite.Name)

	// Run suite setup
	if suite.Setup != nil {
		require.NoError(t, suite.Setup(), "Suite setup should succeed")
	}

	// Run cleanup at the end
	if suite.Cleanup != nil {
		defer func() {
			if err := suite.Cleanup(); err != nil {
				ait.logger.Error("Suite cleanup failed", "suite", suite.Name, "error", err)
			}
		}()
	}

	// Run individual tests
	for _, testCase := range suite.Tests {
		t.Run(testCase.Name, func(t *testing.T) {
			ait.runAPITest(t, testCase)
		})
	}
}

// runAPITest runs a single API test case
func (ait *APIIntegrationTest) runAPITest(t *testing.T, testCase APITestCase) {
	start := time.Now()
	
	ait.logger.Info("Running API test", "test", testCase.Name)

	// Run test setup
	if testCase.Setup != nil {
		require.NoError(t, testCase.Setup(), "Test setup should succeed")
	}

	// Run cleanup at the end
	if testCase.Cleanup != nil {
		defer func() {
			if err := testCase.Cleanup(); err != nil {
				ait.logger.Error("Test cleanup failed", "test", testCase.Name, "error", err)
			}
		}()
	}

	// Prepare request
	var body io.Reader
	if testCase.Body != nil {
		jsonBody, err := json.Marshal(testCase.Body)
		require.NoError(t, err, "Request body should marshal successfully")
		body = bytes.NewReader(jsonBody)
	}

	url := ait.baseURL + testCase.Path
	req, err := http.NewRequest(testCase.Method, url, body)
	require.NoError(t, err, "Request creation should succeed")

	// Set headers
	if testCase.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	for key, value := range testCase.Headers {
		req.Header.Set(key, value)
	}

	// Add auth token if available
	if ait.authToken != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+ait.authToken)
	}

	// Make request
	resp, err := ait.client.Do(req)
	require.NoError(t, err, "HTTP request should succeed")
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Response body should be readable")

	duration := time.Since(start)

	// Log request/response details
	ait.logger.Info("API test completed",
		"test", testCase.Name,
		"method", testCase.Method,
		"path", testCase.Path,
		"status", resp.StatusCode,
		"duration", duration,
		"response_size", len(respBody),
	)

	// Validate status code
	assert.Equal(t, testCase.ExpectedStatus, resp.StatusCode, 
		"Status code should match expected. Response: %s", string(respBody))

	// Validate expected body if provided
	if testCase.ExpectedBody != nil {
		var actualBody interface{}
		err := json.Unmarshal(respBody, &actualBody)
		require.NoError(t, err, "Response body should be valid JSON")
		
		expectedJSON, _ := json.Marshal(testCase.ExpectedBody)
		actualJSON, _ := json.Marshal(actualBody)
		
		assert.JSONEq(t, string(expectedJSON), string(actualJSON), 
			"Response body should match expected")
	}

	// Run custom validator if provided
	if testCase.Validator != nil {
		require.NoError(t, testCase.Validator(resp, respBody), 
			"Custom validation should pass")
	}

	// Performance assertion
	assert.Less(t, duration, 5*time.Second, 
		"API response should be within 5 seconds")
}

// createHealthCheckTests creates health check API tests
func (ait *APIIntegrationTest) createHealthCheckTests() APITestSuite {
	return APITestSuite{
		Name:        "HealthCheck",
		Description: "Tests for health check endpoints",
		Tests: []APITestCase{
			{
				Name:           "BasicHealthCheck",
				Method:         "GET",
				Path:           "/health",
				ExpectedStatus: http.StatusOK,
				Validator: func(resp *http.Response, body []byte) error {
					var healthResp map[string]interface{}
					if err := json.Unmarshal(body, &healthResp); err != nil {
						return err
					}
					
					status, ok := healthResp["status"].(string)
					if !ok || (status != "healthy" && status != "degraded") {
						return fmt.Errorf("invalid health status: %v", status)
					}
					
					return nil
				},
			},
			{
				Name:           "ReadinessCheck",
				Method:         "GET",
				Path:           "/ready",
				ExpectedStatus: http.StatusOK,
				ExpectedBody: map[string]interface{}{
					"status": "ready",
				},
			},
			{
				Name:           "LivenessCheck",
				Method:         "GET",
				Path:           "/live",
				ExpectedStatus: http.StatusOK,
				ExpectedBody: map[string]interface{}{
					"status": "alive",
				},
			},
		},
	}
}

// createAuthenticationTests creates authentication API tests
func (ait *APIIntegrationTest) createAuthenticationTests() APITestSuite {
	return APITestSuite{
		Name:        "Authentication",
		Description: "Tests for authentication endpoints",
		Tests: []APITestCase{
			{
				Name:   "LoginWithValidCredentials",
				Method: "POST",
				Path:   "/api/v1/auth/login",
				Body: map[string]interface{}{
					"email":    "test@example.com",
					"password": "testpassword123",
				},
				ExpectedStatus: http.StatusOK,
				Validator: func(resp *http.Response, body []byte) error {
					var loginResp map[string]interface{}
					if err := json.Unmarshal(body, &loginResp); err != nil {
						return err
					}
					
					token, ok := loginResp["token"].(string)
					if !ok || token == "" {
						return fmt.Errorf("missing or empty token")
					}
					
					// Store token for subsequent tests
					ait.authToken = token
					
					return nil
				},
			},
			{
				Name:   "LoginWithInvalidCredentials",
				Method: "POST",
				Path:   "/api/v1/auth/login",
				Body: map[string]interface{}{
					"email":    "test@example.com",
					"password": "wrongpassword",
				},
				ExpectedStatus: http.StatusUnauthorized,
			},
			{
				Name:   "LoginWithMissingFields",
				Method: "POST",
				Path:   "/api/v1/auth/login",
				Body: map[string]interface{}{
					"email": "test@example.com",
				},
				ExpectedStatus: http.StatusBadRequest,
			},
			{
				Name:   "RegisterNewUser",
				Method: "POST",
				Path:   "/api/v1/auth/register",
				Body: map[string]interface{}{
					"email":    "newuser@example.com",
					"password": "newpassword123",
					"name":     "New User",
				},
				ExpectedStatus: http.StatusCreated,
			},
			{
				Name:   "RegisterExistingUser",
				Method: "POST",
				Path:   "/api/v1/auth/register",
				Body: map[string]interface{}{
					"email":    "test@example.com",
					"password": "testpassword123",
					"name":     "Test User",
				},
				ExpectedStatus: http.StatusConflict,
			},
			{
				Name:           "GetCurrentUser",
				Method:         "GET",
				Path:           "/api/v1/auth/me",
				ExpectedStatus: http.StatusOK,
				Validator: func(resp *http.Response, body []byte) error {
					var userResp map[string]interface{}
					if err := json.Unmarshal(body, &userResp); err != nil {
						return err
					}
					
					user, ok := userResp["user"].(map[string]interface{})
					if !ok {
						return fmt.Errorf("missing user object")
					}
					
					email, ok := user["email"].(string)
					if !ok || email == "" {
						return fmt.Errorf("missing or empty email")
					}
					
					return nil
				},
			},
			{
				Name:   "GetCurrentUserWithoutAuth",
				Method: "GET",
				Path:   "/api/v1/auth/me",
				Headers: map[string]string{
					"Authorization": "", // Override auth token
				},
				ExpectedStatus: http.StatusUnauthorized,
			},
		},
	}
}

// createRateLimitingTests creates rate limiting API tests
func (ait *APIIntegrationTest) createRateLimitingTests() APITestSuite {
	return APITestSuite{
		Name:        "RateLimiting",
		Description: "Tests for rate limiting functionality",
		Tests: []APITestCase{
			{
				Name:           "RateLimitExceeded",
				Method:         "POST",
				Path:           "/api/v1/auth/login",
				ExpectedStatus: http.StatusTooManyRequests,
				Setup: func() error {
					// Make multiple requests to exceed rate limit
					for i := 0; i < 10; i++ {
						body := map[string]interface{}{
							"email":    "test@example.com",
							"password": "wrongpassword",
						}
						jsonBody, _ := json.Marshal(body)
						
						req, _ := http.NewRequest("POST", ait.baseURL+"/api/v1/auth/login", 
							bytes.NewReader(jsonBody))
						req.Header.Set("Content-Type", "application/json")
						
						ait.client.Do(req)
					}
					return nil
				},
			},
		},
	}
}

// createSecurityTests creates security API tests
func (ait *APIIntegrationTest) createSecurityTests() APITestSuite {
	return APITestSuite{
		Name:        "Security",
		Description: "Tests for security headers and policies",
		Tests: []APITestCase{
			{
				Name:           "SecurityHeaders",
				Method:         "GET",
				Path:           "/health",
				ExpectedStatus: http.StatusOK,
				Validator: func(resp *http.Response, body []byte) error {
					// Check for security headers
					requiredHeaders := []string{
						"X-Frame-Options",
						"X-Content-Type-Options",
						"X-XSS-Protection",
						"Referrer-Policy",
					}
					
					for _, header := range requiredHeaders {
						if resp.Header.Get(header) == "" {
							return fmt.Errorf("missing security header: %s", header)
						}
					}
					
					return nil
				},
			},
			{
				Name:   "CORSHeaders",
				Method: "OPTIONS",
				Path:   "/api/v1/auth/login",
				Headers: map[string]string{
					"Origin":                        "https://app.agentscan.io",
					"Access-Control-Request-Method": "POST",
				},
				ExpectedStatus: http.StatusNoContent,
				Validator: func(resp *http.Response, body []byte) error {
					if resp.Header.Get("Access-Control-Allow-Origin") == "" {
						return fmt.Errorf("missing CORS allow origin header")
					}
					return nil
				},
			},
			{
				Name:   "InvalidCORSOrigin",
				Method: "OPTIONS",
				Path:   "/api/v1/auth/login",
				Headers: map[string]string{
					"Origin":                        "https://malicious.com",
					"Access-Control-Request-Method": "POST",
				},
				ExpectedStatus: http.StatusNoContent,
				Validator: func(resp *http.Response, body []byte) error {
					if resp.Header.Get("Access-Control-Allow-Origin") != "" {
						return fmt.Errorf("CORS should not allow malicious origin")
					}
					return nil
				},
			},
		},
	}
}

// createCRUDTests creates CRUD operation API tests
func (ait *APIIntegrationTest) createCRUDTests() APITestSuite {
	var createdResourceID string
	
	return APITestSuite{
		Name:        "CRUD",
		Description: "Tests for CRUD operations",
		Tests: []APITestCase{
			{
				Name:   "CreateResource",
				Method: "POST",
				Path:   "/api/v1/repositories",
				Body: map[string]interface{}{
					"name": "test-repo",
					"url":  "https://github.com/test/repo.git",
				},
				ExpectedStatus: http.StatusCreated,
				Validator: func(resp *http.Response, body []byte) error {
					var createResp map[string]interface{}
					if err := json.Unmarshal(body, &createResp); err != nil {
						return err
					}
					
					repo, ok := createResp["repository"].(map[string]interface{})
					if !ok {
						return fmt.Errorf("missing repository object")
					}
					
					id, ok := repo["id"].(string)
					if !ok || id == "" {
						return fmt.Errorf("missing or empty repository ID")
					}
					
					createdResourceID = id
					return nil
				},
			},
			{
				Name:           "GetResource",
				Method:         "GET",
				Path:           "/api/v1/repositories/" + createdResourceID,
				ExpectedStatus: http.StatusOK,
				Setup: func() error {
					if createdResourceID == "" {
						return fmt.Errorf("no resource ID available")
					}
					return nil
				},
			},
			{
				Name:   "UpdateResource",
				Method: "PUT",
				Path:   "/api/v1/repositories/" + createdResourceID,
				Body: map[string]interface{}{
					"name": "updated-test-repo",
					"url":  "https://github.com/test/updated-repo.git",
				},
				ExpectedStatus: http.StatusOK,
				Setup: func() error {
					if createdResourceID == "" {
						return fmt.Errorf("no resource ID available")
					}
					return nil
				},
			},
			{
				Name:           "DeleteResource",
				Method:         "DELETE",
				Path:           "/api/v1/repositories/" + createdResourceID,
				ExpectedStatus: http.StatusNoContent,
				Setup: func() error {
					if createdResourceID == "" {
						return fmt.Errorf("no resource ID available")
					}
					return nil
				},
			},
			{
				Name:           "GetDeletedResource",
				Method:         "GET",
				Path:           "/api/v1/repositories/" + createdResourceID,
				ExpectedStatus: http.StatusNotFound,
			},
		},
	}
}

// createErrorHandlingTests creates error handling API tests
func (ait *APIIntegrationTest) createErrorHandlingTests() APITestSuite {
	return APITestSuite{
		Name:        "ErrorHandling",
		Description: "Tests for error handling and validation",
		Tests: []APITestCase{
			{
				Name:           "NotFoundEndpoint",
				Method:         "GET",
				Path:           "/api/v1/nonexistent",
				ExpectedStatus: http.StatusNotFound,
			},
			{
				Name:           "MethodNotAllowed",
				Method:         "PATCH",
				Path:           "/health",
				ExpectedStatus: http.StatusMethodNotAllowed,
			},
			{
				Name:   "InvalidJSON",
				Method: "POST",
				Path:   "/api/v1/auth/login",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body:           "invalid json",
				ExpectedStatus: http.StatusBadRequest,
			},
			{
				Name:   "ValidationError",
				Method: "POST",
				Path:   "/api/v1/repositories",
				Body: map[string]interface{}{
					"name": "", // Invalid empty name
					"url":  "invalid-url",
				},
				ExpectedStatus: http.StatusBadRequest,
				Validator: func(resp *http.Response, body []byte) error {
					var errorResp map[string]interface{}
					if err := json.Unmarshal(body, &errorResp); err != nil {
						return err
					}
					
					// Check for validation error structure
					if errorResp["error"] == nil {
						return fmt.Errorf("missing error field in response")
					}
					
					return nil
				},
			},
		},
	}
}

// createPerformanceTests creates performance API tests
func (ait *APIIntegrationTest) createPerformanceTests() APITestSuite {
	return APITestSuite{
		Name:        "Performance",
		Description: "Tests for API performance and concurrency",
		Tests: []APITestCase{
			{
				Name:           "ConcurrentRequests",
				Method:         "GET",
				Path:           "/health",
				ExpectedStatus: http.StatusOK,
				Setup: func() error {
					return ait.testConcurrentRequests()
				},
			},
			{
				Name:           "LargePayload",
				Method:         "POST",
				Path:           "/api/v1/repositories",
				ExpectedStatus: http.StatusCreated,
				Setup: func() error {
					// Create a large but valid payload
					largeDescription := strings.Repeat("A", 1000)
					return ait.testLargePayload(largeDescription)
				},
			},
		},
	}
}

// testConcurrentRequests tests concurrent API requests
func (ait *APIIntegrationTest) testConcurrentRequests() error {
	concurrency := 50
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)
	
	start := time.Now()
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			req, err := http.NewRequest("GET", ait.baseURL+"/health", nil)
			if err != nil {
				errors <- fmt.Errorf("worker %d: request creation failed: %w", id, err)
				return
			}
			
			resp, err := ait.client.Do(req)
			if err != nil {
				errors <- fmt.Errorf("worker %d: request failed: %w", id, err)
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("worker %d: unexpected status %d", id, resp.StatusCode)
				return
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	duration := time.Since(start)
	
	// Check for errors
	var errorCount int
	for err := range errors {
		ait.logger.Error("Concurrent request error", "error", err)
		errorCount++
	}
	
	ait.logger.Info("Concurrent requests test completed",
		"concurrency", concurrency,
		"duration", duration,
		"errors", errorCount,
		"requests_per_second", float64(concurrency)/duration.Seconds(),
	)
	
	if errorCount > concurrency/10 { // Allow up to 10% errors
		return fmt.Errorf("too many concurrent request errors: %d/%d", errorCount, concurrency)
	}
	
	return nil
}

// testLargePayload tests handling of large payloads
func (ait *APIIntegrationTest) testLargePayload(description string) error {
	payload := map[string]interface{}{
		"name":        "large-payload-test",
		"url":         "https://github.com/test/large-repo.git",
		"description": description,
	}
	
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal large payload: %w", err)
	}
	
	req, err := http.NewRequest("POST", ait.baseURL+"/api/v1/repositories", 
		bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if ait.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+ait.authToken)
	}
	
	start := time.Now()
	resp, err := ait.client.Do(req)
	duration := time.Since(start)
	
	if err != nil {
		return fmt.Errorf("large payload request failed: %w", err)
	}
	defer resp.Body.Close()
	
	ait.logger.Info("Large payload test completed",
		"payload_size", len(jsonBody),
		"duration", duration,
		"status", resp.StatusCode,
	)
	
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status for large payload: %d", resp.StatusCode)
	}
	
	return nil
}