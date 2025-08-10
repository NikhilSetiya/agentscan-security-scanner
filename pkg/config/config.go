package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	Agents   AgentsConfig   `json:"agents"`
	Auth     AuthConfig     `json:"auth"`
	Logging  LoggingConfig  `json:"logging"`
	GitHub   GitHubConfig   `json:"github"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Name            string `json:"name"`
	User            string `json:"user"`
	Password        string `json:"password"`
	SSLMode         string `json:"ssl_mode"`
	MaxOpenConns    int    `json:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	PoolSize int    `json:"pool_size"`
}

// AgentsConfig contains agent execution configuration
type AgentsConfig struct {
	MaxConcurrent   int           `json:"max_concurrent"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	DockerRegistry  string        `json:"docker_registry"`
	TempDir         string        `json:"temp_dir"`
	MaxMemoryMB     int           `json:"max_memory_mb"`
	MaxCPUCores     float64       `json:"max_cpu_cores"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	JWTSecret       string        `json:"jwt_secret"`
	JWTExpiration   time.Duration `json:"jwt_expiration"`
	GitHubClientID  string        `json:"github_client_id"`
	GitHubSecret    string        `json:"github_secret"`
	GitLabClientID  string        `json:"gitlab_client_id"`
	GitLabSecret    string        `json:"gitlab_secret"`
}

// GitHubConfig holds GitHub App configuration
type GitHubConfig struct {
	AppID      int64  `json:"app_id"`
	PrivateKey string `json:"private_key"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Host:         getEnvString("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnvString("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			Name:            getEnvString("DB_NAME", "agentscan"),
			User:            getEnvString("DB_USER", "agentscan"),
			Password:        getEnvString("DB_PASSWORD", ""),
			SSLMode:         getEnvString("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnvString("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnvString("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
		},
		Agents: AgentsConfig{
			MaxConcurrent:  getEnvInt("AGENTS_MAX_CONCURRENT", 10),
			DefaultTimeout: getEnvDuration("AGENTS_DEFAULT_TIMEOUT", 10*time.Minute),
			DockerRegistry: getEnvString("AGENTS_DOCKER_REGISTRY", ""),
			TempDir:        getEnvString("AGENTS_TEMP_DIR", "/tmp/agentscan"),
			MaxMemoryMB:    getEnvInt("AGENTS_MAX_MEMORY_MB", 1024),
			MaxCPUCores:    getEnvFloat("AGENTS_MAX_CPU_CORES", 1.0),
		},
		Auth: AuthConfig{
			JWTSecret:      getEnvString("JWT_SECRET", ""),
			JWTExpiration:  getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
			GitHubClientID: getEnvString("GITHUB_CLIENT_ID", ""),
			GitHubSecret:   getEnvString("GITHUB_SECRET", ""),
			GitLabClientID: getEnvString("GITLAB_CLIENT_ID", ""),
			GitLabSecret:   getEnvString("GITLAB_SECRET", ""),
		},
		Logging: LoggingConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
			Output: getEnvString("LOG_OUTPUT", "stdout"),
		},
		GitHub: GitHubConfig{
			AppID:      getEnvInt64("GITHUB_APP_ID", 0),
			PrivateKey: getEnvString("GITHUB_PRIVATE_KEY", ""),
		},
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}
	
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if c.Auth.GitHubClientID == "" || c.Auth.GitHubSecret == "" {
		return fmt.Errorf("GitHub OAuth credentials are required")
	}

	return nil
}

// DatabaseURL returns the database connection URL
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// RedisURL returns the Redis connection URL
func (c *Config) RedisURL() string {
	if c.Redis.Password != "" {
		return fmt.Sprintf("redis://:%s@%s:%d/%d",
			c.Redis.Password,
			c.Redis.Host,
			c.Redis.Port,
			c.Redis.DB,
		)
	}
	return fmt.Sprintf("redis://%s:%d/%d",
		c.Redis.Host,
		c.Redis.Port,
		c.Redis.DB,
	)
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}