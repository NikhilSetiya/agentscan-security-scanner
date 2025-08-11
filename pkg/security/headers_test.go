package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultSecurityHeadersConfig()
	
	router := gin.New()
	router.Use(SecurityHeadersMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check security headers
	headers := w.Header()
	
	// CSP header
	csp := headers.Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "object-src 'none'")

	// HSTS header
	hsts := headers.Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.Contains(t, hsts, "preload")

	// Other security headers
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "off", headers.Get("X-DNS-Prefetch-Control"))
	assert.Equal(t, "noopen", headers.Get("X-Download-Options"))
	assert.Equal(t, "none", headers.Get("X-Permitted-Cross-Domain-Policies"))
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
	assert.Equal(t, "AgentScan", headers.Get("Server"))

	// Permissions Policy
	pp := headers.Get("Permissions-Policy")
	assert.Contains(t, pp, "camera=('none')")
	assert.Contains(t, pp, "microphone=('none')")
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultSecurityHeadersConfig()
	
	router := gin.New()
	router.Use(CORSMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	headers := w.Header()
	assert.Equal(t, "http://localhost:3000", headers.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, headers.Get("Access-Control-Allow-Methods"), "GET")
	assert.Equal(t, "true", headers.Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_UnallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultSecurityHeadersConfig()
	
	router := gin.New()
	router.Use(CORSMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Test with unallowed origin
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	headers := w.Header()
	assert.Empty(t, headers.Get("Access-Control-Allow-Origin"))
}

func TestMatchOrigin(t *testing.T) {
	tests := []struct {
		origin   string
		pattern  string
		expected bool
	}{
		{"https://example.com", "https://example.com", true},
		{"https://sub.example.com", "https://*.example.com", true},
		{"https://example.com", "https://*.example.com", true},
		{"https://evil.com", "https://*.example.com", false},
		{"http://localhost:3000", "http://localhost:3000", true},
		{"https://app.agentscan.dev", "https://*.agentscan.dev", true},
		{"https://agentscan.dev", "https://*.agentscan.dev", true},
		{"https://evil.agentscan.dev.evil.com", "https://*.agentscan.dev", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin+"_vs_"+tt.pattern, func(t *testing.T) {
			result := matchOrigin(tt.origin, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestSizeMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestSizeMiddleware(10)) // 10 bytes limit
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Test request within limit
	req := httptest.NewRequest("POST", "/test", strings.NewReader("small"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test request exceeding limit
	req = httptest.NewRequest("POST", "/test", strings.NewReader("this is a very long request body"))
	req.Header.Set("Content-Length", "35")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestRequestTimeoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestTimeoutMiddleware(100 * time.Millisecond))
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "slow"})
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusRequestTimeout, w.Code)
}

func TestBuildCSP(t *testing.T) {
	directives := map[string][]string{
		"default-src": {"'self'"},
		"script-src":  {"'self'", "https://cdn.example.com"},
		"style-src":   {"'self'", "'unsafe-inline'"},
	}

	csp := buildCSP(directives)
	
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self' https://cdn.example.com")
	assert.Contains(t, csp, "style-src 'self' 'unsafe-inline'")
}

func TestBuildHSTS(t *testing.T) {
	tests := []struct {
		maxAge            int
		includeSubdomains bool
		preload           bool
		expected          string
	}{
		{31536000, true, true, "max-age=31536000; includeSubDomains; preload"},
		{31536000, true, false, "max-age=31536000; includeSubDomains"},
		{31536000, false, false, "max-age=31536000"},
	}

	for _, tt := range tests {
		result := buildHSTS(tt.maxAge, tt.includeSubdomains, tt.preload)
		assert.Equal(t, tt.expected, result)
	}
}

func TestBuildPermissionsPolicy(t *testing.T) {
	policies := map[string][]string{
		"camera":     {"'none'"},
		"microphone": {"'none'"},
		"geolocation": {"'self'", "https://example.com"},
	}

	pp := buildPermissionsPolicy(policies)
	
	assert.Contains(t, pp, "camera=('none')")
	assert.Contains(t, pp, "microphone=('none')")
	assert.Contains(t, pp, "geolocation=('self' https://example.com)")
}