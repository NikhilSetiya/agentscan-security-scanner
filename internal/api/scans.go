package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/orchestrator"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/types"
)

// ScanHandler handles scan-related endpoints
type ScanHandler struct {
	repos        *database.Repositories
	orchestrator orchestrator.OrchestrationService
	queue        *queue.Queue
}

// NewScanHandler creates a new scan handler
func NewScanHandler(repos *database.Repositories, orch orchestrator.OrchestrationService, q *queue.Queue) *ScanHandler {
	return &ScanHandler{
		repos:        repos,
		orchestrator: orch,
		queue:        q,
	}
}

// CreateScan creates a new security scan
func (h *ScanHandler) CreateScan(c *gin.Context) {
	var req CreateScanJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestResponse(c, "Invalid request body: "+err.Error())
		return
	}

	userID, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Create orchestrator scan request
	scanReq := &orchestrator.ScanRequest{
		RepositoryID: uuid.New(), // TODO: Get or create repository from URL
		UserID:       &userID,
		RepoURL:      req.RepositoryURL,
		Branch:       req.Branch,
		CommitSHA:    req.CommitSHA,
		ScanType:     req.ScanType,
		Priority:     req.Priority,
		Agents:       req.AgentsRequested,
		Metadata:     make(map[string]interface{}),
	}

	// Set defaults
	if scanReq.Branch == "" {
		scanReq.Branch = "main"
	}
	if scanReq.Priority == 0 {
		scanReq.Priority = types.PriorityMedium
	}
	if len(scanReq.Agents) == 0 {
		scanReq.Agents = []string{"semgrep", "eslint-security"}
	}

	// Submit to orchestrator
	scanJob, err := h.orchestrator.SubmitScan(c.Request.Context(), scanReq)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	CreatedResponse(c, ToScanJobDTO(scanJob))
}

// GetScan retrieves a scan by ID
func (h *ScanHandler) GetScan(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequestResponse(c, "Invalid scan ID")
		return
	}

	scanJob, err := h.repos.ScanJobs.GetByID(c.Request.Context(), scanID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// TODO: Check user permissions for this scan

	SuccessResponse(c, ToScanJobDTO(scanJob))
}

// ListScans lists scans with optional filtering
func (h *ScanHandler) ListScans(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	scanType := c.Query("scan_type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	userID, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Build filter
	filter := &database.ScanJobFilter{
		UserID:   &userID,
		Status:   status,
		ScanType: scanType,
	}

	pagination := &database.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	// Get scans from database
	scanJobs, total, err := h.repos.ScanJobs.List(c.Request.Context(), filter, pagination)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Convert to DTOs
	scanDTOs := make([]*ScanJobDTO, len(scanJobs))
	for i, job := range scanJobs {
		scanDTOs[i] = ToScanJobDTO(job)
	}

	// Calculate pagination metadata
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	meta := &Meta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	SuccessResponseWithMeta(c, scanDTOs, meta)
}

// GetScanStatus retrieves the status of a scan
func (h *ScanHandler) GetScanStatus(c *gin.Context) {
	scanID := c.Param("id")
	if scanID == "" {
		BadRequestResponse(c, "Invalid scan ID")
		return
	}

	status, err := h.orchestrator.GetScanStatus(c.Request.Context(), scanID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	SuccessResponse(c, status)
}

// CancelScan cancels a running scan
func (h *ScanHandler) CancelScan(c *gin.Context) {
	scanID := c.Param("id")
	if scanID == "" {
		BadRequestResponse(c, "Invalid scan ID")
		return
	}

	// TODO: Check user permissions

	// Cancel the scan
	if err := h.orchestrator.CancelScan(c.Request.Context(), scanID); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	SuccessResponse(c, map[string]string{
		"message": "Scan cancelled successfully",
	})
}

// GetScanResults retrieves the results of a completed scan
func (h *ScanHandler) GetScanResults(c *gin.Context) {
	scanID := c.Param("id")
	if scanID == "" {
		BadRequestResponse(c, "Invalid scan ID")
		return
	}

	// TODO: Check user permissions

	// Get scan results from orchestrator
	results, err := h.orchestrator.GetScanResults(c.Request.Context(), scanID, nil)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	SuccessResponse(c, results)
}

// UpdateScanStatus updates the status of a scan (internal use)
func (h *ScanHandler) UpdateScanStatus(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequestResponse(c, "Invalid scan ID")
		return
	}

	var req UpdateScanJobStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestResponse(c, "Invalid request body: "+err.Error())
		return
	}

	// Update scan status
	if err := h.repos.ScanJobs.UpdateStatus(c.Request.Context(), scanID, req.Status); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	SuccessResponse(c, map[string]string{
		"message": "Scan status updated successfully",
	})
}

// GetScanMetrics returns scan metrics and statistics
func (h *ScanHandler) GetScanMetrics(c *gin.Context) {
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Calculate basic metrics
	// TODO: Implement more sophisticated metrics calculation from database
	metrics := map[string]interface{}{
		"total_scans":      100, // Placeholder - would come from database query
		"completed_scans":  0,
		"failed_scans":     0,
		"total_findings":   0,
		"high_findings":    0,
		"medium_findings":  0,
		"low_findings":     0,
		"recent_scans":     []interface{}{},
	}

	SuccessResponse(c, metrics)
}

// RetryFailedScan retries a failed scan
func (h *ScanHandler) RetryFailedScan(c *gin.Context) {
	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequestResponse(c, "Invalid scan ID")
		return
	}

	// Check if scan exists and user has permission
	scanJob, err := h.repos.ScanJobs.GetByID(c.Request.Context(), scanID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	userID, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Check if user owns this scan
	if scanJob.UserID == nil || *scanJob.UserID != userID {
		ForbiddenResponse(c, "You don't have permission to retry this scan")
		return
	}

	// Check if scan can be retried
	if scanJob.Status != types.ScanJobStatusFailed {
		BadRequestResponse(c, "Only failed scans can be retried")
		return
	}

	// Reset scan status and resubmit
	scanJob.Status = types.ScanJobStatusQueued
	scanJob.ErrorMessage = ""
	scanJob.StartedAt = nil
	scanJob.CompletedAt = nil
	scanJob.AgentsCompleted = []string{}

	if err := h.repos.ScanJobs.Update(c.Request.Context(), scanJob); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Create orchestrator scan request for retry
	scanReq := &orchestrator.ScanRequest{
		RepositoryID: scanJob.RepositoryID,
		UserID:       scanJob.UserID,
		RepoURL:      "", // TODO: Get from repository
		Branch:       scanJob.Branch,
		CommitSHA:    scanJob.CommitSHA,
		ScanType:     scanJob.ScanType,
		Priority:     scanJob.Priority,
		Agents:       scanJob.AgentsRequested,
		Metadata:     scanJob.Metadata,
	}

	// Resubmit to orchestrator
	if _, err := h.orchestrator.SubmitScan(c.Request.Context(), scanReq); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	SuccessResponse(c, ToScanJobDTO(scanJob))
}