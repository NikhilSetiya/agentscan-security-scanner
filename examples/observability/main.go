package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agentscan/agentscan/internal/observability"
	"github.com/agentscan/agentscan/pkg/alerting"
	"github.com/agentscan/agentscan/pkg/health"
	"github.com/agentscan/agentscan/pkg/logging"
	"github.com/agentscan/agentscan/pkg/metrics"
	"github.com/agentscan/agentscan/pkg/tracing"
)

func main() {
	// Create observability configuration
	config := &observability.Config{
		ServiceName:    "agentscan-example",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Logging: &logging.Config{
			Level:       "info",
			Format:      "json",
			Output:      "stdout",
			ServiceName: "agentscan-example",
			Version:     "1.0.0",
		},
		Metrics: &metrics.Config{
			Namespace: "agentscan_example",
			Enabled:   true,
		},
		Health: &health.Config{
			Timeout: 5 * time.Second,
			Metadata: map[string]string{
				"service":     "agentscan-example",
				"version":     "1.0.0",
				"environment": "development",
			},
		},
		Tracing: &tracing.Config{
			ServiceName:     "agentscan-example",
			ServiceVersion:  "1.0.0",
			Environment:     "development",
			JaegerEndpoint:  "http://localhost:14268/api/traces",
			SamplingRate:    1.0,
			Enabled:         true,
		},
		Alerting: &alerting.Config{
			Enabled:           true,
			DefaultSeverity:   alerting.SeverityWarning,
			AlertTimeout:      5 * time.Minute,
			ResolutionTimeout: 15 * time.Minute,
			MaxAlerts:         100,
		},
	}

	// Initialize observability service
	obs, err := observability.NewService(config)
	if err != nil {
		log.Fatalf("Failed to initialize observability service: %v", err)
	}

	// Setup alert channels (optional - only if environment variables are set)
	slackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if slackWebhookURL != "" {
		obs.SetupAlertChannels(
			slackWebhookURL,
			"", 0, "", "", "", nil, // No email for this example
		)
	}

	// Create Gin router
	router := gin.New()

	// Setup observability middleware
	obs.SetupMiddleware(router)

	// Setup observability routes
	obs.SetupRoutes(router)

	// Add example routes
	setupExampleRoutes(router, obs)

	// Setup health checks
	setupHealthChecks(obs)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start background monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go obs.MonitorSystemHealth(ctx, 30*time.Second)

	// Start metrics collector
	metricsCollector := metrics.NewMetricsCollector(obs.Metrics(), 15*time.Second)
	go metricsCollector.Start(ctx)

	// Start server
	go func() {
		obs.Logger().WithFields(logging.Fields{
			"addr": server.Addr,
		}).Info("Starting HTTP server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			obs.Logger().WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	obs.Logger().Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		obs.Logger().WithError(err).Error("Server forced to shutdown")
	}

	// Shutdown observability service
	if err := obs.Shutdown(shutdownCtx); err != nil {
		obs.Logger().WithError(err).Error("Failed to shutdown observability service")
	}

	obs.Logger().Info("Server exited")
}

func setupExampleRoutes(router *gin.Engine, obs *observability.Service) {
	api := router.Group("/api/v1")

	// Example endpoint that demonstrates observability features
	api.GET("/example", func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Start a custom span
		ctx, span := obs.Tracing().StartSpan(ctx, "example_operation")
		defer span.End()

		// Log structured event
		obs.LogStructuredEvent(ctx, "example_request", "api", map[string]interface{}{
			"endpoint": "/api/v1/example",
			"method":   "GET",
		})

		// Record business metric
		obs.RecordBusinessMetric("api_request", map[string]string{
			"endpoint": "/api/v1/example",
			"method":   "GET",
		}, 1)

		// Simulate some work
		time.Sleep(100 * time.Millisecond)

		c.JSON(http.StatusOK, gin.H{
			"message":        "Example endpoint",
			"correlation_id": logging.GetCorrelationID(ctx),
			"trace_id":       tracing.GetTraceID(ctx),
			"span_id":        tracing.GetSpanID(ctx),
		})
	})

	// Example endpoint that triggers an error
	api.GET("/error", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Log error
		err := fmt.Errorf("example error")
		obs.Logger().LogError(ctx, err, "Example error occurred", logging.Fields{
			"endpoint": "/api/v1/error",
		})

		// Record error metric
		obs.Metrics().RecordError("api", "example_error")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          "Internal server error",
			"correlation_id": logging.GetCorrelationID(ctx),
		})
	})

	// Example endpoint that triggers an alert
	api.POST("/trigger-alert", func(c *gin.Context) {
		ctx := c.Request.Context()

		alert := &alerting.Alert{
			Title:       "Example Alert",
			Description: "This is an example alert triggered via API",
			Severity:    alerting.SeverityWarning,
			Component:   "api",
			Labels: map[string]string{
				"endpoint": "/api/v1/trigger-alert",
				"category": "example",
			},
			Annotations: map[string]string{
				"summary": "Example alert for demonstration",
			},
		}

		if err := obs.Alerting().TriggerAlert(ctx, alert); err != nil {
			obs.Logger().LogError(ctx, err, "Failed to trigger alert", nil)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to trigger alert",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Alert triggered successfully",
			"alert_id": alert.ID,
		})
	})

	// Example endpoint that simulates a slow operation
	api.GET("/slow", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Start a custom span
		ctx, span := obs.Tracing().StartSpan(ctx, "slow_operation")
		defer span.End()

		// Simulate slow work
		duration := 2 * time.Second
		time.Sleep(duration)

		// Record performance metric
		obs.Logger().LogPerformanceEvent(ctx, "slow_operation", duration, logging.Fields{
			"endpoint": "/api/v1/slow",
		})

		c.JSON(http.StatusOK, gin.H{
			"message":  "Slow operation completed",
			"duration": duration.String(),
		})
	})

	// Example endpoint that demonstrates database tracing
	api.GET("/database", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Start database span
		ctx, span := obs.Tracing().StartDatabaseSpan(ctx, "SELECT", "users")
		defer span.End()

		// Simulate database query
		start := time.Now()
		time.Sleep(50 * time.Millisecond)
		duration := time.Since(start)

		// Record database metrics
		obs.Metrics().RecordDatabaseQuery("SELECT", "users", duration)

		c.JSON(http.StatusOK, gin.H{
			"message":       "Database query completed",
			"query_time_ms": duration.Milliseconds(),
		})
	})

	// Example endpoint that demonstrates cache tracing
	api.GET("/cache", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Start cache span
		ctx, span := obs.Tracing().StartCacheSpan(ctx, "GET", "user:123")
		defer span.End()

		// Simulate cache operation
		start := time.Now()
		time.Sleep(10 * time.Millisecond)
		duration := time.Since(start)

		// Record cache metrics
		obs.Metrics().RecordCacheOperation("GET", "redis", duration)

		c.JSON(http.StatusOK, gin.H{
			"message":         "Cache operation completed",
			"operation_time_ms": duration.Milliseconds(),
		})
	})
}

func setupHealthChecks(obs *observability.Service) {
	// Add custom health checks
	obs.Health().RegisterChecker("external_service", health.NewHTTPChecker(
		"https://httpbin.org/status/200",
		"External Service",
		5*time.Second,
	))

	obs.Health().RegisterChecker("custom_check", health.NewCustomChecker(
		"custom_business_logic",
		func(ctx context.Context) (health.Status, string, error) {
			// Simulate some business logic check
			time.Sleep(10 * time.Millisecond)
			return health.StatusHealthy, "Business logic is healthy", nil
		},
	))
}