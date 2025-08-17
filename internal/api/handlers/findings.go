package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/findings"
)

// FindingsHandler handles HTTP requests for findings management
type FindingsHandler struct {
	service *findings.Service
}

// NewFindingsHandler creates a new findings handler
func NewFindingsHandler(service *findings.Service) *FindingsHandler {
	return &FindingsHandler{service: service}
}

// GetFinding retrieves a specific finding by ID
func (h *FindingsHandler) GetFinding(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid finding ID"})
		return
	}

	finding, err := h.service.GetFinding(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "finding not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Finding not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get finding"})
		return
	}

	c.JSON(http.StatusOK, finding)
}

// ListFindings retrieves findings with filtering and pagination
func (h *FindingsHandler) ListFindings(c *gin.Context) {
	var filter findings.FindingFilter

	// Parse scan job ID
	if scanJobIDStr := c.Query("scan_job_id"); scanJobIDStr != "" {
		scanJobID, err := uuid.Parse(scanJobIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan job ID"})
			return
		}
		filter.ScanJobID = &scanJobID
	}

	// Parse severity filter
	if severities := c.QueryArray("severity"); len(severities) > 0 {
		filter.Severity = severities
	}

	// Parse status filter
	if statuses := c.QueryArray("status"); len(statuses) > 0 {
		var statusEnums []findings.FindingStatus
		for _, status := range statuses {
			statusEnums = append(statusEnums, findings.FindingStatus(status))
		}
		filter.Status = statusEnums
	}

	// Parse tool filter
	if tools := c.QueryArray("tool"); len(tools) > 0 {
		filter.Tool = tools
	}

	// Parse file path filter
	if filePath := c.Query("file_path"); filePath != "" {
		filter.FilePath = &filePath
	}

	// Parse minimum confidence
	if minConfStr := c.Query("min_confidence"); minConfStr != "" {
		minConf, err := strconv.ParseFloat(minConfStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid minimum confidence value"})
			return
		}
		filter.MinConfidence = &minConf
	}

	// Parse search query
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	// Parse pagination
	limit := 50 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	findingsList, err := h.service.ListFindings(c.Request.Context(), filter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list findings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"findings": findingsList,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(findingsList),
		},
	})
}

// UpdateFindingStatus updates the status of a finding
func (h *FindingsHandler) UpdateFindingStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid finding ID"})
		return
	}

	var req struct {
		Status  findings.FindingStatus `json:"status" binding:"required"`
		Comment *string                `json:"comment,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	err = h.service.UpdateFindingStatus(c.Request.Context(), id, userUUID, req.Status, req.Comment)
	if err != nil {
		if err.Error() == "finding not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Finding not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update finding status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Finding status updated successfully"})
}

// SuppressFinding creates a suppression rule for a finding
func (h *FindingsHandler) SuppressFinding(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid finding ID"})
		return
	}

	var req struct {
		Reason    string     `json:"reason" binding:"required"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	err = h.service.SuppressFinding(c.Request.Context(), id, userUUID, req.Reason, req.ExpiresAt)
	if err != nil {
		if err.Error() == "finding not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Finding not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suppress finding"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Finding suppressed successfully"})
}

// GetSuppressions retrieves suppression rules for the current user
func (h *FindingsHandler) GetSuppressions(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	suppressions, err := h.service.GetSuppressions(c.Request.Context(), userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get suppressions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"suppressions": suppressions})
}

// DeleteSuppression removes a suppression rule
func (h *FindingsHandler) DeleteSuppression(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suppression ID"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	err = h.service.DeleteSuppression(c.Request.Context(), id, userUUID)
	if err != nil {
		if err.Error() == "suppression not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Suppression not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete suppression"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Suppression deleted successfully"})
}

// GetFindingStats retrieves statistics about findings for a scan job
func (h *FindingsHandler) GetFindingStats(c *gin.Context) {
	scanJobIDStr := c.Param("scan_job_id")
	scanJobID, err := uuid.Parse(scanJobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan job ID"})
		return
	}

	stats, err := h.service.GetFindingStats(c.Request.Context(), scanJobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get finding stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ExportFindings exports findings in the specified format
func (h *FindingsHandler) ExportFindings(c *gin.Context) {
	var req findings.ExportRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate export format
	switch req.Format {
	case findings.ExportFormatJSON, findings.ExportFormatPDF, findings.ExportFormatCSV:
		// Valid formats
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export format"})
		return
	}

	result, err := h.service.ExportFindings(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export findings"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// BulkUpdateFindings updates multiple findings at once
func (h *FindingsHandler) BulkUpdateFindings(c *gin.Context) {
	var req struct {
		FindingIDs []uuid.UUID            `json:"finding_ids" binding:"required"`
		Status     findings.FindingStatus `json:"status" binding:"required"`
		Comment    *string                `json:"comment,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.FindingIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No finding IDs provided"})
		return
	}

	if len(req.FindingIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Too many findings (max 100)"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	err := h.service.BulkUpdateFindings(c.Request.Context(), req.FindingIDs, userUUID, req.Status, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update findings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Findings updated successfully",
		"count":   len(req.FindingIDs),
	})
}

// GetUserFeedback retrieves user feedback for ML training
func (h *FindingsHandler) GetUserFeedback(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Parse pagination
	limit := 50 // default
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	feedback, err := h.service.GetUserFeedback(c.Request.Context(), userUUID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"feedback": feedback,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(feedback),
		},
	})
}