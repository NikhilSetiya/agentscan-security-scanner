package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/alerting"
	"github.com/agentscan/agentscan/pkg/health"
	"github.com/agentscan/agentscan/pkg/logging"
	"github.com/agentscan/agentscan/pkg/metrics"
	"github.com/agentscan/agentscan/pkg/tracing"
)

// Service provides comprehensive observability functionality
type Service struct {
	logger   *logging.Logger
	metrics  *metrics.Metrics
	health   *health.Service
	tracing  *tracing.TracingService
	alerting *alerting.Service
	config   *Config
}

// Config holds observability configuration
type Config struct {
	ServiceName    string                `json:"service_name"`
	ServiceVersion string                `json:"service_version"`
	Environment    string                `json:"environment"`
	Logging        *logging.Config       `json:"logging"`
	Metrics        *metrics.Config       `json:"metrics"`
	Health         *health.Config        `json:"health"`
	Tracing        *tracing.Config       `json:"tracing"`
	Alerting       *alerting.Config      `json:"alerting"`
}

// DefaultConfig returns default observability configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "agentscan",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Logging:        nil, // Will use defaults
		Metrics:        nil, // Will use defaults
		Health:         nil, // Will use defaults
		Tracing:        nil, // Will use defaults
		Alerting:       nil, // Will use defaults
	}
}

// NewService creates a new observability service
func NewService(config *Config) (*Service, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize logging
	if config.Logging == nil {
		config.Logging = &logging.Config{
			Level:       "info",
			Format:      "json",
			Output:      "stdout",
			ServiceName: config.ServiceName,
			Version:     config.ServiceVersion,
		}
	}

	logger, err := logging.NewLogger(config.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize metrics
	if config.Metrics == nil {
		config.Metrics = metrics.DefaultConfig()
		config.Metrics.Namespace = config.ServiceName
	}

	metricsService := metrics.NewMetrics(config.Metrics)

	// Initialize health checks
	if config.Health == nil {
		config.Health = health.DefaultConfig()
		config.Health.Metadata = map[string]string{
			"service":     config.ServiceName,
			"version":     config.ServiceVersion,
			"environment": config.Environment,
		}
	}

	healthService := health.NewService(logger, config.Health)

	// Initialize tracing
	if config.Tracing == nil {
		config.Tracing = tracing.DefaultConfig()
		config.Tracing.ServiceName = config.ServiceName
		config.Tracing.ServiceVersion = config.ServiceVersion
		config.Tracing.Environment = config.Environment
	}

	tracingService, err := tracing.NewTracingService(config.Tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize alerting
	if config.Alerting == nil {
		config.Alerting = alerting.DefaultConfig()
	}

	alertingService := alerting.NewService(logger, config.Alerting)

	// Add predefined alert rules
	for _, rule := range alerting.PredefinedAlerts {
		alertingService.AddRule(rule)
	}

	return &Service{
		logger:   logger,
		metrics:  metricsService,
		health:   healthService,
		tracing:  tracingService,
		alerting: alertingService,
		config:   config,
	}, nil
}

// Logger returns the logger instance
func (s *Service) Logger() *logging.Logger {
	return s.logger
}

// Metrics returns the metrics instance
func (s *Service) Metrics() *metrics.Metrics {
	return s.metrics
}

// Health returns the health service instance
func (s *Service) Health() *health.Service {
	return s.health
}

// Tracing returns the tracing service instance
func (s *Service) Tracing() *tracing.TracingService {
	return s.tracing
}

// Alerting returns the alerting service instance
func (s *Service) Alerting() *alerting.Service {
	return s.alerting
}

// SetupHealthChecks sets up standard health checks
func (s *Service) SetupHealthChecks(db *database.DB, redis *queue.RedisClient) {
	// Database health check
	if db != nil {
		s.health.RegisterChecker("database", health.NewDatabaseChecker(db, "PostgreSQL"))
	}

	// Redis health check
	if redis != nil {
		s.health.RegisterChecker("redis", health.NewRedisChecker(redis, "Redis"))
	}

	// Custom application health check
	s.health.RegisterChecker("application", health.NewCustomChecker(
		"application",
		func(ctx context.Context) (health.Status, string, error) {
			// Check if the application is ready to serve requests
			return health.StatusHealthy, "Application is healthy", nil
		},
	))

	// Disk space health check
	s.health.RegisterChecker("disk_space", health.NewDiskSpaceChecker(
		"/tmp",
		"Temporary directory disk space",
		0.9, // Alert when 90% full
	))
}

// SetupAlertChannels sets up notification channels for alerts
func (s *Service) SetupAlertChannels(slackWebhookURL, emailSMTPHost string, emailSMTPPort int, emailUsername, emailPassword, emailFrom string, emailTo []string) {
	// Slack channel
	if slackWebhookURL != "" {
		slackChannel := alerting.NewSlackChannel(
			slackWebhookURL,
			"#alerts",
			"AgentScan Bot",
			":warning:",
		)
		s.alerting.AddChannel(slackChannel)
	}

	// Email channel
	if emailSMTPHost != "" && len(emailTo) > 0 {
		emailChannel := alerting.NewEmailChannel(
			emailSMTPHost,
			emailSMTPPort,
			emailUsername,
			emailPassword,
			emailFrom,
			emailTo,
		)
		s.alerting.AddChannel(emailChannel)
	}
}

// SetupMiddleware sets up observability middleware for Gin
func (s *Service) SetupMiddleware(router *gin.Engine) {
	// Logging middleware
	router.Use(s.logger.LoggingMiddleware())
	router.Use(s.logger.ErrorLoggingMiddleware())
	router.Use(s.logger.RecoveryMiddleware())

	// Metrics middleware
	router.Use(s.metrics.PrometheusMiddleware())

	// Tracing middleware
	router.Use(s.tracing.TracingMiddleware())
}

// SetupRoutes sets up observability endpoints
func (s *Service) SetupRoutes(router *gin.Engine) {
	// Health check endpoints
	router.GET("/health", s.health.Handler())
	router.GET("/health/live", s.health.LivenessHandler())
	router.GET("/health/ready", s.health.ReadinessHandler())

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(s.metrics.Handler()))

	// Observability API group
	obs := router.Group("/api/v1/observability")
	{
		obs.GET("/alerts", s.getAlertsHandler())
		obs.POST("/alerts/:id/resolve", s.resolveAlertHandler())
		obs.GET("/health", s.getHealthHandler())
		obs.GET("/metrics/summary", s.getMetricsSummaryHandler())
	}
}

// getAlertsHandler returns active alerts
func (s *Service) getAlertsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		alerts := s.alerting.GetActiveAlerts()
		c.JSON(http.StatusOK, gin.H{
			"alerts": alerts,
			"count":  len(alerts),
		})
	}
}

// resolveAlertHandler resolves an alert
func (s *Service) resolveAlertHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		alertID := c.Param("id")
		if alertID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Alert ID is required",
			})
			return
		}

		err := s.alerting.ResolveAlert(c.Request.Context(), alertID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Alert resolved successfully",
		})
	}
}

// getHealthHandler returns detailed health information
func (s *Service) getHealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		healthResponse := s.health.CheckHealth(ctx)
		
		statusCode := http.StatusOK
		switch healthResponse.Status {
		case health.StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		case health.StatusDegraded:
			statusCode = http.StatusPartialContent
		}

		c.JSON(statusCode, healthResponse)
	}
}

// getMetricsSummaryHandler returns a summary of key metrics
func (s *Service) getMetricsSummaryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This would collect current metric values
		// For now, return a placeholder response
		c.JSON(http.StatusOK, gin.H{
			"summary": map[string]interface{}{
				"http_requests_total":     "Available at /metrics",
				"scan_duration_seconds":   "Available at /metrics",
				"database_connections":    "Available at /metrics",
				"active_alerts":          len(s.alerting.GetActiveAlerts()),
			},
			"endpoints": map[string]string{
				"metrics":    "/metrics",
				"health":     "/health",
				"alerts":     "/api/v1/observability/alerts",
			},
		})
	}
}

// MonitorSystemHealth continuously monitors system health and triggers alerts
func (s *Service) MonitorSystemHealth(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAndTriggerAlerts(ctx)
		}
	}
}

// checkAndTriggerAlerts checks system health and triggers alerts if needed
func (s *Service) checkAndTriggerAlerts(ctx context.Context) {
	healthResponse := s.health.CheckHealth(ctx)

	// Check for unhealthy components
	for name, check := range healthResponse.Checks {
		if check.Status == health.StatusUnhealthy {
			alert := &alerting.Alert{
				ID:          fmt.Sprintf("health_check_%s", name),
				Title:       fmt.Sprintf("Health check failed: %s", name),
				Description: fmt.Sprintf("Health check for %s failed: %s", name, check.Message),
				Severity:    alerting.SeverityCritical,
				Component:   name,
				Labels: map[string]string{
					"check_name": name,
					"category":   "health",
				},
				Annotations: map[string]string{
					"error":    check.Error,
					"duration": check.Duration.String(),
				},
			}

			s.alerting.TriggerAlert(ctx, alert)
		} else if check.Status == health.StatusDegraded {
			alert := &alerting.Alert{
				ID:          fmt.Sprintf("health_check_%s_degraded", name),
				Title:       fmt.Sprintf("Health check degraded: %s", name),
				Description: fmt.Sprintf("Health check for %s is degraded: %s", name, check.Message),
				Severity:    alerting.SeverityWarning,
				Component:   name,
				Labels: map[string]string{
					"check_name": name,
					"category":   "health",
				},
				Annotations: map[string]string{
					"message":  check.Message,
					"duration": check.Duration.String(),
				},
			}

			s.alerting.TriggerAlert(ctx, alert)
		}
	}
}

// Shutdown gracefully shuts down the observability service
func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.WithContext(ctx).Info("Shutting down observability service")

	// Shutdown tracing
	if err := s.tracing.Shutdown(ctx); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to shutdown tracing service")
		return err
	}

	s.logger.WithContext(ctx).Info("Observability service shutdown complete")
	return nil
}

// RecordBusinessMetric records a business-specific metric
func (s *Service) RecordBusinessMetric(metricType string, labels map[string]string, value float64) {
	switch metricType {
	case "scan_completed":
		if status, ok := labels["status"]; ok {
			if repo, ok := labels["repository"]; ok {
				if branch, ok := labels["branch"]; ok {
					s.metrics.RecordScan(status, repo, branch, time.Duration(value)*time.Second)
				}
			}
		}
	case "finding_detected":
		if severity, ok := labels["severity"]; ok {
			if tool, ok := labels["tool"]; ok {
				if ruleID, ok := labels["rule_id"]; ok {
					if repo, ok := labels["repository"]; ok {
						s.metrics.RecordFinding(severity, tool, ruleID, repo)
					}
				}
			}
		}
	case "agent_execution":
		if agent, ok := labels["agent"]; ok {
			if status, ok := labels["status"]; ok {
				s.metrics.RecordAgentExecution(agent, status, time.Duration(value)*time.Second)
			}
		}
	}
}

// LogStructuredEvent logs a structured event with context
func (s *Service) LogStructuredEvent(ctx context.Context, event string, component string, fields map[string]interface{}) {
	logFields := logging.Fields{
		"event":     event,
		"component": component,
	}

	for k, v := range fields {
		logFields[k] = v
	}

	s.logger.WithContext(ctx).WithFields(logFields).Info("Structured event")
}