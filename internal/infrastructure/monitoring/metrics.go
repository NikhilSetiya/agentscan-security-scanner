package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/shared/logging"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal     *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
	HTTPRequestSize       *prometheus.HistogramVec
	HTTPResponseSize      *prometheus.HistogramVec
	
	// Application metrics
	ActiveConnections     prometheus.Gauge
	DatabaseConnections   *prometheus.GaugeVec
	CacheHits             *prometheus.CounterVec
	CacheMisses           *prometheus.CounterVec
	
	// Business metrics
	ScansTotal            *prometheus.CounterVec
	ScanDuration          *prometheus.HistogramVec
	FindingsTotal         *prometheus.CounterVec
	UsersTotal            prometheus.Gauge
	RepositoriesTotal     prometheus.Gauge
	
	// System metrics
	GoRoutines            prometheus.Gauge
	MemoryUsage           *prometheus.GaugeVec
	CPUUsage              prometheus.Gauge
	
	// Error metrics
	ErrorsTotal           *prometheus.CounterVec
	PanicTotal            prometheus.Counter
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		HTTPRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		HTTPResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),
		
		// Application metrics
		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_connections",
				Help: "Number of active connections",
			},
		),
		DatabaseConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "database_connections",
				Help: "Number of database connections",
			},
			[]string{"state"}, // open, idle, in_use
		),
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_name"},
		),
		CacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_name"},
		),
		
		// Business metrics
		ScansTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "scans_total",
				Help: "Total number of scans",
			},
			[]string{"status", "scan_type"},
		),
		ScanDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "scan_duration_seconds",
				Help:    "Scan duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
			},
			[]string{"scan_type"},
		),
		FindingsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "findings_total",
				Help: "Total number of findings",
			},
			[]string{"severity", "category"},
		),
		UsersTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "users_total",
				Help: "Total number of users",
			},
		),
		RepositoriesTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "repositories_total",
				Help: "Total number of repositories",
			},
		),
		
		// System metrics
		GoRoutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "go_goroutines",
				Help: "Number of goroutines",
			},
		),
		MemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"type"}, // heap, stack, etc.
		),
		CPUUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "cpu_usage_percent",
				Help: "CPU usage percentage",
			},
		),
		
		// Error metrics
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors",
			},
			[]string{"type", "component"},
		),
		PanicTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "panics_total",
				Help: "Total number of panics",
			},
		),
	}
	
	// Register all metrics
	prometheus.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestSize,
		m.HTTPResponseSize,
		m.ActiveConnections,
		m.DatabaseConnections,
		m.CacheHits,
		m.CacheMisses,
		m.ScansTotal,
		m.ScanDuration,
		m.FindingsTotal,
		m.UsersTotal,
		m.RepositoriesTotal,
		m.GoRoutines,
		m.MemoryUsage,
		m.CPUUsage,
		m.ErrorsTotal,
		m.PanicTotal,
	)
	
	return m
}

// MetricsMiddleware creates a Gin middleware for collecting HTTP metrics
func (m *Metrics) MetricsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		
		// Increment active connections
		m.ActiveConnections.Inc()
		defer m.ActiveConnections.Dec()
		
		// Record request size
		if c.Request.ContentLength > 0 {
			m.HTTPRequestSize.WithLabelValues(
				c.Request.Method,
				c.FullPath(),
			).Observe(float64(c.Request.ContentLength))
		}
		
		c.Next()
		
		// Record metrics after request completion
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())
		
		m.HTTPRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			statusCode,
		).Inc()
		
		m.HTTPRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(duration)
		
		m.HTTPResponseSize.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(float64(c.Writer.Size()))
		
		// Record errors
		if c.Writer.Status() >= 400 {
			errorType := "client_error"
			if c.Writer.Status() >= 500 {
				errorType = "server_error"
			}
			
			m.ErrorsTotal.WithLabelValues(
				errorType,
				"http",
			).Inc()
		}
	})
}

// RecordScan records scan metrics
func (m *Metrics) RecordScan(scanType, status string, duration time.Duration) {
	m.ScansTotal.WithLabelValues(status, scanType).Inc()
	m.ScanDuration.WithLabelValues(scanType).Observe(duration.Seconds())
}

// RecordFinding records finding metrics
func (m *Metrics) RecordFinding(severity, category string) {
	m.FindingsTotal.WithLabelValues(severity, category).Inc()
}

// RecordCacheHit records cache hit
func (m *Metrics) RecordCacheHit(cacheName string) {
	m.CacheHits.WithLabelValues(cacheName).Inc()
}

// RecordCacheMiss records cache miss
func (m *Metrics) RecordCacheMiss(cacheName string) {
	m.CacheMisses.WithLabelValues(cacheName).Inc()
}

// RecordError records an error
func (m *Metrics) RecordError(errorType, component string) {
	m.ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

// RecordPanic records a panic
func (m *Metrics) RecordPanic() {
	m.PanicTotal.Inc()
}

// UpdateSystemMetrics updates system-level metrics
func (m *Metrics) UpdateSystemMetrics() {
	// This would typically be called periodically
	// Implementation would collect actual system metrics
	// For now, we'll just update goroutines count
	m.GoRoutines.Set(float64(runtime.NumGoroutine()))
}

// MetricsServer creates an HTTP server for serving Prometheus metrics
type MetricsServer struct {
	server *http.Server
	config *config.MonitoringConfig
	logger logging.Logger
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(config *config.MonitoringConfig) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle(config.MetricsPath, promhttp.Handler())
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.MetricsPort),
		Handler: mux,
	}
	
	return &MetricsServer{
		server: server,
		config: config,
		logger: logging.GetLogger(),
	}
}

// Start starts the metrics server
func (ms *MetricsServer) Start() error {
	if !ms.config.MetricsEnabled {
		ms.logger.Info("Metrics server disabled")
		return nil
	}
	
	ms.logger.Info("Starting metrics server", 
		"port", ms.config.MetricsPort,
		"path", ms.config.MetricsPath,
	)
	
	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ms.logger.Error("Metrics server error", "error", err)
		}
	}()
	
	return nil
}

// Stop stops the metrics server
func (ms *MetricsServer) Stop(ctx context.Context) error {
	if !ms.config.MetricsEnabled {
		return nil
	}
	
	ms.logger.Info("Stopping metrics server")
	return ms.server.Shutdown(ctx)
}

// Global metrics instance
var globalMetrics *Metrics

// InitMetrics initializes the global metrics instance
func InitMetrics() *Metrics {
	if globalMetrics == nil {
		globalMetrics = NewMetrics()
	}
	return globalMetrics
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	if globalMetrics == nil {
		globalMetrics = NewMetrics()
	}
	return globalMetrics
}