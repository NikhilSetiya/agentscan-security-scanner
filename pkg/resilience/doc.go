// Package resilience provides comprehensive error handling, circuit breaker,
// retry logic, and graceful degradation capabilities for the AgentScan system.
//
// This package implements the following patterns:
//
// # Circuit Breaker Pattern
//
// The circuit breaker pattern prevents cascading failures by monitoring
// the failure rate of external service calls and temporarily blocking
// requests when the failure rate exceeds a threshold.
//
//	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
//		Name:        "external-service",
//		MaxRequests: 3,
//		Interval:    time.Minute,
//		Timeout:     30 * time.Second,
//	})
//
//	result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
//		return externalService.Call(ctx, data)
//	})
//
// # Retry with Exponential Backoff
//
// The retry mechanism automatically retries failed operations with
// exponential backoff and jitter to avoid thundering herd problems.
//
//	retrier := resilience.NewRetrier(resilience.DefaultRetryConfig())
//	err := retrier.Execute(ctx, func(ctx context.Context) error {
//		return riskyOperation(ctx)
//	})
//
// # Graceful Degradation
//
// The degradation system monitors service health and automatically
// adjusts system behavior based on the current degradation level.
//
//	dm := resilience.NewDegradationManager()
//	dm.RegisterService("critical-service", resilience.LevelCritical)
//	
//	// Update service health
//	dm.UpdateServiceHealth("critical-service", false, 500*time.Millisecond, "Service down")
//	
//	// Check current degradation level
//	level := dm.GetCurrentDegradationLevel()
//
// # Error Alerting
//
// The alerting system generates and routes alerts based on error patterns
// and system health changes.
//
//	am := resilience.NewAlertManager()
//	am.AddHandler(resilience.NewLoggingAlertHandler())
//	
//	eag := resilience.NewErrorAlertGenerator(am)
//	eag.HandleError(ctx, err, "service-name", metadata)
//
// # Combined Usage
//
// For maximum resilience, combine all patterns using RetryableOperation:
//
//	op := resilience.NewRetryableOperation("service-name", cbConfig, retryConfig)
//	result, err := op.Execute(ctx, func(ctx context.Context) (interface{}, error) {
//		return externalService.Call(ctx, data)
//	})
//
// The package is designed to be thread-safe and can handle high-concurrency
// scenarios typical in distributed systems.
package resilience