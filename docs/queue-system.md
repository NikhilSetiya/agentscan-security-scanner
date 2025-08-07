# Queue System Documentation

The AgentScan queue system provides a robust, Redis-based job queue with priority levels, retry mechanisms, and worker pools for processing security scan jobs.

## Architecture

The queue system consists of several key components:

- **RedisClient**: Wrapper around Redis client with health checks and error handling
- **Queue**: Main job queue with priority levels and scheduling
- **Job**: Represents a unit of work with metadata and configuration
- **Worker**: Processes jobs from the queue
- **WorkerPool**: Manages multiple workers for concurrent processing

## Key Features

### Priority Levels
- **High Priority (10)**: Critical scans that need immediate processing
- **Medium Priority (5)**: Standard scans with normal processing
- **Low Priority (1)**: Background scans that can wait

### Job Scheduling
- **Immediate Execution**: Jobs are processed as soon as workers are available
- **Scheduled Execution**: Jobs can be scheduled for future execution
- **Retry Logic**: Failed jobs are automatically retried with exponential backoff

### Reliability
- **Timeout Handling**: Jobs that exceed their timeout are automatically failed
- **Dead Letter Queue**: Jobs that exceed max retries are moved to dead letter queue
- **Graceful Shutdown**: Workers can be stopped gracefully with proper cleanup

## Usage Examples

### Basic Job Enqueuing

```go
// Create Redis client
redis, err := queue.NewRedisClient(&config.RedisConfig{
    Host: "localhost",
    Port: 6379,
    DB:   0,
})
if err != nil {
    log.Fatal(err)
}

// Create queue
jobQueue := queue.NewQueue(redis, "scans", queue.DefaultQueueConfig())

// Create and enqueue job
job := queue.NewJob("security_scan", queue.PriorityHigh, map[string]interface{}{
    "repo_url": "https://github.com/user/repo",
    "branch":   "main",
})

err = jobQueue.Enqueue(context.Background(), job)
if err != nil {
    log.Printf("Failed to enqueue job: %v", err)
}
```

### Job Configuration

```go
job := queue.NewJob("scan", queue.PriorityMedium, payload).
    WithTimeout(10 * time.Minute).
    WithRetries(3, 30 * time.Second).
    WithScheduledAt(time.Now().Add(1 * time.Hour)).
    WithTags("urgent", "security")
```

### Creating Job Handlers

```go
type ScanJobHandler struct{}

func (h *ScanJobHandler) Handle(ctx context.Context, job *queue.Job) (*queue.JobResult, error) {
    // Extract job data
    repoURL := job.Payload["repo_url"].(string)
    
    // Perform scan work
    findings, err := performSecurityScan(ctx, repoURL)
    if err != nil {
        return nil, err
    }
    
    // Return result
    return &queue.JobResult{
        JobID:   job.ID,
        Success: true,
        Result: map[string]interface{}{
            "findings_count": len(findings),
            "scan_duration":  "2m30s",
        },
    }, nil
}

func (h *ScanJobHandler) CanHandle(jobType string) bool {
    return jobType == "security_scan"
}
```

### Worker Pool Setup

```go
// Create worker pool
poolConfig := queue.DefaultWorkerPoolConfig()
poolConfig.NumWorkers = 5
poolConfig.WorkerConfig.Concurrency = 3

workerPool := queue.NewWorkerPool(jobQueue, poolConfig)

// Register handlers
workerPool.RegisterHandler("security_scan", &ScanJobHandler{})
workerPool.RegisterHandler("dependency_scan", &DependencyJobHandler{})

// Start processing
ctx := context.Background()
if err := workerPool.Start(ctx); err != nil {
    log.Fatal(err)
}

// Graceful shutdown
defer workerPool.Stop()
```

## Configuration

### Queue Configuration

```go
type QueueConfig struct {
    MaxConcurrency  int           // Maximum concurrent jobs
    DefaultTimeout  time.Duration // Default job timeout
    RetryDelay      time.Duration // Default retry delay
    CleanupInterval time.Duration // Cleanup interval for expired jobs
}
```

### Worker Configuration

```go
type WorkerConfig struct {
    Concurrency     int           // Number of concurrent jobs per worker
    PollInterval    time.Duration // How often to poll for jobs
    ShutdownTimeout time.Duration // Timeout for graceful shutdown
}
```

### Redis Configuration

```go
type RedisConfig struct {
    Host     string // Redis host
    Port     int    // Redis port
    Password string // Redis password (optional)
    DB       int    // Redis database number
    PoolSize int    // Connection pool size
}
```

## Job Lifecycle

1. **Enqueued**: Job is added to the appropriate priority queue
2. **Scheduled**: Job is scheduled for future execution (optional)
3. **Running**: Worker picks up job and starts processing
4. **Completed**: Job finishes successfully
5. **Failed**: Job fails and may be retried
6. **Retrying**: Job is scheduled for retry after failure
7. **Cancelled**: Job is cancelled before execution

## Monitoring and Observability

### Queue Statistics

```go
stats, err := jobQueue.GetStats(ctx)
if err != nil {
    log.Printf("Failed to get stats: %v", err)
} else {
    fmt.Printf("Total jobs: %d\n", stats.Total)
    fmt.Printf("By status: %+v\n", stats.ByStatus)
    fmt.Printf("By priority: %+v\n", stats.ByPriority)
}
```

### Worker Statistics

```go
workerStats := workerPool.GetStats()
for i, stat := range workerStats {
    fmt.Printf("Worker %d: processed=%d, succeeded=%d, failed=%d\n",
        i, stat.JobsProcessed, stat.JobsSucceeded, stat.JobsFailed)
}
```

### Health Checks

```go
// Check Redis health
if err := redis.Health(ctx); err != nil {
    log.Printf("Redis unhealthy: %v", err)
}

// Check worker status
if !workerPool.IsRunning() {
    log.Println("Worker pool is not running")
}
```

## Error Handling

### Job Failures

Jobs can fail for various reasons:
- **Timeout**: Job exceeds configured timeout
- **Handler Error**: Job handler returns an error
- **Context Cancellation**: Context is cancelled during execution
- **System Error**: Redis connection issues, etc.

### Retry Logic

Failed jobs are automatically retried based on configuration:
- **Max Retries**: Maximum number of retry attempts
- **Retry Delay**: Delay between retry attempts
- **Exponential Backoff**: Delay increases with each retry

### Dead Letter Queue

Jobs that exceed max retries are moved to a dead letter queue for manual inspection:

```go
// Get dead letter jobs
deadJobs, err := jobQueue.ListJobs(ctx, queue.JobFilter{
    Status: queue.JobStatusFailed,
}, 100, 0)
```

## Best Practices

### Job Design
- Keep jobs idempotent when possible
- Use appropriate timeouts for different job types
- Include sufficient context in job payload
- Handle partial failures gracefully

### Error Handling
- Use structured error messages
- Log errors with sufficient context
- Implement proper retry logic
- Monitor dead letter queue

### Performance
- Use appropriate priority levels
- Configure worker concurrency based on workload
- Monitor queue depths and processing times
- Scale workers based on demand

### Monitoring
- Set up alerts for queue depth
- Monitor job success/failure rates
- Track processing times
- Monitor Redis health and performance

## Testing

### Unit Tests

```go
func TestJobCreation(t *testing.T) {
    job := queue.NewJob("test", queue.PriorityMedium, map[string]interface{}{
        "key": "value",
    })
    
    assert.Equal(t, "test", job.Type)
    assert.Equal(t, queue.PriorityMedium, job.Priority)
    assert.Equal(t, queue.JobStatusQueued, job.Status)
}
```

### Integration Tests

```go
func TestQueueIntegration(t *testing.T) {
    // Requires Redis connection
    if os.Getenv("INTEGRATION_TESTS") != "1" {
        t.Skip("Skipping integration test")
    }
    
    // Test complete job lifecycle
    // ... test implementation
}
```

### Running Tests

```bash
# Unit tests
go test ./internal/queue

# Integration tests (requires Redis)
INTEGRATION_TESTS=1 go test -tags=integration ./internal/queue
```

## Troubleshooting

### Common Issues

1. **Redis Connection Failed**
   - Check Redis server is running
   - Verify connection parameters
   - Check network connectivity

2. **Jobs Not Processing**
   - Verify workers are running
   - Check job handlers are registered
   - Monitor for errors in logs

3. **High Memory Usage**
   - Check for job accumulation
   - Monitor Redis memory usage
   - Implement job cleanup

4. **Slow Processing**
   - Monitor job processing times
   - Check worker concurrency settings
   - Look for blocking operations in handlers

### Debugging Commands

```bash
# Check Redis connection
redis-cli ping

# Monitor Redis commands
redis-cli monitor

# Check queue lengths
redis-cli llen queue:scans:priority:10

# View job data
redis-cli get job:scans:job-id-here
```

This queue system provides a solid foundation for processing security scan jobs with reliability, scalability, and observability built in.