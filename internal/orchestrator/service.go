package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/types"
)

// OrchestrationService manages the execution of security scans across multiple agents
type OrchestrationService interface {
	// SubmitScan queues a new scan job
	SubmitScan(ctx context.Context, req *ScanRequest) (*types.ScanJob, error)

	// GetScanStatus retrieves current scan status
	GetScanStatus(ctx context.Context, jobID string) (*ScanStatus, error)

	// GetScanResults retrieves scan results with filtering
	GetScanResults(ctx context.Context, jobID string, filter *ResultFilter) (*ScanResults, error)

	// CancelScan cancels a running scan
	CancelScan(ctx context.Context, jobID string) error

	// ListScans lists scans with pagination
	ListScans(ctx context.Context, filter *ScanFilter, pagination *Pagination) (*ScanList, error)

	// Start starts the orchestration service
	Start(ctx context.Context) error

	// Stop stops the orchestration service
	Stop(ctx context.Context) error

	// Health returns the health status of the service
	Health(ctx context.Context) error
}

// Service implements the OrchestrationService interface
type Service struct {
	db           database.Repository
	queue        queue.QueueInterface
	agentManager *AgentManager
	config       *Config
	
	// Internal state
	running      bool
	workers      []*Worker
	workerWg     sync.WaitGroup
	stopCh       chan struct{}
	mu           sync.RWMutex
}

// Config contains orchestration service configuration
type Config struct {
	MaxConcurrentScans int           `json:"max_concurrent_scans"`
	DefaultTimeout     time.Duration `json:"default_timeout"`
	WorkerCount        int           `json:"worker_count"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
}

// DefaultConfig returns default orchestration configuration
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrentScans:  100,
		DefaultTimeout:      10 * time.Minute,
		WorkerCount:         5,
		HealthCheckInterval: 30 * time.Second,
		CleanupInterval:     5 * time.Minute,
	}
}

// NewService creates a new orchestration service
func NewService(db database.Repository, queue queue.QueueInterface, agentManager *AgentManager, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		db:           db,
		queue:        queue,
		agentManager: agentManager,
		config:       config,
		stopCh:       make(chan struct{}),
	}
}

// SubmitScan queues a new scan job
func (s *Service) SubmitScan(ctx context.Context, req *ScanRequest) (*types.ScanJob, error) {
	if err := s.validateScanRequest(req); err != nil {
		return nil, err
	}

	// Create scan job record
	scanJob := &types.ScanJob{
		ID:              uuid.New(),
		RepositoryID:    req.RepositoryID,
		UserID:          req.UserID,
		Branch:          req.Branch,
		CommitSHA:       req.CommitSHA,
		ScanType:        req.ScanType,
		Priority:        req.Priority,
		Status:          types.ScanJobStatusQueued,
		AgentsRequested: req.Agents,
		AgentsCompleted: []string{},
		Metadata:        req.Metadata,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Store scan job in database
	if err := s.db.CreateScanJob(ctx, scanJob); err != nil {
		return nil, errors.NewInternalError("failed to create scan job").WithCause(err)
	}

	// Create queue job
	queueJob := queue.NewJob("scan", s.mapPriority(req.Priority), map[string]interface{}{
		"scan_job_id":   scanJob.ID.String(),
		"repository_id": req.RepositoryID.String(),
		"repo_url":      req.RepoURL,
		"branch":        req.Branch,
		"commit_sha":    req.CommitSHA,
		"scan_type":     req.ScanType,
		"agents":        req.Agents,
		"options":       req.Options,
	})

	// Set timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = s.config.DefaultTimeout
	}
	queueJob.WithTimeout(timeout)

	// Add tags for filtering
	tags := []string{"scan", req.ScanType}
	if req.UserID != nil {
		tags = append(tags, fmt.Sprintf("user:%s", req.UserID.String()))
	}
	queueJob.WithTags(tags...)

	// Enqueue the job
	if err := s.queue.Enqueue(ctx, queueJob); err != nil {
		// Try to clean up the database record
		s.db.DeleteScanJob(ctx, scanJob.ID)
		return nil, errors.NewInternalError("failed to enqueue scan job").WithCause(err)
	}

	return scanJob, nil
}

// GetScanStatus retrieves current scan status
func (s *Service) GetScanStatus(ctx context.Context, jobID string) (*ScanStatus, error) {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return nil, errors.NewValidationError("invalid job ID format")
	}

	// Get scan job from database
	scanJob, err := s.db.GetScanJob(ctx, jobUUID)
	if err != nil {
		return nil, err
	}

	// Get scan results
	results, err := s.db.GetScanResults(ctx, jobUUID)
	if err != nil {
		return nil, err
	}

	// Calculate progress
	totalAgents := len(scanJob.AgentsRequested)
	completedAgents := len(scanJob.AgentsCompleted)
	progress := float64(0)
	if totalAgents > 0 {
		progress = float64(completedAgents) / float64(totalAgents) * 100
	}

	status := &ScanStatus{
		JobID:           scanJob.ID.String(),
		Status:          scanJob.Status,
		Progress:        progress,
		StartedAt:       scanJob.StartedAt,
		CompletedAt:     scanJob.CompletedAt,
		Duration:        s.calculateDuration(scanJob),
		AgentsRequested: scanJob.AgentsRequested,
		AgentsCompleted: scanJob.AgentsCompleted,
		ErrorMessage:    scanJob.ErrorMessage,
		Results:         make([]AgentResult, len(results)),
	}

	// Convert scan results to agent results
	for i, result := range results {
		status.Results[i] = AgentResult{
			AgentName:     result.AgentName,
			Status:        result.Status,
			FindingsCount: result.FindingsCount,
			Duration:      time.Duration(result.DurationMS) * time.Millisecond,
			ErrorMessage:  result.ErrorMessage,
		}
	}

	return status, nil
}

// GetScanResults retrieves scan results with filtering
func (s *Service) GetScanResults(ctx context.Context, jobID string, filter *ResultFilter) (*ScanResults, error) {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return nil, errors.NewValidationError("invalid job ID format")
	}

	// Get scan job
	scanJob, err := s.db.GetScanJob(ctx, jobUUID)
	if err != nil {
		return nil, err
	}

	// Get findings with filter
	findings, err := s.db.GetFindings(ctx, jobUUID, s.convertResultFilter(filter))
	if err != nil {
		return nil, err
	}

	// Get scan results for metadata
	scanResults, err := s.db.GetScanResults(ctx, jobUUID)
	if err != nil {
		return nil, err
	}

	results := &ScanResults{
		JobID:       scanJob.ID.String(),
		Status:      scanJob.Status,
		Repository:  scanJob.RepositoryID.String(),
		Branch:      scanJob.Branch,
		CommitSHA:   scanJob.CommitSHA,
		ScanType:    scanJob.ScanType,
		StartedAt:   scanJob.StartedAt,
		CompletedAt: scanJob.CompletedAt,
		Duration:    s.calculateDuration(scanJob),
		Findings:    convertFindings(findings),
		Summary: ResultSummary{
			TotalFindings: len(findings),
			BySeverity:    make(map[string]int),
			ByTool:        make(map[string]int),
			ByCategory:    make(map[string]int),
		},
		AgentResults: make([]AgentResult, len(scanResults)),
	}

	// Calculate summary statistics
	for _, finding := range findings {
		results.Summary.BySeverity[finding.Severity]++
		results.Summary.ByTool[finding.Tool]++
		results.Summary.ByCategory[finding.Category]++
	}

	// Convert scan results
	for i, result := range scanResults {
		results.AgentResults[i] = AgentResult{
			AgentName:     result.AgentName,
			Status:        result.Status,
			FindingsCount: result.FindingsCount,
			Duration:      time.Duration(result.DurationMS) * time.Millisecond,
			ErrorMessage:  result.ErrorMessage,
		}
	}

	return results, nil
}

// CancelScan cancels a running scan
func (s *Service) CancelScan(ctx context.Context, jobID string) error {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return errors.NewValidationError("invalid job ID format")
	}

	// Get scan job
	scanJob, err := s.db.GetScanJob(ctx, jobUUID)
	if err != nil {
		return err
	}

	// Check if scan can be cancelled
	if scanJob.Status == types.ScanJobStatusCompleted || 
	   scanJob.Status == types.ScanJobStatusFailed || 
	   scanJob.Status == types.ScanJobStatusCancelled {
		return errors.NewValidationError("scan cannot be cancelled in current status")
	}

	// Update scan job status
	scanJob.Status = types.ScanJobStatusCancelled
	scanJob.CompletedAt = &time.Time{}
	*scanJob.CompletedAt = time.Now()
	scanJob.UpdatedAt = time.Now()

	if err := s.db.UpdateScanJob(ctx, scanJob); err != nil {
		return errors.NewInternalError("failed to update scan job").WithCause(err)
	}

	// Try to cancel the queue job (best effort)
	// Note: This assumes we can derive the queue job ID from the scan job ID
	// In a real implementation, you might store the queue job ID in the scan job
	queueJobID := fmt.Sprintf("scan-%s", jobID)
	s.queue.Cancel(ctx, queueJobID)

	return nil
}

// ListScans lists scans with pagination
func (s *Service) ListScans(ctx context.Context, filter *ScanFilter, pagination *Pagination) (*ScanList, error) {
	// Convert filter and pagination
	dbFilter := s.convertScanFilter(filter)
	dbPagination := s.convertPagination(pagination)

	// Get scan jobs from database
	scanJobs, total, err := s.db.ListScanJobs(ctx, dbFilter, dbPagination)
	if err != nil {
		return nil, err
	}

	// Convert to API format
	scans := make([]ScanSummary, len(scanJobs))
	for i, job := range scanJobs {
		scans[i] = ScanSummary{
			JobID:       job.ID.String(),
			Repository:  job.RepositoryID.String(),
			Branch:      job.Branch,
			CommitSHA:   job.CommitSHA,
			ScanType:    job.ScanType,
			Status:      job.Status,
			Priority:    job.Priority,
			StartedAt:   job.StartedAt,
			CompletedAt: job.CompletedAt,
			Duration:    s.calculateDuration(job),
			CreatedAt:   job.CreatedAt,
		}
	}

	return &ScanList{
		Scans:      scans,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: (total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize),
	}, nil
}

// Start starts the orchestration service
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.NewValidationError("orchestration service is already running")
	}

	// Initialize workers
	s.workers = make([]*Worker, s.config.WorkerCount)
	for i := 0; i < s.config.WorkerCount; i++ {
		worker := NewWorker(fmt.Sprintf("worker-%d", i), s.queue, s.agentManager, s.db)
		s.workers[i] = worker
	}

	// Start workers
	for _, worker := range s.workers {
		s.workerWg.Add(1)
		go func(w *Worker) {
			defer s.workerWg.Done()
			w.Start(ctx, s.stopCh)
		}(worker)
	}

	// Start health check routine
	s.workerWg.Add(1)
	go func() {
		defer s.workerWg.Done()
		s.healthCheckLoop(ctx)
	}()

	// Start cleanup routine
	s.workerWg.Add(1)
	go func() {
		defer s.workerWg.Done()
		s.cleanupLoop(ctx)
	}()

	s.running = true
	return nil
}

// Stop stops the orchestration service
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// Signal all goroutines to stop
	close(s.stopCh)

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		s.workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers stopped gracefully
	case <-time.After(30 * time.Second):
		// Timeout waiting for workers to stop
		return errors.NewInternalError("timeout waiting for workers to stop")
	}

	s.running = false
	return nil
}

// Health returns the health status of the service
func (s *Service) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return errors.NewInternalError("orchestration service is not running")
	}

	// Check database health
	if err := s.db.Health(ctx); err != nil {
		return errors.NewInternalError("database health check failed").WithCause(err)
	}

	// Check queue health
	if _, err := s.queue.GetStats(ctx); err != nil {
		return errors.NewInternalError("queue health check failed").WithCause(err)
	}

	// Check agent manager health
	if err := s.agentManager.Health(ctx); err != nil {
		return errors.NewInternalError("agent manager health check failed").WithCause(err)
	}

	return nil
}

// Helper methods

func (s *Service) validateScanRequest(req *ScanRequest) error {
	if req == nil {
		return errors.NewValidationError("scan request is required")
	}
	if req.RepositoryID == uuid.Nil {
		return errors.NewValidationError("repository ID is required")
	}
	if req.RepoURL == "" {
		return errors.NewValidationError("repository URL is required")
	}
	if req.Branch == "" {
		return errors.NewValidationError("branch is required")
	}
	if req.CommitSHA == "" {
		return errors.NewValidationError("commit SHA is required")
	}
	if req.ScanType == "" {
		return errors.NewValidationError("scan type is required")
	}
	if len(req.Agents) == 0 {
		return errors.NewValidationError("at least one agent must be specified")
	}
	return nil
}

func (s *Service) mapPriority(priority int) queue.Priority {
	switch {
	case priority >= 10:
		return queue.PriorityHigh
	case priority >= 5:
		return queue.PriorityMedium
	default:
		return queue.PriorityLow
	}
}

func (s *Service) calculateDuration(job *types.ScanJob) time.Duration {
	if job.StartedAt == nil {
		return 0
	}
	if job.CompletedAt == nil {
		return time.Since(*job.StartedAt)
	}
	return job.CompletedAt.Sub(*job.StartedAt)
}

func (s *Service) convertResultFilter(filter *ResultFilter) *database.FindingFilter {
	if filter == nil {
		return &database.FindingFilter{}
	}
	return &database.FindingFilter{
		Severity: filter.Severity,
		Tool:     filter.Tool,
		Category: filter.Category,
		Status:   filter.Status,
		File:     filter.File,
	}
}

func (s *Service) convertScanFilter(filter *ScanFilter) *database.ScanJobFilter {
	if filter == nil {
		return &database.ScanJobFilter{}
	}
	return &database.ScanJobFilter{
		RepositoryID: filter.RepositoryID,
		UserID:       filter.UserID,
		Status:       filter.Status,
		ScanType:     filter.ScanType,
		Since:        filter.Since,
		Until:        filter.Until,
	}
}

func (s *Service) convertPagination(pagination *Pagination) *database.Pagination {
	if pagination == nil {
		return &database.Pagination{
			Page:     1,
			PageSize: 50,
		}
	}
	return &database.Pagination{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}
}

func (s *Service) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			// Perform health checks
			if err := s.agentManager.HealthCheckAll(ctx); err != nil {
				// Log error but continue
				// TODO: Add proper logging
			}
		}
	}
}

func (s *Service) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			// Perform cleanup
			if err := s.queue.Cleanup(ctx); err != nil {
				// Log error but continue
				// TODO: Add proper logging
			}
		}
	}
}

// convertFindings converts database findings to API findings
func convertFindings(dbFindings []*types.Finding) []Finding {
	findings := make([]Finding, len(dbFindings))
	for i, dbFinding := range dbFindings {
		findings[i] = Finding{
			ID:            dbFinding.ID.String(),
			Tool:          dbFinding.Tool,
			RuleID:        dbFinding.RuleID,
			Severity:      dbFinding.Severity,
			Category:      dbFinding.Category,
			Title:         dbFinding.Title,
			Description:   dbFinding.Description,
			File:          dbFinding.FilePath,
			Line:          dbFinding.LineNumber,
			Column:        dbFinding.ColumnNumber,
			Code:          dbFinding.CodeSnippet,
			Confidence:    dbFinding.Confidence,
			Status:        dbFinding.Status,
			FixSuggestion: dbFinding.FixSuggestion,
			References:    dbFinding.References,
			CreatedAt:     dbFinding.CreatedAt,
			UpdatedAt:     dbFinding.UpdatedAt,
		}
	}
	return findings
}