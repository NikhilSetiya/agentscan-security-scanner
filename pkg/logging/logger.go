package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Logger wraps logrus with additional functionality
type Logger struct {
	*logrus.Logger
	serviceName string
	version     string
}

// Config holds logging configuration
type Config struct {
	Level       string `json:"level"`
	Format      string `json:"format"`
	Output      string `json:"output"`
	ServiceName string `json:"service_name"`
	Version     string `json:"version"`
}

// ContextKey type for context keys
type ContextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey ContextKey = "correlation_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"
	// SpanIDKey is the context key for span ID
	SpanIDKey ContextKey = "span_id"
)

// NewLogger creates a new structured logger
func NewLogger(config *Config) (*Logger, error) {
	if config == nil {
		config = &Config{
			Level:       "info",
			Format:      "json",
			Output:      "stdout",
			ServiceName: "agentscan",
			Version:     "unknown",
		}
	}

	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(level)

	// Set formatter
	switch strings.ToLower(config.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
				logrus.FieldKeyFile:  "file",
			},
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	default:
		return nil, fmt.Errorf("unsupported log format: %s", config.Format)
	}

	// Set output
	switch strings.ToLower(config.Output) {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		// Assume it's a file path
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.SetOutput(file)
	}

	// Enable caller reporting
	logger.SetReportCaller(true)

	return &Logger{
		Logger:      logger,
		serviceName: config.ServiceName,
		version:     config.Version,
	}, nil
}

// WithContext creates a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.Logger.WithFields(logrus.Fields{
		"service": l.serviceName,
		"version": l.version,
	})

	// Add correlation ID if present
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		entry = entry.WithField("correlation_id", correlationID)
	}

	// Add user ID if present
	if userID := ctx.Value(UserIDKey); userID != nil {
		entry = entry.WithField("user_id", userID)
	}

	// Add request ID if present
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}

	// Add trace ID if present
	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		entry = entry.WithField("trace_id", traceID)
	}

	// Add span ID if present
	if spanID := ctx.Value(SpanIDKey); spanID != nil {
		entry = entry.WithField("span_id", spanID)
	}

	return entry
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	baseFields := logrus.Fields{
		"service": l.serviceName,
		"version": l.version,
	}

	// Merge fields
	for k, v := range fields {
		baseFields[k] = v
	}

	return l.Logger.WithFields(baseFields)
}

// WithError creates a logger with error field
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.WithFields(logrus.Fields{
		"error": err.Error(),
		"error_type": fmt.Sprintf("%T", err),
	})
}

// WithComponent creates a logger with component field
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.WithFields(logrus.Fields{
		"component": component,
	})
}

// WithOperation creates a logger with operation field
func (l *Logger) WithOperation(operation string) *logrus.Entry {
	return l.WithFields(logrus.Fields{
		"operation": operation,
	})
}

// WithDuration creates a logger with duration field
func (l *Logger) WithDuration(duration time.Duration) *logrus.Entry {
	return l.WithFields(logrus.Fields{
		"duration_ms": duration.Milliseconds(),
		"duration":    duration.String(),
	})
}

// LogRequest logs HTTP request details
func (l *Logger) LogRequest(ctx context.Context, method, path, userAgent, clientIP string, statusCode int, duration time.Duration) {
	l.WithContext(ctx).WithFields(logrus.Fields{
		"http_method":     method,
		"http_path":       path,
		"http_status":     statusCode,
		"user_agent":      userAgent,
		"client_ip":       clientIP,
		"response_time_ms": duration.Milliseconds(),
	}).Info("HTTP request processed")
}

// LogScanEvent logs scan-related events
func (l *Logger) LogScanEvent(ctx context.Context, event string, jobID, repoURL, agentName string, fields logrus.Fields) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"event":      event,
		"job_id":     jobID,
		"repo_url":   repoURL,
		"agent_name": agentName,
	})

	if fields != nil {
		entry = entry.WithFields(fields)
	}

	entry.Info("Scan event")
}

// LogAgentEvent logs agent-related events
func (l *Logger) LogAgentEvent(ctx context.Context, event string, agentName string, fields logrus.Fields) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"event":      event,
		"agent_name": agentName,
	})

	if fields != nil {
		entry = entry.WithFields(fields)
	}

	entry.Info("Agent event")
}

// LogAuthEvent logs authentication-related events
func (l *Logger) LogAuthEvent(ctx context.Context, event string, userID, provider string, success bool, fields logrus.Fields) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"event":    event,
		"user_id":  userID,
		"provider": provider,
		"success":  success,
	})

	if fields != nil {
		entry = entry.WithFields(fields)
	}

	if success {
		entry.Info("Authentication event")
	} else {
		entry.Warn("Authentication event failed")
	}
}

// LogPerformanceEvent logs performance-related events
func (l *Logger) LogPerformanceEvent(ctx context.Context, operation string, duration time.Duration, fields logrus.Fields) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"operation":       operation,
		"duration_ms":     duration.Milliseconds(),
		"duration":        duration.String(),
	})

	if fields != nil {
		entry = entry.WithFields(fields)
	}

	entry.Info("Performance event")
}

// LogError logs error with context and stack trace
func (l *Logger) LogError(ctx context.Context, err error, message string, fields logrus.Fields) {
	entry := l.WithContext(ctx).WithError(err)

	if fields != nil {
		entry = entry.WithFields(fields)
	}

	// Add stack trace for debugging
	if l.Logger.Level >= logrus.DebugLevel {
		entry = entry.WithField("stack_trace", getStackTrace())
	}

	entry.Error(message)
}

// LogPanic logs panic with full context
func (l *Logger) LogPanic(ctx context.Context, recovered interface{}, message string) {
	l.WithContext(ctx).WithFields(logrus.Fields{
		"panic":       recovered,
		"stack_trace": getStackTrace(),
	}).Fatal(message)
}

// NewCorrelationID generates a new correlation ID
func NewCorrelationID() string {
	return uuid.New().String()
}

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID adds span ID to context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// GetCorrelationID retrieves correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id := ctx.Value(CorrelationIDKey); id != nil {
		return id.(string)
	}
	return ""
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if id := ctx.Value(UserIDKey); id != nil {
		return id.(string)
	}
	return ""
}

// getStackTrace returns the current stack trace
func getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// SetOutput sets the logger output
func (l *Logger) SetOutput(output io.Writer) {
	l.Logger.SetOutput(output)
}

// SetLevel sets the logger level
func (l *Logger) SetLevel(level logrus.Level) {
	l.Logger.SetLevel(level)
}

// GetLevel returns the current logger level
func (l *Logger) GetLevel() logrus.Level {
	return l.Logger.GetLevel()
}

// Global logger instance
var globalLogger *Logger

// init initializes the global logger
func init() {
	var err error
	globalLogger, err = NewLogger(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize global logger: %v", err))
	}
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	return globalLogger
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// Info logs an info message with key-value pairs
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.WithFields(parseKeysAndValues(keysAndValues)).Info(msg)
}

// Warn logs a warning message with key-value pairs
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.WithFields(parseKeysAndValues(keysAndValues)).Warn(msg)
}

// Error logs an error message with key-value pairs
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.WithFields(parseKeysAndValues(keysAndValues)).Error(msg)
}

// Debug logs a debug message with key-value pairs
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.WithFields(parseKeysAndValues(keysAndValues)).Debug(msg)
}

// parseKeysAndValues converts key-value pairs to logrus.Fields
func parseKeysAndValues(keysAndValues []interface{}) logrus.Fields {
	fields := make(logrus.Fields)
	
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			value := keysAndValues[i+1]
			fields[key] = value
		}
	}
	
	return fields
}