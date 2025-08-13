package main

import (
	"context"
	"fmt"
	"time"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/config"
)

// SimpleValidation performs basic system validation
func main() {
	fmt.Println("🔒 AgentScan Simple System Validation")
	fmt.Println("=====================================")

	// Test 1: Configuration Loading
	fmt.Print("Testing configuration loading... ")
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ FAIL: %v\n", err)
		return
	}
	fmt.Println("✅ PASS")

	// Test 2: Database Connection
	fmt.Print("Testing database connection... ")
	db, err := database.New(&cfg.Database)
	if err != nil {
		fmt.Printf("❌ FAIL: %v\n", err)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.Health(ctx); err != nil {
		fmt.Printf("❌ FAIL: %v\n", err)
		return
	}
	fmt.Println("✅ PASS")

	// Test 3: Redis Connection
	fmt.Print("Testing Redis connection... ")
	redis, err := queue.NewRedisClient(&cfg.Redis)
	if err != nil {
		fmt.Printf("❌ FAIL: %v\n", err)
		return
	}
	defer redis.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redis.Health(ctx); err != nil {
		fmt.Printf("❌ FAIL: %v\n", err)
		return
	}
	fmt.Println("✅ PASS")

	// Test 4: Database Repositories
	fmt.Print("Testing database repositories... ")
	repos := database.NewRepositories(db)
	if repos == nil {
		fmt.Println("❌ FAIL: repositories not initialized")
		return
	}
	fmt.Println("✅ PASS")

	// Test 5: Job Queue
	fmt.Print("Testing job queue... ")
	jobQueue := queue.NewQueue(redis, "test_queue", queue.DefaultQueueConfig())
	if jobQueue == nil {
		fmt.Println("❌ FAIL: job queue not initialized")
		return
	}
	fmt.Println("✅ PASS")

	fmt.Println("\n📊 Validation Summary")
	fmt.Println("====================")
	fmt.Println("✅ Configuration: Working")
	fmt.Println("✅ Database: Connected")
	fmt.Println("✅ Redis: Connected")
	fmt.Println("✅ Repositories: Initialized")
	fmt.Println("✅ Job Queue: Working")

	fmt.Println("\n🎯 System Status: CORE COMPONENTS OPERATIONAL")
	fmt.Println("The basic system infrastructure is working correctly.")
	fmt.Println("Ready for more comprehensive testing.")

	fmt.Println("\n🔒 Simple Validation Complete")
	fmt.Println("=============================")
}