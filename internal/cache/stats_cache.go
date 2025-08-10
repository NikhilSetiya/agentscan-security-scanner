package cache

import (
	"context"
	"fmt"
	"time"
)

// StatsCache provides caching for performance statistics and metrics
type StatsCache struct {
	service *Service
}

// NewStatsCache creates a new statistics cache
func NewStatsCache(service *Service) *StatsCache {
	return &StatsCache{
		service: service,
	}
}

// IncrementScanCount increments the scan counter for a repository
func (sc *StatsCache) IncrementScanCount(ctx context.Context, repoID string) (int64, error) {
	key := CacheKey{Prefix: "scan_count", ID: repoID}
	return sc.service.Increment(ctx, key, 1, 24*time.Hour)
}

// IncrementFindingCount increments the finding counter by severity
func (sc *StatsCache) IncrementFindingCount(ctx context.Context, repoID, severity string) (int64, error) {
	key := CacheKey{Prefix: "finding_count", ID: fmt.Sprintf("%s:%s", repoID, severity)}
	return sc.service.Increment(ctx, key, 1, 24*time.Hour)
}

// SetDashboardStats caches dashboard statistics
func (sc *StatsCache) SetDashboardStats(ctx context.Context, userID string, stats *DashboardStats) error {
	key := CacheKey{Prefix: "dashboard_stats", ID: userID}
	return sc.service.Set(ctx, key, stats, 15*time.Minute)
}

// GetDashboardStats retrieves cached dashboard statistics
func (sc *StatsCache) GetDashboardStats(ctx context.Context, userID string) (*DashboardStats, error) {
	key := CacheKey{Prefix: "dashboard_stats", ID: userID}
	var stats DashboardStats
	if err := sc.service.Get(ctx, key, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// SetRepositoryStats caches repository-specific statistics
func (sc *StatsCache) SetRepositoryStats(ctx context.Context, repoID string, stats *RepositoryStats) error {
	key := CacheKey{Prefix: "repo_stats", ID: repoID}
	return sc.service.Set(ctx, key, stats, 30*time.Minute)
}

// GetRepositoryStats retrieves cached repository statistics
func (sc *StatsCache) GetRepositoryStats(ctx context.Context, repoID string) (*RepositoryStats, error) {
	key := CacheKey{Prefix: "repo_stats", ID: repoID}
	var stats RepositoryStats
	if err := sc.service.Get(ctx, key, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// SetAgentPerformance caches agent performance metrics
func (sc *StatsCache) SetAgentPerformance(ctx context.Context, agentName string, metrics *AgentPerformanceMetrics) error {
	key := CacheKey{Prefix: "agent_perf", ID: agentName}
	return sc.service.Set(ctx, key, metrics, 1*time.Hour)
}

// GetAgentPerformance retrieves cached agent performance metrics
func (sc *StatsCache) GetAgentPerformance(ctx context.Context, agentName string) (*AgentPerformanceMetrics, error) {
	key := CacheKey{Prefix: "agent_perf", ID: agentName}
	var metrics AgentPerformanceMetrics
	if err := sc.service.Get(ctx, key, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}

// RecordScanDuration records scan execution time for performance tracking
func (sc *StatsCache) RecordScanDuration(ctx context.Context, agentName string, duration time.Duration) error {
	key := CacheKey{Prefix: "scan_duration", ID: fmt.Sprintf("%s:%d", agentName, time.Now().Unix()/3600)}
	
	// Store duration in milliseconds
	durationMs := duration.Milliseconds()
	_, err := sc.service.Increment(ctx, key, durationMs, 24*time.Hour)
	return err
}

// GetAverageScanDuration calculates average scan duration for an agent
func (sc *StatsCache) GetAverageScanDuration(ctx context.Context, agentName string, hours int) (time.Duration, error) {
	var totalDuration int64
	var count int64
	
	now := time.Now().Unix() / 3600
	for i := 0; i < hours; i++ {
		key := CacheKey{Prefix: "scan_duration", ID: fmt.Sprintf("%s:%d", agentName, now-int64(i))}
		duration, err := sc.service.GetCounter(ctx, key)
		if err == nil && duration > 0 {
			totalDuration += duration
			count++
		}
	}
	
	if count == 0 {
		return 0, nil
	}
	
	avgMs := totalDuration / count
	return time.Duration(avgMs) * time.Millisecond, nil
}

// SetSystemMetrics caches system-wide performance metrics
func (sc *StatsCache) SetSystemMetrics(ctx context.Context, metrics *SystemMetrics) error {
	key := CacheKey{Prefix: "system_metrics", ID: "current"}
	return sc.service.Set(ctx, key, metrics, 5*time.Minute)
}

// GetSystemMetrics retrieves cached system metrics
func (sc *StatsCache) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	key := CacheKey{Prefix: "system_metrics", ID: "current"}
	var metrics SystemMetrics
	if err := sc.service.Get(ctx, key, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}

// InvalidateStatsCache removes all cached statistics
func (sc *StatsCache) InvalidateStatsCache(ctx context.Context) error {
	patterns := []string{
		"dashboard_stats:*",
		"repo_stats:*",
		"agent_perf:*",
		"system_metrics:*",
	}
	
	for _, pattern := range patterns {
		if err := sc.service.InvalidatePattern(ctx, pattern); err != nil {
			return err
		}
	}
	
	return nil
}

// DashboardStats represents cached dashboard statistics
type DashboardStats struct {
	TotalScans       int64                    `json:"total_scans"`
	TotalFindings    int64                    `json:"total_findings"`
	FindingsBySeverity map[string]int64       `json:"findings_by_severity"`
	RecentScans      []RecentScanInfo         `json:"recent_scans"`
	TrendData        []TrendDataPoint         `json:"trend_data"`
	TopRepositories  []RepositorySummary      `json:"top_repositories"`
	UpdatedAt        time.Time                `json:"updated_at"`
}

// RepositoryStats represents cached repository statistics
type RepositoryStats struct {
	RepositoryID     string                   `json:"repository_id"`
	TotalScans       int64                    `json:"total_scans"`
	TotalFindings    int64                    `json:"total_findings"`
	FindingsBySeverity map[string]int64       `json:"findings_by_severity"`
	FindingsByTool   map[string]int64         `json:"findings_by_tool"`
	AverageScanTime  time.Duration            `json:"average_scan_time"`
	LastScanAt       *time.Time               `json:"last_scan_at"`
	TrendData        []TrendDataPoint         `json:"trend_data"`
	UpdatedAt        time.Time                `json:"updated_at"`
}

// AgentPerformanceMetrics represents cached agent performance data
type AgentPerformanceMetrics struct {
	AgentName        string        `json:"agent_name"`
	TotalExecutions  int64         `json:"total_executions"`
	SuccessfulRuns   int64         `json:"successful_runs"`
	FailedRuns       int64         `json:"failed_runs"`
	AverageRunTime   time.Duration `json:"average_run_time"`
	AverageFindings  float64       `json:"average_findings"`
	SuccessRate      float64       `json:"success_rate"`
	LastExecutionAt  *time.Time    `json:"last_execution_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// SystemMetrics represents cached system-wide metrics
type SystemMetrics struct {
	ActiveScans      int64         `json:"active_scans"`
	QueuedScans      int64         `json:"queued_scans"`
	TotalAgents      int           `json:"total_agents"`
	HealthyAgents    int           `json:"healthy_agents"`
	CPUUsage         float64       `json:"cpu_usage"`
	MemoryUsage      float64       `json:"memory_usage"`
	DatabaseConnections int        `json:"database_connections"`
	RedisConnections int           `json:"redis_connections"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// RecentScanInfo represents information about recent scans
type RecentScanInfo struct {
	JobID        string    `json:"job_id"`
	RepositoryName string  `json:"repository_name"`
	Branch       string    `json:"branch"`
	Status       string    `json:"status"`
	FindingsCount int      `json:"findings_count"`
	Duration     time.Duration `json:"duration"`
	CompletedAt  time.Time `json:"completed_at"`
}

// TrendDataPoint represents a data point in trend analysis
type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int64     `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// RepositorySummary represents summary information about a repository
type RepositorySummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ScanCount    int64     `json:"scan_count"`
	FindingCount int64     `json:"finding_count"`
	LastScanAt   *time.Time `json:"last_scan_at"`
}