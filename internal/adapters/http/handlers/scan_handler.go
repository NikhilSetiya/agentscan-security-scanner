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

// ScanHandler handles scan-related HTTP requests
type ScanHandler struct {
	*BaseHandler
	scanService *services.ScanService
}

// NewScanHandler creates a new scan handler
func NewScanHandler(repos repositories.Repositories, scanService *services.ScanService) *ScanHandler {
	return &ScanHandler{
		BaseHandler: NewBaseHandler(repos),
		scanService: scanService,
	}
}

// CreateScan creates a new security scan
func (h *ScanHandler) CreateScan(c *gin.Context) {
	var request dto.CreateScanRequest

	h.HandleCreate[types.ScanJob](
		c,
		&request,
		func(ctx context.Context, req interface{}) (*types.ScanJob, error) {
			createReq := req.(*dto.CreateScanRequest)
			return h.scanService.CreateScan(ctx, createReq)
		},
		"scan",
	)
}

// GetScan retrieves a scan by ID
func (h *ScanHandler) GetScan(c *gin.Context) {
	h.HandleGet[types.ScanJob](
		c,
		func(ctx context.Context, id uuid.UUID) (*types.ScanJob, error) {
			return h.scanService.GetScan(ctx, id)
		},
		"scan",
	)
}

// ListScans lists scans with filtering and pagination
func (h *ScanHandler) ListScans(c *gin.Context) {
	h.HandleList[types.ScanJob](
		c,
		func(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*types.ScanJob, int, error) {
			return h.scanService.ListScans(ctx, filters, limit, offset)
		},
		h.buildScanFilters,
		"scan",
	)
}

// UpdateScan updates a scan
func (h *ScanHandler) UpdateScan(c *gin.Context) {
	var request dto.UpdateScanRequest

	h.HandleUpdate[types.ScanJob](
		c,
		&request,
		func(ctx context.Context, id uuid.UUID, req interface{}) (*types.ScanJob, error) {
			updateReq := req.(*dto.UpdateScanRequest)
			return h.scanService.UpdateScan(ctx, id, updateReq)
		},
		"scan",
	)
}

// DeleteScan deletes a scan
func (h *ScanHandler) DeleteScan(c *gin.Context) {
	h.HandleDelete(
		c,
		func(ctx context.Context, id uuid.UUID) error {
			return h.scanService.DeleteScan(ctx, id)
		},
		"scan",
	)
}

// StartScan starts a scan job
func (h *ScanHandler) StartScan(c *gin.Context) {
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

	err = h.scanService.StartScan(c.Request.Context(), id)
	if err != nil {
		h.LogError(c, err, "Failed to start scan", map[string]interface{}{
			"scan_id": id,
		})
		h.Error(c, err)
		return
	}

	h.LogAction(c, "start", "scan", map[string]interface{}{
		"scan_id": id,
	})

	h.Success(c, gin.H{"message": "Scan started successfully"})
}

// StopScan stops a running scan
func (h *ScanHandler) StopScan(c *gin.Context) {
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

	err = h.scanService.StopScan(c.Request.Context(), id)
	if err != nil {
		h.LogError(c, err, "Failed to stop scan", map[string]interface{}{
			"scan_id": id,
		})
		h.Error(c, err)
		return
	}

	h.LogAction(c, "stop", "scan", map[string]interface{}{
		"scan_id": id,
	})

	h.Success(c, gin.H{"message": "Scan stopped successfully"})
}

// GetScanResults retrieves scan results
func (h *ScanHandler) GetScanResults(c *gin.Context) {
	id, err := h.GetUUIDParam(c, "id")
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

	// Build filters for findings
	filters := h.buildFindingFilters(c)
	filters["scan_job_id"] = id

	findings, total, err := h.scanService.GetScanResults(c.Request.Context(), id, filters, limit, offset)
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

	h.SuccessWithMeta(c, findings, meta)
}

// GetScanStatistics retrieves scan statistics
func (h *ScanHandler) GetScanStatistics(c *gin.Context) {
	id, err := h.GetUUIDParam(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	stats, err := h.scanService.GetScanStatistics(c.Request.Context(), id)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, stats)
}

// GetUserScans retrieves scans for the current user
func (h *ScanHandler) GetUserScans(c *gin.Context) {
	userID, _, _, err := h.GetUserContext(c)
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
	filters := h.buildScanFilters(c)
	filters["user_id"] = userID

	scans, total, err := h.scanService.ListScans(c.Request.Context(), filters, limit, offset)
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

	h.SuccessWithMeta(c, scans, meta)
}

// GetRecentScans retrieves recent scans
func (h *ScanHandler) GetRecentScans(c *gin.Context) {
	limit, err := h.GetIntQuery(c, "limit", 10)
	if err != nil {
		h.Error(c, err)
		return
	}

	scans, err := h.scanService.GetRecentScans(c.Request.Context(), limit)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, scans)
}

// buildScanFilters builds filters from query parameters for scans
func (h *ScanHandler) buildScanFilters(c *gin.Context) map[string]interface{} {
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

	return filters
}

// buildFindingFilters builds filters from query parameters for findings
func (h *ScanHandler) buildFindingFilters(c *gin.Context) map[string]interface{} {
	filters := make(map[string]interface{})

	// String filters
	if severity := c.Query("severity"); severity != "" {
		filters["severity"] = severity
	}

	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	if agentName := c.Query("agent_name"); agentName != "" {
		filters["agent_name"] = agentName
	}

	if filePath := c.Query("file_path"); filePath != "" {
		filters["file_path"] = "%" + filePath + "%"
	}

	// Boolean filters
	if suppressed := c.Query("suppressed"); suppressed != "" {
		if suppressed == "true" {
			filters["status"] = "suppressed"
		} else if suppressed == "false" {
			filters["status !="] = "suppressed"
		}
	}

	return filters
}