package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadProductionConfig loads configuration specifically for production environment
func LoadProductionConfig() (*Config, error) {
	config := &Config{}
	
	// Load application configuration
	config.App = AppConfig{
		Name:        getEnvOrDefault("APP_NAME", "AgentScan"),
		Version:     getEnvOrDefault("APP_VERSION", "1.0.0"),
		Environment: "production",
		Debug:       false, // Always false in production
	}
	
	// Load server configuration
	port, err := strconv.Atoi(getEnvOrDefault("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}
	
	config.Server = ServerConfig{
		Host:            getEnvOrDefault("HOST", "0.0.0.0"),
		Port:            port,
		ReadTimeout:     parseDurationOrDefault("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:    parseDurationOrDefault("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:     parseDurationOrDefault("IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: parseDurationOrDefault("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
	
	// Load database configuration
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required in production")
	}
	
	config.Database = DatabaseConfig{
		URL:             databaseURL,
		MaxOpenConns:    parseIntOrDefault("DATABASE_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    parseIntOrDefault("DATABASE_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: parseDurationOrDefault("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
		ConnMaxIdleTime: parseDurationOrDefault("DATABASE_CONN_MAX_IDLE_TIME", 5*time.Minute),
	}
	
	// Load Redis configuration
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required in production")
	}
	
	config.Redis = RedisConfig{
		URL:         redisURL,
		MaxRetries:  parseIntOrDefault("REDIS_MAX_RETRIES", 3),
		PoolSize:    parseIntOrDefault("REDIS_POOL_SIZE", 10),
		PoolTimeout: parseDurationOrDefault("REDIS_POOL_TIMEOUT", 4*time.Second),
	}
	
	// Load security configuration
	config.Security = &SecurityConfig{}
	
	// JWT configuration
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required in production")
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}
	
	config.Security.JWT = JWTConfig{
		Secret:    jwtSecret,
		Algorithm: getEnvOrDefault("JWT_ALGORITHM", "HS256"),
		Issuer:    getEnvOrDefault("JWT_ISSUER", "agentscan-prod"),
		Audience:  getEnvOrDefault("JWT_AUDIENCE", "agentscan-api"),
		TTL:       parseDurationOrDefault("JWT_TTL", 24*time.Hour),
	}
	
	// HTTPS configuration
	config.Security.HTTPS = HTTPSConfig{
		Enabled:      parseBoolOrDefault("HTTPS_ENABLED", true),
		CertFile:     os.Getenv("HTTPS_CERT_FILE"),
		KeyFile:      os.Getenv("HTTPS_KEY_FILE"),
		RedirectHTTP: parseBoolOrDefault("HTTPS_REDIRECT_HTTP", true),
	}
	
	// CORS configuration
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if corsOrigins != "" {
		allowedOrigins = strings.Split(corsOrigins, ",")
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	}
	
	config.Security.CORS = CORSConfig{
		Enabled:        parseBoolOrDefault("CORS_ENABLED", true),
		AllowedOrigins: allowedOrigins,
		AllowedMethods: strings.Split(getEnvOrDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"), ","),
		AllowedHeaders: strings.Split(getEnvOrDefault("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Requested-With"), ","),
		ExposedHeaders: strings.Split(getEnvOrDefault("CORS_EXPOSED_HEADERS", "X-Total-Count"), ","),
		AllowCredentials: parseBoolOrDefault("CORS_ALLOW_CREDENTIALS", true),
		MaxAge:         parseIntOrDefault("CORS_MAX_AGE", 3600),
	}
	
	// Security headers configuration
	config.Security.Headers = SecurityHeaders{
		Enabled: parseBoolOrDefault("SECURITY_HEADERS_ENABLED", true),
		HSTS: HSTSConfig{
			Enabled:           parseBoolOrDefault("HSTS_ENABLED", true),
			MaxAge:            parseIntOrDefault("HSTS_MAX_AGE", 31536000),
			IncludeSubdomains: parseBoolOrDefault("HSTS_INCLUDE_SUBDOMAINS", true),
			Preload:           parseBoolOrDefault("HSTS_PRELOAD", true),
		},
		CSP: CSPConfig{
			Enabled:   parseBoolOrDefault("CSP_ENABLED", true),
			Policy:    getEnvOrDefault("CSP_POLICY", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"),
			ReportURI: getEnvOrDefault("CSP_REPORT_URI", "/csp-report"),
		},
	}
	
	// Rate limiting configuration
	config.Security.RateLimit = RateLimitConfig{
		Enabled:  parseBoolOrDefault("RATE_LIMIT_ENABLED", true),
		Requests: parseIntOrDefault("RATE_LIMIT_REQUESTS", 1000),
		Window:   parseDurationOrDefault("RATE_LIMIT_WINDOW", time.Hour),
		Burst:    parseIntOrDefault("RATE_LIMIT_BURST", 100),
	}
	
	// Load monitoring configuration
	config.Monitoring = MonitoringConfig{
		Enabled:      parseBoolOrDefault("METRICS_ENABLED", true),
		MetricsPort:  parseIntOrDefault("METRICS_PORT", 9090),
		MetricsPath:  getEnvOrDefault("METRICS_PATH", "/metrics"),
		HealthCheck:  parseBoolOrDefault("HEALTH_CHECK_ENABLED", true),
		PprofEnabled: parseBoolOrDefault("PPROF_ENABLED", false), // Disabled in production
	}
	
	// Load logging configuration
	config.Logging = LoggingConfig{
		Level:  getEnvOrDefault("LOG_LEVEL", "info"),
		Format: getEnvOrDefault("LOG_FORMAT", "json"),
		Output: getEnvOrDefault("LOG_OUTPUT", "stdout"),
	}
	
	// Load Supabase configuration
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL is required in production")
	}
	
	supabaseServiceRoleKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if supabaseServiceRoleKey == "" {
		return nil, fmt.Errorf("SUPABASE_SERVICE_ROLE_KEY is required in production")
	}
	
	config.Supabase = SupabaseConfig{
		URL:            supabaseURL,
		ServiceRoleKey: supabaseServiceRoleKey,
		AnonKey:        os.Getenv("SUPABASE_ANON_KEY"), // Optional for backend
	}
	
	// Validate production configuration
	if err := validateProductionConfig(config); err != nil {
		return nil, fmt.Errorf("production configuration validation failed: %w", err)
	}
	
	return config, nil
}

// validateProductionConfig performs production-specific validation
func validateProductionConfig(config *Config) error {
	// Ensure debug is disabled
	if config.App.Debug {
		return fmt.Errorf("debug mode must be disabled in production")
	}
	
	// Ensure HTTPS is enabled
	if !config.Security.HTTPS.Enabled {
		return fmt.Errorf("HTTPS must be enabled in production")
	}
	
	// Ensure security headers are enabled
	if !config.Security.Headers.Enabled {
		return fmt.Errorf("security headers must be enabled in production")
	}
	
	// Ensure HSTS is enabled
	if !config.Security.Headers.HSTS.Enabled {
		return fmt.Errorf("HSTS must be enabled in production")
	}
	
	// Ensure CSP is enabled
	if !config.Security.Headers.CSP.Enabled {
		return fmt.Errorf("CSP must be enabled in production")
	}
	
	// Ensure rate limiting is enabled
	if !config.Security.RateLimit.Enabled {
		return fmt.Errorf("rate limiting must be enabled in production")
	}
	
	// Ensure monitoring is enabled
	if !config.Monitoring.Enabled {
		return fmt.Errorf("monitoring must be enabled in production")
	}
	
	// Ensure pprof is disabled
	if config.Monitoring.PprofEnabled {
		return fmt.Errorf("pprof must be disabled in production")
	}
	
	// Validate JWT secret strength
	if len(config.Security.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters in production")
	}
	
	// Validate database URL format
	if !strings.Contains(config.Database.URL, "sslmode=require") && 
	   !strings.Contains(config.Database.URL, "sslmode=verify-full") {
		return fmt.Errorf("database must use SSL in production (sslmode=require or verify-full)")
	}
	
	// Validate CORS origins
	if len(config.Security.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("CORS allowed origins must be specified in production")
	}
	
	for _, origin := range config.Security.CORS.AllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("wildcard CORS origin (*) is not allowed in production")
		}
		if !strings.HasPrefix(origin, "https://") {
			return fmt.Errorf("all CORS origins must use HTTPS in production: %s", origin)
		}
	}
	
	return nil
}

// Helper functions for parsing environment variables

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// GetProductionReadinessReport generates a report of production readiness
func GetProductionReadinessReport() (bool, []string) {
	var issues []string
	
	// Check required environment variables
	requiredVars := []string{
		"DATABASE_URL",
		"REDIS_URL",
		"SUPABASE_URL",
		"SUPABASE_SERVICE_ROLE_KEY",
		"JWT_SECRET",
	}
	
	for _, varName := range requiredVars {
		if os.Getenv(varName) == "" {
			issues = append(issues, fmt.Sprintf("Missing required environment variable: %s", varName))
		}
	}
	
	// Check JWT secret strength
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" && len(jwtSecret) < 32 {
		issues = append(issues, "JWT_SECRET must be at least 32 characters")
	}
	
	// Check database SSL requirement
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		if !strings.Contains(dbURL, "sslmode=require") && !strings.Contains(dbURL, "sslmode=verify-full") {
			issues = append(issues, "Database must use SSL (sslmode=require or verify-full)")
		}
	}
	
	// Check CORS configuration
	if corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); corsOrigins != "" {
		origins := strings.Split(corsOrigins, ",")
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if origin == "*" {
				issues = append(issues, "Wildcard CORS origin (*) is not allowed in production")
			}
			if !strings.HasPrefix(origin, "https://") {
				issues = append(issues, fmt.Sprintf("CORS origin must use HTTPS: %s", origin))
			}
		}
	}
	
	// Check security settings
	if !parseBoolOrDefault("HTTPS_ENABLED", true) {
		issues = append(issues, "HTTPS must be enabled in production")
	}
	
	if !parseBoolOrDefault("SECURITY_HEADERS_ENABLED", true) {
		issues = append(issues, "Security headers must be enabled in production")
	}
	
	if !parseBoolOrDefault("RATE_LIMIT_ENABLED", true) {
		issues = append(issues, "Rate limiting must be enabled in production")
	}
	
	if parseBoolOrDefault("DEBUG", false) {
		issues = append(issues, "Debug mode must be disabled in production")
	}
	
	if parseBoolOrDefault("PPROF_ENABLED", false) {
		issues = append(issues, "Pprof must be disabled in production")
	}
	
	return len(issues) == 0, issues
}