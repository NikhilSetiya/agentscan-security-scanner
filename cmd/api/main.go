package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentscan/agentscan/internal/api"
	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.Health(ctx); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}
	cancel()

	log.Println("Database connection established")

	// Initialize Redis connection
	redis, err := queue.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Test Redis connection
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	if err := redis.Health(ctx); err != nil {
		log.Fatalf("Redis health check failed: %v", err)
	}
	cancel()

	log.Println("Redis connection established")

	// Initialize repositories
	repos := database.NewRepositories(db)
	_ = repos // TODO: Use repositories in handlers

	// Initialize job queue
	jobQueue := queue.NewQueue(redis, "agentscan", queue.DefaultQueueConfig())
	_ = jobQueue // TODO: Use job queue in handlers

	// TODO: Initialize authentication middleware
	// TODO: Initialize logging middleware

	// Create HTTP router
	mux := http.NewServeMux()

	// Health check endpoint
	healthHandler := api.NewHealthHandler(db, redis)
	mux.Handle("/health", healthHandler)

	// TODO: Add API routes
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "AgentScan API v1.0", "status": "ok"}`))
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting API server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}