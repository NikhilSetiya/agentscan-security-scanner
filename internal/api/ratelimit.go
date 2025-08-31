package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
)

// RateLimiter interface for different rate limiting strategies
type RateLimiter interface {
	Allow(key string) (bool, *RateLimitInfo, error)
	Reset(key string) error
	Cleanup() error
}

// RateLimitInfo contains information about rate limit status
type RateLimitInfo struct {
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
	ResetTime time.Time     `json:"reset_time"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Global rate limits
	GlobalRPM     int           `json:"global_rpm"`     // Requests per minute globally
	GlobalBurst   int           `json:"global_burst"`   // Burst capacity
	
	// Per-IP rate limits
	IPRPM         int           `json:"ip_rpm"`         // Requests per minute per IP
	IPBurst       int           `json:"ip_burst"`       // Burst capacity per IP
	
	// Per-User rate limits
	UserRPM       int           `json:"user_rpm"`       // Requests per minute per user
	UserBurst     int           `json:"user_burst"`     // Burst capacity per user
	
	// Endpoint-specific limits
	EndpointLimits map[string]EndpointLimit `json:"endpoint_limits"`
	
	// Configuration
	WindowSize    time.Duration `json:"window_size"`    // Rate limit window
	CleanupInterval time.Duration `json:"cleanup_interval"` // Cleanup interval for expired entries
	Enabled       bool          `json:"enabled"`        // Enable/disable rate limiting
	Whitelist     []string      `json:"whitelist"`      // IP whitelist
	Blacklist     []string      `json:"blacklist"`      // IP blacklist
}

// EndpointLimit defines rate limits for specific endpoints
type EndpointLimit struct {
	RPM   int `json:"rpm"`   // Requests per minute
	Burst int `json:"burst"` // Burst capacity
}

// InMemoryRateLimiter implements rate limiting using in-memory storage
type InMemoryRateLimiter struct {
	config    *RateLimitConfig
	buckets   map[string]*TokenBucket
	mutex     sync.RWMutex
	lastCleanup time.Time
}

// RedisRateLimiter implements rate limiting using Redis
type RedisRateLimiter struct {
	config *RateLimitConfig
	redis  *queue.RedisClient
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	Capacity     int       `json:"capacity"`
	Tokens       int       `json:"tokens"`
	RefillRate   int       `json:"refill_rate"`
	LastRefill   time.Time `json:"last_refill"`
	LastAccess   time.Time `json:"last_access"`
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter(config *RateLimitConfig) *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		config:    config,
		buckets:   make(map[string]*TokenBucket),
		lastCleanup: time.Now(),
	}
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(config *RateLimitConfig, redis *queue.RedisClient) *RedisRateLimiter {
	return &RedisRateLimiter{
		config: config,
		redis:  redis,
	}
}

// Allow checks if a request should be allowed for in-memory rate limiter
func (rl *InMemoryRateLimiter) Allow(key string) (bool, *RateLimitInfo, error) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// Cleanup old entries periodically
	if time.Since(rl.lastCleanup) > rl.config.CleanupInterval {
		rl.cleanup()
		rl.lastCleanup = time.Now()
	}
	
	bucket, exists := rl.buckets[key]
	if !exists {
		// Create new bucket
		bucket = &TokenBucket{
			Capacity:   rl.config.IPBurst, // Default to IP burst
			Tokens:     rl.config.IPBurst,
			RefillRate: rl.config.IPRPM,
			LastRefill: time.Now(),
			LastAccess: time.Now(),
		}
		rl.buckets[key] = bucket
	}
	
	// Refill tokens
	now := time.Now()
	timePassed := now.Sub(bucket.LastRefill)
	tokensToAdd := int(timePassed.Minutes() * float64(bucket.RefillRate))
	
	if tokensToAdd > 0 {
		bucket.Tokens = min(bucket.Capacity, bucket.Tokens+tokensToAdd)
		bucket.LastRefill = now
	}
	
	bucket.LastAccess = now
	
	// Check if request can be allowed
	if bucket.Tokens > 0 {
		bucket.Tokens--
		return true, &RateLimitInfo{
			Limit:     bucket.Capacity,
			Remaining: bucket.Tokens,
			ResetTime: bucket.LastRefill.Add(time.Minute),
		}, nil
	}
	
	// Request denied
	retryAfter := time.Minute - time.Since(bucket.LastRefill)
	return false, &RateLimitInfo{
		Limit:      bucket.Capacity,
		Remaining:  0,
		ResetTime:  bucket.LastRefill.Add(time.Minute),
		RetryAfter: retryAfter,
	}, nil
}

// Reset resets the rate limit for a key
func (rl *InMemoryRateLimiter) Reset(key string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	delete(rl.buckets, key)
	return nil
}

// Cleanup removes expired entries
func (rl *InMemoryRateLimiter) Cleanup() error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.cleanup()
	return nil
}

// cleanup removes expired entries (internal method)
func (rl *InMemoryRateLimiter) cleanup() {
	now := time.Now()
	for key, bucket := range rl.buckets {
		// Remove buckets that haven't been accessed for 2x cleanup interval
		if now.Sub(bucket.LastAccess) > 2*rl.config.CleanupInterval {
			delete(rl.buckets, key)
		}
	}
}

// Allow checks if a request should be allowed for Redis rate limiter
func (rl *RedisRateLimiter) Allow(key string) (bool, *RateLimitInfo, error) {
	ctx := context.Background()
	
	// Use Redis sliding window rate limiting
	now := time.Now()
	windowStart := now.Add(-rl.config.WindowSize)
	
	// Remove old entries
	err := rl.redis.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.Unix()))
	if err != nil {
		return false, nil, err
	}
	
	// Count current requests in window
	count, err := rl.redis.ZCard(ctx, key)
	if err != nil {
		return false, nil, err
	}
	
	limit := rl.config.IPRPM // Default to IP limit
	
	if int(count) >= limit {
		// Rate limit exceeded
		retryAfter := rl.config.WindowSize - time.Since(windowStart)
		return false, &RateLimitInfo{
			Limit:      limit,
			Remaining:  0,
			ResetTime:  now.Add(retryAfter),
			RetryAfter: retryAfter,
		}, nil
	}
	
	// Add current request
	member := redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	}
	err = rl.redis.ZAdd(ctx, key, member)
	if err != nil {
		return false, nil, err
	}
	
	// Set expiration
	err = rl.redis.Expire(ctx, key, rl.config.WindowSize)
	if err != nil {
		return false, nil, err
	}
	
	return true, &RateLimitInfo{
		Limit:     limit,
		Remaining: limit - int(count) - 1,
		ResetTime: now.Add(rl.config.WindowSize),
	}, nil
}

// Reset resets the rate limit for a key in Redis
func (rl *RedisRateLimiter) Reset(key string) error {
	ctx := context.Background()
	_, err := rl.redis.Del(ctx, key)
	return err
}

// Cleanup is not needed for Redis as it handles expiration automatically
func (rl *RedisRateLimiter) Cleanup() error {
	return nil
}

// RateLimitMiddleware creates a rate limiting middleware
type RateLimitMiddleware struct {
	limiter RateLimiter
	config  *RateLimitConfig
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(limiter RateLimiter, config *RateLimitConfig) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		config:  config,
	}
}

// Handler returns the HTTP middleware handler
func (rlm *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting if disabled
		if !rlm.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		
		// Get client IP
		clientIP := getClientIP(r)
		
		// Check if IP is whitelisted
		if rlm.isWhitelisted(clientIP) {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check if IP is blacklisted
		if rlm.isBlacklisted(clientIP) {
			http.Error(w, "IP address is blacklisted", http.StatusForbidden)
			return
		}
		
		// Determine rate limit key and limits
		keys := rlm.getRateLimitKeys(r, clientIP)
		
		// Check each rate limit
		for keyType, key := range keys {
			allowed, info, err := rlm.limiter.Allow(key)
			if err != nil {
				http.Error(w, "Rate limiting error", http.StatusInternalServerError)
				return
			}
			
			// Set rate limit headers
			rlm.setRateLimitHeaders(w, keyType, info)
			
			if !allowed {
				// Rate limit exceeded
				rlm.handleRateLimitExceeded(w, r, keyType, info)
				return
			}
		}
		
		next.ServeHTTP(w, r)
	})
}

// getRateLimitKeys returns the keys to check for rate limiting
func (rlm *RateLimitMiddleware) getRateLimitKeys(r *http.Request, clientIP string) map[string]string {
	keys := make(map[string]string)
	
	// IP-based rate limiting
	keys["ip"] = fmt.Sprintf("rate_limit:ip:%s", clientIP)
	
	// User-based rate limiting (if authenticated)
	if userID := r.Context().Value("user_id"); userID != nil {
		keys["user"] = fmt.Sprintf("rate_limit:user:%v", userID)
	}
	
	// Endpoint-specific rate limiting
	endpoint := getEndpointKey(r)
	if limit, exists := rlm.config.EndpointLimits[endpoint]; exists && limit.RPM > 0 {
		keys["endpoint"] = fmt.Sprintf("rate_limit:endpoint:%s:%s", endpoint, clientIP)
	}
	
	// Global rate limiting
	keys["global"] = "rate_limit:global"
	
	return keys
}

// setRateLimitHeaders sets rate limit headers in the response
func (rlm *RateLimitMiddleware) setRateLimitHeaders(w http.ResponseWriter, keyType string, info *RateLimitInfo) {
	prefix := fmt.Sprintf("X-RateLimit-%s", strings.Title(keyType))
	
	w.Header().Set(fmt.Sprintf("%s-Limit", prefix), strconv.Itoa(info.Limit))
	w.Header().Set(fmt.Sprintf("%s-Remaining", prefix), strconv.Itoa(info.Remaining))
	w.Header().Set(fmt.Sprintf("%s-Reset", prefix), strconv.FormatInt(info.ResetTime.Unix(), 10))
	
	if info.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(info.RetryAfter.Seconds())))
	}
}

// handleRateLimitExceeded handles rate limit exceeded responses
func (rlm *RateLimitMiddleware) handleRateLimitExceeded(w http.ResponseWriter, r *http.Request, keyType string, info *RateLimitInfo) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	
	errorResponse := map[string]interface{}{
		"error": "Rate limit exceeded",
		"message": fmt.Sprintf("%s rate limit exceeded", keyType),
		"limit": info.Limit,
		"remaining": info.Remaining,
		"reset_time": info.ResetTime.Unix(),
	}
	
	if info.RetryAfter > 0 {
		errorResponse["retry_after"] = int(info.RetryAfter.Seconds())
	}
	
	json.NewEncoder(w).Encode(errorResponse)
}

// isWhitelisted checks if an IP is whitelisted
func (rlm *RateLimitMiddleware) isWhitelisted(ip string) bool {
	for _, whitelistedIP := range rlm.config.Whitelist {
		if ip == whitelistedIP || rlm.matchesCIDR(ip, whitelistedIP) {
			return true
		}
	}
	return false
}

// isBlacklisted checks if an IP is blacklisted
func (rlm *RateLimitMiddleware) isBlacklisted(ip string) bool {
	for _, blacklistedIP := range rlm.config.Blacklist {
		if ip == blacklistedIP || rlm.matchesCIDR(ip, blacklistedIP) {
			return true
		}
	}
	return false
}

// matchesCIDR checks if an IP matches a CIDR range
func (rlm *RateLimitMiddleware) matchesCIDR(ip, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}
	
	return network.Contains(ipAddr)
}

// Helper functions

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// getEndpointKey generates a key for endpoint-specific rate limiting
func getEndpointKey(r *http.Request) string {
	// Normalize the endpoint path
	path := r.URL.Path
	method := r.Method
	
	// Remove trailing slashes
	path = strings.TrimSuffix(path, "/")
	
	// Replace path parameters with placeholders
	// This is a simple implementation - you might want to use a router for this
	if strings.Contains(path, "/api/v1/repositories/") {
		path = "/api/v1/repositories/{id}"
	}
	if strings.Contains(path, "/api/v1/scans/") {
		path = "/api/v1/scans/{id}"
	}
	
	return fmt.Sprintf("%s %s", method, path)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DefaultRateLimitConfig returns a default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		GlobalRPM:   1000,
		GlobalBurst: 100,
		IPRPM:       100,
		IPBurst:     20,
		UserRPM:     200,
		UserBurst:   50,
		EndpointLimits: map[string]EndpointLimit{
			"POST /api/v1/scans": {RPM: 10, Burst: 5},
			"POST /api/v1/repositories": {RPM: 20, Burst: 10},
			"GET /api/v1/dashboard": {RPM: 60, Burst: 20},
		},
		WindowSize:      time.Minute,
		CleanupInterval: 5 * time.Minute,
		Enabled:         true,
		Whitelist:       []string{"127.0.0.1", "::1"},
		Blacklist:       []string{},
	}
}