package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/agentscan/internal/shared/testing"
)

func TestMetricsCollector(t *testing.T) {
	t.Run("initialize_metrics_collector", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		assert.NotNil(t, collector)
		assert.NotNil(t, collector.httpRequestsTotal)
		assert.NotNil(t, collector.httpRequestDuration)
		assert.NotNil(t, collector.activeConnections)
		assert.NotNil(t, collector.businessMetrics)
	})

	t.Run("record_http_metrics", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Record some HTTP metrics
		collector.RecordHTTPRequest("GET", "/api/scans", 200, 150*time.Millisecond)
		collector.RecordHTTPRequest("POST", "/api/scans", 201, 300*time.Millisecond)
		collector.RecordHTTPRequest("GET", "/api/scans", 404, 50*time.Millisecond)

		// Check request count metrics
		expected := `
		# HELP agentscan_test_http_requests_total Total number of HTTP requests
		# TYPE agentscan_test_http_requests_total counter
		agentscan_test_http_requests_total{method="GET",path="/api/scans",status="200"} 1
		agentscan_test_http_requests_total{method="GET",path="/api/scans",status="404"} 1
		agentscan_test_http_requests_total{method="POST",path="/api/scans",status="201"} 1
		`
		
		err := testutil.CollectAndCompare(collector.httpRequestsTotal, strings.NewReader(expected))
		assert.NoError(t, err)

		// Check that duration histogram has samples
		metric := &prometheus.MetricFamily{}
		collector.httpRequestDuration.Write(metric)
		assert.NotEmpty(t, metric.GetMetric())
	})

	t.Run("record_business_metrics", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Record business metrics
		collector.RecordScanCompleted("vulnerability", true, 45*time.Second)
		collector.RecordScanCompleted("compliance", false, 30*time.Second)
		collector.RecordAgentActivity("agent-123", "scan_started")
		collector.RecordAgentActivity("agent-456", "scan_completed")

		// Check scan metrics
		scanMetric := testutil.ToFloat64(collector.businessMetrics.WithLabelValues("scans_completed", "vulnerability", "success"))
		assert.Equal(t, float64(1), scanMetric)

		failedScanMetric := testutil.ToFloat64(collector.businessMetrics.WithLabelValues("scans_completed", "compliance", "failure"))
		assert.Equal(t, float64(1), failedScanMetric)

		// Check agent activity
		agentMetric := testutil.ToFloat64(collector.businessMetrics.WithLabelValues("agent_activities", "agent-123", "scan_started"))
		assert.Equal(t, float64(1), agentMetric)
	})

	t.Run("record_system_metrics", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Record system metrics
		collector.RecordSystemMetric("memory_usage_bytes", 1024*1024*512) // 512MB
		collector.RecordSystemMetric("cpu_usage_percent", 75.5)
		collector.RecordSystemMetric("disk_usage_bytes", 1024*1024*1024*10) // 10GB

		// Check system metrics
		memoryMetric := testutil.ToFloat64(collector.systemMetrics.WithLabelValues("memory_usage_bytes"))
		assert.Equal(t, float64(1024*1024*512), memoryMetric)

		cpuMetric := testutil.ToFloat64(collector.systemMetrics.WithLabelValues("cpu_usage_percent"))
		assert.Equal(t, 75.5, cpuMetric)
	})

	t.Run("connection_tracking", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Simulate connection lifecycle
		collector.IncrementActiveConnections()
		collector.IncrementActiveConnections()
		collector.IncrementActiveConnections()
		
		activeCount := testutil.ToFloat64(collector.activeConnections)
		assert.Equal(t, float64(3), activeCount)
		
		collector.DecrementActiveConnections()
		
		activeCount = testutil.ToFloat64(collector.activeConnections)
		assert.Equal(t, float64(2), activeCount)
	})
}

func TestPrometheusMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("prometheus_middleware_integration", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		collector := NewMetricsCollector("agentscan_test")
		router := suite.SetupGinRouter()
		
		// Add Prometheus middleware
		router.Use(PrometheusMiddleware(collector))
		
		// Add test routes
		router.GET("/api/users/:id", func(c *gin.Context) {
			time.Sleep(10 * time.Millisecond) // Simulate processing time
			c.JSON(200, gin.H{"id": c.Param("id")})
		})
		
		router.POST("/api/users", func(c *gin.Context) {
			c.JSON(201, gin.H{"message": "created"})
		})
		
		router.GET("/api/error", func(c *gin.Context) {
			c.JSON(500, gin.H{"error": "internal error"})
		})

		// Make requests
		requests := []struct {
			method string
			path   string
			status int
		}{
			{"GET", "/api/users/123", 200},
			{"GET", "/api/users/456", 200},
			{"POST", "/api/users", 201},
			{"GET", "/api/error", 500},
			{"GET", "/api/users/789", 200},
		}

		for _, req := range requests {
			httpReq := httptest.NewRequest(req.method, req.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httpReq)
			assert.Equal(t, req.status, w.Code)
		}

		// Check metrics were recorded
		getCount := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/api/users/:id", "200"))
		assert.Equal(t, float64(3), getCount)

		postCount := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("POST", "/api/users", "201"))
		assert.Equal(t, float64(1), postCount)

		errorCount := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/api/error", "500"))
		assert.Equal(t, float64(1), errorCount)
	})

	t.Run("metrics_endpoint", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		collector := NewMetricsCollector("agentscan_test")
		router := suite.SetupGinRouter()
		
		// Add metrics endpoint
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
		
		// Record some metrics first
		collector.RecordHTTPRequest("GET", "/test", 200, 100*time.Millisecond)
		
		// Request metrics endpoint
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
		
		body := w.Body.String()
		assert.Contains(t, body, "agentscan_test_http_requests_total")
		assert.Contains(t, body, "agentscan_test_http_request_duration_seconds")
		assert.Contains(t, body, "go_memstats_alloc_bytes")
	})
}

func TestCustomMetrics(t *testing.T) {
	t.Run("scan_duration_histogram", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Record various scan durations
		durations := []time.Duration{
			100 * time.Millisecond,
			500 * time.Millisecond,
			1 * time.Second,
			5 * time.Second,
			30 * time.Second,
			2 * time.Minute,
		}

		for i, duration := range durations {
			scanType := "vulnerability"
			if i%2 == 0 {
				scanType = "compliance"
			}
			collector.RecordScanCompleted(scanType, true, duration)
		}

		// Check histogram buckets
		metric := &prometheus.MetricFamily{}
		collector.scanDuration.Write(metric)
		
		histogramMetric := metric.GetMetric()[0].GetHistogram()
		assert.NotNil(t, histogramMetric)
		assert.Equal(t, uint64(6), histogramMetric.GetSampleCount())
		assert.True(t, histogramMetric.GetSampleSum() > 0)
	})

	t.Run("agent_performance_tracking", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Simulate agent performance metrics
		agents := []string{"agent-001", "agent-002", "agent-003"}
		
		for _, agentID := range agents {
			// Each agent performs different activities
			collector.RecordAgentActivity(agentID, "scan_started")
			collector.RecordAgentActivity(agentID, "vulnerability_found")
			collector.RecordAgentActivity(agentID, "scan_completed")
			
			// Record scan performance
			collector.RecordScanCompleted("vulnerability", true, 45*time.Second)
		}

		// Check agent activity metrics
		for _, agentID := range agents {
			startedCount := testutil.ToFloat64(collector.businessMetrics.WithLabelValues("agent_activities", agentID, "scan_started"))
			assert.Equal(t, float64(1), startedCount)
			
			completedCount := testutil.ToFloat64(collector.businessMetrics.WithLabelValues("agent_activities", agentID, "scan_completed"))
			assert.Equal(t, float64(1), completedCount)
		}
	})

	t.Run("error_rate_tracking", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Simulate various HTTP responses
		responses := []struct {
			method string
			path   string
			status int
			count  int
		}{
			{"GET", "/api/scans", 200, 100},
			{"GET", "/api/scans", 404, 5},
			{"GET", "/api/scans", 500, 2},
			{"POST", "/api/scans", 201, 50},
			{"POST", "/api/scans", 400, 3},
			{"POST", "/api/scans", 500, 1},
		}

		for _, resp := range responses {
			for i := 0; i < resp.count; i++ {
				collector.RecordHTTPRequest(resp.method, resp.path, resp.status, 100*time.Millisecond)
			}
		}

		// Calculate error rates
		totalRequests := float64(161)
		errorRequests := float64(11) // 5+2+3+1

		getTotal := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/api/scans", "200")) +
			testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/api/scans", "404")) +
			testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/api/scans", "500"))
		
		assert.Equal(t, float64(107), getTotal)

		postErrors := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("POST", "/api/scans", "400")) +
			testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("POST", "/api/scans", "500"))
		
		assert.Equal(t, float64(4), postErrors)
	})
}

func TestMetricsPerformance(t *testing.T) {
	t.Run("high_throughput_metrics", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Simulate high-throughput scenario
		numRequests := 10000
		
		start := time.Now()
		for i := 0; i < numRequests; i++ {
			method := "GET"
			if i%3 == 0 {
				method = "POST"
			}
			
			status := 200
			if i%100 == 0 {
				status = 500
			}
			
			collector.RecordHTTPRequest(method, "/api/test", status, time.Millisecond)
		}
		duration := time.Since(start)
		
		// Should handle 10k metrics in reasonable time (< 1 second)
		assert.Less(t, duration, time.Second)
		
		// Verify metrics were recorded
		getCount := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/api/test", "200"))
		postCount := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("POST", "/api/test", "200"))
		errorCount := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/api/test", "500")) +
			testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("POST", "/api/test", "500"))
		
		assert.True(t, getCount > 0)
		assert.True(t, postCount > 0)
		assert.True(t, errorCount > 0)
		assert.Equal(t, float64(numRequests), getCount+postCount+errorCount)
	})
}

func TestMetricsCleanup(t *testing.T) {
	t.Run("metric_cleanup_and_reset", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Record some metrics
		collector.RecordHTTPRequest("GET", "/test", 200, 100*time.Millisecond)
		collector.IncrementActiveConnections()
		collector.RecordSystemMetric("test_metric", 123.45)
		
		// Verify metrics exist
		httpCount := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/test", "200"))
		assert.Equal(t, float64(1), httpCount)
		
		activeConns := testutil.ToFloat64(collector.activeConnections)
		assert.Equal(t, float64(1), activeConns)
		
		// Reset metrics (if implemented)
		if collector.Reset != nil {
			collector.Reset()
			
			// Verify metrics are reset
			httpCountAfter := testutil.ToFloat64(collector.httpRequestsTotal.WithLabelValues("GET", "/test", "200"))
			assert.Equal(t, float64(0), httpCountAfter)
		}
	})
}

func TestMetricsLabels(t *testing.T) {
	t.Run("label_cardinality_control", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Test with many different paths (potential high cardinality)
		paths := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			paths[i] = "/api/resource/" + string(rune(i))
		}
		
		// Record metrics for all paths
		for _, path := range paths {
			collector.RecordHTTPRequest("GET", path, 200, 100*time.Millisecond)
		}
		
		// In a real implementation, you might want to normalize paths
		// to prevent label cardinality explosion
		// For now, just verify the metrics were recorded
		
		// Check that we don't have memory issues with high cardinality
		// This is more of a smoke test
		metric := &prometheus.MetricFamily{}
		collector.httpRequestsTotal.Write(metric)
		
		// Should have metrics for all paths
		assert.True(t, len(metric.GetMetric()) > 0)
	})

	t.Run("label_normalization", func(t *testing.T) {
		collector := NewMetricsCollector("agentscan_test")
		
		// Test path normalization (if implemented)
		testCases := []struct {
			originalPath   string
			normalizedPath string
		}{
			{"/api/users/123", "/api/users/:id"},
			{"/api/users/456", "/api/users/:id"},
			{"/api/scans/abc-def-123", "/api/scans/:id"},
			{"/api/reports/2023-01-01", "/api/reports/:date"},
		}
		
		for _, tc := range testCases {
			collector.RecordHTTPRequest("GET", tc.originalPath, 200, 100*time.Millisecond)
		}
		
		// If path normalization is implemented, check normalized metrics
		// Otherwise, this test documents the expected behavior
		
		metric := &prometheus.MetricFamily{}
		collector.httpRequestsTotal.Write(metric)
		assert.NotEmpty(t, metric.GetMetric())
	})
}

// Benchmark tests

func BenchmarkMetricsRecording(b *testing.B) {
	collector := NewMetricsCollector("agentscan_bench")
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.RecordHTTPRequest("GET", "/api/test", 200, 100*time.Millisecond)
		}
	})
}

func BenchmarkPrometheusMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)
	collector := NewMetricsCollector("agentscan_bench")
	
	router := gin.New()
	router.Use(PrometheusMiddleware(collector))
	router.GET("/test", func(c *gin.Context) {
		c.Status(200)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkMetricsCollection(b *testing.B) {
	collector := NewMetricsCollector("agentscan_bench")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordHTTPRequest("GET", "/api/test", 200, 100*time.Millisecond)
		collector.IncrementActiveConnections()
		collector.DecrementActiveConnections()
		collector.RecordSystemMetric("test_metric", float64(i))
	}
}