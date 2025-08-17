package incremental

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// ScanCache represents a cached scan result for a file
type ScanCache struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	RepositoryID uuid.UUID              `json:"repository_id" db:"repository_id"`
	FilePath     string                 `json:"file_path" db:"file_path"`
	ContentHash  string                 `json:"content_hash" db:"content_hash"`
	ToolName     string                 `json:"tool_name" db:"tool_name"`
	ToolVersion  string                 `json:"tool_version" db:"tool_version"`
	ConfigHash   string                 `json:"config_hash" db:"config_hash"`
	Results      map[string]interface{} `json:"results" db:"results"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// FileChange represents a changed file in a git diff
type FileChange struct {
	Path       string     `json:"path"`
	ChangeType ChangeType `json:"change_type"`
	OldPath    string     `json:"old_path,omitempty"` // For renames
}

// ChangeType represents the type of file change
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeDeleted  ChangeType = "deleted"
	ChangeTypeRenamed  ChangeType = "renamed"
)

// ScanStrategy represents the scanning strategy to use
type ScanStrategy string

const (
	ScanStrategyFull        ScanStrategy = "full"
	ScanStrategyIncremental ScanStrategy = "incremental"
)

// IncrementalScanRequest represents a request for incremental scanning
type IncrementalScanRequest struct {
	RepositoryID   uuid.UUID `json:"repository_id"`
	CurrentCommit  string    `json:"current_commit"`
	PreviousCommit string    `json:"previous_commit,omitempty"`
	Branch         string    `json:"branch"`
	WorkingDir     string    `json:"working_dir"`
	Tools          []string  `json:"tools"`
}

// IncrementalScanResult represents the result of incremental scan analysis
type IncrementalScanResult struct {
	Strategy      ScanStrategy   `json:"strategy"`
	FilesToScan   []string       `json:"files_to_scan"`
	CachedResults []*ScanCache   `json:"cached_results"`
	ChangedFiles  []*FileChange  `json:"changed_files"`
	Reason        string         `json:"reason"`
	CacheHitRate  float64        `json:"cache_hit_rate"`
}

// Service provides incremental scanning functionality
type Service struct {
	cacheRepo CacheRepository
	gitRepo   GitRepository
}

// CacheRepository defines the interface for cache operations
type CacheRepository interface {
	GetCache(ctx context.Context, repoID uuid.UUID, filePath, toolName, contentHash, configHash string) (*ScanCache, error)
	SetCache(ctx context.Context, cache *ScanCache) error
	InvalidateCache(ctx context.Context, repoID uuid.UUID, filePaths []string) error
	GetCacheStats(ctx context.Context, repoID uuid.UUID) (*CacheStats, error)
	CleanupExpiredCache(ctx context.Context, maxAge time.Duration) error
}

// GitRepository defines the interface for git operations
type GitRepository interface {
	GetFileChanges(ctx context.Context, workingDir, fromCommit, toCommit string) ([]*FileChange, error)
	GetFileContent(ctx context.Context, workingDir, filePath string) ([]byte, error)
	GetLastCommit(ctx context.Context, workingDir, branch string) (string, error)
	HasConfigChanges(ctx context.Context, workingDir, fromCommit, toCommit string) (bool, error)
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalEntries int64   `json:"total_entries"`
	HitRate      float64 `json:"hit_rate"`
	MissRate     float64 `json:"miss_rate"`
	SizeBytes    int64   `json:"size_bytes"`
}

// NewService creates a new incremental scanning service
func NewService(cacheRepo CacheRepository, gitRepo GitRepository) *Service {
	return &Service{
		cacheRepo: cacheRepo,
		gitRepo:   gitRepo,
	}
}

// AnalyzeScanStrategy analyzes whether to use incremental or full scanning
func (s *Service) AnalyzeScanStrategy(ctx context.Context, req *IncrementalScanRequest) (*IncrementalScanResult, error) {
	result := &IncrementalScanResult{
		Strategy:      ScanStrategyFull,
		FilesToScan:   []string{},
		CachedResults: []*ScanCache{},
		ChangedFiles:  []*FileChange{},
		Reason:        "Initial analysis",
	}

	// If no previous commit, do full scan
	if req.PreviousCommit == "" {
		result.Reason = "No previous commit found - performing full scan"
		return result, nil
	}

	// Check for configuration changes that would invalidate cache
	hasConfigChanges, err := s.gitRepo.HasConfigChanges(ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to check config changes: %w", err)
	}

	if hasConfigChanges {
		result.Reason = "Configuration changes detected - performing full scan"
		return result, nil
	}

	// Get file changes between commits
	changes, err := s.gitRepo.GetFileChanges(ctx, req.WorkingDir, req.PreviousCommit, req.CurrentCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get file changes: %w", err)
	}

	result.ChangedFiles = changes

	// If too many files changed, do full scan
	if len(changes) > 100 { // Configurable threshold
		result.Reason = fmt.Sprintf("Too many files changed (%d) - performing full scan", len(changes))
		return result, nil
	}

	// Analyze each changed file
	var filesToScan []string
	var cachedResults []*ScanCache
	totalFiles := 0
	cachedFiles := 0

	for _, change := range changes {
		if change.ChangeType == ChangeTypeDeleted {
			// Skip deleted files but invalidate their cache
			s.cacheRepo.InvalidateCache(ctx, req.RepositoryID, []string{change.Path})
			continue
		}

		filePath := change.Path
		if change.ChangeType == ChangeTypeRenamed && change.OldPath != "" {
			// Invalidate old path cache
			s.cacheRepo.InvalidateCache(ctx, req.RepositoryID, []string{change.OldPath})
		}

		// Check if file should be scanned (based on extension)
		if !s.shouldScanFile(filePath) {
			continue
		}

		totalFiles++

		// Get file content hash
		contentHash, err := s.getFileContentHash(ctx, req.WorkingDir, filePath)
		if err != nil {
			// If we can't get content hash, add to scan list
			filesToScan = append(filesToScan, filePath)
			continue
		}

		// Check cache for each tool
		fileNeedsScanning := false
		for _, toolName := range req.Tools {
			configHash := s.getToolConfigHash(toolName) // Tool-specific config hash

			cache, err := s.cacheRepo.GetCache(ctx, req.RepositoryID, filePath, toolName, contentHash, configHash)
			if err != nil || cache == nil {
				// Cache miss - need to scan
				fileNeedsScanning = true
				break
			}

			// Cache hit
			cachedResults = append(cachedResults, cache)
		}

		if fileNeedsScanning {
			filesToScan = append(filesToScan, filePath)
		} else {
			cachedFiles++
		}
	}

	// Calculate cache hit rate
	if totalFiles > 0 {
		result.CacheHitRate = float64(cachedFiles) / float64(totalFiles)
	}

	// Decide strategy based on cache hit rate
	if result.CacheHitRate > 0.3 { // If more than 30% cache hits, use incremental
		result.Strategy = ScanStrategyIncremental
		result.FilesToScan = filesToScan
		result.CachedResults = cachedResults
		result.Reason = fmt.Sprintf("Incremental scan: %d files to scan, %d cached (%.1f%% hit rate)", 
			len(filesToScan), cachedFiles, result.CacheHitRate*100)
	} else {
		result.Strategy = ScanStrategyFull
		result.Reason = fmt.Sprintf("Low cache hit rate (%.1f%%) - performing full scan", result.CacheHitRate*100)
	}

	return result, nil
}

// CacheResults caches the scan results for future incremental scans
func (s *Service) CacheResults(ctx context.Context, repoID uuid.UUID, filePath, toolName, toolVersion string, results map[string]interface{}) error {
	// Get current file content hash
	contentHash, err := s.getFileContentHashFromResults(results)
	if err != nil {
		return fmt.Errorf("failed to get content hash: %w", err)
	}

	configHash := s.getToolConfigHash(toolName)

	cache := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: repoID,
		FilePath:     filePath,
		ContentHash:  contentHash,
		ToolName:     toolName,
		ToolVersion:  toolVersion,
		ConfigHash:   configHash,
		Results:      results,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return s.cacheRepo.SetCache(ctx, cache)
}

// InvalidateCache invalidates cache entries for specific files
func (s *Service) InvalidateCache(ctx context.Context, repoID uuid.UUID, filePaths []string) error {
	return s.cacheRepo.InvalidateCache(ctx, repoID, filePaths)
}

// GetCacheStats returns cache statistics for a repository
func (s *Service) GetCacheStats(ctx context.Context, repoID uuid.UUID) (*CacheStats, error) {
	return s.cacheRepo.GetCacheStats(ctx, repoID)
}

// CleanupExpiredCache removes old cache entries
func (s *Service) CleanupExpiredCache(ctx context.Context, maxAge time.Duration) error {
	return s.cacheRepo.CleanupExpiredCache(ctx, maxAge)
}

// Helper methods

func (s *Service) shouldScanFile(filePath string) bool {
	// Define scannable file extensions
	scannableExtensions := map[string]bool{
		".go":   true,
		".js":   true,
		".ts":   true,
		".jsx":  true,
		".tsx":  true,
		".py":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".hpp":  true,
		".cs":   true,
		".php":  true,
		".rb":   true,
		".rs":   true,
		".kt":   true,
		".scala": true,
		".swift": true,
		".m":    true,
		".mm":   true,
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	return scannableExtensions[ext]
}

func (s *Service) getFileContentHash(ctx context.Context, workingDir, filePath string) (string, error) {
	content, err := s.gitRepo.GetFileContent(ctx, workingDir, filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash), nil
}

func (s *Service) getFileContentHashFromResults(results map[string]interface{}) (string, error) {
	// Extract content hash from scan results if available
	if hash, ok := results["content_hash"].(string); ok {
		return hash, nil
	}

	// If not available in results, we'll need to compute it
	// This is a fallback and should ideally be provided by the scanner
	return "", fmt.Errorf("content hash not available in results")
}

func (s *Service) getToolConfigHash(toolName string) string {
	// Generate a hash of the tool configuration
	// This should include tool version, rules, and any configuration that affects results
	// For now, we'll use a simple approach
	
	// TODO: Implement proper tool configuration hashing
	// This should include:
	// - Tool version
	// - Rule configuration
	// - Custom rules
	// - Exclusion patterns
	// - Severity thresholds
	
	configString := fmt.Sprintf("%s-v1.0", toolName) // Placeholder
	hash := sha256.Sum256([]byte(configString))
	return fmt.Sprintf("%x", hash)
}

// GetIncrementalScanPlan creates a scan plan based on incremental analysis
func (s *Service) GetIncrementalScanPlan(ctx context.Context, req *IncrementalScanRequest) (*types.ScanJob, error) {
	analysis, err := s.AnalyzeScanStrategy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze scan strategy: %w", err)
	}

	scanJob := &types.ScanJob{
		ID:           uuid.New(),
		RepositoryID: req.RepositoryID,
		Branch:       req.Branch,
		CommitSHA:    req.CurrentCommit,
		ScanType:     string(analysis.Strategy),
		Priority:     types.PriorityMedium,
		Status:       types.ScanJobStatusQueued,
		Metadata: map[string]interface{}{
			"incremental_analysis": analysis,
			"files_to_scan":       analysis.FilesToScan,
			"cache_hit_rate":      analysis.CacheHitRate,
			"strategy_reason":     analysis.Reason,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set agents based on strategy
	if analysis.Strategy == ScanStrategyIncremental {
		scanJob.AgentsRequested = req.Tools
		scanJob.ScanType = types.ScanTypeIncremental
	} else {
		scanJob.AgentsRequested = req.Tools
		scanJob.ScanType = types.ScanTypeFull
	}

	return scanJob, nil
}