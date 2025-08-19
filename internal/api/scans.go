package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
		ValidationErrorResponse(c, "Invalid request body", map[string]interface{}{
			"validation_errors": err.Error(),
		})
		return
	}

	userID, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Find or create repository
	repo, err := h.repos.Repositories.GetByURL(c.Request.Context(), req.RepositoryURL)
	if err != nil {
		// Repository doesn't exist, create it
		repo = &types.Repository{
			ID:             uuid.New(),
			OrganizationID: uuid.New(), // TODO: Get from user's organization
			Name:           h.extractRepoNameFromURL(req.RepositoryURL),
			URL:            req.RepositoryURL,
			Provider:       h.extractProviderFromURL(req.RepositoryURL),
			ProviderID:     h.extractProviderIDFromURL(req.RepositoryURL),
			DefaultBranch:  req.Branch,
			Language:       "Unknown", // Will be detected during scan
			Languages:      []string{},
			Settings:       make(map[string]interface{}),
			IsActive:       true,
		}

		if repo.DefaultBranch == "" {
			repo.DefaultBranch = "main"
		}

		if err := h.repos.Repositories.Create(c.Request.Context(), repo); err != nil {
			ErrorResponseFromError(c, err)
			return
		}
	}

	// Set defaults
	branch := req.Branch
	if branch == "" {
		branch = repo.DefaultBranch
	}

	commitSHA := req.CommitSHA
	if commitSHA == "" {
		commitSHA = "HEAD" // Will be resolved during scan
	}

	priority := req.Priority
	if priority == 0 {
		priority = types.PriorityMedium
	}

	agents := req.AgentsRequested
	if len(agents) == 0 {
		agents = []string{"semgrep", "eslint-security"}
	}

	// Create scan job directly in database
	scanJob := &types.ScanJob{
		ID:               uuid.New(),
		RepositoryID:     repo.ID,
		UserID:           &userID,
		Branch:           branch,
		CommitSHA:        commitSHA,
		ScanType:         req.ScanType,
		Priority:         priority,
		Status:           types.ScanJobStatusQueued,
		AgentsRequested:  agents,
		AgentsCompleted:  []string{},
		Metadata:         make(map[string]interface{}),
	}

	if err := h.repos.ScanJobs.Create(c.Request.Context(), scanJob); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Submit to orchestrator for processing
	scanReq := &orchestrator.ScanRequest{
		RepositoryID: repo.ID,
		UserID:       &userID,
		RepoURL:      req.RepositoryURL,
		Branch:       branch,
		CommitSHA:    commitSHA,
		ScanType:     req.ScanType,
		Priority:     priority,
		Agents:       agents,
		Metadata:     make(map[string]interface{}),
	}

	// Submit to orchestrator (this will update the scan job status)
	_, err = h.orchestrator.SubmitScan(c.Request.Context(), scanReq)
	if err != nil {
		// If orchestrator fails, mark scan as failed
		h.repos.ScanJobs.SetFailed(c.Request.Context(), scanJob.ID, err.Error())
		ErrorResponseFromError(c, err)
		return
	}

	CreatedResponse(c, ToScanJobDTO(scanJob))
}

// Helper methods for repository URL parsing

// extractRepoNameFromURL extracts repository name from URL
func (h *ScanHandler) extractRepoNameFromURL(repoURL string) string {
	// Simple extraction - in production, you'd use a proper URL parser
	parts := strings.Split(strings.TrimSuffix(repoURL, ".git"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown-repo"
}

// extractProviderFromURL extracts provider from URL
func (h *ScanHandler) extractProviderFromURL(repoURL string) string {
	if strings.Contains(repoURL, "github.com") {
		return "github"
	} else if strings.Contains(repoURL, "gitlab.com") {
		return "gitlab"
	} else if strings.Contains(repoURL, "bitbucket.org") {
		return "bitbucket"
	}
	return "git"
}

// extractProviderIDFromURL extracts provider-specific ID from URL
func (h *ScanHandler) extractProviderIDFromURL(repoURL string) string {
	// Extract owner/repo from URL
	if strings.Contains(repoURL, "github.com") || strings.Contains(repoURL, "gitlab.com") || strings.Contains(repoURL, "bitbucket.org") {
		parts := strings.Split(strings.TrimSuffix(repoURL, ".git"), "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("%s/%s", parts[len(parts)-2], parts[len(parts)-1])
		}
	}
	return repoURL
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
	scanType := c.Query("scan_type")
	repositoryIDStr := c.Query("repository_id")

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
		UserID: &userID,
	}

	if status != "" {
		filter.Status = status
	}

	if scanType != "" {
		filter.ScanType = scanType
	}

	if repositoryIDStr != "" {
		if repoID, err := uuid.Parse(repositoryIDStr); err == nil {
			filter.RepositoryID = &repoID
		}
	}

	pagination := &database.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	// Get scan jobs from database
	scanJobs, total, err := h.repos.ScanJobs.List(c.Request.Context(), filter, pagination)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Convert to API format
	var scans []map[string]interface{}
	for _, job := range scanJobs {
		// Get repository information
		repo, err := h.repos.Repositories.GetByID(c.Request.Context(), job.RepositoryID)
		if err != nil {
			// Skip if repository not found, don't fail the whole request
			continue
		}

		// Calculate progress based on status
		progress := 0
		switch job.Status {
		case "completed":
			progress = 100
		case "running":
			progress = 50 // Assume 50% for running scans
		case "failed", "cancelled":
			progress = 100 // Show as complete for failed/cancelled
		}

		// Calculate duration
		duration := ""
		if job.StartedAt != nil && job.CompletedAt != nil {
			durationSeconds := int(job.CompletedAt.Sub(*job.StartedAt).Seconds())
			if durationSeconds >= 60 {
				minutes := durationSeconds / 60
				seconds := durationSeconds % 60
				duration = fmt.Sprintf("%dm %ds", minutes, seconds)
			} else {
				duration = fmt.Sprintf("%ds", durationSeconds)
			}
		}

		// Get findings count for this scan
		findings, err := h.repos.Findings.ListByScanJob(c.Request.Context(), job.ID)
		findingsCount := 0
		if err == nil {
			findingsCount = len(findings)
		}

		// Get user information for triggered_by
		triggeredBy := "system"
		if job.UserID != nil {
			if user, err := h.repos.Users.GetByID(c.Request.Context(), *job.UserID); err == nil {
				triggeredBy = user.Email
			}
		}

		scanData := map[string]interface{}{
			"id":              job.ID.String(),
			"repository_id":   job.RepositoryID.String(),
			"repository": map[string]interface{}{
				"id":       repo.ID.String(),
				"name":     repo.Name,
				"url":      repo.URL,
				"language": repo.Language,
				"branch":   repo.DefaultBranch,
				"created_at": repo.CreatedAt.Format(time.RFC3339),
			},
			"status":         job.Status,
			"progress":       progress,
			"findings_count": findingsCount,
			"branch":         job.Branch,
			"commit":         job.CommitSHA,
			"scan_type":      job.ScanType,
			"triggered_by":   triggeredBy,
			"created_at":     job.CreatedAt.Format(time.RFC3339),
		}

		if repo.LastScanAt != nil {
			scanData["repository"].(map[string]interface{})["last_scan_at"] = repo.LastScanAt.Format(time.RFC3339)
		}

		if job.StartedAt != nil {
			scanData["started_at"] = job.StartedAt.Format(time.RFC3339)
		}

		if job.CompletedAt != nil {
			scanData["completed_at"] = job.CompletedAt.Format(time.RFC3339)
		}

		if duration != "" {
			scanData["duration"] = duration
		}

		if job.ErrorMessage != "" {
			scanData["error_message"] = job.ErrorMessage
		}

		// Add commit message placeholder (would come from git integration)
		scanData["commit_message"] = fmt.Sprintf("Commit %s", job.CommitSHA[:8])

		scans = append(scans, scanData)
	}

	// Return in format expected by frontend (ScanListResponse) using new pagination structure
	responseData := map[string]interface{}{
		"scans": scans,
	}
	
	PaginatedResponse(c, responseData, page, pageSize, total)
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