package main

import (
	"fmt"
	"log"
	"os"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/config"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create migrator
	migrator, err := database.NewMigrator(&cfg.Database, "migrations")
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	switch command {
	case "up":
		handleUp(migrator)
	case "down":
		handleDown(migrator)
	case "steps":
		handleSteps(migrator, os.Args[2:])
	case "version":
		handleVersion(migrator)
	case "force":
		handleForce(migrator, os.Args[2:])
	case "drop":
		handleDrop(migrator)
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("AgentScan Database Migration Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  migrate <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up           Run all available migrations")
	fmt.Println("  down         Rollback all migrations")
	fmt.Println("  steps <n>    Run n migrations up (positive) or down (negative)")
	fmt.Println("  version      Show current migration version")
	fmt.Println("  force <v>    Force set migration version without running migrations")
	fmt.Println("  drop         Drop entire database schema (DANGEROUS)")
	fmt.Println("  help         Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  migrate up")
	fmt.Println("  migrate down")
	fmt.Println("  migrate steps 1")
	fmt.Println("  migrate steps -1")
	fmt.Println("  migrate version")
	fmt.Println("  migrate force 1")
}

func handleUp(migrator *database.Migrator) {
	fmt.Println("Running migrations...")
	if err := migrator.Up(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	fmt.Println("Migrations completed successfully")
}

func handleDown(migrator *database.Migrator) {
	fmt.Println("Rolling back migrations...")
	if err := migrator.Down(); err != nil {
		log.Fatalf("Failed to rollback migrations: %v", err)
	}
	fmt.Println("Rollback completed successfully")
}

func handleSteps(migrator *database.Migrator, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Steps command requires a number argument\n")
		os.Exit(1)
	}

	var steps int
	if _, err := fmt.Sscanf(args[0], "%d", &steps); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid steps argument: %s\n", args[0])
		os.Exit(1)
	}

	fmt.Printf("Running %d migration steps...\n", steps)
	if err := migrator.Steps(steps); err != nil {
		log.Fatalf("Failed to run migration steps: %v", err)
	}
	fmt.Println("Migration steps completed successfully")
}

func handleVersion(migrator *database.Migrator) {
	version, dirty, err := migrator.Version()
	if err != nil {
		log.Fatalf("Failed to get migration version: %v", err)
	}

	fmt.Printf("Current migration version: %d\n", version)
	if dirty {
		fmt.Println("WARNING: Database is in a dirty state")
	}
}

func handleForce(migrator *database.Migrator, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Force command requires a version argument\n")
		os.Exit(1)
	}

	var version int
	if _, err := fmt.Sscanf(args[0], "%d", &version); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid version argument: %s\n", args[0])
		os.Exit(1)
	}

	fmt.Printf("Forcing migration version to %d...\n", version)
	if err := migrator.Force(version); err != nil {
		log.Fatalf("Failed to force migration version: %v", err)
	}
	fmt.Println("Migration version forced successfully")
}

func handleDrop(migrator *database.Migrator) {
	fmt.Println("WARNING: This will drop the entire database schema!")
	fmt.Print("Are you sure? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("Operation cancelled")
		return
	}

	fmt.Println("Dropping database schema...")
	if err := migrator.Drop(); err != nil {
		log.Fatalf("Failed to drop database schema: %v", err)
	}
	fmt.Println("Database schema dropped successfully")
}