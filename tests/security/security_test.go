package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/api"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
)

// SecurityTestSuite performs security testing on the AgentScan system
type SecurityTestSuite struct {
	suite.Suite
	db           *database.DB
	redis        *queue.RedisClient
	orchestrator *orchestrator.Service
	apiServer    *httptest.Server
	testConfig   *config.Config
}

func TestSecurityTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}
	
	suite.Run(t, new(SecurityTestSuite))
}

func (s *SecurityTestSuite) SetupSuite() {
	// Setup test environment
	s.testConfig = &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "agentscan_security_test",
			User:     "postgres",
			Password: "postgres",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       3, // Different DB for security tests
		},
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         0,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Agents: config.AgentsConfig{
			MaxConcurrent:  5,
			DefaultTimeout: 2 * time.Minute,
		},
	}

	// Initialize components
	var err error
	s.db, err = database.New(&s.testConfig.Database)
	s.Require().NoError(err)

	s.redis, err = queue.NewRedisClient(&s.testConfig.Redis)
	s.Require().NoError(err)

	// Setup orchestrator and API server
	repos := database.NewRepositories(s.db)
	repoAdapter := database.NewRepositoryAdapter(s.db, repos)
	jobQueue := queue.NewQueue(s.redis, "security_test_scans", queue.DefaultQueueConfig())
	
	agentManager := orchestrator.NewAgentManager()
	orchestratorConfig := orchestrator.DefaultConfig()
	
	s.orchestrator = orchestrator.NewService(repoAdapter, jobQueue, agentManager, orchestratorConfig)
	
	ctx := context.Background()
	err = s.orchestrator.Start(ctx)
	s.Require().NoError(err)

	router := api.SetupRoutes(s.testConfig, s.db, s.redis, repos, s.orchestrator, jobQueue)
	s.apiServer = httptest.NewServer(router)
}

func (s *SecurityTestSuite) TearDownSuite() {
	if s.apiServer != nil {
		s.apiServer.Close()
	}

	if s.orchestrator != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.orchestrator.Stop(ctx)
	}

	if s.redis != nil {
		s.redis.FlushDB(context.Background())
		s.redis.Close()
	}

	if s.db != nil {
		s.db.Close()
	}
}

// TestSQLInjectionProtection tests protection against SQL injection attacks
func (s *SecurityTestSuite) TestSQLInjectionProtection() {
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"' UNION SELECT * FROM users --",
		"'; INSERT INTO users (email) VALUES ('hacker@evil.com'); --",
		"' OR 1=1 --",
		"admin'--",
		"admin'/*",
		"' OR 'x'='x",
		"'; EXEC xp_cmdshell('dir'); --",
	}

	for _, payload := range sqlInjectionPayloads {
		s.T().Run(fmt.Sprintf("SQLInjection_%s", payload), func(t *testing.T) {
			// Test SQL injection in various endpoints
			endpoints := []struct {
				method string
				path   string
				body   map[string]interface{}
			}{
				{
					method: "POST",
					path:   "/api/v1/scans",
					body: map[string]interface{}{
						"repo_url": payload,
						"branch":   "main",
						"commit":   "abc123",
					},
				},
				{
					method: "GET",
					path:   fmt.Sprintf("/api/v1/scans?repo_url=%s", payload),
					body:   nil,
				},
			}

			for _, endpoint := range endpoints {
				resp := s.makeAPIRequest(endpoint.method, endpoint.path, endpoint.body)
				
				// Should not return 500 (internal server error) which might indicate SQL injection
				assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
					"SQL injection payload '%s' caused internal server error", payload)
				
				// Should return appropriate error status (400 or 422)
				assert.True(t, resp.StatusCode == http.StatusBadRequest || 
					resp.StatusCode == http.StatusUnprocessableEntity ||
					resp.StatusCode == http.StatusNotFound,
					"SQL injection payload '%s' should be rejected with appropriate error", payload)
				
				resp.Body.Close()
			}
		})
	}
}

// TestXSSProtection tests protection against Cross-Site Scripting attacks
func (s *SecurityTestSuite) TestXSSProtection() {
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<svg onload=alert('XSS')>",
		"<iframe src=javascript:alert('XSS')></iframe>",
		"<body onload=alert('XSS')>",
		"<input onfocus=alert('XSS') autofocus>",
		"<select onfocus=alert('XSS') autofocus>",
		"<textarea onfocus=alert('XSS') autofocus>",
		"<keygen onfocus=alert('XSS') autofocus>",
	}

	for _, payload := range xssPayloads {
		s.T().Run(fmt.Sprintf("XSS_%s", payload), func(t *testing.T) {
			// Test XSS in scan submission
			scanRequest := map[string]interface{}{
				"repo_url": fmt.Sprintf("https://github.com/test/%s", payload),
				"branch":   payload,
				"commit":   "abc123",
			}

			resp := s.makeAPIRequest("POST", "/api/v1/scans", scanRequest)
			
			// Should handle XSS payload appropriately
			assert.True(t, resp.StatusCode == http.StatusBadRequest || 
				resp.StatusCode == http.StatusUnprocessableEntity,
				"XSS payload '%s' should be rejected", payload)

			// Check response doesn't contain unescaped payload
			var responseBody bytes.Buffer
			responseBody.ReadFrom(resp.Body)
			responseContent := responseBody.String()
			
			assert.NotContains(t, responseContent, "<script>",
				"Response should not contain unescaped script tags")
			assert.NotContains(t, responseContent, "javascript:",
				"Response should not contain javascript: protocol")
			
			resp.Body.Close()
		})
	}
}

// TestCSRFProtection tests Cross-Site Request Forgery protection
func (s *SecurityTestSuite) TestCSRFProtection() {
	// Test that state-changing operations require proper CSRF protection
	stateChangingEndpoints := []struct {
		method string
		path   string
		body   map[string]interface{}
	}{
		{
			method: "POST",
			path:   "/api/v1/scans",
			body: map[string]interface{}{
				"repo_url": "https://github.com/test/repo",
				"branch":   "main",
				"commit":   "abc123",
			},
		},
		{
			method: "DELETE",
			path:   "/api/v1/scans/test-scan-id",
			body:   nil,
		},
	}

	for _, endpoint := range stateChangingEndpoints {
		s.T().Run(fmt.Sprintf("CSRF_%s_%s", endpoint.method, endpoint.path), func(t *testing.T) {
			// Make request without proper CSRF token/headers
			req := s.createRawRequest(endpoint.method, endpoint.path, endpoint.body)
			
			// Remove any CSRF-related headers that might be automatically added
			req.Header.Del("X-CSRF-Token")
			req.Header.Del("X-Requested-With")
			
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should require proper CSRF protection for state-changing operations
			// Note: This depends on the actual CSRF implementation
			// For now, we just ensure the endpoint doesn't crash
			assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
				"CSRF test should not cause internal server error")
		})
	}
}

// TestAuthenticationBypass tests for authentication bypass vulnerabilities
func (s *SecurityTestSuite) TestAuthenticationBypass() {
	authBypassPayloads := []map[string]string{
		{"Authorization": "Bearer invalid-token"},
		{"Authorization": "Bearer "},
		{"Authorization": "Basic invalid"},
		{"Authorization": ""},
		{"X-API-Key": "invalid-key"},
		{"X-User-ID": "admin"},
		{"X-Role": "admin"},
	}

	protectedEndpoints := []string{
		"/api/v1/scans",
		"/api/v1/users/profile",
		"/api/v1/organizations",
	}

	for _, endpoint := range protectedEndpoints {
		for i, headers := range authBypassPayloads {
			s.T().Run(fmt.Sprintf("AuthBypass_%s_%d", endpoint, i), func(t *testing.T) {
				req, err := http.NewRequest("GET", s.apiServer.URL+endpoint, nil)
				require.NoError(t, err)

				// Add potentially malicious headers
				for key, value := range headers {
					req.Header.Set(key, value)
				}

				client := &http.Client{Timeout: 30 * time.Second}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				// Should not grant unauthorized access
				assert.True(t, resp.StatusCode == http.StatusUnauthorized || 
					resp.StatusCode == http.StatusForbidden ||
					resp.StatusCode == http.StatusNotFound,
					"Authentication bypass attempt should be rejected")
			})
		}
	}
}

// TestInputValidation tests comprehensive input validation
func (s *SecurityTestSuite) TestInputValidation() {
	invalidInputs := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "ExtremelyLongString",
			input: map[string]interface{}{
				"repo_url": strings.Repeat("a", 10000),
				"branch":   "main",
				"commit":   "abc123",
			},
		},
		{
			name: "NullBytes",
			input: map[string]interface{}{
				"repo_url": "https://github.com/test/repo\x00",
				"branch":   "main\x00",
				"commit":   "abc123",
			},
		},
		{
			name: "UnicodeExploits",
			input: map[string]interface{}{
				"repo_url": "https://github.com/test/repo\u202e",
				"branch":   "main\u200b",
				"commit":   "abc123",
			},
		},
		{
			name: "InvalidJSON",
			input: map[string]interface{}{
				"repo_url": map[string]interface{}{"nested": "object"},
				"branch":   []string{"array", "value"},
				"commit":   123,
			},
		},
		{
			name: "EmptyValues",
			input: map[string]interface{}{
				"repo_url": "",
				"branch":   "",
				"commit":   "",
			},
		},
	}

	for _, testCase := range invalidInputs {
		s.T().Run(testCase.name, func(t *testing.T) {
			resp := s.makeAPIRequest("POST", "/api/v1/scans", testCase.input)
			defer resp.Body.Close()

			// Should reject invalid input with appropriate error
			assert.True(t, resp.StatusCode == http.StatusBadRequest || 
				resp.StatusCode == http.StatusUnprocessableEntity,
				"Invalid input should be rejected with appropriate error")

			// Should not cause internal server error
			assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
				"Invalid input should not cause internal server error")
		})
	}
}

// TestRateLimiting tests rate limiting protection
func (s *SecurityTestSuite) TestRateLimiting() {
	const (
		numRequests = 100
		timeWindow  = 1 * time.Minute
	)

	// Make rapid requests to test rate limiting
	var successCount, rateLimitedCount int

	for i := 0; i < numRequests; i++ {
		scanRequest := map[string]interface{}{
			"repo_url": fmt.Sprintf("https://github.com/test/rate-limit-test-%d", i),
			"branch":   "main",
			"commit":   fmt.Sprintf("commit-%d", i),
		}

		resp := s.makeAPIRequest("POST", "/api/v1/scans", scanRequest)
		
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		} else if resp.StatusCode == http.StatusCreated {
			successCount++
		}
		
		resp.Body.Close()

		// Small delay to avoid overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}

	s.T().Logf("Rate limiting test results: %d successful, %d rate limited", successCount, rateLimitedCount)

	// Should have some rate limiting in effect for rapid requests
	// Note: This depends on the actual rate limiting implementation
	assert.True(t, rateLimitedCount > 0 || successCount < numRequests,
		"Rate limiting should be in effect for rapid requests")
}

// TestSecurityHeaders tests that proper security headers are set
func (s *SecurityTestSuite) TestSecurityHeaders() {
	resp := s.makeAPIRequest("GET", "/health", nil)
	defer resp.Body.Close()

	// Check for important security headers
	securityHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Strict-Transport-Security": "", // Should be present
		"Content-Security-Policy":   "", // Should be present
	}

	for header, expectedValue := range securityHeaders {
		headerValue := resp.Header.Get(header)
		if expectedValue == "" {
			assert.NotEmpty(t, headerValue, "Security header %s should be present", header)
		} else {
			assert.Equal(t, expectedValue, headerValue, "Security header %s should have correct value", header)
		}
	}

	// Check that sensitive headers are not exposed
	sensitiveHeaders := []string{
		"Server",
		"X-Powered-By",
		"X-AspNet-Version",
		"X-AspNetMvc-Version",
	}

	for _, header := range sensitiveHeaders {
		headerValue := resp.Header.Get(header)
		assert.Empty(t, headerValue, "Sensitive header %s should not be exposed", header)
	}
}

// TestDataEncryption tests that sensitive data is properly encrypted
func (s *SecurityTestSuite) TestDataEncryption() {
	// This test would verify that sensitive data is encrypted at rest
	// For now, we'll test that the encryption service is working
	
	// Test would include:
	// 1. Storing sensitive data (API keys, tokens, etc.)
	// 2. Verifying it's encrypted in the database
	// 3. Verifying it can be decrypted correctly
	
	s.T().Log("Data encryption test - would verify sensitive data encryption")
	// Implementation would depend on the actual encryption service
}

// TestAuditLogging tests that security events are properly logged
func (s *SecurityTestSuite) TestAuditLogging() {
	// Test that security-relevant events are logged
	securityEvents := []struct {
		action   string
		endpoint string
		method   string
		body     map[string]interface{}
	}{
		{
			action:   "failed_authentication",
			endpoint: "/api/v1/scans",
			method:   "POST",
			body: map[string]interface{}{
				"repo_url": "https://github.com/test/repo",
				"branch":   "main",
				"commit":   "abc123",
			},
		},
		{
			action:   "invalid_input",
			endpoint: "/api/v1/scans",
			method:   "POST",
			body: map[string]interface{}{
				"repo_url": "'; DROP TABLE users; --",
				"branch":   "main",
				"commit":   "abc123",
			},
		},
	}

	for _, event := range securityEvents {
		s.T().Run(event.action, func(t *testing.T) {
			// Make request that should trigger audit logging
			resp := s.makeAPIRequest(event.method, event.endpoint, event.body)
			resp.Body.Close()

			// In a real implementation, we would:
			// 1. Check that the event was logged to the audit system
			// 2. Verify the log contains appropriate details
			// 3. Ensure sensitive data is not logged in plain text

			s.T().Logf("Audit logging test for %s - would verify event is logged", event.action)
		})
	}
}

// TestPrivilegeEscalation tests for privilege escalation vulnerabilities
func (s *SecurityTestSuite) TestPrivilegeEscalation() {
	privilegeEscalationAttempts := []struct {
		name    string
		headers map[string]string
		body    map[string]interface{}
	}{
		{
			name: "RoleManipulation",
			headers: map[string]string{
				"X-User-Role": "admin",
			},
			body: map[string]interface{}{
				"repo_url": "https://github.com/test/repo",
				"branch":   "main",
				"commit":   "abc123",
			},
		},
		{
			name: "UserIDManipulation",
			headers: map[string]string{
				"X-User-ID": "1",
			},
			body: map[string]interface{}{
				"user_id": "admin",
				"role":    "admin",
			},
		},
	}

	for _, attempt := range privilegeEscalationAttempts {
		s.T().Run(attempt.name, func(t *testing.T) {
			req := s.createRawRequest("POST", "/api/v1/scans", attempt.body)
			
			// Add potentially malicious headers
			for key, value := range attempt.headers {
				req.Header.Set(key, value)
			}

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should not grant elevated privileges
			assert.True(t, resp.StatusCode == http.StatusUnauthorized || 
				resp.StatusCode == http.StatusForbidden ||
				resp.StatusCode == http.StatusBadRequest,
				"Privilege escalation attempt should be rejected")
		})
	}
}

// Helper methods

func (s *SecurityTestSuite) makeAPIRequest(method, path string, body interface{}) *http.Response {
	var reqBody strings.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		s.Require().NoError(err)
		reqBody = *strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequest(method, s.apiServer.URL+path, &reqBody)
	s.Require().NoError(err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)

	return resp
}

func (s *SecurityTestSuite) createRawRequest(method, path string, body interface{}) *http.Request {
	var reqBody strings.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		s.Require().NoError(err)
		reqBody = *strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequest(method, s.apiServer.URL+path, &reqBody)
	s.Require().NoError(err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}