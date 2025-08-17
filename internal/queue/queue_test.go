package queue

import (
	"context"
	"testing"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
)

func TestNewJob(t *testing.T) {
	payload := map[string]interface{}{
		"repo_url": "https://github.com/test/repo",
		"branch":   "main",
	}

	job := NewJob("scan", PriorityHigh, payload)

	if job.ID == "" {
		t.Error("Job ID should be set")
	}

	if job.Type != "scan" {
		t.Errorf("Expected job type 'scan', got %s", job.Type)
	}

	if job.Priority != PriorityHigh {
		t.Errorf("Expected priority %d, got %d", PriorityHigh, job.Priority)
	}

	if job.Status != JobStatusQueued {
		t.Errorf("Expected status %s, got %s", JobStatusQueued, job.Status)
	}

	if job.Payload["repo_url"] != "https://github.com/test/repo" {
		t.Error("Payload not set correctly")
	}
}

func TestJob_WithTimeout(t *testing.T) {
	job := NewJob("test", PriorityMedium, nil)
	timeout := 5 * time.Minute

	job.WithTimeout(timeout)

	if job.Metadata.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, job.Metadata.Timeout)
	}
}

func TestJob_WithRetries(t *testing.T) {
	job := NewJob("test", PriorityMedium, nil)
	maxRetries := 5
	retryDelay := 1 * time.Minute

	job.WithRetries(maxRetries, retryDelay)

	if job.Metadata.MaxRetries != maxRetries {
		t.Errorf("Expected max retries %d, got %d", maxRetries, job.Metadata.MaxRetries)
	}

	if job.Metadata.RetryDelay != retryDelay {
		t.Errorf("Expected retry delay %v, got %v", retryDelay, job.Metadata.RetryDelay)
	}
}

func TestJob_WithScheduledAt(t *testing.T) {
	job := NewJob("test", PriorityMedium, nil)
	scheduledAt := time.Now().Add(1 * time.Hour)

	job.WithScheduledAt(scheduledAt)

	if job.ScheduledAt == nil {
		t.Error("ScheduledAt should be set")
	}

	if !job.ScheduledAt.Equal(scheduledAt) {
		t.Errorf("Expected scheduled at %v, got %v", scheduledAt, *job.ScheduledAt)
	}
}

func TestJob_CanRetry(t *testing.T) {
	job := NewJob("test", PriorityMedium, nil)
	job.WithRetries(3, time.Second)

	// Should be able to retry initially
	if !job.CanRetry() {
		t.Error("Job should be able to retry initially")
	}

	// After max retries, should not be able to retry
	job.Metadata.RetryCount = 3
	if job.CanRetry() {
		t.Error("Job should not be able to retry after max retries")
	}
}

func TestJob_ShouldExecute(t *testing.T) {
	job := NewJob("test", PriorityMedium, nil)

	// Should execute immediately if no scheduled time
	if !job.ShouldExecute() {
		t.Error("Job should execute immediately if no scheduled time")
	}

	// Should not execute if scheduled in the future
	future := time.Now().Add(1 * time.Hour)
	job.WithScheduledAt(future)
	if job.ShouldExecute() {
		t.Error("Job should not execute if scheduled in the future")
	}

	// Should execute if scheduled in the past
	past := time.Now().Add(-1 * time.Hour)
	job.WithScheduledAt(past)
	if !job.ShouldExecute() {
		t.Error("Job should execute if scheduled in the past")
	}
}

func TestJob_ToJSON_FromJSON(t *testing.T) {
	original := NewJob("test", PriorityHigh, map[string]interface{}{
		"key": "value",
	})

	// Convert to JSON
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert job to JSON: %v", err)
	}

	// Convert back from JSON
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("Failed to convert job from JSON: %v", err)
	}

	// Verify fields
	if restored.ID != original.ID {
		t.Errorf("Expected ID %s, got %s", original.ID, restored.ID)
	}

	if restored.Type != original.Type {
		t.Errorf("Expected type %s, got %s", original.Type, restored.Type)
	}

	if restored.Priority != original.Priority {
		t.Errorf("Expected priority %d, got %d", original.Priority, restored.Priority)
	}
}

func TestNewRedisClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.RedisConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &config.RedisConfig{
				Host:     "localhost",
				Port:     6379,
				Password: "",
				DB:       0,
				PoolSize: 10,
			},
			wantErr: true, // Will fail without actual Redis
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRedisClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRedisClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestQueue_Operations(t *testing.T) {
	// Skip this test if Redis is not available
	t.Skip("Skipping queue operations test - requires Redis")

	// This test would require a real Redis connection
	// In a real test environment, you would set up a test Redis instance

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       1, // Use different DB for tests
		PoolSize: 10,
	}

	redis, err := NewRedisClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redis.Close()

	queue := NewQueue(redis, "test", DefaultQueueConfig())
	ctx := context.Background()

	// Test enqueue
	job := NewJob("test_job", PriorityMedium, map[string]interface{}{
		"data": "test",
	})

	err = queue.Enqueue(ctx, job)
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
}

func TestDefaultConfigs(t *testing.T) {
	queueConfig := DefaultQueueConfig()
	if queueConfig.MaxConcurrency <= 0 {
		t.Error("Default queue config should have positive max concurrency")
	}

	workerConfig := DefaultWorkerConfig()
	if workerConfig.Concurrency <= 0 {
		t.Error("Default worker config should have positive concurrency")
	}

	poolConfig := DefaultWorkerPoolConfig()
	if poolConfig.NumWorkers <= 0 {
		t.Error("Default worker pool config should have positive number of workers")
	}
}