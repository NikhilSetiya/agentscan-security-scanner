# Resilience Package

This package provides comprehensive error handling, circuit breaker patterns, retry logic, and graceful degradation capabilities for the AgentScan security scanner system.

## Components Implemented

### 1. Circuit Breaker Pattern (`circuit_breaker.go`)

Implements the circuit breaker pattern to prevent cascading failures by monitoring the failure rate of external service calls.

**Features:**
- Three states: Closed, Open, Half-Open
- Configurable failure thresholds and timeouts
- Automatic state transitions based on success/failure rates
- Request counting and statistics
- Panic recovery

**Usage:**
```go
cb := NewCircuitBreaker(CircuitBreakerConfig{
    Name:        "external-service",
    MaxRequests: 3,
    Interval:    time.Minute,
    Timeout:     30 * time.Second,
})

result, err := cb.Execute(ctx, func(ctx context.Context) (interface{}, error) {
    return externalService.Call(ctx, data)
})
```

### 2. Retry with Exponential Backoff (`retry.go`)

Provides automatic retry logic with exponential backoff and jitter to handle transient failures.

**Features:**
- Configurable maximum attempts and delays
- Exponential backoff with jitter
- Custom retryable error detection
- Context cancellation support
- Retry callbacks for monitoring

**Usage:**
```go
retrier := NewRetrier(DefaultRetryConfig())
err := retrier.Execute(ctx, func(ctx context.Context) error {
    return riskyOperation(ctx)
})
```

### 3. Graceful Degradation (`degradation.go`)

Monitors service health and automatically adjusts system behavior based on degradation levels.

**Features:**
- Four degradation levels: Normal, Partial, Severe, Critical
- Service health monitoring with error thresholds
- Automatic degradation level calculation
- Agent fallback mechanisms
- Scan capability assessment

**Usage:**
```go
dm := NewDegradationManager()
dm.RegisterService("critical-service", LevelCritical)

// Update service health
dm.UpdateServiceHealth("critical-service", false, 500*time.Millisecond, "Service down")

// Check current degradation level
level := dm.GetCurrentDegradationLevel()
```

### 4. Error Alerting (`alerting.go`)

Generates and routes alerts based on error patterns and system health changes.

**Features:**
- Multiple alert severity levels
- Pluggable alert handlers
- Rate limiting to prevent alert storms
- Error classification and alert generation
- System health monitoring with automatic alerts

**Usage:**
```go
am := NewAlertManager()
am.AddHandler(NewLoggingAlertHandler())

eag := NewErrorAlertGenerator(am)
eag.HandleError(ctx, err, "service-name", metadata)
```

### 5. Combined Resilience (`RetryableOperation`)

Combines circuit breaker and retry patterns for maximum resilience.

**Usage:**
```go
op := NewRetryableOperation("service-name", cbConfig, retryConfig)
result, err := op.Execute(ctx, func(ctx context.Context) (interface{}, error) {
    return externalService.Call(ctx, data)
})
```

## Integration with AgentScan

The resilience package is designed to integrate seamlessly with the AgentScan system:

1. **Agent Failure Handling**: Circuit breakers protect against failing security scanning agents
2. **External Service Resilience**: Retry logic handles transient failures with Git providers, notification services, etc.
3. **Graceful Degradation**: System continues operating with reduced functionality when agents fail
4. **Comprehensive Alerting**: Operations teams are notified of system health issues

## Testing

The package includes comprehensive tests:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test complete error handling workflows
- **Concurrent Tests**: Verify thread safety under load
- **Failure Scenario Tests**: Test various failure modes and recovery

Run tests with:
```bash
go test ./pkg/resilience/... -v
```

## Requirements Satisfied

This implementation satisfies the following requirements from the AgentScan specification:

- **10.1**: Individual agent failures are handled gracefully with circuit breakers
- **10.2**: Automatic retry logic with exponential backoff for transient failures
- **10.4**: Comprehensive error logging and alerting system
- **10.5**: System maintains high availability through graceful degradation

The system can continue operating even when individual components fail, providing a robust foundation for the AgentScan security scanning platform.