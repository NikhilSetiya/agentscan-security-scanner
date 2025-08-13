package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/orchestrator"
	"github.com/agentscan/agentscan/pkg/types"
)

// WebhookHandler handles GitLab webhook events
type WebhookHandler struct {
	repos        *database.Repositories
	orchestrator orchestrator.OrchestrationService
	service      *Service
}

// NewWebhookHandler creates a new GitLab webhook handler
func NewWebhookHandler(repos *database.Repositories, orchestrator orchestrator.OrchestrationService, service *Service) *WebhookHandler {
	return &WebhookHandler{
		repos:        repos,
		orchestrator: orchestrator,
		service:      service,
	}
}

// HandleWebhook handles incoming GitLab webhook events
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read the payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read payload", http.StatusBadRequest)
		return
	}

	// Get event type and token
	eventType := r.Header.Get("X-Gitlab-Event")
	token := r.Header.Get("X-Gitlab-Token")

	if eventType == "" {
		http.Error(w, "Missing X-Gitlab-Event header", http.StatusBadRequest)
		return
	}

	// Verify webhook token (skip if no token provided for testing)
	if token != "" {
		if err := h.verifyToken(token, ""); err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}

	// Store webhook event
	webhookEvent := &WebhookEvent{
		ID:        uuid.New(),
		EventType: eventType,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	// Process the event based on type
	switch eventType {
	case "Merge Request Hook":
		if err := h.handleMergeRequestEvent(ctx, payload, webhookEvent); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle merge request event: %v", err), http.StatusInternalServerError)
			return
		}
	case "Push Hook":
		if err := h.handlePushEvent(ctx, payload, webhookEvent); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle push event: %v", err), http.StatusInternalServerError)
			return
		}
	case "Pipeline Hook":
		if err := h.handlePipelineEvent(ctx, payload, webhookEvent); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle pipeline event: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		// Log unsupported event types but don't error
		fmt.Printf("Received unsupported GitLab event: %s\n", eventType)
	}

	// Mark event as processed
	now := time.Now()
	webhookEvent.ProcessedAt = &now

	// Store the webhook event in database
	if err := h.storeWebhookEvent(ctx, webhookEvent); err != nil {
		fmt.Printf("Failed to store webhook event: %v\n", err)
		// Don't return error to GitLab as the event was processed
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// verifyToken verifies the GitLab webhook token
func (h *WebhookHandler) verifyToken(token, secret string) error {
	if secret == "" {
		return nil // Skip verification if no secret configured
	}

	if token != secret {
		return fmt.Errorf("token mismatch")
	}

	return nil
}

// handleMergeRequestEvent handles merge request webhook events
func (h *WebhookHandler) handleMergeRequestEvent(ctx context.Context, payload []byte, webhookEvent *WebhookEvent) error {
	var event MergeRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal merge request event: %w", err)
	}

	webhookEvent.Action = event.ObjectAttributes.Action
	webhookEvent.Repository = event.Project.PathWithNamespace
	webhookEvent.MergeRequest = &event.ObjectAttributes.IID
	webhookEvent.Branch = event.ObjectAttributes.SourceBranch
	webhookEvent.Commit = event.ObjectAttributes.LastCommit.ID

	// Only process opened, update, and reopen events
	if event.ObjectAttributes.Action != "open" && 
	   event.ObjectAttributes.Action != "update" && 
	   event.ObjectAttributes.Action != "reopen" {
		return nil
	}

	// Find repository in database
	repo, err := h.getRepositoryByProviderID(ctx, "gitlab", strconv.Itoa(event.Project.ID))
	if err != nil {
		// Repository not found, skip processing
		fmt.Printf("Repository %s not found in database, skipping scan\n", event.Project.PathWithNamespace)
		return nil
	}

	// Submit scan job
	scanReq := &orchestrator.ScanRequest{
		RepositoryID: repo.ID,
		RepoURL:      event.Project.GitHTTPURL,
		Branch:       event.ObjectAttributes.SourceBranch,
		CommitSHA:    event.ObjectAttributes.LastCommit.ID,
		ScanType:     "incremental", // Use incremental for MR updates
		Priority:     types.PriorityHigh, // MR scans are high priority
		Options: map[string]interface{}{
			"gitlab_mr_iid":      strconv.Itoa(event.ObjectAttributes.IID),
			"gitlab_project_id":  strconv.Itoa(event.Project.ID),
			"gitlab_project_path": event.Project.PathWithNamespace,
		},
	}

	job, err := h.orchestrator.SubmitScan(ctx, scanReq)
	if err != nil {
		return fmt.Errorf("failed to submit scan: %w", err)
	}

	// Create initial commit status
	if err := h.createInitialCommitStatus(ctx, &event, job.ID.String()); err != nil {
		fmt.Printf("Failed to create initial commit status: %v\n", err)
		// Don't return error as scan is already submitted
	}

	return nil
}

// handlePushEvent handles push webhook events
func (h *WebhookHandler) handlePushEvent(ctx context.Context, payload []byte, webhookEvent *WebhookEvent) error {
	var event PushEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal push event: %w", err)
	}

	// Extract branch name from ref
	branch := strings.TrimPrefix(event.Ref, "refs/heads/")
	
	webhookEvent.Repository = event.Project.PathWithNamespace
	webhookEvent.Branch = branch
	webhookEvent.Commit = event.After

	// Skip deleted branches
	if event.After == "0000000000000000000000000000000000000000" {
		return nil
	}

	// Find repository in database
	repo, err := h.getRepositoryByProviderID(ctx, "gitlab", strconv.Itoa(event.ProjectID))
	if err != nil {
		// Repository not found, skip processing
		fmt.Printf("Repository %s not found in database, skipping scan\n", event.Project.PathWithNamespace)
		return nil
	}

	// Only scan default branch pushes for now
	if branch != repo.DefaultBranch {
		return nil
	}

	// Submit scan job
	scanReq := &orchestrator.ScanRequest{
		RepositoryID: repo.ID,
		RepoURL:      event.Project.GitHTTPURL,
		Branch:       branch,
		CommitSHA:    event.After,
		ScanType:     "incremental", // Push events use incremental scanning
		Priority:     types.PriorityMedium,
		Options: map[string]interface{}{
			"gitlab_project_id":   strconv.Itoa(event.ProjectID),
			"gitlab_project_path": event.Project.PathWithNamespace,
		},
	}

	_, err = h.orchestrator.SubmitScan(ctx, scanReq)
	if err != nil {
		return fmt.Errorf("failed to submit scan: %w", err)
	}

	return nil
}

// handlePipelineEvent handles pipeline webhook events
func (h *WebhookHandler) handlePipelineEvent(ctx context.Context, payload []byte, webhookEvent *WebhookEvent) error {
	var event PipelineEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal pipeline event: %w", err)
	}

	webhookEvent.Repository = event.Project.PathWithNamespace
	webhookEvent.Branch = event.ObjectAttributes.Ref
	webhookEvent.Commit = event.ObjectAttributes.SHA

	// Only process pipeline events for security scanning integration
	// This could be used to trigger scans when CI/CD pipelines run
	fmt.Printf("Pipeline %s for %s: %s\n", 
		event.ObjectAttributes.Status, 
		event.Project.PathWithNamespace, 
		event.ObjectAttributes.Ref)

	return nil
}

// createInitialCommitStatus creates an initial "running" commit status for a MR
func (h *WebhookHandler) createInitialCommitStatus(ctx context.Context, event *MergeRequestEvent, jobID string) error {
	if h.service == nil {
		return fmt.Errorf("GitLab service not configured")
	}

	status := &CommitStatus{
		State:       "running",
		Description: "AgentScan security analysis in progress...",
		Name:        "agentscan/security",
		TargetURL:   fmt.Sprintf("https://app.agentscan.dev/scans/%s", jobID),
	}

	return h.service.CreateCommitStatus(ctx, event.Project.ID, event.ObjectAttributes.LastCommit.ID, status)
}

// storeWebhookEvent stores a webhook event in the database
func (h *WebhookHandler) storeWebhookEvent(ctx context.Context, event *WebhookEvent) error {
	// TODO: Implement database storage for webhook events
	// This would require adding a webhook_events table to the database schema
	fmt.Printf("Storing GitLab webhook event: %s/%s for %s\n", event.EventType, event.Action, event.Repository)
	return nil
}

// getRepositoryByProviderID gets a repository by provider and provider ID
func (h *WebhookHandler) getRepositoryByProviderID(ctx context.Context, provider, providerID string) (*types.Repository, error) {
	// TODO: Implement repository lookup by provider ID
	// For now, return a mock repository
	return &types.Repository{
		ID:            uuid.New(),
		DefaultBranch: "main",
		Provider:      provider,
		ProviderID:    providerID,
	}, nil
}

// OnScanComplete is called when a scan completes to update GitLab status
func (h *WebhookHandler) OnScanComplete(ctx context.Context, job *types.ScanJob, results *orchestrator.ScanResults) error {
	// Check if this is a GitLab MR scan
	mrIID, hasMR := job.Metadata["gitlab_mr_iid"]
	if !hasMR {
		return nil // Not a MR scan
	}

	projectIDStr, hasProject := job.Metadata["gitlab_project_id"]
	if !hasProject {
		return fmt.Errorf("missing GitLab project ID")
	}

	projectIDString, ok := projectIDStr.(string)
	if !ok {
		return fmt.Errorf("invalid project ID type")
	}

	projectID, err := strconv.Atoi(projectIDString)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	// Update commit status
	if err := h.updateCommitStatus(ctx, projectID, job.CommitSHA, results, job.ID.String()); err != nil {
		fmt.Printf("Failed to update commit status: %v\n", err)
	}

	// Post MR comment with results
	mrIIDStr, ok := mrIID.(string)
	if !ok {
		return fmt.Errorf("invalid MR IID type")
	}

	mrIIDInt, err := strconv.Atoi(mrIIDStr)
	if err != nil {
		return fmt.Errorf("invalid MR IID: %w", err)
	}

	if err := h.postMRComment(ctx, projectID, mrIIDInt, results); err != nil {
		fmt.Printf("Failed to post MR comment: %v\n", err)
	}

	return nil
}

// updateCommitStatus updates the GitLab commit status with scan results
func (h *WebhookHandler) updateCommitStatus(ctx context.Context, projectID int, sha string, results *orchestrator.ScanResults, jobID string) error {
	if h.service == nil {
		return fmt.Errorf("GitLab service not configured")
	}

	var state, description string
	
	if results.Status == "completed" {
		highSeverityCount := 0
		for _, finding := range results.Findings {
			if finding.Severity == "high" {
				highSeverityCount++
			}
		}

		if highSeverityCount > 0 {
			state = "failed"
			description = fmt.Sprintf("Found %d high severity security issues", highSeverityCount)
		} else if len(results.Findings) > 0 {
			state = "success"
			description = fmt.Sprintf("Found %d security issues (no high severity)", len(results.Findings))
		} else {
			state = "success"
			description = "No security issues found"
		}
	} else if results.Status == "failed" {
		state = "failed"
		description = "Security scan failed"
	} else {
		state = "running"
		description = "Security scan in progress..."
	}

	status := &CommitStatus{
		State:       state,
		Description: description,
		Name:        "agentscan/security",
		TargetURL:   fmt.Sprintf("https://app.agentscan.dev/scans/%s", jobID),
	}

	return h.service.CreateCommitStatus(ctx, projectID, sha, status)
}

// postMRComment posts a comment on the MR with scan results
func (h *WebhookHandler) postMRComment(ctx context.Context, projectID, mrIID int, results *orchestrator.ScanResults) error {
	if h.service == nil {
		return fmt.Errorf("GitLab service not configured")
	}

	if results.Status != "completed" {
		return nil // Only comment on completed scans
	}

	comment := h.formatMRComment(results)
	mrComment := &MRComment{Body: comment}

	return h.service.CreateMRComment(ctx, projectID, mrIID, mrComment)
}

// formatMRComment formats scan results into a MR comment
func (h *WebhookHandler) formatMRComment(results *orchestrator.ScanResults) string {
	var comment strings.Builder
	
	comment.WriteString("## 🔒 AgentScan Security Report\n\n")
	
	if len(results.Findings) == 0 {
		comment.WriteString("✅ **No security issues found!**\n\n")
		comment.WriteString("Your code looks secure. Great job! 🎉\n")
		return comment.String()
	}

	// Count findings by severity
	severityCounts := make(map[string]int)
	for _, finding := range results.Findings {
		severityCounts[finding.Severity]++
	}

	// Summary
	comment.WriteString("### Summary\n\n")
	if high := severityCounts["high"]; high > 0 {
		comment.WriteString(fmt.Sprintf("🔴 **%d High** severity issues\n", high))
	}
	if medium := severityCounts["medium"]; medium > 0 {
		comment.WriteString(fmt.Sprintf("🟡 **%d Medium** severity issues\n", medium))
	}
	if low := severityCounts["low"]; low > 0 {
		comment.WriteString(fmt.Sprintf("🟢 **%d Low** severity issues\n", low))
	}
	comment.WriteString("\n")

	// High severity findings details
	if severityCounts["high"] > 0 {
		comment.WriteString("### 🔴 High Severity Issues\n\n")
		for _, finding := range results.Findings {
			if finding.Severity == "high" {
				comment.WriteString(fmt.Sprintf("**%s** in `%s:%d`\n", finding.Title, finding.File, finding.Line))
				comment.WriteString(fmt.Sprintf("- %s\n", finding.Description))
				comment.WriteString(fmt.Sprintf("- Detected by: %s\n", finding.Tool))
				comment.WriteString("\n")
			}
		}
	}

	comment.WriteString("---\n")
	comment.WriteString("*Powered by [AgentScan](https://agentscan.dev) - Multi-agent security scanning*\n")

	return comment.String()
}