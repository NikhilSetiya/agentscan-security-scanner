package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/agentscan/agentscan/internal/shared/logging"
	"github.com/agentscan/agentscan/pkg/errors"
)

// ErrorHandlingMiddleware provides centralized error handling with logging
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Handle panic
				handlePanic(c, r)
			}
		}()

		c.Next()

		// Handle errors that were added to the context
		if len(c.Errors) > 0 {
			handleErrors(c)
		}
	}
}

// handlePanic handles panic recovery
func handlePanic(c *gin.Context, r interface{}) {
	logger := logging.GetLogger()
	
	// Create error from panic
	panicErr := errors.NewInternalError(fmt.Sprintf("panic recovered: %v", r))
	
	// Capture stack trace
	stack := strings.Split(string(debug.Stack()), "\n")
	panicErr = panicErr.WithStack(stack)
	
	// Log the panic with full context
	logger.LogError(c, panicErr, "Panic recovered", map[string]interface{}{
		"panic_value": r,
		"stack_trace": stack,
	})
	
	// Send error response
	sendErrorResponse(c, panicErr)
}

// handleErrors handles errors from the Gin context
func handleErrors(c *gin.Context) {
	logger := logging.GetLogger()
	
	// Get the last error (most recent)
	lastError := c.Errors.Last()
	if lastError == nil {
		return
	}
	
	err := lastError.Err
	
	// Log the error with context
	logError(c, err, logger)
	
	// Send appropriate error response
	sendErrorResponse(c, err)
}

// logError logs an error with appropriate level and context
func logError(c *gin.Context, err error, logger *logging.Logger) {
	fields := map[string]interface{}{
		"method":      c.Request.Method,
		"path":        c.Request.URL.Path,
		"query":       c.Request.URL.RawQuery,
		"user_agent":  c.GetHeader("User-Agent"),
		"remote_addr": c.ClientIP(),
	}
	
	// Add user information if available
	if userID, exists := c.Get("user_id"); exists {
		fields["user_id"] = userID
	}
	
	// Determine log level based on error type
	if appErr, ok := err.(*errors.AppError); ok {
		switch appErr.Type {
		case errors.ErrorTypeValidation, errors.ErrorTypeAuthentication, errors.ErrorTypeAuthorization, errors.ErrorTypeNotFound:
			// These are expected errors, log at info level
			logger.Info(fmt.Sprintf("Client error: %s", appErr.Message), fields)
		case errors.ErrorTypeRateLimit:
			// Rate limit errors are security-related
			logger.LogSecurityEvent(c, "rate_limit_exceeded", appErr.Message, fields)
		case errors.ErrorTypeSecurity:
			// Security errors need special attention
			logger.LogSecurityEvent(c, "security_violation", appErr.Message, fields)
		case errors.ErrorTypeTimeout, errors.ErrorTypeExternal:
			// External service issues
			logger.Warn(fmt.Sprintf("External service error: %s", appErr.Message), fields)
		default:
			// Internal errors need investigation
			logger.LogError(c, appErr, "Internal server error", fields)
		}
	} else {
		// Unknown error type, log as error
		logger.LogError(c, err, "Unknown error occurred", fields)
	}
}

// sendErrorResponse sends an appropriate error response
func sendErrorResponse(c *gin.Context, err error) {
	// Avoid double responses
	if c.Writer.Written() {
		return
	}
	
	// Import the response functions (assuming they're in the same package or imported)
	// For now, we'll implement a simple version here
	
	var statusCode int
	var errorCode, message string
	var details map[string]interface{}
	
	if appErr, ok := err.(*errors.AppError); ok {
		statusCode = mapErrorTypeToHTTPStatus(appErr.Type)
		errorCode = appErr.Code
		message = appErr.Message
		details = appErr.Details
	} else {
		statusCode = http.StatusInternalServerError
		errorCode = "INTERNAL_ERROR"
		message = "An internal error occurred"
	}
	
	// Get context information
	requestID, _ := c.Get("request_id")
	correlationID, _ := c.Get("correlation_id")
	
	response := map[string]interface{}{
		"success":   false,
		"timestamp": time.Now(),
		"version":   "v1",
		"error": map[string]interface{}{
			"code":      errorCode,
			"message":   message,
			"type":      getErrorTypeName(err),
			"timestamp": time.Now(),
		},
	}
	
	if requestID != nil {
		response["request_id"] = requestID
		response["error"].(map[string]interface{})["request_id"] = requestID
	}
	
	if correlationID != nil {
		response["correlation_id"] = correlationID
		response["error"].(map[string]interface{})["correlation_id"] = correlationID
	}
	
	if details != nil && len(details) > 0 {
		response["error"].(map[string]interface{})["details"] = details
	}
	
	c.JSON(statusCode, response)
}

// mapErrorTypeToHTTPStatus maps error types to HTTP status codes
func mapErrorTypeToHTTPStatus(errorType errors.ErrorType) int {
	switch errorType {
	case errors.ErrorTypeValidation:
		return http.StatusBadRequest
	case errors.ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case errors.ErrorTypeAuthorization:
		return http.StatusForbidden
	case errors.ErrorTypeNotFound:
		return http.StatusNotFound
	case errors.ErrorTypeConflict:
		return http.StatusConflict
	case errors.ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case errors.ErrorTypeTimeout:
		return http.StatusRequestTimeout
	case errors.ErrorTypeExternal:
		return http.StatusBadGateway
	case errors.ErrorTypeSecurity:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// getErrorTypeName returns the error type name
func getErrorTypeName(err error) string {
	if appErr, ok := err.(*errors.AppError); ok {
		return string(appErr.Type)
	}
	return string(errors.ErrorTypeInternal)
}

// RequestLoggingMiddleware logs all HTTP requests
func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Process request
		c.Next()
		
		// Log request completion
		duration := time.Since(start)
		logger := logging.GetLogger()
		logger.LogRequest(c, duration)
	}
}

// SecurityEventMiddleware logs security-related events
func SecurityEventMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := logging.GetLogger()
		
		// Check for suspicious patterns
		userAgent := c.GetHeader("User-Agent")
		if isSuspiciousUserAgent(userAgent) {
			logger.LogSecurityEvent(c, "suspicious_user_agent", "Suspicious user agent detected", map[string]interface{}{
				"user_agent": userAgent,
			})
		}
		
		// Check for suspicious paths
		path := c.Request.URL.Path
		if isSuspiciousPath(path) {
			logger.LogSecurityEvent(c, "suspicious_path", "Suspicious path accessed", map[string]interface{}{
				"path": path,
			})
		}
		
		c.Next()
	}
}

// Helper functions for security detection

func isSuspiciousUserAgent(userAgent string) bool {
	suspiciousPatterns := []string{
		"sqlmap", "nikto", "nmap", "masscan", "zap", "burp",
		"<script", "javascript:", "vbscript:",
	}
	
	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}
	
	return false
}

func isSuspiciousPath(path string) bool {
	suspiciousPaths := []string{
		"/.env", "/config", "/admin", "/phpmyadmin", "/wp-admin",
		"/.git", "/backup", "/test", "/debug",
	}
	
	pathLower := strings.ToLower(path)
	for _, suspicious := range suspiciousPaths {
		if strings.Contains(pathLower, suspicious) {
			return true
		}
	}
	
	// Check for path traversal attempts
	if strings.Contains(path, "..") || strings.Contains(path, "%2e%2e") {
		return true
	}
	
	return false
}