package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/logging"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity int

const (
	// SeverityInfo - informational alerts
	SeverityInfo AlertSeverity = iota
	// SeverityWarning - warning alerts that need attention
	SeverityWarning
	// SeverityError - error alerts that need immediate attention
	SeverityError
	// SeverityCritical - critical alerts that need urgent attention
	SeverityCritical
)

func (s AlertSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Alert represents an alert that needs to be sent
type Alert struct {
	ID          string                 `json:"id"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AlertHandler defines the interface for handling alerts
type AlertHandler interface {
	HandleAlert(ctx context.Context, alert Alert) error
	Name() string
}

// AlertManager manages alert generation and routing
type AlertManager struct {
	handlers []AlertHandler
	mutex    sync.RWMutex
	logger   *logging.Logger
	
	// Rate limiting
	alertCounts map[string]int
	lastReset   time.Time
	rateLimit   int
	resetInterval time.Duration
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		handlers:      make([]AlertHandler, 0),
		logger:        logging.GetLogger(),
		alertCounts:   make(map[string]int),
		lastReset:     time.Now(),
		rateLimit:     100, // 100 alerts per reset interval
		resetInterval: time.Hour,
	}
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	am.handlers = append(am.handlers, handler)
	am.logger.Info("Alert handler added", "handler", handler.Name())
}

// SendAlert sends an alert to all registered handlers
func (am *AlertManager) SendAlert(ctx context.Context, alert Alert) error {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	
	// Check rate limiting
	if !am.checkRateLimit(alert.Source) {
		am.logger.Warn("Alert rate limit exceeded",
			"source", alert.Source,
			"title", alert.Title,
		)
		return fmt.Errorf("alert rate limit exceeded for source: %s", alert.Source)
	}
	
	// Set timestamp if not provided
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}
	
	// Generate ID if not provided
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("%s-%d", alert.Source, alert.Timestamp.Unix())
	}
	
	am.logger.Info("Sending alert",
		"id", alert.ID,
		"severity", alert.Severity.String(),
		"source", alert.Source,
		"title", alert.Title,
	)
	
	var lastErr error
	successCount := 0
	
	for _, handler := range am.handlers {
		if err := handler.HandleAlert(ctx, alert); err != nil {
			am.logger.Error("Alert handler failed",
				"handler", handler.Name(),
				"alert_id", alert.ID,
				"error", err,
			)
			lastErr = err
		} else {
			successCount++
		}
	}
	
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("all alert handlers failed: %w", lastErr)
	}
	
	return nil
}

func (am *AlertManager) checkRateLimit(source string) bool {
	now := time.Now()
	
	// Reset counters if interval has passed
	if now.Sub(am.lastReset) >= am.resetInterval {
		am.alertCounts = make(map[string]int)
		am.lastReset = now
	}
	
	count := am.alertCounts[source]
	if count >= am.rateLimit {
		return false
	}
	
	am.alertCounts[source] = count + 1
	return true
}

// LoggingAlertHandler logs alerts to the application logger
type LoggingAlertHandler struct {
	logger *logging.Logger
}

// NewLoggingAlertHandler creates a new logging alert handler
func NewLoggingAlertHandler() *LoggingAlertHandler {
	return &LoggingAlertHandler{
		logger: logging.GetLogger(),
	}
}

// HandleAlert handles an alert by logging it
func (h *LoggingAlertHandler) HandleAlert(ctx context.Context, alert Alert) error {
	fields := []interface{}{
		"alert_id", alert.ID,
		"severity", alert.Severity.String(),
		"source", alert.Source,
		"title", alert.Title,
		"description", alert.Description,
		"timestamp", alert.Timestamp,
	}
	
	// Add tags as fields
	for key, value := range alert.Tags {
		fields = append(fields, fmt.Sprintf("tag_%s", key), value)
	}
	
	// Add metadata as fields
	for key, value := range alert.Metadata {
		fields = append(fields, fmt.Sprintf("meta_%s", key), value)
	}
	
	switch alert.Severity {
	case SeverityInfo:
		h.logger.Info("ALERT: "+alert.Title, fields...)
	case SeverityWarning:
		h.logger.Warn("ALERT: "+alert.Title, fields...)
	case SeverityError:
		h.logger.Error("ALERT: "+alert.Title, fields...)
	case SeverityCritical:
		h.logger.Error("CRITICAL ALERT: "+alert.Title, fields...)
	}
	
	return nil
}

// Name returns the name of the handler
func (h *LoggingAlertHandler) Name() string {
	return "logging"
}

// ErrorAlertGenerator generates alerts from errors
type ErrorAlertGenerator struct {
	alertManager *AlertManager
	logger       *logging.Logger
}

// NewErrorAlertGenerator creates a new error alert generator
func NewErrorAlertGenerator(alertManager *AlertManager) *ErrorAlertGenerator {
	return &ErrorAlertGenerator{
		alertManager: alertManager,
		logger:       logging.GetLogger(),
	}
}

// HandleError processes an error and generates appropriate alerts
func (eag *ErrorAlertGenerator) HandleError(ctx context.Context, err error, source string, metadata map[string]interface{}) {
	if err == nil {
		return
	}
	
	severity := eag.determineSeverity(err)
	
	alert := Alert{
		Severity:    severity,
		Title:       eag.generateTitle(err),
		Description: err.Error(),
		Source:      source,
		Tags:        eag.generateTags(err),
		Metadata:    metadata,
	}
	
	if alertErr := eag.alertManager.SendAlert(ctx, alert); alertErr != nil {
		eag.logger.Error("Failed to send error alert",
			"original_error", err,
			"alert_error", alertErr,
			"source", source,
		)
	}
}

func (eag *ErrorAlertGenerator) determineSeverity(err error) AlertSeverity {
	// Check for circuit breaker errors
	if IsCircuitBreakerError(err) {
		return SeverityError
	}
	
	// Check error types
	switch errors.GetType(err) {
	case errors.ErrorTypeTimeout:
		return SeverityWarning
	case errors.ErrorTypeExternal:
		return SeverityWarning
	case errors.ErrorTypeInternal:
		return SeverityError
	case errors.ErrorTypeValidation:
		return SeverityInfo
	case errors.ErrorTypeAuthentication, errors.ErrorTypeAuthorization:
		return SeverityWarning
	default:
		return SeverityError
	}
}

func (eag *ErrorAlertGenerator) generateTitle(err error) string {
	errorType := errors.GetType(err)
	errorCode := errors.GetCode(err)
	
	switch errorType {
	case errors.ErrorTypeTimeout:
		return "Operation Timeout"
	case errors.ErrorTypeExternal:
		return "External Service Error"
	case errors.ErrorTypeInternal:
		return "Internal System Error"
	case errors.ErrorTypeValidation:
		return "Validation Error"
	case errors.ErrorTypeAuthentication:
		return "Authentication Error"
	case errors.ErrorTypeAuthorization:
		return "Authorization Error"
	default:
		return fmt.Sprintf("Error: %s", errorCode)
	}
}

func (eag *ErrorAlertGenerator) generateTags(err error) map[string]string {
	tags := make(map[string]string)
	
	tags["error_type"] = string(errors.GetType(err))
	tags["error_code"] = errors.GetCode(err)
	
	// Add specific tags for circuit breaker errors
	if IsCircuitBreakerError(err) {
		tags["circuit_breaker"] = "true"
	}
	
	return tags
}

// SystemHealthMonitor monitors system health and generates alerts
type SystemHealthMonitor struct {
	alertManager       *AlertManager
	degradationManager *DegradationManager
	logger             *logging.Logger
	
	// Monitoring configuration
	checkInterval      time.Duration
	lastDegradationLevel DegradationLevel
	stopChan           chan struct{}
	running            bool
	mutex              sync.Mutex
}

// NewSystemHealthMonitor creates a new system health monitor
func NewSystemHealthMonitor(alertManager *AlertManager, degradationManager *DegradationManager) *SystemHealthMonitor {
	return &SystemHealthMonitor{
		alertManager:         alertManager,
		degradationManager:   degradationManager,
		logger:               logging.GetLogger(),
		checkInterval:        30 * time.Second,
		lastDegradationLevel: LevelNormal,
		stopChan:             make(chan struct{}),
	}
}

// Start starts the health monitoring
func (shm *SystemHealthMonitor) Start(ctx context.Context) {
	shm.mutex.Lock()
	defer shm.mutex.Unlock()
	
	if shm.running {
		return
	}
	
	shm.running = true
	go shm.monitorLoop(ctx)
	shm.logger.Info("System health monitor started")
}

// Stop stops the health monitoring
func (shm *SystemHealthMonitor) Stop() {
	shm.mutex.Lock()
	defer shm.mutex.Unlock()
	
	if !shm.running {
		return
	}
	
	close(shm.stopChan)
	shm.running = false
	shm.logger.Info("System health monitor stopped")
}

func (shm *SystemHealthMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(shm.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-shm.stopChan:
			return
		case <-ticker.C:
			shm.checkSystemHealth(ctx)
		}
	}
}

func (shm *SystemHealthMonitor) checkSystemHealth(ctx context.Context) {
	currentLevel := shm.degradationManager.GetCurrentDegradationLevel()
	
	// Check if degradation level has changed
	if currentLevel != shm.lastDegradationLevel {
		shm.sendDegradationAlert(ctx, shm.lastDegradationLevel, currentLevel)
		shm.lastDegradationLevel = currentLevel
	}
	
	// Check individual service health
	unhealthyServices := shm.degradationManager.GetUnhealthyServices()
	for _, service := range unhealthyServices {
		if serviceHealth, exists := shm.degradationManager.GetServiceHealth(service); exists {
			shm.sendServiceHealthAlert(ctx, serviceHealth)
		}
	}
}

func (shm *SystemHealthMonitor) sendDegradationAlert(ctx context.Context, from, to DegradationLevel) {
	var severity AlertSeverity
	switch to {
	case LevelNormal:
		severity = SeverityInfo
	case LevelPartial:
		severity = SeverityWarning
	case LevelSevere:
		severity = SeverityError
	case LevelCritical:
		severity = SeverityCritical
	}
	
	alert := Alert{
		Severity:    severity,
		Title:       "System Degradation Level Changed",
		Description: fmt.Sprintf("System degradation level changed from %s to %s", from.String(), to.String()),
		Source:      "system_health_monitor",
		Tags: map[string]string{
			"component":      "system",
			"previous_level": from.String(),
			"current_level":  to.String(),
		},
		Metadata: map[string]interface{}{
			"degradation_status": shm.degradationManager.GetAllServiceHealth(),
		},
	}
	
	if err := shm.alertManager.SendAlert(ctx, alert); err != nil {
		shm.logger.Error("Failed to send degradation alert", "error", err)
	}
}

func (shm *SystemHealthMonitor) sendServiceHealthAlert(ctx context.Context, serviceHealth *ServiceHealth) {
	alert := Alert{
		Severity:    SeverityError,
		Title:       "Service Health Alert",
		Description: fmt.Sprintf("Service '%s' is unhealthy: %s", serviceHealth.Name, serviceHealth.Message),
		Source:      "system_health_monitor",
		Tags: map[string]string{
			"component":    "service",
			"service_name": serviceHealth.Name,
			"healthy":      fmt.Sprintf("%t", serviceHealth.Healthy),
		},
		Metadata: map[string]interface{}{
			"error_count":   serviceHealth.ErrorCount,
			"response_time": serviceHealth.ResponseTime.String(),
			"last_check":    serviceHealth.LastCheck,
		},
	}
	
	if err := shm.alertManager.SendAlert(ctx, alert); err != nil {
		shm.logger.Error("Failed to send service health alert", "error", err)
	}
}