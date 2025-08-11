package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal     *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
	HTTPRequestsInFlight  *prometheus.GaugeVec

	// Business metrics
	ScansTotal            *prometheus.CounterVec
	ScanDuration          *prometheus.HistogramVec
	FindingsTotal         *prometheus.CounterVec
	AgentExecutions       *prometheus.CounterVec
	AgentExecutionDuration *prometheus.HistogramVec
	ConsensusOperations   *prometheus.CounterVec
	
	// System metrics
	DatabaseConnections   *prometheus.GaugeVec
	RedisConnections      *prometheus.GaugeVec
	QueueSize             *prometheus.GaugeVec
	ActiveScans           *prometheus.GaugeVec
	
	// Performance metrics
	CacheHitRatio         *prometheus.GaugeVec
	DatabaseQueryDuration *prometheus.HistogramVec
	CacheOperationDuration *prometheus.HistogramVec
	
	// Error metrics
	ErrorsTotal           *prometheus.CounterVec
	PanicsTotal           *prometheus.CounterVec
	
	// Authentication metrics
	AuthenticationAttempts *prometheus.CounterVec
	AuthenticationDuration *prometheus.HistogramVec
	
	// Resource metrics
	CPUUsage              *prometheus.GaugeVec
	MemoryUsage           *prometheus.GaugeVec
	DiskUsage             *prometheus.GaugeVec
}

// Config holds metrics configuration
type Config struct {
	Namespace string `json:"namespace"`
	Subsystem string `json:"subsystem"`
	Enabled   bool   `json:"enabled"`
}

// DefaultConfig returns default metrics configuration
func DefaultConfig() *Config {
	return &Config{
		Namespace: "agentscan",
		Subsystem: "",
		Enabled:   true,
	}
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(config *Config) *Metrics {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		return &Metrics{}
	}

	m := &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path", "status_code"},
		),
		HTTPRequestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
			[]string{"method", "path"},
		),

		// Business metrics
		ScansTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "scans_total",
				Help:      "Total number of scans performed",
			},
			[]string{"status", "repository", "branch"},
		),
		ScanDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "scan_duration_seconds",
				Help:      "Scan duration in seconds",
				Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1200, 1800},
			},
			[]string{"status", "repository"},
		),
		FindingsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "findings_total",
				Help:      "Total number of findings detected",
			},
			[]string{"severity", "tool", "rule_id", "repository"},
		),
		AgentExecutions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "agent_executions_total",
				Help:      "Total number of agent executions",
			},
			[]string{"agent", "status"},
		),
		AgentExecutionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "agent_execution_duration_seconds",
				Help:      "Agent execution duration in seconds",
				Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
			},
			[]string{"agent", "status"},
		),
		ConsensusOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "consensus_operations_total",
				Help:      "Total number of consensus operations",
			},
			[]string{"operation", "status"},
		),

		// System metrics
		DatabaseConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "database_connections",
				Help:      "Number of database connections",
			},
			[]string{"state"},
		),
		RedisConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "redis_connections",
				Help:      "Number of Redis connections",
			},
			[]string{"state"},
		),
		QueueSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "queue_size",
				Help:      "Number of items in queue",
			},
			[]string{"queue", "priority"},
		),
		ActiveScans: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "active_scans",
				Help:      "Number of currently active scans",
			},
			[]string{"status"},
		),

		// Performance metrics
		CacheHitRatio: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "cache_hit_ratio",
				Help:      "Cache hit ratio",
			},
			[]string{"cache_type"},
		),
		DatabaseQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "database_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
			},
			[]string{"operation", "table"},
		),
		CacheOperationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "cache_operation_duration_seconds",
				Help:      "Cache operation duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
			},
			[]string{"operation", "cache_type"},
		),

		// Error metrics
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"component", "error_type"},
		),
		PanicsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "panics_total",
				Help:      "Total number of panics",
			},
			[]string{"component"},
		),

		// Authentication metrics
		AuthenticationAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "authentication_attempts_total",
				Help:      "Total number of authentication attempts",
			},
			[]string{"provider", "status"},
		),
		AuthenticationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "authentication_duration_seconds",
				Help:      "Authentication duration in seconds",
				Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10},
			},
			[]string{"provider", "status"},
		),

		// Resource metrics
		CPUUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "cpu_usage_percent",
				Help:      "CPU usage percentage",
			},
			[]string{"component"},
		),
		MemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "memory_usage_bytes",
				Help:      "Memory usage in bytes",
			},
			[]string{"component", "type"},
		),
		DiskUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "disk_usage_bytes",
				Help:      "Disk usage in bytes",
			},
			[]string{"component", "type"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestsInFlight,
		m.ScansTotal,
		m.ScanDuration,
		m.FindingsTotal,
		m.AgentExecutions,
		m.AgentExecutionDuration,
		m.ConsensusOperations,
		m.DatabaseConnections,
		m.RedisConnections,
		m.QueueSize,
		m.ActiveScans,
		m.CacheHitRatio,
		m.DatabaseQueryDuration,
		m.CacheOperationDuration,
		m.ErrorsTotal,
		m.PanicsTotal,
		m.AuthenticationAttempts,
		m.AuthenticationDuration,
		m.CPUUsage,
		m.MemoryUsage,
		m.DiskUsage,
	)

	return m
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	if m.HTTPRequestsTotal == nil {
		return
	}

	statusStr := strconv.Itoa(statusCode)
	m.HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, statusStr).Observe(duration.Seconds())
}

// RecordScan records scan metrics
func (m *Metrics) RecordScan(status, repository, branch string, duration time.Duration) {
	if m.ScansTotal == nil {
		return
	}

	m.ScansTotal.WithLabelValues(status, repository, branch).Inc()
	m.ScanDuration.WithLabelValues(status, repository).Observe(duration.Seconds())
}

// RecordFinding records finding metrics
func (m *Metrics) RecordFinding(severity, tool, ruleID, repository string) {
	if m.FindingsTotal == nil {
		return
	}

	m.FindingsTotal.WithLabelValues(severity, tool, ruleID, repository).Inc()
}

// RecordAgentExecution records agent execution metrics
func (m *Metrics) RecordAgentExecution(agent, status string, duration time.Duration) {
	if m.AgentExecutions == nil {
		return
	}

	m.AgentExecutions.WithLabelValues(agent, status).Inc()
	m.AgentExecutionDuration.WithLabelValues(agent, status).Observe(duration.Seconds())
}

// RecordConsensusOperation records consensus operation metrics
func (m *Metrics) RecordConsensusOperation(operation, status string) {
	if m.ConsensusOperations == nil {
		return
	}

	m.ConsensusOperations.WithLabelValues(operation, status).Inc()
}

// UpdateDatabaseConnections updates database connection metrics
func (m *Metrics) UpdateDatabaseConnections(open, idle, max int) {
	if m.DatabaseConnections == nil {
		return
	}

	m.DatabaseConnections.WithLabelValues("open").Set(float64(open))
	m.DatabaseConnections.WithLabelValues("idle").Set(float64(idle))
	m.DatabaseConnections.WithLabelValues("max").Set(float64(max))
}

// UpdateRedisConnections updates Redis connection metrics
func (m *Metrics) UpdateRedisConnections(total, idle, stale int) {
	if m.RedisConnections == nil {
		return
	}

	m.RedisConnections.WithLabelValues("total").Set(float64(total))
	m.RedisConnections.WithLabelValues("idle").Set(float64(idle))
	m.RedisConnections.WithLabelValues("stale").Set(float64(stale))
}

// UpdateQueueSize updates queue size metrics
func (m *Metrics) UpdateQueueSize(queue, priority string, size int64) {
	if m.QueueSize == nil {
		return
	}

	m.QueueSize.WithLabelValues(queue, priority).Set(float64(size))
}

// UpdateActiveScans updates active scan metrics
func (m *Metrics) UpdateActiveScans(status string, count int64) {
	if m.ActiveScans == nil {
		return
	}

	m.ActiveScans.WithLabelValues(status).Set(float64(count))
}

// UpdateCacheHitRatio updates cache hit ratio metrics
func (m *Metrics) UpdateCacheHitRatio(cacheType string, ratio float64) {
	if m.CacheHitRatio == nil {
		return
	}

	m.CacheHitRatio.WithLabelValues(cacheType).Set(ratio)
}

// RecordDatabaseQuery records database query metrics
func (m *Metrics) RecordDatabaseQuery(operation, table string, duration time.Duration) {
	if m.DatabaseQueryDuration == nil {
		return
	}

	m.DatabaseQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordCacheOperation records cache operation metrics
func (m *Metrics) RecordCacheOperation(operation, cacheType string, duration time.Duration) {
	if m.CacheOperationDuration == nil {
		return
	}

	m.CacheOperationDuration.WithLabelValues(operation, cacheType).Observe(duration.Seconds())
}

// RecordError records error metrics
func (m *Metrics) RecordError(component, errorType string) {
	if m.ErrorsTotal == nil {
		return
	}

	m.ErrorsTotal.WithLabelValues(component, errorType).Inc()
}

// RecordPanic records panic metrics
func (m *Metrics) RecordPanic(component string) {
	if m.PanicsTotal == nil {
		return
	}

	m.PanicsTotal.WithLabelValues(component).Inc()
}

// RecordAuthentication records authentication metrics
func (m *Metrics) RecordAuthentication(provider, status string, duration time.Duration) {
	if m.AuthenticationAttempts == nil {
		return
	}

	m.AuthenticationAttempts.WithLabelValues(provider, status).Inc()
	m.AuthenticationDuration.WithLabelValues(provider, status).Observe(duration.Seconds())
}

// UpdateResourceUsage updates resource usage metrics
func (m *Metrics) UpdateResourceUsage(component string, cpuPercent float64, memoryBytes, diskBytes int64) {
	if m.CPUUsage != nil {
		m.CPUUsage.WithLabelValues(component).Set(cpuPercent)
	}
	if m.MemoryUsage != nil {
		m.MemoryUsage.WithLabelValues(component, "used").Set(float64(memoryBytes))
	}
	if m.DiskUsage != nil {
		m.DiskUsage.WithLabelValues(component, "used").Set(float64(diskBytes))
	}
}

// PrometheusMiddleware creates a middleware for Prometheus metrics collection
func (m *Metrics) PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.HTTPRequestsInFlight != nil {
			m.HTTPRequestsInFlight.WithLabelValues(c.Request.Method, c.FullPath()).Inc()
			defer m.HTTPRequestsInFlight.WithLabelValues(c.Request.Method, c.FullPath()).Dec()
		}

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		m.RecordHTTPRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), duration)
	}
}

// Handler returns the Prometheus metrics HTTP handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// MetricsCollector collects and updates system metrics periodically
type MetricsCollector struct {
	metrics  *Metrics
	interval time.Duration
	stopCh   chan struct{}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(metrics *Metrics, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		metrics:  metrics,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins metrics collection
func (mc *MetricsCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopCh:
			return
		case <-ticker.C:
			mc.collectMetrics()
		}
	}
}

// Stop stops metrics collection
func (mc *MetricsCollector) Stop() {
	close(mc.stopCh)
}

// collectMetrics collects system metrics
func (mc *MetricsCollector) collectMetrics() {
	// This would collect actual system metrics
	// For now, we'll just update some basic metrics
	
	// Example: Update resource usage (would be replaced with actual system calls)
	mc.metrics.UpdateResourceUsage("api", 25.5, 512*1024*1024, 1024*1024*1024)
	mc.metrics.UpdateResourceUsage("orchestrator", 15.2, 256*1024*1024, 512*1024*1024)
}