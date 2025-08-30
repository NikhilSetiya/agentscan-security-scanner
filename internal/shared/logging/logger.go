package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/pkg/errors"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Logger provides structured logging with correlation IDs
type Logger struct {
	slogger       *slog.Logger
	level         LogLevel
	serviceName   string
	serviceVersion string
	environment   string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	Timestamp     time.Time              `json:"timestamp"`
	ServiceName   string                 `json:"service_name"`
	ServiceVersion string                `json:"service_version"`
	Environment   string                 `json:"environment"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	SpanID        string                 `json:"span_id,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
	Error         *ErrorDetails          `json:"error,omitempty"`
	Stack         []string               `json:"stack,omitempty"`
	Duration      *time.Duration         `json:"duration,omitempty"`
	HTTPRequest   *HTTPRequestDetails    `json:"http_request,omitempty"`
	HTTPResponse  *HTTPResponseDetails   `json:"http_response,omitempty"`
}

// ErrorDetails represents error information in logs
type ErrorDetails struct {
	Type          string                 `json:"type"`
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details,omitempty"`
	Stack         []string               `json:"stack,omitempty"`
	Cause         string                 `json:"cause,omitempty"`
}

// HTTPRequestDetails represents HTTP request information
type HTTPRequestDetails struct {
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Path       string            `json:"path"`
	Query      string            `json:"query,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	RemoteAddr string            `json:"remote_addr,omitempty"`
	Size       int64             `json:"size,omitempty"`
}

// HTTPResponseDetails represents HTTP response information
type HTTPResponseDetails struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Size       int64             `json:"size,omitempty"`
}

// NewLogger creates a new structured logger
func NewLogger(serviceName, serviceVersion, environment string, level LogLevel) *Logger {
	// Configure slog based on environment
	var handler slog.Handler
	
	if environment == "production" {
		// JSON handler for production
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: convertLogLevel(level),
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Customize attribute names for better compatibility
				if a.Key == slog.TimeKey {
					a.Key = "timestamp"
				}
				if a.Key == slog.LevelKey {
					a.Key = "level"
				}
				if a.Key == slog.MessageKey {
					a.Key = "message"
				}
				return a
			},
		})
	} else {
		// Text handler for development
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: convertLogLevel(level),
		})
	}
	
	return &Logger{
		slogger:        slog.New(handler),
		level:          level,
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		environment:    environment,
	}
}

// convertLogLevel converts our LogLevel to slog.Level
func convertLogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext creates a logger with context information
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract context information if available
	// This would be implemented based on your context structure
	return l
}

// WithGinContext creates a logger with Gin context information
func (l *Logger) WithGinContext(c *gin.Context) *Logger {
	// Create a new logger instance with context
	return l
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	l.log(LevelDebug, message, nil, nil, fields...)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	l.log(LevelInfo, message, nil, nil, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	l.log(LevelWarn, message, nil, nil, fields...)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, fields ...map[string]interface{}) {
	l.log(LevelError, message, err, nil, fields...)
}

// ErrorWithStack logs an error with stack trace
func (l *Logger) ErrorWithStack(message string, err error, fields ...map[string]interface{}) {
	stack := captureStackTrace(3) // Skip 3 frames to get to the caller
	l.log(LevelError, message, err, stack, fields...)
}

// LogRequest logs an HTTP request
func (l *Logger) LogRequest(c *gin.Context, duration time.Duration) {
	requestDetails := &HTTPRequestDetails{
		Method:     c.Request.Method,
		URL:        c.Request.URL.String(),
		Path:       c.Request.URL.Path,
		Query:      c.Request.URL.RawQuery,
		UserAgent:  c.GetHeader("User-Agent"),
		RemoteAddr: c.ClientIP(),
	}
	
	// Sanitize headers (remove sensitive information)
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if !isSensitiveHeader(key) && len(values) > 0 {
			headers[key] = values[0]
		}
	}
	requestDetails.Headers = headers
	
	responseDetails := &HTTPResponseDetails{
		StatusCode: c.Writer.Status(),
		Size:       int64(c.Writer.Size()),
	}
	
	fields := map[string]interface{}{
		"http_method":     c.Request.Method,
		"http_path":       c.Request.URL.Path,
		"http_status":     c.Writer.Status(),
		"response_size":   c.Writer.Size(),
		"duration_ms":     duration.Milliseconds(),
	}
	
	// Add user information if available
	if userID, exists := c.Get("user_id"); exists {
		fields["user_id"] = userID
	}
	
	entry := l.createLogEntry(LevelInfo, "HTTP Request", nil, nil, fields)
	entry.Duration = &duration
	entry.HTTPRequest = requestDetails
	entry.HTTPResponse = responseDetails
	
	l.writeLogEntry(entry, c)
}

// LogError logs an application error with full context
func (l *Logger) LogError(c *gin.Context, err error, message string, fields ...map[string]interface{}) {
	var stack []string
	if appErr, ok := err.(*errors.AppError); ok {
		stack = appErr.Stack
	}
	
	if len(stack) == 0 {
		stack = captureStackTrace(3)
	}
	
	l.logWithContext(c, LevelError, message, err, stack, fields...)
}

// LogSecurityEvent logs security-related events
func (l *Logger) LogSecurityEvent(c *gin.Context, eventType, message string, fields ...map[string]interface{}) {
	securityFields := map[string]interface{}{
		"security_event": true,
		"event_type":     eventType,
		"remote_addr":    c.ClientIP(),
		"user_agent":     c.GetHeader("User-Agent"),
	}
	
	// Merge with provided fields
	if len(fields) > 0 {
		for k, v := range fields[0] {
			securityFields[k] = v
		}
	}
	
	l.logWithContext(c, LevelWarn, message, nil, nil, securityFields)
}

// LogAuditEvent logs audit events
func (l *Logger) LogAuditEvent(c *gin.Context, action, resource string, fields ...map[string]interface{}) {
	auditFields := map[string]interface{}{
		"audit_event": true,
		"action":      action,
		"resource":    resource,
		"timestamp":   time.Now(),
	}
	
	// Add user information if available
	if userID, exists := c.Get("user_id"); exists {
		auditFields["user_id"] = userID
	}
	
	// Merge with provided fields
	if len(fields) > 0 {
		for k, v := range fields[0] {
			auditFields[k] = v
		}
	}
	
	l.logWithContext(c, LevelInfo, fmt.Sprintf("Audit: %s %s", action, resource), nil, nil, auditFields)
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, message string, err error, stack []string, fields ...map[string]interface{}) {
	var mergedFields map[string]interface{}
	if len(fields) > 0 {
		mergedFields = fields[0]
	}
	
	entry := l.createLogEntry(level, message, err, stack, mergedFields)
	l.writeLogEntry(entry, nil)
}

// logWithContext logs with Gin context
func (l *Logger) logWithContext(c *gin.Context, level LogLevel, message string, err error, stack []string, fields ...map[string]interface{}) {
	var mergedFields map[string]interface{}
	if len(fields) > 0 {
		mergedFields = fields[0]
	}
	
	entry := l.createLogEntry(level, message, err, stack, mergedFields)
	l.writeLogEntry(entry, c)
}

// createLogEntry creates a structured log entry
func (l *Logger) createLogEntry(level LogLevel, message string, err error, stack []string, fields map[string]interface{}) *LogEntry {
	entry := &LogEntry{
		Level:          string(level),
		Message:        message,
		Timestamp:      time.Now(),
		ServiceName:    l.serviceName,
		ServiceVersion: l.serviceVersion,
		Environment:    l.environment,
		Fields:         fields,
		Stack:          stack,
	}
	
	// Add error details if present
	if err != nil {
		entry.Error = l.createErrorDetails(err)
	}
	
	return entry
}

// createErrorDetails creates error details from an error
func (l *Logger) createErrorDetails(err error) *ErrorDetails {
	details := &ErrorDetails{
		Message: err.Error(),
	}
	
	if appErr, ok := err.(*errors.AppError); ok {
		details.Type = string(appErr.Type)
		details.Code = appErr.Code
		details.Details = appErr.Details
		details.Stack = appErr.Stack
		
		if appErr.Cause != nil {
			details.Cause = appErr.Cause.Error()
		}
	} else {
		details.Type = "unknown"
		details.Code = "UNKNOWN_ERROR"
	}
	
	return details
}

// writeLogEntry writes the log entry
func (l *Logger) writeLogEntry(entry *LogEntry, c *gin.Context) {
	// Add context information if Gin context is available
	if c != nil {
		if requestID, exists := c.Get("request_id"); exists {
			if id, ok := requestID.(string); ok {
				entry.RequestID = id
			}
		}
		
		if correlationID, exists := c.Get("correlation_id"); exists {
			if id, ok := correlationID.(string); ok {
				entry.CorrelationID = id
			}
		}
		
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(string); ok {
				entry.UserID = id
			}
		}
	}
	
	// Convert to slog attributes
	attrs := []slog.Attr{
		slog.String("service_name", entry.ServiceName),
		slog.String("service_version", entry.ServiceVersion),
		slog.String("environment", entry.Environment),
	}
	
	if entry.RequestID != "" {
		attrs = append(attrs, slog.String("request_id", entry.RequestID))
	}
	
	if entry.CorrelationID != "" {
		attrs = append(attrs, slog.String("correlation_id", entry.CorrelationID))
	}
	
	if entry.UserID != "" {
		attrs = append(attrs, slog.String("user_id", entry.UserID))
	}
	
	if entry.Fields != nil {
		for k, v := range entry.Fields {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	
	if entry.Error != nil {
		attrs = append(attrs, slog.Any("error", entry.Error))
	}
	
	if entry.Duration != nil {
		attrs = append(attrs, slog.Duration("duration", *entry.Duration))
	}
	
	if entry.HTTPRequest != nil {
		attrs = append(attrs, slog.Any("http_request", entry.HTTPRequest))
	}
	
	if entry.HTTPResponse != nil {
		attrs = append(attrs, slog.Any("http_response", entry.HTTPResponse))
	}
	
	// Log with appropriate level
	switch LogLevel(entry.Level) {
	case LevelDebug:
		l.slogger.LogAttrs(context.Background(), slog.LevelDebug, entry.Message, attrs...)
	case LevelInfo:
		l.slogger.LogAttrs(context.Background(), slog.LevelInfo, entry.Message, attrs...)
	case LevelWarn:
		l.slogger.LogAttrs(context.Background(), slog.LevelWarn, entry.Message, attrs...)
	case LevelError:
		l.slogger.LogAttrs(context.Background(), slog.LevelError, entry.Message, attrs...)
	}
}

// Helper functions

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) []string {
	var stack []string
	
	for i := skip; i < skip+10; i++ { // Capture up to 10 frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			break
		}
		
		// Format: function (file:line)
		frame := fmt.Sprintf("%s (%s:%d)", fn.Name(), file, line)
		stack = append(stack, frame)
	}
	
	return stack
}

// isSensitiveHeader checks if a header contains sensitive information
func isSensitiveHeader(header string) bool {
	sensitiveHeaders := []string{
		"authorization", "cookie", "x-api-key", "x-auth-token",
		"x-access-token", "x-csrf-token", "x-xsrf-token",
	}
	
	headerLower := strings.ToLower(header)
	for _, sensitive := range sensitiveHeaders {
		if headerLower == sensitive {
			return true
		}
	}
	
	return false
}

// Global logger instance
var defaultLogger *Logger

// InitializeLogger initializes the global logger
func InitializeLogger(serviceName, serviceVersion, environment string, level LogLevel) {
	defaultLogger = NewLogger(serviceName, serviceVersion, environment, level)
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		// Initialize with defaults if not already initialized
		defaultLogger = NewLogger("agentscan", "1.0.0", "development", LevelInfo)
	}
	return defaultLogger
}

// Convenience functions for global logger

// Debug logs a debug message using the global logger
func Debug(message string, fields ...map[string]interface{}) {
	GetLogger().Debug(message, fields...)
}

// Info logs an info message using the global logger
func Info(message string, fields ...map[string]interface{}) {
	GetLogger().Info(message, fields...)
}

// Warn logs a warning message using the global logger
func Warn(message string, fields ...map[string]interface{}) {
	GetLogger().Warn(message, fields...)
}

// Error logs an error message using the global logger
func Error(message string, err error, fields ...map[string]interface{}) {
	GetLogger().Error(message, err, fields...)
}

// ErrorWithStack logs an error with stack trace using the global logger
func ErrorWithStack(message string, err error, fields ...map[string]interface{}) {
	GetLogger().ErrorWithStack(message, err, fields...)
}