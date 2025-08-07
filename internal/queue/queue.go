package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/agentscan/agentscan/pkg/errors"
)

// Queue represents a Redis-based job queue
type Queue struct {
	redis  *RedisClient
	name   string
	config QueueConfig
}

// QueueConfig contains queue configuration
type QueueConfig struct {
	MaxConcurrency int           `json:"max_concurrency"`
	DefaultTimeout time.Duration `json:"default_timeout"`
	RetryDelay     time.Duration `json:"retry_delay"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// DefaultQueueConfig returns default queue configuration
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		MaxConcurrency:  10,
		DefaultTimeout:  10 * time.Minute,
		RetryDelay:      30 * time.Second,
		CleanupInterval: 1 * time.Hour,
	}
}

// NewQueue creates a new job queue
func NewQueue(redis *RedisClient, name string, config QueueConfig) *Queue {
	return &Queue{
		redis:  redis,
		name:   name,
		config: config,
	}
}

// Redis key patterns
func (q *Queue) queueKey(priority Priority) string {
	return fmt.Sprintf("queue:%s:priority:%d", q.name, priority)
}

func (q *Queue) jobKey(jobID string) string {
	return fmt.Sprintf("job:%s:%s", q.name, jobID)
}

func (q *Queue) processingKey() string {
	return fmt.Sprintf("processing:%s", q.name)
}

func (q *Queue) scheduledKey() string {
	return fmt.Sprintf("scheduled:%s", q.name)
}

func (q *Queue) statsKey() string {
	return fmt.Sprintf("stats:%s", q.name)
}

func (q *Queue) deadLetterKey() string {
	return fmt.Sprintf("dead:%s", q.name)
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(ctx context.Context, job *Job) error {
	if job == nil {
		return errors.NewValidationError("job cannot be nil")
	}

	// Update job metadata
	job.Status = JobStatusQueued
	job.UpdatedAt = time.Now()

	// Serialize job
	jobData, err := job.ToJSON()
	if err != nil {
		return errors.NewInternalError("failed to serialize job").WithCause(err)
	}

	// Store job data
	if err := q.redis.Set(ctx, q.jobKey(job.ID), jobData, 24*time.Hour); err != nil {
		return errors.NewInternalError("failed to store job data").WithCause(err)
	}

	// Add to appropriate queue
	if job.ScheduledAt != nil && job.ScheduledAt.After(time.Now()) {
		// Add to scheduled jobs
		score := float64(job.ScheduledAt.Unix())
		if err := q.redis.ZAdd(ctx, q.scheduledKey(), redis.Z{
			Score:  score,
			Member: job.ID,
		}); err != nil {
			return errors.NewInternalError("failed to schedule job").WithCause(err)
		}
	} else {
		// Add to priority queue
		if err := q.redis.LPush(ctx, q.queueKey(job.Priority), job.ID); err != nil {
			return errors.NewInternalError("failed to enqueue job").WithCause(err)
		}
	}

	// Update stats
	if err := q.updateStats(ctx, "enqueued", job.Type); err != nil {
		// Log error but don't fail the operation
		// TODO: Add proper logging
	}

	return nil
}

// Dequeue removes and returns the next job from the queue
func (q *Queue) Dequeue(ctx context.Context, workerID string) (*Job, error) {
	// First, move any scheduled jobs that are ready
	if err := q.moveScheduledJobs(ctx); err != nil {
		// Log error but continue
		// TODO: Add proper logging
	}

	// Try to get job from priority queues (high to low)
	priorities := []Priority{PriorityHigh, PriorityMedium, PriorityLow}
	
	for _, priority := range priorities {
		queueKey := q.queueKey(priority)
		
		// Use blocking pop with short timeout to avoid busy waiting
		result, err := q.redis.BRPop(ctx, 1*time.Second, queueKey)
		if err != nil {
			if errors.IsType(err, errors.ErrorTypeNotFound) {
				continue // No jobs in this priority queue
			}
			return nil, errors.NewInternalError("failed to dequeue job").WithCause(err)
		}

		if len(result) < 2 {
			continue // Invalid result
		}

		jobID := result[1]
		
		// Get job data
		job, err := q.getJob(ctx, jobID)
		if err != nil {
			// Job data not found, remove from processing and continue
			q.redis.Del(ctx, q.jobKey(jobID))
			continue
		}

		// Mark job as running
		job.Status = JobStatusRunning
		job.StartedAt = &time.Time{}
		*job.StartedAt = time.Now()
		job.UpdatedAt = time.Now()
		job.Metadata.WorkerID = workerID

		// Update job data
		if err := q.updateJob(ctx, job); err != nil {
			return nil, errors.NewInternalError("failed to update job status").WithCause(err)
		}

		// Add to processing set with expiration
		processingKey := q.processingKey()
		if err := q.redis.ZAdd(ctx, processingKey, redis.Z{
			Score:  float64(time.Now().Add(job.Metadata.Timeout).Unix()),
			Member: jobID,
		}); err != nil {
			return nil, errors.NewInternalError("failed to add job to processing").WithCause(err)
		}

		return job, nil
	}

	return nil, errors.NewNotFoundError("job")
}

// Complete marks a job as completed
func (q *Queue) Complete(ctx context.Context, jobID string, result *JobResult) error {
	job, err := q.getJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Update job status
	job.Status = JobStatusCompleted
	job.CompletedAt = &time.Time{}
	*job.CompletedAt = time.Now()
	job.UpdatedAt = time.Now()

	// Update job data
	if err := q.updateJob(ctx, job); err != nil {
		return err
	}

	// Remove from processing
	if err := q.redis.client.ZRem(ctx, q.processingKey(), jobID).Err(); err != nil {
		// Log error but don't fail
		// TODO: Add proper logging
	}

	// Store result if provided
	if result != nil {
		resultData, _ := json.Marshal(result)
		resultKey := fmt.Sprintf("result:%s:%s", q.name, jobID)
		q.redis.Set(ctx, resultKey, resultData, 24*time.Hour)
	}

	// Update stats
	q.updateStats(ctx, "completed", job.Type)

	return nil
}

// Fail marks a job as failed and handles retries
func (q *Queue) Fail(ctx context.Context, jobID string, errorMsg string) error {
	job, err := q.getJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Metadata.ErrorMsg = errorMsg
	job.Metadata.RetryCount++
	job.UpdatedAt = time.Now()

	// Check if job can be retried
	if job.CanRetry() {
		// Schedule for retry
		job.Status = JobStatusRetrying
		retryAt := time.Now().Add(job.Metadata.RetryDelay)
		job.ScheduledAt = &retryAt

		// Update job data
		if err := q.updateJob(ctx, job); err != nil {
			return err
		}

		// Add to scheduled jobs
		score := float64(retryAt.Unix())
		if err := q.redis.ZAdd(ctx, q.scheduledKey(), redis.Z{
			Score:  score,
			Member: jobID,
		}); err != nil {
			return errors.NewInternalError("failed to schedule retry").WithCause(err)
		}
	} else {
		// Mark as failed permanently
		job.Status = JobStatusFailed
		job.CompletedAt = &time.Time{}
		*job.CompletedAt = time.Now()

		// Update job data
		if err := q.updateJob(ctx, job); err != nil {
			return err
		}

		// Move to dead letter queue
		if err := q.redis.LPush(ctx, q.deadLetterKey(), jobID); err != nil {
			// Log error but don't fail
			// TODO: Add proper logging
		}
	}

	// Remove from processing
	q.redis.client.ZRem(ctx, q.processingKey(), jobID)

	// Update stats
	if job.Status == JobStatusFailed {
		q.updateStats(ctx, "failed", job.Type)
	} else {
		q.updateStats(ctx, "retried", job.Type)
	}

	return nil
}

// Cancel cancels a job
func (q *Queue) Cancel(ctx context.Context, jobID string) error {
	job, err := q.getJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Can only cancel queued or scheduled jobs
	if job.Status != JobStatusQueued && job.ScheduledAt == nil {
		return errors.NewValidationError("job cannot be cancelled in current status")
	}

	// Update job status
	job.Status = JobStatusCancelled
	job.CompletedAt = &time.Time{}
	*job.CompletedAt = time.Now()
	job.UpdatedAt = time.Now()

	// Update job data
	if err := q.updateJob(ctx, job); err != nil {
		return err
	}

	// Remove from queues
	for _, priority := range []Priority{PriorityHigh, PriorityMedium, PriorityLow} {
		q.redis.client.LRem(ctx, q.queueKey(priority), 0, jobID)
	}
	q.redis.client.ZRem(ctx, q.scheduledKey(), jobID)
	q.redis.client.ZRem(ctx, q.processingKey(), jobID)

	// Update stats
	q.updateStats(ctx, "cancelled", job.Type)

	return nil
}

// GetJob retrieves a job by ID
func (q *Queue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	return q.getJob(ctx, jobID)
}

// ListJobs lists jobs with optional filtering
func (q *Queue) ListJobs(ctx context.Context, filter JobFilter, limit, offset int) ([]*Job, error) {
	// This is a simplified implementation
	// In production, you might want to use Redis modules or secondary indexes
	
	var jobs []*Job
	pattern := fmt.Sprintf("job:%s:*", q.name)
	
	keys, err := q.redis.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if end > len(keys) {
		end = len(keys)
	}
	if start >= len(keys) {
		return jobs, nil
	}

	for i := start; i < end; i++ {
		jobData, err := q.redis.Get(ctx, keys[i])
		if err != nil {
			continue // Skip missing jobs
		}

		job, err := FromJSON([]byte(jobData))
		if err != nil {
			continue // Skip invalid jobs
		}

		// Apply filters
		if q.matchesFilter(job, filter) {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

// GetStats returns queue statistics
func (q *Queue) GetStats(ctx context.Context) (*JobStats, error) {
	stats := &JobStats{
		ByStatus:   make(map[JobStatus]int64),
		ByType:     make(map[string]int64),
		ByPriority: make(map[Priority]int64),
	}

	// Get queue lengths
	for _, priority := range []Priority{PriorityHigh, PriorityMedium, PriorityLow} {
		length, err := q.redis.LLen(ctx, q.queueKey(priority))
		if err == nil {
			stats.ByPriority[priority] = length
			stats.Total += length
		}
	}

	// Get scheduled jobs count
	scheduledCount, err := q.redis.ZCard(ctx, q.scheduledKey())
	if err == nil {
		stats.Total += scheduledCount
	}

	// Get processing jobs count
	processingCount, err := q.redis.ZCard(ctx, q.processingKey())
	if err == nil {
		stats.ByStatus[JobStatusRunning] = processingCount
		stats.Total += processingCount
	}

	return stats, nil
}

// Cleanup removes expired jobs and handles timeouts
func (q *Queue) Cleanup(ctx context.Context) error {
	now := time.Now()

	// Handle expired processing jobs
	expiredJobs, err := q.redis.client.ZRangeByScore(ctx, q.processingKey(), &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(now.Unix(), 10),
	}).Result()
	if err == nil {
		for _, jobID := range expiredJobs {
			// Move expired job back to queue or fail it
			job, err := q.getJob(ctx, jobID)
			if err != nil {
				continue
			}

			if job.CanRetry() {
				// Retry the job
				q.Fail(ctx, jobID, "job timeout")
			} else {
				// Mark as failed
				q.Fail(ctx, jobID, "job timeout - max retries exceeded")
			}
		}
	}

	// Move scheduled jobs that are ready
	return q.moveScheduledJobs(ctx)
}

// Helper methods

func (q *Queue) getJob(ctx context.Context, jobID string) (*Job, error) {
	jobData, err := q.redis.Get(ctx, q.jobKey(jobID))
	if err != nil {
		return nil, err
	}

	job, err := FromJSON([]byte(jobData))
	if err != nil {
		return nil, errors.NewInternalError("failed to deserialize job").WithCause(err)
	}

	return job, nil
}

func (q *Queue) updateJob(ctx context.Context, job *Job) error {
	jobData, err := job.ToJSON()
	if err != nil {
		return errors.NewInternalError("failed to serialize job").WithCause(err)
	}

	return q.redis.Set(ctx, q.jobKey(job.ID), jobData, 24*time.Hour)
}

func (q *Queue) moveScheduledJobs(ctx context.Context) error {
	now := time.Now()
	
	// Get jobs that are ready to run
	readyJobs, err := q.redis.client.ZRangeByScore(ctx, q.scheduledKey(), &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(now.Unix(), 10),
	}).Result()
	if err != nil {
		return err
	}

	for _, jobID := range readyJobs {
		job, err := q.getJob(ctx, jobID)
		if err != nil {
			continue
		}

		// Remove from scheduled
		q.redis.client.ZRem(ctx, q.scheduledKey(), jobID)

		// Add to priority queue
		job.Status = JobStatusQueued
		job.ScheduledAt = nil
		job.UpdatedAt = time.Now()

		if err := q.updateJob(ctx, job); err != nil {
			continue
		}

		if err := q.redis.LPush(ctx, q.queueKey(job.Priority), jobID); err != nil {
			continue
		}
	}

	return nil
}

func (q *Queue) updateStats(ctx context.Context, action, jobType string) error {
	// Simple stats implementation
	// In production, you might want more sophisticated metrics
	statsKey := q.statsKey()
	field := fmt.Sprintf("%s:%s", action, jobType)
	return q.redis.client.HIncrBy(ctx, statsKey, field, 1).Err()
}

func (q *Queue) matchesFilter(job *Job, filter JobFilter) bool {
	if filter.Type != "" && job.Type != filter.Type {
		return false
	}
	if filter.Status != "" && job.Status != filter.Status {
		return false
	}
	if filter.Priority != 0 && job.Priority != filter.Priority {
		return false
	}
	if !filter.Since.IsZero() && job.CreatedAt.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && job.CreatedAt.After(filter.Until) {
		return false
	}
	return true
}