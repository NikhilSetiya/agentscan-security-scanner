package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/infrastructure/monitoring"
	"github.com/your-org/agentscan/internal/shared/logging"
	"github.com/your-org/agentscan/internal/testing"
)

func main() {
	var (
		configFile = flag.String("config", "", "Configuration file path")
		envFile    = flag.String("env", "", "Environment file path")
		output     = flag.String("output", "", "Output file for test results (JSON)")
		verbose    = flag.Bool("verbose", false, "Verbose output")
		suite      = flag.String("suite", "", "Run specific test suite (database, redis, api, performance, security, monitoring)")
		timeout    = flag.Duration("timeout", 10*time.Minute, "Test timeout")
	)
	flag.Parse()

	// Initialize logging
	logger := logging.GetLogger()
	logger.Info("Starting AgentScan Integration Tests")

	// Load environment configuration
	envManager := config.NewEnvManager()
	if *envFile != "" {
		if err := envManager.LoadEnvFiles(*envFile); err != nil {
			logger.Error("Failed to load environment file", "error", err)
			os.Exit(1)
		}
	} else {
		if err := envManager.LoadEnvFiles(); err != nil {
			logger.Error("Failed to load environment files", "error", err)
			os.Exit(1)
		}
	}

	// Load production configuration
	prodConfig, err := config.LoadProductionConfig()
	if err != nil {
		logger.Error("Failed to load production config", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := prodConfig.Validate(); err != nil {
		logger.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully",
		"environment", prodConfig.App.Environment,
		"database_max_conns", prodConfig.Database.MaxOpenConns,
		"redis_pool_size", prodConfig.Redis.PoolSize,
	)

	// Initialize database connection
	db, err := initializeDatabase(prodConfig)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Redis connection
	redisClient, err := initializeRedis(prodConfig)
	if err != nil {
		logger.Error("Failed to initialize Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize monitoring
	metrics := monitoring.InitMetrics()
	healthService := monitoring.NewHealthService(prodConfig.App.Version)

	// Register health checkers
	healthService.RegisterChecker(monitoring.NewDatabaseHealthChecker(db, "primary_database"))
	healthService.RegisterChecker(monitoring.NewRedisHealthChecker(redisClient, "primary_redis"))

	// Create test runner
	testRunner := testing.NewIntegrationTestRunner(
		prodConfig,
		db,
		redisClient,
		metrics,
		healthService,
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Run tests
	logger.Info("Starting integration test execution",
		"timeout", *timeout,
		"suite_filter", *suite,
	)

	results, err := testRunner.RunAllTests(ctx)
	if err != nil {
		logger.Error("Integration tests failed", "error", err)
		os.Exit(1)
	}

	// Log results to Observe (if available)
	if err := logResultsToObserve(results); err != nil {
		logger.Warn("Failed to log results to Observe", "error", err)
	}

	// Output results
	if err := outputResults(results, *output, *verbose); err != nil {
		logger.Error("Failed to output results", "error", err)
		os.Exit(1)
	}

	// Determine exit code
	exitCode := 0
	if results.FailedTests > 0 {
		exitCode = 1
		logger.Error("Integration tests completed with failures",
			"total", results.TotalTests,
			"passed", results.PassedTests,
			"failed", results.FailedTests,
			"duration", results.Duration,
		)
	} else {
		logger.Info("Integration tests completed successfully",
			"total", results.TotalTests,
			"passed", results.PassedTests,
			"duration", results.Duration,
		)
	}

	os.Exit(exitCode)
}

func initializeDatabase(config *config.ProductionConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", config.GetDatabaseConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.Database.MaxOpenConns)
	db.SetMaxIdleConns(config.Database.MaxIdleConns)
	db.SetConnMaxLifetime(config.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.Database.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func initializeRedis(config *config.ProductionConfig) (*redis.Client, error) {
	opt, err := redis.ParseURL(config.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure Redis client
	opt.Password = config.Redis.Password
	opt.MaxRetries = config.Redis.MaxRetries
	opt.PoolSize = config.Redis.PoolSize
	opt.MinIdleConns = config.Redis.MinIdleConns
	opt.PoolTimeout = config.Redis.PoolTimeout
	opt.IdleTimeout = config.Redis.IdleTimeout
	opt.ReadTimeout = config.Redis.ReadTimeout
	opt.WriteTimeout = config.Redis.WriteTimeout

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}

func logResultsToObserve(results *testing.TestResults) error {
	// Create structured log entry for Observe
	logData := map[string]interface{}{
		"event_type":    "integration_test_results",
		"timestamp":     results.StartTime,
		"duration_ms":   results.Duration.Milliseconds(),
		"total_tests":   results.TotalTests,
		"passed_tests":  results.PassedTests,
		"failed_tests":  results.FailedTests,
		"skipped_tests": results.SkippedTests,
		"success_rate":  float64(results.PassedTests) / float64(results.TotalTests) * 100,
		"test_suites":   make(map[string]interface{}),
	}

	// Add suite-level metrics
	for suiteName, suite := range results.TestSuites {
		logData["test_suites"].(map[string]interface{})[suiteName] = map[string]interface{}{
			"duration_ms":   suite.Duration.Milliseconds(),
			"total_tests":   suite.TotalTests,
			"passed_tests":  suite.PassedTests,
			"failed_tests":  suite.FailedTests,
			"success_rate":  float64(suite.PassedTests) / float64(suite.TotalTests) * 100,
		}
	}

	// Log to standard logger (which should be configured to send to Observe)
	logger := logging.GetLogger()
	logger.Info("Integration test results", "observe_data", logData)

	// Also log individual test failures for detailed analysis
	for suiteName, suite := range results.TestSuites {
		for testName, test := range suite.Tests {
			if test.Status == testing.TestStatusFailed {
				logger.Error("Integration test failure",
					"suite", suiteName,
					"test", testName,
					"duration_ms", test.Duration.Milliseconds(),
					"error", test.Error,
					"observe_data", map[string]interface{}{
						"event_type": "integration_test_failure",
						"suite":      suiteName,
						"test":       testName,
						"error":      test.Error,
						"duration":   test.Duration.Milliseconds(),
					},
				)
			}
		}
	}

	return nil
}

func outputResults(results *testing.TestResults, outputFile string, verbose bool) error {
	// Create summary
	summary := map[string]interface{}{
		"summary": map[string]interface{}{
			"start_time":    results.StartTime.Format(time.RFC3339),
			"end_time":      results.EndTime.Format(time.RFC3339),
			"duration":      results.Duration.String(),
			"total_tests":   results.TotalTests,
			"passed_tests":  results.PassedTests,
			"failed_tests":  results.FailedTests,
			"skipped_tests": results.SkippedTests,
			"success_rate":  fmt.Sprintf("%.2f%%", float64(results.PassedTests)/float64(results.TotalTests)*100),
		},
		"test_suites": make(map[string]interface{}),
	}

	// Add suite summaries
	for suiteName, suite := range results.TestSuites {
		suiteData := map[string]interface{}{
			"duration":      suite.Duration.String(),
			"total_tests":   suite.TotalTests,
			"passed_tests":  suite.PassedTests,
			"failed_tests":  suite.FailedTests,
			"skipped_tests": suite.SkippedTests,
			"success_rate":  fmt.Sprintf("%.2f%%", float64(suite.PassedTests)/float64(suite.TotalTests)*100),
		}

		if verbose {
			tests := make(map[string]interface{})
			for testName, test := range suite.Tests {
				tests[testName] = map[string]interface{}{
					"status":   test.Status,
					"duration": test.Duration.String(),
					"error":    test.Error,
				}
			}
			suiteData["tests"] = tests
		}

		summary["test_suites"].(map[string]interface{})[suiteName] = suiteData
	}

	// Add errors if any
	if len(results.Errors) > 0 {
		summary["errors"] = results.Errors
	}

	// Output to console
	fmt.Println("\n" + "="*80)
	fmt.Println("INTEGRATION TEST RESULTS")
	fmt.Println("="*80)
	fmt.Printf("Duration: %s\n", results.Duration)
	fmt.Printf("Total Tests: %d\n", results.TotalTests)
	fmt.Printf("Passed: %d\n", results.PassedTests)
	fmt.Printf("Failed: %d\n", results.FailedTests)
	fmt.Printf("Skipped: %d\n", results.SkippedTests)
	fmt.Printf("Success Rate: %.2f%%\n", float64(results.PassedTests)/float64(results.TotalTests)*100)

	fmt.Println("\nTest Suite Results:")
	for suiteName, suite := range results.TestSuites {
		status := "✅ PASS"
		if suite.FailedTests > 0 {
			status = "❌ FAIL"
		}
		fmt.Printf("  %s %s: %d/%d passed (%.2f%%) in %s\n",
			status,
			suiteName,
			suite.PassedTests,
			suite.TotalTests,
			float64(suite.PassedTests)/float64(suite.TotalTests)*100,
			suite.Duration,
		)

		if verbose && suite.FailedTests > 0 {
			for testName, test := range suite.Tests {
				if test.Status == testing.TestStatusFailed {
					fmt.Printf("    ❌ %s: %s\n", testName, test.Error)
				}
			}
		}
	}

	if len(results.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range results.Errors {
			fmt.Printf("  ❌ %s\n", err)
		}
	}

	// Output to file if specified
	if outputFile != "" {
		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results: %w", err)
		}

		if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write results file: %w", err)
		}

		fmt.Printf("\nDetailed results written to: %s\n", outputFile)
	}

	return nil
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "AgentScan Integration Test Runner\n\n")
		fmt.Fprintf(os.Stderr, "This tool runs comprehensive integration tests for the AgentScan application,\n")
		fmt.Fprintf(os.Stderr, "including database, Redis, API, performance, security, and monitoring tests.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                                    # Run all tests\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -suite database                    # Run only database tests\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -verbose -output results.json      # Verbose output to file\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -env .env.production -timeout 5m   # Use production env with 5min timeout\n", os.Args[0])
	}
}