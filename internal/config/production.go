package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProductionConfig holds production-specific configuration
type ProductionConfig struct {
	// Application settings
	App AppConfig `json:"app"`
	
	// Server settings
	Server ServerConfig `json:"server"`
	
	// Database settings
	Database DatabaseConfig `json:"database"`
	
	// Redis settings
	Redis RedisConfig `json:"redis"`
	
	// Security settings
	Security *SecurityConfig `json:"security"`
	
	// Logging settings
	Logging LoggingConfig `json:"logging"`
	
	// Monitoring settings
	Monitoring MonitoringConfig `json:"monitoring"`
	
	// Performance settings
	Performance PerformanceConfig `json:"performance"`
	
	// External services
	External ExternalConfig `json:"external"`
}

type AppConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Debug       bool   `json:"debug"`
}

type ServerConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

type DatabaseConfig struct {
	URL                string        `json:"url"`
	MaxOpenConns       int           `json:"max_open_conns"`
	MaxIdleConns       int           `json:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime    time.Duration `json:"conn_max_idle_time"`
	SSLMode            string        `json:"ssl_mode"`
	MigrationsPath     string        `json:"migrations_path"`
	QueryTimeout       time.Duration `json:"query_timeout"`
	SlowQueryThreshold time.Duration `json:"slow_query_threshold"`
}

type RedisConfig struct {
	URL              string        `json:"url"`
	Password         string        `json:"-"` // Never serialize passwords
	MaxRetries       int           `json:"max_retries"`
	PoolSize         int           `json:"pool_size"`
	MinIdleConns     int           `json:"min_idle_conns"`
	PoolTimeout      time.Duration `json:"pool_timeout"`
	IdleTimeout      time.Duration `json:"idle_timeout"`
	ReadTimeout      time.Duration `json:"read_timeout"`
	WriteTimeout     time.Duration `json:"write_timeout"`
}

type LoggingConfig struct {
	Level       string `json:"level"`
	Format      string `json:"format"`
	Output      string `json:"output"`
	FilePath    string `json:"file_path"`
	MaxSize     int    `json:"max_size"`
	MaxBackups  int    `json:"max_backups"`
	MaxAge      int    `json:"max_age"`
	Compress    bool   `json:"compress"`
	EnableCaller bool  `json:"enable_caller"`
}

type MonitoringConfig struct {
	Enabled         bool   `json:"enabled"`
	MetricsEnabled  bool   `json:"metrics_enabled"`
	MetricsPort     int    `json:"metrics_port"`
	MetricsPath     string `json:"metrics_path"`
	HealthEnabled   bool   `json:"health_enabled"`
	HealthPath      string `json:"health_path"`
	PprofEnabled    bool   `json:"pprof_enabled"`
	TracingEnabled  bool   `json:"tracing_enabled"`
	TracingEndpoint string `json:"tracing_endpoint"`
}

type PerformanceConfig struct {
	CacheTTL              time.Duration `json:"cache_ttl"`
	CacheCleanupInterval  time.Duration `json:"cache_cleanup_interval"`
	MaxConcurrentScans    int           `json:"max_concurrent_scans"`
	ScanTimeout           time.Duration `json:"scan_timeout"`
	WorkerConcurrency     int           `json:"worker_concurrency"`
	WorkerQueueSize       int           `json:"worker_queue_size"`
	JobTimeout            time.Duration `json:"job_timeout"`
	JobRetryAttempts      int           `json:"job_retry_attempts"`
	JobRetryDelay         time.Duration `json:"job_retry_delay"`
}

type ExternalConfig struct {
	Supabase SupabaseConfig `json:"supabase"`
	GitHub   GitHubConfig   `json:"github"`
	SMTP     SMTPConfig     `json:"smtp"`
	Storage  StorageConfig  `json:"storage"`
}

type SupabaseConfig struct {
	URL            string `json:"url"`
	ServiceRoleKey string `json:"-"` // Never serialize keys
	JWTSecret      string `json:"-"` // Never serialize secrets
	DatabaseURL    string `json:"database_url"`
}

type GitHubConfig struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"-"` // Never serialize secrets
	WebhookSecret string `json:"-"` // Never serialize secrets
}

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"-"` // Never serialize passwords
	From     string `json:"from"`
	TLS      bool   `json:"tls"`
}

type StorageConfig struct {
	Type        string `json:"type"`
	LocalPath   string `json:"local_path"`
	MaxFileSize int64  `json:"max_file_size"`
	S3Bucket    string `json:"s3_bucket"`
	S3Region    string `json:"s3_region"`
}

// LoadProductionConfig loads production configuration from environment variables
func LoadProductionConfig() (*ProductionConfig, error) {
	config := &ProductionConfig{
		App: AppConfig{
			Name:        getEnvString("APP_NAME", "AgentScan Security Scanner"),
			Version:     getEnvString("APP_VERSION", "1.0.0"),
			Environment: getEnvString("GO_ENV", "production"),
			Debug:       getEnvBool("APP_DEBUG", false),
		},
		Server: ServerConfig{
			Host:            getEnvString("HOST", "0.0.0.0"),
			Port:            getEnvInt("PORT", 8080),
			ReadTimeout:     getEnvDuration("READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			URL:                getEnvString("DATABASE_URL", ""),
			MaxOpenConns:       getEnvInt("DATABASE_MAX_OPEN_CONNS", 25),
			MaxIdleConns:       getEnvInt("DATABASE_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime:    getEnvDuration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime:    getEnvDuration("DATABASE_CONN_MAX_IDLE_TIME", 5*time.Minute),
			SSLMode:            getEnvString("DATABASE_SSL_MODE", "require"),
			MigrationsPath:     getEnvString("DATABASE_MIGRATIONS_PATH", "migrations"),
			QueryTimeout:       getEnvDuration("DATABASE_QUERY_TIMEOUT", 30*time.Second),
			SlowQueryThreshold: getEnvDuration("DATABASE_SLOW_QUERY_THRESHOLD", 1*time.Second),
		},
		Redis: RedisConfig{
			URL:          getEnvString("REDIS_URL", "redis://localhost:6379/0"),
			Password:     getEnvString("REDIS_PASSWORD", ""),
			MaxRetries:   getEnvInt("REDIS_MAX_RETRIES", 3),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
			PoolTimeout:  getEnvDuration("REDIS_POOL_TIMEOUT", 4*time.Second),
			IdleTimeout:  getEnvDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
			ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},
		Security: LoadSecurityConfig(),
		Logging: LoggingConfig{
			Level:        getEnvString("LOG_LEVEL", "info"),
			Format:       getEnvString("LOG_FORMAT", "json"),
			Output:       getEnvString("LOG_OUTPUT", "stdout"),
			FilePath:     getEnvString("LOG_FILE_PATH", "/var/log/agentscan/app.log"),
			MaxSize:      getEnvInt("LOG_MAX_SIZE", 100),
			MaxBackups:   getEnvInt("LOG_MAX_BACKUPS", 5),
			MaxAge:       getEnvInt("LOG_MAX_AGE", 30),
			Compress:     getEnvBool("LOG_COMPRESS", true),
			EnableCaller: getEnvBool("LOG_ENABLE_CALLER", false),
		},
		Monitoring: MonitoringConfig{
			Enabled:         getEnvBool("MONITORING_ENABLED", true),
			MetricsEnabled:  getEnvBool("METRICS_ENABLED", true),
			MetricsPort:     getEnvInt("METRICS_PORT", 9090),
			MetricsPath:     getEnvString("METRICS_PATH", "/metrics"),
			HealthEnabled:   getEnvBool("HEALTH_CHECK_ENABLED", true),
			HealthPath:      getEnvString("HEALTH_CHECK_PATH", "/health"),
			PprofEnabled:    getEnvBool("PPROF_ENABLED", false),
			TracingEnabled:  getEnvBool("TRACING_ENABLED", false),
			TracingEndpoint: getEnvString("TRACING_ENDPOINT", ""),
		},
		Performance: PerformanceConfig{
			CacheTTL:              getEnvDuration("CACHE_TTL", time.Hour),
			CacheCleanupInterval:  getEnvDuration("CACHE_CLEANUP_INTERVAL", 10*time.Minute),
			MaxConcurrentScans:    getEnvInt("MAX_CONCURRENT_SCANS", 5),
			ScanTimeout:           getEnvDuration("SCAN_TIMEOUT", 10*time.Minute),
			WorkerConcurrency:     getEnvInt("WORKER_CONCURRENCY", 10),
			WorkerQueueSize:       getEnvInt("WORKER_QUEUE_SIZE", 1000),
			JobTimeout:            getEnvDuration("JOB_TIMEOUT", 5*time.Minute),
			JobRetryAttempts:      getEnvInt("JOB_RETRY_ATTEMPTS", 3),
			JobRetryDelay:         getEnvDuration("JOB_RETRY_DELAY", 30*time.Second),
		},
		External: ExternalConfig{
			Supabase: SupabaseConfig{
				URL:            getEnvString("SUPABASE_URL", ""),
				ServiceRoleKey: getEnvString("SUPABASE_SERVICE_ROLE_KEY", ""),
				JWTSecret:      getEnvString("SUPABASE_JWT_SECRET", ""),
				DatabaseURL:    getEnvString("SUPABASE_DB_URL", ""),
			},
			GitHub: GitHubConfig{
				ClientID:      getEnvString("GITHUB_CLIENT_ID", ""),
				ClientSecret:  getEnvString("GITHUB_CLIENT_SECRET", ""),
				WebhookSecret: getEnvString("GITHUB_WEBHOOK_SECRET", ""),
			},
			SMTP: SMTPConfig{
				Host:     getEnvString("SMTP_HOST", ""),
				Port:     getEnvInt("SMTP_PORT", 587),
				Username: getEnvString("SMTP_USERNAME", ""),
				Password: getEnvString("SMTP_PASSWORD", ""),
				From:     getEnvString("SMTP_FROM", ""),
				TLS:      getEnvBool("SMTP_TLS", true),
			},
			Storage: StorageConfig{
				Type:        getEnvString("STORAGE_TYPE", "local"),
				LocalPath:   getEnvString("STORAGE_LOCAL_PATH", "/var/lib/agentscan/uploads"),
				MaxFileSize: getEnvInt64("STORAGE_MAX_FILE_SIZE", 10*1024*1024),
				S3Bucket:    getEnvString("STORAGE_S3_BUCKET", ""),
				S3Region:    getEnvString("STORAGE_S3_REGION", "us-east-1"),
			},
		},
	}
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid production configuration: %w", err)
	}
	
	return config, nil
}

// Validate validates the production configuration
func (pc *ProductionConfig) Validate() error {
	// Validate required fields
	if pc.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	
	if pc.External.Supabase.URL == "" {
		return fmt.Errorf("SUPABASE_URL is required")
	}
	
	if pc.External.Supabase.ServiceRoleKey == "" {
		return fmt.Errorf("SUPABASE_SERVICE_ROLE_KEY is required")
	}
	
	// Validate security configuration
	if err := pc.Security.ValidateSecurityConfig(); err != nil {
		return fmt.Errorf("security configuration invalid: %w", err)
	}
	
	// Validate server configuration
	if pc.Server.Port <= 0 || pc.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", pc.Server.Port)
	}
	
	// Validate database configuration
	if pc.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database max open connections must be positive")
	}
	
	// Validate performance configuration
	if pc.Performance.MaxConcurrentScans <= 0 {
		return fmt.Errorf("max concurrent scans must be positive")
	}
	
	return nil
}

// GetDatabaseConnectionString returns the database connection string with SSL mode
func (pc *ProductionConfig) GetDatabaseConnectionString() string {
	if strings.Contains(pc.Database.URL, "sslmode=") {
		return pc.Database.URL
	}
	
	separator := "?"
	if strings.Contains(pc.Database.URL, "?") {
		separator = "&"
	}
	
	return fmt.Sprintf("%s%ssslmode=%s", pc.Database.URL, separator, pc.Database.SSLMode)
}

// IsProductionReady checks if the configuration is ready for production
func (pc *ProductionConfig) IsProductionReady() (bool, []string) {
	var issues []string
	
	// Check HTTPS configuration
	if !pc.Security.HTTPS.Enabled {
		issues = append(issues, "HTTPS is not enabled")
	}
	
	// Check if using default JWT secret
	if len(pc.Security.JWT.Secret) < 32 {
		issues = append(issues, "JWT secret is too short (minimum 32 characters)")
	}
	
	// Check if debug mode is disabled
	if pc.App.Debug {
		issues = append(issues, "Debug mode is enabled")
	}
	
	// Check SSL mode for database
	if pc.Database.SSLMode != "require" && pc.Database.SSLMode != "verify-full" {
		issues = append(issues, "Database SSL mode should be 'require' or 'verify-full'")
	}
	
	// Check if monitoring is enabled
	if !pc.Monitoring.Enabled {
		issues = append(issues, "Monitoring is not enabled")
	}
	
	// Check if pprof is disabled in production
	if pc.Monitoring.PprofEnabled {
		issues = append(issues, "pprof should be disabled in production")
	}
	
	// Check log level
	if pc.Logging.Level == "debug" || pc.Logging.Level == "trace" {
		issues = append(issues, "Log level should not be debug or trace in production")
	}
	
	return len(issues) == 0, issues
}

// CreateDirectories creates necessary directories for the application
func (pc *ProductionConfig) CreateDirectories() error {
	directories := []string{}
	
	// Add log file directory
	if pc.Logging.Output == "file" && pc.Logging.FilePath != "" {
		directories = append(directories, filepath.Dir(pc.Logging.FilePath))
	}
	
	// Add storage directory
	if pc.External.Storage.Type == "local" && pc.External.Storage.LocalPath != "" {
		directories = append(directories, pc.External.Storage.LocalPath)
	}
	
	// Create directories
	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// GetLogLevel returns the log level as a structured logging level
func (pc *ProductionConfig) GetLogLevel() string {
	level := strings.ToLower(pc.Logging.Level)
	switch level {
	case "trace", "debug", "info", "warn", "error", "fatal", "panic":
		return level
	default:
		return "info"
	}
}