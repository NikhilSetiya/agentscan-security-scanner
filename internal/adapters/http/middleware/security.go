package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/agentscan/agentscan/pkg/errors"
)

// SecurityMiddleware provides comprehensive security middleware
type SecurityMiddleware struct {
	redisClient *redis.Client
	rateLimits  map[string]RateLimit
	mu          sync.RWMutex
}

// RateLimit defines rate limiting configuration
type RateLimit struct {
	Requests int           // Number of requests allowed
	Window   time.Duration // Time window
	Burst    int           // Burst capacity
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(redisClient *redis.Client) *SecurityMiddleware {
	return &SecurityMiddleware{
		redisClient: redisClient,
		rateLimits: map[string]RateLimit{
			"default": {Requests: 100, Window: time.Minute, Burst: 10},
			"auth":    {Requests: 5, Window: time.Minute, Burst: 2},
			"api":     {Requests: 1000, Window: time.Hour, Burst: 50},
			"scan":    {Requests: 10, Window: time.Minute, Burst: 3},
			"upload":  {Requests: 5, Window: time.Minute, Burst: 1},
		},
	}
}

// SecurityHeaders adds comprehensive security headers
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Control referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net https://unpkg.com; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; " +
			"font-src 'self' https://fonts.gstatic.com; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self' https://api.supabase.co wss://realtime.supabase.co; " +
			"object-src 'none'; " +
			"base-uri 'self'"
		c.Header("Content-Security-Policy", csp)
		
		// Strict Transport Security (HTTPS only)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		
		// Permissions Policy
		permissionsPolicy := "camera=(), microphone=(), geolocation=(), payment=(), usb=()"
		c.Header("Permissions-Policy", permissionsPolicy)
		
		c.Next()
	}
}

// CORS configures Cross-Origin Resource Sharing
func (sm *SecurityMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Define allowed origins based on environment
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"https://agentscan-frontend.vercel.app",
			"https://agentscan.dev",
		}
		
		// Check if origin is allowed
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}
		
		if isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "null")
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// RateLimit implements enhanced rate limiting with comprehensive features
func (sm *SecurityMiddleware) RateLimit(limitType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier (IP + User ID if available)
		clientID := sm.getClientID(c)
		clientIP := c.ClientIP()
		
		// Check IP whitelist/blacklist first
		if sm.isIPBlacklisted(clientIP) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "IP address is blacklisted",
				"code":  "IP_BLACKLISTED",
			})
			c.Abort()
			return
		}
		
		if sm.isIPWhitelisted(clientIP) {
			c.Next()
			return
		}
		
		// Get rate limit configuration
		sm.mu.RLock()
		limit, exists := sm.rateLimits[limitType]
		if !exists {
			limit = sm.rateLimits["default"]
		}
		sm.mu.RUnlock()
		
		// Check multiple rate limits (IP, User, Endpoint, Global)
		rateLimitKeys := sm.getRateLimitKeys(c, clientID, limitType)
		
		for keyType, key := range rateLimitKeys {
			allowed, remaining, resetTime, err := sm.checkRateLimit(c.Request.Context(), key, limitType, limit)
			if err != nil {
				// Log error but don't block request
				continue
			}
			
			// Add rate limit headers for each type
			headerPrefix := fmt.Sprintf("X-RateLimit-%s", strings.Title(keyType))
			c.Header(fmt.Sprintf("%s-Limit", headerPrefix), strconv.Itoa(limit.Requests))
			c.Header(fmt.Sprintf("%s-Remaining", headerPrefix), strconv.Itoa(remaining))
			c.Header(fmt.Sprintf("%s-Reset", headerPrefix), strconv.FormatInt(resetTime, 10))
			
			if !allowed {
				retryAfter := resetTime - time.Now().Unix()
				if retryAfter > 0 {
					c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
				}
				
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":      "Rate limit exceeded",
					"message":    fmt.Sprintf("%s rate limit exceeded", keyType),
					"limit_type": keyType,
					"limit":      limit.Requests,
					"remaining":  remaining,
					"reset_time": resetTime,
					"retry_after": retryAfter,
				})
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}

// RequestLogging logs requests with security context
func (sm *SecurityMiddleware) RequestLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Generate request ID if not present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			c.Header("X-Request-ID", requestID)
		}
		c.Set("request_id", requestID)
		
		// Log request start
		logRequestStart(c, requestID)
		
		c.Next()
		
		// Log request completion
		logRequestEnd(c, requestID, time.Since(start))
	}
}

// InputSanitization sanitizes request inputs
func (sm *SecurityMiddleware) InputSanitization() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Sanitize headers
		sm.sanitizeHeaders(c)
		
		// Sanitize query parameters
		sm.sanitizeQueryParams(c)
		
		c.Next()
	}
}

// getClientID generates a unique client identifier for rate limiting
func (sm *SecurityMiddleware) getClientID(c *gin.Context) string {
	// Try to get user ID from context first
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}
	
	// Fall back to IP address
	clientIP := c.ClientIP()
	
	// Consider X-Forwarded-For header for load balancers
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			clientIP = strings.TrimSpace(ips[0])
		}
	}
	
	return fmt.Sprintf("ip:%s", clientIP)
}

// checkRateLimit checks if request is within rate limits using Redis
func (sm *SecurityMiddleware) checkRateLimit(ctx context.Context, clientID, limitType string, limit RateLimit) (bool, int, int64, error) {
	if sm.redisClient == nil {
		// If Redis is not available, allow all requests
		return true, limit.Requests, time.Now().Add(limit.Window).Unix(), nil
	}
	
	key := fmt.Sprintf("rate_limit:%s:%s", limitType, clientID)
	now := time.Now()
	window := now.Truncate(limit.Window)
	
	// Use Redis pipeline for atomic operations
	pipe := sm.redisClient.Pipeline()
	
	// Increment counter
	incrCmd := pipe.Incr(ctx, key)
	
	// Set expiration if key is new
	pipe.ExpireAt(ctx, key, window.Add(limit.Window))
	
	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, 0, err
	}
	
	count := int(incrCmd.Val())
	remaining := limit.Requests - count
	if remaining < 0 {
		remaining = 0
	}
	
	resetTime := window.Add(limit.Window).Unix()
	allowed := count <= limit.Requests
	
	return allowed, remaining, resetTime, nil
}

// sanitizeHeaders removes potentially dangerous headers
func (sm *SecurityMiddleware) sanitizeHeaders(c *gin.Context) {
	dangerousHeaders := []string{
		"X-Forwarded-Host",
		"X-Original-URL",
		"X-Rewrite-URL",
	}
	
	for _, header := range dangerousHeaders {
		c.Request.Header.Del(header)
	}
}

// sanitizeQueryParams sanitizes query parameters
func (sm *SecurityMiddleware) sanitizeQueryParams(c *gin.Context) {
	query := c.Request.URL.Query()
	
	for key, values := range query {
		for i, value := range values {
			// Remove null bytes and control characters
			sanitized := strings.ReplaceAll(value, "\x00", "")
			sanitized = strings.Map(func(r rune) rune {
				if r < 32 && r != 9 && r != 10 && r != 13 {
					return -1
				}
				return r
			}, sanitized)
			
			query[key][i] = sanitized
		}
	}
	
	c.Request.URL.RawQuery = query.Encode()
}

// getRateLimitKeys returns multiple rate limit keys to check
func (sm *SecurityMiddleware) getRateLimitKeys(c *gin.Context, clientID, limitType string) map[string]string {
	keys := make(map[string]string)
	
	// IP-based rate limiting
	keys["ip"] = fmt.Sprintf("rate_limit:ip:%s:%s", limitType, c.ClientIP())
	
	// User-based rate limiting (if authenticated)
	if userID, exists := c.Get("user_id"); exists {
		keys["user"] = fmt.Sprintf("rate_limit:user:%s:%v", limitType, userID)
	}
	
	// Endpoint-specific rate limiting
	endpoint := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
	keys["endpoint"] = fmt.Sprintf("rate_limit:endpoint:%s:%s", endpoint, c.ClientIP())
	
	// Global rate limiting
	keys["global"] = fmt.Sprintf("rate_limit:global:%s", limitType)
	
	return keys
}

// isIPWhitelisted checks if an IP is in the whitelist
func (sm *SecurityMiddleware) isIPWhitelisted(ip string) bool {
	// Default whitelist for localhost and common development IPs
	whitelist := []string{
		"127.0.0.1",
		"::1",
		"localhost",
	}
	
	for _, whitelistedIP := range whitelist {
		if ip == whitelistedIP {
			return true
		}
	}
	return false
}

// isIPBlacklisted checks if an IP is in the blacklist
func (sm *SecurityMiddleware) isIPBlacklisted(ip string) bool {
	// This would typically be loaded from configuration or database
	// For now, return false (no IPs blacklisted by default)
	return false
}

// UpdateRateLimit allows dynamic rate limit configuration
func (sm *SecurityMiddleware) UpdateRateLimit(limitType string, limit RateLimit) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.rateLimits[limitType] = limit
}

// GetRateLimit returns the current rate limit configuration
func (sm *SecurityMiddleware) GetRateLimit(limitType string) (RateLimit, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	limit, exists := sm.rateLimits[limitType]
	return limit, exists
}

// ResetRateLimit resets rate limiting for a specific key
func (sm *SecurityMiddleware) ResetRateLimit(ctx context.Context, key string) error {
	if sm.redisClient == nil {
		return nil
	}
	return sm.redisClient.Del(ctx, key).Err()
}

// Helper functions for logging

func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

func logRequestStart(c *gin.Context, requestID string) {
	// Log request start with security context
	fmt.Printf("[%s] %s %s %s - User-Agent: %s, IP: %s\n",
		requestID,
		time.Now().Format(time.RFC3339),
		c.Request.Method,
		c.Request.URL.Path,
		c.GetHeader("User-Agent"),
		c.ClientIP(),
	)
}

func logRequestEnd(c *gin.Context, requestID string, duration time.Duration) {
	// Log request completion
	fmt.Printf("[%s] Completed in %v - Status: %d\n",
		requestID,
		duration,
		c.Writer.Status(),
	)
}