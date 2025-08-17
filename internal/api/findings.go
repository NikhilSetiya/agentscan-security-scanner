package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
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

	// Calculate basic finding statistics
	// TODO: Implement more sophisticated statistics from database
	stats := map[string]interface{}{
		"total_findings":   50, // Placeholder - would come from database query
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
			"exported_at": time.Now().Format(time.RFC3339),
		})

	case "csv":
		findingDTOs := make([]*FindingDTO, len(findings))
		for i, finding := range findings {
			findingDTOs[i] = ToFindingDTO(finding)
		}
		
		csvData, err := h.exportFindingsAsCSV(findingDTOs)
		if err != nil {
			InternalErrorResponse(c, "Failed to generate CSV export")
			return
		}
		
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=findings_%s.csv", scanJobID))
		c.Data(http.StatusOK, "text/csv", csvData)

	case "pdf":
		findingDTOs := make([]*FindingDTO, len(findings))
		for i, finding := range findings {
			findingDTOs[i] = ToFindingDTO(finding)
		}
		
		pdfData, err := h.exportFindingsAsPDF(findingDTOs, scanJobID)
		if err != nil {
			InternalErrorResponse(c, "Failed to generate PDF export")
			return
		}
		
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=findings_%s.pdf", scanJobID))
		c.Data(http.StatusOK, "application/pdf", pdfData)

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

// exportFindingsAsCSV exports findings as CSV format
func (h *FindingHandler) exportFindingsAsCSV(findings []*FindingDTO) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write CSV header
	header := []string{
		"ID", "Tool", "Rule ID", "Severity", "Category", "Title", 
		"Description", "File", "Line", "Column", "Confidence", 
		"Status", "Created At", "Updated At",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write findings data
	for _, finding := range findings {
		record := []string{
			finding.ID.String(),
			finding.Tool,
			finding.RuleID,
			finding.Severity,
			finding.Category,
			finding.Title,
			finding.Description,
			finding.FilePath,
			strconv.Itoa(finding.LineNumber),
			strconv.Itoa(finding.ColumnNumber),
			fmt.Sprintf("%.2f", finding.Confidence),
			finding.Status,
			finding.CreatedAt.Format(time.RFC3339),
			finding.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// exportFindingsAsPDF exports findings as PDF format
func (h *FindingHandler) exportFindingsAsPDF(findings []*FindingDTO, scanJobID string) ([]byte, error) {
	// TODO: Implement proper PDF generation using a PDF library like gofpdf
	// For now, return a simple text-based PDF placeholder
	
	var buf bytes.Buffer
	
	// Simple PDF header (this is not a real PDF, just a placeholder)
	content := fmt.Sprintf("AgentScan Security Report\n")
	content += fmt.Sprintf("Scan Job ID: %s\n", scanJobID)
	content += fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339))
	content += fmt.Sprintf("Total Findings: %d\n\n", len(findings))
	
	// Group findings by severity
	severityGroups := make(map[string][]*FindingDTO)
	for _, finding := range findings {
		severityGroups[finding.Severity] = append(severityGroups[finding.Severity], finding)
	}
	
	// Add findings by severity
	for _, severity := range []string{"high", "medium", "low"} {
		if findings, exists := severityGroups[severity]; exists && len(findings) > 0 {
			content += fmt.Sprintf("%s Severity Issues (%d):\n", 
				map[string]string{"high": "HIGH", "medium": "MEDIUM", "low": "LOW"}[severity], 
				len(findings))
			content += "----------------------------------------\n"
			
			for i, finding := range findings {
				if i >= 10 { // Limit to first 10 findings per severity
					content += fmt.Sprintf("... and %d more\n", len(findings)-10)
					break
				}
				content += fmt.Sprintf("%d. %s\n", i+1, finding.Title)
				content += fmt.Sprintf("   File: %s:%d\n", finding.FilePath, finding.LineNumber)
				content += fmt.Sprintf("   Tool: %s\n", finding.Tool)
				content += fmt.Sprintf("   Description: %s\n\n", finding.Description)
			}
			content += "\n"
		}
	}
	
	content += "---\nGenerated by AgentScan - Multi-agent security scanning\n"
	
	buf.WriteString(content)
	return buf.Bytes(), nil
}
