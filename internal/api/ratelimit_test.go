package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestInMemoryRateLimiter tests the in-memory rate limiter
func TestInMemoryRateLimiter(t *testing.T) {
	config := &RateLimitConfig{
		IPRPM:           10,
		IPBurst:         5,
		WindowSize:      time.Minute,
		CleanupInterval: time.Minute,
		Enabled:         true,
	}
	
	limiter := NewInMemoryRateLimiter(config)
	key := "test-key"
	
	// Test initial requests (should be allowed)
	for i := 0; i < 5; i++ {
		allowed, info, err := limiter.Allow(key)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
		if info.Remaining != 4-i {
			t.Fatalf("Expected remaining %d, got %d", 4-i, info.Remaining)
		}
	}
	
	// Test rate limit exceeded
	allowed, info, err := limiter.Allow(key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("Request should be rate limited")
	}
	if info.Remaining != 0 {
		t.Fatalf("Expected remaining 0, got %d", info.Remaining)
	}
	if info.RetryAfter <= 0 {
		t.Fatal("RetryAfter should be positive")
	}
}

// TestRateLimitMiddleware tests the rate limiting middleware
func TestRateLimitMiddleware(t *testing.T) {
	config := &RateLimitConfig{
		IPRPM:           2,
		IPBurst:         2,
		WindowSize:      time.Minute,
		CleanupInterval: time.Minute,
		Enabled:         true,
		Whitelist:       []string{},
		Blacklist:       []string{},
		EndpointLimits:  make(map[string]EndpointLimit),
	}
	
	limiter := NewInMemoryRateLimiter(config)
	middleware := NewRateLimitMiddleware(limiter, config)
	
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	handler := middleware.Handler(testHandler)
	
	// Test allowed requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Fatalf("Request %d should be allowed, got status %d", i+1, w.Code)
		}
		
		// Check rate limit headers
		if w.Header().Get("X-RateLimit-Ip-Limit") == "" {
			t.Fatal("Rate limit headers should be present")
		}
	}
	
	// Test rate limited request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Request should be rate limited, got status %d", w.Code)
	}
	
	// Check retry-after header
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header should be present")
	}
}

// TestWhitelistBlacklist tests IP whitelist and blacklist functionality
func TestWhitelistBlacklist(t *testing.T) {
	config := &RateLimitConfig{
		IPRPM:           1,
		IPBurst:         1,
		WindowSize:      time.Minute,
		CleanupInterval: time.Minute,
		Enabled:         true,
		Whitelist:       []string{"192.168.1.100"},
		Blacklist:       []string{"192.168.1.200"},
		EndpointLimits:  make(map[string]EndpointLimit),
	}
	
	limiter := NewInMemoryRateLimiter(config)
	middleware := NewRateLimitMiddleware(limiter, config)
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	handler := middleware.Handler(testHandler)
	
	// Test whitelisted IP (should always be allowed)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Fatalf("Whitelisted IP should always be allowed, got status %d", w.Code)
		}
	}
	
	// Test blacklisted IP (should be forbidden)
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.200:12345"
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusForbidden {
		t.Fatalf("Blacklisted IP should be forbidden, got status %d", w.Code)
	}
}

// TestEndpointSpecificLimits tests endpoint-specific rate limiting
func TestEndpointSpecificLimits(t *testing.T) {
	config := &RateLimitConfig{
		IPRPM:           100,
		IPBurst:         50,
		WindowSize:      time.Minute,
		CleanupInterval: time.Minute,
		Enabled:         true,
		Whitelist:       []string{},
		Blacklist:       []string{},
		EndpointLimits: map[string]EndpointLimit{
			"POST /api/v1/scans": {RPM: 2, Burst: 1},
		},
	}
	
	limiter := NewInMemoryRateLimiter(config)
	middleware := NewRateLimitMiddleware(limiter, config)
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	handler := middleware.Handler(testHandler)
	
	// Test endpoint-specific limit
	req := httptest.NewRequest("POST", "/api/v1/scans", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Fatalf("First request should be allowed, got status %d", w.Code)
	}
	
	// Second request should be rate limited due to endpoint-specific limit
	req2 := httptest.NewRequest("POST", "/api/v1/scans", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	
	handler.ServeHTTP(w2, req2)
	
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("Second request should be rate limited, got status %d", w2.Code)
	}
}

// TestRateLimitManager tests the rate limit manager
func TestRateLimitManager(t *testing.T) {
	manager := NewRateLimitManager(nil) // Use in-memory limiter
	
	// Test getting configuration
	config := manager.GetConfig()
	if config == nil {
		t.Fatal("Config should not be nil")
	}
	
	// Test updating configuration
	newConfig := DefaultRateLimitConfig()
	newConfig.IPRPM = 50
	
	err := manager.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}
	
	updatedConfig := manager.GetConfig()
	if updatedConfig.IPRPM != 50 {
		t.Fatalf("Expected IPRPM 50, got %d", updatedConfig.IPRPM)
	}
	
	// Test invalid configuration
	invalidConfig := &RateLimitConfig{
		IPRPM: -1, // Invalid
	}
	
	err = manager.UpdateConfig(invalidConfig)
	if err == nil {
		t.Fatal("Should reject invalid configuration")
	}
	
	// Test whitelist/blacklist management
	manager.AddToWhitelist("192.168.1.100")
	config = manager.GetConfig()
	found := false
	for _, ip := range config.Whitelist {
		if ip == "192.168.1.100" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("IP should be in whitelist")
	}
	
	manager.RemoveFromWhitelist("192.168.1.100")
	config = manager.GetConfig()
	for _, ip := range config.Whitelist {
		if ip == "192.168.1.100" {
			t.Fatal("IP should be removed from whitelist")
		}
	}
}

// TestTokenBucketRefill tests token bucket refill logic
func TestTokenBucketRefill(t *testing.T) {
	config := &RateLimitConfig{
		IPRPM:           60, // 1 token per second
		IPBurst:         5,
		WindowSize:      time.Minute,
		CleanupInterval: time.Minute,
		Enabled:         true,
	}
	
	limiter := NewInMemoryRateLimiter(config)
	key := "refill-test"
	
	// Exhaust the bucket
	for i := 0; i < 5; i++ {
		allowed, _, err := limiter.Allow(key)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}
	
	// Next request should be denied
	allowed, _, err := limiter.Allow(key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("Request should be denied")
	}
	
	// Wait for refill (simulate time passing)
	// In a real test, you might want to use a mock clock
	time.Sleep(2 * time.Second)
	
	// Should have tokens available now
	allowed, info, err := limiter.Allow(key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("Request should be allowed after refill")
	}
	if info.Remaining < 1 {
		t.Fatal("Should have tokens after refill")
	}
}

// BenchmarkRateLimiter benchmarks the rate limiter performance
func BenchmarkRateLimiter(b *testing.B) {
	config := DefaultRateLimitConfig()
	limiter := NewInMemoryRateLimiter(config)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench-key-" + string(rune(i%100))
			limiter.Allow(key)
			i++
		}
	})
}