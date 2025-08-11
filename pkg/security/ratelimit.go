package security

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
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Global rate limits
	GlobalRPS     int           // Requests per second globally
	GlobalBurst   int           // Burst capacity globally
	GlobalWindow  time.Duration // Time window for global limits
	
	// Per-IP rate limits
	PerIPRPS      int           // Requests per second per IP
	PerIPBurst    int           // Burst capacity per IP
	PerIPWindow   time.Duration // Time window for per-IP limits
	
	// Per-user rate limits (authenticated users)
	PerUserRPS    int           // Requests per second per user
	PerUserBurst  int           // Burst capacity per user
	PerUserWindow time.Duration // Time window for per-user limits
	
	// Endpoint-specific limits
	EndpointLimits map[string]EndpointLimit
	
	// DDoS protection
	DDoSThreshold     int           // Requests per second to trigger DDoS protection
	DDoSWindow        time.Duration // Time window for DDoS detection
	DDoSBlockDuration time.Duration // How long to block suspected DDoS sources
	
	// Whitelist and blacklist
	WhitelistedIPs []string // IPs that bypass rate limiting
	BlacklistedIPs []string // IPs that are always blocked
	
	// Redis configuration for distributed rate limiting
	RedisClient *redis.Client
	KeyPrefix   string
}

// EndpointLimit defines rate limits for specific endpoints
type EndpointLimit struct {
	RPS    int           // Requests per second
	Burst  int           // Burst capacity
	Window time.Duration // Time window
}

// DefaultRateLimitConfig returns a secure default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRPS:     1000,
		GlobalBurst:   2000,
		GlobalWindow:  time.Minute,
		PerIPRPS:      100,
		PerIPBurst:    200,
		PerIPWindow:   time.Minute,
		PerUserRPS:    500,
		PerUserBurst:  1000,
		PerUserWindow: time.Minute,
		EndpointLimits: map[string]EndpointLimit{
			"/api/auth/login": {
				RPS:    5,
				Burst:  10,
				Window: time.Minute,
			},
			"/api/auth/register": {
				RPS:    2,
				Burst:  5,
				Window: time.Minute,
			},
			"/api/scans": {
				RPS:    50,
				Burst:  100,
				Window: time.Minute,
			},
			"/api/scans/*/results": {
				RPS:    200,
				Burst:  400,
				Window: time.Minute,
			},
		},
		DDoSThreshold:     500,
		DDoSWindow:        time.Minute,
		DDoSBlockDuration: 15 * time.Minute,
		WhitelistedIPs:    []string{"127.0.0.1", "::1"},
		BlacklistedIPs:    []string{},
		KeyPrefix:         "agentscan:ratelimit:",
	}
}

// RateLimiter handles rate limiting with multiple strategies
type RateLimiter struct {
	config      RateLimitConfig
	redisClient *redis.Client
	localCache  *sync.Map // Fallback for when Redis is unavailable
	auditLogger *AuditLogger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig, auditLogger *AuditLogger) *RateLimiter {
	return &RateLimiter{
		config:      config,
		redisClient: config.RedisClient,
		localCache:  &sync.Map{},
		auditLogger: auditLogger,
	}
}

// RateLimitMiddleware returns a Gin middleware for rate limiting
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c.Request)
		userID := getUserIDFromContext(c)
		endpoint := c.Request.URL.Path
		
		// Check blacklist first
		if rl.isBlacklisted(clientIP) {
			rl.auditLogger.LogSecurityEvent(c.Request.Context(), EventTypeSecurityViolation,
				"Blacklisted IP attempted access", map[string]interface{}{
					"ip_address": clientIP,
					"endpoint":   endpoint,
				})
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
			})
			c.Abort()
			return
		}
		
		// Check whitelist
		if rl.isWhitelisted(clientIP) {
			c.Next()
			return
		}
		
		// Check DDoS protection
		if rl.isDDoSAttack(c.Request.Context(), clientIP) {
			rl.auditLogger.LogSecurityEvent(c.Request.Context(), EventTypeSuspiciousActivity,
				"Potential DDoS attack detected", map[string]interface{}{
					"ip_address": clientIP,
					"endpoint":   endpoint,
				})
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded - DDoS protection activated",
				"retry_after": int(rl.config.DDoSBlockDuration.Seconds()),
			})
			c.Abort()
			return
		}
		
		// Check rate limits in order of specificity
		limits := []struct {
			key    string
			limit  EndpointLimit
			reason string
		}{
			// Endpoint-specific limits (most specific)
			{
				key:    fmt.Sprintf("endpoint:%s:%s", endpoint, clientIP),
				limit:  rl.getEndpointLimit(endpoint),
				reason: "endpoint",
			},
			// Per-user limits (if authenticated)
			{
				key:    fmt.Sprintf("user:%s", userID),
				limit:  EndpointLimit{RPS: rl.config.PerUserRPS, Burst: rl.config.PerUserBurst, Window: rl.config.PerUserWindow},
				reason: "user",
			},
			// Per-IP limits
			{
				key:    fmt.Sprintf("ip:%s", clientIP),
				limit:  EndpointLimit{RPS: rl.config.PerIPRPS, Burst: rl.config.PerIPBurst, Window: rl.config.PerIPWindow},
				reason: "ip",
			},
			// Global limits (least specific)
			{
				key:    "global",
				limit:  EndpointLimit{RPS: rl.config.GlobalRPS, Burst: rl.config.GlobalBurst, Window: rl.config.GlobalWindow},
				reason: "global",
			},
		}
		
		for _, l := range limits {
			if l.limit.RPS == 0 {
				continue // Skip if limit is not configured
			}
			
			allowed, remaining, resetTime, err := rl.checkLimit(c.Request.Context(), l.key, l.limit)
			if err != nil {
				// Log error but don't block request
				rl.auditLogger.LogSecurityEvent(c.Request.Context(), EventTypeSecurityViolation,
					"Rate limit check failed", map[string]interface{}{
						"error":      err.Error(),
						"ip_address": clientIP,
						"endpoint":   endpoint,
					})
				continue
			}
			
			// Set rate limit headers
			c.Header("X-RateLimit-Limit", strconv.Itoa(l.limit.RPS))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
			
			if !allowed {
				rl.auditLogger.LogSecurityEvent(c.Request.Context(), EventTypeRateLimitExceeded,
					fmt.Sprintf("Rate limit exceeded (%s)", l.reason), map[string]interface{}{
						"ip_address": clientIP,
						"user_id":    userID,
						"endpoint":   endpoint,
						"limit_type": l.reason,
						"limit_rps":  l.limit.RPS,
					})
				
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":       "Rate limit exceeded",
					"limit_type":  l.reason,
					"retry_after": int(resetTime.Sub(time.Now()).Seconds()),
				})
				c.Abort()
				return
			}
		}
		
		c.Next()
	}
}

// checkLimit checks if a request is within the rate limit
func (rl *RateLimiter) checkLimit(ctx context.Context, key string, limit EndpointLimit) (allowed bool, remaining int, resetTime time.Time, err error) {
	fullKey := rl.config.KeyPrefix + key
	now := time.Now()
	windowStart := now.Truncate(limit.Window)
	resetTime = windowStart.Add(limit.Window)
	
	if rl.redisClient != nil {
		return rl.checkLimitRedis(ctx, fullKey, limit, windowStart, resetTime)
	}
	
	return rl.checkLimitLocal(fullKey, limit, windowStart, resetTime)
}

// checkLimitRedis checks rate limit using Redis
func (rl *RateLimiter) checkLimitRedis(ctx context.Context, key string, limit EndpointLimit, windowStart, resetTime time.Time) (bool, int, time.Time, error) {
	pipe := rl.redisClient.Pipeline()
	
	// Increment counter
	incrCmd := pipe.Incr(ctx, key)
	// Set expiration
	pipe.ExpireAt(ctx, key, resetTime)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, resetTime, fmt.Errorf("redis pipeline failed: %w", err)
	}
	
	count := int(incrCmd.Val())
	remaining := limit.Burst - count
	if remaining < 0 {
		remaining = 0
	}
	
	allowed := count <= limit.Burst
	return allowed, remaining, resetTime, nil
}

// checkLimitLocal checks rate limit using local cache (fallback)
func (rl *RateLimiter) checkLimitLocal(key string, limit EndpointLimit, windowStart, resetTime time.Time) (bool, int, time.Time, error) {
	type counter struct {
		count     int
		window    time.Time
		mutex     sync.Mutex
	}
	
	value, _ := rl.localCache.LoadOrStore(key, &counter{
		window: windowStart,
	})
	
	c := value.(*counter)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Reset counter if we're in a new window
	if c.window.Before(windowStart) {
		c.count = 0
		c.window = windowStart
	}
	
	c.count++
	remaining := limit.Burst - c.count
	if remaining < 0 {
		remaining = 0
	}
	
	allowed := c.count <= limit.Burst
	return allowed, remaining, resetTime, nil
}

// isDDoSAttack checks if the current request pattern indicates a DDoS attack
func (rl *RateLimiter) isDDoSAttack(ctx context.Context, clientIP string) bool {
	if rl.config.DDoSThreshold == 0 {
		return false
	}
	
	key := rl.config.KeyPrefix + "ddos:" + clientIP
	now := time.Now()
	windowStart := now.Truncate(rl.config.DDoSWindow)
	
	if rl.redisClient != nil {
		// Check if IP is already blocked
		blockKey := rl.config.KeyPrefix + "ddos:blocked:" + clientIP
		blocked, err := rl.redisClient.Exists(ctx, blockKey).Result()
		if err == nil && blocked > 0 {
			return true
		}
		
		// Increment DDoS counter
		count, err := rl.redisClient.Incr(ctx, key).Result()
		if err != nil {
			return false
		}
		
		// Set expiration for the counter
		rl.redisClient.ExpireAt(ctx, key, windowStart.Add(rl.config.DDoSWindow))
		
		// If threshold exceeded, block the IP
		if int(count) > rl.config.DDoSThreshold {
			rl.redisClient.Set(ctx, blockKey, "1", rl.config.DDoSBlockDuration)
			return true
		}
	}
	
	return false
}

// isWhitelisted checks if an IP is whitelisted
func (rl *RateLimiter) isWhitelisted(ip string) bool {
	for _, whiteIP := range rl.config.WhitelistedIPs {
		if ip == whiteIP || matchIPPattern(ip, whiteIP) {
			return true
		}
	}
	return false
}

// isBlacklisted checks if an IP is blacklisted
func (rl *RateLimiter) isBlacklisted(ip string) bool {
	for _, blackIP := range rl.config.BlacklistedIPs {
		if ip == blackIP || matchIPPattern(ip, blackIP) {
			return true
		}
	}
	return false
}

// getEndpointLimit returns the rate limit for a specific endpoint
func (rl *RateLimiter) getEndpointLimit(endpoint string) EndpointLimit {
	// Check exact match first
	if limit, exists := rl.config.EndpointLimits[endpoint]; exists {
		return limit
	}
	
	// Check pattern matches
	for pattern, limit := range rl.config.EndpointLimits {
		if matchEndpointPattern(endpoint, pattern) {
			return limit
		}
	}
	
	return EndpointLimit{} // No specific limit
}

// matchEndpointPattern matches endpoint patterns (supports wildcards)
func matchEndpointPattern(endpoint, pattern string) bool {
	if pattern == endpoint {
		return true
	}
	
	// Simple wildcard matching for patterns like "/api/scans/*/results"
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(endpoint, parts[0]) && strings.HasSuffix(endpoint, parts[1])
		}
	}
	
	return false
}

// matchIPPattern matches IP patterns (basic CIDR support could be added)
func matchIPPattern(ip, pattern string) bool {
	// For now, just exact match. Could be extended to support CIDR notation
	return ip == pattern
}

// getUserIDFromContext extracts user ID from Gin context
func getUserIDFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// AddToBlacklist adds an IP to the blacklist
func (rl *RateLimiter) AddToBlacklist(ip string) {
	rl.config.BlacklistedIPs = append(rl.config.BlacklistedIPs, ip)
}

// RemoveFromBlacklist removes an IP from the blacklist
func (rl *RateLimiter) RemoveFromBlacklist(ip string) {
	for i, blackIP := range rl.config.BlacklistedIPs {
		if blackIP == ip {
			rl.config.BlacklistedIPs = append(rl.config.BlacklistedIPs[:i], rl.config.BlacklistedIPs[i+1:]...)
			break
		}
	}
}

// GetRateLimitStatus returns the current rate limit status for a key
func (rl *RateLimiter) GetRateLimitStatus(ctx context.Context, key string, limit EndpointLimit) (remaining int, resetTime time.Time, err error) {
	fullKey := rl.config.KeyPrefix + key
	now := time.Now()
	windowStart := now.Truncate(limit.Window)
	resetTime = windowStart.Add(limit.Window)
	
	if rl.redisClient != nil {
		count, err := rl.redisClient.Get(ctx, fullKey).Int()
		if err != nil && err != redis.Nil {
			return 0, resetTime, err
		}
		remaining = limit.Burst - count
	} else {
		// Local cache fallback
		if value, exists := rl.localCache.Load(fullKey); exists {
			if c, ok := value.(*struct {
				count  int
				window time.Time
				mutex  sync.Mutex
			}); ok {
				c.mutex.Lock()
				remaining = limit.Burst - c.count
				c.mutex.Unlock()
			}
		} else {
			remaining = limit.Burst
		}
	}
	
	if remaining < 0 {
		remaining = 0
	}
	
	return remaining, resetTime, nil
}