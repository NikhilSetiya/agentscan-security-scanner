package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Level:       "info",
				Format:      "json",
				Output:      "stdout",
				ServiceName: "test-service",
				Version:     "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: &Config{
				Level:  "invalid",
				Format: "json",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: &Config{
				Level:  "info",
				Format: "invalid",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			}
		})
	}
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	
	config := &Config{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	logger.SetOutput(&buf)

	// Create context with correlation ID
	ctx := WithCorrelationID(context.Background(), "test-correlation-id")
	ctx = WithUserID(ctx, "test-user-id")

	// Log with context
	logger.WithContext(ctx).Info("test message")

	// Parse the log output
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify context fields are present
	assert.Equal(t, "test-correlation-id", logEntry["correlation_id"])
	assert.Equal(t, "test-user-id", logEntry["user_id"])
	assert.Equal(t, "test-service", logEntry["service"])
	assert.Equal(t, "1.0.0", logEntry["version"])
	assert.Equal(t, "test message", logEntry["message"])
}

func TestLogger_LogRequest(t *testing.T) {
	var buf bytes.Buffer
	
	config := &Config{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	logger.SetOutput(&buf)

	ctx := WithCorrelationID(context.Background(), "test-correlation-id")
	duration := 100 * time.Millisecond

	logger.LogRequest(ctx, "GET", "/api/test", "test-agent", "127.0.0.1", 200, duration)

	// Parse the log output
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify request fields
	assert.Equal(t, "GET", logEntry["http_method"])
	assert.Equal(t, "/api/test", logEntry["http_path"])
	assert.Equal(t, float64(200), logEntry["http_status"])
	assert.Equal(t, "test-agent", logEntry["user_agent"])
	assert.Equal(t, "127.0.0.1", logEntry["client_ip"])
	assert.Equal(t, float64(100), logEntry["response_time_ms"])
}

func TestLogger_LogScanEvent(t *testing.T) {
	var buf bytes.Buffer
	
	config := &Config{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	logger.SetOutput(&buf)

	ctx := WithCorrelationID(context.Background(), "test-correlation-id")
	fields := logrus.Fields{
		"findings_count": 5,
		"duration":       "30s",
	}

	logger.LogScanEvent(ctx, "scan_completed", "job-123", "https://github.com/test/repo", "semgrep", fields)

	// Parse the log output
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify scan event fields
	assert.Equal(t, "scan_completed", logEntry["event"])
	assert.Equal(t, "job-123", logEntry["job_id"])
	assert.Equal(t, "https://github.com/test/repo", logEntry["repo_url"])
	assert.Equal(t, "semgrep", logEntry["agent_name"])
	assert.Equal(t, float64(5), logEntry["findings_count"])
	assert.Equal(t, "30s", logEntry["duration"])
}

func TestLogger_LogError(t *testing.T) {
	var buf bytes.Buffer
	
	config := &Config{
		Level:       "debug", // Enable debug level to see stack trace
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	logger.SetOutput(&buf)

	ctx := WithCorrelationID(context.Background(), "test-correlation-id")
	testErr := assert.AnError
	fields := logrus.Fields{
		"component": "test-component",
	}

	logger.LogError(ctx, testErr, "test error message", fields)

	// Parse the log output
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify error fields
	assert.Equal(t, "test error message", logEntry["message"])
	assert.Equal(t, testErr.Error(), logEntry["error"])
	assert.Equal(t, "test-component", logEntry["component"])
	assert.Contains(t, logEntry, "stack_trace") // Should have stack trace in debug mode
}

func TestCorrelationIDFunctions(t *testing.T) {
	// Test NewCorrelationID
	id1 := NewCorrelationID()
	id2 := NewCorrelationID()
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // Should be unique

	// Test WithCorrelationID and GetCorrelationID
	ctx := context.Background()
	testID := "test-correlation-id"
	
	ctx = WithCorrelationID(ctx, testID)
	retrievedID := GetCorrelationID(ctx)
	assert.Equal(t, testID, retrievedID)

	// Test empty context
	emptyCtx := context.Background()
	emptyID := GetCorrelationID(emptyCtx)
	assert.Empty(t, emptyID)
}

func TestUserIDFunctions(t *testing.T) {
	ctx := context.Background()
	testUserID := "test-user-123"
	
	ctx = WithUserID(ctx, testUserID)
	retrievedUserID := GetUserID(ctx)
	assert.Equal(t, testUserID, retrievedUserID)

	// Test empty context
	emptyCtx := context.Background()
	emptyUserID := GetUserID(emptyCtx)
	assert.Empty(t, emptyUserID)
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	
	config := &Config{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	logger.SetOutput(&buf)

	fields := logrus.Fields{
		"custom_field": "custom_value",
		"number_field": 42,
	}

	logger.WithFields(fields).Info("test message with fields")

	// Parse the log output
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify custom fields
	assert.Equal(t, "custom_value", logEntry["custom_field"])
	assert.Equal(t, float64(42), logEntry["number_field"])
	assert.Equal(t, "test-service", logEntry["service"])
	assert.Equal(t, "1.0.0", logEntry["version"])
}

func TestLogger_WithError(t *testing.T) {
	var buf bytes.Buffer
	
	config := &Config{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	logger.SetOutput(&buf)

	testErr := assert.AnError
	logger.WithError(testErr).Error("error occurred")

	// Parse the log output
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify error fields
	assert.Equal(t, testErr.Error(), logEntry["error"])
	assert.Contains(t, logEntry["error_type"], "errors.errorString")
}

func TestLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	
	config := &Config{
		Level:       "info",
		Format:      "text",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	logger.SetOutput(&buf)

	logger.WithFields(logrus.Fields{
		"test_field": "test_value",
	}).Info("test message")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "test_field=test_value")
	assert.Contains(t, output, "service=test-service")
}

func BenchmarkLogger_WithContext(b *testing.B) {
	config := &Config{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(b, err)
	logger.SetOutput(&bytes.Buffer{}) // Discard output

	ctx := WithCorrelationID(context.Background(), "test-correlation-id")
	ctx = WithUserID(ctx, "test-user-id")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithContext(ctx).Info("benchmark message")
	}
}

func BenchmarkLogger_WithFields(b *testing.B) {
	config := &Config{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		ServiceName: "test-service",
		Version:     "1.0.0",
	}

	logger, err := NewLogger(config)
	require.NoError(b, err)
	logger.SetOutput(&bytes.Buffer{}) // Discard output

	fields := logrus.Fields{
		"field1": "value1",
		"field2": "value2",
		"field3": 123,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(fields).Info("benchmark message")
	}
}