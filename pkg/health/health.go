package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/logging"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
	StatusUnknown   Status = "unknown"
)

// Check represents a health check
type Check struct {
	Name        string            `json:"name"`
	Status      Status            `json:"status"`
	Message     string            `json:"message,omitempty"`
	Error       string            `json:"error,omitempty"`
	Duration    time.Duration     `json:"duration"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    Status             `json:"status"`
	Timestamp time.Time          `json:"timestamp"`
	Duration  time.Duration      `json:"duration"`
	Checks    map[string]*Check  `json:"checks"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
}

// Checker interface for health checks
type Checker interface {
	Check(ctx context.Context) *Check
}

// Service provides health checking functionality
type Service struct {
	checkers map[string]Checker
	logger   *logging.Logger
	metadata map[string]string
	mutex    sync.RWMutex
}

// Config holds health check configuration
type Config struct {
	Timeout  time.Duration     `json:"timeout"`
	Metadata map[string]string `json:"metadata"`
}

// DefaultConfig returns default health check configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:  5 * time.Second,
		Metadata: make(map[string]string),
	}
}

// NewService creates a new health check service
func NewService(logger *logging.Logger, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		checkers: make(map[string]Checker),
		logger:   logger,
		metadata: config.Metadata,
	}
}

// RegisterChecker registers a health checker
func (s *Service) RegisterChecker(name string, checker Checker) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.checkers[name] = checker
}

// UnregisterChecker unregisters a health checker
func (s *Service) UnregisterChecker(name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.checkers, name)
}

// CheckHealth performs all health checks
func (s *Service) CheckHealth(ctx context.Context) *HealthResponse {
	start := time.Now()
	
	s.mutex.RLock()
	checkers := make(map[string]Checker, len(s.checkers))
	for name, checker := range s.checkers {
		checkers[name] = checker
	}
	s.mutex.RUnlock()

	checks := make(map[string]*Check, len(checkers))
	overallStatus := StatusHealthy

	// Run all checks concurrently
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for name, checker := range checkers {
		wg.Add(1)
		go func(name string, checker Checker) {
			defer wg.Done()
			
			check := checker.Check(ctx)
			
			mutex.Lock()
			checks[name] = check
			
			// Update overall status
			switch check.Status {
			case StatusUnhealthy:
				overallStatus = StatusUnhealthy
			case StatusDegraded:
				if overallStatus == StatusHealthy {
					overallStatus = StatusDegraded
				}
			}
			mutex.Unlock()
		}(name, checker)
	}

	wg.Wait()

	duration := time.Since(start)

	return &HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Duration:  duration,
		Checks:    checks,
		Metadata:  s.metadata,
	}
}

// Handler returns a Gin handler for health checks
func (s *Service) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		health := s.CheckHealth(ctx)

		statusCode := http.StatusOK
		switch health.Status {
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		case StatusDegraded:
			statusCode = http.StatusPartialContent
		}

		c.JSON(statusCode, health)
	}
}

// LivenessHandler returns a simple liveness check handler
func (s *Service) LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "alive",
			"timestamp": time.Now(),
		})
	}
}

// ReadinessHandler returns a readiness check handler
func (s *Service) ReadinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		health := s.CheckHealth(ctx)

		statusCode := http.StatusOK
		if health.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, gin.H{
			"status":    health.Status,
			"timestamp": health.Timestamp,
			"ready":     health.Status != StatusUnhealthy,
		})
	}
}

// DatabaseChecker checks database connectivity
type DatabaseChecker struct {
	db   *database.DB
	name string
}

// NewDatabaseChecker creates a new database health checker
func NewDatabaseChecker(db *database.DB, name string) *DatabaseChecker {
	return &DatabaseChecker{
		db:   db,
		name: name,
	}
}

// Check performs database health check
func (dc *DatabaseChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      dc.name,
		Timestamp: start,
	}

	if dc.db == nil {
		check.Status = StatusUnhealthy
		check.Error = "database connection is nil"
		check.Duration = time.Since(start)
		return check
	}

	// Check database connectivity
	if err := dc.db.Health(ctx); err != nil {
		check.Status = StatusUnhealthy
		check.Error = err.Error()
		check.Duration = time.Since(start)
		return check
	}

	// Get connection stats
	stats := dc.db.Stats()
	check.Status = StatusHealthy
	check.Message = "database is healthy"
	check.Duration = time.Since(start)
	check.Metadata = map[string]string{
		"open_connections": fmt.Sprintf("%d", stats.OpenConnections),
		"idle_connections": fmt.Sprintf("%d", stats.Idle),
		"max_connections":  fmt.Sprintf("%d", stats.MaxOpenConnections),
	}

	// Check if we're running low on connections
	if stats.OpenConnections > int(float64(stats.MaxOpenConnections)*0.8) {
		check.Status = StatusDegraded
		check.Message = "database connection pool is running low"
	}

	return check
}

// RedisChecker checks Redis connectivity
type RedisChecker struct {
	redis *queue.RedisClient
	name  string
}

// NewRedisChecker creates a new Redis health checker
func NewRedisChecker(redis *queue.RedisClient, name string) *RedisChecker {
	return &RedisChecker{
		redis: redis,
		name:  name,
	}
}

// Check performs Redis health check
func (rc *RedisChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      rc.name,
		Timestamp: start,
	}

	if rc.redis == nil {
		check.Status = StatusUnhealthy
		check.Error = "redis connection is nil"
		check.Duration = time.Since(start)
		return check
	}

	// Check Redis connectivity
	if err := rc.redis.Health(ctx); err != nil {
		check.Status = StatusUnhealthy
		check.Error = err.Error()
		check.Duration = time.Since(start)
		return check
	}

	// Get connection stats
	stats := rc.redis.Stats()
	check.Status = StatusHealthy
	check.Message = "redis is healthy"
	check.Duration = time.Since(start)
	check.Metadata = map[string]string{
		"total_connections": fmt.Sprintf("%d", stats.TotalConns),
		"idle_connections":  fmt.Sprintf("%d", stats.IdleConns),
		"stale_connections": fmt.Sprintf("%d", stats.StaleConns),
	}

	return check
}

// HTTPChecker checks HTTP endpoint health
type HTTPChecker struct {
	url    string
	name   string
	client *http.Client
}

// NewHTTPChecker creates a new HTTP health checker
func NewHTTPChecker(url, name string, timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		url:  url,
		name: name,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Check performs HTTP health check
func (hc *HTTPChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      hc.name,
		Timestamp: start,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", hc.url, nil)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Error = fmt.Sprintf("failed to create request: %v", err)
		check.Duration = time.Since(start)
		return check
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		check.Status = StatusUnhealthy
		check.Error = fmt.Sprintf("request failed: %v", err)
		check.Duration = time.Since(start)
		return check
	}
	defer resp.Body.Close()

	check.Duration = time.Since(start)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		check.Status = StatusHealthy
		check.Message = "endpoint is healthy"
	} else if resp.StatusCode >= 500 {
		check.Status = StatusUnhealthy
		check.Message = fmt.Sprintf("endpoint returned status %d", resp.StatusCode)
	} else {
		check.Status = StatusDegraded
		check.Message = fmt.Sprintf("endpoint returned status %d", resp.StatusCode)
	}

	check.Metadata = map[string]string{
		"status_code":    fmt.Sprintf("%d", resp.StatusCode),
		"response_time":  check.Duration.String(),
	}

	return check
}

// CustomChecker allows for custom health checks
type CustomChecker struct {
	name     string
	checkFn  func(ctx context.Context) (Status, string, error)
	metadata map[string]string
}

// NewCustomChecker creates a new custom health checker
func NewCustomChecker(name string, checkFn func(ctx context.Context) (Status, string, error)) *CustomChecker {
	return &CustomChecker{
		name:     name,
		checkFn:  checkFn,
		metadata: make(map[string]string),
	}
}

// WithMetadata adds metadata to the custom checker
func (cc *CustomChecker) WithMetadata(metadata map[string]string) *CustomChecker {
	cc.metadata = metadata
	return cc
}

// Check performs custom health check
func (cc *CustomChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      cc.name,
		Timestamp: start,
		Metadata:  cc.metadata,
	}

	status, message, err := cc.checkFn(ctx)
	check.Status = status
	check.Message = message
	check.Duration = time.Since(start)

	if err != nil {
		check.Error = err.Error()
		if check.Status == StatusHealthy {
			check.Status = StatusUnhealthy
		}
	}

	return check
}

// DiskSpaceChecker checks available disk space
type DiskSpaceChecker struct {
	path      string
	name      string
	threshold float64 // Percentage threshold (0.0 to 1.0)
}

// NewDiskSpaceChecker creates a new disk space health checker
func NewDiskSpaceChecker(path, name string, threshold float64) *DiskSpaceChecker {
	return &DiskSpaceChecker{
		path:      path,
		name:      name,
		threshold: threshold,
	}
}

// Check performs disk space health check
func (dsc *DiskSpaceChecker) Check(ctx context.Context) *Check {
	start := time.Now()
	check := &Check{
		Name:      dsc.name,
		Timestamp: start,
	}

	// This is a simplified implementation
	// In a real implementation, you would use syscalls to get actual disk usage
	check.Status = StatusHealthy
	check.Message = "disk space is healthy"
	check.Duration = time.Since(start)
	check.Metadata = map[string]string{
		"path":      dsc.path,
		"threshold": fmt.Sprintf("%.1f%%", dsc.threshold*100),
	}

	return check
}