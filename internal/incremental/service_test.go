package incremental

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/pkg/types"
)

// MockCacheRepository is a mock implementation of CacheRepository
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) GetCache(ctx context.Context, repoID uuid.UUID, filePath, toolName, contentHash, configHash string) (*ScanCache, error) {
	args := m.Called(ctx, repoID, filePath, toolName, contentHash, configHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ScanCache), args.Error(1)
}

func (m *MockCacheRepository) SetCache(ctx context.Context, cache *ScanCache) error {
	args := m.Called(ctx, cache)
	return args.Error(0)
}

func (m *MockCacheRepository) InvalidateCache(ctx context.Context, repoID uuid.UUID, filePaths []string) error {
	args := m.Called(ctx, repoID, filePaths)
	return args.Error(0)
}

func (m *MockCacheRepository) GetCacheStats(ctx context.Context, repoID uuid.UUID) (*CacheStats, error) {
	args := m.Called(ctx, repoID)
	return args.Get(0).(*CacheStats), args.Error(1)
}

func (m *MockCacheRepository) CleanupExpiredCache(ctx context.Context, maxAge time.Duration) error {
	args := m.Called(ctx, maxAge)
	return args.Error(0)
}

// MockGitRepository is a mock implementation of GitRepository
type MockGitRepository struct {
	mock.Mock
}

func (m *MockGitRepository) GetFileChanges(ctx context.Context, workingDir, fromCommit, toCommit string) ([]*FileChange, error) {
	args := m.Called(ctx, workingDir, fromCommit, toCommit)
	return args.Get(0).([]*FileChange), args.Error(1)
}

func (m *MockGitRepository) GetFileContent(ctx context.Context, workingDir, filePath string) ([]byte, error) {
	args := m.Called(ctx, workingDir, filePath)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockGitRepository) GetLastCommit(ctx context.Context, workingDir, branch string) (string, error) {
	args := m.Called(ctx, workingDir, branch)
	return args.String(0), args.Error(1)
}

func (m *MockGitRepository) HasConfigChanges(ctx context.Context, workingDir, fromCommit, toCommit string) (bool, error) {
	args := m.Called(ctx, workingDir, fromCommit, toCommit)
	return args.Bool(0), args.Error(1)
}

func setupService() (*Service, *MockCacheRepository, *MockGitRepository) {
	mockCache := &MockCacheRepository{}
	mockGit := &MockGitRepository{}
	service := NewService(mockCache, mockGit)
	return service, mockCache, mockGit
}

func TestService_AnalyzeScanStrategy_NoPreviousCommit(t *testing.T) {
	service, _, _ := setupService()
	ctx := context.Background()

	req := &IncrementalScanRequest{
		RepositoryID:   uuid.New(),
		CurrentCommit:  "abc123",
		PreviousCommit: "", // No previous commit
		Branch:         "main",
		WorkingDir:     "/tmp/repo",
		Tools:          []string{"semgrep", "eslint"},
	}

	result, err := service.AnalyzeScanStrategy(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, ScanStrategyFull, result.Strategy)
	assert.Contains(t, result.Reason, "No previous commit")
	assert.Empty(t, result.FilesToScan)
	assert.Empty(t, result.CachedResults)
	assert.Empty(t, result.ChangedFiles)
}

func TestService_AnalyzeScanStrategy_ConfigChanges(t *testing.T) {
	service, _, mockGit := setupService()
	ctx := context.Background()

	req := &IncrementalScanRequest{
		RepositoryID:   uuid.New(),
		CurrentCommit:  "abc123",
		PreviousCommit: "def456",
		Branch:         "main",
		WorkingDir:     "/tmp/repo",
		Tools:          []string{"semgrep"},
	}

	// Mock config changes detected
	mockGit.On("HasConfigChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(true, nil)

	result, err := service.AnalyzeScanStrategy(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, ScanStrategyFull, result.Strategy)
	assert.Contains(t, result.Reason, "Configuration changes")

	mockGit.AssertExpectations(t)
}

func TestService_AnalyzeScanStrategy_TooManyChanges(t *testing.T) {
	service, _, mockGit := setupService()
	ctx := context.Background()

	req := &IncrementalScanRequest{
		RepositoryID:   uuid.New(),
		CurrentCommit:  "abc123",
		PreviousCommit: "def456",
		Branch:         "main",
		WorkingDir:     "/tmp/repo",
		Tools:          []string{"semgrep"},
	}

	// Mock no config changes
	mockGit.On("HasConfigChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(false, nil)

	// Mock many file changes (over threshold)
	changes := make([]*FileChange, 150) // Over the 100 file threshold
	for i := 0; i < 150; i++ {
		changes[i] = &FileChange{
			Path:       fmt.Sprintf("file%d.go", i),
			ChangeType: ChangeTypeModified,
		}
	}
	mockGit.On("GetFileChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(changes, nil)

	result, err := service.AnalyzeScanStrategy(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, ScanStrategyFull, result.Strategy)
	assert.Contains(t, result.Reason, "Too many files changed")

	mockGit.AssertExpectations(t)
}

func TestService_AnalyzeScanStrategy_IncrementalWithCache(t *testing.T) {
	service, mockCache, mockGit := setupService()
	ctx := context.Background()

	req := &IncrementalScanRequest{
		RepositoryID:   uuid.New(),
		CurrentCommit:  "abc123",
		PreviousCommit: "def456",
		Branch:         "main",
		WorkingDir:     "/tmp/repo",
		Tools:          []string{"semgrep"},
	}

	// Mock no config changes
	mockGit.On("HasConfigChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(false, nil)

	// Mock file changes
	changes := []*FileChange{
		{Path: "src/main.go", ChangeType: ChangeTypeModified},
		{Path: "src/utils.go", ChangeType: ChangeTypeAdded},
		{Path: "README.md", ChangeType: ChangeTypeModified}, // Non-scannable file
	}
	mockGit.On("GetFileChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(changes, nil)

	// Mock file content for hash calculation
	mockGit.On("GetFileContent", ctx, req.WorkingDir, "src/main.go").Return([]byte("package main\nfunc main() {}"), nil)
	mockGit.On("GetFileContent", ctx, req.WorkingDir, "src/utils.go").Return([]byte("package main\nfunc utils() {}"), nil)

	// Mock cache hit for main.go
	cachedResult := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: req.RepositoryID,
		FilePath:     "src/main.go",
		ContentHash:  "hash123",
		ToolName:     "semgrep",
		Results:      map[string]interface{}{"findings": []interface{}{}},
	}
	mockCache.On("GetCache", ctx, req.RepositoryID, "src/main.go", "semgrep", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(cachedResult, nil)

	// Mock cache miss for utils.go
	mockCache.On("GetCache", ctx, req.RepositoryID, "src/utils.go", "semgrep", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)

	result, err := service.AnalyzeScanStrategy(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, ScanStrategyIncremental, result.Strategy)
	assert.Contains(t, result.FilesToScan, "src/utils.go")
	assert.NotContains(t, result.FilesToScan, "src/main.go") // Should be cached
	assert.Len(t, result.CachedResults, 1)
	assert.Equal(t, "src/main.go", result.CachedResults[0].FilePath)
	assert.Greater(t, result.CacheHitRate, 0.0)

	mockGit.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestService_AnalyzeScanStrategy_LowCacheHitRate(t *testing.T) {
	service, mockCache, mockGit := setupService()
	ctx := context.Background()

	req := &IncrementalScanRequest{
		RepositoryID:   uuid.New(),
		CurrentCommit:  "abc123",
		PreviousCommit: "def456",
		Branch:         "main",
		WorkingDir:     "/tmp/repo",
		Tools:          []string{"semgrep"},
	}

	// Mock no config changes
	mockGit.On("HasConfigChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(false, nil)

	// Mock many file changes with low cache hit rate
	changes := []*FileChange{
		{Path: "src/file1.go", ChangeType: ChangeTypeModified},
		{Path: "src/file2.go", ChangeType: ChangeTypeModified},
		{Path: "src/file3.go", ChangeType: ChangeTypeModified},
		{Path: "src/file4.go", ChangeType: ChangeTypeModified},
	}
	mockGit.On("GetFileChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(changes, nil)

	// Mock file content for all files
	for _, change := range changes {
		mockGit.On("GetFileContent", ctx, req.WorkingDir, change.Path).Return([]byte("package main"), nil)
		// Mock cache miss for all files
		mockCache.On("GetCache", ctx, req.RepositoryID, change.Path, "semgrep", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)
	}

	result, err := service.AnalyzeScanStrategy(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, ScanStrategyFull, result.Strategy) // Should fall back to full scan due to low cache hit rate
	assert.Contains(t, result.Reason, "Low cache hit rate")

	mockGit.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestService_CacheResults(t *testing.T) {
	service, mockCache, _ := setupService()
	ctx := context.Background()

	repoID := uuid.New()
	filePath := "src/main.go"
	toolName := "semgrep"
	toolVersion := "1.0.0"
	results := map[string]interface{}{
		"content_hash": "abc123",
		"findings":     []interface{}{},
	}

	mockCache.On("SetCache", ctx, mock.MatchedBy(func(cache *ScanCache) bool {
		return cache.RepositoryID == repoID &&
			cache.FilePath == filePath &&
			cache.ToolName == toolName &&
			cache.ToolVersion == toolVersion &&
			cache.ContentHash == "abc123"
	})).Return(nil)

	err := service.CacheResults(ctx, repoID, filePath, toolName, toolVersion, results)

	require.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestService_InvalidateCache(t *testing.T) {
	service, mockCache, _ := setupService()
	ctx := context.Background()

	repoID := uuid.New()
	filePaths := []string{"src/main.go", "src/utils.go"}

	mockCache.On("InvalidateCache", ctx, repoID, filePaths).Return(nil)

	err := service.InvalidateCache(ctx, repoID, filePaths)

	require.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func TestService_GetCacheStats(t *testing.T) {
	service, mockCache, _ := setupService()
	ctx := context.Background()

	repoID := uuid.New()
	expectedStats := &CacheStats{
		TotalEntries: 100,
		HitRate:      0.75,
		MissRate:     0.25,
		SizeBytes:    1024000,
	}

	mockCache.On("GetCacheStats", ctx, repoID).Return(expectedStats, nil)

	stats, err := service.GetCacheStats(ctx, repoID)

	require.NoError(t, err)
	assert.Equal(t, expectedStats, stats)
	mockCache.AssertExpectations(t)
}

func TestService_GetIncrementalScanPlan(t *testing.T) {
	service, mockCache, mockGit := setupService()
	ctx := context.Background()

	req := &IncrementalScanRequest{
		RepositoryID:   uuid.New(),
		CurrentCommit:  "abc123",
		PreviousCommit: "def456",
		Branch:         "main",
		WorkingDir:     "/tmp/repo",
		Tools:          []string{"semgrep"},
	}

	// Mock incremental scan analysis with multiple files for better cache hit rate
	mockGit.On("HasConfigChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(false, nil)
	mockGit.On("GetFileChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return([]*FileChange{
		{Path: "src/main.js", ChangeType: ChangeTypeModified},
		{Path: "src/utils.js", ChangeType: ChangeTypeModified},
		{Path: "src/cached.js", ChangeType: ChangeTypeModified},
	}, nil)
	
	// Mock file content
	mockGit.On("GetFileContent", ctx, req.WorkingDir, "src/main.js").Return([]byte("console.log('hello')"), nil)
	mockGit.On("GetFileContent", ctx, req.WorkingDir, "src/utils.js").Return([]byte("function utils() {}"), nil)
	mockGit.On("GetFileContent", ctx, req.WorkingDir, "src/cached.js").Return([]byte("function cached() {}"), nil)

	// Mock cache miss for some files, hit for others to get good cache hit rate
	mockCache.On("GetCache", ctx, req.RepositoryID, "src/main.js", "semgrep", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)
	mockCache.On("GetCache", ctx, req.RepositoryID, "src/utils.js", "semgrep", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)
	
	// Cache hit for cached.js
	cachedResult := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: req.RepositoryID,
		FilePath:     "src/cached.js",
		ContentHash:  "hash123",
		ToolName:     "semgrep",
		Results:      map[string]interface{}{"findings": []interface{}{}},
	}
	mockCache.On("GetCache", ctx, req.RepositoryID, "src/cached.js", "semgrep", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(cachedResult, nil)

	scanJob, err := service.GetIncrementalScanPlan(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, req.RepositoryID, scanJob.RepositoryID)
	assert.Equal(t, req.Branch, scanJob.Branch)
	assert.Equal(t, req.CurrentCommit, scanJob.CommitSHA)
	assert.Equal(t, types.ScanTypeIncremental, scanJob.ScanType)
	assert.Equal(t, req.Tools, scanJob.AgentsRequested)
	assert.NotNil(t, scanJob.Metadata["incremental_analysis"])

	mockGit.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestService_shouldScanFile(t *testing.T) {
	service, _, _ := setupService()

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "Go file",
			filePath: "main.go",
			expected: true,
		},
		{
			name:     "JavaScript file",
			filePath: "app.js",
			expected: true,
		},
		{
			name:     "TypeScript file",
			filePath: "component.ts",
			expected: true,
		},
		{
			name:     "Python file",
			filePath: "script.py",
			expected: true,
		},
		{
			name:     "Text file",
			filePath: "README.txt",
			expected: false,
		},
		{
			name:     "Markdown file",
			filePath: "README.md",
			expected: false,
		},
		{
			name:     "Image file",
			filePath: "logo.png",
			expected: false,
		},
		{
			name:     "No extension",
			filePath: "Dockerfile",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.shouldScanFile(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileChange_Types(t *testing.T) {
	tests := []struct {
		name       string
		changeType ChangeType
		expected   string
	}{
		{
			name:       "added file",
			changeType: ChangeTypeAdded,
			expected:   "added",
		},
		{
			name:       "modified file",
			changeType: ChangeTypeModified,
			expected:   "modified",
		},
		{
			name:       "deleted file",
			changeType: ChangeTypeDeleted,
			expected:   "deleted",
		},
		{
			name:       "renamed file",
			changeType: ChangeTypeRenamed,
			expected:   "renamed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.changeType))
		})
	}
}

// Benchmark tests
func BenchmarkService_AnalyzeScanStrategy(b *testing.B) {
	service, mockCache, mockGit := setupService()
	ctx := context.Background()

	req := &IncrementalScanRequest{
		RepositoryID:   uuid.New(),
		CurrentCommit:  "abc123",
		PreviousCommit: "def456",
		Branch:         "main",
		WorkingDir:     "/tmp/repo",
		Tools:          []string{"semgrep"},
	}

	// Setup mocks
	mockGit.On("HasConfigChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return(false, nil)
	mockGit.On("GetFileChanges", ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit).Return([]*FileChange{
		{Path: "src/main.go", ChangeType: ChangeTypeModified},
	}, nil)
	mockGit.On("GetFileContent", ctx, req.WorkingDir, "src/main.go").Return([]byte("package main"), nil)
	mockCache.On("GetCache", ctx, req.RepositoryID, "src/main.go", "semgrep", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.AnalyzeScanStrategy(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkService_shouldScanFile(b *testing.B) {
	service, _, _ := setupService()
	filePath := "src/main.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.shouldScanFile(filePath)
	}
}