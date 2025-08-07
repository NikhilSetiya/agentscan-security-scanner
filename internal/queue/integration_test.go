// +build integration

package queue

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/agentscan/agentscan/pkg/config"
)

// TestQueueIntegration tests the complete queue system
// Run with: go test -tags=integration ./internal/queue
func TestQueueIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run.")
	}

	// Load test configuration
	cfg := &config.RedisConfig{
		Host:     getEnvOrDefault("TEST_REDIS_HOST", "localhost"),
		Port:     6379,
		Password: getEnvOrDefault("TEST_REDIS_PASSWORD", ""),
		DB:       1, // Use different DB for tests
		PoolSize: 10,
	}

	// Create Redis client
	redis, err := NewRedisClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redis.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redis.Health(ctx); err != nil {
		t.Fatalf("Redis health check failed: %v", err)
	}

	// Clean up test data
	redis.FlushDB(ctx)

	// Run queue tests
	t.Run("QueueOperations", func(t *testing.T) {
		testQueueOperations(t, redis)
	})

	t.Run("JobScheduling", func(t *testing.T) {
		testJobScheduling(t, redis)
	})

	t.Run("JobRetries", func(t *testing.T) {
		testJobRetries(t, redis)
	})

	t.Run("WorkerProcessing", func(t *testing.T) {
		testWorkerProcessing(t, redis)
	})

	t.Run("WorkerPool", func(t *testing.T) {
		testWorkerPool(t, redis)
	})
}

func testQueueOperations(t *testing.T, redis *RedisClient) {
	queue := NewQueue(redis, "test_ops", DefaultQueueConfig())
	ctx := context.Background()

	// Test enqueue
	job := NewJob("test_job", PriorityMedium, map[string]interface{}{
		"data": "test_data",
	})

	err := queue.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Test dequeue
	dequeuedJob, err := queue.Dequeue(ctx, "test_worker")
	if err != nil {
		t.Fatalf("Failed to dequeue job: %v", err)
	}

	if dequeuedJob.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, dequeuedJob.ID)
	}

	if dequeuedJob.Status != JobStatusRunning {
		t.Errorf("Expected status %s, got %s", JobStatusRunning, dequeuedJob.Status)
	}

	// Test complete
	result := &JobResult{
		JobID:   dequeuedJob.ID,
		Success: true,
		Result:  map[string]interface{}{"status": "completed"},
	}

	err = queue.Complete(ctx, dequeuedJob.ID, result)
	if err != nil {
		t.Fatalf("Failed to complete job: %v", err)
	}

	// Verify job status
	completedJob, err := queue.GetJob(ctx, dequeuedJob.ID)
	if err != nil {
		t.Fatalf("Failed to get completed job: %v", err)
	}

	if completedJob.Status != JobStatusCompleted {
		t.Errorf("Expected status %s, got %s", JobStatusCompleted, completedJob.Status)
	}

	if completedJob.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func testJobScheduling(t *testing.T, redis *RedisClient) {
	queue := NewQueue(redis, "test_schedule", DefaultQueueConfig())
	ctx := context.Background()

	// Create scheduled job
	scheduledAt := time.Now().Add(2 * time.Second)
	job := NewJob("scheduled_job", PriorityHigh, map[string]interface{}{
		"scheduled": true,
	}).WithScheduledAt(scheduledAt)

	err := queue.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("Failed to enqueue scheduled job: %v", err)
	}

	// Try to dequeue immediately (should not get the job)
	_, err = queue.Dequeue(ctx, "test_worker")
	if err == nil {
		t.Error("Should not be able to dequeue scheduled job immediately")
	}

	// Wait for scheduled time
	time.Sleep(3 * time.Second)

	// Now should be able to dequeue
	dequeuedJob, err := queue.Dequeue(ctx, "test_worker")
	if err != nil {
		t.Fatalf("Failed to dequeue scheduled job: %v", err)
	}

	if dequeuedJob.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, dequeuedJob.ID)
	}

	// Complete the job
	queue.Complete(ctx, dequeuedJob.ID, nil)
}

func testJobRetries(t *testing.T, redis *RedisClient) {
	queue := NewQueue(redis, "test_retry", DefaultQueueConfig())
	ctx := context.Background()

	// Create job with retry configuration
	job := NewJob("retry_job", PriorityMedium, map[string]interface{}{
		"will_fail": true,
	}).WithRetries(2, 1*time.Second)

	err := queue.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Dequeue and fail the job
	dequeuedJob, err := queue.Dequeue(ctx, "test_worker")
	if err != nil {
		t.Fatalf("Failed to dequeue job: %v", err)
	}

	err = queue.Fail(ctx, dequeuedJob.ID, "simulated failure")
	if err != nil {
		t.Fatalf("Failed to fail job: %v", err)
	}

	// Check job status
	failedJob, err := queue.GetJob(ctx, dequeuedJob.ID)
	if err != nil {
		t.Fatalf("Failed to get failed job: %v", err)
	}

	if failedJob.Status != JobStatusRetrying {
		t.Errorf("Expected status %s, got %s", JobStatusRetrying, failedJob.Status)
	}

	if failedJob.Metadata.RetryCount != 1 {
		t.Errorf("Expected retry count 1, got %d", failedJob.Metadata.RetryCount)
	}

	// Wait for retry
	time.Sleep(2 * time.Second)

	// Should be able to dequeue again
	retriedJob, err := queue.Dequeue(ctx, "test_worker")
	if err != nil {
		t.Fatalf("Failed to dequeue retried job: %v", err)
	}

	if retriedJob.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, retriedJob.ID)
	}

	// Complete the retried job
	queue.Complete(ctx, retriedJob.ID, nil)
}

func testWorkerProcessing(t *testing.T, redis *RedisClient) {
	queue := NewQueue(redis, "test_worker", DefaultQueueConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create worker
	config := DefaultWorkerConfig()
	config.Concurrency = 1
	config.PollInterval = 100 * time.Millisecond
	worker := NewWorker(queue, config)

	// Register test handler
	processed := make(chan string, 1)
	handler := &TestJobHandler{
		processedCh: processed,
	}
	worker.RegisterHandler("test_worker_job", handler)

	// Start worker
	err := worker.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Enqueue job
	job := NewJob("test_worker_job", PriorityMedium, map[string]interface{}{
		"message": "hello worker",
	})

	err = queue.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for job to be processed
	select {
	case processedJobID := <-processed:
		if processedJobID != job.ID {
			t.Errorf("Expected processed job ID %s, got %s", job.ID, processedJobID)
		}
	case <-time.After(5 * time.Second):
		t.Error("Job was not processed within timeout")
	}

	// Stop worker
	err = worker.Stop()
	if err != nil {
		t.Fatalf("Failed to stop worker: %v", err)
	}

	// Verify job was completed
	completedJob, err := queue.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to get completed job: %v", err)
	}

	if completedJob.Status != JobStatusCompleted {
		t.Errorf("Expected status %s, got %s", JobStatusCompleted, completedJob.Status)
	}
}

func testWorkerPool(t *testing.T, redis *RedisClient) {
	queue := NewQueue(redis, "test_pool", DefaultQueueConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create worker pool
	config := DefaultWorkerPoolConfig()
	config.NumWorkers = 2
	config.WorkerConfig.Concurrency = 1
	config.WorkerConfig.PollInterval = 100 * time.Millisecond
	pool := NewWorkerPool(queue, config)

	// Register test handler
	processed := make(chan string, 10)
	handler := &TestJobHandler{
		processedCh: processed,
	}
	pool.RegisterHandler("test_pool_job", handler)

	// Start worker pool
	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}

	// Enqueue multiple jobs
	numJobs := 5
	jobIDs := make([]string, numJobs)
	for i := 0; i < numJobs; i++ {
		job := NewJob("test_pool_job", PriorityMedium, map[string]interface{}{
			"index": i,
		})
		jobIDs[i] = job.ID

		err = queue.Enqueue(ctx, job)
		if err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
	}

	// Wait for all jobs to be processed
	processedJobs := make(map[string]bool)
	for i := 0; i < numJobs; i++ {
		select {
		case processedJobID := <-processed:
			processedJobs[processedJobID] = true
		case <-time.After(10 * time.Second):
			t.Errorf("Job %d was not processed within timeout", i)
		}
	}

	// Verify all jobs were processed
	for _, jobID := range jobIDs {
		if !processedJobs[jobID] {
			t.Errorf("Job %s was not processed", jobID)
		}
	}

	// Stop worker pool
	err = pool.Stop()
	if err != nil {
		t.Fatalf("Failed to stop worker pool: %v", err)
	}
}

// TestJobHandler for integration tests
type TestJobHandler struct {
	processedCh chan string
}

func (h *TestJobHandler) Handle(ctx context.Context, job *Job) (*JobResult, error) {
	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	// Signal that job was processed
	if h.processedCh != nil {
		select {
		case h.processedCh <- job.ID:
		default:
		}
	}

	return &JobResult{
		JobID:   job.ID,
		Success: true,
		Result: map[string]interface{}{
			"processed_at": time.Now(),
		},
	}, nil
}

func (h *TestJobHandler) CanHandle(jobType string) bool {
	return jobType == "test_worker_job" || jobType == "test_pool_job"
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}