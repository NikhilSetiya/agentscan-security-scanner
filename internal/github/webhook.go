package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

// WebhookHandler handles GitHub webhook events
type WebhookHandler struct {
	repos        *database.Repositories
	orchestrator orchestrator.OrchestrationService
	service      *Service
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(repos *database.Repositories, orchestrator orchestrator.OrchestrationService, service *Service) *WebhookHandler {
	return &WebhookHandler{
		repos:        repos,
		orchestrator: orchestrator,
		service:      service,
	}
}

// HandleWebhook handles incoming GitHub webhook events
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read the payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read payload", http.StatusBadRequest)
		return
	}

	// Get event type and delivery ID
	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")
	signature := r.Header.Get("X-Hub-Signature-256")

	if eventType == "" {
		http.Error(w, "Missing X-GitHub-Event header", http.StatusBadRequest)
		return
	}

	// Verify webhook signature (skip if no signature provided for testing)
	if signature != "" {
		if err := h.verifySignature(payload, signature, ""); err != nil {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
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
	case "pull_request":
		if err := h.handlePullRequestEvent(ctx, payload, webhookEvent); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle pull request event: %v", err), http.StatusInternalServerError)
			return
		}
	case "push":
		if err := h.handlePushEvent(ctx, payload, webhookEvent); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle push event: %v", err), http.StatusInternalServerError)
			return
		}
	case "installation":
		if err := h.handleInstallationEvent(ctx, payload, webhookEvent); err != nil {
			http.Error(w, fmt.Sprintf("Failed to handle installation event: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		// Log unsupported event types but don't error
		fmt.Printf("Received unsupported GitHub event: %s (delivery: %s)\n", eventType, deliveryID)
	}

	// Mark event as processed
	now := time.Now()
	webhookEvent.ProcessedAt = &now

	// Store the webhook event in database
	if err := h.storeWebhookEvent(ctx, webhookEvent); err != nil {
		fmt.Printf("Failed to store webhook event: %v\n", err)
		// Don't return error to GitHub as the event was processed
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// verifySignature verifies the GitHub webhook signature
func (h *WebhookHandler) verifySignature(payload []byte, signature, secret string) error {
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("invalid signature format")
	}

	expectedSignature := signature[7:] // Remove "sha256=" prefix
	
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	actualSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSignature), []byte(actualSignature)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// handlePullRequestEvent handles pull request webhook events
func (h *WebhookHandler) handlePullRequestEvent(ctx context.Context, payload []byte, webhookEvent *WebhookEvent) error {
	var event PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal pull request event: %w", err)
	}

	webhookEvent.Action = event.Action
	webhookEvent.Repository = event.Repository.FullName
	webhookEvent.PullRequest = &event.PullRequest.Number
	webhookEvent.Branch = event.PullRequest.Head.Ref
	webhookEvent.Commit = event.PullRequest.Head.SHA

	// Only process opened, synchronize, and reopened events
	if event.Action != "opened" && event.Action != "synchronize" && event.Action != "reopened" {
		return nil
	}

	// Find repository in database
	repo, err := h.getRepositoryByProviderID(ctx, "github", strconv.Itoa(event.Repository.ID))
	if err != nil {
		// Repository not found, skip processing
		fmt.Printf("Repository %s not found in database, skipping scan\n", event.Repository.FullName)
		return nil
	}

	// Submit scan job
	scanReq := &orchestrator.ScanRequest{
		RepositoryID: repo.ID,
		RepoURL:      event.Repository.CloneURL,
		Branch:       event.PullRequest.Head.Ref,
		CommitSHA:    event.PullRequest.Head.SHA,
		ScanType:     "incremental", // Use incremental for PR updates
		Priority:     types.PriorityHigh,     // PR scans are high priority
		Options: map[string]interface{}{
			"github_pr_number":      strconv.Itoa(event.PullRequest.Number),
			"github_installation_id": strconv.FormatInt(event.Installation.ID, 10),
			"github_repo_owner":     event.Repository.Owner.Login,
			"github_repo_name":      event.Repository.Name,
		},
	}

	job, err := h.orchestrator.SubmitScan(ctx, scanReq)
	if err != nil {
		return fmt.Errorf("failed to submit scan: %w", err)
	}

	// Create initial status check
	if err := h.createInitialStatusCheck(ctx, &event, job.ID.String()); err != nil {
		fmt.Printf("Failed to create initial status check: %v\n", err)
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
	
	webhookEvent.Repository = event.Repository.FullName
	webhookEvent.Branch = branch
	webhookEvent.Commit = event.After

	// Skip deleted branches
	if event.After == "0000000000000000000000000000000000000000" {
		return nil
	}

	// Find repository in database
	repo, err := h.getRepositoryByProviderID(ctx, "github", strconv.Itoa(event.Repository.ID))
	if err != nil {
		// Repository not found, skip processing
		fmt.Printf("Repository %s not found in database, skipping scan\n", event.Repository.FullName)
		return nil
	}

	// Only scan default branch pushes for now
	if branch != repo.DefaultBranch {
		return nil
	}

	// Submit scan job
	scanReq := &orchestrator.ScanRequest{
		RepositoryID: repo.ID,
		RepoURL:      event.Repository.CloneURL,
		Branch:       branch,
		CommitSHA:    event.After,
		ScanType:     "incremental", // Push events use incremental scanning
		Priority:     types.PriorityMedium,
		Options: map[string]interface{}{
			"github_installation_id": strconv.FormatInt(event.Installation.ID, 10),
			"github_repo_owner":     event.Repository.Owner.Login,
			"github_repo_name":      event.Repository.Name,
		},
	}

	_, err = h.orchestrator.SubmitScan(ctx, scanReq)
	if err != nil {
		return fmt.Errorf("failed to submit scan: %w", err)
	}

	return nil
}

// handleInstallationEvent handles installation webhook events
func (h *WebhookHandler) handleInstallationEvent(ctx context.Context, payload []byte, webhookEvent *WebhookEvent) error {
	var event struct {
		Action       string `json:"action"`
		Installation struct {
			ID      int64 `json:"id"`
			Account struct {
				Login string `json:"login"`
				Type  string `json:"type"`
			} `json:"account"`
		} `json:"installation"`
		Repositories []struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			FullName string `json:"full_name"`
		} `json:"repositories"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal installation event: %w", err)
	}

	webhookEvent.Action = event.Action

	switch event.Action {
	case "created":
		// Handle new installation
		fmt.Printf("GitHub App installed for %s (ID: %d)\n", event.Installation.Account.Login, event.Installation.ID)
		
		// TODO: Store installation in database and sync repositories
		// This would involve creating organization records and repository records
		
	case "deleted":
		// Handle installation removal
		fmt.Printf("GitHub App uninstalled for %s (ID: %d)\n", event.Installation.Account.Login, event.Installation.ID)
		
		// TODO: Clean up installation data from database
		
	case "repositories_added", "repositories_removed":
		// Handle repository access changes
		fmt.Printf("Repository access changed for installation %d: %s\n", event.Installation.ID, event.Action)
		
		// TODO: Sync repository access changes
	}

	return nil
}

// createInitialStatusCheck creates an initial "pending" status check for a PR
func (h *WebhookHandler) createInitialStatusCheck(ctx context.Context, event *PullRequestEvent, jobID string) error {
	// Get GitHub client for this installation
	client, err := h.service.GetClientForInstallation(ctx, event.Installation.ID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	status := &StatusCheck{
		State:       "pending",
		Description: "AgentScan security analysis in progress...",
		Context:     "agentscan/security",
		TargetURL:   fmt.Sprintf("https://app.agentscan.dev/scans/%s", jobID),
	}

	return client.CreateStatusCheck(ctx, event.Repository.Owner.Login, event.Repository.Name, event.PullRequest.Head.SHA, status)
}

// storeWebhookEvent stores a webhook event in the database
func (h *WebhookHandler) storeWebhookEvent(ctx context.Context, event *WebhookEvent) error {
	// TODO: Implement database storage for webhook events
	// This would require adding a webhook_events table to the database schema
	fmt.Printf("Storing webhook event: %s/%s for %s\n", event.EventType, event.Action, event.Repository)
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

// OnScanComplete is called when a scan completes to update GitHub status
func (h *WebhookHandler) OnScanComplete(ctx context.Context, job *types.ScanJob, results *orchestrator.ScanResults) error {
	// Check if this is a GitHub PR scan
	prNumber, hasPR := job.Metadata["github_pr_number"]
	if !hasPR {
		return nil // Not a PR scan
	}

	installationIDStr, hasInstallation := job.Metadata["github_installation_id"]
	if !hasInstallation {
		return fmt.Errorf("missing GitHub installation ID")
	}

	installationIDString, ok := installationIDStr.(string)
	if !ok {
		return fmt.Errorf("invalid installation ID type")
	}

	installationID, err := strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid installation ID: %w", err)
	}

	repoOwner, hasOwner := job.Metadata["github_repo_owner"]
	repoName, hasName := job.Metadata["github_repo_name"]
	if !hasOwner || !hasName {
		return fmt.Errorf("missing repository information")
	}

	repoOwnerStr, ok := repoOwner.(string)
	if !ok {
		return fmt.Errorf("invalid repo owner type")
	}

	repoNameStr, ok := repoName.(string)
	if !ok {
		return fmt.Errorf("invalid repo name type")
	}

	// Get GitHub client
	client, err := h.service.GetClientForInstallation(ctx, installationID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// Update status check
	if err := h.updateStatusCheck(ctx, client, repoOwnerStr, repoNameStr, job.CommitSHA, results, job.ID.String()); err != nil {
		fmt.Printf("Failed to update status check: %v\n", err)
	}

	// Post PR comment with results
	prNumberStr, ok := prNumber.(string)
	if !ok {
		return fmt.Errorf("invalid PR number type")
	}

	prNum, err := strconv.Atoi(prNumberStr)
	if err != nil {
		return fmt.Errorf("invalid PR number: %w", err)
	}

	if err := h.postPRComment(ctx, client, repoOwnerStr, repoNameStr, prNum, results); err != nil {
		fmt.Printf("Failed to post PR comment: %v\n", err)
	}

	return nil
}

// updateStatusCheck updates the GitHub status check with scan results
func (h *WebhookHandler) updateStatusCheck(ctx context.Context, client *Client, owner, repo, sha string, results *orchestrator.ScanResults, jobID string) error {
	var state, description string
	
	if results.Status == "completed" {
		highSeverityCount := 0
		for _, finding := range results.Findings {
			if finding.Severity == "high" {
				highSeverityCount++
			}
		}

		if highSeverityCount > 0 {
			state = "failure"
			description = fmt.Sprintf("Found %d high severity security issues", highSeverityCount)
		} else if len(results.Findings) > 0 {
			state = "success"
			description = fmt.Sprintf("Found %d security issues (no high severity)", len(results.Findings))
		} else {
			state = "success"
			description = "No security issues found"
		}
	} else if results.Status == "failed" {
		state = "error"
		description = "Security scan failed"
	} else {
		state = "pending"
		description = "Security scan in progress..."
	}

	status := &StatusCheck{
		State:       state,
		Description: description,
		Context:     "agentscan/security",
		TargetURL:   fmt.Sprintf("https://app.agentscan.dev/scans/%s", jobID),
	}

	return client.CreateStatusCheck(ctx, owner, repo, sha, status)
}

// postPRComment posts a comment on the PR with scan results
func (h *WebhookHandler) postPRComment(ctx context.Context, client *Client, owner, repo string, prNumber int, results *orchestrator.ScanResults) error {
	if results.Status != "completed" {
		return nil // Only comment on completed scans
	}

	comment := h.formatPRComment(results)
	prComment := &PRComment{Body: comment}

	return client.CreatePRComment(ctx, owner, repo, prNumber, prComment)
}

// formatPRComment formats scan results into a PR comment
func (h *WebhookHandler) formatPRComment(results *orchestrator.ScanResults) string {
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