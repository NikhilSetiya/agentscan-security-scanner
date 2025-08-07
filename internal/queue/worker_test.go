package queue

import (
	"context"
	"testing"
)

// MockJobHandler for testing
type MockJobHandler struct {
	handleFunc func(ctx context.Context, job *Job) (*JobResult, error)
	jobType    string
}

func (m *MockJobHandler) Handle(ctx context.Context, job *Job) (*JobResult, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, job)
	}
	return &JobResult{
		JobID:   job.ID,
		Success: true,
		Result:  map[string]interface{}{"processed": true},
	}, nil
}

func (m *MockJobHandler) CanHandle(jobType string) bool {
	return jobType == m.jobType
}

func TestNewWorker(t *testing.T) {
	// This test doesn't require Redis
	queue := &Queue{} // Mock queue
	config := DefaultWorkerConfig()

	worker := NewWorker(queue, config)

	if worker.id == "" {
		t.Error("Worker ID should be set")
	}

	if worker.queue != queue {
		t.Error("Worker queue should be set")
	}

	if worker.config.Concurrency != config.Concurrency {
		t.Error("Worker config should be set")
	}

	if worker.running {
		t.Error("Worker should not be running initially")
	}
}

func TestWorker_RegisterHandler(t *testing.T) {
	worker := NewWorker(&Queue{}, DefaultWorkerConfig())
	handler := &MockJobHandler{jobType: "test"}

	worker.RegisterHandler("test", handler)

	worker.mu.RLock()
	registeredHandler, exists := worker.handlers["test"]
	worker.mu.RUnlock()

	if !exists {
		t.Error("Handler should be registered")
	}

	if registeredHandler != handler {
		t.Error("Registered handler should match")
	}
}

func TestWorker_IsRunning(t *testing.T) {
	worker := NewWorker(&Queue{}, DefaultWorkerConfig())

	if worker.IsRunning() {
		t.Error("Worker should not be running initially")
	}

	// Simulate running state
	worker.mu.Lock()
	worker.running = true
	worker.mu.Unlock()

	if !worker.IsRunning() {
		t.Error("Worker should be running after setting state")
	}
}

func TestWorker_GetStats(t *testing.T) {
	worker := NewWorker(&Queue{}, DefaultWorkerConfig())

	stats := worker.GetStats()

	if stats.JobsProcessed != 0 {
		t.Error("Initial jobs processed should be 0")
	}

	if stats.JobsSucceeded != 0 {
		t.Error("Initial jobs succeeded should be 0")
	}

	if stats.JobsFailed != 0 {
		t.Error("Initial jobs failed should be 0")
	}

	if stats.StartedAt.IsZero() {
		t.Error("Started at should be set")
	}
}

func TestWorker_GetID(t *testing.T) {
	worker := NewWorker(&Queue{}, DefaultWorkerConfig())

	id := worker.GetID()

	if id == "" {
		t.Error("Worker ID should not be empty")
	}

	if id != worker.id {
		t.Error("GetID should return worker ID")
	}
}

func TestNewWorkerPool(t *testing.T) {
	queue := &Queue{}
	config := DefaultWorkerPoolConfig()

	pool := NewWorkerPool(queue, config)

	if pool.queue != queue {
		t.Error("Pool queue should be set")
	}

	if len(pool.workers) != config.NumWorkers {
		t.Errorf("Expected %d workers, got %d", config.NumWorkers, len(pool.workers))
	}

	if pool.running {
		t.Error("Pool should not be running initially")
	}
}

func TestWorkerPool_RegisterHandler(t *testing.T) {
	pool := NewWorkerPool(&Queue{}, DefaultWorkerPoolConfig())
	handler := &MockJobHandler{jobType: "test"}

	pool.RegisterHandler("test", handler)

	// Check that all workers have the handler registered
	for i, worker := range pool.workers {
		worker.mu.RLock()
		registeredHandler, exists := worker.handlers["test"]
		worker.mu.RUnlock()

		if !exists {
			t.Errorf("Handler should be registered on worker %d", i)
		}

		if registeredHandler != handler {
			t.Errorf("Registered handler should match on worker %d", i)
		}
	}
}

func TestWorkerPool_IsRunning(t *testing.T) {
	pool := NewWorkerPool(&Queue{}, DefaultWorkerPoolConfig())

	if pool.IsRunning() {
		t.Error("Pool should not be running initially")
	}

	// Simulate running state
	pool.mu.Lock()
	pool.running = true
	pool.mu.Unlock()

	if !pool.IsRunning() {
		t.Error("Pool should be running after setting state")
	}
}

func TestWorkerPool_GetStats(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	pool := NewWorkerPool(&Queue{}, config)

	stats := pool.GetStats()

	if len(stats) != config.NumWorkers {
		t.Errorf("Expected %d worker stats, got %d", config.NumWorkers, len(stats))
	}

	for i, stat := range stats {
		if stat.JobsProcessed != 0 {
			t.Errorf("Worker %d should have 0 jobs processed initially", i)
		}
	}
}

func TestWorkerPool_GetWorkers(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	pool := NewWorkerPool(&Queue{}, config)

	workers := pool.GetWorkers()

	if len(workers) != config.NumWorkers {
		t.Errorf("Expected %d workers, got %d", config.NumWorkers, len(workers))
	}

	// Verify workers are the same instances
	for i, worker := range workers {
		if worker != pool.workers[i] {
			t.Errorf("Worker %d should be the same instance", i)
		}
	}
}

// Integration test that would require Redis
func TestWorker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip this test if Redis is not available
	t.Skip("Skipping worker integration test - requires Redis")

	// This test would require a real Redis connection and queue
	// In a real test environment, you would:
	// 1. Set up a test Redis instance
	// 2. Create a real queue
	// 3. Start a worker
	// 4. Enqueue jobs
	// 5. Verify jobs are processed
	// 6. Stop the worker
}

func TestWorkerConfig_Validation(t *testing.T) {
	config := DefaultWorkerConfig()

	if config.Concurrency <= 0 {
		t.Error("Default concurrency should be positive")
	}

	if config.PollInterval <= 0 {
		t.Error("Default poll interval should be positive")
	}

	if config.ShutdownTimeout <= 0 {
		t.Error("Default shutdown timeout should be positive")
	}
}

func TestWorkerPoolConfig_Validation(t *testing.T) {
	config := DefaultWorkerPoolConfig()

	if config.NumWorkers <= 0 {
		t.Error("Default number of workers should be positive")
	}

	if config.WorkerConfig.Concurrency <= 0 {
		t.Error("Default worker concurrency should be positive")
	}

	if config.ShutdownTimeout <= 0 {
		t.Error("Default shutdown timeout should be positive")
	}
}