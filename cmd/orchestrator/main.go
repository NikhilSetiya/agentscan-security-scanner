package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentscan/agentscan/agents/sast/bandit"
	"github.com/agentscan/agentscan/agents/sast/semgrep"
	"github.com/agentscan/agentscan/internal/orchestrator"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting orchestrator service")
	log.Printf("Max concurrent agents: %d", cfg.Agents.MaxConcurrent)
	log.Printf("Default timeout: %v", cfg.Agents.DefaultTimeout)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis connection
	redisClient, err := queue.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize job queue
	queueConfig := queue.DefaultQueueConfig()
	queueConfig.MaxConcurrency = cfg.Agents.MaxConcurrent
	queueConfig.DefaultTimeout = cfg.Agents.DefaultTimeout
	
	jobQueue := queue.NewQueue(redisClient, "scans", queueConfig)

	// Initialize agent manager
	agentManager := orchestrator.NewAgentManager()

	// Register available agents
	if err := registerAgents(agentManager); err != nil {
		log.Fatalf("Failed to register agents: %v", err)
	}

	// TODO: Initialize database connection
	// For now, we'll use a mock database in production you'd initialize a real DB
	log.Println("Warning: Using mock database - implement real database connection")

	// Initialize orchestration service
	orchestratorConfig := orchestrator.DefaultConfig()
	orchestratorConfig.MaxConcurrentScans = cfg.Agents.MaxConcurrent
	orchestratorConfig.DefaultTimeout = cfg.Agents.DefaultTimeout
	orchestratorConfig.WorkerCount = 5

	// TODO: Pass real database instance
	service := orchestrator.NewService(nil, jobQueue, agentManager, orchestratorConfig)

	// Start the orchestration service
	if err := service.Start(ctx); err != nil {
		log.Fatalf("Failed to start orchestration service: %v", err)
	}

	log.Println("Orchestrator service started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down orchestrator...")

	// Cancel context to stop all goroutines
	cancel()

	// Stop the orchestration service with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := service.Stop(shutdownCtx); err != nil {
		log.Printf("Error stopping orchestration service: %v", err)
	}

	log.Println("Orchestrator exited")
}

// registerAgents registers all available security agents
func registerAgents(agentManager *orchestrator.AgentManager) error {
	// Register Semgrep SAST agent
	semgrepAgent := semgrep.NewAgent()
	if err := agentManager.RegisterAgent("semgrep", semgrepAgent); err != nil {
		return err
	}
	log.Println("Registered Semgrep SAST agent")

	// Register Bandit Python agent
	banditAgent := bandit.NewAgent()
	if err := agentManager.RegisterAgent("bandit", banditAgent); err != nil {
		return err
	}
	log.Println("Registered Bandit Python agent")

	// TODO: Register additional agents as they are implemented
	// - ESLint Security agent
	// - Dependency scanning agents
	// - Secret scanning agents

	return nil
}