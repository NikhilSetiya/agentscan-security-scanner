package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// EnvManager manages environment variable loading and validation
type EnvManager struct {
	loaded     bool
	envFiles   []string
	required   []string
	validators map[string]func(string) error
}

// NewEnvManager creates a new environment manager
func NewEnvManager() *EnvManager {
	return &EnvManager{
		validators: make(map[string]func(string) error),
		required:   make([]string, 0),
	}
}

// LoadEnvFiles loads environment variables from files
func (em *EnvManager) LoadEnvFiles(files ...string) error {
	if em.loaded {
		return nil // Already loaded
	}
	
	// Default environment files to try
	defaultFiles := []string{
		".env.local",
		".env." + getEnvironment(),
		".env",
	}
	
	// Use provided files or defaults
	filesToLoad := files
	if len(filesToLoad) == 0 {
		filesToLoad = defaultFiles
	}
	
	// Load each file that exists
	for _, file := range filesToLoad {
		if _, err := os.Stat(file); err == nil {
			if err := godotenv.Load(file); err != nil {
				return fmt.Errorf("failed to load env file %s: %w", file, err)
			}
			em.envFiles = append(em.envFiles, file)
		}
	}
	
	em.loaded = true
	return nil
}

// AddRequired adds required environment variables
func (em *EnvManager) AddRequired(vars ...string) {
	em.required = append(em.required, vars...)
}

// AddValidator adds a validator for an environment variable
func (em *EnvManager) AddValidator(key string, validator func(string) error) {
	em.validators[key] = validator
}

// Validate validates all environment variables
func (em *EnvManager) Validate() error {
	var errors []string
	
	// Check required variables
	for _, key := range em.required {
		if value := os.Getenv(key); value == "" {
			errors = append(errors, fmt.Sprintf("required environment variable %s is not set", key))
		}
	}
	
	// Run validators
	for key, validator := range em.validators {
		if value := os.Getenv(key); value != "" {
			if err := validator(value); err != nil {
				errors = append(errors, fmt.Sprintf("validation failed for %s: %v", key, err))
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("environment validation failed:\n%s", strings.Join(errors, "\n"))
	}
	
	return nil
}

// GetLoadedFiles returns the list of loaded environment files
func (em *EnvManager) GetLoadedFiles() []string {
	return em.envFiles
}

// EnvConfig represents environment configuration with validation
type EnvConfig struct {
	// Application
	AppName        string `env:"APP_NAME" required:"true" default:"AgentScan"`
	AppVersion     string `env:"APP_VERSION" default:"1.0.0"`
	Environment    string `env:"GO_ENV" default:"development"`
	Debug          bool   `env:"APP_DEBUG" default:"false"`
	
	// Server
	Host           string        `env:"HOST" default:"0.0.0.0"`
	Port           int           `env:"PORT" default:"8080" validate:"port"`
	ReadTimeout    time.Duration `env:"READ_TIMEOUT" default:"30s"`
	WriteTimeout   time.Duration `env:"WRITE_TIMEOUT" default:"30s"`
	IdleTimeout    time.Duration `env:"IDLE_TIMEOUT" default:"60s"`
	
	// Database
	DatabaseURL    string `env:"DATABASE_URL" required:"true" validate:"url"`
	DatabaseMaxOpen int   `env:"DATABASE_MAX_OPEN_CONNS" default:"25" validate:"positive"`
	DatabaseMaxIdle int   `env:"DATABASE_MAX_IDLE_CONNS" default:"5" validate:"positive"`
	
	// Redis
	RedisURL       string `env:"REDIS_URL" default:"redis://localhost:6379/0" validate:"url"`
	RedisPassword  string `env:"REDIS_PASSWORD"`
	
	// Security
	JWTSecret      string `env:"JWT_SECRET" required:"true" validate:"min_length:32"`
	HTTPSEnabled   bool   `env:"HTTPS_ENABLED" default:"false"`
	CORSOrigins    string `env:"CORS_ALLOWED_ORIGINS"`
	
	// External Services
	SupabaseURL    string `env:"SUPABASE_URL" required:"true" validate:"url"`
	SupabaseKey    string `env:"SUPABASE_SERVICE_ROLE_KEY" required:"true"`
	
	// Monitoring
	MetricsEnabled bool   `env:"METRICS_ENABLED" default:"true"`
	MetricsPort    int    `env:"METRICS_PORT" default:"9090" validate:"port"`
	LogLevel       string `env:"LOG_LEVEL" default:"info" validate:"log_level"`
}

// LoadEnvConfig loads and validates environment configuration
func LoadEnvConfig() (*EnvConfig, error) {
	config := &EnvConfig{}
	
	if err := loadStructFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load environment config: %w", err)
	}
	
	if err := validateEnvConfig(config); err != nil {
		return nil, fmt.Errorf("environment config validation failed: %w", err)
	}
	
	return config, nil
}

// loadStructFromEnv loads environment variables into a struct
func loadStructFromEnv(config interface{}) error {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must be a pointer to struct")
	}
	
	v = v.Elem()
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		if !field.CanSet() {
			continue
		}
		
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}
		
		required := fieldType.Tag.Get("required") == "true"
		defaultValue := fieldType.Tag.Get("default")
		
		envValue := os.Getenv(envTag)
		
		// Use default if no env value and not required
		if envValue == "" {
			if required {
				return fmt.Errorf("required environment variable %s is not set", envTag)
			}
			envValue = defaultValue
		}
		
		if err := setFieldValue(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}
	
	return nil
}

// setFieldValue sets a struct field value from a string
func setFieldValue(field reflect.Value, value string) error {
	if value == "" {
		return nil
	}
	
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
		
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		field.SetBool(boolVal)
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid duration value: %s", value)
			}
			field.SetInt(int64(duration))
		} else {
			intVal, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer value: %s", value)
			}
			field.SetInt(intVal)
		}
		
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		field.SetUint(uintVal)
		
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(floatVal)
		
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	
	return nil
}

// validateEnvConfig validates environment configuration
func validateEnvConfig(config *EnvConfig) error {
	v := reflect.ValueOf(config).Elem()
	t := v.Type()
	
	var errors []string
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}
		
		fieldValue := getFieldStringValue(field)
		
		if err := validateField(fieldType.Name, fieldValue, validateTag); err != nil {
			errors = append(errors, err.Error())
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("validation errors:\n%s", strings.Join(errors, "\n"))
	}
	
	return nil
}

// getFieldStringValue gets string representation of field value
func getFieldStringValue(field reflect.Value) string {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(field.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(field.Bool())
	default:
		return ""
	}
}

// validateField validates a field value against validation rules
func validateField(fieldName, value, rules string) error {
	if value == "" {
		return nil // Skip validation for empty values
	}
	
	ruleList := strings.Split(rules, ",")
	
	for _, rule := range ruleList {
		rule = strings.TrimSpace(rule)
		
		switch {
		case rule == "url":
			if !isValidURL(value) {
				return fmt.Errorf("field %s: invalid URL format", fieldName)
			}
			
		case rule == "port":
			port, err := strconv.Atoi(value)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("field %s: invalid port number", fieldName)
			}
			
		case rule == "positive":
			num, err := strconv.Atoi(value)
			if err != nil || num <= 0 {
				return fmt.Errorf("field %s: must be a positive number", fieldName)
			}
			
		case rule == "log_level":
			validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
			if !contains(validLevels, strings.ToLower(value)) {
				return fmt.Errorf("field %s: invalid log level", fieldName)
			}
			
		case strings.HasPrefix(rule, "min_length:"):
			minLenStr := strings.TrimPrefix(rule, "min_length:")
			minLen, err := strconv.Atoi(minLenStr)
			if err != nil {
				return fmt.Errorf("field %s: invalid min_length rule", fieldName)
			}
			if len(value) < minLen {
				return fmt.Errorf("field %s: minimum length is %d", fieldName, minLen)
			}
			
		case strings.HasPrefix(rule, "max_length:"):
			maxLenStr := strings.TrimPrefix(rule, "max_length:")
			maxLen, err := strconv.Atoi(maxLenStr)
			if err != nil {
				return fmt.Errorf("field %s: invalid max_length rule", fieldName)
			}
			if len(value) > maxLen {
				return fmt.Errorf("field %s: maximum length is %d", fieldName, maxLen)
			}
		}
	}
	
	return nil
}

// Helper functions
func getEnvironment() string {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = os.Getenv("NODE_ENV")
	}
	if env == "" {
		env = "development"
	}
	return env
}

func isValidURL(str string) bool {
	return strings.HasPrefix(str, "http://") || 
		   strings.HasPrefix(str, "https://") || 
		   strings.HasPrefix(str, "redis://") || 
		   strings.HasPrefix(str, "postgresql://") ||
		   strings.HasPrefix(str, "postgres://")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// EnvDocGenerator generates documentation for environment variables
type EnvDocGenerator struct {
	config interface{}
}

// NewEnvDocGenerator creates a new environment documentation generator
func NewEnvDocGenerator(config interface{}) *EnvDocGenerator {
	return &EnvDocGenerator{config: config}
}

// GenerateMarkdown generates markdown documentation for environment variables
func (edg *EnvDocGenerator) GenerateMarkdown() (string, error) {
	var doc strings.Builder
	
	doc.WriteString("# Environment Variables\n\n")
	doc.WriteString("This document describes all environment variables used by the application.\n\n")
	
	v := reflect.ValueOf(edg.config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	
	// Group by category
	categories := make(map[string][]envVar)
	
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		envTag := fieldType.Tag.Get("env")
		
		if envTag == "" {
			continue
		}
		
		category := getCategory(fieldType.Name)
		
		envVar := envVar{
			Name:        envTag,
			Description: getDescription(fieldType.Name),
			Required:    fieldType.Tag.Get("required") == "true",
			Default:     fieldType.Tag.Get("default"),
			Validation:  fieldType.Tag.Get("validate"),
			Type:        fieldType.Type.String(),
		}
		
		categories[category] = append(categories[category], envVar)
	}
	
	// Write categories
	for category, vars := range categories {
		doc.WriteString(fmt.Sprintf("## %s\n\n", category))
		doc.WriteString("| Variable | Type | Required | Default | Description | Validation |\n")
		doc.WriteString("|----------|------|----------|---------|-------------|------------|\n")
		
		for _, envVar := range vars {
			required := "No"
			if envVar.Required {
				required = "Yes"
			}
			
			doc.WriteString(fmt.Sprintf("| `%s` | %s | %s | `%s` | %s | %s |\n",
				envVar.Name,
				envVar.Type,
				required,
				envVar.Default,
				envVar.Description,
				envVar.Validation,
			))
		}
		
		doc.WriteString("\n")
	}
	
	return doc.String(), nil
}

type envVar struct {
	Name        string
	Description string
	Required    bool
	Default     string
	Validation  string
	Type        string
}

func getCategory(fieldName string) string {
	switch {
	case strings.HasPrefix(fieldName, "App"):
		return "Application"
	case strings.HasPrefix(fieldName, "Host") || strings.HasPrefix(fieldName, "Port") || strings.Contains(fieldName, "Timeout"):
		return "Server"
	case strings.HasPrefix(fieldName, "Database"):
		return "Database"
	case strings.HasPrefix(fieldName, "Redis"):
		return "Redis"
	case strings.Contains(fieldName, "JWT") || strings.Contains(fieldName, "HTTPS") || strings.Contains(fieldName, "CORS"):
		return "Security"
	case strings.HasPrefix(fieldName, "Supabase"):
		return "External Services"
	case strings.Contains(fieldName, "Metrics") || strings.Contains(fieldName, "Log"):
		return "Monitoring"
	default:
		return "Other"
	}
}

func getDescription(fieldName string) string {
	descriptions := map[string]string{
		"AppName":        "Application name",
		"AppVersion":     "Application version",
		"Environment":    "Application environment (development, production, etc.)",
		"Debug":          "Enable debug mode",
		"Host":           "Server host address",
		"Port":           "Server port number",
		"ReadTimeout":    "HTTP read timeout",
		"WriteTimeout":   "HTTP write timeout",
		"IdleTimeout":    "HTTP idle timeout",
		"DatabaseURL":    "Database connection URL",
		"DatabaseMaxOpen": "Maximum number of open database connections",
		"DatabaseMaxIdle": "Maximum number of idle database connections",
		"RedisURL":       "Redis connection URL",
		"RedisPassword":  "Redis password",
		"JWTSecret":      "JWT signing secret",
		"HTTPSEnabled":   "Enable HTTPS",
		"CORSOrigins":    "Allowed CORS origins",
		"SupabaseURL":    "Supabase project URL",
		"SupabaseKey":    "Supabase service role key",
		"MetricsEnabled": "Enable Prometheus metrics",
		"MetricsPort":    "Metrics server port",
		"LogLevel":       "Logging level",
	}
	
	if desc, exists := descriptions[fieldName]; exists {
		return desc
	}
	
	return "No description available"
}

// SaveEnvTemplate saves an environment template file
func SaveEnvTemplate(filename string, config interface{}) error {
	var content strings.Builder
	
	content.WriteString("# Environment Configuration Template\n")
	content.WriteString("# Copy this file to .env and update the values\n\n")
	
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	
	currentCategory := ""
	
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		envTag := fieldType.Tag.Get("env")
		
		if envTag == "" {
			continue
		}
		
		category := getCategory(fieldType.Name)
		if category != currentCategory {
			content.WriteString(fmt.Sprintf("\n# %s\n", category))
			currentCategory = category
		}
		
		required := fieldType.Tag.Get("required") == "true"
		defaultValue := fieldType.Tag.Get("default")
		description := getDescription(fieldType.Name)
		
		content.WriteString(fmt.Sprintf("# %s", description))
		if required {
			content.WriteString(" (Required)")
		}
		content.WriteString("\n")
		
		if required && defaultValue == "" {
			content.WriteString(fmt.Sprintf("%s=\n", envTag))
		} else {
			content.WriteString(fmt.Sprintf("%s=%s\n", envTag, defaultValue))
		}
	}
	
	return os.WriteFile(filename, []byte(content.String()), 0644)
}