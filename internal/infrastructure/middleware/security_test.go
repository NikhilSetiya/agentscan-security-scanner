package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/shared/testing"
)

func TestSecurityMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("security_headers", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		
		securityConfig := &config.SecurityConfig{
			Headers: config.SecurityHeaders{
				Enabled: true,
				HSTS: config.HSTSConfig{
					Enabled:           true,
					MaxAge:            31536000,
					IncludeSubdomains: true,
					Preload:           true,
				},
				CSP: config.CSPConfig{
					Enabled:   true,
					Policy:    "default-src 'self'",
					ReportURI: "/csp-report",
				},
			},
		}

		router.Use(SecurityHeaders(securityConfig))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "max-age=31536000; includeSubDomains; preload", w.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	})

	t.Run("cors_middleware", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		
		corsConfig := &config.CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			MaxAge:         3600,
		}

		router.Use(CORS(corsConfig))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// Test preflight request
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 204, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Equal(t, "3600", w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("cors_invalid_origin", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		
		corsConfig := &config.CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com"},
		}

		router.Use(CORS(corsConfig))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("rate_limiting", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		
		rateLimitConfig := &config.RateLimitConfig{
			Enabled:  true,
			Requests: 2,
			Window:   time.Second,
		}

		router.Use(RateLimit(rateLimitConfig))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// First request should succeed
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "127.0.0.1:12345"
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, 200, w1.Code)

		// Second request should succeed
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "127.0.0.1:12345"
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, 200, w2.Code)

		// Third request should be rate limited
		req3 := httptest.NewRequest("GET", "/test", nil)
		req3.RemoteAddr = "127.0.0.1:12345"
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)
		assert.Equal(t, 429, w3.Code)
	})

	t.Run("request_size_limit", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		
		router.Use(RequestSizeLimit(100)) // 100 bytes limit
		router.POST("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// Small request should succeed
		smallBody := strings.NewReader(`{"test": "data"}`)
		req1 := httptest.NewRequest("POST", "/test", smallBody)
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, 200, w1.Code)

		// Large request should be rejected
		largeBody := strings.NewReader(strings.Repeat("a", 200))
		req2 := httptest.NewRequest("POST", "/test", largeBody)
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, 413, w2.Code)
	})

	t.Run("timeout_middleware", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		
		router.Use(Timeout(100 * time.Millisecond))
		router.GET("/fast", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "fast"})
		})
		router.GET("/slow", func(c *gin.Context) {
			time.Sleep(200 * time.Millisecond)
			c.JSON(200, gin.H{"message": "slow"})
		})

		// Fast request should succeed
		req1 := httptest.NewRequest("GET", "/fast", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, 200, w1.Code)

		// Slow request should timeout
		req2 := httptest.NewRequest("GET", "/slow", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, 408, w2.Code)
	})
}

func TestJWTMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtConfig := &config.JWTConfig{
		Secret:    "test_secret_32_characters_long_enough",
		Algorithm: "HS256",
		Issuer:    "agentscan-test",
		Audience:  "agentscan-api",
		TTL:       time.Hour,
	}

	t.Run("valid_jwt_token", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		router.Use(JWTAuth(jwtConfig))
		router.GET("/protected", func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			assert.True(t, exists)
			c.JSON(200, gin.H{"user_id": userID})
		})

		// Generate valid token
		token, err := GenerateJWT("test-user-123", jwtConfig)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "test-user-123", response["user_id"])
	})

	t.Run("missing_authorization_header", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		router.Use(JWTAuth(jwtConfig))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("invalid_token_format", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		router.Use(JWTAuth(jwtConfig))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Invalid token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("expired_token", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		// Create config with very short TTL
		shortTTLConfig := &config.JWTConfig{
			Secret:    "test_secret_32_characters_long_enough",
			Algorithm: "HS256",
			Issuer:    "agentscan-test",
			Audience:  "agentscan-api",
			TTL:       time.Millisecond, // Very short TTL
		}

		router := suite.SetupGinRouter()
		router.Use(JWTAuth(jwtConfig))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// Generate token that will expire quickly
		token, err := GenerateJWT("test-user-123", shortTTLConfig)
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})
}

func TestCSPReporting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("csp_violation_report", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		
		var reportReceived bool
		var reportData map[string]interface{}

		router.POST("/csp-report", func(c *gin.Context) {
			reportReceived = true
			c.BindJSON(&reportData)
			c.Status(204)
		})

		cspReport := map[string]interface{}{
			"csp-report": map[string]interface{}{
				"document-uri":        "https://example.com/page",
				"referrer":           "",
				"violated-directive": "script-src 'self'",
				"effective-directive": "script-src",
				"original-policy":    "default-src 'self'; script-src 'self'",
				"blocked-uri":        "https://malicious.com/script.js",
				"status-code":        200,
			},
		}

		jsonData, err := json.Marshal(cspReport)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/csp-report", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/csp-report")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 204, w.Code)
		assert.True(t, reportReceived)
		assert.NotNil(t, reportData)
	})
}

func TestSecurityAuditLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("security_event_logging", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		logger := suite.Logger()
		router := suite.SetupGinRouter()
		
		auditConfig := &config.AuditConfig{
			Enabled: true,
			Events: []string{
				"authentication_failure",
				"rate_limit_exceeded",
				"cors_violation",
			},
		}

		router.Use(SecurityAuditLog(auditConfig, logger))
		router.GET("/test", func(c *gin.Context) {
			// Simulate security event
			LogSecurityEvent(c, "authentication_failure", map[string]interface{}{
				"user_id": "test-user",
				"ip":      c.ClientIP(),
				"reason":  "invalid_token",
			})
			c.JSON(401, gin.H{"error": "unauthorized"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
		
		// Check if security event was logged
		testLogger := logger.(*testing.TestLogger)
		entries := testLogger.GetEntriesWithMessage("Security event")
		assert.NotEmpty(t, entries)
		
		securityEntry := entries[0]
		assert.Equal(t, "authentication_failure", securityEntry.Fields["event_type"])
		assert.Equal(t, "test-user", securityEntry.Fields["user_id"])
		assert.Contains(t, securityEntry.Fields["ip"], "192.168.1.100")
	})
}

func TestInputValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("xss_protection", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		router.Use(XSSProtection())
		router.POST("/test", func(c *gin.Context) {
			var input map[string]interface{}
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, input)
		})

		maliciousInput := map[string]interface{}{
			"name":    "<script>alert('xss')</script>",
			"comment": "Normal text with <img src=x onerror=alert('xss')>",
		}

		jsonData, err := json.Marshal(maliciousInput)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		// Check that XSS content was sanitized
		assert.NotContains(t, response["name"], "<script>")
		assert.NotContains(t, response["comment"], "onerror=")
	})

	t.Run("sql_injection_protection", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		router.Use(SQLInjectionProtection())
		router.GET("/users", func(c *gin.Context) {
			userID := c.Query("id")
			// Simulate database query parameter
			c.JSON(200, gin.H{"user_id": userID})
		})

		// Test with SQL injection attempt
		req := httptest.NewRequest("GET", "/users?id=1' OR '1'='1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "potentially malicious")
	})
}

func TestSecurityIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("full_security_stack", func(t *testing.T) {
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		router := suite.SetupGinRouter()
		
		// Full security configuration
		securityConfig := &config.SecurityConfig{
			Headers: config.SecurityHeaders{
				Enabled: true,
				HSTS: config.HSTSConfig{
					Enabled: true,
					MaxAge:  31536000,
				},
				CSP: config.CSPConfig{
					Enabled: true,
					Policy:  "default-src 'self'",
				},
			},
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"https://app.example.com"},
			},
			JWT: config.JWTConfig{
				Secret:    "test_secret_32_characters_long_enough",
				Algorithm: "HS256",
			},
			RateLimit: config.RateLimitConfig{
				Enabled:  true,
				Requests: 10,
				Window:   time.Minute,
			},
		}

		// Apply all security middleware
		router.Use(SecurityHeaders(&securityConfig.Headers))
		router.Use(CORS(&securityConfig.CORS))
		router.Use(RateLimit(&securityConfig.RateLimit))
		router.Use(RequestSizeLimit(1024))
		router.Use(XSSProtection())
		router.Use(SQLInjectionProtection())
		
		// Protected route
		router.GET("/protected", JWTAuth(&securityConfig.JWT), func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "success"})
		})

		// Public route
		router.GET("/public", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "public"})
		})

		// Test public route with security headers
		req1 := httptest.NewRequest("GET", "/public", nil)
		req1.Header.Set("Origin", "https://app.example.com")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		assert.Equal(t, 200, w1.Code)
		assert.NotEmpty(t, w1.Header().Get("Strict-Transport-Security"))
		assert.NotEmpty(t, w1.Header().Get("Content-Security-Policy"))
		assert.Equal(t, "https://app.example.com", w1.Header().Get("Access-Control-Allow-Origin"))

		// Test protected route with valid JWT
		token, err := GenerateJWT("test-user", &securityConfig.JWT)
		require.NoError(t, err)

		req2 := httptest.NewRequest("GET", "/protected", nil)
		req2.Header.Set("Authorization", "Bearer "+token)
		req2.Header.Set("Origin", "https://app.example.com")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, 200, w2.Code)
	})
}

// Benchmark tests

func BenchmarkSecurityHeaders(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	securityConfig := &config.SecurityConfig{
		Headers: config.SecurityHeaders{
			Enabled: true,
			HSTS: config.HSTSConfig{
				Enabled: true,
				MaxAge:  31536000,
			},
		},
	}

	router.Use(SecurityHeaders(&securityConfig.Headers))
	router.GET("/test", func(c *gin.Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkJWTValidation(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	jwtConfig := &config.JWTConfig{
		Secret:    "test_secret_32_characters_long_enough",
		Algorithm: "HS256",
	}

	token, _ := GenerateJWT("test-user", jwtConfig)
	
	router.Use(JWTAuth(jwtConfig))
	router.GET("/test", func(c *gin.Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}