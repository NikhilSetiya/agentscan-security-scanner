package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/agentscan/agentscan/pkg/types"
)

// UserRepository defines user-specific repository operations
type UserRepository interface {
	BaseRepository[types.User, uuid.UUID]
	RepositoryWithAudit[types.User, uuid.UUID]
	RepositoryWithCache[types.User, uuid.UUID]
	
	// User-specific methods
	GetByEmail(ctx context.Context, email string) (*types.User, error)
	GetBySupabaseID(ctx context.Context, supabaseID string) (*types.User, error)
	GetByGitHubID(ctx context.Context, githubID int) (*types.User, error)
	GetByGitLabID(ctx context.Context, gitlabID int) (*types.User, error)
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID, loginTime time.Time) error
	GetActiveUsers(ctx context.Context) ([]*types.User, error)
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	ActivateUser(ctx context.Context, id uuid.UUID) error
	SearchUsers(ctx context.Context, query string, limit, offset int) ([]*types.User, int, error)
}

// OrganizationRepository defines organization-specific repository operations
type OrganizationRepository interface {
	BaseRepository[types.Organization, uuid.UUID]
	RepositoryWithAudit[types.Organization, uuid.UUID]
	RepositoryWithCache[types.Organization, uuid.UUID]
	
	// Organization-specific methods
	GetByName(ctx context.Context, name string) (*types.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*types.Organization, error)
	GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*types.Organization, error)
	AddUserToOrganization(ctx context.Context, orgID, userID uuid.UUID, role string) error
	RemoveUserFromOrganization(ctx context.Context, orgID, userID uuid.UUID) error
	GetOrganizationUsers(ctx context.Context, orgID uuid.UUID) ([]*types.User, error)
	UpdateUserRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
	GetUserRole(ctx context.Context, orgID, userID uuid.UUID) (string, error)
	IsUserMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error)
}

// RepositoryRepository defines repository-specific operations
type RepositoryRepository interface {
	BaseRepository[types.Repository, uuid.UUID]
	RepositoryWithAudit[types.Repository, uuid.UUID]
	RepositoryWithCache[types.Repository, uuid.UUID]
	
	// Repository-specific methods
	GetByURL(ctx context.Context, url string) (*types.Repository, error)
	GetByProviderID(ctx context.Context, provider, providerID string) (*types.Repository, error)
	GetByOrganization(ctx context.Context, orgID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.Repository, int, error)
	UpdateLastScanAt(ctx context.Context, id uuid.UUID, scanTime time.Time) error
	GetActiveRepositories(ctx context.Context) ([]*types.Repository, error)
	GetRepositoriesByLanguage(ctx context.Context, language string) ([]*types.Repository, error)
	GetRepositoriesByProvider(ctx context.Context, provider string) ([]*types.Repository, error)
	SearchRepositories(ctx context.Context, query string, orgID *uuid.UUID, limit, offset int) ([]*types.Repository, int, error)
	GetRepositoryStatistics(ctx context.Context, id uuid.UUID) (*RepositoryStatistics, error)
}

// ScanJobRepository defines scan job-specific operations
type ScanJobRepository interface {
	BaseRepository[types.ScanJob, uuid.UUID]
	RepositoryWithAudit[types.ScanJob, uuid.UUID]
	RepositoryWithMetrics[types.ScanJob, uuid.UUID]
	
	// Scan job-specific methods
	GetByRepository(ctx context.Context, repoID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.ScanJob, int, error)
	GetByUser(ctx context.Context, userID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.ScanJob, int, error)
	GetByStatus(ctx context.Context, status string) ([]*types.ScanJob, error)
	GetByStatusAndPriority(ctx context.Context, status string, minPriority int) ([]*types.ScanJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, message *string) error
	GetRunningJobs(ctx context.Context) ([]*types.ScanJob, error)
	GetQueuedJobs(ctx context.Context, limit int) ([]*types.ScanJob, error)
	GetJobsInQueue(ctx context.Context, maxPriority int, limit int) ([]*types.ScanJob, error)
	MarkAsStarted(ctx context.Context, id uuid.UUID) error
	MarkAsCompleted(ctx context.Context, id uuid.UUID) error
	MarkAsFailed(ctx context.Context, id uuid.UUID, errorMessage string) error
	GetJobStatistics(ctx context.Context, filters map[string]interface{}) (*ScanJobStatistics, error)
	GetRecentJobs(ctx context.Context, limit int) ([]*types.ScanJob, error)
	CleanupOldJobs(ctx context.Context, olderThan time.Time) (int, error)
}

// FindingRepository defines finding-specific operations
type FindingRepository interface {
	BaseRepository[types.Finding, uuid.UUID]
	RepositoryWithAudit[types.Finding, uuid.UUID]
	RepositoryWithAdvancedQuery[types.Finding, uuid.UUID]
	
	// Finding-specific methods
	GetByScanJob(ctx context.Context, scanJobID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error)
	GetBySeverity(ctx context.Context, severity string, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error)
	GetByAgent(ctx context.Context, agentName string, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error)
	GetByRepository(ctx context.Context, repoID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	SuppressFinding(ctx context.Context, id uuid.UUID, reason string, userID uuid.UUID) error
	UnsuppressFinding(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	GetStatistics(ctx context.Context, filters map[string]interface{}) (*FindingStatistics, error)
	BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status string, userID uuid.UUID) error
	GetSimilarFindings(ctx context.Context, findingID uuid.UUID, threshold float64) ([]*types.Finding, error)
	GetFindingTrends(ctx context.Context, days int, filters map[string]interface{}) ([]*FindingTrend, error)
	SearchFindings(ctx context.Context, query string, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error)
}

// ScanResultRepository defines scan result-specific operations
type ScanResultRepository interface {
	BaseRepository[types.ScanResult, uuid.UUID]
	RepositoryWithMetrics[types.ScanResult, uuid.UUID]
	
	// Scan result-specific methods
	GetByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*types.ScanResult, error)
	GetByAgent(ctx context.Context, agentName string, limit, offset int) ([]*types.ScanResult, int, error)
	GetByStatus(ctx context.Context, status string) ([]*types.ScanResult, error)
	GetAgentStatistics(ctx context.Context, agentName string, days int) (*AgentStatistics, error)
	GetPerformanceMetrics(ctx context.Context, filters map[string]interface{}) (*PerformanceMetrics, error)
	CleanupOldResults(ctx context.Context, olderThan time.Time) (int, error)
}

// UserFeedbackRepository defines user feedback-specific operations
type UserFeedbackRepository interface {
	BaseRepository[types.UserFeedback, uuid.UUID]
	RepositoryWithAudit[types.UserFeedback, uuid.UUID]
	
	// User feedback-specific methods
	GetByFinding(ctx context.Context, findingID uuid.UUID) ([]*types.UserFeedback, error)
	GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*types.UserFeedback, int, error)
	GetByAction(ctx context.Context, action string, limit, offset int) ([]*types.UserFeedback, int, error)
	GetFeedbackStatistics(ctx context.Context, filters map[string]interface{}) (*FeedbackStatistics, error)
	GetUserFeedbackSummary(ctx context.Context, userID uuid.UUID) (*UserFeedbackSummary, error)
}

// Statistics and aggregation types

// RepositoryStatistics represents repository statistics
type RepositoryStatistics struct {
	TotalScans       int                    `json:"total_scans"`
	CompletedScans   int                    `json:"completed_scans"`
	FailedScans      int                    `json:"failed_scans"`
	AverageDuration  float64                `json:"average_duration_seconds"`
	TotalFindings    int                    `json:"total_findings"`
	FindingsBySeverity map[string]int       `json:"findings_by_severity"`
	FindingsByStatus   map[string]int       `json:"findings_by_status"`
	HealthScore      float64                `json:"health_score"`
	LastScanAt       *time.Time             `json:"last_scan_at"`
	Trends           []StatisticTrend       `json:"trends"`
}

// ScanJobStatistics represents scan job statistics
type ScanJobStatistics struct {
	TotalJobs        int                    `json:"total_jobs"`
	JobsByStatus     map[string]int         `json:"jobs_by_status"`
	JobsByType       map[string]int         `json:"jobs_by_type"`
	AverageDuration  float64                `json:"average_duration_seconds"`
	SuccessRate      float64                `json:"success_rate"`
	QueueLength      int                    `json:"queue_length"`
	ProcessingRate   float64                `json:"processing_rate_per_hour"`
	Trends           []StatisticTrend       `json:"trends"`
}

// FindingStatistics represents finding statistics
type FindingStatistics struct {
	Total        int                    `json:"total"`
	BySeverity   map[string]int         `json:"by_severity"`
	ByStatus     map[string]int         `json:"by_status"`
	ByCategory   map[string]int         `json:"by_category"`
	ByTool       map[string]int         `json:"by_tool"`
	Trends       []FindingTrend         `json:"trends"`
	TopFiles     []FileStatistic        `json:"top_files"`
	Resolution   ResolutionStatistics   `json:"resolution"`
}

// FindingTrend represents finding trends over time
type FindingTrend struct {
	Date     time.Time          `json:"date"`
	Severity string             `json:"severity"`
	Count    int                `json:"count"`
	Status   map[string]int     `json:"status"`
}

// AgentStatistics represents agent performance statistics
type AgentStatistics struct {
	AgentName       string             `json:"agent_name"`
	TotalRuns       int                `json:"total_runs"`
	SuccessfulRuns  int                `json:"successful_runs"`
	FailedRuns      int                `json:"failed_runs"`
	SuccessRate     float64            `json:"success_rate"`
	AverageDuration float64            `json:"average_duration_ms"`
	MedianDuration  float64            `json:"median_duration_ms"`
	P95Duration     float64            `json:"p95_duration_ms"`
	TotalFindings   int                `json:"total_findings"`
	AverageFindings float64            `json:"average_findings_per_run"`
	FirstRunAt      time.Time          `json:"first_run_at"`
	LastRunAt       time.Time          `json:"last_run_at"`
	Trends          []StatisticTrend   `json:"trends"`
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	TotalOperations    int64         `json:"total_operations"`
	AverageLatency     time.Duration `json:"average_latency"`
	P50Latency         time.Duration `json:"p50_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`
	ErrorRate          float64       `json:"error_rate"`
	ThroughputPerSecond float64      `json:"throughput_per_second"`
	ConcurrentOperations int         `json:"concurrent_operations"`
}

// FeedbackStatistics represents user feedback statistics
type FeedbackStatistics struct {
	TotalFeedback    int                    `json:"total_feedback"`
	FeedbackByAction map[string]int         `json:"feedback_by_action"`
	FeedbackByUser   map[string]int         `json:"feedback_by_user"`
	AveragePerUser   float64                `json:"average_per_user"`
	Trends           []StatisticTrend       `json:"trends"`
}

// UserFeedbackSummary represents a user's feedback summary
type UserFeedbackSummary struct {
	UserID          uuid.UUID          `json:"user_id"`
	TotalFeedback   int                `json:"total_feedback"`
	FeedbackByAction map[string]int    `json:"feedback_by_action"`
	LastFeedbackAt  *time.Time         `json:"last_feedback_at"`
	Trends          []StatisticTrend   `json:"trends"`
}

// ResolutionStatistics represents finding resolution statistics
type ResolutionStatistics struct {
	AverageTimeToResolve time.Duration      `json:"average_time_to_resolve"`
	MedianTimeToResolve  time.Duration      `json:"median_time_to_resolve"`
	ResolutionRate       float64            `json:"resolution_rate"`
	ResolutionByMethod   map[string]int     `json:"resolution_by_method"`
}

// StatisticTrend represents a trend data point
type StatisticTrend struct {
	Date  time.Time          `json:"date"`
	Value float64            `json:"value"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// FileStatistic represents file-level statistics
type FileStatistic struct {
	FilePath string `json:"file_path"`
	Count    int    `json:"count"`
	Severity map[string]int `json:"severity"`
}

// Repositories aggregator interface
type Repositories interface {
	// Core repositories
	Users() UserRepository
	Organizations() OrganizationRepository
	Repositories() RepositoryRepository
	ScanJobs() ScanJobRepository
	Findings() FindingRepository
	ScanResults() ScanResultRepository
	UserFeedback() UserFeedbackRepository
	
	// Transaction support
	WithTransaction(ctx context.Context, fn func(Repositories) error) error
	
	// Health and maintenance
	HealthCheck(ctx context.Context) error
	GetMetrics(ctx context.Context) (*RepositoryMetrics, error)
	
	// Cleanup operations
	CleanupOldData(ctx context.Context, config *CleanupConfig) (*CleanupResult, error)
	
	// Close resources
	Close() error
}

// CleanupConfig configures data cleanup operations
type CleanupConfig struct {
	ScanJobsOlderThan    time.Duration `json:"scan_jobs_older_than"`
	ScanResultsOlderThan time.Duration `json:"scan_results_older_than"`
	AuditLogsOlderThan   time.Duration `json:"audit_logs_older_than"`
	DryRun               bool          `json:"dry_run"`
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	ScanJobsDeleted    int `json:"scan_jobs_deleted"`
	ScanResultsDeleted int `json:"scan_results_deleted"`
	AuditLogsDeleted   int `json:"audit_logs_deleted"`
	TotalDeleted       int `json:"total_deleted"`
	Duration           time.Duration `json:"duration"`
}