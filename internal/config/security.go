package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// SecurityConfig holds all security-related configuration
type SecurityConfig struct {
	// HTTPS Configuration
	HTTPS HTTPSConfig `json:"https"`
	
	// CORS Configuration
	CORS CORSConfig `json:"cors"`
	
	// Rate Limiting
	RateLimit RateLimitConfig `json:"rate_limit"`
	
	// Session Security
	Session SessionConfig `json:"session"`
	
	// JWT Configuration
	JWT JWTConfig `json:"jwt"`
	
	// Content Security Policy
	CSP CSPConfig `json:"csp"`
	
	// Security Headers
	Headers SecurityHeaders `json:"headers"`
	
	// API Security
	API APISecurityConfig `json:"api"`
}

type HTTPSConfig struct {
	Enabled          bool   `json:"enabled"`
	Port             int    `json:"port"`
	CertFile         string `json:"cert_file"`
	KeyFile          string `json:"key_file"`
	RedirectHTTP     bool   `json:"redirect_http"`
	HSTSMaxAge       int    `json:"hsts_max_age"`
	HSTSIncludeSubdomains bool `json:"hsts_include_subdomains"`
	HSTSPreload      bool   `json:"hsts_preload"`
}

type CORSConfig struct {
	Enabled          bool     `json:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

type RateLimitConfig struct {
	Enabled        bool                    `json:"enabled"`
	GlobalLimit    RateLimit              `json:"global_limit"`
	EndpointLimits map[string]RateLimit   `json:"endpoint_limits"`
	UserLimits     map[string]RateLimit   `json:"user_limits"`
	IPWhitelist    []string               `json:"ip_whitelist"`
	IPBlacklist    []string               `json:"ip_blacklist"`
}

type RateLimit struct {
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
	Burst    int           `json:"burst"`
}

type SessionConfig struct {
	CookieName     string        `json:"cookie_name"`
	CookieSecure   bool          `json:"cookie_secure"`
	CookieHTTPOnly bool          `json:"cookie_http_only"`
	CookieSameSite string        `json:"cookie_same_site"`
	CookieDomain   string        `json:"cookie_domain"`
	CookiePath     string        `json:"cookie_path"`
	MaxAge         time.Duration `json:"max_age"`
	IdleTimeout    time.Duration `json:"idle_timeout"`
}

type JWTConfig struct {
	Secret          string        `json:"-"` // Never serialize secrets
	Algorithm       string        `json:"algorithm"`
	AccessTokenTTL  time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl"`
	Issuer          string        `json:"issuer"`
	Audience        []string      `json:"audience"`
}

type CSPConfig struct {
	Enabled         bool     `json:"enabled"`
	DefaultSrc      []string `json:"default_src"`
	ScriptSrc       []string `json:"script_src"`
	StyleSrc        []string `json:"style_src"`
	ImgSrc          []string `json:"img_src"`
	ConnectSrc      []string `json:"connect_src"`
	FontSrc         []string `json:"font_src"`
	ObjectSrc       []string `json:"object_src"`
	MediaSrc        []string `json:"media_src"`
	FrameSrc        []string `json:"frame_src"`
	ReportURI       string   `json:"report_uri"`
	ReportOnly      bool     `json:"report_only"`
}

type SecurityHeaders struct {
	XFrameOptions           string `json:"x_frame_options"`
	XContentTypeOptions     string `json:"x_content_type_options"`
	XSSProtection           string `json:"xss_protection"`
	ReferrerPolicy          string `json:"referrer_policy"`
	PermissionsPolicy       string `json:"permissions_policy"`
	CrossOriginEmbedderPolicy string `json:"cross_origin_embedder_policy"`
	CrossOriginOpenerPolicy   string `json:"cross_origin_opener_policy"`
	CrossOriginResourcePolicy string `json:"cross_origin_resource_policy"`
}

type APISecurityConfig struct {
	RequireAPIKey       bool     `json:"require_api_key"`
	AllowedAPIKeys      []string `json:"-"` // Never serialize API keys
	RequireAuthentication bool   `json:"require_authentication"`
	AllowedUserAgents   []string `json:"allowed_user_agents"`
	BlockedUserAgents   []string `json:"blocked_user_agents"`
	MaxRequestSize      int64    `json:"max_request_size"`
	RequestTimeout      time.Duration `json:"request_timeout"`
}

// LoadSecurityConfig loads security configuration from environment variables
func LoadSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		HTTPS: HTTPSConfig{
			Enabled:               getEnvBool("HTTPS_ENABLED", true),
			Port:                  getEnvInt("HTTPS_PORT", 443),
			CertFile:              getEnvString("TLS_CERT_FILE", "/etc/ssl/certs/server.crt"),
			KeyFile:               getEnvString("TLS_KEY_FILE", "/etc/ssl/private/server.key"),
			RedirectHTTP:          getEnvBool("HTTPS_REDIRECT_HTTP", true),
			HSTSMaxAge:            getEnvInt("HSTS_MAX_AGE", 31536000), // 1 year
			HSTSIncludeSubdomains: getEnvBool("HSTS_INCLUDE_SUBDOMAINS", true),
			HSTSPreload:           getEnvBool("HSTS_PRELOAD", true),
		},
		CORS: CORSConfig{
			Enabled: getEnvBool("CORS_ENABLED", true),
			AllowedOrigins: getEnvStringSlice("CORS_ALLOWED_ORIGINS", []string{
				"https://agentscan.vercel.app",
				"https://agentscan-security-scanner.vercel.app",
				"https://app.agentscan.io",
			}),
			AllowedMethods: getEnvStringSlice("CORS_ALLOWED_METHODS", []string{
				"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH",
			}),
			AllowedHeaders: getEnvStringSlice("CORS_ALLOWED_HEADERS", []string{
				"Accept", "Authorization", "Content-Type", "X-CSRF-Token",
				"X-Requested-With", "X-API-Key", "X-Client-Version",
			}),
			ExposedHeaders: getEnvStringSlice("CORS_EXPOSED_HEADERS", []string{
				"X-Total-Count", "X-Page-Count", "X-Rate-Limit-Remaining",
			}),
			AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getEnvInt("CORS_MAX_AGE", 86400), // 24 hours
		},
		RateLimit: RateLimitConfig{
			Enabled: getEnvBool("RATE_LIMIT_ENABLED", true),
			GlobalLimit: RateLimit{
				Requests: getEnvInt("RATE_LIMIT_GLOBAL_REQUESTS", 1000),
				Window:   getEnvDuration("RATE_LIMIT_GLOBAL_WINDOW", time.Hour),
				Burst:    getEnvInt("RATE_LIMIT_GLOBAL_BURST", 100),
			},
			EndpointLimits: map[string]RateLimit{
				"/api/v1/auth/login": {
					Requests: getEnvInt("RATE_LIMIT_LOGIN_REQUESTS", 5),
					Window:   getEnvDuration("RATE_LIMIT_LOGIN_WINDOW", 15*time.Minute),
					Burst:    getEnvInt("RATE_LIMIT_LOGIN_BURST", 2),
				},
				"/api/v1/auth/register": {
					Requests: getEnvInt("RATE_LIMIT_REGISTER_REQUESTS", 3),
					Window:   getEnvDuration("RATE_LIMIT_REGISTER_WINDOW", time.Hour),
					Burst:    getEnvInt("RATE_LIMIT_REGISTER_BURST", 1),
				},
				"/api/v1/scans": {
					Requests: getEnvInt("RATE_LIMIT_SCANS_REQUESTS", 10),
					Window:   getEnvDuration("RATE_LIMIT_SCANS_WINDOW", time.Hour),
					Burst:    getEnvInt("RATE_LIMIT_SCANS_BURST", 3),
				},
			},
			UserLimits: map[string]RateLimit{
				"free": {
					Requests: getEnvInt("RATE_LIMIT_FREE_REQUESTS", 100),
					Window:   getEnvDuration("RATE_LIMIT_FREE_WINDOW", 24*time.Hour),
					Burst:    getEnvInt("RATE_LIMIT_FREE_BURST", 10),
				},
				"pro": {
					Requests: getEnvInt("RATE_LIMIT_PRO_REQUESTS", 1000),
					Window:   getEnvDuration("RATE_LIMIT_PRO_WINDOW", 24*time.Hour),
					Burst:    getEnvInt("RATE_LIMIT_PRO_BURST", 50),
				},
			},
			IPWhitelist: getEnvStringSlice("RATE_LIMIT_IP_WHITELIST", []string{}),
			IPBlacklist: getEnvStringSlice("RATE_LIMIT_IP_BLACKLIST", []string{}),
		},
		Session: SessionConfig{
			CookieName:     getEnvString("SESSION_COOKIE_NAME", "agentscan_session"),
			CookieSecure:   getEnvBool("SESSION_COOKIE_SECURE", true),
			CookieHTTPOnly: getEnvBool("SESSION_COOKIE_HTTP_ONLY", true),
			CookieSameSite: getEnvString("SESSION_COOKIE_SAME_SITE", "Strict"),
			CookieDomain:   getEnvString("SESSION_COOKIE_DOMAIN", ""),
			CookiePath:     getEnvString("SESSION_COOKIE_PATH", "/"),
			MaxAge:         getEnvDuration("SESSION_MAX_AGE", 24*time.Hour),
			IdleTimeout:    getEnvDuration("SESSION_IDLE_TIMEOUT", 2*time.Hour),
		},
		JWT: JWTConfig{
			Secret:          getJWTSecret(),
			Algorithm:       getEnvString("JWT_ALGORITHM", "HS256"),
			AccessTokenTTL:  getEnvDuration("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getEnvDuration("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
			Issuer:          getEnvString("JWT_ISSUER", "agentscan.io"),
			Audience:        getEnvStringSlice("JWT_AUDIENCE", []string{"agentscan.io"}),
		},
		CSP: CSPConfig{
			Enabled: getEnvBool("CSP_ENABLED", true),
			DefaultSrc: getEnvStringSlice("CSP_DEFAULT_SRC", []string{
				"'self'",
			}),
			ScriptSrc: getEnvStringSlice("CSP_SCRIPT_SRC", []string{
				"'self'",
				"'unsafe-inline'", // Required for some React builds
				"https://cdn.jsdelivr.net",
				"https://unpkg.com",
			}),
			StyleSrc: getEnvStringSlice("CSP_STYLE_SRC", []string{
				"'self'",
				"'unsafe-inline'", // Required for CSS-in-JS
				"https://fonts.googleapis.com",
			}),
			ImgSrc: getEnvStringSlice("CSP_IMG_SRC", []string{
				"'self'",
				"data:",
				"https:",
				"blob:",
			}),
			ConnectSrc: getEnvStringSlice("CSP_CONNECT_SRC", []string{
				"'self'",
				"https://api.agentscan.io",
				"wss://api.agentscan.io",
				"https://*.supabase.co",
			}),
			FontSrc: getEnvStringSlice("CSP_FONT_SRC", []string{
				"'self'",
				"https://fonts.gstatic.com",
			}),
			ObjectSrc: getEnvStringSlice("CSP_OBJECT_SRC", []string{
				"'none'",
			}),
			MediaSrc: getEnvStringSlice("CSP_MEDIA_SRC", []string{
				"'self'",
			}),
			FrameSrc: getEnvStringSlice("CSP_FRAME_SRC", []string{
				"'none'",
			}),
			ReportURI:  getEnvString("CSP_REPORT_URI", "/api/v1/csp-report"),
			ReportOnly: getEnvBool("CSP_REPORT_ONLY", false),
		},
		Headers: SecurityHeaders{
			XFrameOptions:             getEnvString("X_FRAME_OPTIONS", "DENY"),
			XContentTypeOptions:       getEnvString("X_CONTENT_TYPE_OPTIONS", "nosniff"),
			XSSProtection:             getEnvString("XSS_PROTECTION", "1; mode=block"),
			ReferrerPolicy:            getEnvString("REFERRER_POLICY", "strict-origin-when-cross-origin"),
			PermissionsPolicy:         getEnvString("PERMISSIONS_POLICY", "geolocation=(), microphone=(), camera=()"),
			CrossOriginEmbedderPolicy: getEnvString("CROSS_ORIGIN_EMBEDDER_POLICY", "require-corp"),
			CrossOriginOpenerPolicy:   getEnvString("CROSS_ORIGIN_OPENER_POLICY", "same-origin"),
			CrossOriginResourcePolicy: getEnvString("CROSS_ORIGIN_RESOURCE_POLICY", "same-origin"),
		},
		API: APISecurityConfig{
			RequireAPIKey:         getEnvBool("API_REQUIRE_API_KEY", false),
			AllowedAPIKeys:        getEnvStringSlice("API_ALLOWED_KEYS", []string{}),
			RequireAuthentication: getEnvBool("API_REQUIRE_AUTH", true),
			AllowedUserAgents:     getEnvStringSlice("API_ALLOWED_USER_AGENTS", []string{}),
			BlockedUserAgents: getEnvStringSlice("API_BLOCKED_USER_AGENTS", []string{
				"curl", "wget", "python-requests", // Block common bot user agents
			}),
			MaxRequestSize: getEnvInt64("API_MAX_REQUEST_SIZE", 10*1024*1024), // 10MB
			RequestTimeout: getEnvDuration("API_REQUEST_TIMEOUT", 30*time.Second),
		},
	}
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	
	// Generate a random secret if none is provided (not recommended for production)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("Failed to generate JWT secret: %v", err))
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// ValidateSecurityConfig validates the security configuration
func (sc *SecurityConfig) ValidateSecurityConfig() error {
	// Validate HTTPS configuration
	if sc.HTTPS.Enabled {
		if sc.HTTPS.CertFile == "" {
			return fmt.Errorf("HTTPS enabled but no certificate file specified")
		}
		if sc.HTTPS.KeyFile == "" {
			return fmt.Errorf("HTTPS enabled but no key file specified")
		}
	}
	
	// Validate CORS origins
	for _, origin := range sc.CORS.AllowedOrigins {
		if origin != "*" {
			if _, err := url.Parse(origin); err != nil {
				return fmt.Errorf("invalid CORS origin: %s", origin)
			}
		}
	}
	
	// Validate JWT configuration
	if sc.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if len(sc.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}
	
	// Validate rate limits
	if sc.RateLimit.GlobalLimit.Requests <= 0 {
		return fmt.Errorf("global rate limit requests must be positive")
	}
	
	return nil
}

// GetCSPHeader returns the Content Security Policy header value
func (csp *CSPConfig) GetCSPHeader() string {
	if !csp.Enabled {
		return ""
	}
	
	var policies []string
	
	if len(csp.DefaultSrc) > 0 {
		policies = append(policies, fmt.Sprintf("default-src %s", strings.Join(csp.DefaultSrc, " ")))
	}
	if len(csp.ScriptSrc) > 0 {
		policies = append(policies, fmt.Sprintf("script-src %s", strings.Join(csp.ScriptSrc, " ")))
	}
	if len(csp.StyleSrc) > 0 {
		policies = append(policies, fmt.Sprintf("style-src %s", strings.Join(csp.StyleSrc, " ")))
	}
	if len(csp.ImgSrc) > 0 {
		policies = append(policies, fmt.Sprintf("img-src %s", strings.Join(csp.ImgSrc, " ")))
	}
	if len(csp.ConnectSrc) > 0 {
		policies = append(policies, fmt.Sprintf("connect-src %s", strings.Join(csp.ConnectSrc, " ")))
	}
	if len(csp.FontSrc) > 0 {
		policies = append(policies, fmt.Sprintf("font-src %s", strings.Join(csp.FontSrc, " ")))
	}
	if len(csp.ObjectSrc) > 0 {
		policies = append(policies, fmt.Sprintf("object-src %s", strings.Join(csp.ObjectSrc, " ")))
	}
	if len(csp.MediaSrc) > 0 {
		policies = append(policies, fmt.Sprintf("media-src %s", strings.Join(csp.MediaSrc, " ")))
	}
	if len(csp.FrameSrc) > 0 {
		policies = append(policies, fmt.Sprintf("frame-src %s", strings.Join(csp.FrameSrc, " ")))
	}
	if csp.ReportURI != "" {
		policies = append(policies, fmt.Sprintf("report-uri %s", csp.ReportURI))
	}
	
	return strings.Join(policies, "; ")
}

// IsProductionEnvironment checks if we're running in production
func IsProductionEnvironment() bool {
	env := strings.ToLower(os.Getenv("GO_ENV"))
	return env == "production" || env == "prod"
}

// IsDevelopmentEnvironment checks if we're running in development
func IsDevelopmentEnvironment() bool {
	env := strings.ToLower(os.Getenv("GO_ENV"))
	return env == "development" || env == "dev" || env == ""
}