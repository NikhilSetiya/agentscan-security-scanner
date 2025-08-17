package monitoring

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/cache"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
)

// Service provides system monitoring and resource tracking
type Service struct {
	db          *database.DB
	redis       *queue.RedisClient
	statsCache  *cache.StatsCache
	config      *Config
	metrics     *SystemMetrics
	metricsMux  sync.RWMutex
	stopCh      chan struct{}
	running     bool
}

// Config holds monitoring configuration
type Config struct {
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	CPUThreshold       float64       `json:"cpu_threshold"`
	MemoryThreshold    float64       `json:"memory_threshold"`
	QueueThreshold     int64         `json:"queue_threshold"`
	EnableAutoScaling  bool          `json:"enable_auto_scaling"`
	ScaleUpThreshold   float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64       `json:"scale_down_threshold"`
}

// DefaultConfig returns default monitoring configuration
func DefaultConfig() *Config {
	return &Config{
		CollectionInterval: 30 * time.Second,
		RetentionPeriod:    24 * time.Hour,
		CPUThreshold:       80.0,
		MemoryThreshold:    85.0,
		QueueThreshold:     1000,
		EnableAutoScaling:  true,
		ScaleUpThreshold:   75.0,
		ScaleDownThreshold: 25.0,
	}
}

// NewService creates a new monitoring service
func NewService(db *database.DB, redis *queue.RedisClient, statsCache *cache.StatsCache, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		db:         db,
		redis:      redis,
		statsCache: statsCache,
		config:     config,
		metrics:    &SystemMetrics{},
		stopCh:     make(chan struct{}),
	}
}

// Start begins the monitoring service
func (s *Service) Start(ctx context.Context) error {
	if s.running {
		return errors.NewValidationError("monitoring service is already running")
	}

	s.running = true
	
	// Start metrics collection goroutine
	go s.collectMetrics(ctx)
	
	// Start auto-scaling goroutine if enabled
	if s.config.EnableAutoScaling {
		go s.autoScaler(ctx)
	}

	return nil
}

// Stop stops the monitoring service
func (s *Service) Stop() error {
	if !s.running {
		return nil
	}

	close(s.stopCh)
	s.running = false
	return nil
}

// GetMetrics returns current system metrics
func (s *Service) GetMetrics() *SystemMetrics {
	s.metricsMux.RLock()
	defer s.metricsMux.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *s.metrics
	return &metrics
}

// collectMetrics periodically collects system metrics
func (s *Service) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(s.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.updateMetrics(ctx); err != nil {
				// Log error but continue collecting
				continue
			}
		}
	}
}

// updateMetrics collects and updates current system metrics
func (s *Service) updateMetrics(ctx context.Context) error {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// Collect runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	metrics.MemoryUsage = float64(memStats.Alloc) / float64(memStats.Sys) * 100
	metrics.GoroutineCount = int64(runtime.NumGoroutine())
	metrics.GCPauseTime = time.Duration(memStats.PauseTotalNs)

	// Collect database metrics
	if s.db != nil {
		dbStats := s.db.Stats()
		metrics.DatabaseConnections = int64(dbStats.OpenConnections)
		metrics.DatabaseIdleConnections = int64(dbStats.Idle)
		metrics.DatabaseMaxConnections = int64(dbStats.MaxOpenConnections)
	}

	// Collect Redis metrics
	if s.redis != nil {
		redisStats := s.redis.Stats()
		metrics.RedisConnections = int64(redisStats.TotalConns)
		metrics.RedisIdleConnections = int64(redisStats.IdleConns)
		metrics.RedisStaleConnections = int64(redisStats.StaleConns)
	}

	// Collect queue metrics
	queueMetrics, err := s.collectQueueMetrics(ctx)
	if err == nil {
		metrics.QueuedScans = queueMetrics.QueuedJobs
		metrics.ActiveScans = queueMetrics.ActiveJobs
		metrics.FailedScans = queueMetrics.FailedJobs
	}

	// Calculate CPU usage (simplified - in production, use proper CPU monitoring)
	metrics.CPUUsage = s.calculateCPUUsage()

	// Update metrics
	s.metricsMux.Lock()
	s.metrics = metrics
	s.metricsMux.Unlock()

	// Cache metrics for API access
	cacheMetrics := &cache.SystemMetrics{
		ActiveScans:         metrics.ActiveScans,
		QueuedScans:         metrics.QueuedScans,
		TotalAgents:         int(metrics.TotalAgents),
		HealthyAgents:       int(metrics.HealthyAgents),
		CPUUsage:            metrics.CPUUsage,
		MemoryUsage:         metrics.MemoryUsage,
		DatabaseConnections: int(metrics.DatabaseConnections),
		RedisConnections:    int(metrics.RedisConnections),
		AverageResponseTime: metrics.AverageResponseTime,
		UpdatedAt:           metrics.Timestamp,
	}

	return s.statsCache.SetSystemMetrics(ctx, cacheMetrics)
}

// collectQueueMetrics collects job queue metrics
func (s *Service) collectQueueMetrics(ctx context.Context) (*QueueMetrics, error) {
	metrics := &QueueMetrics{}

	// Count queued jobs in different priority queues
	highPriorityCount, err := s.redis.LLen(ctx, "jobs:high")
	if err == nil {
		metrics.QueuedJobs += highPriorityCount
	}

	mediumPriorityCount, err := s.redis.LLen(ctx, "jobs:medium")
	if err == nil {
		metrics.QueuedJobs += mediumPriorityCount
	}

	lowPriorityCount, err := s.redis.LLen(ctx, "jobs:low")
	if err == nil {
		metrics.QueuedJobs += lowPriorityCount
	}

	// Count active jobs (simplified - in production, track this more accurately)
	activeCount, err := s.redis.ZCard(ctx, "jobs:active")
	if err == nil {
		metrics.ActiveJobs = activeCount
	}

	// Count failed jobs
	failedCount, err := s.redis.ZCard(ctx, "jobs:failed")
	if err == nil {
		metrics.FailedJobs = failedCount
	}

	return metrics, nil
}

// calculateCPUUsage calculates current CPU usage (simplified implementation)
func (s *Service) calculateCPUUsage() float64 {
	// In a real implementation, you would use system calls or libraries
	// to get actual CPU usage. This is a placeholder.
	return float64(runtime.NumGoroutine()) / 1000.0 * 100
}

// autoScaler monitors metrics and triggers scaling decisions
func (s *Service) autoScaler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.evaluateScaling(ctx)
		}
	}
}

// evaluateScaling evaluates whether scaling is needed
func (s *Service) evaluateScaling(ctx context.Context) {
	metrics := s.GetMetrics()

	// Check if scale up is needed
	if s.shouldScaleUp(metrics) {
		s.triggerScaleUp(ctx, metrics)
	} else if s.shouldScaleDown(metrics) {
		s.triggerScaleDown(ctx, metrics)
	}
}

// shouldScaleUp determines if scaling up is needed
func (s *Service) shouldScaleUp(metrics *SystemMetrics) bool {
	return metrics.CPUUsage > s.config.ScaleUpThreshold ||
		   metrics.MemoryUsage > s.config.ScaleUpThreshold ||
		   metrics.QueuedScans > s.config.QueueThreshold
}

// shouldScaleDown determines if scaling down is needed
func (s *Service) shouldScaleDown(metrics *SystemMetrics) bool {
	return metrics.CPUUsage < s.config.ScaleDownThreshold &&
		   metrics.MemoryUsage < s.config.ScaleDownThreshold &&
		   metrics.QueuedScans < s.config.QueueThreshold/4
}

// triggerScaleUp triggers scale up actions
func (s *Service) triggerScaleUp(ctx context.Context, metrics *SystemMetrics) {
	// In a real implementation, this would trigger container orchestration
	// or cloud auto-scaling. For now, we'll just log the event.
	
	event := &ScalingEvent{
		Type:      ScaleUp,
		Timestamp: time.Now(),
		Reason:    s.getScaleUpReason(metrics),
		Metrics:   *metrics,
	}

	s.recordScalingEvent(ctx, event)
}

// triggerScaleDown triggers scale down actions
func (s *Service) triggerScaleDown(ctx context.Context, metrics *SystemMetrics) {
	event := &ScalingEvent{
		Type:      ScaleDown,
		Timestamp: time.Now(),
		Reason:    "Low resource utilization",
		Metrics:   *metrics,
	}

	s.recordScalingEvent(ctx, event)
}

// getScaleUpReason determines the reason for scaling up
func (s *Service) getScaleUpReason(metrics *SystemMetrics) string {
	if metrics.CPUUsage > s.config.ScaleUpThreshold {
		return "High CPU usage"
	}
	if metrics.MemoryUsage > s.config.ScaleUpThreshold {
		return "High memory usage"
	}
	if metrics.QueuedScans > s.config.QueueThreshold {
		return "High queue backlog"
	}
	return "Resource threshold exceeded"
}

// recordScalingEvent records a scaling event for audit purposes
func (s *Service) recordScalingEvent(ctx context.Context, event *ScalingEvent) {
	// Store scaling event in cache for recent history
	key := cache.CacheKey{Prefix: "scaling_event", ID: event.Timestamp.Format("20060102150405")}
	s.statsCache.Set(ctx, key, event, 24*time.Hour)
}

// GetResourceAlerts returns current resource alerts
func (s *Service) GetResourceAlerts() []ResourceAlert {
	metrics := s.GetMetrics()
	var alerts []ResourceAlert

	if metrics.CPUUsage > s.config.CPUThreshold {
		alerts = append(alerts, ResourceAlert{
			Type:      "cpu",
			Level:     AlertLevelWarning,
			Message:   "High CPU usage detected",
			Value:     metrics.CPUUsage,
			Threshold: s.config.CPUThreshold,
			Timestamp: time.Now(),
		})
	}

	if metrics.MemoryUsage > s.config.MemoryThreshold {
		alerts = append(alerts, ResourceAlert{
			Type:      "memory",
			Level:     AlertLevelWarning,
			Message:   "High memory usage detected",
			Value:     metrics.MemoryUsage,
			Threshold: s.config.MemoryThreshold,
			Timestamp: time.Now(),
		})
	}

	if metrics.QueuedScans > s.config.QueueThreshold {
		alerts = append(alerts, ResourceAlert{
			Type:      "queue",
			Level:     AlertLevelCritical,
			Message:   "High queue backlog detected",
			Value:     float64(metrics.QueuedScans),
			Threshold: float64(s.config.QueueThreshold),
			Timestamp: time.Now(),
		})
	}

	return alerts
}

// SystemMetrics represents system performance metrics
type SystemMetrics struct {
	Timestamp                time.Time     `json:"timestamp"`
	CPUUsage                 float64       `json:"cpu_usage"`
	MemoryUsage              float64       `json:"memory_usage"`
	GoroutineCount           int64         `json:"goroutine_count"`
	GCPauseTime              time.Duration `json:"gc_pause_time"`
	DatabaseConnections      int64         `json:"database_connections"`
	DatabaseIdleConnections  int64         `json:"database_idle_connections"`
	DatabaseMaxConnections   int64         `json:"database_max_connections"`
	RedisConnections         int64         `json:"redis_connections"`
	RedisIdleConnections     int64         `json:"redis_idle_connections"`
	RedisStaleConnections    int64         `json:"redis_stale_connections"`
	QueuedScans              int64         `json:"queued_scans"`
	ActiveScans              int64         `json:"active_scans"`
	FailedScans              int64         `json:"failed_scans"`
	TotalAgents              int64         `json:"total_agents"`
	HealthyAgents            int64         `json:"healthy_agents"`
	AverageResponseTime      time.Duration `json:"average_response_time"`
}

// QueueMetrics represents job queue metrics
type QueueMetrics struct {
	QueuedJobs int64 `json:"queued_jobs"`
	ActiveJobs int64 `json:"active_jobs"`
	FailedJobs int64 `json:"failed_jobs"`
}

// ScalingEvent represents an auto-scaling event
type ScalingEvent struct {
	Type      ScalingType    `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Reason    string         `json:"reason"`
	Metrics   SystemMetrics  `json:"metrics"`
}

// ScalingType represents the type of scaling action
type ScalingType string

const (
	ScaleUp   ScalingType = "scale_up"
	ScaleDown ScalingType = "scale_down"
)

// ResourceAlert represents a resource usage alert
type ResourceAlert struct {
	Type      string     `json:"type"`
	Level     AlertLevel `json:"level"`
	Message   string     `json:"message"`
	Value     float64    `json:"value"`
	Threshold float64    `json:"threshold"`
	Timestamp time.Time  `json:"timestamp"`
}

// AlertLevel represents the severity level of an alert
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)