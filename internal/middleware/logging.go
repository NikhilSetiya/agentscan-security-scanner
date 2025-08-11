package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agentscan/agentscan/pkg/logging"
)

// LoggingMiddleware creates a middleware for request logging with correlation IDs
func LoggingMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate correlation ID if not present
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = logging.NewCorrelationID()
		}

		// Generate request ID
		requestID := logging.NewCorrelationID()

		// Add IDs to context
		ctx := logging.WithCorrelationID(c.Request.Context(), correlationID)
		ctx = logging.WithRequestID(ctx, requestID)

		// Add user ID if authenticated
		if userID, exists := c.Get("user_id"); exists {
			ctx = logging.WithUserID(ctx, userID.(string))
		}

		// Update request context
		c.Request = c.Request.WithContext(ctx)

		// Set response headers
		c.Header("X-Correlation-ID", correlationID)
		c.Header("X-Request-ID", requestID)

		// Process request
		c.Next()

		// Log request completion
		duration := time.Since(start)
		logger.LogRequest(
			ctx,
			c.Request.Method,
			c.Request.URL.Path,
			c.Request.UserAgent(),
			c.ClientIP(),
			c.Writer.Status(),
			duration,
		)
	}
}

// ErrorLoggingMiddleware logs errors with context
func ErrorLoggingMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log errors if any occurred
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.LogError(
					c.Request.Context(),
					err.Err,
					"Request processing error",
					gin.H{
						"error_type": err.Type,
						"meta":       err.Meta,
					},
				)
			}
		}
	}
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.LogPanic(
			c.Request.Context(),
			recovered,
			"Request panic recovered",
		)

		c.JSON(500, gin.H{
			"error":          "Internal server error",
			"correlation_id": logging.GetCorrelationID(c.Request.Context()),
		})
	})
}