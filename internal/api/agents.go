package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// AgentResultSubmissionRequest represents the request payload for agent result submission
type AgentResultSubmissionRequest struct {
	ScanJobID    string                 `json:"scan_job_id" binding:"required"`
	AgentName    string                 `json:"agent_name" binding:"required"`
	Status       string                 `json:"status" binding:"required"` // completed, failed
	Findings     []FindingSubmission    `json:"findings"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Duration     int64                  `json:"duration_ms"` // Duration in milliseconds
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FindingSubmission represents a finding submitted by an agent
type FindingSubmission struct {
	Tool          string                 `json:"tool" binding:"required"`
	RuleID        string                 `json:"rule_id" binding:"required"`
	Severity      string                 `json:"severity" binding:"required"`
	Category      string                 `json:"category" binding:"required"`
	Title         string                 `json:"title" binding:"required"`
	Description   string                 `json:"description" binding:"required"`
	FilePath      string                 `json:"file_path" binding:"required"`
	LineNumber    int                    `json:"line_number"`
	ColumnNumber  int                    `json:"column_number"`
	CodeSnippet   string                 `json:"code_snippet,omitempty"`
	Confidence    string                 `json:"confidence,omitempty"`
	FixSuggestion string                 `json:"fix_suggestion,omitempty"`
	References    []string               `json:"references,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AgentResultHandler handles agent result submissions
type AgentResultHandler struct {
	repos        *database.Repositories
	orchestrator orchestrator.OrchestrationService
}

// NewAgentResultHandler creates a new agent result handler
func NewAgentResultHandler(repos *database.Repositories, orchestrator orchestrator.OrchestrationService) *AgentResultHandler {
	return &AgentResultHandler{
		repos:        repos,
		orchestrator: orchestrator,
	}
}

// SubmitResults handles agent result submission
func (h *AgentResultHandler) SubmitResults(c *gin.Context) {
	var req AgentResultSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, err)
		return
	}

	// Parse scan job ID
	scanJobID, err := uuid.Parse(req.ScanJobID)
	if err != nil {
		BadRequestResponse(c, "Invalid scan job ID format")
		return
	}

	// Verify scan job exists
	scanJob, err := h.repos.ScanJobs.GetByID(c.Request.Context(), scanJobID)
	if err != nil {
		if err.Error() == "scan job not found" {
			NotFoundResponse(c, "Scan job not found")
		} else {
			InternalErrorResponse(c, "Failed to get scan job")
		}
		return
	}

	// Verify agent is authorized for this scan job
	if !h.isAgentAuthorized(req.AgentName, scanJob.AgentsRequested) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Agent not authorized for this scan job",
		})
		return
	}

	// Create scan result record
	scanResult := &types.ScanResult{
		ID:            uuid.New(),
		ScanJobID:     scanJobID,
		AgentName:     req.AgentName,
		Status:        req.Status,
		FindingsCount: len(req.Findings),
		DurationMS:    req.Duration,
		ErrorMessage:  req.ErrorMessage,
		Metadata:      req.Metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Store scan result
	if err := h.storeScanResult(c.Request.Context(), scanResult); err != nil {
		InternalErrorResponse(c, "Failed to store scan result")
		return
	}

	// Store findings
	for _, findingReq := range req.Findings {
		finding := &types.Finding{
			ID:            uuid.New(),
			ScanResultID:  scanResult.ID,
			ScanJobID:     scanJobID,
			Tool:          findingReq.Tool,
			RuleID:        findingReq.RuleID,
			Severity:      findingReq.Severity,
			Category:      findingReq.Category,
			Title:         findingReq.Title,
			Description:   findingReq.Description,
			FilePath:      findingReq.FilePath,
			LineNumber:    findingReq.LineNumber,
			ColumnNumber:  findingReq.ColumnNumber,
			CodeSnippet:   findingReq.CodeSnippet,
			Confidence:    findingReq.Confidence,
			Status:        "open", // Default status for new findings
			FixSuggestion: findingReq.FixSuggestion,
			References:    findingReq.References,
			Metadata:      findingReq.Metadata,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := h.repos.Findings.Create(c.Request.Context(), finding); err != nil {
			// Log error but continue processing other findings
			// TODO: Add proper logging
		}
	}

	// Update scan job with completed agent
	if err := h.updateScanJobProgress(c.Request.Context(), scanJob, req.AgentName, req.Status); err != nil {
		// Log error but don't fail the request as results are already stored
		// TODO: Add proper logging
	}

	// Check if scan is complete and notify orchestrator
	if h.isScanComplete(scanJob, req.AgentName) {
		if err := h.notifyScanComplete(c.Request.Context(), scanJob); err != nil {
			// Log error but don't fail the request
			// TODO: Add proper logging
		}
	}

	SuccessResponse(c, map[string]interface{}{
		"message":         "Results submitted successfully",
		"scan_result_id":  scanResult.ID,
		"findings_count":  len(req.Findings),
		"scan_complete":   h.isScanComplete(scanJob, req.AgentName),
	})
}

// isAgentAuthorized checks if the agent is authorized for the scan job
func (h *AgentResultHandler) isAgentAuthorized(agentName string, requestedAgents []string) bool {
	for _, requested := range requestedAgents {
		if requested == agentName {
			return true
		}
	}
	return false
}

// storeScanResult stores a scan result in the database
func (h *AgentResultHandler) storeScanResult(ctx context.Context, result *types.ScanResult) error {
	// TODO: Implement scan result storage in database
	// For now, we'll just log it
	// This requires implementing the ScanResult repository methods
	return nil
}

// updateScanJobProgress updates the scan job with completed agent
func (h *AgentResultHandler) updateScanJobProgress(ctx context.Context, scanJob *types.ScanJob, agentName, status string) error {
	// Add agent to completed list if not already there
	for _, completed := range scanJob.AgentsCompleted {
		if completed == agentName {
			return nil // Already marked as completed
		}
	}

	scanJob.AgentsCompleted = append(scanJob.AgentsCompleted, agentName)
	scanJob.UpdatedAt = time.Now()

	// If this was the last agent, mark scan as completed
	if len(scanJob.AgentsCompleted) >= len(scanJob.AgentsRequested) {
		scanJob.Status = types.ScanJobStatusCompleted
		now := time.Now()
		scanJob.CompletedAt = &now
	}

	return h.repos.ScanJobs.Update(ctx, scanJob)
}

// isScanComplete checks if all agents have completed
func (h *AgentResultHandler) isScanComplete(scanJob *types.ScanJob, completedAgent string) bool {
	// Count unique completed agents
	completedSet := make(map[string]bool)
	for _, agent := range scanJob.AgentsCompleted {
		completedSet[agent] = true
	}
	completedSet[completedAgent] = true

	return len(completedSet) >= len(scanJob.AgentsRequested)
}

// notifyScanComplete notifies the orchestrator that the scan is complete
func (h *AgentResultHandler) notifyScanComplete(ctx context.Context, scanJob *types.ScanJob) error {
	// TODO: Implement scan completion notification
	// This could trigger webhooks, notifications, etc.
	return nil
}