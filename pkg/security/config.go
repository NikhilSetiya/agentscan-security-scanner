package security

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// SecurityConfig holds all security-related configuration
type SecurityConfig struct {
	// Encryption configuration
	EncryptionKey string `json:"encryption_key" env:"ENCRYPTION_KEY"`
	
	// TLS configuration
	TLSCertFile string `json:"tls_cert_file" env:"TLS_CERT_FILE"`
	TLSKeyFile  string `json:"tls_key_file" env:"TLS_KEY_FILE"`
	TLSMinVersion string `json:"tls_min_version" env:"TLS_MIN_VERSION"` // "1.2" or "1.3"
	
	// Security headers configuration
	Headers SecurityHeadersConfig `json:"headers"`
	
	// Rate limiting configuration
	RateLimit RateLimitConfig `json:"rate_limit"`
	
	// Audit logging configuration
	AuditLog AuditLogConfig `json:"audit_log"`
	
	// Session configuration
	Session SessionConfig `json:"session"`
	
	// CORS configuration (embedded in headers but also here for clarity)
	CORS CORSConfig `json:"cors"`
	
	// Security testing configuration
	SecurityTesting SecurityTestingConfig `json:"security_testing"`
}

// AuditLogConfig holds audit logging configuration
type AuditLogConfig struct {
	Enabled         bool          `json:"enabled" env:"AUDIT_LOG_ENABLED"`
	RetentionPeriod time.Duration `json:"retention_period" env:"AUDIT_LOG_RETENTION"`
	EncryptLogs     bool          `json:"encrypt_logs" env:"AUDIT_LOG_ENCRYPT"`
	LogLevel        string        `json:"log_level" env:"AUDIT_LOG_LEVEL"` // "info", "warn", "error"
}

// SessionConfig holds session management configuration
type SessionConfig struct {
	SecureCookies    bool          `json:"secure_cookies" env:"SESSION_SECURE_COOKIES"`
	HTTPOnlyCookies  bool          `json:"http_only_cookies" env:"SESSION_HTTP_ONLY"`
	SameSiteCookies  string        `json:"same_site_cookies" env:"SESSION_SAME_SITE"` // "strict", "lax", "none"
	SessionTimeout   time.Duration `json:"session_timeout" env:"SESSION_TIMEOUT"`
	IdleTimeout      time.Duration `json:"idle_timeout" env:"SESSION_IDLE_TIMEOUT"`
	CookieName       string        `json:"cookie_name" env:"SESSION_COOKIE_NAME"`
	CookieDomain     string        `json:"cookie_domain" env:"SESSION_COOKIE_DOMAIN"`
	CookiePath       string        `json:"cookie_path" env:"SESSION_COOKIE_PATH"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string      `json:"allowed_origins" env:"CORS_ALLOWED_ORIGINS"`
	AllowedMethods   []string      `json:"allowed_methods" env:"CORS_ALLOWED_METHODS"`
	AllowedHeaders   []string      `json:"allowed_headers" env:"CORS_ALLOWED_HEADERS"`
	ExposedHeaders   []string      `json:"exposed_headers" env:"CORS_EXPOSED_HEADERS"`
	AllowCredentials bool          `json:"allow_credentials" env:"CORS_ALLOW_CREDENTIALS"`
	MaxAge           time.Duration `json:"max_age" env:"CORS_MAX_AGE"`
}

// SecurityTestingConfig holds security testing configuration
type SecurityTestingConfig struct {
	Enabled           bool     `json:"enabled" env:"SECURITY_TESTING_ENABLED"`
	TestEndpoints     []string `json:"test_endpoints" env:"SECURITY_TEST_ENDPOINTS"`
	SkipTLSVerify     bool     `json:"skip_tls_verify" env:"SECURITY_TEST_SKIP_TLS"`
	TestInterval      time.Duration `json:"test_interval" env:"SECURITY_TEST_INTERVAL"`
	AlertOnFailure    bool     `json:"alert_on_failure" env:"SECURITY_TEST_ALERT"`
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EncryptionKey: "", // Must be provided via environment
		TLSMinVersion: "1.2",
		Headers:       DefaultSecurityHeadersConfig(),
		RateLimit:     DefaultRateLimitConfig(),
		AuditLog: AuditLogConfig{
			Enabled:         true,
			RetentionPeriod: 7 * 365 * 24 * time.Hour, // 7 years
			EncryptLogs:     true,
			LogLevel:        "info",
		},
		Session: SessionConfig{
			SecureCookies:   true,
			HTTPOnlyCookies: true,
			SameSiteCookies: "strict",
			SessionTimeout:  24 * time.Hour,
			IdleTimeout:     2 * time.Hour,
			CookieName:      "agentscan_session",
			CookieDomain:    "",
			CookiePath:      "/",
		},
		CORS: CORSConfig{
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
		},
		SecurityTesting: SecurityTestingConfig{
			Enabled:       false, // Disabled by default in production
			TestEndpoints: []string{"/api/health", "/api/auth/login"},
			SkipTLSVerify: false,
			TestInterval:  24 * time.Hour,
			AlertOnFailure: true,
		},
	}
}

// ProductionSecurityConfig returns a production-ready security configuration
func ProductionSecurityConfig() SecurityConfig {
	config := DefaultSecurityConfig()
	
	// Production-specific overrides
	config.Headers.HSTSMaxAge = 31536000 // 1 year
	config.Headers.HSTSPreload = true
	config.Session.SecureCookies = true
	config.Session.SameSiteCookies = "strict"
	config.SecurityTesting.Enabled = false
	
	// Stricter CSP for production
	config.Headers.CSPDirectives = map[string][]string{
		"default-src": {"'self'"},
		"script-src":  {"'self'"},
		"style-src":   {"'self'", "https://fonts.googleapis.com"},
		"font-src":    {"'self'", "https://fonts.gstatic.com"},
		"img-src":     {"'self'", "data:", "https:"},
		"connect-src": {"'self'", "wss:", "https:"},
		"media-src":   {"'none'"},
		"object-src":  {"'none'"},
		"frame-src":   {"'none'"},
		"base-uri":    {"'self'"},
		"form-action": {"'self'"},
		"upgrade-insecure-requests": {},
	}
	
	// Stricter rate limits for production
	config.RateLimit.PerIPRPS = 50
	config.RateLimit.PerIPBurst = 100
	config.RateLimit.DDoSThreshold = 200
	
	return config
}

// DevelopmentSecurityConfig returns a development-friendly security configuration
func DevelopmentSecurityConfig() SecurityConfig {
	config := DefaultSecurityConfig()
	
	// Development-specific overrides
	config.Session.SecureCookies = false // Allow HTTP in development
	config.SecurityTesting.Enabled = true
	config.SecurityTesting.SkipTLSVerify = true
	
	// More permissive CSP for development
	config.Headers.CSPDirectives["script-src"] = []string{"'self'", "'unsafe-inline'", "'unsafe-eval'"}
	config.Headers.CSPDirectives["style-src"] = []string{"'self'", "'unsafe-inline'"}
	
	// More permissive CORS for development
	config.CORS.AllowedOrigins = append(config.CORS.AllowedOrigins,
		"http://localhost:3000",
		"http://localhost:3001",
		"http://localhost:8080",
		"http://127.0.0.1:3000",
	)
	
	// Higher rate limits for development
	config.RateLimit.PerIPRPS = 1000
	config.RateLimit.PerIPBurst = 2000
	
	return config
}

// SecurityManager manages all security components
type SecurityManager struct {
	config          SecurityConfig
	encryptionSvc   *EncryptionService
	auditLogger     *AuditLogger
	rateLimiter     *RateLimiter
	testSuite       *SecurityTestSuite
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config SecurityConfig, redisClient *redis.Client) (*SecurityManager, error) {
	// Initialize encryption service
	if config.EncryptionKey == "" {
		return nil, fmt.Errorf("encryption key is required")
	}
	encryptionSvc := NewEncryptionService(config.EncryptionKey)
	
	// Initialize audit logger
	auditLogger := NewAuditLogger("agentscan", "1.0.0", encryptionSvc)
	
	// Initialize rate limiter
	config.RateLimit.RedisClient = redisClient
	rateLimiter := NewRateLimiter(config.RateLimit, auditLogger)
	
	// Initialize security test suite (if enabled)
	var testSuite *SecurityTestSuite
	if config.SecurityTesting.Enabled {
		testSuite = NewSecurityTestSuite("https://localhost:8080") // Default base URL
	}
	
	return &SecurityManager{
		config:        config,
		encryptionSvc: encryptionSvc,
		auditLogger:   auditLogger,
		rateLimiter:   rateLimiter,
		testSuite:     testSuite,
	}, nil
}

// GetEncryptionService returns the encryption service
func (sm *SecurityManager) GetEncryptionService() *EncryptionService {
	return sm.encryptionSvc
}

// GetAuditLogger returns the audit logger
func (sm *SecurityManager) GetAuditLogger() *AuditLogger {
	return sm.auditLogger
}

// GetRateLimiter returns the rate limiter
func (sm *SecurityManager) GetRateLimiter() *RateLimiter {
	return sm.rateLimiter
}

// GetSecurityTestSuite returns the security test suite
func (sm *SecurityManager) GetSecurityTestSuite() *SecurityTestSuite {
	return sm.testSuite
}

// GetSecurityMiddleware returns all security middleware
func (sm *SecurityManager) GetSecurityMiddleware() []gin.HandlerFunc {
	middleware := SecurityMiddleware(sm.config.Headers)
	middleware = append(middleware, sm.rateLimiter.RateLimitMiddleware())
	return middleware
}

// ValidateConfig validates the security configuration
func (sm *SecurityManager) ValidateConfig() error {
	if sm.config.EncryptionKey == "" {
		return fmt.Errorf("encryption key is required")
	}
	
	if len(sm.config.EncryptionKey) < 32 {
		return fmt.Errorf("encryption key must be at least 32 characters")
	}
	
	if sm.config.Session.SessionTimeout <= 0 {
		return fmt.Errorf("session timeout must be positive")
	}
	
	if sm.config.RateLimit.PerIPRPS <= 0 {
		return fmt.Errorf("per-IP rate limit must be positive")
	}
	
	return nil
}

// GetConfig returns the security configuration
func (sm *SecurityManager) GetConfig() SecurityConfig {
	return sm.config
}