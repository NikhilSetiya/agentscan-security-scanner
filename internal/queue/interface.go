package queue

import "context"

// QueueInterface defines the interface for job queues
type QueueInterface interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, job *Job) error

	// Dequeue removes and returns the next job from the queue
	Dequeue(ctx context.Context, workerID string) (*Job, error)

	// Complete marks a job as completed
	Complete(ctx context.Context, jobID string, result *JobResult) error

	// Fail marks a job as failed and handles retries
	Fail(ctx context.Context, jobID string, errorMsg string) error

	// Cancel cancels a job
	Cancel(ctx context.Context, jobID string) error

	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, jobID string) (*Job, error)

	// ListJobs lists jobs with optional filtering
	ListJobs(ctx context.Context, filter JobFilter, limit, offset int) ([]*Job, error)

	// GetStats returns queue statistics
	GetStats(ctx context.Context) (*JobStats, error)

	// Cleanup removes expired jobs and handles timeouts
	Cleanup(ctx context.Context) error
}

// Ensure Queue implements QueueInterface
var _ QueueInterface = (*Queue)(nil)