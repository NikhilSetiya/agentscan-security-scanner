package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/agentscan/agentscan/pkg/types"
)

// ScanCache provides caching for scan-related data
type ScanCache struct {
	service *Service
}

// NewScanCache creates a new scan cache
func NewScanCache(service *Service) *ScanCache {
	return &ScanCache{
		service: service,
	}
}

// SetScanResult caches a scan result
func (sc *ScanCache) SetScanResult(ctx context.Context, jobID string, result *types.ScanResult) error {
	key := CacheKey{Prefix: PrefixScanResult, ID: jobID}
	return sc.service.Set(ctx, key, result, sc.service.config.ScanResultTTL)
}

// GetScanResult retrieves a cached scan result
func (sc *ScanCache) GetScanResult(ctx context.Context, jobID string) (*types.ScanResult, error) {
	key := CacheKey{Prefix: PrefixScanResult, ID: jobID}
	var result types.ScanResult
	if err := sc.service.Get(ctx, key, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetScanStatus caches scan status
func (sc *ScanCache) SetScanStatus(ctx context.Context, jobID string, status types.ScanStatus) error {
	key := CacheKey{Prefix: PrefixScanStatus, ID: jobID}
	return sc.service.Set(ctx, key, status, sc.service.config.DefaultTTL)
}

// GetScanStatus retrieves cached scan status
func (sc *ScanCache) GetScanStatus(ctx context.Context, jobID string) (types.ScanStatus, error) {
	key := CacheKey{Prefix: PrefixScanStatus, ID: jobID}
	var status types.ScanStatus
	if err := sc.service.Get(ctx, key, &status); err != nil {
		return "", err
	}
	return status, nil
}

// SetFindings caches scan findings
func (sc *ScanCache) SetFindings(ctx context.Context, jobID string, findings []types.Finding) error {
	key := CacheKey{Prefix: PrefixFindings, ID: jobID}
	return sc.service.SetList(ctx, key, interfaceSlice(findings), sc.service.config.FindingsTTL)
}

// GetFindings retrieves cached findings
func (sc *ScanCache) GetFindings(ctx context.Context, jobID string) ([]types.Finding, error) {
	key := CacheKey{Prefix: PrefixFindings, ID: jobID}
	var findings []types.Finding
	if err := sc.service.GetList(ctx, key, &findings); err != nil {
		return nil, err
	}
	return findings, nil
}

// SetScanProgress caches scan progress information
func (sc *ScanCache) SetScanProgress(ctx context.Context, jobID string, progress *ScanProgress) error {
	key := CacheKey{Prefix: "scan_progress", ID: jobID}
	return sc.service.Set(ctx, key, progress, 5*time.Minute)
}

// GetScanProgress retrieves cached scan progress
func (sc *ScanCache) GetScanProgress(ctx context.Context, jobID string) (*ScanProgress, error) {
	key := CacheKey{Prefix: "scan_progress", ID: jobID}
	var progress ScanProgress
	if err := sc.service.Get(ctx, key, &progress); err != nil {
		return nil, err
	}
	return &progress, nil
}

// SetRepositoryMetadata caches repository metadata
func (sc *ScanCache) SetRepositoryMetadata(ctx context.Context, repoID string, metadata *RepositoryMetadata) error {
	key := CacheKey{Prefix: PrefixRepository, ID: repoID}
	return sc.service.Set(ctx, key, metadata, sc.service.config.RepositoryTTL)
}

// GetRepositoryMetadata retrieves cached repository metadata
func (sc *ScanCache) GetRepositoryMetadata(ctx context.Context, repoID string) (*RepositoryMetadata, error) {
	key := CacheKey{Prefix: PrefixRepository, ID: repoID}
	var metadata RepositoryMetadata
	if err := sc.service.Get(ctx, key, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

// InvalidateScanCache removes all cached data for a scan
func (sc *ScanCache) InvalidateScanCache(ctx context.Context, jobID string) error {
	keys := []CacheKey{
		{Prefix: PrefixScanResult, ID: jobID},
		{Prefix: PrefixScanStatus, ID: jobID},
		{Prefix: PrefixFindings, ID: jobID},
		{Prefix: "scan_progress", ID: jobID},
	}

	for _, key := range keys {
		if err := sc.service.Delete(ctx, key); err != nil {
			// Log error but continue with other keys
			continue
		}
	}

	return nil
}

// InvalidateRepositoryCache removes all cached data for a repository
func (sc *ScanCache) InvalidateRepositoryCache(ctx context.Context, repoID string) error {
	// Invalidate repository metadata
	repoKey := CacheKey{Prefix: PrefixRepository, ID: repoID}
	if err := sc.service.Delete(ctx, repoKey); err != nil {
		return err
	}

	// Invalidate all scan results for this repository
	pattern := fmt.Sprintf("%s:*", PrefixScanResult)
	return sc.service.InvalidatePattern(ctx, pattern)
}

// ScanProgress represents scan execution progress
type ScanProgress struct {
	JobID           string                 `json:"job_id"`
	Status          types.ScanStatus       `json:"status"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	TotalAgents     int                    `json:"total_agents"`
	CompletedAgents int                    `json:"completed_agents"`
	FailedAgents    int                    `json:"failed_agents"`
	AgentProgress   map[string]AgentStatus `json:"agent_progress"`
	EstimatedETA    *time.Time             `json:"estimated_eta,omitempty"`
}

// AgentStatus represents individual agent execution status
type AgentStatus struct {
	Name        string           `json:"name"`
	Status      types.ScanStatus `json:"status"`
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Error       string           `json:"error,omitempty"`
	Progress    float64          `json:"progress"` // 0.0 to 1.0
}

// RepositoryMetadata represents cached repository information
type RepositoryMetadata struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	DefaultBranch string            `json:"default_branch"`
	Languages     []string          `json:"languages"`
	LastScanAt    *time.Time        `json:"last_scan_at,omitempty"`
	ScanCount     int64             `json:"scan_count"`
	Settings      map[string]string `json:"settings"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Helper function to convert typed slice to interface slice
func interfaceSlice[T any](slice []T) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}