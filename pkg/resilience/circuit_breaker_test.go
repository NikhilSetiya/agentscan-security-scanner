package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_DefaultBehavior(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     time.Second,
	})

	// Initially closed
	assert.Equal(t, StateClosed, cb.State())

	// Successful requests should keep it closed
	for i := 0; i < 5; i++ {
		result, err := cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
			return "success", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, StateClosed, cb.State())
	}
}

func TestCircuitBreaker_TripsOnFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     100 * time.Millisecond,
	})

	// Generate enough failures to trip the circuit breaker
	for i := 0; i < 5; i++ {
		_, err := cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
			return nil, errors.New("test error")
		})
		require.Error(t, err)
	}

	// Circuit breaker should be open now
	assert.Equal(t, StateOpen, cb.State())

	// Requests should be rejected
	_, err := cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "should not execute", nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker")
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 2,
		Interval:    time.Second,
		Timeout:     50 * time.Millisecond,
	})

	// Trip the circuit breaker
	for i := 0; i < 5; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
			return nil, errors.New("test error")
		})
	}
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should be half-open now
	_, err := cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})
	require.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())

	// Another successful request should close the circuit
	_, err = cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})
	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 2,
		Interval:    time.Second,
		Timeout:     50 * time.Millisecond,
	})

	// Trip the circuit breaker
	for i := 0; i < 5; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
			return nil, errors.New("test error")
		})
	}
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Fail in half-open state should open the circuit again
	_, err := cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("test error")
	})
	require.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_CustomReadyToTrip(t *testing.T) {
	tripped := false
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     time.Second,
		ReadyToTrip: func(counts Counts) bool {
			// Trip after 2 consecutive failures
			return counts.ConsecutiveFailures >= 2
		},
		OnStateChange: func(name string, from CircuitState, to CircuitState) {
			if to == StateOpen {
				tripped = true
			}
		},
	})

	// First failure
	cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("test error")
	})
	assert.Equal(t, StateClosed, cb.State())
	assert.False(t, tripped)

	// Second failure should trip
	cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("test error")
	})
	assert.Equal(t, StateOpen, cb.State())
	assert.True(t, tripped)
}

func TestCircuitBreaker_Counts(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     time.Second,
	})

	// Execute some requests
	cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})
	cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("error")
	})
	cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})

	counts := cb.Counts()
	assert.Equal(t, uint32(3), counts.Requests)
	assert.Equal(t, uint32(2), counts.TotalSuccesses)
	assert.Equal(t, uint32(1), counts.TotalFailures)
	assert.Equal(t, uint32(1), counts.ConsecutiveSuccesses)
	assert.Equal(t, uint32(0), counts.ConsecutiveFailures)
}

func TestCircuitBreaker_Panic(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     time.Second,
	})

	// Test that panics are properly handled
	assert.Panics(t, func() {
		cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
			panic("test panic")
		})
	})

	// Circuit breaker should record this as a failure
	counts := cb.Counts()
	assert.Equal(t, uint32(1), counts.Requests)
	assert.Equal(t, uint32(0), counts.TotalSuccesses)
	assert.Equal(t, uint32(1), counts.TotalFailures)
}

func TestCircuitBreaker_Call(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 3,
		Interval:    time.Second,
		Timeout:     time.Second,
	})

	// Test the Call convenience method
	result, err := cb.Call(func() (interface{}, error) {
		return "success", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "success", result)

	// Test Call with error
	_, err = cb.Call(func() (interface{}, error) {
		return nil, errors.New("test error")
	})
	require.Error(t, err)
	assert.Equal(t, "test error", err.Error())
}

func TestIsCircuitBreakerError(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:        "test-cb",
		MaxRequests: 1,
		Interval:    time.Second,
		Timeout:     time.Second,
	})

	// Trip the circuit breaker
	for i := 0; i < 5; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
			return nil, errors.New("test error")
		})
	}

	// Try to execute when circuit is open
	_, err := cb.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		return "should not execute", nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker")

	// Test with non-circuit breaker error
	regularErr := errors.New("regular error")
	assert.False(t, IsCircuitBreakerError(regularErr))
}