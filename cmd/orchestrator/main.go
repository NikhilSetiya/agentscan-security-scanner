package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/agents/dast/zap"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/sast/bandit"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/sast/eslint"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/sast/semgrep"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/sca/govulncheck"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/sca/npm"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/sca/pip"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/secrets/gitsecrets"
	"github.com/NikhilSetiya/agentscan-security-scanner/agents/secrets/trufflehog"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
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

	// Initialize database connection
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	repos := database.NewRepositories(db)
	
	// Create repository adapter
	dbAdapter := database.NewRepositoryAdapter(db, repos)

	log.Printf("Database connection established successfully")

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

	// Initialize orchestration service
	orchestratorConfig := orchestrator.DefaultConfig()
	orchestratorConfig.MaxConcurrentScans = cfg.Agents.MaxConcurrent
	orchestratorConfig.DefaultTimeout = cfg.Agents.DefaultTimeout
	orchestratorConfig.WorkerCount = 5

	// Initialize orchestration service with real database
	service := orchestrator.NewService(dbAdapter, jobQueue, agentManager, orchestratorConfig)

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
	// Register SAST agents
	semgrepAgent := semgrep.NewAgent()
	if err := agentManager.RegisterAgent("semgrep", semgrepAgent); err != nil {
		return err
	}
	log.Println("Registered Semgrep SAST agent")

	banditAgent := bandit.NewAgent()
	if err := agentManager.RegisterAgent("bandit", banditAgent); err != nil {
		return err
	}
	log.Println("Registered Bandit Python agent")

	eslintAgent := eslint.NewAgent()
	if err := agentManager.RegisterAgent("eslint-security", eslintAgent); err != nil {
		return err
	}
	log.Println("Registered ESLint Security agent")

	// Register SCA (dependency scanning) agents
	npmAgent := npm.NewAgent()
	if err := agentManager.RegisterAgent("npm-audit", npmAgent); err != nil {
		return err
	}
	log.Println("Registered npm audit SCA agent")

	pipAgent := pip.NewAgent()
	if err := agentManager.RegisterAgent("pip-audit", pipAgent); err != nil {
		return err
	}
	log.Println("Registered pip-audit SCA agent")

	govulncheckAgent := govulncheck.NewAgent()
	if err := agentManager.RegisterAgent("govulncheck", govulncheckAgent); err != nil {
		return err
	}
	log.Println("Registered govulncheck SCA agent")

	// Register secret scanning agents
	trufflehogAgent := trufflehog.NewAgent()
	if err := agentManager.RegisterAgent("trufflehog", trufflehogAgent); err != nil {
		return err
	}
	log.Println("Registered TruffleHog secret scanning agent")

	gitSecretsAgent := gitsecrets.NewAgent()
	if err := agentManager.RegisterAgent("git-secrets", gitSecretsAgent); err != nil {
		return err
	}
	log.Println("Registered git-secrets secret scanning agent")

	// Register DAST agents
	zapAgent := zap.NewAgent()
	if err := agentManager.RegisterAgent("zap", zapAgent); err != nil {
		return err
	}
	log.Println("Registered OWASP ZAP DAST agent")

	return nil
}