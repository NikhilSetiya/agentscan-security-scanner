package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/agents/sast/semgrep"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/agentscan/agentscan/pkg/types"
)

// TestOrchestrationWorkflow_Integration tests the complete orchestration workflow
func TestOrchestrationWorkflow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	service, mockDB, mockQueue, agentManager := setupTestService(t)
	ctx := context.Background()

	// Register a real Semgrep agent for testing
	semgrepAgent := semgrep.NewAgent()
	err := agentManager.RegisterAgent("semgrep", semgrepAgent)
	require.NoError(t, err)

	// Setup test data
	repoID := uuid.New()
	userID := uuid.New()

	scanRequest := &ScanRequest{
		RepositoryID: repoID,
		UserID:       &userID,
		RepoURL:      "https://github.com/OWASP/NodeGoat.git", // Known vulnerable repo
		Branch:       "master",
		CommitSHA:    "abc123",
		ScanType:     types.ScanTypeFull,
		Priority:     5,
		Agents:       []string{"semgrep"},
		Timeout:      5 * time.Minute,
	}

	// Setup database mocks
	mockDB.On("CreateScanJob", ctx, mock.AnythingOfType("*types.ScanJob")).Return(nil)
	mockDB.On("GetScanJob", ctx, mock.AnythingOfType("uuid.UUID")).Return(&types.ScanJob{
		ID:              uuid.New(),
		RepositoryID:    repoID,
		UserID:          &userID,
		Branch:          "master",
		CommitSHA:       "abc123",
		ScanType:        types.ScanTypeFull,
		Status:          types.ScanJobStatusQueued,
		AgentsRequested: []string{"semgrep"},
		AgentsCompleted: []string{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil)
	mockDB.On("UpdateScanJob", ctx, mock.AnythingOfType("*types.ScanJob")).Return(nil)
	mockDB.On("CreateScanResult", ctx, mock.AnythingOfType("*types.ScanResult")).Return(nil)
	mockDB.On("CreateFinding", ctx, mock.AnythingOfType("*types.Finding")).Return(nil)

	// Setup queue mocks
	mockQueue.On("Enqueue", ctx, mock.AnythingOfType("*queue.Job")).Return(nil)

	// Submit scan
	scanJob, err := service.SubmitScan(ctx, scanRequest)
	require.NoError(t, err)
	assert.NotNil(t, scanJob)
	assert.Equal(t, types.ScanJobStatusQueued, scanJob.Status)

	// Verify mocks were called
	mockDB.AssertExpectations(t)
	mockQueue.AssertExpectations(t)

	t.Logf("Integration test completed successfully - scan job %s submitted", scanJob.ID)
}

// TestWorkerProcessing_Integration tests worker processing with real agents
func TestWorkerProcessing_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	_, _, _, agentManager := setupTestService(t)
	ctx := context.Background()

	// Create a real queue for testing
	// Note: In a real integration test, you'd use a real Redis instance
	// For this example, we'll use mocks but structure it like a real test

	// Register a mock agent that simulates successful scanning
	mockAgent := &MockAgent{}
	config := agent.AgentConfig{
		Name:           "test-agent",
		SupportedLangs: []string{"javascript"},
		Categories:     []agent.VulnCategory{agent.CategoryXSS},
	}

	scanResult := &agent.ScanResult{
		AgentID: "test-agent",
		Status:  agent.ScanStatusCompleted,
		Findings: []agent.Finding{
			{
				ID:          "test-finding-1",
				Tool:        "test-agent",
				RuleID:      "test-rule-1",
				Severity:    agent.SeverityHigh,
				Category:    agent.CategoryXSS,
				Title:       "Cross-site scripting vulnerability",
				Description: "Potential XSS vulnerability detected",
				File:        "app.js",
				Line:        42,
				Confidence:  0.9,
			},
		},
		Duration: 30 * time.Second,
	}

	mockAgent.On("GetConfig").Return(config)
	mockAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	mockAgent.On("Scan", ctx, mock.AnythingOfType("agent.ScanConfig")).Return(scanResult, nil)

	err := agentManager.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)

	// Create a test scan job
	scanJobID := uuid.New()
	scanJob := &types.ScanJob{
		ID:              scanJobID,
		RepositoryID:    uuid.New(),
		Branch:          "main",
		CommitSHA:       "abc123",
		ScanType:        types.ScanTypeFull,
		Status:          types.ScanJobStatusQueued,
		AgentsRequested: []string{"test-agent"},
		AgentsCompleted: []string{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Create a queue job
	queueJob := queue.NewJob("scan", queue.PriorityMedium, map[string]interface{}{
		"scan_job_id": scanJobID.String(),
		"repo_url":    "https://github.com/test/repo.git",
		"branch":      "main",
		"commit_sha":  "abc123",
		"scan_type":   types.ScanTypeFull,
		"agents":      []string{"test-agent"},
	})

	// Create a mock database for the worker
	mockDB := &MockRepository{}
	
	// Setup database mocks for worker processing
	mockDB.On("GetScanJob", ctx, scanJobID).Return(scanJob, nil)
	mockDB.On("UpdateScanJob", ctx, mock.AnythingOfType("*types.ScanJob")).Return(nil)
	mockDB.On("CreateScanResult", ctx, mock.AnythingOfType("*types.ScanResult")).Return(nil)
	mockDB.On("CreateFinding", ctx, mock.AnythingOfType("*types.Finding")).Return(nil)

	// Create a mock queue that implements the interface
	// For this test, we'll skip the queue operations since we're testing the worker directly
	worker := NewWorker("test-worker", nil, agentManager, mockDB)

	// Process the job
	worker.processJob(ctx, queueJob)

	// Verify that the agent was called and database operations occurred
	mockAgent.AssertExpectations(t)
	mockDB.AssertExpectations(t)

	// Verify worker stats
	stats := worker.GetStats()
	assert.Equal(t, "test-worker", stats.WorkerID)
	assert.Equal(t, int64(1), stats.JobsProcessed)
	assert.Equal(t, int64(0), stats.JobsFailed)
	assert.NotNil(t, stats.LastJobAt)

	t.Logf("Worker processing test completed successfully")
}

// TestAgentFailureHandling_Integration tests how the system handles agent failures
func TestAgentFailureHandling_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	_, _, _, agentManager := setupTestService(t)
	ctx := context.Background()

	// Register agents - one that succeeds and one that fails
	successAgent := &MockAgent{}
	failureAgent := &MockAgent{}

	successConfig := agent.AgentConfig{Name: "success-agent"}
	failureConfig := agent.AgentConfig{Name: "failure-agent"}

	successResult := &agent.ScanResult{
		AgentID:  "success-agent",
		Status:   agent.ScanStatusCompleted,
		Findings: []agent.Finding{},
		Duration: 10 * time.Second,
	}

	// Setup mocks
	successAgent.On("GetConfig").Return(successConfig)
	successAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	successAgent.On("Scan", ctx, mock.AnythingOfType("agent.ScanConfig")).Return(successResult, nil)

	failureAgent.On("GetConfig").Return(failureConfig)
	failureAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	failureAgent.On("Scan", ctx, mock.AnythingOfType("agent.ScanConfig")).Return(nil, assert.AnError)

	// Register agents
	err := agentManager.RegisterAgent("success-agent", successAgent)
	require.NoError(t, err)

	err = agentManager.RegisterAgent("failure-agent", failureAgent)
	require.NoError(t, err)

	// Execute parallel scans
	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
		Commit:  "abc123",
	}

	results, err := agentManager.ExecuteParallelScans(ctx, []string{"success-agent", "failure-agent"}, scanConfig)

	// Should get partial results even with one failure
	assert.Error(t, err) // Error because one agent failed
	assert.Equal(t, 1, len(results))
	assert.Contains(t, results, "success-agent")
	assert.NotContains(t, results, "failure-agent")

	// Verify the successful result
	assert.Equal(t, successResult, results["success-agent"])

	successAgent.AssertExpectations(t)
	failureAgent.AssertExpectations(t)

	t.Logf("Agent failure handling test completed successfully")
}

// TestConcurrentScans_Integration tests concurrent scan processing
func TestConcurrentScans_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	service, mockDB, mockQueue, agentManager := setupTestService(t)
	ctx := context.Background()

	// Register a mock agent
	mockAgent := &MockAgent{}
	config := agent.AgentConfig{Name: "test-agent"}
	mockAgent.On("GetConfig").Return(config)

	err := agentManager.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)

	// Setup mocks for multiple scans
	const numScans = 10

	for i := 0; i < numScans; i++ {
		mockDB.On("CreateScanJob", ctx, mock.AnythingOfType("*types.ScanJob")).Return(nil)
		mockQueue.On("Enqueue", ctx, mock.AnythingOfType("*queue.Job")).Return(nil)
	}

	// Submit multiple scans concurrently
	results := make(chan *types.ScanJob, numScans)
	errors := make(chan error, numScans)

	for i := 0; i < numScans; i++ {
		go func(index int) {
			scanRequest := &ScanRequest{
				RepositoryID: uuid.New(),
				RepoURL:      "https://github.com/test/repo.git",
				Branch:       "main",
				CommitSHA:    "abc123",
				ScanType:     types.ScanTypeFull,
				Priority:     5,
				Agents:       []string{"test-agent"},
			}

			scanJob, err := service.SubmitScan(ctx, scanRequest)
			if err != nil {
				errors <- err
			} else {
				results <- scanJob
			}
		}(i)
	}

	// Collect results
	var scanJobs []*types.ScanJob
	var scanErrors []error

	for i := 0; i < numScans; i++ {
		select {
		case job := <-results:
			scanJobs = append(scanJobs, job)
		case err := <-errors:
			scanErrors = append(scanErrors, err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for scan results")
		}
	}

	// Verify results
	assert.Equal(t, numScans, len(scanJobs))
	assert.Equal(t, 0, len(scanErrors))

	// Verify all scans have unique IDs
	seenIDs := make(map[uuid.UUID]bool)
	for _, job := range scanJobs {
		assert.False(t, seenIDs[job.ID], "Duplicate scan job ID found")
		seenIDs[job.ID] = true
		assert.Equal(t, types.ScanJobStatusQueued, job.Status)
	}

	mockDB.AssertExpectations(t)
	mockQueue.AssertExpectations(t)

	t.Logf("Concurrent scans test completed successfully - processed %d scans", numScans)
}

// TestHealthCheckMonitoring_Integration tests the health check monitoring system
func TestHealthCheckMonitoring_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	agentManager := NewAgentManager()
	ctx := context.Background()

	// Register agents with different health statuses
	healthyAgent := &MockAgent{}
	unhealthyAgent := &MockAgent{}

	healthyConfig := agent.AgentConfig{Name: "healthy-agent"}
	unhealthyConfig := agent.AgentConfig{Name: "unhealthy-agent"}

	healthyAgent.On("GetConfig").Return(healthyConfig)
	healthyAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)

	unhealthyAgent.On("GetConfig").Return(unhealthyConfig)
	unhealthyAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(assert.AnError)

	// Register agents
	err := agentManager.RegisterAgent("healthy-agent", healthyAgent)
	require.NoError(t, err)

	err = agentManager.RegisterAgent("unhealthy-agent", unhealthyAgent)
	require.NoError(t, err)

	// Perform health checks
	err = agentManager.HealthCheckAll(ctx)
	assert.Error(t, err) // Should return error because one agent is unhealthy

	// Verify health statuses
	healthyStatus, err := agentManager.GetAgentHealth("healthy-agent")
	require.NoError(t, err)
	assert.Equal(t, AgentStatusHealthy, healthyStatus.Status)
	assert.Equal(t, int64(1), healthyStatus.CheckCount)
	assert.Equal(t, int64(0), healthyStatus.FailureCount)

	unhealthyStatus, err := agentManager.GetAgentHealth("unhealthy-agent")
	require.NoError(t, err)
	assert.Equal(t, AgentStatusUnhealthy, unhealthyStatus.Status)
	assert.Equal(t, int64(1), unhealthyStatus.CheckCount)
	assert.Equal(t, int64(1), unhealthyStatus.FailureCount)
	assert.NotEmpty(t, unhealthyStatus.LastError)

	// Get overall stats
	stats := agentManager.GetStats()
	assert.Equal(t, 2, stats.TotalAgents)
	assert.Equal(t, 1, stats.HealthyAgents)
	assert.Equal(t, 1, stats.UnhealthyAgents)
	assert.Equal(t, 0, stats.UnknownAgents)

	healthyAgent.AssertExpectations(t)
	unhealthyAgent.AssertExpectations(t)

	t.Logf("Health check monitoring test completed successfully")
}

// TestServiceLifecycle_Integration tests starting and stopping the service
func TestServiceLifecycle_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	service, mockDB, mockQueue, agentManager := setupTestService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Register a mock agent
	mockAgent := &MockAgent{}
	config := agent.AgentConfig{Name: "test-agent"}
	mockAgent.On("GetConfig").Return(config)
	mockAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)

	err := agentManager.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)

	// Setup health check mocks
	mockDB.On("Health", ctx).Return(nil)
	mockQueue.On("GetStats", ctx).Return(&queue.JobStats{}, nil)
	mockQueue.On("Cleanup", ctx).Return(nil)

	// Test service health before starting
	err = service.Health(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start the service
	err = service.Start(ctx)
	require.NoError(t, err)

	// Wait a moment for workers to start
	time.Sleep(100 * time.Millisecond)

	// Test service health after starting
	err = service.Health(ctx)
	require.NoError(t, err)

	// Try to start again (should fail)
	err = service.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Stop the service
	err = service.Stop(ctx)
	require.NoError(t, err)

	// Test service health after stopping
	err = service.Health(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Try to stop again (should not fail)
	err = service.Stop(ctx)
	require.NoError(t, err)

	t.Logf("Service lifecycle test completed successfully")
}

// Benchmark tests for integration scenarios
func BenchmarkOrchestrationService_SubmitScan(b *testing.B) {
	service, mockDB, mockQueue, _ := setupTestService(&testing.T{})
	ctx := context.Background()

	// Setup mocks
	mockDB.On("CreateScanJob", ctx, mock.AnythingOfType("*types.ScanJob")).Return(nil)
	mockQueue.On("Enqueue", ctx, mock.AnythingOfType("*queue.Job")).Return(nil)

	scanRequest := &ScanRequest{
		RepositoryID: uuid.New(),
		RepoURL:      "https://github.com/test/repo.git",
		Branch:       "main",
		CommitSHA:    "abc123",
		ScanType:     types.ScanTypeFull,
		Priority:     5,
		Agents:       []string{"semgrep"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.SubmitScan(ctx, scanRequest)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAgentManager_ParallelScans(b *testing.B) {
	agentManager := NewAgentManager()
	ctx := context.Background()

	// Register multiple mock agents
	for i := 0; i < 5; i++ {
		mockAgent := &MockAgent{}
		config := agent.AgentConfig{Name: fmt.Sprintf("agent-%d", i)}
		result := &agent.ScanResult{
			AgentID:  fmt.Sprintf("agent-%d", i),
			Status:   agent.ScanStatusCompleted,
			Findings: []agent.Finding{},
		}

		mockAgent.On("GetConfig").Return(config)
		mockAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
		mockAgent.On("Scan", ctx, mock.AnythingOfType("agent.ScanConfig")).Return(result, nil)

		err := agentManager.RegisterAgent(fmt.Sprintf("agent-%d", i), mockAgent)
		if err != nil {
			b.Fatal(err)
		}
	}

	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}

	agentNames := []string{"agent-0", "agent-1", "agent-2", "agent-3", "agent-4"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agentManager.ExecuteParallelScans(ctx, agentNames, scanConfig)
		if err != nil {
			b.Fatal(err)
		}
	}
}