package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CorrelationIDMiddleware adds correlation ID to requests
func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if correlation ID is already present in headers
		correlationID := c.GetHeader("X-Correlation-ID")
		
		// If not present, generate a new one
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		
		// Set correlation ID in context and response header
		c.Set("correlation_id", correlationID)
		c.Header("X-Correlation-ID", correlationID)
		
		c.Next()
	}
}

// RequestIDMiddleware adds request ID to requests (if not already present)
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID is already present
		requestID := c.GetHeader("X-Request-ID")
		
		// If not present, generate a new one
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}