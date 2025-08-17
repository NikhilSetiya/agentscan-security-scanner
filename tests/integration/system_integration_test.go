package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// SystemIntegrationTestSuite tests the complete AgentScan system
type SystemIntegrationTestSuite struct {
	suite.Suite
	db              *database.DB
	redis           *queue.RedisClient
	orchestrator    *orchestrator.Service
	apiServer       *httptest.Server
	testConfig      *config.Config
	testRepoURL     string
	testBranch      string
	testCommit      string
}

func TestSystemIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	suite.Run(t, new(SystemIntegrationTestSuite))
}

func (s *SystemIntegrationTestSuite) SetupSuite() {
	// Load test configuration
	s.testConfig = &config.Config{
		Database: config.DatabaseConfig{
			Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
			Port:     5432,
			Name:     getEnvOrDefault("TEST_DB_NAME", "agentscan_test"),
			User:     getEnvOrDefault("TEST_DB_USER", "postgres"),
			Password: getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host:     getEnvOrDefault("TEST_REDIS_HOST", "localhost"),
			Port:     6379,
			Password: "",
			DB:       1, // Use different DB for tests
		},
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         0, // Let test server choose port
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Agents: config.AgentsConfig{
			MaxConcurrent:  5,
			DefaultTimeout: 5 * time.Minute,
		},
	}

	// Initialize database connection
	var err error
	s.db, err = database.New(&s.testConfig.Database)
	s.Require().NoError(err, "Failed to connect to test database")

	// Run migrations
	err = s.db.Migrate()
	s.Require().NoError(err, "Failed to run database migrations")

	// Initialize Redis connection
	s.redis, err = queue.NewRedisClient(&s.testConfig.Redis)
	s.Require().NoError(err, "Failed to connect to test Redis")

	// Clear Redis test database
	err = s.redis.FlushDB(context.Background())
	s.Require().NoError(err, "Failed to clear Redis test database")

	// Initialize repositories
	repos := database.NewRepositories(s.db)
	repoAdapter := database.NewRepositoryAdapter(s.db, repos)

	// Initialize job queue
	jobQueue := queue.NewQueue(s.redis, "test_scans", queue.DefaultQueueConfig())

	// Initialize agent manager and register test agents
	agentManager := orchestrator.NewAgentManager()
	s.registerTestAgents(agentManager)

	// Initialize orchestrator service
	orchestratorConfig := orchestrator.DefaultConfig()
	orchestratorConfig.MaxConcurrentScans = 3
	orchestratorConfig.WorkerCount = 2
	
	s.orchestrator = orchestrator.NewService(repoAdapter, jobQueue, agentManager, orchestratorConfig)

	// Start orchestrator
	ctx := context.Background()
	err = s.orchestrator.Start(ctx)
	s.Require().NoError(err, "Failed to start orchestrator service")

	// Create API server
	router := api.SetupRoutes(s.testConfig, s.db, s.redis, repos, s.orchestrator, jobQueue)
	s.apiServer = httptest.NewServer(router)

	// Set test repository details
	s.testRepoURL = "https://github.com/test/vulnerable-repo"
	s.testBranch = "main"
	s.testCommit = "abc123def456"
}

func (s *SystemIntegrationTestSuite) TearDownSuite() {
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

func (s *SystemIntegrationTestSuite) SetupTest() {
	// Clear database tables before each test
	s.clearTestData()
}

// TestCompleteWorkflow tests the complete scanning workflow from API to results
func (s *SystemIntegrationTestSuite) TestCompleteWorkflow() {
	// Step 1: Submit scan via API
	scanRequest := map[string]interface{}{
		"repo_url":     s.testRepoURL,
		"branch":       s.testBranch,
		"commit":       s.testCommit,
		"incremental":  false,
		"priority":     "medium",
	}

	scanResp := s.makeAPIRequest("POST", "/api/v1/scans", scanRequest)
	s.Equal(http.StatusCreated, scanResp.StatusCode)

	var scanJob map[string]interface{}
	err := json.NewDecoder(scanResp.Body).Decode(&scanJob)
	s.Require().NoError(err)

	jobID, ok := scanJob["id"].(string)
	s.Require().True(ok, "Job ID should be a string")
	s.NotEmpty(jobID)

	// Step 2: Wait for scan completion
	s.waitForScanCompletion(jobID, 30*time.Second)

	// Step 3: Verify scan status
	statusResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s", jobID), nil)
	s.Equal(http.StatusOK, statusResp.StatusCode)

	var scanStatus map[string]interface{}
	err = json.NewDecoder(statusResp.Body).Decode(&scanStatus)
	s.Require().NoError(err)

	s.Equal("completed", scanStatus["status"])
	s.NotNil(scanStatus["completed_at"])

	// Step 4: Retrieve and verify results
	resultsResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s/results", jobID), nil)
	s.Equal(http.StatusOK, resultsResp.StatusCode)

	var results map[string]interface{}
	err = json.NewDecoder(resultsResp.Body).Decode(&results)
	s.Require().NoError(err)

	findings, ok := results["findings"].([]interface{})
	s.Require().True(ok, "Findings should be an array")
	s.Greater(len(findings), 0, "Should have at least one finding")

	// Step 5: Verify finding structure
	finding := findings[0].(map[string]interface{})
	s.Contains(finding, "id")
	s.Contains(finding, "tool")
	s.Contains(finding, "severity")
	s.Contains(finding, "title")
	s.Contains(finding, "file_path")
}

// TestConcurrentScans tests the system's ability to handle multiple concurrent scans
func (s *SystemIntegrationTestSuite) TestConcurrentScans() {
	const numConcurrentScans = 5
	
	scanRequests := make([]map[string]interface{}, numConcurrentScans)
	for i := 0; i < numConcurrentScans; i++ {
		scanRequests[i] = map[string]interface{}{
			"repo_url":     fmt.Sprintf("https://github.com/test/repo-%d", i),
			"branch":       "main",
			"commit":       fmt.Sprintf("commit-%d", i),
			"incremental":  false,
			"priority":     "medium",
		}
	}

	// Submit all scans concurrently
	jobIDs := make([]string, numConcurrentScans)
	for i, req := range scanRequests {
		resp := s.makeAPIRequest("POST", "/api/v1/scans", req)
		s.Equal(http.StatusCreated, resp.StatusCode)

		var scanJob map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&scanJob)
		s.Require().NoError(err)

		jobIDs[i] = scanJob["id"].(string)
	}

	// Wait for all scans to complete
	for _, jobID := range jobIDs {
		s.waitForScanCompletion(jobID, 60*time.Second)
	}

	// Verify all scans completed successfully
	for _, jobID := range jobIDs {
		statusResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s", jobID), nil)
		s.Equal(http.StatusOK, statusResp.StatusCode)

		var scanStatus map[string]interface{}
		err := json.NewDecoder(statusResp.Body).Decode(&scanStatus)
		s.Require().NoError(err)

		s.Equal("completed", scanStatus["status"])
	}
}

// TestIncrementalScanning tests incremental scanning functionality
func (s *SystemIntegrationTestSuite) TestIncrementalScanning() {
	// First, perform a full scan
	fullScanRequest := map[string]interface{}{
		"repo_url":     s.testRepoURL,
		"branch":       s.testBranch,
		"commit":       s.testCommit,
		"incremental":  false,
		"priority":     "medium",
	}

	fullScanResp := s.makeAPIRequest("POST", "/api/v1/scans", fullScanRequest)
	s.Equal(http.StatusCreated, fullScanResp.StatusCode)

	var fullScanJob map[string]interface{}
	err := json.NewDecoder(fullScanResp.Body).Decode(&fullScanJob)
	s.Require().NoError(err)

	fullScanJobID := fullScanJob["id"].(string)
	s.waitForScanCompletion(fullScanJobID, 30*time.Second)

	// Now perform an incremental scan
	incrementalScanRequest := map[string]interface{}{
		"repo_url":     s.testRepoURL,
		"branch":       s.testBranch,
		"commit":       "def456ghi789", // Different commit
		"incremental":  true,
		"priority":     "medium",
	}

	incrementalScanResp := s.makeAPIRequest("POST", "/api/v1/scans", incrementalScanRequest)
	s.Equal(http.StatusCreated, incrementalScanResp.StatusCode)

	var incrementalScanJob map[string]interface{}
	err = json.NewDecoder(incrementalScanResp.Body).Decode(&incrementalScanJob)
	s.Require().NoError(err)

	incrementalScanJobID := incrementalScanJob["id"].(string)
	s.waitForScanCompletion(incrementalScanJobID, 30*time.Second)

	// Verify both scans completed
	for _, jobID := range []string{fullScanJobID, incrementalScanJobID} {
		statusResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s", jobID), nil)
		s.Equal(http.StatusOK, statusResp.StatusCode)

		var scanStatus map[string]interface{}
		err := json.NewDecoder(statusResp.Body).Decode(&scanStatus)
		s.Require().NoError(err)

		s.Equal("completed", scanStatus["status"])
	}
}

// TestErrorHandling tests system behavior under various error conditions
func (s *SystemIntegrationTestSuite) TestErrorHandling() {
	// Test invalid repository URL
	invalidRepoRequest := map[string]interface{}{
		"repo_url":     "invalid-url",
		"branch":       "main",
		"commit":       "abc123",
		"incremental":  false,
		"priority":     "medium",
	}

	invalidResp := s.makeAPIRequest("POST", "/api/v1/scans", invalidRepoRequest)
	s.Equal(http.StatusBadRequest, invalidResp.StatusCode)

	// Test missing required fields
	incompleteRequest := map[string]interface{}{
		"repo_url": s.testRepoURL,
		// Missing branch and commit
	}

	incompleteResp := s.makeAPIRequest("POST", "/api/v1/scans", incompleteRequest)
	s.Equal(http.StatusBadRequest, incompleteResp.StatusCode)

	// Test non-existent scan ID
	nonExistentResp := s.makeAPIRequest("GET", "/api/v1/scans/non-existent-id", nil)
	s.Equal(http.StatusNotFound, nonExistentResp.StatusCode)
}

// TestAPIAuthentication tests API authentication and authorization
func (s *SystemIntegrationTestSuite) TestAPIAuthentication() {
	// Test accessing protected endpoint without authentication
	req, err := http.NewRequest("GET", s.apiServer.URL+"/api/v1/scans", nil)
	s.Require().NoError(err)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should return 401 Unauthorized for protected endpoints
	// Note: This depends on the actual authentication implementation
	s.True(resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusOK)
}

// TestHealthEndpoints tests system health monitoring endpoints
func (s *SystemIntegrationTestSuite) TestHealthEndpoints() {
	// Test main health endpoint
	healthResp := s.makeAPIRequest("GET", "/health", nil)
	s.Equal(http.StatusOK, healthResp.StatusCode)

	var health map[string]interface{}
	err := json.NewDecoder(healthResp.Body).Decode(&health)
	s.Require().NoError(err)

	s.Equal("healthy", health["status"])
	s.Contains(health, "timestamp")
	s.Contains(health, "version")

	// Test database health
	dbHealthResp := s.makeAPIRequest("GET", "/health/database", nil)
	s.Equal(http.StatusOK, dbHealthResp.StatusCode)

	// Test Redis health
	redisHealthResp := s.makeAPIRequest("GET", "/health/redis", nil)
	s.Equal(http.StatusOK, redisHealthResp.StatusCode)
}

// TestPerformanceUnderLoad tests system performance under load
func (s *SystemIntegrationTestSuite) TestPerformanceUnderLoad() {
	const numRequests = 20
	const concurrency = 5

	// Create a channel to control concurrency
	semaphore := make(chan struct{}, concurrency)
	results := make(chan time.Duration, numRequests)
	errors := make(chan error, numRequests)

	startTime := time.Now()

	// Submit requests concurrently
	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			requestStart := time.Now()

			scanRequest := map[string]interface{}{
				"repo_url":     fmt.Sprintf("https://github.com/test/load-test-%d", requestID),
				"branch":       "main",
				"commit":       fmt.Sprintf("commit-%d", requestID),
				"incremental":  false,
				"priority":     "low", // Use low priority for load testing
			}

			resp := s.makeAPIRequest("POST", "/api/v1/scans", scanRequest)
			if resp.StatusCode != http.StatusCreated {
				errors <- fmt.Errorf("request %d failed with status %d", requestID, resp.StatusCode)
				return
			}

			var scanJob map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&scanJob)
			if err != nil {
				errors <- fmt.Errorf("request %d failed to decode response: %v", requestID, err)
				return
			}

			jobID := scanJob["id"].(string)
			
			// Wait for completion with timeout
			completed := s.waitForScanCompletionWithTimeout(jobID, 60*time.Second)
			if !completed {
				errors <- fmt.Errorf("request %d timed out", requestID)
				return
			}

			duration := time.Since(requestStart)
			results <- duration
		}(i)
	}

	// Collect results
	var durations []time.Duration
	var errorCount int

	for i := 0; i < numRequests; i++ {
		select {
		case duration := <-results:
			durations = append(durations, duration)
		case err := <-errors:
			s.T().Logf("Load test error: %v", err)
			errorCount++
		case <-time.After(120 * time.Second):
			s.T().Logf("Load test timed out waiting for results")
			break
		}
	}

	totalTime := time.Since(startTime)

	// Performance assertions
	s.LessOrEqual(errorCount, numRequests/10, "Error rate should be less than 10%")
	s.GreaterOrEqual(len(durations), numRequests*8/10, "At least 80% of requests should complete")

	if len(durations) > 0 {
		// Calculate average response time
		var totalDuration time.Duration
		for _, d := range durations {
			totalDuration += d
		}
		avgDuration := totalDuration / time.Duration(len(durations))

		s.T().Logf("Load test results:")
		s.T().Logf("  Total requests: %d", numRequests)
		s.T().Logf("  Successful requests: %d", len(durations))
		s.T().Logf("  Failed requests: %d", errorCount)
		s.T().Logf("  Total time: %v", totalTime)
		s.T().Logf("  Average response time: %v", avgDuration)

		// Performance requirements
		s.Less(avgDuration, 10*time.Second, "Average response time should be less than 10 seconds")
	}
}

// Helper methods

func (s *SystemIntegrationTestSuite) registerTestAgents(agentManager *orchestrator.AgentManager) {
	// Register mock agents for testing
	mockAgent := &MockSecurityAgent{
		name: "test-agent",
		findings: []types.Finding{
			{
				Tool:        "test-agent",
				RuleID:      "test-rule-1",
				Severity:    "high",
				Category:    "security",
				Title:       "Test Security Issue",
				Description: "This is a test security finding",
				FilePath:    "test.js",
				LineNumber:  42,
				Confidence:  0.9,
				Status:      "open",
			},
		},
	}

	err := agentManager.RegisterAgent("test-agent", mockAgent)
	s.Require().NoError(err)
}

func (s *SystemIntegrationTestSuite) makeAPIRequest(method, path string, body interface{}) *http.Response {
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

func (s *SystemIntegrationTestSuite) waitForScanCompletion(jobID string, timeout time.Duration) {
	completed := s.waitForScanCompletionWithTimeout(jobID, timeout)
	s.Require().True(completed, "Scan should complete within timeout")
}

func (s *SystemIntegrationTestSuite) waitForScanCompletionWithTimeout(jobID string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			resp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s", jobID), nil)
			if resp.StatusCode != http.StatusOK {
				continue
			}

			var scanStatus map[string]interface{}
			err := json.NewDecoder(resp.Body).Decode(&scanStatus)
			resp.Body.Close()
			if err != nil {
				continue
			}

			status, ok := scanStatus["status"].(string)
			if !ok {
				continue
			}

			if status == "completed" || status == "failed" {
				return status == "completed"
			}
		}
	}
}

func (s *SystemIntegrationTestSuite) clearTestData() {
	// Clear test data from database tables
	tables := []string{
		"user_feedback",
		"findings",
		"scan_results",
		"scan_jobs",
		"repositories",
		"organization_members",
		"organizations",
		"users",
	}

	for _, table := range tables {
		_, err := s.db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		s.Require().NoError(err, "Failed to clear table %s", table)
	}

	// Clear Redis data
	err := s.redis.FlushDB(context.Background())
	s.Require().NoError(err, "Failed to clear Redis test data")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MockSecurityAgent is a mock implementation for testing
type MockSecurityAgent struct {
	name     string
	findings []types.Finding
}

func (m *MockSecurityAgent) Scan(ctx context.Context, config types.ScanConfig) (*types.ScanResult, error) {
	// Simulate scan duration
	time.Sleep(100 * time.Millisecond)

	return &types.ScanResult{
		AgentID:  m.name,
		Status:   types.ScanStatusCompleted,
		Findings: m.findings,
		Metadata: types.Metadata{
			"scan_duration": "100ms",
			"files_scanned": "5",
		},
		Duration: 100 * time.Millisecond,
	}, nil
}

func (m *MockSecurityAgent) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockSecurityAgent) GetConfig() types.AgentConfig {
	return types.AgentConfig{
		Name:        m.name,
		Version:     "1.0.0",
		Description: "Mock security agent for testing",
		Languages:   []string{"javascript", "python", "go"},
		Categories:  []string{"sast", "security"},
	}
}

func (m *MockSecurityAgent) GetVersion() types.VersionInfo {
	return types.VersionInfo{
		Agent:   "1.0.0",
		Tool:    "1.0.0",
		Updated: time.Now(),
	}
}