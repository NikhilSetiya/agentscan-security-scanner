package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/agentscan/internal/shared/testing"
)

func TestMain(m *testing.M) {
	// Setup test environment
	os.Setenv("GO_ENV", "test")
	// Run tests
	code := m.Run()
	// Cleanup
	os.Exit(code)
}

func TestLoadEnvConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		validate    func(*testing.T, *EnvConfig)
	}{
		{
			name: "valid_minimal_config",
			envVars: map[string]string{
				"DATABASE_URL":               "postgresql://test:test@localhost:5432/test",
				"SUPABASE_URL":              "https://test.supabase.co",
				"SUPABASE_SERVICE_ROLE_KEY": "test_service_role_key_32_chars_long",
				"JWT_SECRET":                "test_jwt_secret_32_characters_long",
			},
			expectError: false,
			validate: func(t *testing.T, config *EnvConfig) {
				assert.Equal(t, "AgentScan", config.AppName)
				assert.Equal(t, 8080, config.Port)
				assert.True(t, config.MetricsEnabled)
			},
		},
		{
			name: "missing_required_database_url",
			envVars: map[string]string{
				"SUPABASE_URL":              "https://test.supabase.co",
				"SUPABASE_SERVICE_ROLE_KEY": "test_service_role_key_32_chars_long",
				"JWT_SECRET":                "test_jwt_secret_32_characters_long",
			},
			expectError: true,
		},
		{
			name: "invalid_port_number",
			envVars: map[string]string{
				"PORT":                      "99999",
				"DATABASE_URL":              "postgresql://test:test@localhost:5432/test",
				"SUPABASE_URL":              "https://test.supabase.co",
				"SUPABASE_SERVICE_ROLE_KEY": "test_service_role_key_32_chars_long",
				"JWT_SECRET":                "test_jwt_secret_32_characters_long",
			},
			expectError: true,
		},
		{
			name: "short_jwt_secret",
			envVars: map[string]string{
				"DATABASE_URL":              "postgresql://test:test@localhost:5432/test",
				"SUPABASE_URL":              "https://test.supabase.co",
				"SUPABASE_SERVICE_ROLE_KEY": "test_service_role_key_32_chars_long",
				"JWT_SECRET":                "short",
			},
			expectError: true,
		},
		{
			name: "invalid_log_level",
			envVars: map[string]string{
				"LOG_LEVEL":                 "invalid",
				"DATABASE_URL":              "postgresql://test:test@localhost:5432/test",
				"SUPABASE_URL":              "https://test.supabase.co",
				"SUPABASE_SERVICE_ROLE_KEY": "test_service_role_key_32_chars_long",
				"JWT_SECRET":                "test_jwt_secret_32_characters_long",
			},
			expectError: true,
		},
		{
			name: "custom_timeouts",
			envVars: map[string]string{
				"READ_TIMEOUT":              "45s",
				"WRITE_TIMEOUT":             "60s",
				"IDLE_TIMEOUT":              "120s",
				"DATABASE_URL":              "postgresql://test:test@localhost:5432/test",
				"SUPABASE_URL":              "https://test.supabase.co",
				"SUPABASE_SERVICE_ROLE_KEY": "test_service_role_key_32_chars_long",
				"JWT_SECRET":                "test_jwt_secret_32_characters_long",
			},
			expectError: false,
			validate: func(t *testing.T, config *EnvConfig) {
				assert.Equal(t, 45*time.Second, config.ReadTimeout)
				assert.Equal(t, 60*time.Second, config.WriteTimeout)
				assert.Equal(t, 120*time.Second, config.IdleTimeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearTestEnv()
			
			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearTestEnv()

			config, err := LoadEnvConfig()
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestSecurityConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *SecurityConfig
		expectError bool
		validate    func(*testing.T, *SecurityConfig)
	}{
		{
			name: "valid_security_config",
			setupConfig: func() *SecurityConfig {
				return &SecurityConfig{
					HTTPS: HTTPSConfig{
						Enabled:  true,
						CertFile: "/path/to/cert.pem",
						KeyFile:  "/path/to/key.pem",
					},
					JWT: JWTConfig{
						Secret:    "test_secret_32_characters_long_enough",
						Algorithm: "HS256",
					},
					CORS: CORSConfig{
						Enabled:        true,
						AllowedOrigins: []string{"https://example.com"},
					},
				}
			},
			expectError: false,
		},
		{
			name: "https_enabled_missing_cert",
			setupConfig: func() *SecurityConfig {
				return &SecurityConfig{
					HTTPS: HTTPSConfig{
						Enabled: true,
						KeyFile: "/path/to/key.pem",
					},
					JWT: JWTConfig{
						Secret: "test_secret_32_characters_long_enough",
					},
				}
			},
			expectError: true,
		},
		{
			name: "short_jwt_secret",
			setupConfig: func() *SecurityConfig {
				return &SecurityConfig{
					JWT: JWTConfig{
						Secret: "short",
					},
				}
			},
			expectError: true,
		},
		{
			name: "invalid_cors_origin",
			setupConfig: func() *SecurityConfig {
				return &SecurityConfig{
					JWT: JWTConfig{
						Secret: "test_secret_32_characters_long_enough",
					},
					CORS: CORSConfig{
						Enabled:        true,
						AllowedOrigins: []string{"invalid-url"},
					},
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setupConfig()
			err := config.ValidateSecurityConfig()
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestProductionConfig(t *testing.T) {
	t.Run("production_readiness_check", func(t *testing.T) {
		config := &ProductionConfig{
			App: AppConfig{
				Debug: false,
			},
			Security: &SecurityConfig{
				HTTPS: HTTPSConfig{
					Enabled: true,
				},
				JWT: JWTConfig{
					Secret: "production_secret_32_characters_long",
				},
			},
			Database: DatabaseConfig{
				SSLMode: "require",
			},
			Monitoring: MonitoringConfig{
				Enabled:      true,
				PprofEnabled: false,
			},
			Logging: LoggingConfig{
				Level: "info",
			},
		}

		ready, issues := config.IsProductionReady()
		assert.True(t, ready)
		assert.Empty(t, issues)
	})

	t.Run("production_readiness_issues", func(t *testing.T) {
		config := &ProductionConfig{
			App: AppConfig{
				Debug: true, // Should be false in production
			},
			Security: &SecurityConfig{
				HTTPS: HTTPSConfig{
					Enabled: false, // Should be true in production
				},
				JWT: JWTConfig{
					Secret: "short", // Too short
				},
			},
			Database: DatabaseConfig{
				SSLMode: "disable", // Should be require or verify-full
			},
			Monitoring: MonitoringConfig{
				Enabled:      false, // Should be true
				PprofEnabled: true,  // Should be false in production
			},
			Logging: LoggingConfig{
				Level: "debug", // Should not be debug in production
			},
		}

		ready, issues := config.IsProductionReady()
		assert.False(t, ready)
		assert.NotEmpty(t, issues)
		assert.Contains(t, issues, "Debug mode is enabled")
		assert.Contains(t, issues, "HTTPS is not enabled")
	})
}

func TestEnvManager(t *testing.T) {
	t.Run("load_env_files", func(t *testing.T) {
		// Create temporary env file
		suite := testing.NewTestSuite(t)
		defer suite.Cleanup()

		envContent := `TEST_VAR=test_value
ANOTHER_VAR=another_value`
		envFile := suite.CreateTempFile(envContent)

		manager := NewEnvManager()
		err := manager.LoadEnvFiles(envFile)
		assert.NoError(t, err)

		loadedFiles := manager.GetLoadedFiles()
		assert.Contains(t, loadedFiles, envFile)
		assert.Equal(t, "test_value", os.Getenv("TEST_VAR"))
	})

	t.Run("validation_with_required_vars", func(t *testing.T) {
		manager := NewEnvManager()
		manager.AddRequired("REQUIRED_VAR")

		// Should fail without required var
		err := manager.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "REQUIRED_VAR")

		// Should pass with required var
		os.Setenv("REQUIRED_VAR", "value")
		defer os.Unsetenv("REQUIRED_VAR")

		err = manager.Validate()
		assert.NoError(t, err)
	})

	t.Run("validation_with_custom_validator", func(t *testing.T) {
		manager := NewEnvManager()
		manager.AddValidator("TEST_PORT", func(value string) error {
			if value == "invalid" {
				return fmt.Errorf("invalid port")
			}
			return nil
		})

		os.Setenv("TEST_PORT", "invalid")
		defer os.Unsetenv("TEST_PORT")

		err := manager.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid port")
	})
}

func TestFieldValidation(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		value       string
		rules       string
		expectError bool
	}{
		{
			name:        "valid_url",
			fieldName:   "test_url",
			value:       "https://example.com",
			rules:       "url",
			expectError: false,
		},
		{
			name:        "invalid_url",
			fieldName:   "test_url",
			value:       "not-a-url",
			rules:       "url",
			expectError: true,
		},
		{
			name:        "valid_port",
			fieldName:   "test_port",
			value:       "8080",
			rules:       "port",
			expectError: false,
		},
		{
			name:        "invalid_port_high",
			fieldName:   "test_port",
			value:       "99999",
			rules:       "port",
			expectError: true,
		},
		{
			name:        "invalid_port_low",
			fieldName:   "test_port",
			value:       "0",
			rules:       "port",
			expectError: true,
		},
		{
			name:        "valid_positive",
			fieldName:   "test_positive",
			value:       "10",
			rules:       "positive",
			expectError: false,
		},
		{
			name:        "invalid_positive",
			fieldName:   "test_positive",
			value:       "-1",
			rules:       "positive",
			expectError: true,
		},
		{
			name:        "valid_log_level",
			fieldName:   "test_log_level",
			value:       "info",
			rules:       "log_level",
			expectError: false,
		},
		{
			name:        "invalid_log_level",
			fieldName:   "test_log_level",
			value:       "invalid",
			rules:       "log_level",
			expectError: true,
		},
		{
			name:        "valid_min_length",
			fieldName:   "test_min_length",
			value:       "long_enough",
			rules:       "min_length:5",
			expectError: false,
		},
		{
			name:        "invalid_min_length",
			fieldName:   "test_min_length",
			value:       "short",
			rules:       "min_length:10",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewFieldValidator()
			err := validator.ValidateField(tt.fieldName, tt.value, tt.rules)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigIntegration(t *testing.T) {
	t.Run("full_config_load_and_validate", func(t *testing.T) {
		// Setup complete environment
		envVars := map[string]string{
			"APP_NAME":                   "AgentScan Test",
			"APP_VERSION":                "1.0.0-test",
			"GO_ENV":                     "test",
			"PORT":                       "8080",
			"HOST":                       "localhost",
			"DATABASE_URL":               "postgresql://test:test@localhost:5432/agentscan_test",
			"REDIS_URL":                  "redis://localhost:6379/0",
			"SUPABASE_URL":              "https://test.supabase.co",
			"SUPABASE_SERVICE_ROLE_KEY": "test_service_role_key_32_chars_long",
			"JWT_SECRET":                "test_jwt_secret_32_characters_long",
			"LOG_LEVEL":                 "debug",
			"METRICS_ENABLED":           "true",
			"HEALTH_CHECK_ENABLED":      "true",
			"CORS_ENABLED":              "true",
			"CORS_ALLOWED_ORIGINS":      "https://localhost:3000,https://app.example.com",
			"RATE_LIMIT_ENABLED":        "true",
			"RATE_LIMIT_REQUESTS":       "100",
			"RATE_LIMIT_WINDOW":         "1m",
		}

		clearTestEnv()
		for key, value := range envVars {
			os.Setenv(key, value)
		}
		defer clearTestEnv()

		config, err := LoadConfig()
		require.NoError(t, err)
		require.NotNil(t, config)

		// Validate all sections
		assert.Equal(t, "AgentScan Test", config.App.Name)
		assert.Equal(t, "1.0.0-test", config.App.Version)
		assert.Equal(t, "test", config.App.Environment)
		assert.Equal(t, 8080, config.Server.Port)
		assert.Equal(t, "localhost", config.Server.Host)
		assert.Contains(t, config.Database.URL, "postgresql://")
		assert.Contains(t, config.Redis.URL, "redis://")
		assert.True(t, config.Monitoring.Enabled)
		assert.True(t, config.Security.CORS.Enabled)
		assert.Len(t, config.Security.CORS.AllowedOrigins, 2)
	})
}

// Helper functions

func clearTestEnv() {
	envVars := []string{
		"APP_NAME", "APP_VERSION", "GO_ENV", "PORT", "HOST",
		"DATABASE_URL", "REDIS_URL", "SUPABASE_URL", "SUPABASE_SERVICE_ROLE_KEY",
		"JWT_SECRET", "LOG_LEVEL", "METRICS_ENABLED", "HEALTH_CHECK_ENABLED",
		"CORS_ENABLED", "CORS_ALLOWED_ORIGINS", "RATE_LIMIT_ENABLED",
		"RATE_LIMIT_REQUESTS", "RATE_LIMIT_WINDOW", "READ_TIMEOUT",
		"WRITE_TIMEOUT", "IDLE_TIMEOUT", "TEST_VAR", "ANOTHER_VAR",
		"REQUIRED_VAR", "TEST_PORT",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

// Benchmark tests

func BenchmarkLoadConfig(b *testing.B) {
	// Setup environment
	envVars := map[string]string{
		"DATABASE_URL":               "postgresql://test:test@localhost:5432/test",
		"SUPABASE_URL":              "https://test.supabase.co",
		"SUPABASE_SERVICE_ROLE_KEY": "test_service_role_key_32_chars_long",
		"JWT_SECRET":                "test_jwt_secret_32_characters_long",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}
	defer clearTestEnv()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateConfig(b *testing.B) {
	config := &Config{
		App: AppConfig{
			Name:        "AgentScan",
			Version:     "1.0.0",
			Environment: "test",
		},
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			URL: "postgresql://test:test@localhost:5432/test",
		},
		Security: &SecurityConfig{
			JWT: JWTConfig{
				Secret: "test_jwt_secret_32_characters_long",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}