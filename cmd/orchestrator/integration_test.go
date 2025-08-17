package main

import (
	"context"
	"testing"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestratorDatabaseIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Load test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Name:            "agentscan_test",
			User:            "agentscan",
			Password:        "password",
			SSLMode:         "disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       1, // Use test database
			PoolSize: 5,
		},
		Agents: config.AgentsConfig{
			MaxConcurrent:  5,
			DefaultTimeout: 5 * time.Minute,
		},
	}

	ctx := context.Background()

	// Initialize database connection
	db, err := database.New(&cfg.Database)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Test database health
	err = db.Health(ctx)
	require.NoError(t, err, "Database health check should pass")

	// Initialize repositories
	repos := database.NewRepositories(db)
	dbAdapter := database.NewRepositoryAdapter(db, repos)

	// Test repository adapter health
	err = dbAdapter.Health(ctx)
	require.NoError(t, err, "Repository adapter health check should pass")

	// Initialize Redis connection (skip if not available)
	redisClient, err := queue.NewRedisClient(&cfg.Redis)
	if err != nil {
		t.Skipf("Failed to connect to test Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize job queue
	queueConfig := queue.DefaultQueueConfig()
	queueConfig.MaxConcurrency = cfg.Agents.MaxConcurrent
	queueConfig.DefaultTimeout = cfg.Agents.DefaultTimeout
	
	jobQueue := queue.NewQueue(redisClient, "test_scans", queueConfig)

	// Initialize agent manager
	agentManager := orchestrator.NewAgentManager()

	// Register a test agent (just semgrep for simplicity)
	// Note: This would require the actual agent implementations to be available
	// For now, we'll just test the orchestrator service creation

	// Initialize orchestration service
	orchestratorConfig := orchestrator.DefaultConfig()
	orchestratorConfig.MaxConcurrentScans = cfg.Agents.MaxConcurrent
	orchestratorConfig.DefaultTimeout = cfg.Agents.DefaultTimeout
	orchestratorConfig.WorkerCount = 2

	// Create orchestration service with real database
	service := orchestrator.NewService(dbAdapter, jobQueue, agentManager, orchestratorConfig)

	// Test service health
	err = service.Health(ctx)
	assert.NoError(t, err, "Orchestration service health check should pass")

	// Test service start and stop
	err = service.Start(ctx)
	require.NoError(t, err, "Service should start successfully")

	// Give it a moment to initialize
	time.Sleep(100 * time.Millisecond)

	// Test health again after starting
	err = service.Health(ctx)
	assert.NoError(t, err, "Service health should be good after starting")

	// Stop the service
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	err = service.Stop(stopCtx)
	assert.NoError(t, err, "Service should stop gracefully")
}

func TestDatabaseConnectionOnly(t *testing.T) {
	// This test only requires database connection, not Redis
	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		Name:            "agentscan_test",
		User:            "agentscan",
		Password:        "password",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	ctx := context.Background()

	// Initialize database connection
	db, err := database.New(cfg)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Test database health
	err = db.Health(ctx)
	require.NoError(t, err, "Database health check should pass")

	// Initialize repositories
	repos := database.NewRepositories(db)
	assert.NotNil(t, repos, "Repositories should be initialized")
	assert.NotNil(t, repos.Users, "User repository should be initialized")
	assert.NotNil(t, repos.ScanJobs, "ScanJob repository should be initialized")
	assert.NotNil(t, repos.Findings, "Finding repository should be initialized")

	// Create repository adapter
	dbAdapter := database.NewRepositoryAdapter(db, repos)
	assert.NotNil(t, dbAdapter, "Repository adapter should be initialized")

	// Test repository adapter health
	err = dbAdapter.Health(ctx)
	require.NoError(t, err, "Repository adapter health check should pass")
}