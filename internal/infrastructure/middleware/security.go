package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/shared/logging"
)

// SecurityMiddleware applies comprehensive security headers and policies
func SecurityMiddleware(securityConfig *config.SecurityConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Apply security headers
		applySecurityHeaders(c, securityConfig)
		
		// Apply CORS if enabled
		if securityConfig.CORS.Enabled {
			applyCORS(c, &securityConfig.CORS)
		}
		
		// Apply Content Security Policy
		if securityConfig.CSP.Enabled {
			applyCSP(c, &securityConfig.CSP)
		}
		
		// Check for blocked user agents
		if isBlockedUserAgent(c, securityConfig.API.BlockedUserAgents) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "User agent not allowed",
			})
			return
		}
		
		// Enforce HTTPS in production
		if config.IsProductionEnvironment() && securityConfig.HTTPS.Enabled {
			if !isHTTPS(c) {
				if securityConfig.HTTPS.RedirectHTTP {
					redirectToHTTPS(c)
					return
				} else {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error": "HTTPS required",
					})
					return
				}
			}
		}
		
		c.Next()
	})
}

// applySecurityHeaders applies standard security headers
func applySecurityHeaders(c *gin.Context, securityConfig *config.SecurityConfig) {
	headers := securityConfig.Headers
	
	// X-Frame-Options
	if headers.XFrameOptions != "" {
		c.Header("X-Frame-Options", headers.XFrameOptions)
	}
	
	// X-Content-Type-Options
	if headers.XContentTypeOptions != "" {
		c.Header("X-Content-Type-Options", headers.XContentTypeOptions)
	}
	
	// X-XSS-Protection
	if headers.XSSProtection != "" {
		c.Header("X-XSS-Protection", headers.XSSProtection)
	}
	
	// Referrer-Policy
	if headers.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", headers.ReferrerPolicy)
	}
	
	// Permissions-Policy
	if headers.PermissionsPolicy != "" {
		c.Header("Permissions-Policy", headers.PermissionsPolicy)
	}
	
	// Cross-Origin-Embedder-Policy
	if headers.CrossOriginEmbedderPolicy != "" {
		c.Header("Cross-Origin-Embedder-Policy", headers.CrossOriginEmbedderPolicy)
	}
	
	// Cross-Origin-Opener-Policy
	if headers.CrossOriginOpenerPolicy != "" {
		c.Header("Cross-Origin-Opener-Policy", headers.CrossOriginOpenerPolicy)
	}
	
	// Cross-Origin-Resource-Policy
	if headers.CrossOriginResourcePolicy != "" {
		c.Header("Cross-Origin-Resource-Policy", headers.CrossOriginResourcePolicy)
	}
	
	// HSTS (HTTP Strict Transport Security)
	if securityConfig.HTTPS.Enabled && isHTTPS(c) {
		hstsValue := fmt.Sprintf("max-age=%d", securityConfig.HTTPS.HSTSMaxAge)
		if securityConfig.HTTPS.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		if securityConfig.HTTPS.HSTSPreload {
			hstsValue += "; preload"
		}
		c.Header("Strict-Transport-Security", hstsValue)
	}
	
	// Remove server information
	c.Header("Server", "")
	c.Header("X-Powered-By", "")
}

// applyCORS applies CORS headers
func applyCORS(c *gin.Context, corsConfig *config.CORSConfig) {
	origin := c.Request.Header.Get("Origin")
	
	// Check if origin is allowed
	if isAllowedOrigin(origin, corsConfig.AllowedOrigins) {
		c.Header("Access-Control-Allow-Origin", origin)
	}
	
	// Set other CORS headers
	if len(corsConfig.AllowedMethods) > 0 {
		c.Header("Access-Control-Allow-Methods", strings.Join(corsConfig.AllowedMethods, ", "))
	}
	
	if len(corsConfig.AllowedHeaders) > 0 {
		c.Header("Access-Control-Allow-Headers", strings.Join(corsConfig.AllowedHeaders, ", "))
	}
	
	if len(corsConfig.ExposedHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(corsConfig.ExposedHeaders, ", "))
	}
	
	if corsConfig.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	
	if corsConfig.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", corsConfig.MaxAge))
	}
	
	// Handle preflight requests
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
}

// applyCSP applies Content Security Policy headers
func applyCSP(c *gin.Context, cspConfig *config.CSPConfig) {
	cspHeader := cspConfig.GetCSPHeader()
	if cspHeader != "" {
		headerName := "Content-Security-Policy"
		if cspConfig.ReportOnly {
			headerName = "Content-Security-Policy-Report-Only"
		}
		c.Header(headerName, cspHeader)
	}
}

// isAllowedOrigin checks if an origin is in the allowed list
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		
		// Support wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, "."+domain) || origin == domain {
				return true
			}
		}
	}
	
	return false
}

// isBlockedUserAgent checks if a user agent is blocked
func isBlockedUserAgent(c *gin.Context, blockedAgents []string) bool {
	userAgent := strings.ToLower(c.Request.Header.Get("User-Agent"))
	
	for _, blocked := range blockedAgents {
		if strings.Contains(userAgent, strings.ToLower(blocked)) {
			return true
		}
	}
	
	return false
}

// isHTTPS checks if the request is using HTTPS
func isHTTPS(c *gin.Context) bool {
	// Check X-Forwarded-Proto header (common in load balancers)
	if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto == "https" {
		return true
	}
	
	// Check X-Forwarded-SSL header
	if ssl := c.Request.Header.Get("X-Forwarded-SSL"); ssl == "on" {
		return true
	}
	
	// Check direct TLS connection
	return c.Request.TLS != nil
}

// redirectToHTTPS redirects HTTP requests to HTTPS
func redirectToHTTPS(c *gin.Context) {
	httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
	c.Redirect(http.StatusMovedPermanently, httpsURL)
}

// RequestSizeMiddleware limits request body size
func RequestSizeMiddleware(maxSize int64) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
				"max_size": maxSize,
			})
			return
		}
		
		// Limit the request body reader
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		
		c.Next()
	})
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		
		// Replace the request context
		c.Request = c.Request.WithContext(ctx)
		
		// Channel to signal completion
		finished := make(chan struct{})
		
		go func() {
			defer close(finished)
			c.Next()
		}()
		
		select {
		case <-finished:
			// Request completed normally
		case <-ctx.Done():
			// Request timed out
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
				"error": "Request timeout",
				"timeout": timeout.String(),
			})
		}
	})
}

// CSPReportHandler handles CSP violation reports
func CSPReportHandler() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		var report map[string]interface{}
		
		if err := c.ShouldBindJSON(&report); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report format"})
			return
		}
		
		// Log CSP violation
		logger := logging.GetLogger()
		logger.Warn("CSP Violation Report", 
			"report", report,
			"user_agent", c.Request.Header.Get("User-Agent"),
			"ip", c.ClientIP(),
		)
		
		c.Status(http.StatusNoContent)
	})
}

// SecurityAuditMiddleware logs security-relevant events
func SecurityAuditMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		// Log security-relevant requests
		if isSecurityRelevant(c) {
			logger := logging.GetLogger()
			logger.Info("Security Audit Log",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"status", c.Writer.Status(),
				"ip", c.ClientIP(),
				"user_agent", c.Request.Header.Get("User-Agent"),
				"duration", time.Since(start),
				"user_id", getUserID(c),
			)
		}
	})
}

// isSecurityRelevant checks if a request is security-relevant
func isSecurityRelevant(c *gin.Context) bool {
	path := c.Request.URL.Path
	method := c.Request.Method
	status := c.Writer.Status()
	
	// Log authentication endpoints
	if strings.Contains(path, "/auth/") {
		return true
	}
	
	// Log admin endpoints
	if strings.Contains(path, "/admin/") {
		return true
	}
	
	// Log failed requests
	if status >= 400 {
		return true
	}
	
	// Log sensitive operations
	if method == "DELETE" || method == "PUT" || method == "PATCH" {
		return true
	}
	
	return false
}

// getUserID extracts user ID from context (if available)
func getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}