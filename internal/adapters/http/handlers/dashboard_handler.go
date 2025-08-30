package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/internal/application/services"
	"github.com/agentscan/agentscan/internal/domain/repositories"
)

// DashboardHandler handles dashboard-related HTTP requests
type DashboardHandler struct {
	*BaseHandler
	dashboardService *services.DashboardService
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(repos repositories.Repositories, dashboardService *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		BaseHandler:      NewBaseHandler(repos),
		dashboardService: dashboardService,
	}
}

// GetStats retrieves dashboard statistics
func (h *DashboardHandler) GetStats(c *gin.Context) {
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get organization context if available
	var orgID *uuid.UUID
	if oid, err := h.GetOrganizationContext(c); err == nil {
		orgID = &oid
	}

	stats, err := h.dashboardService.GetStats(c.Request.Context(), userID, orgID)
	if err != nil {
		h.LogError(c, err, "Failed to get dashboard stats", map[string]interface{}{
			"user_id": userID,
			"org_id":  orgID,
		})
		h.Error(c, err)
		return
	}

	h.Success(c, stats)
}

// GetRecentScans retrieves recent scans for dashboard
func (h *DashboardHandler) GetRecentScans(c *gin.Context) {
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	limit, err := h.GetIntQuery(c, "limit", 10)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get organization context if available
	var orgID *uuid.UUID
	if oid, err := h.GetOrganizationContext(c); err == nil {
		orgID = &oid
	}

	scans, err := h.dashboardService.GetRecentScans(c.Request.Context(), userID, orgID, limit)
	if err != nil {
		h.LogError(c, err, "Failed to get recent scans", map[string]interface{}{
			"user_id": userID,
			"org_id":  orgID,
			"limit":   limit,
		})
		h.Error(c, err)
		return
	}

	h.Success(c, scans)
}

// GetScanTrends retrieves scan trends for dashboard charts
func (h *DashboardHandler) GetScanTrends(c *gin.Context) {
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	days, err := h.GetIntQuery(c, "days", 30)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get organization context if available
	var orgID *uuid.UUID
	if oid, err := h.GetOrganizationContext(c); err == nil {
		orgID = &oid
	}

	trends, err := h.dashboardService.GetScanTrends(c.Request.Context(), userID, orgID, days)
	if err != nil {
		h.LogError(c, err, "Failed to get scan trends", map[string]interface{}{
			"user_id": userID,
			"org_id":  orgID,
			"days":    days,
		})
		h.Error(c, err)
		return
	}

	h.Success(c, trends)
}

// GetFindingsSummary retrieves findings summary for dashboard
func (h *DashboardHandler) GetFindingsSummary(c *gin.Context) {
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get organization context if available
	var orgID *uuid.UUID
	if oid, err := h.GetOrganizationContext(c); err == nil {
		orgID = &oid
	}

	summary, err := h.dashboardService.GetFindingsSummary(c.Request.Context(), userID, orgID)
	if err != nil {
		h.LogError(c, err, "Failed to get findings summary", map[string]interface{}{
			"user_id": userID,
			"org_id":  orgID,
		})
		h.Error(c, err)
		return
	}

	h.Success(c, summary)
}

// GetTopVulnerabilities retrieves top vulnerabilities for dashboard
func (h *DashboardHandler) GetTopVulnerabilities(c *gin.Context) {
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	limit, err := h.GetIntQuery(c, "limit", 10)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get organization context if available
	var orgID *uuid.UUID
	if oid, err := h.GetOrganizationContext(c); err == nil {
		orgID = &oid
	}

	vulnerabilities, err := h.dashboardService.GetTopVulnerabilities(c.Request.Context(), userID, orgID, limit)
	if err != nil {
		h.LogError(c, err, "Failed to get top vulnerabilities", map[string]interface{}{
			"user_id": userID,
			"org_id":  orgID,
			"limit":   limit,
		})
		h.Error(c, err)
		return
	}

	h.Success(c, vulnerabilities)
}

// GetRepositoryHealth retrieves repository health metrics
func (h *DashboardHandler) GetRepositoryHealth(c *gin.Context) {
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get organization context if available
	var orgID *uuid.UUID
	if oid, err := h.GetOrganizationContext(c); err == nil {
		orgID = &oid
	}

	health, err := h.dashboardService.GetRepositoryHealth(c.Request.Context(), userID, orgID)
	if err != nil {
		h.LogError(c, err, "Failed to get repository health", map[string]interface{}{
			"user_id": userID,
			"org_id":  orgID,
		})
		h.Error(c, err)
		return
	}

	h.Success(c, health)
}

// GetScanQueue retrieves current scan queue status
func (h *DashboardHandler) GetScanQueue(c *gin.Context) {
	// Check admin permissions for queue visibility
	if err := h.RequireRole(c, "admin"); err != nil {
		h.Error(c, err)
		return
	}

	queue, err := h.dashboardService.GetScanQueue(c.Request.Context())
	if err != nil {
		h.LogError(c, err, "Failed to get scan queue", nil)
		h.Error(c, err)
		return
	}

	h.Success(c, queue)
}

// GetSystemHealth retrieves system health metrics
func (h *DashboardHandler) GetSystemHealth(c *gin.Context) {
	// Check admin permissions for system health
	if err := h.RequireRole(c, "admin"); err != nil {
		h.Error(c, err)
		return
	}

	health, err := h.dashboardService.GetSystemHealth(c.Request.Context())
	if err != nil {
		h.LogError(c, err, "Failed to get system health", nil)
		h.Error(c, err)
		return
	}

	h.Success(c, health)
}