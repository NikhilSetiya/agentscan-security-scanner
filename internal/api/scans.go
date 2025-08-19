package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
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
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20")) // Frontend uses 'limit' not 'page_size'
	status := c.Query("status")
	_ = c.Query("scan_type") // scanType not used in mock

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Return mock scans data matching frontend Scan interface exactly
	mockScans := []map[string]interface{}{
		{
			"id":              "scan-1",
			"repository_id":   "repo-1",
			"repository": map[string]interface{}{
				"id":           "repo-1",
				"name":         "demo-repo",
				"url":          "https://github.com/demo/repo",
				"language":     "JavaScript",
				"branch":       "main",
				"created_at":   "2025-08-17T10:00:00Z",
				"last_scan_at": "2025-08-18T22:32:15Z",
			},
			"status":           "completed",
			"progress":         100,
			"findings_count":   7,
			"started_at":       "2025-08-18T22:30:00Z",
			"completed_at":     "2025-08-18T22:32:15Z",
			"duration":         "2m 15s",
			"branch":           "main",
			"commit":           "abc123",
			"commit_message":   "Fix security vulnerability in authentication",
			"triggered_by":     "user@example.com",
			"scan_type":        "full",
		},
		{
			"id":              "scan-2",
			"repository_id":   "repo-2",
			"repository": map[string]interface{}{
				"id":           "repo-2",
				"name":         "api-service", 
				"url":          "https://github.com/demo/api",
				"language":     "Python",
				"branch":       "develop",
				"created_at":   "2025-08-16T14:00:00Z",
				"last_scan_at": "2025-08-18T23:15:00Z",
			},
			"status":         "running",
			"progress":       65,
			"findings_count": 3,
			"started_at":     "2025-08-18T23:15:00Z",
			"branch":         "develop",
			"commit":         "def456",
			"commit_message": "Add new API endpoint",
			"triggered_by":   "admin@example.com",
			"scan_type":      "incremental",
		},
		{
			"id":              "scan-3",
			"repository_id":   "repo-3",
			"repository": map[string]interface{}{
				"id":         "repo-3",
				"name":       "frontend-app",
				"url":        "https://github.com/demo/frontend",
				"language":   "TypeScript",
				"branch":     "main",
				"created_at": "2025-08-15T09:30:00Z",
			},
			"status":         "queued",
			"progress":       0,
			"findings_count": 0,
			"started_at":     "2025-08-19T08:00:00Z",
			"branch":         "main",
			"commit":         "ghi789",
			"commit_message": "Update dependencies",
			"triggered_by":   "user@example.com",
			"scan_type":      "full",
		},
	}

	// Filter by status if provided
	filteredScans := mockScans
	if status != "" {
		filteredScans = []map[string]interface{}{}
		for _, scan := range mockScans {
			if scan["status"] == status {
				filteredScans = append(filteredScans, scan)
			}
		}
	}

	// Calculate pagination
	total := int64(len(filteredScans))
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(filteredScans) {
		filteredScans = []map[string]interface{}{}
	} else {
		if end > len(filteredScans) {
			end = len(filteredScans)
		}
		filteredScans = filteredScans[start:end]
	}

	// Return in format expected by frontend (ScanListResponse)
	SuccessResponse(c, map[string]interface{}{
		"scans": filteredScans,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
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