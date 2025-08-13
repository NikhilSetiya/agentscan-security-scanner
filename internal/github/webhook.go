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

	// Get changed files for PR-specific scanning
	client, err := h.service.GetClientForInstallation(ctx, event.Installation.ID)
	if err != nil {
		fmt.Printf("Failed to get GitHub client for changed files: %v\n", err)
		// Continue without changed files - will do full scan
	}

	var changedFiles []string
	if client != nil {
		changedFiles, err = h.getPRChangedFiles(ctx, client, event.Repository.Owner.Login, event.Repository.Name, event.PullRequest.Number)
		if err != nil {
			fmt.Printf("Failed to get PR changed files: %v\n", err)
			// Continue without changed files - will do full scan
		}
	}

	// Submit scan job with PR-specific options
	scanReq := &orchestrator.ScanRequest{
		RepositoryID: repo.ID,
		RepoURL:      event.Repository.CloneURL,
		Branch:       event.PullRequest.Head.Ref,
		CommitSHA:    event.PullRequest.Head.SHA,
		BaseSHA:      event.PullRequest.Base.SHA, // Add base SHA for comparison
		ScanType:     "incremental", // Use incremental for PR updates
		Priority:     types.PriorityHigh,     // PR scans are high priority
		ChangedFiles: changedFiles,   // Only scan changed files
		Options: map[string]interface{}{
			"github_pr_number":      strconv.Itoa(event.PullRequest.Number),
			"github_installation_id": strconv.FormatInt(event.Installation.ID, 10),
			"github_repo_owner":     event.Repository.Owner.Login,
			"github_repo_name":      event.Repository.Name,
			"is_pr_scan":           "true",
			"pr_base_sha":          event.PullRequest.Base.SHA,
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

// getPRChangedFiles gets the list of files changed in a pull request
func (h *WebhookHandler) getPRChangedFiles(ctx context.Context, client *Client, owner, repo string, prNumber int) ([]string, error) {
	files, err := client.GetPRFiles(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR files: %w", err)
	}

	var changedFiles []string
	for _, file := range files {
		// Only include files that are added or modified (not deleted)
		if file.Status == "added" || file.Status == "modified" || file.Status == "renamed" {
			changedFiles = append(changedFiles, file.Filename)
		}
	}

	return changedFiles, nil
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

	if err := h.postPRComment(ctx, client, repoOwnerStr, repoNameStr, prNum, results, job.ID.String()); err != nil {
		fmt.Printf("Failed to post PR comment: %v\n", err)
	}

	return nil
}

// updateStatusCheck updates the GitHub status check with scan results
func (h *WebhookHandler) updateStatusCheck(ctx context.Context, client *Client, owner, repo, sha string, results *orchestrator.ScanResults, jobID string) error {
	var state, description string
	
	if results.Status == "completed" {
		// Count findings by severity
		severityCounts := make(map[string]int)
		newFindings := make(map[string]int)
		
		for _, finding := range results.Findings {
			severityCounts[finding.Severity]++
			
			// Check if this is a new finding in the PR
			if finding.Metadata != nil {
				if isNew, exists := finding.Metadata["is_new_in_pr"]; exists && isNew == "true" {
					newFindings[finding.Severity]++
				}
			}
		}

		// Determine status based on severity and whether issues are new
		if severityCounts["high"] > 0 {
			state = "failure"
			if newFindings["high"] > 0 {
				description = fmt.Sprintf("🚨 %d high severity issues found (%d new) - Fix before merging", severityCounts["high"], newFindings["high"])
			} else {
				description = fmt.Sprintf("🚨 %d high severity issues found - Fix before merging", severityCounts["high"])
			}
		} else if severityCounts["medium"] > 0 || severityCounts["low"] > 0 {
			state = "success"
			totalIssues := severityCounts["medium"] + severityCounts["low"]
			newIssues := newFindings["medium"] + newFindings["low"]
			if newIssues > 0 {
				description = fmt.Sprintf("✅ %d security issues found (%d new) - No high severity", totalIssues, newIssues)
			} else {
				description = fmt.Sprintf("✅ %d security issues found - No high severity", totalIssues)
			}
		} else {
			state = "success"
			description = "✅ No security issues found - All clear!"
		}
	} else if results.Status == "failed" {
		state = "error"
		description = "❌ Security scan failed - Check logs for details"
	} else {
		state = "pending"
		description = "🔍 Security scan in progress..."
	}

	// Create both status check and check run for better GitHub integration
	status := &StatusCheck{
		State:       state,
		Description: description,
		Context:     "agentscan/security",
		TargetURL:   fmt.Sprintf("https://app.agentscan.dev/scans/%s", jobID),
	}

	// Create status check
	if err := client.CreateStatusCheck(ctx, owner, repo, sha, status); err != nil {
		return fmt.Errorf("failed to create status check: %w", err)
	}

	// Also create/update check run for richer display
	return h.updateCheckRun(ctx, client, owner, repo, sha, results, jobID)
}

// updateCheckRun creates or updates a GitHub check run with detailed results
func (h *WebhookHandler) updateCheckRun(ctx context.Context, client *Client, owner, repo, sha string, results *orchestrator.ScanResults, jobID string) error {
	var status, conclusion string
	var output *CheckRunOutput

	if results.Status == "completed" {
		status = "completed"
		
		// Count findings by severity
		severityCounts := make(map[string]int)
		for _, finding := range results.Findings {
			severityCounts[finding.Severity]++
		}

		// Determine conclusion and create detailed output
		if severityCounts["high"] > 0 {
			conclusion = "failure"
			output = &CheckRunOutput{
				Title:   fmt.Sprintf("🚨 Security Issues Found - %d High Severity", severityCounts["high"]),
				Summary: h.generateCheckRunSummary(results, severityCounts),
				Annotations: h.generateCheckRunAnnotations(results),
			}
		} else if len(results.Findings) > 0 {
			conclusion = "neutral"
			totalIssues := severityCounts["medium"] + severityCounts["low"]
			output = &CheckRunOutput{
				Title:   fmt.Sprintf("✅ Security Scan Complete - %d Issues (No High Severity)", totalIssues),
				Summary: h.generateCheckRunSummary(results, severityCounts),
				Annotations: h.generateCheckRunAnnotations(results),
			}
		} else {
			conclusion = "success"
			output = &CheckRunOutput{
				Title:   "✅ Security Scan Complete - No Issues Found",
				Summary: "🎉 Your code is secure! No security vulnerabilities were detected.",
			}
		}
	} else if results.Status == "failed" {
		status = "completed"
		conclusion = "failure"
		output = &CheckRunOutput{
			Title:   "❌ Security Scan Failed",
			Summary: "The security scan encountered an error. Please check the logs and try again.",
		}
	} else {
		status = "in_progress"
		output = &CheckRunOutput{
			Title:   "🔍 Security Scan in Progress",
			Summary: "AgentScan is analyzing your code for security vulnerabilities...",
		}
	}

	checkRun := &CheckRun{
		Name:       "AgentScan Security",
		HeadSHA:    sha,
		Status:     status,
		Conclusion: conclusion,
		Output:     output,
	}

	// For now, create a new check run each time
	// TODO: Store check run ID to update existing one
	return client.CreateCheckRun(ctx, owner, repo, checkRun)
}

// generateCheckRunSummary generates a summary for the check run
func (h *WebhookHandler) generateCheckRunSummary(results *orchestrator.ScanResults, severityCounts map[string]int) string {
	var summary strings.Builder
	
	summary.WriteString("## Security Scan Results\n\n")
	
	if len(results.Findings) == 0 {
		summary.WriteString("🎉 **No security issues found!** Your code looks secure.\n\n")
		return summary.String()
	}

	summary.WriteString("### Issue Summary\n\n")
	if high := severityCounts["high"]; high > 0 {
		summary.WriteString(fmt.Sprintf("- 🔴 **%d High** severity issues\n", high))
	}
	if medium := severityCounts["medium"]; medium > 0 {
		summary.WriteString(fmt.Sprintf("- 🟡 **%d Medium** severity issues\n", medium))
	}
	if low := severityCounts["low"]; low > 0 {
		summary.WriteString(fmt.Sprintf("- 🟢 **%d Low** severity issues\n", low))
	}

	summary.WriteString("\n### Next Steps\n\n")
	if severityCounts["high"] > 0 {
		summary.WriteString("⚠️ **High severity issues must be fixed before merging.**\n\n")
	}
	summary.WriteString("📊 View detailed results and fix suggestions in the AgentScan dashboard.\n")

	return summary.String()
}

// generateCheckRunAnnotations generates annotations for the check run (max 50)
func (h *WebhookHandler) generateCheckRunAnnotations(results *orchestrator.ScanResults) []CheckRunAnnotation {
	var annotations []CheckRunAnnotation
	
	// Prioritize high severity findings
	highSeverityCount := 0
	for _, finding := range results.Findings {
		if finding.Severity == "high" && len(annotations) < 50 {
			level := "failure"
			if finding.Severity == "medium" {
				level = "warning"
			} else if finding.Severity == "low" {
				level = "notice"
			}

			annotation := CheckRunAnnotation{
				Path:            finding.File,
				StartLine:       finding.Line,
				EndLine:         finding.Line,
				AnnotationLevel: level,
				Message:         fmt.Sprintf("%s: %s", finding.RuleID, finding.Description),
				Title:           finding.Title,
			}

			annotations = append(annotations, annotation)
			highSeverityCount++
		}
	}

	// Add medium/low severity if we have space
	for _, finding := range results.Findings {
		if finding.Severity != "high" && len(annotations) < 50 {
			level := "warning"
			if finding.Severity == "low" {
				level = "notice"
			}

			annotation := CheckRunAnnotation{
				Path:            finding.File,
				StartLine:       finding.Line,
				EndLine:         finding.Line,
				AnnotationLevel: level,
				Message:         fmt.Sprintf("%s: %s", finding.RuleID, finding.Description),
				Title:           finding.Title,
			}

			annotations = append(annotations, annotation)
		}
	}

	return annotations
}

// postPRComment posts or updates a comment on the PR with scan results
func (h *WebhookHandler) postPRComment(ctx context.Context, client *Client, owner, repo string, prNumber int, results *orchestrator.ScanResults, jobID string) error {
	if results.Status != "completed" {
		return nil // Only comment on completed scans
	}

	comment := h.formatPRComment(results, jobID)
	
	// Try to find existing AgentScan comment to update instead of creating new one
	existingCommentID, err := h.findExistingAgentScanComment(ctx, client, owner, repo, prNumber)
	if err != nil {
		fmt.Printf("Failed to find existing comment: %v\n", err)
		// Continue with creating new comment
	}

	if existingCommentID > 0 {
		// Update existing comment
		return client.UpdatePRComment(ctx, owner, repo, existingCommentID, &PRComment{Body: comment})
	} else {
		// Create new comment
		return client.CreatePRComment(ctx, owner, repo, prNumber, &PRComment{Body: comment})
	}
}

// findExistingAgentScanComment finds an existing AgentScan comment on the PR
func (h *WebhookHandler) findExistingAgentScanComment(ctx context.Context, client *Client, owner, repo string, prNumber int) (int64, error) {
	comments, err := client.GetPRComments(ctx, owner, repo, prNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to get PR comments: %w", err)
	}

	// Look for comment that contains AgentScan signature
	for _, comment := range comments {
		if strings.Contains(comment.Body, "🛡️ AgentScan Security Report") {
			return comment.ID, nil
		}
	}

	return 0, nil // No existing comment found
}

// formatPRComment formats scan results into a rich, actionable PR comment
func (h *WebhookHandler) formatPRComment(results *orchestrator.ScanResults, jobID string) string {
	var comment strings.Builder
	
	// Header with scan status and link to detailed results
	comment.WriteString("## 🛡️ AgentScan Security Report\n\n")
	
	if len(results.Findings) == 0 {
		comment.WriteString("### ✅ All Clear!\n\n")
		comment.WriteString("**No security vulnerabilities detected** in your changes. Your code looks secure! 🎉\n\n")
		comment.WriteString(fmt.Sprintf("📊 [View detailed scan results](%s/scans/%s)\n\n", "https://app.agentscan.dev", jobID))
		comment.WriteString("---\n")
		comment.WriteString("*🔒 Secured by [AgentScan](https://agentscan.dev) - Multi-agent security scanning with intelligent deduplication*\n")
		return comment.String()
	}

	// Count findings by severity and new vs existing
	severityCounts := make(map[string]int)
	newFindings := make(map[string]int)
	toolCoverage := make(map[string]bool)
	
	for _, finding := range results.Findings {
		severityCounts[finding.Severity]++
		toolCoverage[finding.Tool] = true
		
		// Check if this is a new finding in the PR (this would need to be set during scanning)
		if finding.Metadata != nil {
			if isNew, exists := finding.Metadata["is_new_in_pr"]; exists && isNew == "true" {
				newFindings[finding.Severity]++
			}
		}
	}

	// Summary with visual indicators
	comment.WriteString("### 📊 Security Summary\n\n")
	comment.WriteString("| Severity | Count | New in PR |\n")
	comment.WriteString("|----------|-------|----------|\n")
	
	if high := severityCounts["high"]; high > 0 {
		newHigh := newFindings["high"]
		comment.WriteString(fmt.Sprintf("| 🔴 **High** | **%d** | %d |\n", high, newHigh))
	}
	if medium := severityCounts["medium"]; medium > 0 {
		newMedium := newFindings["medium"]
		comment.WriteString(fmt.Sprintf("| 🟡 **Medium** | **%d** | %d |\n", medium, newMedium))
	}
	if low := severityCounts["low"]; low > 0 {
		newLow := newFindings["low"]
		comment.WriteString(fmt.Sprintf("| 🟢 **Low** | **%d** | %d |\n", low, newLow))
	}
	
	comment.WriteString("\n")

	// Tool coverage information
	tools := make([]string, 0, len(toolCoverage))
	for tool := range toolCoverage {
		tools = append(tools, tool)
	}
	comment.WriteString(fmt.Sprintf("**🔍 Scanned with:** %s\n\n", strings.Join(tools, ", ")))

	// Critical issues that need immediate attention
	if severityCounts["high"] > 0 {
		comment.WriteString("### 🚨 Critical Issues Requiring Attention\n\n")
		comment.WriteString("> **⚠️ High severity vulnerabilities detected.** Please review and fix before merging.\n\n")
		
		highCount := 0
		for _, finding := range results.Findings {
			if finding.Severity == "high" && highCount < 3 { // Show max 3 high severity issues
				comment.WriteString(fmt.Sprintf("#### %s\n", finding.Title))
				comment.WriteString(fmt.Sprintf("**📁 File:** `%s:%d`\n", finding.File, finding.Line))
				comment.WriteString(fmt.Sprintf("**🔍 Rule:** `%s`\n", finding.RuleID))
				comment.WriteString(fmt.Sprintf("**🛠️ Tool:** %s", finding.Tool))
				
				// Add confidence score if available
				if finding.Confidence > 0 {
					comment.WriteString(fmt.Sprintf(" (Confidence: %.0f%%)", finding.Confidence*100))
				}
				comment.WriteString("\n\n")
				
				comment.WriteString(fmt.Sprintf("**📝 Description:** %s\n\n", finding.Description))
				
				// Add fix suggestion if available
				if finding.FixSuggestion != nil {
					if desc, ok := finding.FixSuggestion["description"].(string); ok && desc != "" {
						comment.WriteString(fmt.Sprintf("**💡 Suggested Fix:** %s\n\n", desc))
					}
				}
				
				// Add references if available
				if len(finding.References) > 0 {
					comment.WriteString("**📚 References:**\n")
					for _, ref := range finding.References {
						comment.WriteString(fmt.Sprintf("- %s\n", ref))
					}
					comment.WriteString("\n")
				}
				
				comment.WriteString("---\n\n")
				highCount++
			}
		}
		
		if severityCounts["high"] > 3 {
			remaining := severityCounts["high"] - 3
			comment.WriteString(fmt.Sprintf("*... and %d more high severity issues. [View all findings](%s/scans/%s)*\n\n", remaining, "https://app.agentscan.dev", jobID))
		}
	}

	// Medium severity summary (collapsed by default)
	if severityCounts["medium"] > 0 {
		comment.WriteString("### 🟡 Medium Severity Issues\n\n")
		comment.WriteString("<details>\n")
		comment.WriteString(fmt.Sprintf("<summary>%d medium severity issues found (click to expand)</summary>\n\n", severityCounts["medium"]))
		
		mediumCount := 0
		for _, finding := range results.Findings {
			if finding.Severity == "medium" && mediumCount < 5 { // Show max 5 medium issues
				comment.WriteString(fmt.Sprintf("- **%s** in `%s:%d` - %s\n", finding.Title, finding.File, finding.Line, finding.Tool))
				mediumCount++
			}
		}
		
		if severityCounts["medium"] > 5 {
			remaining := severityCounts["medium"] - 5
			comment.WriteString(fmt.Sprintf("- *... and %d more medium severity issues*\n", remaining))
		}
		
		comment.WriteString("\n</details>\n\n")
	}

	// Action items and next steps
	comment.WriteString("### 🎯 Next Steps\n\n")
	
	if severityCounts["high"] > 0 {
		comment.WriteString("1. **🔴 Fix high severity issues** before merging this PR\n")
		comment.WriteString("2. 📊 [Review detailed findings](%s/scans/%s) in the AgentScan dashboard\n")
		comment.WriteString("3. 💬 Comment `/agentscan rescan` to re-run the security scan after fixes\n\n")
	} else if severityCounts["medium"] > 0 {
		comment.WriteString("1. 🟡 Consider addressing medium severity issues\n")
		comment.WriteString("2. 📊 [Review detailed findings](%s/scans/%s) in the AgentScan dashboard\n")
		comment.WriteString("3. ✅ No blocking issues - safe to merge\n\n")
	}

	// Footer with branding and links
	comment.WriteString("---\n")
	comment.WriteString("📊 **[View Full Report](%s/scans/%s)** | ")
	comment.WriteString("📖 **[Documentation](https://docs.agentscan.dev)** | ")
	comment.WriteString("🐛 **[Report Issue](https://github.com/agentscan/agentscan/issues)**\n\n")
	comment.WriteString("*🛡️ Secured by [AgentScan](https://agentscan.dev) - Multi-agent security scanning with 80% fewer false positives*\n")

	return fmt.Sprintf(comment.String(), "https://app.agentscan.dev", jobID, "https://app.agentscan.dev", jobID, "https://app.agentscan.dev", jobID)
}