package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
)

// DashboardHandler handles dashboard-related endpoints
type DashboardHandler struct {
	repos *database.Repositories
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(repos *database.Repositories) *DashboardHandler {
	return &DashboardHandler{
		repos: repos,
	}
}

// GetStats retrieves dashboard statistics
func (h *DashboardHandler) GetStats(c *gin.Context) {
	// Get current user for organization context
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// For now, we'll use nil orgID to get all data
	// In a real implementation, you'd get the user's organization
	var orgID *uuid.UUID = nil

	// Get basic stats
	stats, err := h.repos.Dashboard.GetStats(c.Request.Context(), orgID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Get recent scans
	recentScans, err := h.repos.Dashboard.GetRecentScans(c.Request.Context(), orgID, 10)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Get trend data (last 7 days)
	trendData, err := h.getTrendData(c, orgID, 7)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Combine all data
	dashboardData := map[string]interface{}{
		"total_scans":         stats["total_scans"],
		"total_repositories":  stats["total_repositories"],
		"findings_by_severity": stats["findings_by_severity"],
		"recent_scans":        recentScans,
		"trend_data":          trendData,
	}

	SuccessResponse(c, dashboardData)
}

// getTrendData retrieves trend data for the last N days
func (h *DashboardHandler) getTrendData(c *gin.Context, orgID *uuid.UUID, days int) ([]map[string]interface{}, error) {
	// For now, return mock trend data
	// In a real implementation, you'd query the database for daily statistics
	var trendData []map[string]interface{}
	
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		
		// Mock data - in production, this would come from database queries
		trendData = append(trendData, map[string]interface{}{
			"date":     date.Format("2006-01-02"),
			"critical": 0 + i%3,
			"high":     2 + i%4,
			"medium":   5 + i%3,
			"low":      10 + i%5,
			"info":     1 + i%2,
		})
	}
	
	return trendData, nil
}

// GetRepositoryStats retrieves statistics for a specific repository
func (h *DashboardHandler) GetRepositoryStats(c *gin.Context) {
	repoIDStr := c.Param("id")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		BadRequestResponse(c, "Invalid repository ID")
		return
	}

	// Get current user for authorization
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Get repository to ensure it exists
	repo, err := h.repos.Repositories.GetByID(c.Request.Context(), repoID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Get scan jobs for this repository
	filter := &database.ScanJobFilter{
		RepositoryID: &repoID,
	}
	pagination := &database.Pagination{
		Page:     1,
		PageSize: 100, // Get recent scans
	}

	scanJobs, total, err := h.repos.ScanJobs.List(c.Request.Context(), filter, pagination)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Calculate statistics
	var completedScans, failedScans int64
	var totalFindings int64
	var findingsBySeverity = map[string]int64{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
	}

	for _, job := range scanJobs {
		switch job.Status {
		case "completed":
			completedScans++
		case "failed":
			failedScans++
		}

		// Get findings for this scan job
		findings, err := h.repos.Findings.ListByScanJob(c.Request.Context(), job.ID)
		if err != nil {
			continue // Skip on error, don't fail the whole request
		}

		totalFindings += int64(len(findings))
		for _, finding := range findings {
			if count, exists := findingsBySeverity[finding.Severity]; exists {
				findingsBySeverity[finding.Severity] = count + 1
			}
		}
	}

	stats := map[string]interface{}{
		"repository": map[string]interface{}{
			"id":          repo.ID,
			"name":        repo.Name,
			"url":         repo.URL,
			"language":    repo.Language,
			"description": repo.Description,
			"last_scan_at": repo.LastScanAt,
		},
		"total_scans":        total,
		"completed_scans":    completedScans,
		"failed_scans":       failedScans,
		"total_findings":     totalFindings,
		"findings_by_severity": findingsBySeverity,
		"recent_scans":       len(scanJobs),
	}

	SuccessResponse(c, stats)
}

// GetScanTrends retrieves scan trends over time
func (h *DashboardHandler) GetScanTrends(c *gin.Context) {
	// Get query parameters
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 365 {
		days = 30
	}

	// Get current user for organization context
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// For now, return mock trend data
	// In a real implementation, you'd query the database for historical data
	var orgID *uuid.UUID = nil
	trendData, err := h.getTrendData(c, orgID, days)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	SuccessResponse(c, map[string]interface{}{
		"trends": trendData,
		"period": map[string]interface{}{
			"days":      days,
			"start_date": time.Now().AddDate(0, 0, -days+1).Format("2006-01-02"),
			"end_date":   time.Now().Format("2006-01-02"),
		},
	})
}

// GetSystemHealth retrieves system health metrics
func (h *DashboardHandler) GetSystemHealth(c *gin.Context) {
	// Get current user for authorization
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Check database health
	dbHealthy := true
	if err := h.repos.Users.(*database.UserRepository).Health(c.Request.Context()); err != nil {
		dbHealthy = false
	}

	// Get system metrics
	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"components": map[string]interface{}{
			"database": map[string]interface{}{
				"status": func() string {
					if dbHealthy {
						return "healthy"
					}
					return "unhealthy"
				}(),
				"response_time_ms": 5, // Mock response time
			},
			"api": map[string]interface{}{
				"status": "healthy",
				"uptime_seconds": 3600, // Mock uptime
			},
			"queue": map[string]interface{}{
				"status": "healthy",
				"pending_jobs": 0, // Mock queue size
			},
		},
		"metrics": map[string]interface{}{
			"active_scans": 0,
			"queued_scans": 0,
			"total_users": 1, // Mock user count
		},
	}

	// Set overall status based on components
	if !dbHealthy {
		health["status"] = "degraded"
	}

	SuccessResponse(c, health)
}