package main

import (
	"testing"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseIntegrationSetup tests that the database integration setup works correctly
func TestDatabaseIntegrationSetup(t *testing.T) {
	// Test configuration
	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		Name:            "test_db",
		User:            "test_user",
		Password:        "test_password",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// Test that database.New can be called with config
	// This will fail to connect but should not panic or have compilation errors
	_, err := database.New(cfg)
	assert.Error(t, err, "Expected connection error with test credentials")
	assert.Contains(t, err.Error(), "failed to connect to database", "Should get connection error")

	// Test that repositories can be created (even without DB connection)
	// This tests the code structure
	db := &database.DB{} // Mock DB for structure testing
	repos := database.NewRepositories(db)
	assert.NotNil(t, repos, "Repositories should be created")
	assert.NotNil(t, repos.Users, "User repository should exist")
	assert.NotNil(t, repos.ScanJobs, "ScanJob repository should exist")
	assert.NotNil(t, repos.Findings, "Finding repository should exist")

	// Test that repository adapter can be created
	dbAdapter := database.NewRepositoryAdapter(db, repos)
	assert.NotNil(t, dbAdapter, "Repository adapter should be created")
}

// TestOrchestratorServiceCreation tests that the orchestrator service can be created with database
func TestOrchestratorServiceCreation(t *testing.T) {
	// Create mock components
	db := &database.DB{} // Mock DB
	repos := database.NewRepositories(db)
	dbAdapter := database.NewRepositoryAdapter(db, repos)

	// Create mock queue (this will also fail to connect but tests structure)
	redisConfig := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 5,
	}

	// This will fail but tests that the interface is correct
	_, err := queue.NewRedisClient(redisConfig)
	assert.Error(t, err, "Expected Redis connection error")

	// Create agent manager
	agentManager := orchestrator.NewAgentManager()
	assert.NotNil(t, agentManager, "Agent manager should be created")

	// Test orchestrator config
	orchestratorConfig := orchestrator.DefaultConfig()
	assert.NotNil(t, orchestratorConfig, "Orchestrator config should be created")
	assert.Greater(t, orchestratorConfig.MaxConcurrentScans, 0, "Max concurrent scans should be positive")
	assert.Greater(t, orchestratorConfig.WorkerCount, 0, "Worker count should be positive")

	// Test that orchestrator service can be created with database adapter
	// Note: We can't test with a real queue here, but we can test the interface
	service := orchestrator.NewService(dbAdapter, nil, agentManager, orchestratorConfig)
	assert.NotNil(t, service, "Orchestrator service should be created with database adapter")
}

// TestRegisterAgentsFunction tests that the registerAgents function works
func TestRegisterAgentsFunction(t *testing.T) {
	agentManager := orchestrator.NewAgentManager()
	
	err := registerAgents(agentManager)
	require.NoError(t, err, "registerAgents should not return error")

	// Test that agents are registered
	agents := agentManager.ListAgents()
	assert.Greater(t, len(agents), 0, "Should have registered agents")

	// Check for specific agents
	expectedAgents := []string{
		"semgrep",
		"bandit", 
		"eslint-security",
		"npm-audit",
		"pip-audit",
		"govulncheck",
		"trufflehog",
		"git-secrets",
		"zap",
	}

	for _, expectedAgent := range expectedAgents {
		found := false
		for _, agent := range agents {
			if agent == expectedAgent {
				found = true
				break
			}
		}
		assert.True(t, found, "Agent %s should be registered", expectedAgent)
	}
}

// TestMainFunctionStructure tests that main function has correct structure
func TestMainFunctionStructure(t *testing.T) {
	// This test verifies that the main function structure is correct
	// by testing the individual components that main() uses

	// Test config loading structure
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Name:            "test",
			User:            "test",
			Password:        "test",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			PoolSize: 5,
		},
		Agents: config.AgentsConfig{
			MaxConcurrent:  10,
			DefaultTimeout: 10 * time.Minute,
		},
	}

	// Test that all required config fields are accessible
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "test", cfg.Database.Name)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, 10, cfg.Agents.MaxConcurrent)
	assert.Equal(t, 10*time.Minute, cfg.Agents.DefaultTimeout)

	// Test that the main function components can be created
	// (even if they fail to connect, the structure should be correct)

	// Database components
	_, err := database.New(&cfg.Database)
	assert.Error(t, err) // Expected to fail with test config

	// Redis components  
	_, err = queue.NewRedisClient(&cfg.Redis)
	assert.Error(t, err) // Expected to fail with test config

	// Agent manager
	agentManager := orchestrator.NewAgentManager()
	assert.NotNil(t, agentManager)

	// Orchestrator config
	orchestratorConfig := orchestrator.DefaultConfig()
	orchestratorConfig.MaxConcurrentScans = cfg.Agents.MaxConcurrent
	orchestratorConfig.DefaultTimeout = cfg.Agents.DefaultTimeout
	orchestratorConfig.WorkerCount = 5

	assert.Equal(t, cfg.Agents.MaxConcurrent, orchestratorConfig.MaxConcurrentScans)
	assert.Equal(t, cfg.Agents.DefaultTimeout, orchestratorConfig.DefaultTimeout)
	assert.Equal(t, 5, orchestratorConfig.WorkerCount)
}