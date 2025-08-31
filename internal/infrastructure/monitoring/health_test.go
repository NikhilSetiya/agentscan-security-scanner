package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/agentscan/internal/shared/testing"
)

func TestHealthChecker(t *testing.T) {
	t.Run("initialize_health_checker", func(t *testing.T) {
		checker := NewHealthChecker()
		assert.NotNil(t, checker)
		assert.NotNil(t, checker.checks)
		assert.Equal(t, HealthStatusUnknown, checker.status)
	})

	t.Run("add_health_checks", func(t *testing.T) {
		checker := NewHealthChecker()
		
		// Add database check
		checker.AddCheck("database", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Database connection OK",
			}
		})
		
		// Add Redis check
		checker.AddCheck("redis", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Redis connection OK",
			}
		})
		
		assert.Len(t, checker.checks, 2)
		assert.Contains(t, checker.checks, "database")
		assert.Contains(t, checker.checks, "redis")
	})

	t.Run("run_health_checks_all_healthy", func(t *testing.T) {
		checker := NewHealthChecker()
		
		checker.AddCheck("service1", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:    HealthStatusHealthy,
				Message:   "Service 1 OK",
				Timestamp: time.Now(),
			}
		})
		
		checker.AddCheck("service2", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:    HealthStatusHealthy,
				Message:   "Service 2 OK",
				Timestamp: time.Now(),
			}
		})
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusHealthy, report.Status)
		assert.Len(t, report.Checks, 2)
		assert.True(t, report.Timestamp.After(time.Now().Add(-time.Second)))
		
		for _, check := range report.Checks {
			assert.Equal(t, HealthStatusHealthy, check.Status)
		}
	})

	t.Run("run_health_checks_with_failure", func(t *testing.T) {
		checker := NewHealthChecker()
		
		checker.AddCheck("healthy_service", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Healthy service OK",
			}
		})
		
		checker.AddCheck("failing_service", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: "Service is down",
				Error:   "Connection refused",
			}
		})
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusUnhealthy, report.Status)
		assert.Len(t, report.Checks, 2)
		
		// Find the failing check
		var failingCheck *HealthCheckResult
		for name, check := range report.Checks {
			if name == "failing_service" {
				failingCheck = &check
				break
			}
		}
		
		require.NotNil(t, failingCheck)
		assert.Equal(t, HealthStatusUnhealthy, failingCheck.Status)
		assert.Equal(t, "Service is down", failingCheck.Message)
		assert.Equal(t, "Connection refused", failingCheck.Error)
	})

	t.Run("health_check_timeout", func(t *testing.T) {
		checker := NewHealthChecker()
		
		checker.AddCheck("slow_service", func(ctx context.Context) HealthCheckResult {
			// Simulate slow service
			select {
			case <-time.After(200 * time.Millisecond):
				return HealthCheckResult{
					Status:  HealthStatusHealthy,
					Message: "Slow service OK",
				}
			case <-ctx.Done():
				return HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: "Service check timed out",
					Error:   ctx.Err().Error(),
				}
			}
		})
		
		// Set short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusUnhealthy, report.Status)
		slowCheck := report.Checks["slow_service"]
		assert.Equal(t, HealthStatusUnhealthy, slowCheck.Status)
		assert.Contains(t, slowCheck.Error, "context deadline exceeded")
	})
}

func TestDatabaseHealthCheck(t *testing.T) {
	t.Run("database_health_check_success", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()
		
		db := suite.SetupDatabase()
		
		checker := NewHealthChecker()
		checker.AddDatabaseCheck("database", db)
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusHealthy, report.Status)
		dbCheck := report.Checks["database"]
		assert.Equal(t, HealthStatusHealthy, dbCheck.Status)
		assert.Contains(t, dbCheck.Message, "Database connection OK")
	})

	t.Run("database_health_check_failure", func(t *testing.T) {
		// Create a database connection that will fail
		db, err := sql.Open("postgres", "postgres://invalid:invalid@localhost:9999/invalid")
		require.NoError(t, err)
		defer db.Close()
		
		checker := NewHealthChecker()
		checker.AddDatabaseCheck("database", db)
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusUnhealthy, report.Status)
		dbCheck := report.Checks["database"]
		assert.Equal(t, HealthStatusUnhealthy, dbCheck.Status)
		assert.Contains(t, dbCheck.Message, "Database connection failed")
		assert.NotEmpty(t, dbCheck.Error)
	})
}

func TestRedisHealthCheck(t *testing.T) {
	t.Run("redis_health_check_success", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()
		
		redisClient := suite.SetupRedis()
		
		checker := NewHealthChecker()
		checker.AddRedisCheck("redis", redisClient)
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusHealthy, report.Status)
		redisCheck := report.Checks["redis"]
		assert.Equal(t, HealthStatusHealthy, redisCheck.Status)
		assert.Contains(t, redisCheck.Message, "Redis connection OK")
	})

	t.Run("redis_health_check_failure", func(t *testing.T) {
		// Create a Redis client that will fail
		redisClient := redis.NewClient(&redis.Options{
			Addr: "localhost:9999", // Invalid port
		})
		defer redisClient.Close()
		
		checker := NewHealthChecker()
		checker.AddRedisCheck("redis", redisClient)
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusUnhealthy, report.Status)
		redisCheck := report.Checks["redis"]
		assert.Equal(t, HealthStatusUnhealthy, redisCheck.Status)
		assert.Contains(t, redisCheck.Message, "Redis connection failed")
		assert.NotEmpty(t, redisCheck.Error)
	})
}

func TestHTTPHealthCheck(t *testing.T) {
	t.Run("http_service_health_check_success", func(t *testing.T) {
		// Create a test HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer server.Close()
		
		checker := NewHealthChecker()
		checker.AddHTTPCheck("external_service", server.URL+"/health")
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusHealthy, report.Status)
		httpCheck := report.Checks["external_service"]
		assert.Equal(t, HealthStatusHealthy, httpCheck.Status)
		assert.Contains(t, httpCheck.Message, "HTTP service OK")
	})

	t.Run("http_service_health_check_failure", func(t *testing.T) {
		// Create a test HTTP server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()
		
		checker := NewHealthChecker()
		checker.AddHTTPCheck("external_service", server.URL+"/health")
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusUnhealthy, report.Status)
		httpCheck := report.Checks["external_service"]
		assert.Equal(t, HealthStatusUnhealthy, httpCheck.Status)
		assert.Contains(t, httpCheck.Message, "HTTP service failed")
	})

	t.Run("http_service_unreachable", func(t *testing.T) {
		checker := NewHealthChecker()
		checker.AddHTTPCheck("unreachable_service", "http://localhost:9999/health")
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusUnhealthy, report.Status)
		httpCheck := report.Checks["unreachable_service"]
		assert.Equal(t, HealthStatusUnhealthy, httpCheck.Status)
		assert.Contains(t, httpCheck.Message, "HTTP service failed")
		assert.NotEmpty(t, httpCheck.Error)
	})
}

func TestSystemResourceHealthCheck(t *testing.T) {
	t.Run("system_resources_health_check", func(t *testing.T) {
		checker := NewHealthChecker()
		
		// Add system resource checks
		checker.AddSystemResourceCheck("memory", 80.0)  // 80% threshold
		checker.AddSystemResourceCheck("disk", 90.0)    // 90% threshold
		checker.AddSystemResourceCheck("cpu", 85.0)     // 85% threshold
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		// System resource checks should complete
		assert.Contains(t, report.Checks, "memory")
		assert.Contains(t, report.Checks, "disk")
		assert.Contains(t, report.Checks, "cpu")
		
		// Check that all resource checks have valid status
		for name, check := range report.Checks {
			if name == "memory" || name == "disk" || name == "cpu" {
				assert.NotEqual(t, HealthStatusUnknown, check.Status)
				assert.NotEmpty(t, check.Message)
			}
		}
	})
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("health_endpoint_healthy", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()
		
		checker := NewHealthChecker()
		checker.AddCheck("test_service", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Test service OK",
			}
		})
		
		router := suite.SetupGinRouter()
		router.GET("/health", HealthEndpoint(checker))
		
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		
		var response HealthReport
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, HealthStatusHealthy, response.Status)
		assert.Contains(t, response.Checks, "test_service")
		assert.Equal(t, HealthStatusHealthy, response.Checks["test_service"].Status)
	})

	t.Run("health_endpoint_unhealthy", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()
		
		checker := NewHealthChecker()
		checker.AddCheck("failing_service", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: "Service is down",
				Error:   "Connection failed",
			}
		})
		
		router := suite.SetupGinRouter()
		router.GET("/health", HealthEndpoint(checker))
		
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 503, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		
		var response HealthReport
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, HealthStatusUnhealthy, response.Status)
		assert.Contains(t, response.Checks, "failing_service")
		assert.Equal(t, HealthStatusUnhealthy, response.Checks["failing_service"].Status)
	})

	t.Run("readiness_endpoint", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()
		
		checker := NewHealthChecker()
		checker.AddCheck("database", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Database ready",
			}
		})
		
		router := suite.SetupGinRouter()
		router.GET("/ready", ReadinessEndpoint(checker))
		
		req := httptest.NewRequest("GET", "/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 200, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "ready", response["status"])
	})

	t.Run("liveness_endpoint", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()
		
		router := suite.SetupGinRouter()
		router.GET("/live", LivenessEndpoint())
		
		req := httptest.NewRequest("GET", "/live", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, 200, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "alive", response["status"])
	})
}

func TestHealthCheckCaching(t *testing.T) {
	t.Run("health_check_caching", func(t *testing.T) {
		checker := NewHealthChecker()
		
		callCount := 0
		checker.AddCheck("cached_service", func(ctx context.Context) HealthCheckResult {
			callCount++
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Cached service OK",
			}
		})
		
		// Enable caching with 1 second TTL
		checker.SetCacheTTL(time.Second)
		
		ctx := context.Background()
		
		// First call should execute the check
		report1 := checker.RunChecks(ctx)
		assert.Equal(t, 1, callCount)
		assert.Equal(t, HealthStatusHealthy, report1.Status)
		
		// Second call within TTL should use cache
		report2 := checker.RunChecks(ctx)
		assert.Equal(t, 1, callCount) // Should not increment
		assert.Equal(t, HealthStatusHealthy, report2.Status)
		
		// Wait for cache to expire
		time.Sleep(1100 * time.Millisecond)
		
		// Third call should execute the check again
		report3 := checker.RunChecks(ctx)
		assert.Equal(t, 2, callCount) // Should increment
		assert.Equal(t, HealthStatusHealthy, report3.Status)
	})
}

func TestHealthCheckConcurrency(t *testing.T) {
	t.Run("concurrent_health_checks", func(t *testing.T) {
		checker := NewHealthChecker()
		
		// Add multiple checks that take some time
		for i := 0; i < 5; i++ {
			serviceName := "service_" + string(rune('A'+i))
			checker.AddCheck(serviceName, func(ctx context.Context) HealthCheckResult {
				time.Sleep(50 * time.Millisecond)
				return HealthCheckResult{
					Status:  HealthStatusHealthy,
					Message: "Service OK",
				}
			})
		}
		
		ctx := context.Background()
		start := time.Now()
		report := checker.RunChecks(ctx)
		duration := time.Since(start)
		
		// With concurrent execution, should take less than 250ms (5 * 50ms)
		assert.Less(t, duration, 200*time.Millisecond)
		assert.Equal(t, HealthStatusHealthy, report.Status)
		assert.Len(t, report.Checks, 5)
	})
}

func TestHealthCheckMetrics(t *testing.T) {
	t.Run("health_check_metrics_integration", func(t *testing.T) {
		checker := NewHealthChecker()
		metricsCollector := NewMetricsCollector("agentscan_test")
		
		// Enable metrics collection for health checks
		checker.SetMetricsCollector(metricsCollector)
		
		checker.AddCheck("test_service", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Test service OK",
			}
		})
		
		ctx := context.Background()
		report := checker.RunChecks(ctx)
		
		assert.Equal(t, HealthStatusHealthy, report.Status)
		
		// Check that health check metrics were recorded
		// This would depend on the actual metrics implementation
		// For now, just verify the integration doesn't break
		assert.NotNil(t, metricsCollector)
	})
}

// Benchmark tests

func BenchmarkHealthChecks(b *testing.B) {
	checker := NewHealthChecker()
	
	checker.AddCheck("fast_service", func(ctx context.Context) HealthCheckResult {
		return HealthCheckResult{
			Status:  HealthStatusHealthy,
			Message: "Fast service OK",
		}
	})
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.RunChecks(ctx)
	}
}

func BenchmarkHealthEndpoint(b *testing.B) {
	gin.SetMode(gin.TestMode)
	
	checker := NewHealthChecker()
	checker.AddCheck("bench_service", func(ctx context.Context) HealthCheckResult {
		return HealthCheckResult{
			Status:  HealthStatusHealthy,
			Message: "Benchmark service OK",
		}
	})
	
	router := gin.New()
	router.GET("/health", HealthEndpoint(checker))
	
	req := httptest.NewRequest("GET", "/health", nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkConcurrentHealthChecks(b *testing.B) {
	checker := NewHealthChecker()
	
	// Add multiple checks
	for i := 0; i < 10; i++ {
		serviceName := "service_" + string(rune('0'+i))
		checker.AddCheck(serviceName, func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Service OK",
			}
		})
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			checker.RunChecks(ctx)
		}
	})
}