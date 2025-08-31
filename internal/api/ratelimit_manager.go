package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
)

// RateLimitManager manages rate limiting configuration and enforcement
type RateLimitManager struct {
	limiter    RateLimiter
	config     *RateLimitConfig
	middleware *RateLimitMiddleware
	mutex      sync.RWMutex
}

// NewRateLimitManager creates a new rate limit manager
func NewRateLimitManager(redis *queue.RedisClient) *RateLimitManager {
	config := DefaultRateLimitConfig()
	
	var limiter RateLimiter
	if redis != nil {
		limiter = NewRedisRateLimiter(config, redis)
	} else {
		limiter = NewInMemoryRateLimiter(config)
	}
	
	middleware := NewRateLimitMiddleware(limiter, config)
	
	return &RateLimitManager{
		limiter:    limiter,
		config:     config,
		middleware: middleware,
	}
}

// GetMiddleware returns the rate limiting middleware
func (rlm *RateLimitManager) GetMiddleware() *RateLimitMiddleware {
	return rlm.middleware
}

// UpdateConfig updates the rate limiting configuration
func (rlm *RateLimitManager) UpdateConfig(newConfig *RateLimitConfig) error {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	
	// Validate configuration
	if err := rlm.validateConfig(newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Update configuration
	rlm.config = newConfig
	rlm.middleware.config = newConfig
	
	return nil
}

// GetConfig returns the current rate limiting configuration
func (rlm *RateLimitManager) GetConfig() *RateLimitConfig {
	rlm.mutex.RLock()
	defer rlm.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	configCopy := *rlm.config
	return &configCopy
}

// GetStats returns rate limiting statistics
func (rlm *RateLimitManager) GetStats() (*RateLimitStats, error) {
	// This would typically query the underlying storage for statistics
	// For now, return basic stats
	return &RateLimitStats{
		Enabled:        rlm.config.Enabled,
		TotalRequests:  0, // Would be tracked in production
		BlockedRequests: 0, // Would be tracked in production
		ActiveKeys:     0, // Would be tracked in production
		LastReset:      time.Now(),
	}, nil
}

// ResetKey resets rate limiting for a specific key
func (rlm *RateLimitManager) ResetKey(key string) error {
	return rlm.limiter.Reset(key)
}

// ResetIP resets rate limiting for a specific IP
func (rlm *RateLimitManager) ResetIP(ip string) error {
	key := fmt.Sprintf("rate_limit:ip:%s", ip)
	return rlm.limiter.Reset(key)
}

// ResetUser resets rate limiting for a specific user
func (rlm *RateLimitManager) ResetUser(userID string) error {
	key := fmt.Sprintf("rate_limit:user:%s", userID)
	return rlm.limiter.Reset(key)
}

// AddToWhitelist adds an IP to the whitelist
func (rlm *RateLimitManager) AddToWhitelist(ip string) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	
	// Check if already in whitelist
	for _, whitelistedIP := range rlm.config.Whitelist {
		if whitelistedIP == ip {
			return
		}
	}
	
	rlm.config.Whitelist = append(rlm.config.Whitelist, ip)
}

// RemoveFromWhitelist removes an IP from the whitelist
func (rlm *RateLimitManager) RemoveFromWhitelist(ip string) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	
	for i, whitelistedIP := range rlm.config.Whitelist {
		if whitelistedIP == ip {
			rlm.config.Whitelist = append(rlm.config.Whitelist[:i], rlm.config.Whitelist[i+1:]...)
			return
		}
	}
}

// AddToBlacklist adds an IP to the blacklist
func (rlm *RateLimitManager) AddToBlacklist(ip string) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	
	// Check if already in blacklist
	for _, blacklistedIP := range rlm.config.Blacklist {
		if blacklistedIP == ip {
			return
		}
	}
	
	rlm.config.Blacklist = append(rlm.config.Blacklist, ip)
}

// RemoveFromBlacklist removes an IP from the blacklist
func (rlm *RateLimitManager) RemoveFromBlacklist(ip string) {
	rlm.mutex.Lock()
	defer rlm.mutex.Unlock()
	
	for i, blacklistedIP := range rlm.config.Blacklist {
		if blacklistedIP == ip {
			rlm.config.Blacklist = append(rlm.config.Blacklist[:i], rlm.config.Blacklist[i+1:]...)
			return
		}
	}
}

// validateConfig validates a rate limiting configuration
func (rlm *RateLimitManager) validateConfig(config *RateLimitConfig) error {
	if config.GlobalRPM < 0 {
		return fmt.Errorf("global RPM cannot be negative")
	}
	if config.IPRPM < 0 {
		return fmt.Errorf("IP RPM cannot be negative")
	}
	if config.UserRPM < 0 {
		return fmt.Errorf("user RPM cannot be negative")
	}
	if config.WindowSize <= 0 {
		return fmt.Errorf("window size must be positive")
	}
	if config.CleanupInterval <= 0 {
		return fmt.Errorf("cleanup interval must be positive")
	}
	
	// Validate endpoint limits
	for endpoint, limit := range config.EndpointLimits {
		if limit.RPM < 0 {
			return fmt.Errorf("endpoint %s RPM cannot be negative", endpoint)
		}
		if limit.Burst < 0 {
			return fmt.Errorf("endpoint %s burst cannot be negative", endpoint)
		}
	}
	
	return nil
}

// RateLimitStats represents rate limiting statistics
type RateLimitStats struct {
	Enabled         bool      `json:"enabled"`
	TotalRequests   int64     `json:"total_requests"`
	BlockedRequests int64     `json:"blocked_requests"`
	ActiveKeys      int       `json:"active_keys"`
	LastReset       time.Time `json:"last_reset"`
	BlockRate       float64   `json:"block_rate"`
}

// RateLimitHandler provides HTTP endpoints for rate limit management
type RateLimitHandler struct {
	manager *RateLimitManager
}

// NewRateLimitHandler creates a new rate limit handler
func NewRateLimitHandler(manager *RateLimitManager) *RateLimitHandler {
	return &RateLimitHandler{
		manager: manager,
	}
}

// ServeHTTP handles rate limit management requests
func (rlh *RateLimitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/admin/ratelimit/config":
		rlh.handleConfig(w, r)
	case "/admin/ratelimit/stats":
		rlh.handleStats(w, r)
	case "/admin/ratelimit/reset":
		rlh.handleReset(w, r)
	case "/admin/ratelimit/whitelist":
		rlh.handleWhitelist(w, r)
	case "/admin/ratelimit/blacklist":
		rlh.handleBlacklist(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleConfig handles configuration management
func (rlh *RateLimitHandler) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get current configuration
		config := rlh.manager.GetConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
		
	case http.MethodPut:
		// Update configuration
		var newConfig RateLimitConfig
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		if err := rlh.manager.UpdateConfig(&newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleStats handles statistics requests
func (rlh *RateLimitHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	stats, err := rlh.manager.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleReset handles reset requests
func (rlh *RateLimitHandler) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var resetReq struct {
		Type  string `json:"type"`  // "ip", "user", "key"
		Value string `json:"value"` // IP address, user ID, or key
	}
	
	if err := json.NewDecoder(r.Body).Decode(&resetReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	var err error
	switch resetReq.Type {
	case "ip":
		err = rlh.manager.ResetIP(resetReq.Value)
	case "user":
		err = rlh.manager.ResetUser(resetReq.Value)
	case "key":
		err = rlh.manager.ResetKey(resetReq.Value)
	default:
		http.Error(w, "Invalid reset type", http.StatusBadRequest)
		return
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reset"})
}

// handleWhitelist handles whitelist management
func (rlh *RateLimitHandler) handleWhitelist(w http.ResponseWriter, r *http.Request) {
	var ipReq struct {
		IP string `json:"ip"`
	}
	
	switch r.Method {
	case http.MethodPost:
		// Add to whitelist
		if err := json.NewDecoder(r.Body).Decode(&ipReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		rlh.manager.AddToWhitelist(ipReq.IP)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "added"})
		
	case http.MethodDelete:
		// Remove from whitelist
		if err := json.NewDecoder(r.Body).Decode(&ipReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		rlh.manager.RemoveFromWhitelist(ipReq.IP)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
		
	case http.MethodGet:
		// Get whitelist
		config := rlh.manager.GetConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{"whitelist": config.Whitelist})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBlacklist handles blacklist management
func (rlh *RateLimitHandler) handleBlacklist(w http.ResponseWriter, r *http.Request) {
	var ipReq struct {
		IP string `json:"ip"`
	}
	
	switch r.Method {
	case http.MethodPost:
		// Add to blacklist
		if err := json.NewDecoder(r.Body).Decode(&ipReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		rlh.manager.AddToBlacklist(ipReq.IP)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "added"})
		
	case http.MethodDelete:
		// Remove from blacklist
		if err := json.NewDecoder(r.Body).Decode(&ipReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		rlh.manager.RemoveFromBlacklist(ipReq.IP)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
		
	case http.MethodGet:
		// Get blacklist
		config := rlh.manager.GetConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{"blacklist": config.Blacklist})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}