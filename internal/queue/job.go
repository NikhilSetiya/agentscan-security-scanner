package queue

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Priority represents job priority levels
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 5
	PriorityHigh   Priority = 10
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
	JobStatusRetrying   JobStatus = "retrying"
)

// Job represents a job in the queue
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    Priority               `json:"priority"`
	Status      JobStatus              `json:"status"`
	Payload     map[string]interface{} `json:"payload"`
	Metadata    JobMetadata            `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// JobMetadata contains additional job information
type JobMetadata struct {
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"max_retries"`
	RetryCount  int           `json:"retry_count"`
	RetryDelay  time.Duration `json:"retry_delay"`
	ErrorMsg    string        `json:"error_msg,omitempty"`
	WorkerID    string        `json:"worker_id,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
}

// NewJob creates a new job
func NewJob(jobType string, priority Priority, payload map[string]interface{}) *Job {
	now := time.Now()
	return &Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Priority:  priority,
		Status:    JobStatusQueued,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata: JobMetadata{
			Timeout:    10 * time.Minute, // Default timeout
			MaxRetries: 3,                // Default max retries
			RetryDelay: 30 * time.Second, // Default retry delay
		},
	}
}

// WithTimeout sets the job timeout
func (j *Job) WithTimeout(timeout time.Duration) *Job {
	j.Metadata.Timeout = timeout
	return j
}

// WithRetries sets the retry configuration
func (j *Job) WithRetries(maxRetries int, retryDelay time.Duration) *Job {
	j.Metadata.MaxRetries = maxRetries
	j.Metadata.RetryDelay = retryDelay
	return j
}

// WithScheduledAt sets when the job should be executed
func (j *Job) WithScheduledAt(scheduledAt time.Time) *Job {
	j.ScheduledAt = &scheduledAt
	return j
}

// WithTags adds tags to the job
func (j *Job) WithTags(tags ...string) *Job {
	j.Metadata.Tags = append(j.Metadata.Tags, tags...)
	return j
}

// IsExpired checks if the job has expired
func (j *Job) IsExpired() bool {
	if j.StartedAt == nil {
		return false
	}
	return time.Since(*j.StartedAt) > j.Metadata.Timeout
}

// CanRetry checks if the job can be retried
func (j *Job) CanRetry() bool {
	return j.Metadata.RetryCount < j.Metadata.MaxRetries
}

// ShouldExecute checks if the job should be executed now
func (j *Job) ShouldExecute() bool {
	if j.ScheduledAt == nil {
		return true
	}
	return time.Now().After(*j.ScheduledAt)
}

// ToJSON converts the job to JSON
func (j *Job) ToJSON() ([]byte, error) {
	return json.Marshal(j)
}

// FromJSON creates a job from JSON
func FromJSON(data []byte) (*Job, error) {
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// JobFilter represents filters for job queries
type JobFilter struct {
	Type     string    `json:"type,omitempty"`
	Status   JobStatus `json:"status,omitempty"`
	Priority Priority  `json:"priority,omitempty"`
	Tags     []string  `json:"tags,omitempty"`
	Since    time.Time `json:"since,omitempty"`
	Until    time.Time `json:"until,omitempty"`
}

// JobStats represents job queue statistics
type JobStats struct {
	Total     int64            `json:"total"`
	ByStatus  map[JobStatus]int64 `json:"by_status"`
	ByType    map[string]int64 `json:"by_type"`
	ByPriority map[Priority]int64 `json:"by_priority"`
}

// JobResult represents the result of job execution
type JobResult struct {
	JobID     string                 `json:"job_id"`
	Success   bool                   `json:"success"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}