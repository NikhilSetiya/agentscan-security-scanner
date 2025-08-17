package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
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

		// Parse and validate token
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