package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ObserveMCPConfig holds configuration for Observe MCP integration
type ObserveMCPConfig struct {
	Endpoint    string `json:"endpoint"`
	APIKey      string `json:"api_key"`
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
	Enabled     bool   `json:"enabled"`
}

// ObserveEvent represents a structured event for Observe
type ObserveEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Environment string                 `json:"environment"`
	ProjectID   string                 `json:"project_id"`
	EventType   string                 `json:"event_type"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data"`
	TraceID     string                 `json:"trace_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
}

// ObserveTrace represents a trace for operation tracking
type ObserveTrace struct {
	TraceID     string                 `json:"trace_id"`
	Operation   string                 `json:"operation"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	Success     bool                   `json:"success"`
	Metadata    map[string]interface{} `json:"metadata"`
	Environment string                 `json:"environment"`
	ProjectID   string                 `json:"project_id"`
}

// ObserveLogger provides structured logging to Observe MCP
type ObserveLogger struct {
	config     *ObserveMCPConfig
	httpClient *http.Client
	logger     *slog.Logger
}

// NewObserveLogger creates a new Observe MCP logger
func NewObserveLogger(config *ObserveMCPConfig, logger *slog.Logger) *ObserveLogger {
	return &ObserveLogger{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// LogEvent sends a structured event to Observe
func (o *ObserveLogger) LogEvent(ctx context.Context, eventType, level, message string, data map[string]interface{}) {
	if !o.config.Enabled {
		return
	}

	event := ObserveEvent{
		Timestamp:   time.Now(),
		Environment: o.config.Environment,
		ProjectID:   o.config.ProjectID,
		EventType:   eventType,
		Level:       level,
		Message:     message,
		Data:        o.sanitizeData(data),
		TraceID:     o.getTraceID(ctx),
		UserID:      o.getUserID(ctx),
		RequestID:   o.getRequestID(ctx),
	}

	go o.sendEvent(event)
}

// LogError logs an error with context
func (o *ObserveLogger) LogError(ctx context.Context, err error, context map[string]interface{}) {
	if !o.config.Enabled {
		return
	}

	// Get stack trace
	stack := make([]byte, 4096)
	length := runtime.Stack(stack, false)
	stackTrace := string(stack[:length])

	data := map[string]interface{}{
		"error":       err.Error(),
		"stack_trace": stackTrace,
		"context":     context,
	}

	o.LogEvent(ctx, "error", "error", err.Error(), data)
}

// LogAPICall logs an API request/response
func (o *ObserveLogger) LogAPICall(ctx context.Context, method, path string, statusCode int, duration time.Duration, requestBody, responseBody interface{}) {
	if !o.config.Enabled {
		return
	}

	data := map[string]interface{}{
		"method":        method,
		"path":          path,
		"status_code":   statusCode,
		"duration_ms":   duration.Milliseconds(),
		"request_body":  o.sanitizeData(requestBody),
		"response_body": o.sanitizeData(responseBody),
	}

	level := "info"
	if statusCode >= 400 {
		level = "error"
	} else if statusCode >= 300 {
		level = "warn"
	}

	o.LogEvent(ctx, "api_call", level, fmt.Sprintf("%s %s - %d", method, path, statusCode), data)
}

// LogUserAction logs a user action
func (o *ObserveLogger) LogUserAction(ctx context.Context, action, userID string, metadata map[string]interface{}) {
	if !o.config.Enabled {
		return
	}

	data := map[string]interface{}{
		"action":   action,
		"user_id":  userID,
		"metadata": metadata,
	}

	o.LogEvent(ctx, "user_action", "info", fmt.Sprintf("User action: %s", action), data)
}

// LogScanProgress logs scan progress updates
func (o *ObserveLogger) LogScanProgress(ctx context.Context, scanID string, progress int, stage string, metadata map[string]interface{}) {
	if !o.config.Enabled {
		return
	}

	data := map[string]interface{}{
		"scan_id":  scanID,
		"progress": progress,
		"stage":    stage,
		"metadata": metadata,
	}

	o.LogEvent(ctx, "scan_progress", "info", fmt.Sprintf("Scan %s: %s (%d%%)", scanID, stage, progress), data)
}

// CreateTrace creates a new trace for operation tracking
func (o *ObserveLogger) CreateTrace(ctx context.Context, operationName string) *ObserveTrace {
	traceID := uuid.New().String()
	
	trace := &ObserveTrace{
		TraceID:     traceID,
		Operation:   operationName,
		StartTime:   time.Now(),
		Success:     true,
		Metadata:    make(map[string]interface{}),
		Environment: o.config.Environment,
		ProjectID:   o.config.ProjectID,
	}

	// Store trace in context for later use
	ctx = context.WithValue(ctx, "observe_trace_id", traceID)

	if o.config.Enabled {
		o.LogEvent(ctx, "trace_start", "info", fmt.Sprintf("Started trace: %s", operationName), map[string]interface{}{
			"trace_id":  traceID,
			"operation": operationName,
		})
	}

	return trace
}

// EndTrace completes a trace
func (o *ObserveLogger) EndTrace(ctx context.Context, trace *ObserveTrace, success bool, metadata map[string]interface{}) {
	if !o.config.Enabled {
		return
	}

	endTime := time.Now()
	duration := endTime.Sub(trace.StartTime)
	
	trace.EndTime = &endTime
	trace.Duration = &duration
	trace.Success = success
	
	// Merge metadata
	for k, v := range metadata {
		trace.Metadata[k] = v
	}

	data := map[string]interface{}{
		"trace_id":    trace.TraceID,
		"operation":   trace.Operation,
		"duration_ms": duration.Milliseconds(),
		"success":     success,
		"metadata":    trace.Metadata,
	}

	level := "info"
	if !success {
		level = "error"
	}

	o.LogEvent(ctx, "trace_end", level, fmt.Sprintf("Completed trace: %s", trace.Operation), data)
}

// sendEvent sends an event to Observe MCP
func (o *ObserveLogger) sendEvent(event ObserveEvent) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		o.logger.Error("Failed to marshal Observe event", "error", err)
		return
	}

	req, err := http.NewRequest("POST", o.config.Endpoint+"/events", bytes.NewBuffer(jsonData))
	if err != nil {
		o.logger.Error("Failed to create Observe request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.config.APIKey)
	req.Header.Set("X-Observe-Project", o.config.ProjectID)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		o.logger.Error("Failed to send event to Observe", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		o.logger.Error("Observe API returned error", "status", resp.StatusCode)
	}
}

// sanitizeData removes sensitive information from data
func (o *ObserveLogger) sanitizeData(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	sensitiveKeys := []string{
		"password", "token", "secret", "key", "authorization",
		"cookie", "session", "auth", "credential", "private",
	}

	switch v := data.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{})
		for key, value := range v {
			lowerKey := strings.ToLower(key)
			isSensitive := false
			for _, sensitive := range sensitiveKeys {
				if strings.Contains(lowerKey, sensitive) {
					isSensitive = true
					break
				}
			}
			
			if isSensitive {
				sanitized[key] = "[REDACTED]"
			} else {
				sanitized[key] = o.sanitizeData(value)
			}
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(v))
		for i, item := range v {
			sanitized[i] = o.sanitizeData(item)
		}
		return sanitized
	default:
		return data
	}
}

// Helper functions to extract context information
func (o *ObserveLogger) getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value("observe_trace_id").(string); ok {
		return traceID
	}
	return ""
}

func (o *ObserveLogger) getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}

func (o *ObserveLogger) getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// ObserveMiddleware creates a Gin middleware for automatic request logging
type ObserveMiddleware struct {
	logger *ObserveLogger
}

// NewObserveMiddleware creates a new Observe middleware
func NewObserveMiddleware(logger *ObserveLogger) *ObserveMiddleware {
	return &ObserveMiddleware{
		logger: logger,
	}
}

// LogRequest logs HTTP requests
func (m *ObserveMiddleware) LogRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		
		// Create trace for the request
		trace := m.logger.CreateTrace(c.Request.Context(), fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path))
		c.Set("observe_trace", trace)

		// Process request
		c.Next()

		// Log the completed request
		duration := time.Since(start)
		
		// Get user ID if available
		userID := ""
		if user, exists := c.Get("user_id"); exists {
			if uid, ok := user.(string); ok {
				userID = uid
			}
		}

		// End trace
		success := c.Writer.Status() < 400
		m.logger.EndTrace(c.Request.Context(), trace, success, map[string]interface{}{
			"status_code": c.Writer.Status(),
			"user_id":     userID,
		})

		// Log API call
		m.logger.LogAPICall(
			c.Request.Context(),
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
			nil, // Request body (could be captured if needed)
			nil, // Response body (could be captured if needed)
		)
	}
}