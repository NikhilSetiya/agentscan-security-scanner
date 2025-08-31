package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/commands"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/queries"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/services"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// ScanJobHandler handles HTTP requests for scan job operations
type ScanJobHandler struct {
	appService *services.ApplicationService
}

// NewScanJobHandler creates a new scan job handler
func NewScanJobHandler(appService *services.ApplicationService) *ScanJobHandler {
	return &ScanJobHandler{
		appService: appService,
	}
}

// CreateScanJobRequest represents the request to create a scan job
type CreateScanJobRequest struct {
	RepositoryID uuid.UUID         `json:"repository_id" binding:"required"`
	UserID       *uuid.UUID        `json:"user_id,omitempty"`
	Branch       string            `json:"branch" binding:"omitempty,max=255"`
	CommitSHA    string            `json:"commit_sha" binding:"omitempty,max=40"`
	ScanType     entities.ScanType `json:"scan_type" binding:"required"`
	Priority     entities.Priority `json:"priority" binding:"min=1,max=10"`
	Agents       []string          `json:"agents" binding:"omitempty,dive,min=1,max=50"`
}

// UpdateScanJobMetadataRequest represents the request to update scan job metadata
type UpdateScanJobMetadataRequest struct {
	Metadata map[string]interface{} `json:"metadata" binding:"required"`
}

// AddCompletedAgentRequest represents the request to add a completed agent
type AddCompletedAgentRequest struct {
	Agent string `json:"agent" binding:"required,min=1,max=50"`
}

// FailScanJobRequest represents the request to fail a scan job
type FailScanJobRequest struct {
	ErrorMessage string `json:"error_message" binding:"required,max=1000"`
}

// CreateScanJob handles POST /scan-jobs
func (h *ScanJobHandler) CreateScanJob(c *gin.Context) {
	var req CreateScanJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.CreateScanJobCommand{
		RepositoryID: req.RepositoryID,
		UserID:       req.UserID,
		Branch:       req.Branch,
		CommitSHA:    req.CommitSHA,
		ScanType:     req.ScanType,
		Priority:     req.Priority,
		Agents:       req.Agents,
	}

	scanJob, err := h.appService.CreateScanJob(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    scanJob,
	})
}

// GetScanJob handles GET /scan-jobs/:id
func (h *ScanJobHandler) GetScanJob(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	query := queries.GetScanJobByIDQuery{
		ScanJobID: scanJobID,
	}

	scanJob, err := h.appService.GetScanJobByID(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    scanJob,
	})
}

// GetScanJobWithDetails handles GET /scan-jobs/:id/details
func (h *ScanJobHandler) GetScanJobWithDetails(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	query := queries.GetScanJobWithDetailsQuery{
		ScanJobID: scanJobID,
	}

	scanJobDetails, err := h.appService.GetScanJobWithDetails(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    scanJobDetails,
	})
}

// ListScanJobs handles GET /scan-jobs
func (h *ScanJobHandler) ListScanJobs(c *gin.Context) {
	// Parse pagination parameters
	page, pageSize := parsePagination(c)
	
	// Parse filter parameters
	filter := repositories.Filter{
		Search:    c.Query("search"),
		Status:    c.Query("status"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	pagination := repositories.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	query := queries.ListScanJobsQuery{
		Filter:     filter,
		Pagination: pagination,
	}

	scanJobs, total, err := h.appService.ListScanJobs(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    scanJobs,
		"meta": gin.H{
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
			},
		},
	})
}

// ListScanJobsByRepository handles GET /repositories/:id/scan-jobs
func (h *ScanJobHandler) ListScanJobsByRepository(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	// Parse pagination parameters
	page, pageSize := parsePagination(c)
	
	// Parse filter parameters
	filter := repositories.Filter{
		Search:    c.Query("search"),
		Status:    c.Query("status"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	pagination := repositories.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	query := queries.ListScanJobsByRepositoryQuery{
		RepositoryID: repositoryID,
		Filter:       filter,
		Pagination:   pagination,
	}

	scanJobs, total, err := h.appService.ListScanJobsByRepository(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    scanJobs,
		"meta": gin.H{
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
			},
		},
	})
}

// ListScanJobsByUser handles GET /users/:id/scan-jobs
func (h *ScanJobHandler) ListScanJobsByUser(c *gin.Context) {
	userID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Parse pagination parameters
	page, pageSize := parsePagination(c)
	
	// Parse filter parameters
	filter := repositories.Filter{
		Search:    c.Query("search"),
		Status:    c.Query("status"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	pagination := repositories.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	query := queries.ListScanJobsByUserQuery{
		UserID:     userID,
		Filter:     filter,
		Pagination: pagination,
	}

	scanJobs, total, err := h.appService.ListScanJobsByUser(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    scanJobs,
		"meta": gin.H{
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
			},
		},
	})
}

// GetQueuedJobs handles GET /scan-jobs/queued
func (h *ScanJobHandler) GetQueuedJobs(c *gin.Context) {
	limit := 50 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := parseIntQuery(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	query := queries.GetQueuedJobsQuery{
		Limit: limit,
	}

	scanJobs, err := h.appService.GetQueuedJobs(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    scanJobs,
	})
}

// GetRunningJobs handles GET /scan-jobs/running
func (h *ScanJobHandler) GetRunningJobs(c *gin.Context) {
	query := queries.GetRunningJobsQuery{}

	scanJobs, err := h.appService.GetRunningJobs(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    scanJobs,
	})
}

// StartScanJob handles POST /scan-jobs/:id/start
func (h *ScanJobHandler) StartScanJob(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	cmd := commands.StartScanJobCommand{
		ScanJobID: scanJobID,
	}

	err = h.appService.StartScanJob(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Scan job started successfully",
	})
}

// CompleteScanJob handles POST /scan-jobs/:id/complete
func (h *ScanJobHandler) CompleteScanJob(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	cmd := commands.CompleteScanJobCommand{
		ScanJobID: scanJobID,
	}

	err = h.appService.CompleteScanJob(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Scan job completed successfully",
	})
}

// FailScanJob handles POST /scan-jobs/:id/fail
func (h *ScanJobHandler) FailScanJob(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	var req FailScanJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.FailScanJobCommand{
		ScanJobID:    scanJobID,
		ErrorMessage: req.ErrorMessage,
	}

	err = h.appService.FailScanJob(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Scan job marked as failed",
	})
}

// CancelScanJob handles POST /scan-jobs/:id/cancel
func (h *ScanJobHandler) CancelScanJob(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user context",
		})
		return
	}

	cmd := commands.CancelScanJobCommand{
		ScanJobID: scanJobID,
		UserID:    uid,
	}

	err = h.appService.CancelScanJob(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Scan job cancelled successfully",
	})
}

// RetryScanJob handles POST /scan-jobs/:id/retry
func (h *ScanJobHandler) RetryScanJob(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user context",
		})
		return
	}

	cmd := commands.RetryScanJobCommand{
		ScanJobID: scanJobID,
		UserID:    uid,
	}

	newScanJob, err := h.appService.RetryScanJob(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    newScanJob,
		"message": "Scan job retried successfully",
	})
}

// AddCompletedAgent handles POST /scan-jobs/:id/completed-agents
func (h *ScanJobHandler) AddCompletedAgent(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	var req AddCompletedAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.AddCompletedAgentCommand{
		ScanJobID: scanJobID,
		Agent:     req.Agent,
	}

	err = h.appService.AddCompletedAgent(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Agent marked as completed",
	})
}

// UpdateScanJobMetadata handles PUT /scan-jobs/:id/metadata
func (h *ScanJobHandler) UpdateScanJobMetadata(c *gin.Context) {
	scanJobID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scan job ID",
		})
		return
	}

	var req UpdateScanJobMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.UpdateScanJobMetadataCommand{
		ScanJobID: scanJobID,
		Metadata:  req.Metadata,
	}

	err = h.appService.UpdateScanJobMetadata(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Scan job metadata updated successfully",
	})
}