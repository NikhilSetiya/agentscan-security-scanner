package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// CORSMiddleware handles CORS headers with environment-aware configuration
func CORSMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		originAllowed := isOriginAllowed(origin)
		
		// Set CORS headers
		if originAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			// In development, allow all origins for easier testing
			if gin.Mode() == gin.DebugMode {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				// In production, be more restrictive
				c.Header("Access-Control-Allow-Origin", "")
			}
		}
		
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Header("Access-Control-Max-Age", "86400") // Cache preflight for 24 hours

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

// isOriginAllowed checks if the origin is allowed based on environment and patterns
func isOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}

	// Always allow localhost for development
	localOrigins := []string{
		"http://localhost:3000",
		"http://localhost:4173",
		"http://localhost:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:4173",
		"http://127.0.0.1:5173",
	}
	
	for _, localOrigin := range localOrigins {
		if origin == localOrigin {
			return true
		}
	}

	// Allow Vercel preview and production deployments
	if strings.Contains(origin, "vercel.app") {
		// Allow any Vercel deployment for this project
		if strings.Contains(origin, "nikhilsetiyas-projects.vercel.app") ||
		   strings.Contains(origin, "agentscan") ||
		   strings.Contains(origin, "frontend-") {
			return true
		}
	}

	// Allow custom domains (add your production domains here)
	productionDomains := []string{
		"https://agentscan.dev",
		"https://www.agentscan.dev",
		"https://app.agentscan.dev",
	}
	
	for _, domain := range productionDomains {
		if origin == domain {
			return true
		}
	}

	return false
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
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

// LoggingMiddleware provides structured logging for requests
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
	})
}

// ErrorHandlingMiddleware handles panics and errors
func ErrorHandlingMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Name   string    `json:"name"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			UnauthorizedResponse(c, "Authorization header is required")
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			UnauthorizedResponse(c, "Authorization header must be in format 'Bearer <token>'")
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Try Supabase token validation first
		if user, err := validateSupabaseTokenInAPI(c.Request.Context(), tokenString); err == nil {
			c.Set("user", user)
			c.Set("user_id", user.ID)
			c.Next()
			return
		}

		// Fallback to JWT token validation
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Auth.JWTSecret), nil
		})

		if err != nil {
			UnauthorizedResponse(c, "Invalid or expired token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok || !token.Valid {
			UnauthorizedResponse(c, "Invalid token claims")
			c.Abort()
			return
		}

		// Check token expiration
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			UnauthorizedResponse(c, "Token has expired")
			c.Abort()
			return
		}

		// Set user context
		user := &types.User{
			ID:    claims.UserID,
			Email: claims.Email,
			Name:  claims.Name,
		}
		c.Set("user", user)
		c.Set("user_id", claims.UserID)

		c.Next()
	})
}

// OptionalAuthMiddleware validates JWT tokens if present but doesn't require them
func OptionalAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := tokenParts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Auth.JWTSecret), nil
		})

		if err == nil {
			if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
				// Check token expiration
				if claims.ExpiresAt == nil || claims.ExpiresAt.Time.After(time.Now()) {
					// Set user context
					user := &types.User{
						ID:    claims.UserID,
						Email: claims.Email,
						Name:  claims.Name,
					}
					c.Set("user", user)
					c.Set("user_id", claims.UserID)
				}
			}
		}

		c.Next()
	})
}

// RateLimitMiddleware provides Redis-based rate limiting
func RateLimitMiddleware(redis *queue.RedisClient) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if redis == nil {
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()
		
		// Create rate limit key
		key := fmt.Sprintf("rate_limit:%s", clientIP)
		
		// Get current request count
		ctx := c.Request.Context()
		count, err := redis.Client().Get(ctx, key).Int()
		if err != nil && err.Error() != "redis: nil" {
			// Redis error, allow request but log error
			c.Next()
			return
		}

		// Rate limit: 100 requests per minute per IP
		limit := 100
		window := 60 // seconds

		if count >= limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"retry_after": window,
			})
			c.Abort()
			return
		}

		// Increment counter
		pipe := redis.Client().Pipeline()
		pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Duration(window)*time.Second)
		_, err = pipe.Exec(ctx)
		if err != nil {
			// Redis error, allow request but log error
			// TODO: Add proper logging
		}

		c.Next()
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

// validateSupabaseTokenInAPI validates a Supabase JWT token and returns user info for API middleware
func validateSupabaseTokenInAPI(ctx context.Context, tokenString string) (*types.User, error) {
	// Parse the JWT token to extract claims
	// Supabase tokens are JWT tokens signed with the project's secret
	supabaseJWTSecret := os.Getenv("SUPABASE_JWT_SECRET")
	if supabaseJWTSecret == "" {
		return nil, fmt.Errorf("SUPABASE_JWT_SECRET environment variable is required")
	}

	// Parse and validate Supabase JWT token
	token, err := jwt.ParseWithClaims(tokenString, &SupabaseJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(supabaseJWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse Supabase token: %w", err)
	}

	claims, ok := token.Claims.(*SupabaseJWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid Supabase token claims")
	}

	// Check token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("Supabase token has expired")
	}

	// Extract user information from claims
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in Supabase token: %w", err)
	}

	// Extract email and name from user metadata
	email := ""
	name := ""
	
	if claims.Email != "" {
		email = claims.Email
	}

	if claims.UserMetadata != nil {
		if nameVal, ok := claims.UserMetadata["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
			}
		}
	}

	// If no name in metadata, try to extract from email
	if name == "" && email != "" {
		if atIndex := strings.Index(email, "@"); atIndex > 0 {
			name = email[:atIndex]
		}
	}

	user := &types.User{
		ID:    userID,
		Email: email,
		Name:  name,
	}

	return user, nil
}

// SupabaseJWTClaims represents the JWT token claims from Supabase
type SupabaseJWTClaims struct {
	Email        string                 `json:"email"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	AppMetadata  map[string]interface{} `json:"app_metadata"`
	Role         string                 `json:"role"`
	jwt.RegisteredClaims
}