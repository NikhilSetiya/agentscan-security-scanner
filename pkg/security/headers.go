package security

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig holds configuration for security headers
type SecurityHeadersConfig struct {
	// Content Security Policy
	CSPDirectives map[string][]string
	
	// HSTS configuration
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	HSTSPreload           bool
	
	// CORS configuration
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
	
	// Feature Policy / Permissions Policy
	PermissionsPolicy map[string][]string
	
	// Additional security headers
	ReferrerPolicy        string
	XFrameOptions         string
	XContentTypeOptions   bool
	XDNSPrefetchControl   bool
	XDownloadOptions      bool
	XPermittedCrossDomain string
}

// DefaultSecurityHeadersConfig returns a secure default configuration
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		CSPDirectives: map[string][]string{
			"default-src": {"'self'"},
			"script-src":  {"'self'", "'unsafe-inline'", "https://cdn.jsdelivr.net"},
			"style-src":   {"'self'", "'unsafe-inline'", "https://fonts.googleapis.com"},
			"font-src":    {"'self'", "https://fonts.gstatic.com"},
			"img-src":     {"'self'", "data:", "https:"},
			"connect-src": {"'self'", "wss:", "https:"},
			"media-src":   {"'none'"},
			"object-src":  {"'none'"},
			"frame-src":   {"'none'"},
			"base-uri":    {"'self'"},
			"form-action": {"'self'"},
		},
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"https://*.agentscan.dev",
		},
		AllowedMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowedHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization",
			"X-Requested-With", "X-Request-ID", "X-Correlation-ID",
		},
		ExposedHeaders: []string{
			"X-Request-ID", "X-Correlation-ID", "X-RateLimit-Remaining",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		PermissionsPolicy: map[string][]string{
			"camera":                 {"'none'"},
			"microphone":             {"'none'"},
			"geolocation":            {"'none'"},
			"payment":                {"'none'"},
			"usb":                    {"'none'"},
			"magnetometer":           {"'none'"},
			"gyroscope":              {"'none'"},
			"accelerometer":          {"'none'"},
			"ambient-light-sensor":   {"'none'"},
			"autoplay":               {"'none'"},
			"encrypted-media":        {"'none'"},
			"fullscreen":             {"'self'"},
			"picture-in-picture":     {"'none'"},
			"screen-wake-lock":       {"'none'"},
			"web-share":              {"'self'"},
		},
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		XFrameOptions:         "DENY",
		XContentTypeOptions:   true,
		XDNSPrefetchControl:   false,
		XDownloadOptions:      true,
		XPermittedCrossDomain: "none",
	}
}

// SecurityHeadersMiddleware returns a Gin middleware that sets security headers
func SecurityHeadersMiddleware(config SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy
		if len(config.CSPDirectives) > 0 {
			csp := buildCSP(config.CSPDirectives)
			c.Header("Content-Security-Policy", csp)
		}

		// HTTP Strict Transport Security
		if config.HSTSMaxAge > 0 {
			hsts := buildHSTS(config.HSTSMaxAge, config.HSTSIncludeSubdomains, config.HSTSPreload)
			c.Header("Strict-Transport-Security", hsts)
		}

		// Permissions Policy
		if len(config.PermissionsPolicy) > 0 {
			pp := buildPermissionsPolicy(config.PermissionsPolicy)
			c.Header("Permissions-Policy", pp)
		}

		// Referrer Policy
		if config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", config.ReferrerPolicy)
		}

		// X-Frame-Options
		if config.XFrameOptions != "" {
			c.Header("X-Frame-Options", config.XFrameOptions)
		}

		// X-Content-Type-Options
		if config.XContentTypeOptions {
			c.Header("X-Content-Type-Options", "nosniff")
		}

		// X-DNS-Prefetch-Control
		if config.XDNSPrefetchControl {
			c.Header("X-DNS-Prefetch-Control", "on")
		} else {
			c.Header("X-DNS-Prefetch-Control", "off")
		}

		// X-Download-Options
		if config.XDownloadOptions {
			c.Header("X-Download-Options", "noopen")
		}

		// X-Permitted-Cross-Domain-Policies
		if config.XPermittedCrossDomain != "" {
			c.Header("X-Permitted-Cross-Domain-Policies", config.XPermittedCrossDomain)
		}

		// Additional security headers
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Robots-Tag", "noindex, nofollow, nosnippet, noarchive")
		c.Header("Server", "AgentScan")

		c.Next()
	}
}

// CORSMiddleware returns a CORS middleware with the given configuration
func CORSMiddleware(config SecurityHeadersConfig) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowOrigins:     config.AllowedOrigins,
		AllowMethods:     config.AllowedMethods,
		AllowHeaders:     config.AllowedHeaders,
		ExposeHeaders:    config.ExposedHeaders,
		AllowCredentials: config.AllowCredentials,
		MaxAge:           config.MaxAge,
	}

	// Custom origin validation for wildcard domains
	if containsWildcard(config.AllowedOrigins) {
		corsConfig.AllowOriginFunc = func(origin string) bool {
			return isOriginAllowed(origin, config.AllowedOrigins)
		}
		corsConfig.AllowOrigins = nil // Clear origins when using AllowOriginFunc
	}

	return cors.New(corsConfig)
}

// buildCSP constructs a Content Security Policy header value
func buildCSP(directives map[string][]string) string {
	var parts []string
	for directive, sources := range directives {
		if len(sources) > 0 {
			parts = append(parts, directive+" "+strings.Join(sources, " "))
		}
	}
	return strings.Join(parts, "; ")
}

// buildHSTS constructs an HSTS header value
func buildHSTS(maxAge int, includeSubdomains, preload bool) string {
	hsts := fmt.Sprintf("max-age=%d", maxAge)
	if includeSubdomains {
		hsts += "; includeSubDomains"
	}
	if preload {
		hsts += "; preload"
	}
	return hsts
}

// buildPermissionsPolicy constructs a Permissions Policy header value
func buildPermissionsPolicy(policies map[string][]string) string {
	var parts []string
	for feature, allowlist := range policies {
		if len(allowlist) > 0 {
			parts = append(parts, feature+"=("+strings.Join(allowlist, " ")+")")
		}
	}
	return strings.Join(parts, ", ")
}

// containsWildcard checks if any origin contains a wildcard
func containsWildcard(origins []string) bool {
	for _, origin := range origins {
		if strings.Contains(origin, "*") {
			return true
		}
	}
	return false
}

// isOriginAllowed checks if an origin is allowed based on patterns
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if matchOrigin(origin, allowed) {
			return true
		}
	}
	return false
}

// matchOrigin checks if an origin matches a pattern (supports wildcards)
func matchOrigin(origin, pattern string) bool {
	if pattern == "*" {
		return true
	}
	
	if !strings.Contains(pattern, "*") {
		return origin == pattern
	}
	
	// Handle subdomain wildcards like https://*.example.com
	if strings.HasPrefix(pattern, "https://*.") {
		domain := pattern[10:] // Remove "https://*."
		return strings.HasSuffix(origin, "."+domain) || origin == "https://"+domain
	}
	
	if strings.HasPrefix(pattern, "http://*.") {
		domain := pattern[9:] // Remove "http://*."
		return strings.HasSuffix(origin, "."+domain) || origin == "http://"+domain
	}
	
	return false
}

// SecurityMiddleware combines all security middlewares
func SecurityMiddleware(config SecurityHeadersConfig) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		CORSMiddleware(config),
		SecurityHeadersMiddleware(config),
		RequestSizeMiddleware(10 << 20), // 10MB limit
		RequestTimeoutMiddleware(30 * time.Second),
	}
}

// RequestSizeMiddleware limits the size of request bodies
func RequestSizeMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
				"max_size": maxSize,
			})
			c.Abort()
			return
		}
		
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// RequestTimeoutMiddleware adds a timeout to requests
func RequestTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		
		// Replace the request context
		c.Request = c.Request.WithContext(ctx)
		
		// Channel to signal completion
		done := make(chan struct{})
		
		go func() {
			defer close(done)
			c.Next()
		}()
		
		select {
		case <-done:
			// Request completed normally
			return
		case <-ctx.Done():
			// Request timed out
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":   "Request timeout",
				"timeout": timeout.String(),
			})
			c.Abort()
			return
		}
	}
}