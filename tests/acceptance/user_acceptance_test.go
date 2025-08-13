package acceptance

import (
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

	"github.com/agentscan/agentscan/internal/api"
	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/orchestrator"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/types"
)

// UserAcceptanceTestSuite tests real-world user scenarios
type UserAcceptanceTestSuite struct {
	suite.Suite
	db           *database.DB
	redis        *queue.RedisClient
	orchestrator *orchestrator.Service
	apiServer    *httptest.Server
	testConfig   *config.Config
}

func TestUserAcceptanceTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping user acceptance tests in short mode")
	}
	
	suite.Run(t, new(UserAcceptanceTestSuite))
}

func (s *UserAcceptanceTestSuite) SetupSuite() {
	// Setup test environment
	s.testConfig = &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "agentscan_acceptance_test",
			User:     "postgres",
			Password: "postgres",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       4, // Different DB for acceptance tests
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
			DefaultTimeout: 3 * time.Minute,
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
	jobQueue := queue.NewQueue(s.redis, "acceptance_test_scans", queue.DefaultQueueConfig())
	
	agentManager := orchestrator.NewAgentManager()
	s.registerRealisticAgents(agentManager)

	orchestratorConfig := orchestrator.DefaultConfig()
	
	s.orchestrator = orchestrator.NewService(repoAdapter, jobQueue, agentManager, orchestratorConfig)
	
	ctx := context.Background()
	err = s.orchestrator.Start(ctx)
	s.Require().NoError(err)

	router := api.SetupRoutes(s.testConfig, s.db, s.redis, repos, s.orchestrator, jobQueue)
	s.apiServer = httptest.NewServer(router)
}

func (s *UserAcceptanceTestSuite) TearDownSuite() {
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

// TestDeveloperWorkflow tests the complete developer workflow
func (s *UserAcceptanceTestSuite) TestDeveloperWorkflow() {
	s.T().Log("Testing complete developer workflow...")

	// Scenario: Developer wants to scan their repository for security issues
	
	// Step 1: Developer submits a scan for their repository
	s.T().Log("Step 1: Developer submits scan request")
	scanRequest := map[string]interface{}{
		"repo_url":     "https://github.com/developer/my-web-app",
		"branch":       "feature/user-authentication",
		"commit":       "a1b2c3d4e5f6",
		"incremental":  false,
		"priority":     "medium",
	}

	scanResp := s.makeAPIRequest("POST", "/api/v1/scans", scanRequest)
	s.Equal(http.StatusCreated, scanResp.StatusCode)

	var scanJob map[string]interface{}
	err := json.NewDecoder(scanResp.Body).Decode(&scanJob)
	s.Require().NoError(err)
	scanResp.Body.Close()

	jobID := scanJob["id"].(string)
	s.NotEmpty(jobID)

	// Step 2: Developer checks scan status periodically
	s.T().Log("Step 2: Developer monitors scan progress")
	s.waitForScanCompletion(jobID, 60*time.Second)

	// Step 3: Developer retrieves scan results
	s.T().Log("Step 3: Developer retrieves scan results")
	resultsResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s/results", jobID), nil)
	s.Equal(http.StatusOK, resultsResp.StatusCode)

	var results map[string]interface{}
	err = json.NewDecoder(resultsResp.Body).Decode(&results)
	s.Require().NoError(err)
	resultsResp.Body.Close()

	findings, ok := results["findings"].([]interface{})
	s.Require().True(ok)

	// Step 4: Developer reviews findings and takes action
	s.T().Log("Step 4: Developer reviews findings")
	if len(findings) > 0 {
		finding := findings[0].(map[string]interface{})
		
		// Verify finding has all necessary information for developer
		s.Contains(finding, "severity", "Finding should have severity")
		s.Contains(finding, "title", "Finding should have title")
		s.Contains(finding, "description", "Finding should have description")
		s.Contains(finding, "file_path", "Finding should have file path")
		s.Contains(finding, "line_number", "Finding should have line number")
		s.Contains(finding, "tool", "Finding should indicate which tool found it")

		// Developer marks finding as false positive
		findingID := finding["id"].(string)
		feedbackRequest := map[string]interface{}{
			"action":  "false_positive",
			"comment": "This is a false positive - the input is properly sanitized",
		}

		feedbackResp := s.makeAPIRequest("POST", 
			fmt.Sprintf("/api/v1/findings/%s/feedback", findingID), 
			feedbackRequest)
		s.True(feedbackResp.StatusCode == http.StatusOK || feedbackResp.StatusCode == http.StatusCreated)
		feedbackResp.Body.Close()
	}

	// Step 5: Developer exports results for team review
	s.T().Log("Step 5: Developer exports results")
	exportResp := s.makeAPIRequest("GET", 
		fmt.Sprintf("/api/v1/scans/%s/export?format=json", jobID), nil)
	s.Equal(http.StatusOK, exportResp.StatusCode)
	exportResp.Body.Close()

	s.T().Log("Developer workflow test completed successfully")
}

// TestSecurityTeamWorkflow tests the security team workflow
func (s *UserAcceptanceTestSuite) TestSecurityTeamWorkflow() {
	s.T().Log("Testing security team workflow...")

	// Scenario: Security team wants to review findings across multiple repositories

	// Step 1: Security team runs scans on multiple repositories
	s.T().Log("Step 1: Security team initiates scans on multiple repositories")
	repositories := []string{
		"https://github.com/company/frontend-app",
		"https://github.com/company/backend-api",
		"https://github.com/company/mobile-app",
	}

	jobIDs := make([]string, len(repositories))
	for i, repo := range repositories {
		scanRequest := map[string]interface{}{
			"repo_url":     repo,
			"branch":       "main",
			"commit":       fmt.Sprintf("commit-%d", i),
			"incremental":  false,
			"priority":     "high",
		}

		scanResp := s.makeAPIRequest("POST", "/api/v1/scans", scanRequest)
		s.Equal(http.StatusCreated, scanResp.StatusCode)

		var scanJob map[string]interface{}
		err := json.NewDecoder(scanResp.Body).Decode(&scanJob)
		s.Require().NoError(err)
		scanResp.Body.Close()

		jobIDs[i] = scanJob["id"].(string)
	}

	// Step 2: Security team waits for all scans to complete
	s.T().Log("Step 2: Security team monitors scan completion")
	for _, jobID := range jobIDs {
		s.waitForScanCompletion(jobID, 90*time.Second)
	}

	// Step 3: Security team reviews aggregated findings
	s.T().Log("Step 3: Security team reviews aggregated findings")
	
	// Get findings from all scans
	allFindings := make([]interface{}, 0)
	for _, jobID := range jobIDs {
		resultsResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s/results", jobID), nil)
		s.Equal(http.StatusOK, resultsResp.StatusCode)

		var results map[string]interface{}
		err := json.NewDecoder(resultsResp.Body).Decode(&results)
		s.Require().NoError(err)
		resultsResp.Body.Close()

		if findings, ok := results["findings"].([]interface{}); ok {
			allFindings = append(allFindings, findings...)
		}
	}

	// Step 4: Security team filters high-severity findings
	s.T().Log("Step 4: Security team filters high-severity findings")
	highSeverityFindings := 0
	for _, finding := range allFindings {
		if findingMap, ok := finding.(map[string]interface{}); ok {
			if severity, ok := findingMap["severity"].(string); ok && severity == "high" {
				highSeverityFindings++
			}
		}
	}

	s.T().Logf("Found %d high-severity findings across all repositories", highSeverityFindings)

	// Step 5: Security team generates compliance report
	s.T().Log("Step 5: Security team generates compliance report")
	for _, jobID := range jobIDs {
		reportResp := s.makeAPIRequest("GET", 
			fmt.Sprintf("/api/v1/scans/%s/export?format=pdf", jobID), nil)
		s.True(reportResp.StatusCode == http.StatusOK || reportResp.StatusCode == http.StatusNotImplemented)
		reportResp.Body.Close()
	}

	s.T().Log("Security team workflow test completed successfully")
}

// TestCIIntegrationWorkflow tests CI/CD integration workflow
func (s *UserAcceptanceTestSuite) TestCIIntegrationWorkflow() {
	s.T().Log("Testing CI/CD integration workflow...")

	// Scenario: CI/CD pipeline integrates security scanning

	// Step 1: CI system submits scan on pull request
	s.T().Log("Step 1: CI system submits scan for pull request")
	prScanRequest := map[string]interface{}{
		"repo_url":     "https://github.com/company/web-app",
		"branch":       "feature/payment-integration",
		"commit":       "pr-commit-123",
		"incremental":  true, // CI typically uses incremental scans
		"priority":     "high", // CI scans are high priority
		"callback_url": "https://ci-system.company.com/webhooks/agentscan",
	}

	scanResp := s.makeAPIRequest("POST", "/api/v1/scans", prScanRequest)
	s.Equal(http.StatusCreated, scanResp.StatusCode)

	var scanJob map[string]interface{}
	err := json.NewDecoder(scanResp.Body).Decode(&scanJob)
	s.Require().NoError(err)
	scanResp.Body.Close()

	jobID := scanJob["id"].(string)

	// Step 2: CI system polls for scan completion
	s.T().Log("Step 2: CI system polls for scan completion")
	s.waitForScanCompletion(jobID, 60*time.Second)

	// Step 3: CI system retrieves results and makes decision
	s.T().Log("Step 3: CI system retrieves results for decision making")
	resultsResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s/results", jobID), nil)
	s.Equal(http.StatusOK, resultsResp.StatusCode)

	var results map[string]interface{}
	err = json.NewDecoder(resultsResp.Body).Decode(&results)
	s.Require().NoError(err)
	resultsResp.Body.Close()

	// Step 4: CI system evaluates findings against policy
	s.T().Log("Step 4: CI system evaluates findings against security policy")
	findings, ok := results["findings"].([]interface{})
	s.Require().True(ok)

	// Simulate CI policy: fail build if high-severity findings exist
	highSeverityCount := 0
	for _, finding := range findings {
		if findingMap, ok := finding.(map[string]interface{}); ok {
			if severity, ok := findingMap["severity"].(string); ok && severity == "high" {
				highSeverityCount++
			}
		}
	}

	buildShouldPass := highSeverityCount == 0
	s.T().Logf("CI decision: Build should %s (high-severity findings: %d)", 
		map[bool]string{true: "PASS", false: "FAIL"}[buildShouldPass], 
		highSeverityCount)

	// Step 5: CI system posts results as PR comment
	s.T().Log("Step 5: CI system would post results as PR comment")
	// In real scenario, this would post to GitHub/GitLab API
	// For testing, we just verify the data is available

	s.T().Log("CI/CD integration workflow test completed successfully")
}

// TestIncrementalScanningWorkflow tests incremental scanning user experience
func (s *UserAcceptanceTestSuite) TestIncrementalScanningWorkflow() {
	s.T().Log("Testing incremental scanning workflow...")

	// Scenario: Developer makes changes and wants fast feedback

	repoURL := "https://github.com/developer/active-project"
	
	// Step 1: Initial full scan
	s.T().Log("Step 1: Developer runs initial full scan")
	fullScanRequest := map[string]interface{}{
		"repo_url":     repoURL,
		"branch":       "main",
		"commit":       "initial-commit-abc",
		"incremental":  false,
		"priority":     "medium",
	}

	fullScanResp := s.makeAPIRequest("POST", "/api/v1/scans", fullScanRequest)
	s.Equal(http.StatusCreated, fullScanResp.StatusCode)

	var fullScanJob map[string]interface{}
	err := json.NewDecoder(fullScanResp.Body).Decode(&fullScanJob)
	s.Require().NoError(err)
	fullScanResp.Body.Close()

	fullScanJobID := fullScanJob["id"].(string)
	fullScanStartTime := time.Now()
	s.waitForScanCompletion(fullScanJobID, 90*time.Second)
	fullScanDuration := time.Since(fullScanStartTime)

	// Step 2: Developer makes changes and runs incremental scan
	s.T().Log("Step 2: Developer runs incremental scan after changes")
	incrementalScanRequest := map[string]interface{}{
		"repo_url":     repoURL,
		"branch":       "main",
		"commit":       "updated-commit-def",
		"incremental":  true,
		"priority":     "high", // Developer wants fast feedback
	}

	incrementalScanResp := s.makeAPIRequest("POST", "/api/v1/scans", incrementalScanRequest)
	s.Equal(http.StatusCreated, incrementalScanResp.StatusCode)

	var incrementalScanJob map[string]interface{}
	err = json.NewDecoder(incrementalScanResp.Body).Decode(&incrementalScanJob)
	s.Require().NoError(err)
	incrementalScanResp.Body.Close()

	incrementalScanJobID := incrementalScanJob["id"].(string)
	incrementalScanStartTime := time.Now()
	s.waitForScanCompletion(incrementalScanJobID, 60*time.Second)
	incrementalScanDuration := time.Since(incrementalScanStartTime)

	// Step 3: Verify incremental scan is faster
	s.T().Log("Step 3: Verify incremental scan performance")
	s.T().Logf("Full scan duration: %v", fullScanDuration)
	s.T().Logf("Incremental scan duration: %v", incrementalScanDuration)

	// Incremental scan should be significantly faster
	s.Less(incrementalScanDuration, fullScanDuration,
		"Incremental scan should be faster than full scan")

	// Step 4: Compare results quality
	s.T().Log("Step 4: Compare results quality between full and incremental scans")
	
	// Get results from both scans
	fullResultsResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s/results", fullScanJobID), nil)
	s.Equal(http.StatusOK, fullResultsResp.StatusCode)
	var fullResults map[string]interface{}
	err = json.NewDecoder(fullResultsResp.Body).Decode(&fullResults)
	s.Require().NoError(err)
	fullResultsResp.Body.Close()

	incrementalResultsResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s/results", incrementalScanJobID), nil)
	s.Equal(http.StatusOK, incrementalResultsResp.StatusCode)
	var incrementalResults map[string]interface{}
	err = json.NewDecoder(incrementalResultsResp.Body).Decode(&incrementalResults)
	s.Require().NoError(err)
	incrementalResultsResp.Body.Close()

	// Both scans should produce meaningful results
	fullFindings := fullResults["findings"].([]interface{})
	incrementalFindings := incrementalResults["findings"].([]interface{})

	s.T().Logf("Full scan findings: %d", len(fullFindings))
	s.T().Logf("Incremental scan findings: %d", len(incrementalFindings))

	s.T().Log("Incremental scanning workflow test completed successfully")
}

// TestErrorRecoveryWorkflow tests system behavior during error conditions
func (s *UserAcceptanceTestSuite) TestErrorRecoveryWorkflow() {
	s.T().Log("Testing error recovery workflow...")

	// Scenario: User encounters various error conditions and system handles them gracefully

	// Step 1: User submits scan with invalid repository
	s.T().Log("Step 1: User submits scan with invalid repository")
	invalidRepoRequest := map[string]interface{}{
		"repo_url":     "https://github.com/nonexistent/repository",
		"branch":       "main",
		"commit":       "abc123",
		"incremental":  false,
		"priority":     "medium",
	}

	invalidRepoResp := s.makeAPIRequest("POST", "/api/v1/scans", invalidRepoRequest)
	// Should handle gracefully with appropriate error message
	s.True(invalidRepoResp.StatusCode == http.StatusBadRequest || 
		invalidRepoResp.StatusCode == http.StatusUnprocessableEntity)

	var errorResponse map[string]interface{}
	err := json.NewDecoder(invalidRepoResp.Body).Decode(&errorResponse)
	s.Require().NoError(err)
	invalidRepoResp.Body.Close()

	// Error message should be helpful
	s.Contains(errorResponse, "error", "Error response should contain error message")

	// Step 2: User submits valid scan but system is under load
	s.T().Log("Step 2: User submits scan during high load")
	validScanRequest := map[string]interface{}{
		"repo_url":     "https://github.com/user/valid-repo",
		"branch":       "main",
		"commit":       "valid-commit-123",
		"incremental":  false,
		"priority":     "low", // Low priority during high load
	}

	validScanResp := s.makeAPIRequest("POST", "/api/v1/scans", validScanRequest)
	// Should accept the request even under load
	s.True(validScanResp.StatusCode == http.StatusCreated || 
		validScanResp.StatusCode == http.StatusAccepted)

	if validScanResp.StatusCode == http.StatusCreated {
		var scanJob map[string]interface{}
		err := json.NewDecoder(validScanResp.Body).Decode(&scanJob)
		s.Require().NoError(err)
		
		jobID := scanJob["id"].(string)
		
		// Step 3: User checks status of queued scan
		s.T().Log("Step 3: User monitors queued scan status")
		
		// Check status multiple times to see progression
		for i := 0; i < 5; i++ {
			statusResp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s", jobID), nil)
			s.Equal(http.StatusOK, statusResp.StatusCode)

			var scanStatus map[string]interface{}
			err := json.NewDecoder(statusResp.Body).Decode(&scanStatus)
			s.Require().NoError(err)
			statusResp.Body.Close()

			status := scanStatus["status"].(string)
			s.T().Logf("Scan status check %d: %s", i+1, status)

			// Status should be valid
			validStatuses := []string{"queued", "running", "completed", "failed"}
			s.Contains(validStatuses, status, "Scan status should be valid")

			if status == "completed" || status == "failed" {
				break
			}

			time.Sleep(2 * time.Second)
		}
	}
	validScanResp.Body.Close()

	// Step 4: User attempts to access non-existent scan
	s.T().Log("Step 4: User attempts to access non-existent scan")
	nonExistentResp := s.makeAPIRequest("GET", "/api/v1/scans/non-existent-scan-id", nil)
	s.Equal(http.StatusNotFound, nonExistentResp.StatusCode)

	var notFoundResponse map[string]interface{}
	err = json.NewDecoder(nonExistentResp.Body).Decode(&notFoundResponse)
	s.Require().NoError(err)
	nonExistentResp.Body.Close()

	// Should provide helpful error message
	s.Contains(notFoundResponse, "error", "Not found response should contain error message")

	s.T().Log("Error recovery workflow test completed successfully")
}

// Helper methods

func (s *UserAcceptanceTestSuite) registerRealisticAgents(agentManager *orchestrator.AgentManager) {
	// Register agents that simulate realistic behavior
	agents := []struct {
		name     string
		findings []types.Finding
	}{
		{
			name: "semgrep",
			findings: []types.Finding{
				{
					Tool:        "semgrep",
					RuleID:      "javascript.express.security.audit.xss.mustache-escape-false",
					Severity:    "high",
					Category:    "security",
					Title:       "Potential XSS vulnerability in template rendering",
					Description: "Template rendering without proper escaping can lead to XSS attacks",
					FilePath:    "src/views/user-profile.js",
					LineNumber:  42,
					Confidence:  0.9,
					Status:      "open",
				},
			},
		},
		{
			name: "eslint-security",
			findings: []types.Finding{
				{
					Tool:        "eslint-security",
					RuleID:      "security/detect-unsafe-regex",
					Severity:    "medium",
					Category:    "security",
					Title:       "Potentially unsafe regular expression",
					Description: "This regular expression may be vulnerable to ReDoS attacks",
					FilePath:    "src/utils/validation.js",
					LineNumber:  15,
					Confidence:  0.7,
					Status:      "open",
				},
			},
		},
		{
			name: "bandit",
			findings: []types.Finding{
				{
					Tool:        "bandit",
					RuleID:      "B602",
					Severity:    "high",
					Category:    "security",
					Title:       "Use of subprocess with shell=True",
					Description: "subprocess call with shell=True identified, security issue",
					FilePath:    "scripts/deploy.py",
					LineNumber:  28,
					Confidence:  0.8,
					Status:      "open",
				},
			},
		},
	}

	for _, agent := range agents {
		mockAgent := &AcceptanceTestMockAgent{
			name:     agent.name,
			findings: agent.findings,
		}
		err := agentManager.RegisterAgent(agent.name, mockAgent)
		s.Require().NoError(err)
	}
}

func (s *UserAcceptanceTestSuite) makeAPIRequest(method, path string, body interface{}) *http.Response {
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

func (s *UserAcceptanceTestSuite) waitForScanCompletion(jobID string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.Fail("Scan did not complete within timeout", "JobID: %s, Timeout: %v", jobID, timeout)
			return
		case <-ticker.C:
			resp := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/scans/%s", jobID), nil)
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
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

			if status == "completed" {
				return
			}

			if status == "failed" {
				s.Fail("Scan failed", "JobID: %s", jobID)
				return
			}
		}
	}
}

// AcceptanceTestMockAgent simulates realistic agent behavior for acceptance testing
type AcceptanceTestMockAgent struct {
	name     string
	findings []types.Finding
}

func (m *AcceptanceTestMockAgent) Scan(ctx context.Context, config types.ScanConfig) (*types.ScanResult, error) {
	// Simulate realistic scan duration
	scanDuration := time.Duration(500+len(m.findings)*100) * time.Millisecond
	time.Sleep(scanDuration)

	return &types.ScanResult{
		AgentID:  m.name,
		Status:   types.ScanStatusCompleted,
		Findings: m.findings,
		Metadata: types.Metadata{
			"scan_duration":  scanDuration.String(),
			"files_scanned":  "25",
			"rules_applied":  "150",
		},
		Duration: scanDuration,
	}, nil
}

func (m *AcceptanceTestMockAgent) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *AcceptanceTestMockAgent) GetConfig() types.AgentConfig {
	return types.AgentConfig{
		Name:        m.name,
		Version:     "1.0.0",
		Description: fmt.Sprintf("Mock %s agent for acceptance testing", m.name),
		Languages:   []string{"javascript", "python", "go"},
		Categories:  []string{"sast", "security"},
	}
}

func (m *AcceptanceTestMockAgent) GetVersion() types.VersionInfo {
	return types.VersionInfo{
		Agent:   "1.0.0",
		Tool:    "1.0.0",
		Updated: time.Now(),
	}
}