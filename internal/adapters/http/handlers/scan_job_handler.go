package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/internal/api"
	"github.com/agentscan/agentscan/internal/application/dto"
	"github.com/agentscan/agentscan/internal/application/services"
	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/pkg/types"
)

// ScanJobHandler handles scan job-related HTTP requests
type ScanJobHandler struct {
	*BaseHandler
	scanJobService *services.ScanJobService
}

// NewScanJobHandler creates a new scan job handler
func NewScanJobHandler(repos repositories.Repositories, scanJobService *services.ScanJobService) *ScanJobHandler {
	return &ScanJobHandler{
		BaseHandler:    NewBaseHandler(repos),
		scanJobService: scanJobService,
	}
}

// CreateScanJob creates a new scan job
func (h *ScanJobHandler) CreateScanJob(c *gin.Context) {
	var request dto.CreateScanJobRequest

	h.HandleCreate[types.ScanJob](
		c,
		&request,
		func(ctx context.Context, req interface{}) (*types.ScanJob, error) {
			createReq := req.(*dto.CreateScanJobRequest)
			return h.scanJobService.CreateScanJob(ctx, createReq)
		},
		"scan_job",
	)
}

// GetScanJob retrieves a scan job by ID
func (h *ScanJobHandler) GetScanJob(c *gin.Context) {
	h.HandleGet[types.ScanJob](
		c,
		func(ctx context.Context, id uuid.UUID) (*types.ScanJob, error) {
			return h.scanJobService.GetScanJob(ctx, id)
		},
		"scan_job",
	)
}

// ListScanJobs lists scan jobs with filtering and pagination
func (h *ScanJobHandler) ListScanJobs(c *gin.Context) {
	h.HandleList[types.ScanJob](
		c,
		func(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*types.ScanJob, int, error) {
			return h.scanJobService.ListScanJobs(ctx, filters, limit, offset)
		},
		h.buildScanJobFilters,
		"scan_job",
	)
}

// UpdateScanJob updates a scan job
func (h *ScanJobHandler) UpdateScanJob(c *gin.Context) {
	var request dto.UpdateScanJobRequest

	h.HandleUpdate[types.ScanJob](
		c,
		&request,
		func(ctx context.Context, id uuid.UUID, req interface{}) (*types.ScanJob, error) {
			updateReq := req.(*dto.UpdateScanJobRequest)
			return h.scanJobService.UpdateScanJob(ctx, id, updateReq)
		},
		"scan_job",
	)
}

// DeleteScanJob deletes a scan job
func (h *ScanJobHandler) DeleteScanJob(c *gin.Context) {
	h.HandleDelete(
		c,
		func(ctx context.Context, id uuid.UUID) error {
			return h.scanJobService.DeleteScanJob(ctx, id)
		},
		"scan_job",
	)
}

// GetRepositoryScanJobs retrieves scan jobs for a repository
func (h *ScanJobHandler) GetRepositoryScanJobs(c *gin.Context) {
	repoID, err := h.GetUUIDParam(c, "repoId")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get pagination parameters
	limit, offset, err := h.GetPaginationParams(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Build filters
	filters := h.buildScanJobFilters(c)
	filters["repository_id"] = repoID

	scanJobs, total, err := h.scanJobService.ListScanJobs(c.Request.Context(), filters, limit, offset)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Calculate pagination metadata
	page := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit

	meta := &api.Meta{
		Pagination: &api.Pagination{
			Page:       page,
			PageSize:   limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	}

	h.SuccessWithMeta(c, scanJobs, meta)
}

// GetUserScanJobs retrieves scan jobs for a user
func (h *ScanJobHandler) GetUserScanJobs(c *gin.Context) {
	userID, err := h.GetUUIDParam(c, "userId")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Check if user can access these scan jobs
	currentUserID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Users can only see their own scan jobs unless they're admin
	if currentUserID != userID {
		if err := h.RequireRole(c, "admin"); err != nil {
			h.Error(c, err)
			return
		}
	}

	// Get pagination parameters
	limit, offset, err := h.GetPaginationParams(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Build filters
	filters := h.buildScanJobFilters(c)
	filters["user_id"] = userID

	scanJobs, total, err := h.scanJobService.ListScanJobs(c.Request.Context(), filters, limit, offset)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Calculate pagination metadata
	page := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit

	meta := &api.Meta{
		Pagination: &api.Pagination{
			Page:       page,
			PageSize:   limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	}

	h.SuccessWithMeta(c, scanJobs, meta)
}

// GetScanJobsByStatus retrieves scan jobs by status
func (h *ScanJobHandler) GetScanJobsByStatus(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		h.BadRequest(c, "Status parameter is required")
		return
	}

	scanJobs, err := h.scanJobService.GetScanJobsByStatus(c.Request.Context(), status)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, scanJobs)
}

// StartScanJob starts a scan job
func (h *ScanJobHandler) StartScanJob(c *gin.Context) {
	id, err := h.GetUUIDParam(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Check permissions
	if err := h.RequireRole(c, "user"); err != nil {
		h.Error(c, err)
		return
	}

	err = h.scanJobService.StartScanJob(c.Request.Context(), id)
	if err != nil {
		h.LogError(c, err, "Failed to start scan job", map[string]interface{}{
			"scan_job_id": id,
		})
		h.Error(c, err)
		return
	}

	h.LogAction(c, "start", "scan_job", map[string]interface{}{
		"scan_job_id": id,
	})

	h.Success(c, gin.H{"message": "Scan job started successfully"})
}

// CancelScanJob cancels a scan job
func (h *ScanJobHandler) CancelScanJob(c *gin.Context) {
	id, err := h.GetUUIDParam(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Check permissions
	if err := h.RequireRole(c, "user"); err != nil {
		h.Error(c, err)
		return
	}

	err = h.scanJobService.CancelScanJob(c.Request.Context(), id)
	if err != nil {
		h.LogError(c, err, "Failed to cancel scan job", map[string]interface{}{
			"scan_job_id": id,
		})
		h.Error(c, err)
		return
	}

	h.LogAction(c, "cancel", "scan_job", map[string]interface{}{
		"scan_job_id": id,
	})

	h.Success(c, gin.H{"message": "Scan job cancelled successfully"})
}

// GetRunningJobs retrieves all running scan jobs
func (h *ScanJobHandler) GetRunningJobs(c *gin.Context) {
	// Check permissions - only admins can see all running jobs
	if err := h.RequireRole(c, "admin"); err != nil {
		h.Error(c, err)
		return
	}

	scanJobs, err := h.scanJobService.GetRunningJobs(c.Request.Context())
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, scanJobs)
}

// GetQueuedJobs retrieves queued scan jobs
func (h *ScanJobHandler) GetQueuedJobs(c *gin.Context) {
	// Check permissions - only admins can see queue
	if err := h.RequireRole(c, "admin"); err != nil {
		h.Error(c, err)
		return
	}

	limit, err := h.GetIntQuery(c, "limit", 50)
	if err != nil {
		h.Error(c, err)
		return
	}

	scanJobs, err := h.scanJobService.GetQueuedJobs(c.Request.Context(), limit)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, scanJobs)
}

// GetScanJobStatistics retrieves scan job statistics
func (h *ScanJobHandler) GetScanJobStatistics(c *gin.Context) {
	stats, err := h.scanJobService.GetScanJobStatistics(c.Request.Context())
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, stats)
}

// buildScanJobFilters builds filters from query parameters
func (h *ScanJobHandler) buildScanJobFilters(c *gin.Context) map[string]interface{} {
	filters := make(map[string]interface{})

	// String filters
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	if agentName := c.Query("agent_name"); agentName != "" {
		filters["agent_name"] = agentName
	}

	// UUID filters
	if repoID := c.Query("repository_id"); repoID != "" {
		if parsedUUID, err := uuid.Parse(repoID); err == nil {
			filters["repository_id"] = parsedUUID
		}
	}

	if userID := c.Query("user_id"); userID != "" {
		if parsedUUID, err := uuid.Parse(userID); err == nil {
			filters["user_id"] = parsedUUID
		}
	}

	// Date range filters
	if createdAfter := c.Query("created_after"); createdAfter != "" {
		filters["created_at >="] = createdAfter
	}

	if createdBefore := c.Query("created_before"); createdBefore != "" {
		filters["created_at <="] = createdBefore
	}

	if startedAfter := c.Query("started_after"); startedAfter != "" {
		filters["started_at >="] = startedAfter
	}

	if startedBefore := c.Query("started_before"); startedBefore != "" {
		filters["started_at <="] = startedBefore
	}

	if completedAfter := c.Query("completed_after"); completedAfter != "" {
		filters["completed_at >="] = completedAfter
	}

	if completedBefore := c.Query("completed_before"); completedBefore != "" {
		filters["completed_at <="] = completedBefore
	}

	return filters
}