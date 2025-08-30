package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/pkg/errors"
)

// AuthMiddleware provides JWT-based authentication
type AuthMiddleware struct {
	jwtSecret     []byte
	supabaseURL   string
	supabaseKey   string
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
}

// Claims represents JWT claims structure
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	OrgID     *uuid.UUID `json:"org_id,omitempty"`
	SessionID string    `json:"session_id"`
	jwt.RegisteredClaims
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtSecret, supabaseURL, supabaseKey string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:     []byte(jwtSecret),
		supabaseURL:   supabaseURL,
		supabaseKey:   supabaseKey,
		tokenExpiry:   time.Hour * 24,     // 24 hours
		refreshExpiry: time.Hour * 24 * 7, // 7 days
	}
}

// RequireAuth validates JWT tokens and sets user context
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := am.extractToken(c)
		if err != nil {
			c.Error(errors.NewUnauthorizedError("missing or invalid token").WithCause(err))
			c.Abort()
			return
		}

		claims, err := am.validateToken(token)
		if err != nil {
			c.Error(errors.NewUnauthorizedError("invalid token").WithCause(err))
			c.Abort()
			return
		}

		// Check if token is expired
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			c.Error(errors.NewUnauthorizedError("token expired"))
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("session_id", claims.SessionID)
		if claims.OrgID != nil {
			c.Set("org_id", *claims.OrgID)
		}

		c.Next()
	}
}

// RequireRole validates user has required role
func (am *AuthMiddleware) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.Error(errors.NewForbiddenError("user role not found in context"))
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			c.Error(errors.NewForbiddenError("invalid user role format"))
			c.Abort()
			return
		}

		if !am.hasRequiredRole(role, requiredRole) {
			c.Error(errors.NewForbiddenError("insufficient permissions").WithDetails(map[string]interface{}{
				"required_role": requiredRole,
				"user_role":     role,
			}))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOrganization validates user belongs to organization
func (am *AuthMiddleware) RequireOrganization() gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, exists := c.Get("org_id")
		if !exists {
			c.Error(errors.NewForbiddenError("organization access required"))
			c.Abort()
			return
		}

		// Validate organization ID format
		if _, ok := orgID.(uuid.UUID); !ok {
			c.Error(errors.NewForbiddenError("invalid organization ID"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth validates token if present but doesn't require it
func (am *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := am.extractToken(c)
		if err != nil {
			// No token present, continue without authentication
			c.Next()
			return
		}

		claims, err := am.validateToken(token)
		if err != nil {
			// Invalid token, continue without authentication
			c.Next()
			return
		}

		// Check if token is expired
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			// Expired token, continue without authentication
			c.Next()
			return
		}

		// Set user context if token is valid
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("session_id", claims.SessionID)
		if claims.OrgID != nil {
			c.Set("org_id", *claims.OrgID)
		}

		c.Next()
	}
}

// GenerateToken creates a new JWT token for user
func (am *AuthMiddleware) GenerateToken(userID uuid.UUID, email, role string, orgID *uuid.UUID) (string, string, error) {
	sessionID := uuid.New().String()
	
	// Access token claims
	accessClaims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		OrgID:     orgID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "agentscan",
			Subject:   userID.String(),
		},
	}

	// Refresh token claims
	refreshClaims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		OrgID:     orgID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "agentscan-refresh",
			Subject:   userID.String(),
		},
	}

	// Generate access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(am.jwtSecret)
	if err != nil {
		return "", "", errors.NewInternalError("failed to generate access token").WithCause(err)
	}

	// Generate refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(am.jwtSecret)
	if err != nil {
		return "", "", errors.NewInternalError("failed to generate refresh token").WithCause(err)
	}

	return accessTokenString, refreshTokenString, nil
}

// RefreshToken validates refresh token and generates new access token
func (am *AuthMiddleware) RefreshToken(refreshTokenString string) (string, error) {
	claims, err := am.validateToken(refreshTokenString)
	if err != nil {
		return "", errors.NewUnauthorizedError("invalid refresh token").WithCause(err)
	}

	// Verify this is a refresh token
	if claims.Issuer != "agentscan-refresh" {
		return "", errors.NewUnauthorizedError("invalid token type")
	}

	// Check if refresh token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return "", errors.NewUnauthorizedError("refresh token expired")
	}

	// Generate new access token
	newAccessClaims := Claims{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Role:      claims.Role,
		OrgID:     claims.OrgID,
		SessionID: claims.SessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "agentscan",
			Subject:   claims.UserID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newAccessClaims)
	accessTokenString, err := accessToken.SignedString(am.jwtSecret)
	if err != nil {
		return "", errors.NewInternalError("failed to generate new access token").WithCause(err)
	}

	return accessTokenString, nil
}

// extractToken extracts JWT token from request
func (am *AuthMiddleware) extractToken(c *gin.Context) (string, error) {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1], nil
		}
	}

	// Try cookie as fallback
	token, err := c.Cookie("access_token")
	if err == nil && token != "" {
		return token, nil
	}

	return "", errors.NewUnauthorizedError("no token found")
}

// validateToken validates and parses JWT token
func (am *AuthMiddleware) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.NewUnauthorizedError("invalid token claims")
}

// hasRequiredRole checks if user role meets requirements
func (am *AuthMiddleware) hasRequiredRole(userRole, requiredRole string) bool {
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

// InvalidateSession invalidates a user session (for logout)
func (am *AuthMiddleware) InvalidateSession(ctx context.Context, sessionID string) error {
	// In a production system, you would store invalidated sessions in Redis
	// or database to check against during token validation
	// For now, we'll just return success
	return nil
}

// GetUserFromContext extracts user information from gin context
func GetUserFromContext(c *gin.Context) (uuid.UUID, string, string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, "", "", errors.NewUnauthorizedError("user not authenticated")
	}

	email, exists := c.Get("user_email")
	if !exists {
		return uuid.Nil, "", "", errors.NewUnauthorizedError("user email not found")
	}

	role, exists := c.Get("user_role")
	if !exists {
		return uuid.Nil, "", "", errors.NewUnauthorizedError("user role not found")
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, "", "", errors.NewInternalError("invalid user ID format")
	}

	userEmail, ok := email.(string)
	if !ok {
		return uuid.Nil, "", "", errors.NewInternalError("invalid email format")
	}

	userRole, ok := role.(string)
	if !ok {
		return uuid.Nil, "", "", errors.NewInternalError("invalid role format")
	}

	return uid, userEmail, userRole, nil
}