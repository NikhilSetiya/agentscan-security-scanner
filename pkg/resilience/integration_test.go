package resilience

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	appErrors "github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockExternalService simulates an external service that can fail
type MockExternalService struct {
	name           string
	failureRate    float64
	responseTime   time.Duration
	requestCount   int
	failureCount   int
	mutex          sync.Mutex
	forceFailure   bool
	circuitBreaker *CircuitBreaker
}

func NewMockExternalService(name string, failureRate float64, responseTime time.Duration) *MockExternalService {
	service := &MockExternalService{
		name:         name,
		failureRate:  failureRate,
		responseTime: responseTime,
	}

	// Create circuit breaker for this service
	service.circuitBreaker = NewCircuitBreaker(CircuitBreakerConfig{
		Name:        fmt.Sprintf("cb-%s", name),
		MaxRequests: 3,
		Interval:    5 * time.Second,
		Timeout:     2 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.Requests >= 5 && counts.TotalFailures >= uint32(float64(counts.Requests)*0.6)
		},
	})

	return service
}

func (m *MockExternalService) Call(ctx context.Context, data string) (string, error) {
	result, err := m.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return m.doCall(ctx, data)
	})
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

func (m *MockExternalService) doCall(ctx context.Context, data string) (interface{}, error) {
	m.mutex.Lock()
	m.requestCount++
	requestNum := m.requestCount
	m.mutex.Unlock()

	// Simulate response time
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(m.responseTime):
	}

	// Determine if this request should fail
	shouldFail := m.forceFailure || (float64(requestNum%100) < m.failureRate*100)

	if shouldFail {
		m.mutex.Lock()
		m.failureCount++
		m.mutex.Unlock()
		return nil, appErrors.NewExternalError(m.name, fmt.Sprintf("simulated failure for request %d", requestNum))
	}

	return fmt.Sprintf("success-%s-%d", data, requestNum), nil
}

func (m *MockExternalService) SetForceFailure(force bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.forceFailure = force
}

func (m *MockExternalService) GetStats() (int, int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.requestCount, m.failureCount
}

func (m *MockExternalService) GetCircuitBreakerState() CircuitState {
	return m.circuitBreaker.State()
}

// TestIntegration_ErrorHandlingWorkflow tests the complete error handling workflow
func TestIntegration_ErrorHandlingWorkflow(t *testing.T) {
	// Setup components
	alertManager := NewAlertManager()
	alertHandler := &mockAlertHandler{name: "integration-test"}
	alertManager.AddHandler(alertHandler)

	errorAlertGenerator := NewErrorAlertGenerator(alertManager)

	degradationManager := NewDegradationManager()
	agentDegradationHandler := NewAgentDegradationHandler(2)

	// Register services
	services := []string{"semgrep", "bandit", "eslint"}
	mockServices := make(map[string]*MockExternalService)

	for _, serviceName := range services {
		degradationManager.RegisterService(serviceName, LevelPartial)
		agentDegradationHandler.RegisterAgent(serviceName, LevelPartial, nil)
		mockServices[serviceName] = NewMockExternalService(serviceName, 0.1, 50*time.Millisecond)
	}

	// Create retryable operations for each service
	retryableOps := make(map[string]*RetryableOperation)
	for _, serviceName := range services {
		cbConfig := CircuitBreakerConfig{
			Name:        fmt.Sprintf("cb-%s", serviceName),
			MaxRequests: 3,
			Interval:    time.Second,
			Timeout:     500 * time.Millisecond,
		}
		retryConfig := DefaultRetryConfig()
		retryConfig.MaxAttempts = 3
		retryConfig.InitialDelay = 10 * time.Millisecond

		retryableOps[serviceName] = NewRetryableOperation(serviceName, cbConfig, retryConfig)
	}

	ctx := context.Background()

	// Phase 1: Normal operation
	t.Run("Phase1_NormalOperation", func(t *testing.T) {
		for _, serviceName := range services {
			service := mockServices[serviceName]
			op := retryableOps[serviceName]

			result, err := op.Execute(ctx, func(ctx context.Context) (interface{}, error) {
				return service.Call(ctx, "test-data")
			})

			require.NoError(t, err)
			assert.Contains(t, result.(string), "success")
			assert.Equal(t, StateClosed, op.State())

			// Update service health
			agentDegradationHandler.UpdateAgentHealth(serviceName, true, 50*time.Millisecond, "OK")
		}

		// Check degradation level
		assert.Equal(t, LevelNormal, agentDegradationHandler.degradationManager.GetCurrentDegradationLevel())

		// Should be able to perform all scan types
		canScan, _ := agentDegradationHandler.CanPerformScan("full")
		assert.True(t, canScan)
	})

	// Phase 2: Introduce failures in one service
	t.Run("Phase2_SingleServiceFailure", func(t *testing.T) {
		// Make semgrep service fail
		mockServices["semgrep"].SetForceFailure(true)

		// Try to call the failing service multiple times
		for i := 0; i < 10; i++ {
			_, err := retryableOps["semgrep"].Execute(ctx, func(ctx context.Context) (interface{}, error) {
				return mockServices["semgrep"].Call(ctx, "test-data")
			})

			if err != nil {
				errorAlertGenerator.HandleError(ctx, err, "semgrep", map[string]interface{}{
					"attempt": i + 1,
				})
				agentDegradationHandler.UpdateAgentHealth("semgrep", false, 500*time.Millisecond, err.Error())
			}
		}

		// Circuit breaker should be open
		assert.Equal(t, StateOpen, retryableOps["semgrep"].State())

		// System should be in partial degradation
		assert.Equal(t, LevelPartial, agentDegradationHandler.degradationManager.GetCurrentDegradationLevel())

		// Should have received error alerts
		assert.Greater(t, len(alertHandler.alerts), 0)

		// Check available agents
		availableAgents, err := agentDegradationHandler.GetAvailableAgents(services)
		require.NoError(t, err)
		assert.Len(t, availableAgents, 2) // semgrep should be excluded
		assert.NotContains(t, availableAgents, "semgrep")
	})

	// Phase 3: Multiple service failures
	t.Run("Phase3_MultipleServiceFailures", func(t *testing.T) {
		// Make bandit service fail too
		mockServices["bandit"].SetForceFailure(true)

		for i := 0; i < 5; i++ {
			_, err := retryableOps["bandit"].Execute(ctx, func(ctx context.Context) (interface{}, error) {
				return mockServices["bandit"].Call(ctx, "test-data")
			})

			if err != nil {
				errorAlertGenerator.HandleError(ctx, err, "bandit", nil)
				agentDegradationHandler.UpdateAgentHealth("bandit", false, 500*time.Millisecond, err.Error())
			}
		}

		// Should have insufficient healthy agents
		availableAgents, err := agentDegradationHandler.GetAvailableAgents(services)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient healthy agents")
		assert.Len(t, availableAgents, 1) // Only eslint is healthy

		// Should not be able to perform full scans
		canScan, message := agentDegradationHandler.CanPerformScan("full")
		assert.False(t, canScan)
		assert.Contains(t, message, "disabled")
	})

	// Phase 4: Recovery
	t.Run("Phase4_Recovery", func(t *testing.T) {
		// Fix semgrep service
		mockServices["semgrep"].SetForceFailure(false)

		// Wait for circuit breaker timeout
		time.Sleep(600 * time.Millisecond)

		// Try calling semgrep again - should succeed after circuit breaker opens
		var lastErr error
		for i := 0; i < 5; i++ {
			result, err := retryableOps["semgrep"].Execute(ctx, func(ctx context.Context) (interface{}, error) {
				return mockServices["semgrep"].Call(ctx, "recovery-test")
			})

			if err == nil {
				assert.Contains(t, result.(string), "success")
				agentDegradationHandler.UpdateAgentHealth("semgrep", true, 50*time.Millisecond, "Recovered")
				break
			}
			lastErr = err
			time.Sleep(100 * time.Millisecond)
		}

		// Should eventually succeed
		assert.NoError(t, lastErr)

		// Circuit breaker should be closed again
		assert.Equal(t, StateClosed, retryableOps["semgrep"].State())

		// Should have enough agents again
		availableAgents, err := agentDegradationHandler.GetAvailableAgents([]string{"semgrep", "eslint"})
		require.NoError(t, err)
		assert.Len(t, availableAgents, 2)
	})

	// Verify alert generation
	t.Run("VerifyAlerts", func(t *testing.T) {
		// Should have received multiple alerts
		assert.Greater(t, len(alertHandler.alerts), 5)

		// Check for different types of alerts
		hasExternalAlert := false

		for _, alert := range alertHandler.alerts {
			switch alert.Tags["error_type"] {
			case "external":
				hasExternalAlert = true
			}
		}

		assert.True(t, hasExternalAlert, "Should have external service error alerts")
	})
}

// TestIntegration_ConcurrentFailures tests error handling under concurrent load
func TestIntegration_ConcurrentFailures(t *testing.T) {
	service := NewMockExternalService("concurrent-test", 0.3, 10*time.Millisecond)

	cbConfig := CircuitBreakerConfig{
		Name:        "concurrent-cb",
		MaxRequests: 5,
		Interval:    time.Second,
		Timeout:     100 * time.Millisecond,
	}
	retryConfig := DefaultRetryConfig()
	retryConfig.MaxAttempts = 2
	retryConfig.InitialDelay = 5 * time.Millisecond

	op := NewRetryableOperation("concurrent-test", cbConfig, retryConfig)

	const numGoroutines = 50
	const requestsPerGoroutine = 10

	var wg sync.WaitGroup
	successCount := int64(0)
	errorCount := int64(0)
	var mutex sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				_, err := op.Execute(ctx, func(ctx context.Context) (interface{}, error) {
					return service.Call(ctx, fmt.Sprintf("g%d-r%d", goroutineID, j))
				})

				mutex.Lock()
				if err != nil {
					errorCount++
				} else {
					successCount++
				}
				mutex.Unlock()

				// Small delay between requests
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	totalRequests := int64(numGoroutines * requestsPerGoroutine)
	t.Logf("Total requests: %d, Successes: %d, Errors: %d", totalRequests, successCount, errorCount)
	t.Logf("Circuit breaker state: %s", op.State())
	t.Logf("Circuit breaker counts: %+v", op.Counts())

	serviceRequests, serviceFailures := service.GetStats()
	t.Logf("Service stats - Requests: %d, Failures: %d", serviceRequests, serviceFailures)

	// Verify that we handled the load without panics
	assert.Equal(t, totalRequests, successCount+errorCount)
	assert.Greater(t, successCount, int64(0), "Should have some successful requests")

	// Circuit breaker should have activated if there were enough failures
	if errorCount > totalRequests/2 {
		assert.NotEqual(t, StateClosed, op.State(), "Circuit breaker should be open with high failure rate")
	}
}

// TestIntegration_GracefulDegradation tests the complete graceful degradation workflow
func TestIntegration_GracefulDegradation(t *testing.T) {
	// Setup alert system
	alertManager := NewAlertManager()
	alertHandler := &mockAlertHandler{name: "degradation-test"}
	alertManager.AddHandler(alertHandler)

	// Setup degradation management
	degradationManager := NewDegradationManager()
	healthMonitor := NewSystemHealthMonitor(alertManager, degradationManager)

	// Register critical and non-critical services
	criticalServices := []string{"auth-service", "database"}
	nonCriticalServices := []string{"cache", "metrics"}

	for _, service := range criticalServices {
		degradationManager.RegisterService(service, LevelCritical)
	}
	for _, service := range nonCriticalServices {
		degradationManager.RegisterService(service, LevelPartial)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	healthMonitor.Start(ctx)
	defer healthMonitor.Stop()

	// Phase 1: All services healthy
	assert.Equal(t, LevelNormal, degradationManager.GetCurrentDegradationLevel())

	// Phase 2: Non-critical service fails
	for i := 0; i < 3; i++ {
		degradationManager.UpdateServiceHealth("cache", false, 0, "Cache connection failed")
	}
	time.Sleep(50 * time.Millisecond) // Allow monitor to detect

	assert.Equal(t, LevelPartial, degradationManager.GetCurrentDegradationLevel())

	// Phase 3: Critical service fails
	for i := 0; i < 3; i++ {
		degradationManager.UpdateServiceHealth("auth-service", false, 0, "Auth service down")
	}
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, LevelCritical, degradationManager.GetCurrentDegradationLevel())

	// Phase 4: Recovery
	degradationManager.UpdateServiceHealth("auth-service", true, 100*time.Millisecond, "Auth service recovered")
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, LevelPartial, degradationManager.GetCurrentDegradationLevel())

	degradationManager.UpdateServiceHealth("cache", true, 50*time.Millisecond, "Cache reconnected")
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, LevelNormal, degradationManager.GetCurrentDegradationLevel())

	// Verify alerts were generated
	assert.Greater(t, len(alertHandler.alerts), 0)

	// Check for degradation level change alerts
	foundDegradationAlerts := 0
	for _, alert := range alertHandler.alerts {
		if alert.Title == "System Degradation Level Changed" {
			foundDegradationAlerts++
		}
	}
	assert.Greater(t, foundDegradationAlerts, 0, "Should have received degradation level change alerts")
}