package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

func TestNewAgentManager(t *testing.T) {
	am := NewAgentManager()
	
	assert.NotNil(t, am)
	assert.NotNil(t, am.agents)
	assert.NotNil(t, am.configs)
	assert.NotNil(t, am.health)
	assert.Equal(t, 0, len(am.agents))
}

func TestAgentManager_RegisterAgent_Success(t *testing.T) {
	am := NewAgentManager()
	mockAgent := &MockAgent{}
	
	config := agent.AgentConfig{
		Name:           "test-agent",
		Version:        "1.0.0",
		SupportedLangs: []string{"javascript", "typescript"},
		Categories:     []agent.VulnCategory{agent.CategoryXSS, agent.CategorySQLInjection},
		RequiresDocker: true,
	}
	
	mockAgent.On("GetConfig").Return(config)
	
	err := am.RegisterAgent("test-agent", mockAgent)
	
	require.NoError(t, err)
	assert.Equal(t, 1, len(am.agents))
	assert.Equal(t, 1, len(am.configs))
	assert.Equal(t, 1, len(am.health))
	
	// Verify agent is registered
	registeredAgent, err := am.GetAgent("test-agent")
	require.NoError(t, err)
	assert.Equal(t, mockAgent, registeredAgent)
	
	// Verify config is stored
	registeredConfig, err := am.GetAgentConfig("test-agent")
	require.NoError(t, err)
	assert.Equal(t, config, registeredConfig)
	
	// Verify health is initialized
	health, err := am.GetAgentHealth("test-agent")
	require.NoError(t, err)
	assert.Equal(t, AgentStatusUnknown, health.Status)
	assert.Equal(t, int64(0), health.CheckCount)
	assert.Equal(t, int64(0), health.FailureCount)
	
	mockAgent.AssertExpectations(t)
}

func TestAgentManager_RegisterAgent_ValidationErrors(t *testing.T) {
	am := NewAgentManager()
	mockAgent := &MockAgent{}
	
	tests := []struct {
		name      string
		agentName string
		agent     agent.SecurityAgent
		wantError string
	}{
		{
			name:      "empty name",
			agentName: "",
			agent:     mockAgent,
			wantError: "agent name cannot be empty",
		},
		{
			name:      "nil agent",
			agentName: "test-agent",
			agent:     nil,
			wantError: "agent cannot be nil",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := am.RegisterAgent(tt.agentName, tt.agent)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestAgentManager_RegisterAgent_DuplicateAgent(t *testing.T) {
	am := NewAgentManager()
	mockAgent := &MockAgent{}
	
	config := agent.AgentConfig{Name: "test-agent"}
	mockAgent.On("GetConfig").Return(config)
	
	// Register agent first time
	err := am.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)
	
	// Try to register same agent again
	err = am.RegisterAgent("test-agent", mockAgent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestAgentManager_UnregisterAgent_Success(t *testing.T) {
	am := NewAgentManager()
	mockAgent := &MockAgent{}
	
	config := agent.AgentConfig{Name: "test-agent"}
	mockAgent.On("GetConfig").Return(config)
	
	// Register agent
	err := am.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)
	
	// Unregister agent
	err = am.UnregisterAgent("test-agent")
	require.NoError(t, err)
	
	// Verify agent is removed
	assert.Equal(t, 0, len(am.agents))
	assert.Equal(t, 0, len(am.configs))
	assert.Equal(t, 0, len(am.health))
	
	// Verify agent is not found
	_, err = am.GetAgent("test-agent")
	assert.Error(t, err)
}

func TestAgentManager_UnregisterAgent_NotFound(t *testing.T) {
	am := NewAgentManager()
	
	err := am.UnregisterAgent("nonexistent-agent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAgentManager_ListAgents(t *testing.T) {
	am := NewAgentManager()
	
	// Initially empty
	agents := am.ListAgents()
	assert.Equal(t, 0, len(agents))
	
	// Register multiple agents
	mockAgent1 := &MockAgent{}
	mockAgent2 := &MockAgent{}
	
	config1 := agent.AgentConfig{Name: "agent1"}
	config2 := agent.AgentConfig{Name: "agent2"}
	
	mockAgent1.On("GetConfig").Return(config1)
	mockAgent2.On("GetConfig").Return(config2)
	
	err := am.RegisterAgent("agent1", mockAgent1)
	require.NoError(t, err)
	
	err = am.RegisterAgent("agent2", mockAgent2)
	require.NoError(t, err)
	
	// List agents
	agents = am.ListAgents()
	assert.Equal(t, 2, len(agents))
	assert.Contains(t, agents, "agent1")
	assert.Contains(t, agents, "agent2")
}

func TestAgentManager_HealthCheck_Success(t *testing.T) {
	am := NewAgentManager()
	mockAgent := &MockAgent{}
	ctx := context.Background()
	
	config := agent.AgentConfig{Name: "test-agent"}
	mockAgent.On("GetConfig").Return(config)
	mockAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	
	// Register agent
	err := am.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)
	
	// Perform health check
	err = am.HealthCheck(ctx, "test-agent")
	require.NoError(t, err)
	
	// Verify health status is updated
	health, err := am.GetAgentHealth("test-agent")
	require.NoError(t, err)
	assert.Equal(t, AgentStatusHealthy, health.Status)
	assert.Equal(t, int64(1), health.CheckCount)
	assert.Equal(t, int64(0), health.FailureCount)
	assert.Empty(t, health.LastError)
	
	mockAgent.AssertExpectations(t)
}

func TestAgentManager_HealthCheck_Failure(t *testing.T) {
	am := NewAgentManager()
	mockAgent := &MockAgent{}
	ctx := context.Background()
	
	config := agent.AgentConfig{Name: "test-agent"}
	mockAgent.On("GetConfig").Return(config)
	mockAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(assert.AnError)
	
	// Register agent
	err := am.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)
	
	// Perform health check
	err = am.HealthCheck(ctx, "test-agent")
	assert.Error(t, err)
	
	// Verify health status is updated
	health, err := am.GetAgentHealth("test-agent")
	require.NoError(t, err)
	assert.Equal(t, AgentStatusUnhealthy, health.Status)
	assert.Equal(t, int64(1), health.CheckCount)
	assert.Equal(t, int64(1), health.FailureCount)
	assert.NotEmpty(t, health.LastError)
	
	mockAgent.AssertExpectations(t)
}

func TestAgentManager_ExecuteScan_Success(t *testing.T) {
	am := NewAgentManager()
	mockAgent := &MockAgent{}
	ctx := context.Background()
	
	config := agent.AgentConfig{Name: "test-agent"}
	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}
	
	expectedResult := &agent.ScanResult{
		AgentID: "test-agent",
		Status:  agent.ScanStatusCompleted,
		Findings: []agent.Finding{
			{
				ID:       "finding-1",
				Tool:     "test-agent",
				Severity: agent.SeverityHigh,
				Title:    "Test finding",
			},
		},
	}
	
	mockAgent.On("GetConfig").Return(config)
	mockAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	mockAgent.On("Scan", ctx, scanConfig).Return(expectedResult, nil)
	
	// Register agent
	err := am.RegisterAgent("test-agent", mockAgent)
	require.NoError(t, err)
	
	// Execute scan
	result, err := am.ExecuteScan(ctx, "test-agent", scanConfig)
	
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	
	mockAgent.AssertExpectations(t)
}

func TestAgentManager_ExecuteScan_AgentNotFound(t *testing.T) {
	am := NewAgentManager()
	ctx := context.Background()
	
	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}
	
	result, err := am.ExecuteScan(ctx, "nonexistent-agent", scanConfig)
	
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestAgentManager_ExecuteParallelScans_Success(t *testing.T) {
	am := NewAgentManager()
	ctx := context.Background()
	
	// Setup multiple mock agents
	mockAgent1 := &MockAgent{}
	mockAgent2 := &MockAgent{}
	
	config1 := agent.AgentConfig{Name: "agent1"}
	config2 := agent.AgentConfig{Name: "agent2"}
	
	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}
	
	result1 := &agent.ScanResult{
		AgentID: "agent1",
		Status:  agent.ScanStatusCompleted,
		Findings: []agent.Finding{
			{ID: "finding-1", Tool: "agent1"},
		},
	}
	
	result2 := &agent.ScanResult{
		AgentID: "agent2",
		Status:  agent.ScanStatusCompleted,
		Findings: []agent.Finding{
			{ID: "finding-2", Tool: "agent2"},
		},
	}
	
	// Setup mocks
	mockAgent1.On("GetConfig").Return(config1)
	mockAgent1.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	mockAgent1.On("Scan", ctx, scanConfig).Return(result1, nil)
	
	mockAgent2.On("GetConfig").Return(config2)
	mockAgent2.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	mockAgent2.On("Scan", ctx, scanConfig).Return(result2, nil)
	
	// Register agents
	err := am.RegisterAgent("agent1", mockAgent1)
	require.NoError(t, err)
	
	err = am.RegisterAgent("agent2", mockAgent2)
	require.NoError(t, err)
	
	// Execute parallel scans
	results, err := am.ExecuteParallelScans(ctx, []string{"agent1", "agent2"}, scanConfig)
	
	require.NoError(t, err)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, result1, results["agent1"])
	assert.Equal(t, result2, results["agent2"])
	
	mockAgent1.AssertExpectations(t)
	mockAgent2.AssertExpectations(t)
}

func TestAgentManager_GetAgentsForLanguages(t *testing.T) {
	am := NewAgentManager()
	
	// Setup agents with different language support
	mockAgent1 := &MockAgent{}
	mockAgent2 := &MockAgent{}
	mockAgent3 := &MockAgent{}
	
	config1 := agent.AgentConfig{
		Name:           "js-agent",
		SupportedLangs: []string{"javascript", "typescript"},
	}
	config2 := agent.AgentConfig{
		Name:           "py-agent",
		SupportedLangs: []string{"python"},
	}
	config3 := agent.AgentConfig{
		Name:           "multi-agent",
		SupportedLangs: []string{"javascript", "python", "go"},
	}
	
	mockAgent1.On("GetConfig").Return(config1)
	mockAgent2.On("GetConfig").Return(config2)
	mockAgent3.On("GetConfig").Return(config3)
	
	// Register agents
	err := am.RegisterAgent("js-agent", mockAgent1)
	require.NoError(t, err)
	
	err = am.RegisterAgent("py-agent", mockAgent2)
	require.NoError(t, err)
	
	err = am.RegisterAgent("multi-agent", mockAgent3)
	require.NoError(t, err)
	
	// Test language filtering
	tests := []struct {
		name      string
		languages []string
		expected  []string
	}{
		{
			name:      "javascript only",
			languages: []string{"javascript"},
			expected:  []string{"js-agent", "multi-agent"},
		},
		{
			name:      "python only",
			languages: []string{"python"},
			expected:  []string{"py-agent", "multi-agent"},
		},
		{
			name:      "multiple languages",
			languages: []string{"javascript", "python"},
			expected:  []string{"js-agent", "py-agent", "multi-agent"},
		},
		{
			name:      "unsupported language",
			languages: []string{"rust"},
			expected:  []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := am.GetAgentsForLanguages(tt.languages)
			
			assert.Equal(t, len(tt.expected), len(agents))
			for _, expectedAgent := range tt.expected {
				assert.Contains(t, agents, expectedAgent)
			}
		})
	}
}

func TestAgentManager_GetAgentsForCategories(t *testing.T) {
	am := NewAgentManager()
	
	// Setup agents with different category support
	mockAgent1 := &MockAgent{}
	mockAgent2 := &MockAgent{}
	
	config1 := agent.AgentConfig{
		Name:       "xss-agent",
		Categories: []agent.VulnCategory{agent.CategoryXSS, agent.CategoryCSRF},
	}
	config2 := agent.AgentConfig{
		Name:       "sql-agent",
		Categories: []agent.VulnCategory{agent.CategorySQLInjection, agent.CategoryXSS},
	}
	
	mockAgent1.On("GetConfig").Return(config1)
	mockAgent2.On("GetConfig").Return(config2)
	
	// Register agents
	err := am.RegisterAgent("xss-agent", mockAgent1)
	require.NoError(t, err)
	
	err = am.RegisterAgent("sql-agent", mockAgent2)
	require.NoError(t, err)
	
	// Test category filtering
	tests := []struct {
		name       string
		categories []agent.VulnCategory
		expected   []string
	}{
		{
			name:       "XSS only",
			categories: []agent.VulnCategory{agent.CategoryXSS},
			expected:   []string{"xss-agent", "sql-agent"},
		},
		{
			name:       "SQL injection only",
			categories: []agent.VulnCategory{agent.CategorySQLInjection},
			expected:   []string{"sql-agent"},
		},
		{
			name:       "CSRF only",
			categories: []agent.VulnCategory{agent.CategoryCSRF},
			expected:   []string{"xss-agent"},
		},
		{
			name:       "unsupported category",
			categories: []agent.VulnCategory{agent.CategoryCommandInjection},
			expected:   []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := am.GetAgentsForCategories(tt.categories)
			
			assert.Equal(t, len(tt.expected), len(agents))
			for _, expectedAgent := range tt.expected {
				assert.Contains(t, agents, expectedAgent)
			}
		})
	}
}

func TestAgentManager_GetStats(t *testing.T) {
	am := NewAgentManager()
	
	// Register agents with different health statuses
	mockAgent1 := &MockAgent{}
	mockAgent2 := &MockAgent{}
	mockAgent3 := &MockAgent{}
	
	config1 := agent.AgentConfig{
		Name:           "healthy-agent",
		SupportedLangs: []string{"javascript"},
		Categories:     []agent.VulnCategory{agent.CategoryXSS},
		RequiresDocker: true,
	}
	config2 := agent.AgentConfig{
		Name:           "unhealthy-agent",
		SupportedLangs: []string{"python"},
		Categories:     []agent.VulnCategory{agent.CategorySQLInjection},
		RequiresDocker: false,
	}
	config3 := agent.AgentConfig{
		Name:           "unknown-agent",
		SupportedLangs: []string{"go"},
		Categories:     []agent.VulnCategory{agent.CategoryCommandInjection},
		RequiresDocker: true,
	}
	
	mockAgent1.On("GetConfig").Return(config1)
	mockAgent2.On("GetConfig").Return(config2)
	mockAgent3.On("GetConfig").Return(config3)
	
	// Register agents
	err := am.RegisterAgent("healthy-agent", mockAgent1)
	require.NoError(t, err)
	
	err = am.RegisterAgent("unhealthy-agent", mockAgent2)
	require.NoError(t, err)
	
	err = am.RegisterAgent("unknown-agent", mockAgent3)
	require.NoError(t, err)
	
	// Set different health statuses
	am.health["healthy-agent"] = AgentHealth{
		Status:       AgentStatusHealthy,
		LastCheck:    time.Now(),
		CheckCount:   5,
		FailureCount: 0,
	}
	
	am.health["unhealthy-agent"] = AgentHealth{
		Status:       AgentStatusUnhealthy,
		LastCheck:    time.Now(),
		CheckCount:   3,
		FailureCount: 2,
		LastError:    "connection failed",
	}
	
	// Get stats
	stats := am.GetStats()
	
	// Verify overall stats
	assert.Equal(t, 3, stats.TotalAgents)
	assert.Equal(t, 1, stats.HealthyAgents)
	assert.Equal(t, 1, stats.UnhealthyAgents)
	assert.Equal(t, 1, stats.UnknownAgents)
	assert.Greater(t, stats.Uptime, time.Duration(0))
	
	// Verify individual agent stats
	assert.Equal(t, 3, len(stats.Agents))
	
	healthyStats := stats.Agents["healthy-agent"]
	assert.Equal(t, "healthy-agent", healthyStats.Name)
	assert.Equal(t, AgentStatusHealthy, healthyStats.Status)
	assert.Equal(t, int64(5), healthyStats.CheckCount)
	assert.Equal(t, int64(0), healthyStats.FailureCount)
	assert.Empty(t, healthyStats.LastError)
	assert.Equal(t, []string{"javascript"}, healthyStats.SupportedLangs)
	assert.Equal(t, []agent.VulnCategory{agent.CategoryXSS}, healthyStats.Categories)
	assert.True(t, healthyStats.RequiresDocker)
	
	unhealthyStats := stats.Agents["unhealthy-agent"]
	assert.Equal(t, "unhealthy-agent", unhealthyStats.Name)
	assert.Equal(t, AgentStatusUnhealthy, unhealthyStats.Status)
	assert.Equal(t, int64(3), unhealthyStats.CheckCount)
	assert.Equal(t, int64(2), unhealthyStats.FailureCount)
	assert.Equal(t, "connection failed", unhealthyStats.LastError)
	assert.Equal(t, []string{"python"}, unhealthyStats.SupportedLangs)
	assert.Equal(t, []agent.VulnCategory{agent.CategorySQLInjection}, unhealthyStats.Categories)
	assert.False(t, unhealthyStats.RequiresDocker)
}

// Benchmark tests
func BenchmarkAgentManager_RegisterAgent(b *testing.B) {
	am := NewAgentManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockAgent := &MockAgent{}
		config := agent.AgentConfig{Name: "test-agent"}
		mockAgent.On("GetConfig").Return(config)
		
		agentName := fmt.Sprintf("agent-%d", i)
		err := am.RegisterAgent(agentName, mockAgent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAgentManager_HealthCheck(b *testing.B) {
	am := NewAgentManager()
	mockAgent := &MockAgent{}
	ctx := context.Background()
	
	config := agent.AgentConfig{Name: "test-agent"}
	mockAgent.On("GetConfig").Return(config)
	mockAgent.On("HealthCheck", mock.AnythingOfType("*context.timerCtx")).Return(nil)
	
	err := am.RegisterAgent("test-agent", mockAgent)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := am.HealthCheck(ctx, "test-agent")
		if err != nil {
			b.Fatal(err)
		}
	}
}