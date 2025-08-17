package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// Middleware provides authentication and authorization middleware
type Middleware struct {
	authService *Service
	userService *UserService
	rbacService *RBACService
	repos       *database.Repositories
	config      *config.Config
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(authService *Service, userService *UserService, rbacService *RBACService, repos *database.Repositories, config *config.Config) *Middleware {
	return &Middleware{
		authService: authService,
		userService: userService,
		rbacService: rbacService,
		repos:       repos,
		config:      config,
	}
}

// AuthRequired middleware that requires authentication
func (m *Middleware) AuthRequired() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, err := m.authenticateRequest(c)
		if err != nil {
			m.unauthorizedResponse(c, err.Error())
			c.Abort()
			return
		}

		// Set user context
		c.Set("user", user)
		c.Set("user_id", user.ID)

		// Update last active (async)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.userService.UpdateLastActive(ctx, user.ID)
		}()

		c.Next()
	})
}

// OptionalAuth middleware that optionally authenticates if token is present
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, err := m.authenticateRequest(c)
		if err == nil && user != nil {
			// Set user context if authentication successful
			c.Set("user", user)
			c.Set("user_id", user.ID)

			// Update last active (async)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = m.userService.UpdateLastActive(ctx, user.ID)
			}()
		}

		c.Next()
	})
}

// RequireRole middleware that requires a specific role
func (m *Middleware) RequireRole(role string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		_, exists := GetCurrentUser(c)
		if !exists {
			m.unauthorizedResponse(c, "Authentication required")
			c.Abort()
			return
		}

		// TODO: Implement role checking logic
		// For now, we'll assume all authenticated users have basic access
		if role == "admin" {
			// Check if user is admin (this would be stored in user profile or organization membership)
			m.forbiddenResponse(c, "Admin access required")
			c.Abort()
			return
		}

		c.Next()
	})
}

// RequireOrganizationAccess middleware that requires access to a specific organization
func (m *Middleware) RequireOrganizationAccess(orgIDParam string) gin.HandlerFunc {
	return m.RequirePermission(PermissionOrgRead, "organization", orgIDParam)
}

// RequireRepositoryAccess middleware that requires access to a specific repository
func (m *Middleware) RequireRepositoryAccess(repoIDParam string) gin.HandlerFunc {
	return m.RequirePermission(PermissionRepoRead, "repository", repoIDParam)
}

// RateLimit middleware that implements rate limiting per user
func (m *Middleware) RateLimit(requestsPerMinute int) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// TODO: Implement proper rate limiting with Redis
		// For now, we'll just pass through
		c.Next()
	})
}

// RequirePermission middleware that requires a specific permission for a resource
func (m *Middleware) RequirePermission(permission Permission, resourceType, resourceIDParam string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, exists := GetCurrentUser(c)
		if !exists {
			m.unauthorizedResponse(c, "Authentication required")
			c.Abort()
			return
		}

		var resourceID *uuid.UUID
		if resourceIDParam != "" {
			resourceIDStr := c.Param(resourceIDParam)
			if resourceIDStr == "" {
				m.badRequestResponse(c, fmt.Sprintf("%s ID required", resourceType))
				c.Abort()
				return
			}

			id, err := uuid.Parse(resourceIDStr)
			if err != nil {
				m.badRequestResponse(c, fmt.Sprintf("Invalid %s ID", resourceType))
				c.Abort()
				return
			}
			resourceID = &id
		}

		// Check permission using RBAC service
		hasPermission, err := m.rbacService.CheckPermission(c.Request.Context(), user.ID, permission, resourceType, resourceID)
		if err != nil {
			m.internalErrorResponse(c, "Failed to check permissions")
			c.Abort()
			return
		}

		if !hasPermission {
			m.forbiddenResponse(c, fmt.Sprintf("Insufficient permissions: %s required for %s", permission, resourceType))
			c.Abort()
			return
		}

		// Set resource ID in context if provided
		if resourceID != nil {
			c.Set(fmt.Sprintf("%s_id", resourceType), *resourceID)
		}

		c.Next()
	})
}

// RequireOrganizationPermission middleware that requires a specific permission for an organization
func (m *Middleware) RequireOrganizationPermission(permission Permission, orgIDParam string) gin.HandlerFunc {
	return m.RequirePermission(permission, "organization", orgIDParam)
}

// RequireRepositoryPermission middleware that requires a specific permission for a repository
func (m *Middleware) RequireRepositoryPermission(permission Permission, repoIDParam string) gin.HandlerFunc {
	return m.RequirePermission(permission, "repository", repoIDParam)
}

// RequireSystemPermission middleware that requires a system-level permission
func (m *Middleware) RequireSystemPermission(permission Permission) gin.HandlerFunc {
	return m.RequirePermission(permission, "system", "")
}

// authenticateRequest authenticates a request and returns the user
func (m *Middleware) authenticateRequest(c *gin.Context) (*types.User, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header is required")
	}

	// Extract token from "Bearer <token>"
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return nil, fmt.Errorf("authorization header must be in format 'Bearer <token>'")
	}

	tokenString := tokenParts[1]

	// Validate token
	claims, err := m.authService.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired token: %w", err)
	}

	// Get user from database to ensure they still exist
	user, err := m.userService.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// checkOrganizationAccess checks if a user has access to an organization
func (m *Middleware) checkOrganizationAccess(ctx context.Context, userID, orgID uuid.UUID) (bool, error) {
	// TODO: Implement organization access checking
	// This would check organization membership and roles
	return true, nil
}

// checkRepositoryAccess checks if a user has access to a repository
func (m *Middleware) checkRepositoryAccess(ctx context.Context, userID, repoID uuid.UUID) (bool, error) {
	// TODO: Implement repository access checking
	// This would check repository permissions through organization membership
	return true, nil
}

// Response helper methods
func (m *Middleware) unauthorizedResponse(c *gin.Context, message string) {
	c.JSON(401, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "unauthorized",
			"message": message,
		},
		"timestamp": time.Now(),
	})
}

func (m *Middleware) forbiddenResponse(c *gin.Context, message string) {
	c.JSON(403, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "forbidden",
			"message": message,
		},
		"timestamp": time.Now(),
	})
}

func (m *Middleware) badRequestResponse(c *gin.Context, message string) {
	c.JSON(400, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "bad_request",
			"message": message,
		},
		"timestamp": time.Now(),
	})
}

func (m *Middleware) internalErrorResponse(c *gin.Context, message string) {
	c.JSON(500, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "internal_error",
			"message": message,
		},
		"timestamp": time.Now(),
	})
}

// GetCurrentUser retrieves the current user from the context
func GetCurrentUser(c *gin.Context) (*types.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	
	u, ok := user.(*types.User)
	return u, ok
}

// GetCurrentUserID retrieves the current user ID from the context
func GetCurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	
	id, ok := userID.(uuid.UUID)
	return id, ok
}

// GetOrganizationID retrieves the organization ID from the context
func GetOrganizationID(c *gin.Context) (uuid.UUID, bool) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		return uuid.Nil, false
	}
	
	id, ok := orgID.(uuid.UUID)
	return id, ok
}

// GetRepositoryID retrieves the repository ID from the context
func GetRepositoryID(c *gin.Context) (uuid.UUID, bool) {
	repoID, exists := c.Get("repository_id")
	if !exists {
		return uuid.Nil, false
	}
	
	id, ok := repoID.(uuid.UUID)
	return id, ok
}