package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/config"
)

// ScanJobHandler handles security scan jobs
type ScanJobHandler struct{}

func (h *ScanJobHandler) Handle(ctx context.Context, job *queue.Job) (*queue.JobResult, error) {
	fmt.Printf("Processing scan job %s for repo: %s\n", job.ID, job.Payload["repo_url"])
	
	// Simulate scan work
	time.Sleep(2 * time.Second)
	
	return &queue.JobResult{
		JobID:   job.ID,
		Success: true,
		Result: map[string]interface{}{
			"findings_count": 5,
			"scan_duration":  "2s",
		},
	}, nil
}

func (h *ScanJobHandler) CanHandle(jobType string) bool {
	return jobType == "security_scan"
}

func main() {
	// Redis configuration
	redisConfig := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	// Create Redis client
	redis, err := queue.NewRedisClient(redisConfig)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redis.Close()

	// Create job queue
	jobQueue := queue.NewQueue(redis, "example", queue.DefaultQueueConfig())

	// Create and start worker pool
	poolConfig := queue.DefaultWorkerPoolConfig()
	poolConfig.NumWorkers = 2
	workerPool := queue.NewWorkerPool(jobQueue, poolConfig)

	// Register job handler
	scanHandler := &ScanJobHandler{}
	workerPool.RegisterHandler("security_scan", scanHandler)

	// Start worker pool
	ctx := context.Background()
	if err := workerPool.Start(ctx); err != nil {
		log.Fatalf("Failed to start worker pool: %v", err)
	}

	fmt.Println("Worker pool started, processing jobs...")

	// Enqueue some example jobs
	for i := 0; i < 5; i++ {
		job := queue.NewJob("security_scan", queue.PriorityMedium, map[string]interface{}{
			"repo_url": fmt.Sprintf("https://github.com/example/repo-%d", i),
			"branch":   "main",
		})

		if err := jobQueue.Enqueue(ctx, job); err != nil {
			log.Printf("Failed to enqueue job %d: %v", i, err)
			continue
		}

		fmt.Printf("Enqueued job %s\n", job.ID)
	}

	// Let jobs process
	time.Sleep(10 * time.Second)

	// Get queue statistics
	stats, err := jobQueue.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to get queue stats: %v", err)
	} else {
		fmt.Printf("Queue stats: %+v\n", stats)
	}

	// Get worker statistics
	workerStats := workerPool.GetStats()
	for i, stat := range workerStats {
		fmt.Printf("Worker %d stats: processed=%d, succeeded=%d, failed=%d\n",
			i, stat.JobsProcessed, stat.JobsSucceeded, stat.JobsFailed)
	}

	// Stop worker pool
	if err := workerPool.Stop(); err != nil {
		log.Printf("Failed to stop worker pool: %v", err)
	}

	fmt.Println("Example completed")
}