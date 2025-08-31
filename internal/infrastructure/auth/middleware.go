package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
)

// AuthMiddleware provides Supabase-based authentication middleware
type AuthMiddleware struct {
	supabaseClient *SupabaseClient
	config         *config.Config
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	supabaseClient := NewSupabaseClient(&cfg.Supabase, cfg.Auth.JWTSecret)
	
	return &AuthMiddleware{
		supabaseClient: supabaseClient,
		config:         cfg,
	}
}

// RequireAuth middleware that requires valid authentication
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for health endpoints
		if am.shouldSkipAuth(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		token, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "MISSING_TOKEN",
			})
			c.Abort()
			return
		}

		// Validate JWT token
		claims, err := am.supabaseClient.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"code":  "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		// Add user information to context
		c.Set("user_id", claims.Sub)
		c.Set("user_email", claims.Email)
		c.Set("user_role", am.supabaseClient.GetUserRole(claims))
		c.Set("session_id", claims.SessionID)
		c.Set("jwt_claims", claims)

		// Check if token is close to expiration and needs refresh
		if am.isTokenNearExpiry(claims) {
			c.Header("X-Token-Refresh-Needed", "true")
		}

		c.Next()
	}
}

// RequireRole middleware that requires a specific role
func (am *AuthMiddleware) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "NO_USER_CONTEXT",
			})
			c.Abort()
			return
		}

		if !am.hasRequiredRole(userRole.(string), requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "INSUFFICIENT_ROLE",
				"required_role": requiredRole,
				"user_role": userRole,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole middleware that requires any of the specified roles
func (am *AuthMiddleware) RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "NO_USER_CONTEXT",
			})
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range roles {
			if am.hasRequiredRole(userRole.(string), role) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "INSUFFICIENT_ROLE",
				"required_roles": roles,
				"user_role": userRole,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware that adds user context if token is present but doesn't require it
func (am *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := am.supabaseClient.ValidateJWT(token)
		if err != nil {
			c.Next()
			return
		}

		// Add user information to context
		c.Set("user_id", claims.Sub)
		c.Set("user_email", claims.Email)
		c.Set("user_role", am.supabaseClient.GetUserRole(claims))
		c.Set("session_id", claims.SessionID)
		c.Set("jwt_claims", claims)

		c.Next()
	}
}

// RefreshToken handles token refresh requests
func (am *AuthMiddleware) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Refresh the token using Supabase
	session, err := am.supabaseClient.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Failed to refresh token",
			"code":  "REFRESH_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  session.AccessToken,
		"refresh_token": session.RefreshToken,
		"expires_in":    session.ExpiresIn,
		"expires_at":    session.ExpiresAt,
		"token_type":    session.TokenType,
		"user":          session.User,
	})
}

// GetCurrentUser returns the current user information
func (am *AuthMiddleware) GetCurrentUser(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"code":  "NO_USER_CONTEXT",
		})
		return
	}

	// Get fresh user data from Supabase
	authHeader := c.GetHeader("Authorization")
	token, err := ExtractTokenFromHeader(authHeader)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token",
			"code":  "INVALID_TOKEN",
		})
		return
	}

	user, err := am.supabaseClient.GetUser(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user information",
			"code":  "USER_FETCH_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// SignOut handles user sign out
func (am *AuthMiddleware) SignOut(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	token, err := ExtractTokenFromHeader(authHeader)
	if err != nil {
		// Even if token is invalid, we can still return success for sign out
		c.JSON(http.StatusOK, gin.H{
			"message": "Signed out successfully",
		})
		return
	}

	// Sign out from Supabase
	err = am.supabaseClient.SignOut(c.Request.Context(), token)
	if err != nil {
		// Log the error but still return success to the client
		// This prevents issues if the token is already invalid
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Signed out successfully",
	})
}

// UpdateUserProfile handles user profile updates
func (am *AuthMiddleware) UpdateUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"code":  "NO_USER_CONTEXT",
		})
		return
	}

	var req struct {
		UserMetadata map[string]interface{} `json:"user_metadata"`
		AppMetadata  map[string]interface{} `json:"app_metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Prepare updates
	updates := make(map[string]interface{})
	if req.UserMetadata != nil {
		updates["user_metadata"] = req.UserMetadata
	}

	// Only allow app_metadata updates for admin users
	userRole, _ := c.Get("user_role")
	if userRole == "admin" && req.AppMetadata != nil {
		updates["app_metadata"] = req.AppMetadata
	}

	// Update user in Supabase
	user, err := am.supabaseClient.UpdateUser(c.Request.Context(), userID.(string), updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user profile",
			"code":  "UPDATE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
		"message": "Profile updated successfully",
	})
}

// Helper methods

// shouldSkipAuth checks if authentication should be skipped for a path
func (am *AuthMiddleware) shouldSkipAuth(path string) bool {
	skipPaths := []string{
		"/health",
		"/ready",
		"/live",
		"/metrics",
		"/api/v1/auth/",
		"/api/v1/webhooks/",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

// isTokenNearExpiry checks if token is close to expiration
func (am *AuthMiddleware) isTokenNearExpiry(claims *JWTClaims) bool {
	// Consider token near expiry if it expires within 5 minutes
	expiryThreshold := time.Now().Add(5 * time.Minute).Unix()
	return claims.Exp <= expiryThreshold
}

// hasRequiredRole checks if user has the required role
func (am *AuthMiddleware) hasRequiredRole(userRole, requiredRole string) bool {
	// Define role hierarchy
	roleHierarchy := map[string]int{
		"user":      1,
		"moderator": 2,
		"admin":     3,
		"superuser": 4,
	}

	userLevel, userExists := roleHierarchy[userRole]
	requiredLevel, requiredExists := roleHierarchy[requiredRole]

	// If roles are not in hierarchy, do exact match
	if !userExists || !requiredExists {
		return userRole == requiredRole
	}

	// Check if user level meets or exceeds required level
	return userLevel >= requiredLevel
}

// GetUserFromContext extracts user information from Gin context
func GetUserFromContext(c *gin.Context) (*UserContext, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, errors.NewAuthenticationError("user not found in context")
	}

	userEmail, _ := c.Get("user_email")
	userRole, _ := c.Get("user_role")
	sessionID, _ := c.Get("session_id")

	return &UserContext{
		ID:        userID.(string),
		Email:     userEmail.(string),
		Role:      userRole.(string),
		SessionID: sessionID.(string),
	}, nil
}

// UserContext represents user information in request context
type UserContext struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
}

// HasRole checks if user has a specific role
func (uc *UserContext) HasRole(role string) bool {
	roleHierarchy := map[string]int{
		"user":      1,
		"moderator": 2,
		"admin":     3,
		"superuser": 4,
	}

	userLevel, userExists := roleHierarchy[uc.Role]
	requiredLevel, requiredExists := roleHierarchy[role]

	if !userExists || !requiredExists {
		return uc.Role == role
	}

	return userLevel >= requiredLevel
}

// IsAdmin checks if user is an admin
func (uc *UserContext) IsAdmin() bool {
	return uc.HasRole("admin")
}

// IsModerator checks if user is a moderator or higher
func (uc *UserContext) IsModerator() bool {
	return uc.HasRole("moderator")
}