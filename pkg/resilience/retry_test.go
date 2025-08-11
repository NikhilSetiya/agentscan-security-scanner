package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	appErrors "github.com/agentscan/agentscan/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrier_SuccessOnFirstAttempt(t *testing.T) {
	retrier := NewRetrier(DefaultRetryConfig())

	attempts := 0
	err := retrier.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetrier_SuccessAfterRetries(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond
	retrier := NewRetrier(config)

	attempts := 0
	err := retrier.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return appErrors.NewTimeoutError("test timeout")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetrier_FailureAfterMaxAttempts(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond
	retrier := NewRetrier(config)

	attempts := 0
	err := retrier.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		return appErrors.NewTimeoutError("test timeout")
	})

	require.Error(t, err)
	assert.Equal(t, 3, attempts)
	assert.Contains(t, err.Error(), "operation failed after 3 attempts")
}

func TestRetrier_NonRetryableError(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond
	retrier := NewRetrier(config)

	attempts := 0
	err := retrier.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		return appErrors.NewValidationError("validation failed")
	})

	require.Error(t, err)
	assert.Equal(t, 1, attempts) // Should not retry validation errors
	assert.Contains(t, err.Error(), "validation failed")
}

func TestRetrier_ContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxAttempts = 5
	config.InitialDelay = 100 * time.Millisecond
	retrier := NewRetrier(config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	err := retrier.Execute(ctx, func(ctx context.Context) error {
		attempts++
		return appErrors.NewTimeoutError("test timeout")
	})

	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, 1, attempts) // Should stop after context cancellation
}

func TestRetrier_CustomRetryableErrors(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond
	config.RetryableErrors = func(err error) bool {
		return err.Error() == "retryable"
	}
	retrier := NewRetrier(config)

	// Test retryable error
	attempts := 0
	err := retrier.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("retryable")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)

	// Test non-retryable error
	attempts = 0
	err = retrier.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		return errors.New("not retryable")
	})
	require.Error(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetrier_OnRetryCallback(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond

	var retryAttempts []int
	var retryErrors []error
	var retryDelays []time.Duration

	config.OnRetry = func(attempt int, err error, delay time.Duration) {
		retryAttempts = append(retryAttempts, attempt)
		retryErrors = append(retryErrors, err)
		retryDelays = append(retryDelays, delay)
	}

	retrier := NewRetrier(config)

	attempts := 0
	err := retrier.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return appErrors.NewTimeoutError("test timeout")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
	assert.Len(t, retryAttempts, 2) // 2 retries
	assert.Equal(t, []int{1, 2}, retryAttempts)
	assert.Len(t, retryErrors, 2)
	assert.Len(t, retryDelays, 2)
}

func TestRetrier_ExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       4,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            false, // Disable jitter for predictable testing
		RetryableErrors:   DefaultRetryableErrors,
	}

	retrier := NewRetrier(config)

	var delays []time.Duration
	config.OnRetry = func(attempt int, err error, delay time.Duration) {
		delays = append(delays, delay)
	}
	retrier.config.OnRetry = config.OnRetry

	attempts := 0
	retrier.Execute(context.Background(), func(ctx context.Context) error {
		attempts++
		return appErrors.NewTimeoutError("test timeout")
	})

	require.Len(t, delays, 3) // 3 retries
	
	// Check exponential backoff (approximately, allowing for some variance)
	assert.InDelta(t, 10*time.Millisecond, delays[0], float64(5*time.Millisecond))
	assert.InDelta(t, 20*time.Millisecond, delays[1], float64(5*time.Millisecond))
	assert.InDelta(t, 40*time.Millisecond, delays[2], float64(5*time.Millisecond))
}

func TestRetrier_MaxDelayLimit(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          150 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
		RetryableErrors:   DefaultRetryableErrors,
	}

	retrier := NewRetrier(config)

	var delays []time.Duration
	config.OnRetry = func(attempt int, err error, delay time.Duration) {
		delays = append(delays, delay)
	}
	retrier.config.OnRetry = config.OnRetry

	retrier.Execute(context.Background(), func(ctx context.Context) error {
		return appErrors.NewTimeoutError("test timeout")
	})

	// All delays should be capped at MaxDelay
	for _, delay := range delays {
		assert.LessOrEqual(t, delay, 150*time.Millisecond)
	}
}

func TestRetrier_ExecuteWithResult(t *testing.T) {
	retrier := NewRetrier(DefaultRetryConfig())

	// Test successful execution with result
	result, err := retrier.ExecuteWithResult(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "success", result)

	// Test failed execution
	_, err = retrier.ExecuteWithResult(context.Background(), func(ctx context.Context) (interface{}, error) {
		return nil, appErrors.NewValidationError("validation failed")
	})
	require.Error(t, err)
}

func TestDefaultRetryableErrors(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"nil error", nil, false},
		{"timeout error", appErrors.NewTimeoutError("timeout"), true},
		{"external error", appErrors.NewExternalError("service", "error"), true},
		{"validation error", appErrors.NewValidationError("validation"), false},
		{"authentication error", appErrors.NewAuthenticationError("auth"), false},
		{"authorization error", appErrors.NewAuthorizationError("authz"), false},
		{"not found error", appErrors.NewNotFoundError("resource"), false},
		{"internal error", appErrors.NewInternalError("internal"), true},
		{"circuit breaker error", &CircuitBreakerError{Name: "test", State: StateOpen}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultRetryableErrors(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestRetryConvenienceFunctions(t *testing.T) {
	// Test Retry function
	attempts := 0
	err := Retry(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return appErrors.NewTimeoutError("timeout")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)

	// Test RetryWithResult function
	result, err := RetryWithResult(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "result", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "result", result)
}

func TestRetryableOperation(t *testing.T) {
	cbConfig := CircuitBreakerConfig{
		Name:        "test-op",
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     100 * time.Millisecond,
	}

	retryConfig := DefaultRetryConfig()
	retryConfig.MaxAttempts = 2
	retryConfig.InitialDelay = 10 * time.Millisecond

	op := NewRetryableOperation("test-op", cbConfig, retryConfig)

	// Test successful execution
	result, err := op.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "success", result)

	// Test ExecuteVoid
	err = op.ExecuteVoid(context.Background(), func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)

	// Test state and counts
	assert.Equal(t, StateClosed, op.State())
	counts := op.Counts()
	assert.Equal(t, uint32(2), counts.Requests)
	assert.Equal(t, uint32(2), counts.TotalSuccesses)
}