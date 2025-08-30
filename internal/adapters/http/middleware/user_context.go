package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/pkg/errors"
)

// UserContext represents user information in the request context
type UserContext struct {
	UserID    uuid.UUID  `json:"user_id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	OrgID     *uuid.UUID `json:"org_id,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
}

// RequireUserContext middleware ensures user context is available
func RequireUserContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		userContext := GetUserContextFromGin(c)
		if userContext == nil {
			c.Error(errors.NewUnauthorizedError("user context required"))
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// GetUserContextFromGin extracts user context from Gin context
func GetUserContextFromGin(c *gin.Context) *UserContext {
	// Try to get user ID
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return nil
	}
	
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return nil
	}
	
	// Get email
	email, _ := c.Get("user_email")
	emailStr, _ := email.(string)
	
	// Get role
	role, _ := c.Get("user_role")
	roleStr, _ := role.(string)
	
	// Get organization ID (optional)
	var orgID *uuid.UUID
	if orgValue, exists := c.Get("org_id"); exists {
		if orgUUID, ok := orgValue.(uuid.UUID); ok {
			orgID = &orgUUID
		}
	}
	
	// Get session ID (optional)
	sessionID, _ := c.Get("session_id")
	sessionIDStr, _ := sessionID.(string)
	
	return &UserContext{
		UserID:    userID,
		Email:     emailStr,
		Role:      roleStr,
		OrgID:     orgID,
		SessionID: sessionIDStr,
	}
}

// GetCurrentUserID extracts user ID from context (legacy compatibility)
func GetCurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	userContext := GetUserContextFromGin(c)
	if userContext == nil {
		return uuid.Nil, false
	}
	return userContext.UserID, true
}

// RequireOrganization middleware ensures user belongs to an organization
func RequireOrganization() gin.HandlerFunc {
	return func(c *gin.Context) {
		userContext := GetUserContextFromGin(c)
		if userContext == nil || userContext.OrgID == nil {
			c.Error(errors.NewForbiddenError("organization membership required"))
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// RequireRole middleware ensures user has the required role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userContext := GetUserContextFromGin(c)
		if userContext == nil {
			c.Error(errors.NewUnauthorizedError("user context required"))
			c.Abort()
			return
		}
		
		if !hasRequiredRole(userContext.Role, requiredRole) {
			c.Error(errors.NewForbiddenError("insufficient permissions").WithDetails(map[string]interface{}{
				"required_role": requiredRole,
				"user_role":     userContext.Role,
			}))
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// hasRequiredRole checks if user role meets requirements
func hasRequiredRole(userRole, requiredRole string) bool {
	// Define role hierarchy
	roleHierarchy := map[string]int{
		"user":  1,
		"admin": 2,
		"super": 3,
	}
	
	userLevel, userExists := roleHierarchy[userRole]
	requiredLevel, requiredExists := roleHierarchy[requiredRole]
	
	if !userExists || !requiredExists {
		return false
	}
	
	return userLevel >= requiredLevel
}

// SetUserContext sets user context in Gin context (for testing)
func SetUserContext(c *gin.Context, userContext *UserContext) {
	c.Set("user_id", userContext.UserID)
	c.Set("user_email", userContext.Email)
	c.Set("user_role", userContext.Role)
	if userContext.OrgID != nil {
		c.Set("org_id", *userContext.OrgID)
	}
	if userContext.SessionID != "" {
		c.Set("session_id", userContext.SessionID)
	}
}