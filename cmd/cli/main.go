package main

import (
	"fmt"
	"os"

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
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "scan":
		handleScanCommand(cfg, os.Args[2:])
	case "status":
		handleStatusCommand(cfg, os.Args[2:])
	case "results":
		handleResultsCommand(cfg, os.Args[2:])
	case "version":
		handleVersionCommand()
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("AgentScan CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agentscan <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scan     Start a security scan")
	fmt.Println("  status   Check scan status")
	fmt.Println("  results  Get scan results")
	fmt.Println("  version  Show version information")
	fmt.Println("  help     Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  agentscan scan --repo https://github.com/user/repo")
	fmt.Println("  agentscan status --job-id abc123")
	fmt.Println("  agentscan results --job-id abc123 --format json")
}

func handleScanCommand(cfg *config.Config, args []string) {
	// TODO: Implement scan command
	fmt.Println("Scan command not yet implemented")
}

func handleStatusCommand(cfg *config.Config, args []string) {
	// TODO: Implement status command
	fmt.Println("Status command not yet implemented")
}

func handleResultsCommand(cfg *config.Config, args []string) {
	// TODO: Implement results command
	fmt.Println("Results command not yet implemented")
}

func handleVersionCommand() {
	fmt.Println("AgentScan CLI v0.1.0")
	fmt.Println("Build: development")
}