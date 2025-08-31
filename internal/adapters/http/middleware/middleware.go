package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CORS middleware for handling Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	})
}

// AuthRequired middleware ensures the user is authenticated
func AuthRequired() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// This is a placeholder implementation
		// In a real application, you would:
		// 1. Extract the JWT token from the Authorization header
		// 2. Validate the token
		// 3. Extract user information from the token
		// 4. Set user context in gin.Context
		
		// For now, we'll just check if Authorization header exists
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Mock user context - replace with actual JWT parsing
		userID := uuid.New() // This should come from JWT
		userEmail := "user@example.com" // This should come from JWT
		userRole := "user" // This should come from JWT
		
		c.Set("user_id", userID)
		c.Set("user_email", userEmail)
		c.Set("user_role", userRole)
		
		c.Next()
	})
}

// RequireRole middleware ensures the user has the required role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User role not found",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user role format",
			})
			c.Abort()
			return
		}

		if !hasRequiredRole(role, requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	})
}

// hasRequiredRole checks if the user role has the required permissions
func hasRequiredRole(userRole, requiredRole string) bool {
	roleHierarchy := map[string]int{
		"viewer":    1,
		"user":      2,
		"developer": 3,
		"admin":     4,
		"owner":     5,
	}

	userLevel, exists := roleHierarchy[userRole]
	if !exists {
		return false
	}

	requiredLevel, exists := roleHierarchy[requiredRole]
	if !exists {
		return false
	}

	return userLevel >= requiredLevel
}

// RateLimiting middleware for rate limiting requests
func RateLimiting() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// This is a placeholder implementation
		// In a real application, you would implement rate limiting logic
		// using Redis or an in-memory store
		
		c.Next()
	})
}

// Logging middleware for structured logging
func Logging() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// This is a placeholder implementation
		// In a real application, you would implement structured logging
		// with proper log levels and context
		
		c.Next()
	})
}