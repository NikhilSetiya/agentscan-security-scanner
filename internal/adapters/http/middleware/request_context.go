package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/internal/shared/logging"
)

// RequestContext represents common request information
type RequestContext struct {
	RequestID     string    `json:"request_id"`
	CorrelationID string    `json:"correlation_id"`
	StartTime     time.Time `json:"start_time"`
	UserAgent     string    `json:"user_agent"`
	RemoteAddr    string    `json:"remote_addr"`
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	Query         string    `json:"query,omitempty"`
}

// RequestContextMiddleware enriches the request with context information
func RequestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		// Get or generate request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Get or generate correlation ID
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		
		// Create request context
		requestContext := RequestContext{
			RequestID:     requestID,
			CorrelationID: correlationID,
			StartTime:     startTime,
			UserAgent:     c.GetHeader("User-Agent"),
			RemoteAddr:    c.ClientIP(),
			Method:        c.Request.Method,
			Path:          c.Request.URL.Path,
			Query:         c.Request.URL.RawQuery,
		}
		
		// Set in Gin context
		c.Set("request_context", requestContext)
		c.Set("request_id", requestID)
		c.Set("correlation_id", correlationID)
		c.Set("start_time", startTime)
		
		// Set response headers
		c.Header("X-Request-ID", requestID)
		c.Header("X-Correlation-ID", correlationID)
		
		c.Next()
	}
}

// GetRequestContextFromGin extracts request context from Gin context
func GetRequestContextFromGin(c *gin.Context) *RequestContext {
	if ctx, exists := c.Get("request_context"); exists {
		if requestCtx, ok := ctx.(RequestContext); ok {
			return &requestCtx
		}
	}
	return nil
}

// AuditMiddleware logs important actions for audit purposes
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip audit logging for certain paths
		if shouldSkipAudit(c.Request.URL.Path) {
			c.Next()
			return
		}
		
		// Process request
		c.Next()
		
		// Log audit event after request completion
		logger := logging.GetLogger()
		
		// Determine if this is an audit-worthy action
		if isAuditableAction(c.Request.Method, c.Request.URL.Path, c.Writer.Status()) {
			action := getActionFromRequest(c.Request.Method, c.Request.URL.Path)
			resource := getResourceFromPath(c.Request.URL.Path)
			
			logger.LogAuditEvent(c, action, resource, map[string]interface{}{
				"status_code": c.Writer.Status(),
				"duration_ms": time.Since(getStartTime(c)).Milliseconds(),
			})
		}
	}
}

// Helper functions for audit logging

func shouldSkipAudit(path string) bool {
	skipPaths := []string{
		"/health",
		"/api/v1",
		"/metrics",
		"/favicon.ico",
	}
	
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	
	return false
}

func isAuditableAction(method, path string, statusCode int) bool {
	// Only audit successful state-changing operations
	if statusCode >= 400 {
		return false
	}
	
	// Audit CREATE, UPDATE, DELETE operations
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	case "GET":
		// Audit sensitive GET operations
		return isSensitiveGetOperation(path)
	default:
		return false
	}
}

func isSensitiveGetOperation(path string) bool {
	sensitivePaths := []string{
		"/api/v1/user/me",
		"/api/v1/dashboard/stats",
		"/api/v1/findings",
	}
	
	for _, sensitivePath := range sensitivePaths {
		if path == sensitivePath {
			return true
		}
	}
	
	return false
}

func getActionFromRequest(method, path string) string {
	switch method {
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	case "GET":
		return "read"
	default:
		return method
	}
}

func getResourceFromPath(path string) string {
	// Extract resource from path
	// e.g., /api/v1/repositories/123 -> repositories
	// e.g., /api/v1/scans -> scans
	
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "v1" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	
	return "unknown"
}

func getStartTime(c *gin.Context) time.Time {
	if startTime, exists := c.Get("start_time"); exists {
		if t, ok := startTime.(time.Time); ok {
			return t
		}
	}
	return time.Now()
}

// PerformanceMiddleware tracks request performance
func PerformanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		c.Next()
		
		duration := time.Since(startTime)
		
		// Log slow requests
		if duration > 5*time.Second {
			logger := logging.GetLogger()
			logger.Warn("Slow request detected", map[string]interface{}{
				"method":      c.Request.Method,
				"path":        c.Request.URL.Path,
				"duration_ms": duration.Milliseconds(),
				"status_code": c.Writer.Status(),
			})
		}
		
		// Add performance headers
		c.Header("X-Response-Time", duration.String())
	}
}