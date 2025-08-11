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

// Mock alert handler for testing
type mockAlertHandler struct {
	name   string
	alerts []Alert
	fail   bool
}

func (m *mockAlertHandler) HandleAlert(ctx context.Context, alert Alert) error {
	if m.fail {
		return errors.New("handler failed")
	}
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *mockAlertHandler) Name() string {
	return m.name
}

func TestAlertManager_AddHandler(t *testing.T) {
	am := NewAlertManager()
	handler := &mockAlertHandler{name: "test-handler"}

	am.AddHandler(handler)

	assert.Len(t, am.handlers, 1)
	assert.Equal(t, "test-handler", am.handlers[0].Name())
}

func TestAlertManager_SendAlert(t *testing.T) {
	am := NewAlertManager()
	handler := &mockAlertHandler{name: "test-handler"}
	am.AddHandler(handler)

	alert := Alert{
		Severity:    SeverityError,
		Title:       "Test Alert",
		Description: "Test description",
		Source:      "test-source",
		Tags: map[string]string{
			"component": "test",
		},
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	err := am.SendAlert(context.Background(), alert)
	require.NoError(t, err)

	require.Len(t, handler.alerts, 1)
	receivedAlert := handler.alerts[0]
	assert.Equal(t, SeverityError, receivedAlert.Severity)
	assert.Equal(t, "Test Alert", receivedAlert.Title)
	assert.Equal(t, "Test description", receivedAlert.Description)
	assert.Equal(t, "test-source", receivedAlert.Source)
	assert.NotEmpty(t, receivedAlert.ID)
	assert.False(t, receivedAlert.Timestamp.IsZero())
}

func TestAlertManager_SendAlert_HandlerFailure(t *testing.T) {
	am := NewAlertManager()
	
	successHandler := &mockAlertHandler{name: "success-handler"}
	failHandler := &mockAlertHandler{name: "fail-handler", fail: true}
	
	am.AddHandler(successHandler)
	am.AddHandler(failHandler)

	alert := Alert{
		Severity: SeverityError,
		Title:    "Test Alert",
		Source:   "test-source",
	}

	err := am.SendAlert(context.Background(), alert)
	require.NoError(t, err) // Should succeed because one handler succeeded

	assert.Len(t, successHandler.alerts, 1)
	assert.Len(t, failHandler.alerts, 0)
}

func TestAlertManager_SendAlert_AllHandlersFail(t *testing.T) {
	am := NewAlertManager()
	
	failHandler1 := &mockAlertHandler{name: "fail-handler-1", fail: true}
	failHandler2 := &mockAlertHandler{name: "fail-handler-2", fail: true}
	
	am.AddHandler(failHandler1)
	am.AddHandler(failHandler2)

	alert := Alert{
		Severity: SeverityError,
		Title:    "Test Alert",
		Source:   "test-source",
	}

	err := am.SendAlert(context.Background(), alert)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all alert handlers failed")
}

func TestAlertManager_RateLimit(t *testing.T) {
	am := NewAlertManager()
	am.rateLimit = 2 // Set low rate limit for testing
	
	handler := &mockAlertHandler{name: "test-handler"}
	am.AddHandler(handler)

	alert := Alert{
		Severity: SeverityError,
		Title:    "Test Alert",
		Source:   "test-source",
	}

	// First two alerts should succeed
	err := am.SendAlert(context.Background(), alert)
	require.NoError(t, err)
	
	err = am.SendAlert(context.Background(), alert)
	require.NoError(t, err)

	// Third alert should be rate limited
	err = am.SendAlert(context.Background(), alert)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit exceeded")

	assert.Len(t, handler.alerts, 2)
}

func TestLoggingAlertHandler(t *testing.T) {
	handler := NewLoggingAlertHandler()

	alert := Alert{
		ID:          "test-alert-1",
		Severity:    SeverityWarning,
		Title:       "Test Alert",
		Description: "Test description",
		Source:      "test-source",
		Tags: map[string]string{
			"component": "test",
		},
		Metadata: map[string]interface{}{
			"key": "value",
		},
		Timestamp: time.Now(),
	}

	err := handler.HandleAlert(context.Background(), alert)
	require.NoError(t, err)

	assert.Equal(t, "logging", handler.Name())
}

func TestErrorAlertGenerator_HandleError(t *testing.T) {
	am := NewAlertManager()
	handler := &mockAlertHandler{name: "test-handler"}
	am.AddHandler(handler)

	eag := NewErrorAlertGenerator(am)

	// Test timeout error
	timeoutErr := appErrors.NewTimeoutError("operation timed out")
	eag.HandleError(context.Background(), timeoutErr, "test-service", map[string]interface{}{
		"operation": "scan",
	})

	require.Len(t, handler.alerts, 1)
	alert := handler.alerts[0]
	assert.Equal(t, SeverityWarning, alert.Severity)
	assert.Equal(t, "Operation Timeout", alert.Title)
	assert.Equal(t, "test-service", alert.Source)
	assert.Equal(t, "timeout", alert.Tags["error_type"])

	// Test internal error
	handler.alerts = nil // Reset
	internalErr := appErrors.NewInternalError("internal error")
	eag.HandleError(context.Background(), internalErr, "test-service", nil)

	require.Len(t, handler.alerts, 1)
	alert = handler.alerts[0]
	assert.Equal(t, SeverityError, alert.Severity)
	assert.Equal(t, "Internal System Error", alert.Title)
}

func TestErrorAlertGenerator_DetermineSeverity(t *testing.T) {
	eag := NewErrorAlertGenerator(nil)

	tests := []struct {
		name     string
		err      error
		expected AlertSeverity
	}{
		{"timeout error", appErrors.NewTimeoutError("timeout"), SeverityWarning},
		{"external error", appErrors.NewExternalError("service", "error"), SeverityWarning},
		{"internal error", appErrors.NewInternalError("internal"), SeverityError},
		{"validation error", appErrors.NewValidationError("validation"), SeverityInfo},
		{"authentication error", appErrors.NewAuthenticationError("auth"), SeverityWarning},
		{"authorization error", appErrors.NewAuthorizationError("authz"), SeverityWarning},
		{"circuit breaker error", &CircuitBreakerError{Name: "test", State: StateOpen}, SeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := eag.determineSeverity(tt.err)
			assert.Equal(t, tt.expected, severity)
		})
	}
}

func TestSystemHealthMonitor(t *testing.T) {
	am := NewAlertManager()
	handler := &mockAlertHandler{name: "test-handler"}
	am.AddHandler(handler)

	dm := NewDegradationManager()
	dm.RegisterService("service1", LevelPartial)
	dm.RegisterService("service2", LevelSevere)

	shm := NewSystemHealthMonitor(am, dm)
	shm.checkInterval = 10 * time.Millisecond // Fast interval for testing

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	shm.Start(ctx)
	defer shm.Stop()

	// Make service1 unhealthy to trigger degradation
	for i := 0; i < 3; i++ {
		dm.UpdateServiceHealth("service1", false, 0, "Error")
	}

	// Wait for monitor to detect the change
	time.Sleep(50 * time.Millisecond)

	// Should have received degradation alert
	found := false
	for _, alert := range handler.alerts {
		if alert.Title == "System Degradation Level Changed" {
			found = true
			assert.Equal(t, SeverityWarning, alert.Severity)
			assert.Equal(t, "system_health_monitor", alert.Source)
			break
		}
	}
	assert.True(t, found, "Should have received degradation alert")
}

func TestAlertSeverity_String(t *testing.T) {
	tests := []struct {
		severity AlertSeverity
		expected string
	}{
		{SeverityInfo, "INFO"},
		{SeverityWarning, "WARNING"},
		{SeverityError, "ERROR"},
		{SeverityCritical, "CRITICAL"},
		{AlertSeverity(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.String())
		})
	}
}

func TestSystemHealthMonitor_StartStop(t *testing.T) {
	am := NewAlertManager()
	dm := NewDegradationManager()
	shm := NewSystemHealthMonitor(am, dm)

	// Test multiple starts (should be safe)
	ctx := context.Background()
	shm.Start(ctx)
	shm.Start(ctx) // Should not panic or create multiple goroutines

	assert.True(t, shm.running)

	// Test stop
	shm.Stop()
	assert.False(t, shm.running)

	// Test multiple stops (should be safe)
	shm.Stop() // Should not panic
}

func TestErrorAlertGenerator_NilError(t *testing.T) {
	am := NewAlertManager()
	handler := &mockAlertHandler{name: "test-handler"}
	am.AddHandler(handler)

	eag := NewErrorAlertGenerator(am)

	// Should not generate alert for nil error
	eag.HandleError(context.Background(), nil, "test-service", nil)

	assert.Len(t, handler.alerts, 0)
}