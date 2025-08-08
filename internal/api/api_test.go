package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/orchestrator"
	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/types"
)

// MockOrchestrator is a mock implementation of OrchestrationService
type MockOrchestrator struct {
	mock.Mock
}

func (m *MockOrchestrator) SubmitScan(ctx context.Context, req *orchestrator.ScanRequest) (*types.ScanJob, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*types.ScanJob), args.Error(1)
}

func (m *MockOrchestrator) GetScanStatus(ctx context.Context, jobID string) (*orchestrator.ScanStatus, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).(*orchestrator.ScanStatus), args.Error(1)
}

func (m *MockOrchestrator) GetScanResults(ctx context.Context, jobID string, filter *orchestrator.ResultFilter) (*orchestrator.ScanResults, error) {
	args := m.Called(ctx, jobID, filter)
	return args.Get(0).(*orchestrator.ScanResults), args.Error(1)
}

func (m *MockOrchestrator) CancelScan(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockOrchestrator) ListScans(ctx context.Context, filter *orchestrator.ScanFilter, pagination *orchestrator.Pagination) (*orchestrator.ScanList, error) {
	args := m.Called(ctx, filter, pagination)
	return args.Get(0).(*orchestrator.ScanList), args.Error(1)
}

func (m *MockOrchestrator) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockOrchestrator) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockOrchestrator) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockDB is a mock implementation of database.DB
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockRedisClient is a mock implementation of queue.RedisClient
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockRepositories is a mock implementation of database.Repositories
type MockRepositories struct {
	Users    *MockUserRepository
	ScanJobs *MockScanJobRepository
	Findings *MockFindingRepository
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*types.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *types.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockScanJobRepository struct {
	mock.Mock
}

func (m *MockScanJobRepository) Create(ctx context.Context, job *types.ScanJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockScanJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.ScanJob, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.ScanJob), args.Error(1)
}

func (m *MockScanJobRepository) List(ctx context.Context, filter *database.ScanJobFilter, pagination *database.Pagination) ([]*types.ScanJob, int64, error) {
	args := m.Called(ctx, filter, pagination)
	return args.Get(0).([]*types.ScanJob), args.Get(1).(int64), args.Error(2)
}

func (m *MockScanJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockScanJobRepository) Update(ctx context.Context, job *types.ScanJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockScanJobRepository) SetStarted(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockScanJobRepository) SetCompleted(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockScanJobRepository) SetFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	args := m.Called(ctx, id, errorMsg)
	return args.Error(0)
}

func (m *MockScanJobRepository) ListByRepository(ctx context.Context, repoID uuid.UUID, limit, offset int) ([]*types.ScanJob, error) {
	args := m.Called(ctx, repoID, limit, offset)
	return args.Get(0).([]*types.ScanJob), args.Error(1)
}

type MockFindingRepository struct {
	mock.Mock
}

func (m *MockFindingRepository) Create(ctx context.Context, finding *types.Finding) error {
	args := m.Called(ctx, finding)
	return args.Error(0)
}

func (m *MockFindingRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.Finding, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*types.Finding), args.Error(1)
}

func (m *MockFindingRepository) ListByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*types.Finding, error) {
	args := m.Called(ctx, scanJobID)
	return args.Get(0).([]*types.Finding), args.Error(1)
}

func (m *MockFindingRepository) List(ctx context.Context, filter *database.FindingFilter, pagination *database.Pagination) ([]*types.Finding, int64, error) {
	args := m.Called(ctx, filter, pagination)
	return args.Get(0).([]*types.Finding), args.Get(1).(int64), args.Error(2)
}

func (m *MockFindingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

// Test setup helpers

func setupTestRouter() (*gin.Engine, *MockOrchestrator) {
	gin.SetMode(gin.TestMode)

	// Create a simple test router with minimal setup
	router := gin.New()
	
	// Add basic middleware
	router.Use(RequestIDMiddleware())
	router.Use(CORSMiddleware())
	
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret",
			JWTExpiration: time.Hour,
		},
	}
	
	mockOrch := &MockOrchestrator{}
	
	// Add test routes
	router.GET("/health", func(c *gin.Context) {
		SuccessResponse(c, map[string]string{
			"status": "healthy",
		})
	})
	
	router.GET("/api/v1", func(c *gin.Context) {
		SuccessResponse(c, map[string]interface{}{
			"name":    "AgentScan API",
			"version": "1.0.0",
			"status":  "ok",
		})
	})
	
	// Protected routes
	protected := router.Group("/api/v1")
	protected.Use(AuthMiddleware(cfg))
	{
		protected.GET("/user/me", func(c *gin.Context) {
			user, exists := GetCurrentUser(c)
			if !exists {
				UnauthorizedResponse(c, "User not found")
				return
			}
			SuccessResponse(c, ToUserDTO(user))
		})
		
		protected.POST("/scans", func(c *gin.Context) {
			var req CreateScanJobRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				BadRequestResponse(c, "Invalid request body: "+err.Error())
				return
			}
			
			userID, exists := GetCurrentUserID(c)
			if !exists {
				UnauthorizedResponse(c, "User authentication required")
				return
			}
			
			scanReq := &orchestrator.ScanRequest{
				RepositoryID: uuid.New(),
				UserID:       &userID,
				RepoURL:      req.RepositoryURL,
				Branch:       req.Branch,
				ScanType:     req.ScanType,
				Priority:     req.Priority,
				Agents:       req.AgentsRequested,
			}
			
			scanJob, err := mockOrch.SubmitScan(c.Request.Context(), scanReq)
			if err != nil {
				ErrorResponseFromError(c, err)
				return
			}
			
			CreatedResponse(c, ToScanJobDTO(scanJob))
		})
		
		protected.GET("/scans/:id/status", func(c *gin.Context) {
			scanID := c.Param("id")
			status, err := mockOrch.GetScanStatus(c.Request.Context(), scanID)
			if err != nil {
				ErrorResponseFromError(c, err)
				return
			}
			SuccessResponse(c, status)
		})
	}
	
	// Catch-all route
	router.NoRoute(func(c *gin.Context) {
		NotFoundResponse(c, "Endpoint not found")
	})
	
	return router, mockOrch
}

func generateTestJWT(userID uuid.UUID, email, name string) string {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "agentscan",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("test-secret"))
	return tokenString
}

// Tests

func TestHealthEndpoint(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "healthy", data["status"])
}

func TestAPIVersionEndpoint(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/user/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response.Success)
	assert.Equal(t, "unauthorized", response.Error.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	router, _ := setupTestRouter()

	userID := uuid.New()
	token := generateTestJWT(userID, "test@example.com", "Test User")

	req, _ := http.NewRequest("GET", "/api/v1/user/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
}

func TestCreateScan(t *testing.T) {
	router, mockOrch := setupTestRouter()

	userID := uuid.New()
	token := generateTestJWT(userID, "test@example.com", "Test User")

	// Mock orchestrator to return scan job
	scanJob := &types.ScanJob{
		ID:           uuid.New(),
		RepositoryID: uuid.New(),
		UserID:       &userID,
		Branch:       "main",
		ScanType:     "full",
		Status:       types.ScanJobStatusQueued,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	mockOrch.On("SubmitScan", mock.Anything, mock.AnythingOfType("*orchestrator.ScanRequest")).Return(scanJob, nil)

	requestBody := CreateScanJobRequest{
		RepositoryURL: "https://github.com/test/repo",
		Branch:        "main",
		ScanType:      "full",
		Priority:      5,
	}
	
	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/api/v1/scans", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	mockOrch.AssertExpectations(t)
}

func TestGetScanStatus(t *testing.T) {
	router, mockOrch := setupTestRouter()

	userID := uuid.New()
	token := generateTestJWT(userID, "test@example.com", "Test User")
	scanID := uuid.New().String()

	// Mock orchestrator to return scan status
	scanStatus := &orchestrator.ScanStatus{
		JobID:    scanID,
		Status:   "running",
		Progress: 50.0,
	}
	mockOrch.On("GetScanStatus", mock.Anything, scanID).Return(scanStatus, nil)

	req, _ := http.NewRequest("GET", "/api/v1/scans/"+scanID+"/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	mockOrch.AssertExpectations(t)
}

func TestNotFoundEndpoint(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	
	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response.Success)
	assert.Equal(t, "not_found", response.Error.Code)
}