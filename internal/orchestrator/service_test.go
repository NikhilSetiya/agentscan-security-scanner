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

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/agentscan/agentscan/pkg/types"
)

// MockRepository is a mock implementation of the database interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) CreateScanJob(ctx context.Context, job *types.ScanJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockRepository) GetScanJob(ctx context.Context, id uuid.UUID) (*types.ScanJob, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.ScanJob), args.Error(1)
}

func (m *MockRepository) UpdateScanJob(ctx context.Context, job *types.ScanJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockRepository) DeleteScanJob(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ListScanJobs(ctx context.Context, filter *database.ScanJobFilter, pagination *database.Pagination) ([]*types.ScanJob, int64, error) {
	args := m.Called(ctx, filter, pagination)
	return args.Get(0).([]*types.ScanJob), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) CreateScanResult(ctx context.Context, result *types.ScanResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockRepository) GetScanResult(ctx context.Context, id uuid.UUID) (*types.ScanResult, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.ScanResult), args.Error(1)
}

func (m *MockRepository) GetScanResults(ctx context.Context, scanJobID uuid.UUID) ([]*types.ScanResult, error) {
	args := m.Called(ctx, scanJobID)
	return args.Get(0).([]*types.ScanResult), args.Error(1)
}

func (m *MockRepository) CreateFinding(ctx context.Context, finding *types.Finding) error {
	args := m.Called(ctx, finding)
	return args.Error(0)
}

func (m *MockRepository) GetFinding(ctx context.Context, id uuid.UUID) (*types.Finding, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.Finding), args.Error(1)
}

func (m *MockRepository) GetFindings(ctx context.Context, scanJobID uuid.UUID, filter *database.FindingFilter) ([]*types.Finding, error) {
	args := m.Called(ctx, scanJobID, filter)
	return args.Get(0).([]*types.Finding), args.Error(1)
}

func (m *MockRepository) UpdateFinding(ctx context.Context, finding *types.Finding) error {
	args := m.Called(ctx, finding)
	return args.Error(0)
}

func (m *MockRepository) CreateUserFeedback(ctx context.Context, feedback *types.UserFeedback) error {
	args := m.Called(ctx, feedback)
	return args.Error(0)
}

func (m *MockRepository) GetUserFeedback(ctx context.Context, findingID uuid.UUID) ([]*types.UserFeedback, error) {
	args := m.Called(ctx, findingID)
	return args.Get(0).([]*types.UserFeedback), args.Error(1)
}

func (m *MockRepository) CreateRepository(ctx context.Context, repo *types.Repository) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

func (m *MockRepository) GetRepository(ctx context.Context, id uuid.UUID) (*types.Repository, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.Repository), args.Error(1)
}

func (m *MockRepository) UpdateRepository(ctx context.Context, repo *types.Repository) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

func (m *MockRepository) ListRepositories(ctx context.Context, orgID uuid.UUID) ([]*types.Repository, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]*types.Repository), args.Error(1)
}

func (m *MockRepository) CreateOrganization(ctx context.Context, org *types.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockRepository) GetOrganization(ctx context.Context, id uuid.UUID) (*types.Organization, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.Organization), args.Error(1)
}

func (m *MockRepository) UpdateOrganization(ctx context.Context, org *types.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockRepository) CreateUser(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) GetUser(ctx context.Context, id uuid.UUID) (*types.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockRepository) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockRepository) UpdateUser(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// MockQueue is a mock implementation of the queue interface
type MockQueue struct {
	mock.Mock
}

func (m *MockQueue) Enqueue(ctx context.Context, job *queue.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockQueue) Dequeue(ctx context.Context, workerID string) (*queue.Job, error) {
	args := m.Called(ctx, workerID)
	return args.Get(0).(*queue.Job), args.Error(1)
}

func (m *MockQueue) Complete(ctx context.Context, jobID string, result *queue.JobResult) error {
	args := m.Called(ctx, jobID, result)
	return args.Error(0)
}

func (m *MockQueue) Fail(ctx context.Context, jobID string, errorMsg string) error {
	args := m.Called(ctx, jobID, errorMsg)
	return args.Error(0)
}

func (m *MockQueue) Cancel(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockQueue) GetJob(ctx context.Context, jobID string) (*queue.Job, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).(*queue.Job), args.Error(1)
}

func (m *MockQueue) ListJobs(ctx context.Context, filter queue.JobFilter, limit, offset int) ([]*queue.Job, error) {
	args := m.Called(ctx, filter, limit, offset)
	return args.Get(0).([]*queue.Job), args.Error(1)
}

func (m *MockQueue) GetStats(ctx context.Context) (*queue.JobStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*queue.JobStats), args.Error(1)
}

func (m *MockQueue) Cleanup(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockAgent is a mock implementation of the SecurityAgent interface
type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) Scan(ctx context.Context, config agent.ScanConfig) (*agent.ScanResult, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(*agent.ScanResult), args.Error(1)
}

func (m *MockAgent) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAgent) GetConfig() agent.AgentConfig {
	args := m.Called()
	return args.Get(0).(agent.AgentConfig)
}

func (m *MockAgent) GetVersion() agent.VersionInfo {
	args := m.Called()
	return args.Get(0).(agent.VersionInfo)
}

// Test helper functions
func setupTestService(t *testing.T) (*Service, *MockRepository, *MockQueue, *AgentManager) {
	mockDB := &MockRepository{}
	mockQueue := &MockQueue{}
	agentManager := NewAgentManager()
	
	config := DefaultConfig()
	config.WorkerCount = 1 // Use single worker for tests
	
	service := NewService(mockDB, mockQueue, agentManager, config)
	
	return service, mockDB, mockQueue, agentManager
}

func TestNewService(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	
	assert.NotNil(t, service)
	assert.NotNil(t, service.config)
	assert.Equal(t, 1, service.config.WorkerCount)
	assert.False(t, service.running)
}

func TestService_SubmitScan_Success(t *testing.T) {
	service, mockDB, mockQueue, _ := setupTestService(t)
	ctx := context.Background()
	
	// Setup test data
	repoID := uuid.New()
	userID := uuid.New()
	
	req := &ScanRequest{
		RepositoryID: repoID,
		UserID:       &userID,
		RepoURL:      "https://github.com/test/repo.git",
		Branch:       "main",
		CommitSHA:    "abc123",
		ScanType:     types.ScanTypeFull,
		Priority:     5,
		Agents:       []string{"semgrep", "eslint"},
		Timeout:      5 * time.Minute,
	}
	
	// Setup mocks
	mockDB.On("CreateScanJob", ctx, mock.AnythingOfType("*types.ScanJob")).Return(nil)
	mockQueue.On("Enqueue", ctx, mock.AnythingOfType("*queue.Job")).Return(nil)
	
	// Execute
	result, err := service.SubmitScan(ctx, req)
	
	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, repoID, result.RepositoryID)
	assert.Equal(t, &userID, result.UserID)
	assert.Equal(t, "main", result.Branch)
	assert.Equal(t, "abc123", result.CommitSHA)
	assert.Equal(t, types.ScanTypeFull, result.ScanType)
	assert.Equal(t, types.ScanJobStatusQueued, result.Status)
	assert.Equal(t, []string{"semgrep", "eslint"}, result.AgentsRequested)
	
	mockDB.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
}

func TestService_SubmitScan_ValidationError(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	ctx := context.Background()
	
	tests := []struct {
		name string
		req  *ScanRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "empty repository ID",
			req: &ScanRequest{
				RepositoryID: uuid.Nil,
				RepoURL:      "https://github.com/test/repo.git",
				Branch:       "main",
				CommitSHA:    "abc123",
				ScanType:     types.ScanTypeFull,
				Agents:       []string{"semgrep"},
			},
		},
		{
			name: "empty repo URL",
			req: &ScanRequest{
				RepositoryID: uuid.New(),
				RepoURL:      "",
				Branch:       "main",
				CommitSHA:    "abc123",
				ScanType:     types.ScanTypeFull,
				Agents:       []string{"semgrep"},
			},
		},
		{
			name: "empty agents",
			req: &ScanRequest{
				RepositoryID: uuid.New(),
				RepoURL:      "https://github.com/test/repo.git",
				Branch:       "main",
				CommitSHA:    "abc123",
				ScanType:     types.ScanTypeFull,
				Agents:       []string{},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.SubmitScan(ctx, tt.req)
			
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestService_GetScanStatus_Success(t *testing.T) {
	service, mockDB, _, _ := setupTestService(t)
	ctx := context.Background()
	
	// Setup test data
	jobID := uuid.New()
	startTime := time.Now().Add(-5 * time.Minute)
	
	scanJob := &types.ScanJob{
		ID:              jobID,
		Status:          types.ScanJobStatusRunning,
		AgentsRequested: []string{"semgrep", "eslint"},
		AgentsCompleted: []string{"semgrep"},
		StartedAt:       &startTime,
	}
	
	scanResults := []*types.ScanResult{
		{
			AgentName:     "semgrep",
			Status:        "completed",
			FindingsCount: 5,
			DurationMS:    30000,
		},
	}
	
	// Setup mocks
	mockDB.On("GetScanJob", ctx, jobID).Return(scanJob, nil)
	mockDB.On("GetScanResults", ctx, jobID).Return(scanResults, nil)
	
	// Execute
	status, err := service.GetScanStatus(ctx, jobID.String())
	
	// Verify
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, jobID.String(), status.JobID)
	assert.Equal(t, types.ScanJobStatusRunning, status.Status)
	assert.Equal(t, float64(50), status.Progress) // 1 of 2 agents completed
	assert.Equal(t, []string{"semgrep", "eslint"}, status.AgentsRequested)
	assert.Equal(t, []string{"semgrep"}, status.AgentsCompleted)
	assert.Len(t, status.Results, 1)
	assert.Equal(t, "semgrep", status.Results[0].AgentName)
	assert.Equal(t, "completed", status.Results[0].Status)
	assert.Equal(t, 5, status.Results[0].FindingsCount)
	
	mockDB.AssertExpectations(t)
}

func TestService_GetScanStatus_InvalidJobID(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	ctx := context.Background()
	
	// Execute with invalid job ID
	status, err := service.GetScanStatus(ctx, "invalid-uuid")
	
	// Verify
	assert.Error(t, err)
	assert.Nil(t, status)
}

func TestService_CancelScan_Success(t *testing.T) {
	service, mockDB, mockQueue, _ := setupTestService(t)
	ctx := context.Background()
	
	// Setup test data
	jobID := uuid.New()
	scanJob := &types.ScanJob{
		ID:     jobID,
		Status: types.ScanJobStatusQueued,
	}
	
	// Setup mocks
	mockDB.On("GetScanJob", ctx, jobID).Return(scanJob, nil)
	mockDB.On("UpdateScanJob", ctx, mock.AnythingOfType("*types.ScanJob")).Return(nil)
	mockQueue.On("Cancel", ctx, fmt.Sprintf("scan-%s", jobID.String())).Return(nil)
	
	// Execute
	err := service.CancelScan(ctx, jobID.String())
	
	// Verify
	require.NoError(t, err)
	
	mockDB.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
}

func TestService_Health_Success(t *testing.T) {
	service, mockDB, mockQueue, agentManager := setupTestService(t)
	ctx := context.Background()
	
	// Register a mock agent
	mockAgent := &MockAgent{}
	mockAgent.On("GetConfig").Return(agent.AgentConfig{
		Name:           "test-agent",
		SupportedLangs: []string{"javascript"},
	})
	mockAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	
	err := agentManager.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)
	
	// Perform initial health check to set agent as healthy
	err = agentManager.HealthCheck(ctx, "test-agent")
	require.NoError(t, err)
	
	// Setup mocks
	mockDB.On("Health", ctx).Return(nil)
	mockQueue.On("GetStats", ctx).Return(&queue.JobStats{}, nil)
	
	// Start service
	service.running = true
	
	// Execute
	err = service.Health(ctx)
	
	// Verify
	require.NoError(t, err)
	
	mockDB.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
	mockAgent.AssertExpectations(t)
}

func TestService_Health_NotRunning(t *testing.T) {
	service, _, _, _ := setupTestService(t)
	ctx := context.Background()
	
	// Execute without starting service
	err := service.Health(ctx)
	
	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestService_ListScans_Success(t *testing.T) {
	service, mockDB, _, _ := setupTestService(t)
	ctx := context.Background()
	
	// Setup test data
	repoID := uuid.New()
	jobID1 := uuid.New()
	jobID2 := uuid.New()
	
	scanJobs := []*types.ScanJob{
		{
			ID:           jobID1,
			RepositoryID: repoID,
			Branch:       "main",
			CommitSHA:    "abc123",
			ScanType:     types.ScanTypeFull,
			Status:       types.ScanJobStatusCompleted,
			Priority:     5,
			CreatedAt:    time.Now().Add(-1 * time.Hour),
		},
		{
			ID:           jobID2,
			RepositoryID: repoID,
			Branch:       "develop",
			CommitSHA:    "def456",
			ScanType:     types.ScanTypeIncremental,
			Status:       types.ScanJobStatusRunning,
			Priority:     3,
			CreatedAt:    time.Now().Add(-30 * time.Minute),
		},
	}
	
	filter := &ScanFilter{
		RepositoryID: &repoID,
	}
	
	pagination := &Pagination{
		Page:     1,
		PageSize: 10,
	}
	
	// Setup mocks
	mockDB.On("ListScanJobs", ctx, mock.AnythingOfType("*database.ScanJobFilter"), mock.AnythingOfType("*database.Pagination")).Return(scanJobs, int64(2), nil)
	
	// Execute
	result, err := service.ListScans(ctx, filter, pagination)
	
	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Scans, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PageSize)
	assert.Equal(t, int64(1), result.TotalPages)
	
	// Verify first scan
	scan1 := result.Scans[0]
	assert.Equal(t, jobID1.String(), scan1.JobID)
	assert.Equal(t, repoID.String(), scan1.Repository)
	assert.Equal(t, "main", scan1.Branch)
	assert.Equal(t, "abc123", scan1.CommitSHA)
	assert.Equal(t, types.ScanTypeFull, scan1.ScanType)
	assert.Equal(t, types.ScanJobStatusCompleted, scan1.Status)
	assert.Equal(t, 5, scan1.Priority)
	
	mockDB.AssertExpectations(t)
}

// Benchmark tests
func BenchmarkService_SubmitScan(b *testing.B) {
	service, mockDB, mockQueue, _ := setupTestService(&testing.T{})
	ctx := context.Background()
	
	// Setup mocks
	mockDB.On("CreateScanJob", ctx, mock.AnythingOfType("*types.ScanJob")).Return(nil)
	mockQueue.On("Enqueue", ctx, mock.AnythingOfType("*queue.Job")).Return(nil)
	
	req := &ScanRequest{
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
		_, err := service.SubmitScan(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkService_GetScanStatus(b *testing.B) {
	service, mockDB, _, _ := setupTestService(&testing.T{})
	ctx := context.Background()
	
	jobID := uuid.New()
	scanJob := &types.ScanJob{
		ID:              jobID,
		Status:          types.ScanJobStatusCompleted,
		AgentsRequested: []string{"semgrep"},
		AgentsCompleted: []string{"semgrep"},
	}
	
	scanResults := []*types.ScanResult{
		{
			AgentName:     "semgrep",
			Status:        "completed",
			FindingsCount: 5,
		},
	}
	
	// Setup mocks
	mockDB.On("GetScanJob", ctx, jobID).Return(scanJob, nil)
	mockDB.On("GetScanResults", ctx, jobID).Return(scanResults, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetScanStatus(ctx, jobID.String())
		if err != nil {
			b.Fatal(err)
		}
	}
}