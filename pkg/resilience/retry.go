package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/logging"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int
	// InitialDelay is the initial delay before the first retry
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// Jitter adds randomness to delay to avoid thundering herd
	Jitter bool
	// RetryableErrors is a function that determines if an error is retryable
	RetryableErrors func(error) bool
	// OnRetry is called before each retry attempt
	OnRetry func(attempt int, err error, delay time.Duration)
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
		RetryableErrors:   DefaultRetryableErrors,
	}
}

// DefaultRetryableErrors determines if an error is retryable by default
func DefaultRetryableErrors(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types that are retryable
	if errors.IsType(err, errors.ErrorTypeTimeout) ||
		errors.IsType(err, errors.ErrorTypeExternal) {
		return true
	}

	// Check for circuit breaker errors (not retryable)
	if IsCircuitBreakerError(err) {
		return false
	}

	// Check for validation errors (not retryable)
	if errors.IsType(err, errors.ErrorTypeValidation) ||
		errors.IsType(err, errors.ErrorTypeAuthentication) ||
		errors.IsType(err, errors.ErrorTypeAuthorization) ||
		errors.IsType(err, errors.ErrorTypeNotFound) {
		return false
	}

	return true
}

// Retrier handles retry logic with exponential backoff
type Retrier struct {
	config RetryConfig
	logger *logging.Logger
}

// NewRetrier creates a new retrier with the given configuration
func NewRetrier(config RetryConfig) *Retrier {
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.BackoffMultiplier <= 0 {
		config.BackoffMultiplier = 2.0
	}
	if config.RetryableErrors == nil {
		config.RetryableErrors = DefaultRetryableErrors
	}

	return &Retrier{
		config: config,
		logger: logging.GetLogger(),
	}
}

// Execute executes the given function with retry logic
func (r *Retrier) Execute(ctx context.Context, operation func(context.Context) error) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := operation(ctx)
		if err == nil {
			if attempt > 1 {
				r.logger.Info("Operation succeeded after retry",
					"attempt", attempt,
					"total_attempts", r.config.MaxAttempts,
				)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.config.RetryableErrors(err) {
			r.logger.Debug("Error is not retryable, stopping",
				"error", err.Error(),
				"attempt", attempt,
			)
			return err
		}

		// Don't retry on the last attempt
		if attempt == r.config.MaxAttempts {
			break
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		r.logger.Debug("Operation failed, retrying",
			"error", err.Error(),
			"attempt", attempt,
			"max_attempts", r.config.MaxAttempts,
			"delay", delay,
		)

		// Call retry callback if provided
		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt, err, delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	r.logger.Error("Operation failed after all retry attempts",
		"error", lastErr.Error(),
		"attempts", r.config.MaxAttempts,
	)

	return fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxAttempts, lastErr)
}

// ExecuteWithResult executes the given function with retry logic and returns a result
func (r *Retrier) ExecuteWithResult(ctx context.Context, operation func(context.Context) (interface{}, error)) (interface{}, error) {
	var result interface{}
	err := r.Execute(ctx, func(ctx context.Context) error {
		var err error
		result, err = operation(ctx)
		return err
	})
	return result, err
}

func (r *Retrier) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff delay
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffMultiplier, float64(attempt-1))

	// Apply maximum delay limit
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Add jitter if enabled
	if r.config.Jitter {
		jitter := rand.Float64() * 0.1 * delay // 10% jitter
		delay += jitter
	}

	return time.Duration(delay)
}

// RetryWithConfig is a convenience function to execute an operation with retry
func RetryWithConfig(ctx context.Context, config RetryConfig, operation func(context.Context) error) error {
	retrier := NewRetrier(config)
	return retrier.Execute(ctx, operation)
}

// Retry is a convenience function to execute an operation with default retry configuration
func Retry(ctx context.Context, operation func(context.Context) error) error {
	return RetryWithConfig(ctx, DefaultRetryConfig(), operation)
}

// RetryWithResult is a convenience function to execute an operation with result and default retry configuration
func RetryWithResult(ctx context.Context, operation func(context.Context) (interface{}, error)) (interface{}, error) {
	retrier := NewRetrier(DefaultRetryConfig())
	return retrier.ExecuteWithResult(ctx, operation)
}

// RetryableOperation wraps an operation with both circuit breaker and retry logic
type RetryableOperation struct {
	circuitBreaker *CircuitBreaker
	retrier        *Retrier
	logger         *logging.Logger
}

// NewRetryableOperation creates a new retryable operation with circuit breaker and retry logic
func NewRetryableOperation(name string, cbConfig CircuitBreakerConfig, retryConfig RetryConfig) *RetryableOperation {
	if cbConfig.Name == "" {
		cbConfig.Name = name
	}

	return &RetryableOperation{
		circuitBreaker: NewCircuitBreaker(cbConfig),
		retrier:        NewRetrier(retryConfig),
		logger:         logging.GetLogger(),
	}
}

// Execute executes an operation with both circuit breaker and retry logic
func (ro *RetryableOperation) Execute(ctx context.Context, operation func(context.Context) (interface{}, error)) (interface{}, error) {
	return ro.retrier.ExecuteWithResult(ctx, func(ctx context.Context) (interface{}, error) {
		return ro.circuitBreaker.Execute(ctx, operation)
	})
}

// ExecuteVoid executes an operation that doesn't return a result
func (ro *RetryableOperation) ExecuteVoid(ctx context.Context, operation func(context.Context) error) error {
	_, err := ro.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, operation(ctx)
	})
	return err
}

// State returns the current state of the circuit breaker
func (ro *RetryableOperation) State() CircuitState {
	return ro.circuitBreaker.State()
}

// Counts returns the current counts of the circuit breaker
func (ro *RetryableOperation) Counts() Counts {
	return ro.circuitBreaker.Counts()
}