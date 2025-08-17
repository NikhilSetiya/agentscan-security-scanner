package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// Worker processes scan jobs from the queue
type Worker struct {
	id           string
	queue        queue.QueueInterface
	agentManager *AgentManager
	db           database.Repository
	
	// Statistics
	jobsProcessed int64
	jobsFailed    int64
	lastJobAt     *time.Time
	startTime     time.Time
}

// NewWorker creates a new worker
func NewWorker(id string, queue queue.QueueInterface, agentManager *AgentManager, db database.Repository) *Worker {
	return &Worker{
		id:           id,
		queue:        queue,
		agentManager: agentManager,
		db:           db,
		startTime:    time.Now(),
	}
}

// Start starts the worker processing loop
func (w *Worker) Start(ctx context.Context, stopCh <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		default:
			// Try to get a job from the queue
			job, err := w.queue.Dequeue(ctx, w.id)
			if err != nil {
				if !errors.IsType(err, errors.ErrorTypeNotFound) {
					// Log error but continue
					// TODO: Add proper logging
				}
				// No job available, wait a bit before trying again
				time.Sleep(1 * time.Second)
				continue
			}

			// Process the job
			w.processJob(ctx, job)
		}
	}
}

// processJob processes a single scan job
func (w *Worker) processJob(ctx context.Context, job *queue.Job) {
	startTime := time.Now()
	w.lastJobAt = &startTime

	// Parse job payload
	scanJobID, err := w.parseScanJobID(job)
	if err != nil {
		w.failJob(ctx, job, fmt.Sprintf("failed to parse job payload: %v", err))
		return
	}

	// Get scan job from database
	scanJob, err := w.db.GetScanJob(ctx, scanJobID)
	if err != nil {
		w.failJob(ctx, job, fmt.Sprintf("failed to get scan job: %v", err))
		return
	}

	// Update scan job status to running
	scanJob.Status = types.ScanJobStatusRunning
	scanJob.StartedAt = &startTime
	scanJob.UpdatedAt = time.Now()

	if err := w.db.UpdateScanJob(ctx, scanJob); err != nil {
		w.failJob(ctx, job, fmt.Sprintf("failed to update scan job status: %v", err))
		return
	}

	// Execute the scan
	if err := w.executeScan(ctx, job, scanJob); err != nil {
		w.failScanJob(ctx, job, scanJob, err.Error())
		return
	}

	// Mark job as completed
	w.completeJob(ctx, job, scanJob)
}

// executeScan executes the scan using the specified agents
func (w *Worker) executeScan(ctx context.Context, job *queue.Job, scanJob *types.ScanJob) error {
	// Parse scan configuration from job payload
	scanConfig, err := w.parseScanConfig(job)
	if err != nil {
		return fmt.Errorf("failed to parse scan config: %w", err)
	}

	// Execute scans in parallel using all requested agents
	results, err := w.agentManager.ExecuteParallelScans(ctx, scanJob.AgentsRequested, scanConfig)
	
	// Store results even if some agents failed
	for agentName, result := range results {
		if err := w.storeScanResult(ctx, scanJob, agentName, result, nil); err != nil {
			// Log error but continue with other results
			// TODO: Add proper logging
		}
	}

	// If all agents failed, return error
	if len(results) == 0 && err != nil {
		return fmt.Errorf("all agents failed: %w", err)
	}

	// Update completed agents list
	completedAgents := make([]string, 0, len(results))
	for agentName := range results {
		completedAgents = append(completedAgents, agentName)
	}
	
	scanJob.AgentsCompleted = completedAgents
	scanJob.UpdatedAt = time.Now()

	if err := w.db.UpdateScanJob(ctx, scanJob); err != nil {
		return fmt.Errorf("failed to update scan job: %w", err)
	}

	return nil
}

// parseScanJobID extracts the scan job ID from the queue job payload
func (w *Worker) parseScanJobID(job *queue.Job) (uuid.UUID, error) {
	scanJobIDStr, ok := job.Payload["scan_job_id"].(string)
	if !ok {
		return uuid.Nil, errors.NewValidationError("scan_job_id not found in job payload")
	}

	scanJobID, err := uuid.Parse(scanJobIDStr)
	if err != nil {
		return uuid.Nil, errors.NewValidationError("invalid scan_job_id format")
	}

	return scanJobID, nil
}

// parseScanConfig creates an agent.ScanConfig from the queue job payload
func (w *Worker) parseScanConfig(job *queue.Job) (agent.ScanConfig, error) {
	config := agent.ScanConfig{}

	// Extract required fields
	repoURL, ok := job.Payload["repo_url"].(string)
	if !ok {
		return config, errors.NewValidationError("repo_url not found in job payload")
	}
	config.RepoURL = repoURL

	branch, ok := job.Payload["branch"].(string)
	if !ok {
		return config, errors.NewValidationError("branch not found in job payload")
	}
	config.Branch = branch

	commitSHA, ok := job.Payload["commit_sha"].(string)
	if !ok {
		return config, errors.NewValidationError("commit_sha not found in job payload")
	}
	config.Commit = commitSHA

	// Extract optional fields
	if agents, ok := job.Payload["agents"].([]interface{}); ok {
		config.Languages = make([]string, len(agents))
		for i, agent := range agents {
			if agentStr, ok := agent.(string); ok {
				config.Languages[i] = agentStr
			}
		}
	}

	if options, ok := job.Payload["options"].(map[string]interface{}); ok {
		config.Options = make(map[string]string)
		for key, value := range options {
			if valueStr, ok := value.(string); ok {
				config.Options[key] = valueStr
			}
		}
	}

	// Set timeout from job metadata
	config.Timeout = job.Metadata.Timeout

	return config, nil
}

// storeScanResult stores the result from a single agent
func (w *Worker) storeScanResult(ctx context.Context, scanJob *types.ScanJob, agentName string, result *agent.ScanResult, err error) error {
	scanResult := &types.ScanResult{
		ID:        uuid.New(),
		ScanJobID: scanJob.ID,
		AgentName: agentName,
		CreatedAt: time.Now(),
	}

	if err != nil {
		// Agent execution failed
		scanResult.Status = "failed"
		scanResult.ErrorMessage = err.Error()
		scanResult.FindingsCount = 0
	} else if result != nil {
		// Agent execution succeeded
		scanResult.Status = string(result.Status)
		scanResult.FindingsCount = len(result.Findings)
		scanResult.DurationMS = int(result.Duration.Milliseconds())

		// Store raw output
		if rawOutput, marshalErr := json.Marshal(result); marshalErr == nil {
			rawOutputMap := make(map[string]interface{})
			if unmarshalErr := json.Unmarshal(rawOutput, &rawOutputMap); unmarshalErr == nil {
				scanResult.RawOutput = rawOutputMap
			}
		}

		// Store individual findings
		for _, finding := range result.Findings {
			dbFinding := &types.Finding{
				ID:           uuid.New(),
				ScanResultID: scanResult.ID,
				ScanJobID:    scanJob.ID,
				Tool:         finding.Tool,
				RuleID:       finding.RuleID,
				Severity:     string(finding.Severity),
				Category:     string(finding.Category),
				Title:        finding.Title,
				Description:  finding.Description,
				FilePath:     finding.File,
				LineNumber:   finding.Line,
				ColumnNumber: finding.Column,
				CodeSnippet:  finding.Code,
				Confidence:   finding.Confidence,
				Status:       types.FindingStatusOpen,
				References:   finding.References,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			// Store fix suggestion if available
			if finding.Fix != nil {
				fixSuggestion := map[string]interface{}{
					"description": finding.Fix.Description,
					"code":        finding.Fix.Code,
					"references":  finding.Fix.References,
				}
				dbFinding.FixSuggestion = fixSuggestion
			}

			if err := w.db.CreateFinding(ctx, dbFinding); err != nil {
				// Log error but continue
				// TODO: Add proper logging
			}
		}
	}

	// Store scan result
	return w.db.CreateScanResult(ctx, scanResult)
}

// completeJob marks a job as completed
func (w *Worker) completeJob(ctx context.Context, job *queue.Job, scanJob *types.ScanJob) {
	// Update scan job status
	scanJob.Status = types.ScanJobStatusCompleted
	completedAt := time.Now()
	scanJob.CompletedAt = &completedAt
	scanJob.UpdatedAt = completedAt

	if err := w.db.UpdateScanJob(ctx, scanJob); err != nil {
		// Log error but continue
		// TODO: Add proper logging
	}

	// Mark queue job as completed
	result := &queue.JobResult{
		JobID:     job.ID,
		Success:   true,
		Result:    map[string]interface{}{"scan_job_id": scanJob.ID.String()},
		Duration:  time.Since(*scanJob.StartedAt),
		Timestamp: time.Now(),
	}

	if err := w.queue.Complete(ctx, job.ID, result); err != nil {
		// Log error but continue
		// TODO: Add proper logging
	}

	w.jobsProcessed++
}

// failJob marks a job as failed
func (w *Worker) failJob(ctx context.Context, job *queue.Job, errorMsg string) {
	if err := w.queue.Fail(ctx, job.ID, errorMsg); err != nil {
		// Log error but continue
		// TODO: Add proper logging
	}

	w.jobsFailed++
}

// failScanJob marks a scan job as failed
func (w *Worker) failScanJob(ctx context.Context, job *queue.Job, scanJob *types.ScanJob, errorMsg string) {
	// Update scan job status
	scanJob.Status = types.ScanJobStatusFailed
	scanJob.ErrorMessage = errorMsg
	completedAt := time.Now()
	scanJob.CompletedAt = &completedAt
	scanJob.UpdatedAt = completedAt

	if err := w.db.UpdateScanJob(ctx, scanJob); err != nil {
		// Log error but continue
		// TODO: Add proper logging
	}

	// Mark queue job as failed
	w.failJob(ctx, job, errorMsg)
}

// GetStats returns worker statistics
func (w *Worker) GetStats() WorkerStats {
	return WorkerStats{
		WorkerID:      w.id,
		Status:        "running", // TODO: Add proper status tracking
		JobsProcessed: w.jobsProcessed,
		JobsFailed:    w.jobsFailed,
		LastJobAt:     w.lastJobAt,
		Uptime:        time.Since(w.startTime),
	}
}