package api

import (
	"github.com/gin-gonic/gin"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/api/handlers"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/findings"
)

// NewFindingsHandler creates a new findings handler wrapper
func NewFindingsHandler(service *findings.Service) *handlers.FindingsHandler {
	return handlers.NewFindingsHandler(service)
}

// Middleware to extract user ID and add it to context
func extractUserIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from JWT token (this should be set by AuthMiddleware)
		if userID, exists := c.Get("user_id"); exists {
			c.Set("user_id", userID)
		}
		c.Next()
	}
}