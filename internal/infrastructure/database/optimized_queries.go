package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/types"
)

// OptimizedQueries provides optimized database queries to resolve N+1 problems
type OptimizedQueries struct {
	db *sqlx.DB
}

// NewOptimizedQueries creates a new optimized queries instance
func NewOptimizedQueries(db *sqlx.DB) *OptimizedQueries {
	return &OptimizedQueries{
		db: db,
	}
}

// ScanJobWithDetails represents a scan job with all related data loaded
type ScanJobWithDetails struct {
	types.ScanJob
	Repository    *types.Repository    `json:"repository"`
	User          *types.User          `json:"user,omitempty"`
	ScanResults   []*types.ScanResult  `json:"scan_results"`
	FindingsCount int                  `json:"findings_count"`
	FindingStats  *FindingStatistics   `json:"finding_stats"`
}

// FindingStatistics represents aggregated finding statistics
type FindingStatistics struct {
	Total        int            `json:"total"`
	BySeverity   map[string]int `json:"by_severity"`
	ByStatus     map[string]int `json:"by_status"`
	ByCategory   map[string]int `json:"by_category"`
	ByTool       map[string]int `json:"by_tool"`
}

// RepositoryWithStats represents a repository with scan statistics
type RepositoryWithStats struct {
	types.Repository
	ScanCount       int                `json:"scan_count"`
	LastScanJob     *types.ScanJob     `json:"last_scan_job,omitempty"`
	FindingsCount   int                `json:"findings_count"`
	FindingStats    *FindingStatistics `json:"finding_stats"`
	HealthScore     float64            `json:"health_score"`
}

// DashboardStatistics represents dashboard statistics
type DashboardStatistics struct {
	TotalRepositories   int                          `json:"total_repositories"`
	TotalScans         int                          `json:"total_scans"`
	CompletedScans     int                          `json:"completed_scans"`
	FailedScans        int                          `json:"failed_scans"`
	TotalFindings      int                          `json:"total_findings"`
	FindingsBySeverity map[string]int               `json:"findings_by_severity"`
	RecentScans        []*ScanJobWithDetails        `json:"recent_scans"`
	TopRepositories    []*RepositoryWithStats       `json:"top_repositories"`
	ScanTrends         []DailyStatistic             `json:"scan_trends"`
	FindingTrends      []DailyStatistic             `json:"finding_trends"`
}

// DailyStatistic represents daily statistics
type DailyStatistic struct {
	Date  string         `json:"date"`
	Stats map[string]int `json:"stats"`
}

// ListScansWithDetails retrieves scan jobs with all related data in a single query
func (oq *OptimizedQueries) ListScansWithDetails(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*ScanJobWithDetails, int, error) {
	// Build WHERE clause from filters
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	for field, value := range filters {
		switch field {
		case "repository_id":
			whereClauses = append(whereClauses, fmt.Sprintf("sj.repository_id = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case "user_id":
			whereClauses = append(whereClauses, fmt.Sprintf("sj.user_id = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case "status":
			whereClauses = append(whereClauses, fmt.Sprintf("sj.status = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case "organization_id":
			whereClauses = append(whereClauses, fmt.Sprintf("r.organization_id = $%d", argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Main query with JOINs to avoid N+1 problems
	query := fmt.Sprintf(`
		SELECT 
			-- Scan job fields
			sj.id, sj.repository_id, sj.user_id, sj.branch, sj.commit_sha,
			sj.scan_type, sj.priority, sj.status, sj.agents_requested, sj.agents_completed,
			sj.started_at, sj.completed_at, sj.error_message, sj.metadata,
			sj.created_at, sj.updated_at,
			
			-- Repository fields
			r.id as repo_id, r.organization_id, r.name as repo_name, r.url as repo_url,
			r.provider, r.provider_id, r.default_branch, r.languages as repo_languages,
			r.settings as repo_settings, r.last_scan_at, r.created_at as repo_created_at,
			r.updated_at as repo_updated_at,
			
			-- User fields (nullable)
			u.id as user_id, u.email, u.name as user_name, u.avatar_url,
			u.github_id, u.gitlab_id, u.created_at as user_created_at,
			u.updated_at as user_updated_at,
			
			-- Aggregated statistics
			COALESCE(f_stats.total_findings, 0) as findings_count,
			COALESCE(f_stats.high_count, 0) as high_findings,
			COALESCE(f_stats.medium_count, 0) as medium_findings,
			COALESCE(f_stats.low_count, 0) as low_findings,
			COALESCE(f_stats.info_count, 0) as info_findings,
			COALESCE(f_stats.open_count, 0) as open_findings,
			COALESCE(f_stats.fixed_count, 0) as fixed_findings,
			COALESCE(sr_stats.results_count, 0) as results_count
			
		FROM scan_jobs sj
		INNER JOIN repositories r ON sj.repository_id = r.id
		LEFT JOIN users u ON sj.user_id = u.id
		LEFT JOIN (
			SELECT 
				scan_job_id,
				COUNT(*) as total_findings,
				COUNT(CASE WHEN severity = 'high' THEN 1 END) as high_count,
				COUNT(CASE WHEN severity = 'medium' THEN 1 END) as medium_count,
				COUNT(CASE WHEN severity = 'low' THEN 1 END) as low_count,
				COUNT(CASE WHEN severity = 'info' THEN 1 END) as info_count,
				COUNT(CASE WHEN status = 'open' THEN 1 END) as open_count,
				COUNT(CASE WHEN status = 'fixed' THEN 1 END) as fixed_count
			FROM findings
			GROUP BY scan_job_id
		) f_stats ON sj.id = f_stats.scan_job_id
		LEFT JOIN (
			SELECT 
				scan_job_id,
				COUNT(*) as results_count
			FROM scan_results
			GROUP BY scan_job_id
		) sr_stats ON sj.id = sr_stats.scan_job_id
		WHERE %s
		ORDER BY sj.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	// Execute query
	rows, err := oq.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.NewDatabaseError("list_scans_with_details", "failed to execute query").WithCause(err)
	}
	defer rows.Close()

	var scans []*ScanJobWithDetails
	for rows.Next() {
		scan := &ScanJobWithDetails{}
		repo := &types.Repository{}
		var user *types.User

		var userID, userEmail, userName, userAvatarURL, userCreatedAt, userUpdatedAt interface{}
		var userGithubID, userGitlabID interface{}
		var findingsCount, highFindings, mediumFindings, lowFindings, infoFindings int
		var openFindings, fixedFindings, resultsCount int

		err := rows.Scan(
			// Scan job fields
			&scan.ID, &scan.RepositoryID, &scan.UserID, &scan.Branch, &scan.CommitSHA,
			&scan.ScanType, &scan.Priority, &scan.Status, &scan.AgentsRequested, &scan.AgentsCompleted,
			&scan.StartedAt, &scan.CompletedAt, &scan.ErrorMessage, &scan.Metadata,
			&scan.CreatedAt, &scan.UpdatedAt,
			
			// Repository fields
			&repo.ID, &repo.OrganizationID, &repo.Name, &repo.URL,
			&repo.Provider, &repo.ProviderID, &repo.DefaultBranch, &repo.Languages,
			&repo.Settings, &repo.LastScanAt, &repo.CreatedAt, &repo.UpdatedAt,
			
			// User fields (nullable)
			&userID, &userEmail, &userName, &userAvatarURL,
			&userGithubID, &userGitlabID, &userCreatedAt, &userUpdatedAt,
			
			// Statistics
			&findingsCount, &highFindings, &mediumFindings, &lowFindings, &infoFindings,
			&openFindings, &fixedFindings, &resultsCount,
		)
		if err != nil {
			return nil, 0, errors.NewDatabaseError("list_scans_with_details", "failed to scan row").WithCause(err)
		}

		// Build user object if data exists
		if userID != nil {
			user = &types.User{}
			if uid, ok := userID.(uuid.UUID); ok {
				user.ID = uid
			}
			if email, ok := userEmail.(string); ok {
				user.Email = email
			}
			if name, ok := userName.(string); ok {
				user.Name = name
			}
			// Set other user fields...
		}

		// Build finding statistics
		scan.Repository = repo
		scan.User = user
		scan.FindingsCount = findingsCount
		scan.FindingStats = &FindingStatistics{
			Total: findingsCount,
			BySeverity: map[string]int{
				"high":   highFindings,
				"medium": mediumFindings,
				"low":    lowFindings,
				"info":   infoFindings,
			},
			ByStatus: map[string]int{
				"open":  openFindings,
				"fixed": fixedFindings,
			},
		}

		scans = append(scans, scan)
	}

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM scan_jobs sj
		INNER JOIN repositories r ON sj.repository_id = r.id
		WHERE %s
	`, whereClause)

	var total int
	err = oq.db.GetContext(ctx, &total, countQuery, args[:len(args)-2]...)
	if err != nil {
		return nil, 0, errors.NewDatabaseError("list_scans_with_details", "failed to get count").WithCause(err)
	}

	return scans, total, nil
}

// GetRepositoriesWithStats retrieves repositories with scan statistics
func (oq *OptimizedQueries) GetRepositoriesWithStats(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*RepositoryWithStats, int, error) {
	query := `
		SELECT 
			-- Repository fields
			r.id, r.organization_id, r.name, r.url, r.provider, r.provider_id,
			r.default_branch, r.languages, r.settings, r.last_scan_at,
			r.created_at, r.updated_at,
			
			-- Statistics
			COALESCE(scan_stats.scan_count, 0) as scan_count,
			COALESCE(scan_stats.completed_scans, 0) as completed_scans,
			COALESCE(scan_stats.failed_scans, 0) as failed_scans,
			COALESCE(finding_stats.total_findings, 0) as total_findings,
			COALESCE(finding_stats.high_findings, 0) as high_findings,
			COALESCE(finding_stats.open_findings, 0) as open_findings,
			
			-- Last scan job info
			last_scan.id as last_scan_id,
			last_scan.status as last_scan_status,
			last_scan.created_at as last_scan_created_at,
			
			-- Health score calculation
			CASE 
				WHEN COALESCE(finding_stats.high_findings, 0) = 0 AND COALESCE(finding_stats.open_findings, 0) <= 5 THEN 95.0
				WHEN COALESCE(finding_stats.high_findings, 0) <= 2 AND COALESCE(finding_stats.open_findings, 0) <= 10 THEN 85.0
				WHEN COALESCE(finding_stats.high_findings, 0) <= 5 AND COALESCE(finding_stats.open_findings, 0) <= 20 THEN 75.0
				WHEN COALESCE(finding_stats.high_findings, 0) <= 10 AND COALESCE(finding_stats.open_findings, 0) <= 50 THEN 65.0
				ELSE 50.0
			END as health_score
			
		FROM repositories r
		LEFT JOIN (
			SELECT 
				repository_id,
				COUNT(*) as scan_count,
				COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_scans,
				COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_scans
			FROM scan_jobs
			WHERE created_at >= NOW() - INTERVAL '30 days'
			GROUP BY repository_id
		) scan_stats ON r.id = scan_stats.repository_id
		LEFT JOIN (
			SELECT 
				sj.repository_id,
				COUNT(f.id) as total_findings,
				COUNT(CASE WHEN f.severity = 'high' THEN 1 END) as high_findings,
				COUNT(CASE WHEN f.status = 'open' THEN 1 END) as open_findings
			FROM scan_jobs sj
			INNER JOIN findings f ON sj.id = f.scan_job_id
			WHERE sj.created_at >= NOW() - INTERVAL '30 days'
			GROUP BY sj.repository_id
		) finding_stats ON r.id = finding_stats.repository_id
		LEFT JOIN LATERAL (
			SELECT id, status, created_at
			FROM scan_jobs
			WHERE repository_id = r.id
			ORDER BY created_at DESC
			LIMIT 1
		) last_scan ON true
		WHERE r.organization_id = $1
		ORDER BY health_score DESC, r.last_scan_at DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`

	rows, err := oq.db.QueryContext(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, errors.NewDatabaseError("get_repositories_with_stats", "failed to execute query").WithCause(err)
	}
	defer rows.Close()

	var repositories []*RepositoryWithStats
	for rows.Next() {
		repo := &RepositoryWithStats{}
		var lastScanID, lastScanStatus, lastScanCreatedAt interface{}
		var totalFindings, highFindings, openFindings int
		var scanCount, completedScans, failedScans int

		err := rows.Scan(
			// Repository fields
			&repo.ID, &repo.OrganizationID, &repo.Name, &repo.URL, &repo.Provider, &repo.ProviderID,
			&repo.DefaultBranch, &repo.Languages, &repo.Settings, &repo.LastScanAt,
			&repo.CreatedAt, &repo.UpdatedAt,
			
			// Statistics
			&scanCount, &completedScans, &failedScans, &totalFindings, &highFindings, &openFindings,
			
			// Last scan
			&lastScanID, &lastScanStatus, &lastScanCreatedAt,
			
			// Health score
			&repo.HealthScore,
		)
		if err != nil {
			return nil, 0, errors.NewDatabaseError("get_repositories_with_stats", "failed to scan row").WithCause(err)
		}

		repo.ScanCount = scanCount
		repo.FindingsCount = totalFindings
		repo.FindingStats = &FindingStatistics{
			Total: totalFindings,
			BySeverity: map[string]int{
				"high": highFindings,
			},
			ByStatus: map[string]int{
				"open": openFindings,
			},
		}

		// Build last scan job if exists
		if lastScanID != nil {
			repo.LastScanJob = &types.ScanJob{}
			if id, ok := lastScanID.(uuid.UUID); ok {
				repo.LastScanJob.ID = id
			}
			if status, ok := lastScanStatus.(string); ok {
				repo.LastScanJob.Status = status
			}
			// Set other fields...
		}

		repositories = append(repositories, repo)
	}

	// Get total count
	var total int
	err = oq.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM repositories WHERE organization_id = $1", orgID)
	if err != nil {
		return nil, 0, errors.NewDatabaseError("get_repositories_with_stats", "failed to get count").WithCause(err)
	}

	return repositories, total, nil
}

// GetDashboardStatistics retrieves comprehensive dashboard statistics in optimized queries
func (oq *OptimizedQueries) GetDashboardStatistics(ctx context.Context, orgID *uuid.UUID) (*DashboardStatistics, error) {
	stats := &DashboardStatistics{
		FindingsBySeverity: make(map[string]int),
	}

	// Use the materialized view if available, otherwise compute on-the-fly
	var baseQuery string
	var args []interface{}

	if orgID != nil {
		baseQuery = `
			SELECT 
				total_repositories,
				total_scans,
				completed_scans,
				failed_scans,
				total_findings,
				high_severity_open,
				medium_severity_open,
				low_severity_open
			FROM dashboard_stats 
			WHERE organization_id = $1
		`
		args = append(args, *orgID)
	} else {
		// Fallback to computed query
		baseQuery = `
			SELECT 
				COUNT(DISTINCT r.id) as total_repositories,
				COUNT(DISTINCT sj.id) as total_scans,
				COUNT(DISTINCT CASE WHEN sj.status = 'completed' THEN sj.id END) as completed_scans,
				COUNT(DISTINCT CASE WHEN sj.status = 'failed' THEN sj.id END) as failed_scans,
				COUNT(DISTINCT f.id) as total_findings,
				COUNT(DISTINCT CASE WHEN f.severity = 'high' AND f.status = 'open' THEN f.id END) as high_severity_open,
				COUNT(DISTINCT CASE WHEN f.severity = 'medium' AND f.status = 'open' THEN f.id END) as medium_severity_open,
				COUNT(DISTINCT CASE WHEN f.severity = 'low' AND f.status = 'open' THEN f.id END) as low_severity_open
			FROM repositories r
			LEFT JOIN scan_jobs sj ON r.id = sj.repository_id AND sj.created_at >= NOW() - INTERVAL '90 days'
			LEFT JOIN findings f ON sj.id = f.scan_job_id
		`
		if orgID != nil {
			baseQuery += " WHERE r.organization_id = $1"
			args = append(args, *orgID)
		}
	}

	var highOpen, mediumOpen, lowOpen int
	err := oq.db.QueryRowContext(ctx, baseQuery, args...).Scan(
		&stats.TotalRepositories,
		&stats.TotalScans,
		&stats.CompletedScans,
		&stats.FailedScans,
		&stats.TotalFindings,
		&highOpen,
		&mediumOpen,
		&lowOpen,
	)
	if err != nil {
		return nil, errors.NewDatabaseError("get_dashboard_statistics", "failed to get basic stats").WithCause(err)
	}

	stats.FindingsBySeverity = map[string]int{
		"high":   highOpen,
		"medium": mediumOpen,
		"low":    lowOpen,
	}

	// Get recent scans with details (limit N+1 queries)
	recentScans, _, err := oq.ListScansWithDetails(ctx, map[string]interface{}{}, 10, 0)
	if err != nil {
		return nil, err
	}
	stats.RecentScans = recentScans

	// Get top repositories with stats
	if orgID != nil {
		topRepos, _, err := oq.GetRepositoriesWithStats(ctx, *orgID, 5, 0)
		if err != nil {
			return nil, err
		}
		stats.TopRepositories = topRepos
	}

	return stats, nil
}

// GetScanTrends retrieves scan trends for the last N days
func (oq *OptimizedQueries) GetScanTrends(ctx context.Context, orgID *uuid.UUID, days int) ([]DailyStatistic, error) {
	query := `
		SELECT 
			DATE(sj.created_at) as date,
			COUNT(*) as total_scans,
			COUNT(CASE WHEN sj.status = 'completed' THEN 1 END) as completed_scans,
			COUNT(CASE WHEN sj.status = 'failed' THEN 1 END) as failed_scans,
			COUNT(CASE WHEN sj.status = 'running' THEN 1 END) as running_scans
		FROM scan_jobs sj
		INNER JOIN repositories r ON sj.repository_id = r.id
		WHERE sj.created_at >= NOW() - INTERVAL '%d days'
	`

	args := []interface{}{}
	if orgID != nil {
		query += " AND r.organization_id = $1"
		args = append(args, *orgID)
	}

	query += `
		GROUP BY DATE(sj.created_at)
		ORDER BY date DESC
		LIMIT $%d
	`

	if orgID != nil {
		query = fmt.Sprintf(query, days, 2)
		args = append(args, days)
	} else {
		query = fmt.Sprintf(query, days, 1)
		args = append(args, days)
	}

	rows, err := oq.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewDatabaseError("get_scan_trends", "failed to execute query").WithCause(err)
	}
	defer rows.Close()

	var trends []DailyStatistic
	for rows.Next() {
		var date string
		var totalScans, completedScans, failedScans, runningScans int

		err := rows.Scan(&date, &totalScans, &completedScans, &failedScans, &runningScans)
		if err != nil {
			return nil, errors.NewDatabaseError("get_scan_trends", "failed to scan row").WithCause(err)
		}

		trends = append(trends, DailyStatistic{
			Date: date,
			Stats: map[string]int{
				"total":     totalScans,
				"completed": completedScans,
				"failed":    failedScans,
				"running":   runningScans,
			},
		})
	}

	return trends, nil
}