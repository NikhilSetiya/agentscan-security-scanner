package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/agentscan/agentscan/pkg/errors"
)

// JobHandler defines the interface for handling jobs
type JobHandler interface {
	Handle(ctx context.Context, job *Job) (*JobResult, error)
	CanHandle(jobType string) bool
}

// Worker represents a job worker
type Worker struct {
	id       string
	queue    *Queue
	handlers map[string]JobHandler
	config   WorkerConfig
	
	// Control channels
	stopCh   chan struct{}
	doneCh   chan struct{}
	
	// State
	mu       sync.RWMutex
	running  bool
	stats    WorkerStats
}

// WorkerConfig contains worker configuration
type WorkerConfig struct {
	Concurrency     int           `json:"concurrency"`
	PollInterval    time.Duration `json:"poll_interval"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

// DefaultWorkerConfig returns default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Concurrency:     5,
		PollInterval:    1 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// WorkerStats contains worker statistics
type WorkerStats struct {
	JobsProcessed int64     `json:"jobs_processed"`
	JobsSucceeded int64     `json:"jobs_succeeded"`
	JobsFailed    int64     `json:"jobs_failed"`
	LastJobAt     time.Time `json:"last_job_at"`
	StartedAt     time.Time `json:"started_at"`
}

// NewWorker creates a new worker
func NewWorker(queue *Queue, config WorkerConfig) *Worker {
	return &Worker{
		id:       uuid.New().String(),
		queue:    queue,
		handlers: make(map[string]JobHandler),
		config:   config,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		stats: WorkerStats{
			StartedAt: time.Now(),
		},
	}
}

// RegisterHandler registers a job handler
func (w *Worker) RegisterHandler(jobType string, handler JobHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[jobType] = handler
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return errors.NewValidationError("worker is already running")
	}
	w.running = true
	w.mu.Unlock()

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < w.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerNum int) {
			defer wg.Done()
			w.workerLoop(ctx, fmt.Sprintf("%s-%d", w.id, workerNum))
		}(i)
	}

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(w.doneCh)
	}()

	return nil
}

// Stop stops the worker gracefully
func (w *Worker) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return errors.NewValidationError("worker is not running")
	}
	w.mu.Unlock()

	// Signal workers to stop
	close(w.stopCh)

	// Wait for workers to finish with timeout
	select {
	case <-w.doneCh:
		// All workers finished gracefully
	case <-time.After(w.config.ShutdownTimeout):
		// Timeout reached
		return errors.NewTimeoutError("worker shutdown")
	}

	w.mu.Lock()
	w.running = false
	w.mu.Unlock()

	return nil
}

// IsRunning returns whether the worker is running
func (w *Worker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetStats returns worker statistics
func (w *Worker) GetStats() WorkerStats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stats
}

// GetID returns the worker ID
func (w *Worker) GetID() string {
	return w.id
}

// workerLoop is the main worker loop
func (w *Worker) workerLoop(ctx context.Context, workerID string) {
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processNextJob(ctx, workerID)
		}
	}
}

// processNextJob processes the next available job
func (w *Worker) processNextJob(ctx context.Context, workerID string) {
	// Get next job from queue
	job, err := w.queue.Dequeue(ctx, workerID)
	if err != nil {
		if !errors.IsType(err, errors.ErrorTypeNotFound) {
			// Log error but continue
			// TODO: Add proper logging
		}
		return
	}

	// Update stats
	w.mu.Lock()
	w.stats.JobsProcessed++
	w.stats.LastJobAt = time.Now()
	w.mu.Unlock()

	// Process the job
	w.processJob(ctx, job)
}

// processJob processes a single job
func (w *Worker) processJob(ctx context.Context, job *Job) {
	// Create job context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, job.Metadata.Timeout)
	defer cancel()

	// Find handler for job type
	w.mu.RLock()
	handler, exists := w.handlers[job.Type]
	w.mu.RUnlock()

	if !exists {
		// No handler found
		err := fmt.Sprintf("no handler found for job type: %s", job.Type)
		w.queue.Fail(ctx, job.ID, err)
		w.updateStats(false)
		return
	}

	// Execute job
	result, err := handler.Handle(jobCtx, job)
	if err != nil {
		// Job failed
		w.queue.Fail(ctx, job.ID, err.Error())
		w.updateStats(false)
		return
	}

	// Job succeeded
	if result == nil {
		result = &JobResult{
			JobID:     job.ID,
			Success:   true,
			Timestamp: time.Now(),
		}
	}
	result.JobID = job.ID
	result.Success = true
	result.Timestamp = time.Now()

	w.queue.Complete(ctx, job.ID, result)
	w.updateStats(true)
}

// updateStats updates worker statistics
func (w *Worker) updateStats(success bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if success {
		w.stats.JobsSucceeded++
	} else {
		w.stats.JobsFailed++
	}
}

// WorkerPool manages multiple workers
type WorkerPool struct {
	workers []*Worker
	queue   *Queue
	config  WorkerPoolConfig
	
	mu      sync.RWMutex
	running bool
}

// WorkerPoolConfig contains worker pool configuration
type WorkerPoolConfig struct {
	NumWorkers      int           `json:"num_workers"`
	WorkerConfig    WorkerConfig  `json:"worker_config"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

// DefaultWorkerPoolConfig returns default worker pool configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		NumWorkers:      3,
		WorkerConfig:    DefaultWorkerConfig(),
		ShutdownTimeout: 60 * time.Second,
	}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue *Queue, config WorkerPoolConfig) *WorkerPool {
	pool := &WorkerPool{
		queue:  queue,
		config: config,
	}

	// Create workers
	for i := 0; i < config.NumWorkers; i++ {
		worker := NewWorker(queue, config.WorkerConfig)
		pool.workers = append(pool.workers, worker)
	}

	return pool
}

// RegisterHandler registers a handler with all workers
func (p *WorkerPool) RegisterHandler(jobType string, handler JobHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for _, worker := range p.workers {
		worker.RegisterHandler(jobType, handler)
	}
}

// Start starts all workers in the pool
func (p *WorkerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return errors.NewValidationError("worker pool is already running")
	}
	p.running = true
	p.mu.Unlock()

	// Start all workers
	for _, worker := range p.workers {
		if err := worker.Start(ctx); err != nil {
			// Stop already started workers
			p.Stop()
			return errors.NewInternalError("failed to start worker").WithCause(err)
		}
	}

	return nil
}

// Stop stops all workers in the pool
func (p *WorkerPool) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return errors.NewValidationError("worker pool is not running")
	}
	p.mu.Unlock()

	// Stop all workers
	var wg sync.WaitGroup
	errCh := make(chan error, len(p.workers))

	for _, worker := range p.workers {
		wg.Add(1)
		go func(w *Worker) {
			defer wg.Done()
			if err := w.Stop(); err != nil {
				errCh <- err
			}
		}(worker)
	}

	// Wait for all workers to stop
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers stopped
	case <-time.After(p.config.ShutdownTimeout):
		return errors.NewTimeoutError("worker pool shutdown")
	}

	// Check for errors
	close(errCh)
	for err := range errCh {
		// Log errors but don't fail
		// TODO: Add proper logging
		_ = err
	}

	p.mu.Lock()
	p.running = false
	p.mu.Unlock()

	return nil
}

// IsRunning returns whether the worker pool is running
func (p *WorkerPool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetStats returns aggregated statistics from all workers
func (p *WorkerPool) GetStats() []WorkerStats {
	var stats []WorkerStats
	for _, worker := range p.workers {
		stats = append(stats, worker.GetStats())
	}
	return stats
}

// GetWorkers returns all workers in the pool
func (p *WorkerPool) GetWorkers() []*Worker {
	return p.workers
}