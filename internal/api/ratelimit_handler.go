package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/adapters/http/middleware"
)

// RateLimitAdminHandler provides admin endpoints for rate limit management
type RateLimitAdminHandler struct {
	securityMiddleware *middleware.SecurityMiddleware
}

// NewRateLimitAdminHandler creates a new rate limit admin handler
func NewRateLimitAdminHandler(securityMiddleware *middleware.SecurityMiddleware) *RateLimitAdminHandler {
	return &RateLimitAdminHandler{
		securityMiddleware: securityMiddleware,
	}
}

// GetRateLimitConfig returns the current rate limiting configuration
func (rlh *RateLimitAdminHandler) GetRateLimitConfig(c *gin.Context) {
	limitType := c.DefaultQuery("type", "default")
	
	limit, exists := rlh.securityMiddleware.GetRateLimit(limitType)
	if !exists {
		ErrorResponse(c, http.StatusNotFound, "Rate limit type not found", nil)
		return
	}
	
	SuccessResponse(c, gin.H{
		"limit_type": limitType,
		"requests":   limit.Requests,
		"window":     limit.Window.String(),
		"burst":      limit.Burst,
	})
}

// UpdateRateLimitConfig updates rate limiting configuration
func (rlh *RateLimitAdminHandler) UpdateRateLimitConfig(c *gin.Context) {
	limitType := c.Param("type")
	if limitType == "" {
		ErrorResponse(c, http.StatusBadRequest, "Rate limit type is required", nil)
		return
	}
	
	var req struct {
		Requests int    `json:"requests" binding:"required,min=1"`
		Window   string `json:"window" binding:"required"`
		Burst    int    `json:"burst" binding:"required,min=1"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	// Parse window duration
	window, err := parseDuration(req.Window)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid window duration", err)
		return
	}
	
	// Update rate limit
	newLimit := middleware.RateLimit{
		Requests: req.Requests,
		Window:   window,
		Burst:    req.Burst,
	}
	
	rlh.securityMiddleware.UpdateRateLimit(limitType, newLimit)
	
	SuccessResponse(c, gin.H{
		"message":    "Rate limit updated successfully",
		"limit_type": limitType,
		"requests":   req.Requests,
		"window":     req.Window,
		"burst":      req.Burst,
	})
}

// ResetRateLimit resets rate limiting for a specific key
func (rlh *RateLimitAdminHandler) ResetRateLimit(c *gin.Context) {
	var req struct {
		Type  string `json:"type" binding:"required"`  // "ip", "user", "endpoint", "global"
		Value string `json:"value" binding:"required"` // IP address, user ID, endpoint, etc.
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	// Construct rate limit key
	var key string
	switch req.Type {
	case "ip":
		key = "rate_limit:ip:default:" + req.Value
	case "user":
		key = "rate_limit:user:default:" + req.Value
	case "endpoint":
		key = "rate_limit:endpoint:" + req.Value
	case "global":
		key = "rate_limit:global:" + req.Value
	default:
		ErrorResponse(c, http.StatusBadRequest, "Invalid reset type", nil)
		return
	}
	
	// Reset the rate limit
	err := rlh.securityMiddleware.ResetRateLimit(c.Request.Context(), key)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to reset rate limit", err)
		return
	}
	
	SuccessResponse(c, gin.H{
		"message": "Rate limit reset successfully",
		"type":    req.Type,
		"value":   req.Value,
		"key":     key,
	})
}

// GetRateLimitStats returns rate limiting statistics
func (rlh *RateLimitAdminHandler) GetRateLimitStats(c *gin.Context) {
	// This would typically query Redis for current rate limit usage
	// For now, return basic information
	
	stats := gin.H{
		"enabled": true,
		"types": gin.H{
			"default": gin.H{"requests": 100, "window": "1m", "burst": 10},
			"auth":    gin.H{"requests": 5, "window": "1m", "burst": 2},
			"api":     gin.H{"requests": 1000, "window": "1h", "burst": 50},
			"scan":    gin.H{"requests": 10, "window": "1m", "burst": 3},
			"upload":  gin.H{"requests": 5, "window": "1m", "burst": 1},
		},
		"active_limits": 0, // Would be calculated from Redis
		"blocked_requests": 0, // Would be tracked in metrics
	}
	
	SuccessResponse(c, stats)
}

// TestRateLimit allows testing rate limiting for a specific endpoint
func (rlh *RateLimitAdminHandler) TestRateLimit(c *gin.Context) {
	limitType := c.DefaultQuery("type", "default")
	testCount, _ := strconv.Atoi(c.DefaultQuery("count", "1"))
	
	if testCount > 100 {
		ErrorResponse(c, http.StatusBadRequest, "Test count cannot exceed 100", nil)
		return
	}
	
	results := make([]gin.H, testCount)
	
	for i := 0; i < testCount; i++ {
		// Simulate rate limit check
		limit, exists := rlh.securityMiddleware.GetRateLimit(limitType)
		if !exists {
			limit = middleware.RateLimit{Requests: 100, Window: 60000000000, Burst: 10} // 1 minute
		}
		
		results[i] = gin.H{
			"request_number": i + 1,
			"allowed":        i < limit.Requests, // Simplified simulation
			"limit":          limit.Requests,
			"remaining":      max(0, limit.Requests-i-1),
		}
	}
	
	SuccessResponse(c, gin.H{
		"test_type": limitType,
		"count":     testCount,
		"results":   results,
	})
}

// Helper functions

func parseDuration(s string) (time.Duration, error) {
	// Simple duration parsing - extend as needed
	switch s {
	case "1s", "second":
		return time.Second, nil
	case "1m", "minute":
		return time.Minute, nil
	case "1h", "hour":
		return time.Hour, nil
	case "1d", "day":
		return 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}