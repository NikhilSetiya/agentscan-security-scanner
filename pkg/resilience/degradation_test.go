package resilience

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDegradationManager_RegisterService(t *testing.T) {
	dm := NewDegradationManager()

	dm.RegisterService("test-service", LevelPartial)

	health, exists := dm.GetServiceHealth("test-service")
	require.True(t, exists)
	assert.Equal(t, "test-service", health.Name)
	assert.True(t, health.Healthy)
	assert.Equal(t, 0, health.ErrorCount)
}

func TestDegradationManager_UpdateServiceHealth(t *testing.T) {
	dm := NewDegradationManager()
	dm.RegisterService("test-service", LevelPartial)

	// Update with healthy status
	dm.UpdateServiceHealth("test-service", true, 100*time.Millisecond, "OK")

	health, exists := dm.GetServiceHealth("test-service")
	require.True(t, exists)
	assert.True(t, health.Healthy)
	assert.Equal(t, 0, health.ErrorCount)
	assert.Equal(t, 100*time.Millisecond, health.ResponseTime)
	assert.Equal(t, "OK", health.Message)

	// Update with unhealthy status (should not mark as unhealthy immediately)
	dm.UpdateServiceHealth("test-service", false, 500*time.Millisecond, "Error")

	health, exists = dm.GetServiceHealth("test-service")
	require.True(t, exists)
	assert.True(t, health.Healthy) // Still healthy because error count < threshold
	assert.Equal(t, 1, health.ErrorCount)

	// More failures should mark as unhealthy
	dm.UpdateServiceHealth("test-service", false, 500*time.Millisecond, "Error")
	dm.UpdateServiceHealth("test-service", false, 500*time.Millisecond, "Error")

	health, exists = dm.GetServiceHealth("test-service")
	require.True(t, exists)
	assert.False(t, health.Healthy) // Now unhealthy
	assert.Equal(t, 3, health.ErrorCount)
}

func TestDegradationManager_GetCurrentDegradationLevel(t *testing.T) {
	dm := NewDegradationManager()

	// Initially normal
	assert.Equal(t, LevelNormal, dm.GetCurrentDegradationLevel())

	// Register services with different degradation levels
	dm.RegisterService("critical-service", LevelCritical)
	dm.RegisterService("partial-service", LevelPartial)
	dm.RegisterService("normal-service", LevelNormal)

	// All healthy - should be normal
	assert.Equal(t, LevelNormal, dm.GetCurrentDegradationLevel())

	// Make partial service unhealthy
	for i := 0; i < 3; i++ {
		dm.UpdateServiceHealth("partial-service", false, 0, "Error")
	}
	assert.Equal(t, LevelPartial, dm.GetCurrentDegradationLevel())

	// Make critical service unhealthy - should escalate to critical
	for i := 0; i < 3; i++ {
		dm.UpdateServiceHealth("critical-service", false, 0, "Error")
	}
	assert.Equal(t, LevelCritical, dm.GetCurrentDegradationLevel())

	// Heal critical service - should go back to partial
	dm.UpdateServiceHealth("critical-service", true, 100*time.Millisecond, "OK")
	assert.Equal(t, LevelPartial, dm.GetCurrentDegradationLevel())
}

func TestDegradationManager_GetHealthyServices(t *testing.T) {
	dm := NewDegradationManager()

	dm.RegisterService("service1", LevelNormal)
	dm.RegisterService("service2", LevelPartial)
	dm.RegisterService("service3", LevelSevere)

	// All should be healthy initially
	healthy := dm.GetHealthyServices()
	assert.Len(t, healthy, 3)
	assert.Contains(t, healthy, "service1")
	assert.Contains(t, healthy, "service2")
	assert.Contains(t, healthy, "service3")

	// Make service2 unhealthy
	for i := 0; i < 3; i++ {
		dm.UpdateServiceHealth("service2", false, 0, "Error")
	}

	healthy = dm.GetHealthyServices()
	assert.Len(t, healthy, 2)
	assert.Contains(t, healthy, "service1")
	assert.Contains(t, healthy, "service3")
	assert.NotContains(t, healthy, "service2")

	unhealthy := dm.GetUnhealthyServices()
	assert.Len(t, unhealthy, 1)
	assert.Contains(t, unhealthy, "service2")
}

func TestDegradationManager_PercentageBasedDegradation(t *testing.T) {
	dm := NewDegradationManager()

	// Register 4 services, all with normal degradation level
	for i := 1; i <= 4; i++ {
		dm.RegisterService(fmt.Sprintf("service%d", i), LevelNormal)
	}

	// Make 25% unhealthy (1 out of 4) - should be partial
	for i := 0; i < 3; i++ {
		dm.UpdateServiceHealth("service1", false, 0, "Error")
	}
	assert.Equal(t, LevelPartial, dm.GetCurrentDegradationLevel())

	// Make 50% unhealthy (2 out of 4) - should be severe
	for i := 0; i < 3; i++ {
		dm.UpdateServiceHealth("service2", false, 0, "Error")
	}
	assert.Equal(t, LevelSevere, dm.GetCurrentDegradationLevel())

	// Make 75% unhealthy (3 out of 4) - should be critical
	for i := 0; i < 3; i++ {
		dm.UpdateServiceHealth("service3", false, 0, "Error")
	}
	assert.Equal(t, LevelCritical, dm.GetCurrentDegradationLevel())
}

func TestAgentDegradationHandler_RegisterAgent(t *testing.T) {
	adh := NewAgentDegradationHandler(2)

	adh.RegisterAgent("semgrep", LevelPartial, []string{"bandit"})
	adh.RegisterAgent("bandit", LevelNormal, nil)

	assert.True(t, adh.degradationManager.IsServiceHealthy("semgrep"))
	assert.True(t, adh.degradationManager.IsServiceHealthy("bandit"))
}

func TestAgentDegradationHandler_GetAvailableAgents(t *testing.T) {
	adh := NewAgentDegradationHandler(2)

	adh.RegisterAgent("semgrep", LevelPartial, []string{"bandit"})
	adh.RegisterAgent("bandit", LevelNormal, nil)
	adh.RegisterAgent("eslint", LevelNormal, nil)

	// All agents healthy
	agents, err := adh.GetAvailableAgents([]string{"semgrep", "bandit", "eslint"})
	require.NoError(t, err)
	assert.Len(t, agents, 3)
	assert.Contains(t, agents, "semgrep")
	assert.Contains(t, agents, "bandit")
	assert.Contains(t, agents, "eslint")

	// Make semgrep unhealthy - should use fallback
	for i := 0; i < 3; i++ {
		adh.UpdateAgentHealth("semgrep", false, 0, "Error")
	}

	agents, err = adh.GetAvailableAgents([]string{"semgrep", "eslint"})
	require.NoError(t, err)
	assert.Len(t, agents, 2)
	assert.Contains(t, agents, "bandit") // fallback for semgrep
	assert.Contains(t, agents, "eslint")
	assert.NotContains(t, agents, "semgrep")

	// Make too many agents unhealthy
	for i := 0; i < 3; i++ {
		adh.UpdateAgentHealth("bandit", false, 0, "Error")
		adh.UpdateAgentHealth("eslint", false, 0, "Error")
	}

	agents, err = adh.GetAvailableAgents([]string{"semgrep", "bandit", "eslint"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient healthy agents")
	assert.Len(t, agents, 0) // No healthy agents available
}

func TestAgentDegradationHandler_CanPerformScan(t *testing.T) {
	adh := NewAgentDegradationHandler(2)

	adh.RegisterAgent("agent1", LevelNormal, nil)
	adh.RegisterAgent("agent2", LevelPartial, nil)

	// Normal level - all scans allowed
	canScan, message := adh.CanPerformScan("full")
	assert.True(t, canScan)
	assert.Empty(t, message)

	// Make agent2 unhealthy - partial degradation
	for i := 0; i < 3; i++ {
		adh.UpdateAgentHealth("agent2", false, 0, "Error")
	}

	canScan, message = adh.CanPerformScan("incremental")
	assert.True(t, canScan)
	assert.Contains(t, message, "reduced agent set")

	canScan, message = adh.CanPerformScan("full")
	assert.False(t, canScan)
	assert.Contains(t, message, "disabled during partial degradation")

	// Make agent1 unhealthy too - severe degradation
	for i := 0; i < 3; i++ {
		adh.UpdateAgentHealth("agent1", false, 0, "Error")
	}

	canScan, message = adh.CanPerformScan("basic")
	assert.True(t, canScan)
	assert.Contains(t, message, "minimal agent set")

	canScan, message = adh.CanPerformScan("comprehensive")
	assert.False(t, canScan)
	assert.Contains(t, message, "only basic scans")
}

func TestAgentDegradationHandler_GetDegradationStatus(t *testing.T) {
	adh := NewAgentDegradationHandler(2)

	adh.RegisterAgent("agent1", LevelNormal, nil)
	adh.RegisterAgent("agent2", LevelPartial, nil)

	status := adh.GetDegradationStatus()
	assert.Equal(t, "NORMAL", status["degradation_level"])
	assert.Equal(t, 2, len(status["healthy_agents"].([]string)))
	assert.Equal(t, 0, len(status["unhealthy_agents"].([]string)))
	assert.Equal(t, 2, status["total_agents"])
	assert.True(t, status["can_scan"].(bool))

	// Make one agent unhealthy
	for i := 0; i < 3; i++ {
		adh.UpdateAgentHealth("agent1", false, 0, "Error")
	}

	status = adh.GetDegradationStatus()
	// With 1 out of 2 agents unhealthy (50%), it should be severe degradation
	assert.Equal(t, "SEVERE", status["degradation_level"])
	assert.Equal(t, 1, len(status["healthy_agents"].([]string)))
	assert.Equal(t, 1, len(status["unhealthy_agents"].([]string)))
	assert.False(t, status["can_scan"].(bool)) // Below minimum required agents
}

func TestDegradationLevel_String(t *testing.T) {
	tests := []struct {
		level    DegradationLevel
		expected string
	}{
		{LevelNormal, "NORMAL"},
		{LevelPartial, "PARTIAL"},
		{LevelSevere, "SEVERE"},
		{LevelCritical, "CRITICAL"},
		{DegradationLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}