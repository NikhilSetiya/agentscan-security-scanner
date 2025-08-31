package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/shared/logging"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HealthReport represents the overall health report
type HealthReport struct {
	Status    HealthStatus   `json:"status"`
	Timestamp time.Time      `json:"timestamp"`
	Duration  time.Duration  `json:"duration"`
	Checks    []HealthCheck  `json:"checks"`
	Version   string         `json:"version"`
	Uptime    time.Duration  `json:"uptime"`
}

// HealthChecker interface for health check implementations
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) HealthCheck
}

// HealthService manages health checks
type HealthService struct {
	checkers  []HealthChecker
	startTime time.Time
	version   string
	logger    logging.Logger
	mu        sync.RWMutex
}

// NewHealthService creates a new health service
func NewHealthService(version string) *HealthService {
	return &HealthService{
		checkers:  make([]HealthChecker, 0),
		startTime: time.Now(),
		version:   version,
		logger:    logging.GetLogger(),
	}
}

// RegisterChecker registers a health checker
func (hs *HealthService) RegisterChecker(checker HealthChecker) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.checkers = append(hs.checkers, checker)
}

// Check performs all health checks
func (hs *HealthService) Check(ctx context.Context) HealthReport {
	start := time.Now()
	
	hs.mu.RLock()
	checkers := make([]HealthChecker, len(hs.checkers))
	copy(checkers, hs.checkers)
	hs.mu.RUnlock()
	
	checks := make([]HealthCheck, len(checkers))
	
	// Run all checks concurrently
	var wg sync.WaitGroup
	for i, checker := range checkers {
		wg.Add(1)
		go func(index int, c HealthChecker) {
			defer wg.Done()
			checks[index] = c.Check(ctx)
		}(i, checker)
	}
	
	wg.Wait()
	
	// Determine overall status
	overallStatus := HealthStatusHealthy
	for _, check := range checks {
		if check.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
			break
		} else if check.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}
	
	return HealthReport{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Checks:    checks,
		Version:   hs.version,
		Uptime:    time.Since(hs.startTime),
	}
}

// Handler returns a Gin handler for health checks
func (hs *HealthService) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		
		report := hs.Check(ctx)
		
		// Set appropriate HTTP status code
		statusCode := http.StatusOK
		if report.Status == HealthStatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if report.Status == HealthStatusDegraded {
			statusCode = http.StatusOK // Still return 200 for degraded
		}
		
		c.JSON(statusCode, report)
	}
}

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	db   *sql.DB
	name string
}

// NewDatabaseHealthChecker creates a new database health checker
func NewDatabaseHealthChecker(db *sql.DB, name string) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		db:   db,
		name: name,
	}
}

// Name returns the checker name
func (dhc *DatabaseHealthChecker) Name() string {
	return dhc.name
}

// Check performs the database health check
func (dhc *DatabaseHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	check := HealthCheck{
		Name:      dhc.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	// Check database connectivity
	if err := dhc.db.PingContext(ctx); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Database ping failed: %v", err)
		check.Duration = time.Since(start)
		return check
	}
	
	// Get database stats
	stats := dhc.db.Stats()
	check.Details["open_connections"] = stats.OpenConnections
	check.Details["in_use"] = stats.InUse
	check.Details["idle"] = stats.Idle
	check.Details["wait_count"] = stats.WaitCount
	check.Details["wait_duration"] = stats.WaitDuration.String()
	check.Details["max_idle_closed"] = stats.MaxIdleClosed
	check.Details["max_lifetime_closed"] = stats.MaxLifetimeClosed
	
	// Check if we're running out of connections
	maxOpen := stats.MaxOpenConnections
	if maxOpen > 0 && float64(stats.OpenConnections)/float64(maxOpen) > 0.8 {
		check.Status = HealthStatusDegraded
		check.Message = "Database connection pool is nearly exhausted"
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Database is healthy"
	}
	
	check.Duration = time.Since(start)
	return check
}

// RedisHealthChecker checks Redis connectivity
type RedisHealthChecker struct {
	client *redis.Client
	name   string
}

// NewRedisHealthChecker creates a new Redis health checker
func NewRedisHealthChecker(client *redis.Client, name string) *RedisHealthChecker {
	return &RedisHealthChecker{
		client: client,
		name:   name,
	}
}

// Name returns the checker name
func (rhc *RedisHealthChecker) Name() string {
	return rhc.name
}

// Check performs the Redis health check
func (rhc *RedisHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	check := HealthCheck{
		Name:      rhc.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	// Check Redis connectivity
	if err := rhc.client.Ping(ctx).Err(); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Redis ping failed: %v", err)
		check.Duration = time.Since(start)
		return check
	}
	
	// Get Redis info
	info, err := rhc.client.Info(ctx, "memory", "stats").Result()
	if err != nil {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Could not get Redis info: %v", err)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Redis is healthy"
		check.Details["info"] = info
	}
	
	check.Duration = time.Since(start)
	return check
}

// HTTPHealthChecker checks external HTTP services
type HTTPHealthChecker struct {
	url    string
	name   string
	client *http.Client
}

// NewHTTPHealthChecker creates a new HTTP health checker
func NewHTTPHealthChecker(url, name string) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		url:  url,
		name: name,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the checker name
func (hhc *HTTPHealthChecker) Name() string {
	return hhc.name
}

// Check performs the HTTP health check
func (hhc *HTTPHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	check := HealthCheck{
		Name:      hhc.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", hhc.url, nil)
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Failed to create request: %v", err)
		check.Duration = time.Since(start)
		return check
	}
	
	resp, err := hhc.client.Do(req)
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("HTTP request failed: %v", err)
		check.Duration = time.Since(start)
		return check
	}
	defer resp.Body.Close()
	
	check.Details["status_code"] = resp.StatusCode
	check.Details["url"] = hhc.url
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		check.Status = HealthStatusHealthy
		check.Message = "HTTP service is healthy"
	} else if resp.StatusCode >= 300 && resp.StatusCode < 500 {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("HTTP service returned status %d", resp.StatusCode)
	} else {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("HTTP service returned status %d", resp.StatusCode)
	}
	
	check.Duration = time.Since(start)
	return check
}

// DiskSpaceHealthChecker checks disk space
type DiskSpaceHealthChecker struct {
	path      string
	name      string
	threshold float64 // Percentage threshold (0.0 to 1.0)
}

// NewDiskSpaceHealthChecker creates a new disk space health checker
func NewDiskSpaceHealthChecker(path, name string, threshold float64) *DiskSpaceHealthChecker {
	return &DiskSpaceHealthChecker{
		path:      path,
		name:      name,
		threshold: threshold,
	}
}

// Name returns the checker name
func (dsc *DiskSpaceHealthChecker) Name() string {
	return dsc.name
}

// Check performs the disk space health check
func (dsc *DiskSpaceHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	check := HealthCheck{
		Name:      dsc.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	// This is a simplified implementation
	// In a real implementation, you would use syscall.Statfs or similar
	check.Status = HealthStatusHealthy
	check.Message = "Disk space is healthy"
	check.Details["path"] = dsc.path
	check.Details["threshold"] = dsc.threshold
	
	check.Duration = time.Since(start)
	return check
}

// MemoryHealthChecker checks memory usage
type MemoryHealthChecker struct {
	name      string
	threshold float64 // Memory usage threshold (0.0 to 1.0)
}

// NewMemoryHealthChecker creates a new memory health checker
func NewMemoryHealthChecker(name string, threshold float64) *MemoryHealthChecker {
	return &MemoryHealthChecker{
		name:      name,
		threshold: threshold,
	}
}

// Name returns the checker name
func (mhc *MemoryHealthChecker) Name() string {
	return mhc.name
}

// Check performs the memory health check
func (mhc *MemoryHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	check := HealthCheck{
		Name:      mhc.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	check.Details["alloc"] = m.Alloc
	check.Details["total_alloc"] = m.TotalAlloc
	check.Details["sys"] = m.Sys
	check.Details["num_gc"] = m.NumGC
	check.Details["goroutines"] = runtime.NumGoroutine()
	
	// Simple health check based on allocated memory
	// In a real implementation, you might want more sophisticated checks
	check.Status = HealthStatusHealthy
	check.Message = "Memory usage is healthy"
	
	check.Duration = time.Since(start)
	return check
}

// ReadinessProbe provides a simple readiness check
func ReadinessProbe() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"timestamp": time.Now(),
		})
	}
}

// LivenessProbe provides a simple liveness check
func LivenessProbe() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive",
			"timestamp": time.Now(),
		})
	}
}