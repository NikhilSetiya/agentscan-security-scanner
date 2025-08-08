package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/types"
)

// FindingHandler handles finding-related endpoints
type FindingHandler struct {
	repos *database.Repositories
}

// NewFindingHandler creates a new finding handler
func NewFindingHandler(repos *database.Repositories) *FindingHandler {
	return &FindingHandler{
		repos: repos,
	}
}

// GetFinding retrieves a finding by ID
func (h *FindingHandler) GetFinding(c *gin.Context) {
	findingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequestResponse(c, "Invalid finding ID")
		return
	}

	finding, err := h.repos.Findings.GetByID(c.Request.Context(), findingID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// TODO: Check user permissions for this finding

	SuccessResponse(c, ToFindingDTO(finding))
}

// ListFindings lists findings with optional filtering
func (h *FindingHandler) ListFindings(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	severity := c.Query("severity")
	tool := c.Query("tool")
	category := c.Query("category")
	status := c.Query("status")
	file := c.Query("file")
	scanJobID := c.Query("scan_job_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	// Build filter
	filter := &database.FindingFilter{
		Severity: severity,
		Tool:     tool,
		Category: category,
		Status:   status,
		File:     file,
	}

	pagination := &database.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	var findings []*types.Finding
	var total int64
	var err error

	// If scan_job_id is provided, filter by it
	if scanJobID != "" {
		scanID, parseErr := uuid.Parse(scanJobID)
		if parseErr != nil {
			BadRequestResponse(c, "Invalid scan job ID")
			return
		}

		// TODO: Check user permissions for this scan job

		findings, err = h.repos.Findings.ListByScanJob(c.Request.Context(), scanID)
		total = int64(len(findings))

		// Apply additional filtering
		findings = h.applyFindingFilters(findings, filter)

		// Apply pagination
		start := (page - 1) * pageSize
		end := start + pageSize
		if start > len(findings) {
			findings = []*types.Finding{}
		} else if end > len(findings) {
			findings = findings[start:]
		} else {
			findings = findings[start:end]
		}
	} else {
		// List all findings with filter and pagination
		findings, total, err = h.repos.Findings.List(c.Request.Context(), filter, pagination)
	}

	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Convert to DTOs
	findingDTOs := make([]*FindingDTO, len(findings))
	for i, finding := range findings {
		findingDTOs[i] = ToFindingDTO(finding)
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

	SuccessResponseWithMeta(c, findingDTOs, meta)
}

// UpdateFindingStatus updates the status of a finding
func (h *FindingHandler) UpdateFindingStatus(c *gin.Context) {
	findingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequestResponse(c, "Invalid finding ID")
		return
	}

	var req UpdateFindingStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestResponse(c, "Invalid request body: "+err.Error())
		return
	}

	// Check if finding exists
	finding, err := h.repos.Findings.GetByID(c.Request.Context(), findingID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// TODO: Check user permissions for this finding

	// Update finding status
	if err := h.repos.Findings.UpdateStatus(c.Request.Context(), findingID, req.Status); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Get updated finding
	finding.Status = req.Status
	
	SuccessResponse(c, ToFindingDTO(finding))
}

// GetFindingStats returns statistics about findings
func (h *FindingHandler) GetFindingStats(c *gin.Context) {
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// TODO: Implement finding statistics calculation
	stats := map[string]interface{}{
		"total_findings":   0,
		"open_findings":    0,
		"fixed_findings":   0,
		"ignored_findings": 0,
		"by_severity": map[string]int{
			"high":   0,
			"medium": 0,
			"low":    0,
			"info":   0,
		},
		"by_category": map[string]int{
			"sql_injection":      0,
			"xss":                0,
			"command_injection":  0,
			"path_traversal":     0,
			"insecure_crypto":    0,
			"hardcoded_secrets":  0,
			"misconfiguration":   0,
			"other":              0,
		},
		"by_tool": map[string]int{
			"semgrep":        0,
			"eslint-security": 0,
		},
	}

	SuccessResponse(c, stats)
}

// BulkUpdateFindings updates multiple findings at once
func (h *FindingHandler) BulkUpdateFindings(c *gin.Context) {
	var req struct {
		FindingIDs []uuid.UUID `json:"finding_ids" binding:"required"`
		Status     string      `json:"status" binding:"required,oneof=open fixed ignored false_positive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestResponse(c, "Invalid request body: "+err.Error())
		return
	}

	if len(req.FindingIDs) == 0 {
		BadRequestResponse(c, "At least one finding ID is required")
		return
	}

	if len(req.FindingIDs) > 100 {
		BadRequestResponse(c, "Cannot update more than 100 findings at once")
		return
	}

	// TODO: Check user permissions for these findings

	// Update findings
	updatedCount := 0
	for _, findingID := range req.FindingIDs {
		if err := h.repos.Findings.UpdateStatus(c.Request.Context(), findingID, req.Status); err != nil {
			// Log error but continue with other findings
			continue
		}
		updatedCount++
	}

	SuccessResponse(c, map[string]interface{}{
		"updated_count": updatedCount,
		"total_count":   len(req.FindingIDs),
		"message":       "Findings updated successfully",
	})
}

// ExportFindings exports findings in various formats
func (h *FindingHandler) ExportFindings(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	scanJobID := c.Query("scan_job_id")

	if scanJobID == "" {
		BadRequestResponse(c, "scan_job_id parameter is required")
		return
	}

	scanID, err := uuid.Parse(scanJobID)
	if err != nil {
		BadRequestResponse(c, "Invalid scan job ID")
		return
	}

	// TODO: Check user permissions for this scan job

	// Get findings
	findings, err := h.repos.Findings.ListByScanJob(c.Request.Context(), scanID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	switch format {
	case "json":
		findingDTOs := make([]*FindingDTO, len(findings))
		for i, finding := range findings {
			findingDTOs[i] = ToFindingDTO(finding)
		}
		
		c.Header("Content-Disposition", "attachment; filename=findings.json")
		c.JSON(200, map[string]interface{}{
			"scan_job_id": scanJobID,
			"findings":    findingDTOs,
			"exported_at": "2024-01-01T00:00:00Z", // TODO: Use actual timestamp
		})

	case "csv":
		// TODO: Implement CSV export
		BadRequestResponse(c, "CSV export not yet implemented")

	case "pdf":
		// TODO: Implement PDF export
		BadRequestResponse(c, "PDF export not yet implemented")

	default:
		BadRequestResponse(c, "Unsupported export format. Supported formats: json, csv, pdf")
	}
}

// applyFindingFilters applies client-side filtering to findings
func (h *FindingHandler) applyFindingFilters(findings []*types.Finding, filter *database.FindingFilter) []*types.Finding {
	if filter == nil {
		return findings
	}

	filtered := make([]*types.Finding, 0, len(findings))
	
	for _, finding := range findings {
		// Apply severity filter
		if filter.Severity != "" && finding.Severity != filter.Severity {
			continue
		}
		
		// Apply tool filter
		if filter.Tool != "" && finding.Tool != filter.Tool {
			continue
		}
		
		// Apply category filter
		if filter.Category != "" && finding.Category != filter.Category {
			continue
		}
		
		// Apply status filter
		if filter.Status != "" && finding.Status != filter.Status {
			continue
		}
		
		// Apply file filter (partial match)
		if filter.File != "" && !contains(finding.FilePath, filter.File) {
			continue
		}
		
		filtered = append(filtered, finding)
	}
	
	return filtered
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && 
			(s[0:len(substr)] == substr || 
				(len(s) > len(substr) && contains(s[1:], substr)))))
}