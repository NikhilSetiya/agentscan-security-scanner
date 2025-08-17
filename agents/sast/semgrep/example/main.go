package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/agents/sast/semgrep"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

func main() {
	// Create a new Semgrep agent
	semgrepAgent := semgrep.NewAgent()

	// Check if the agent is healthy (Docker available, image accessible)
	ctx := context.Background()
	if err := semgrepAgent.HealthCheck(ctx); err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Println("✓ Semgrep agent is healthy")

	// Display agent configuration
	config := semgrepAgent.GetConfig()
	fmt.Printf("Agent: %s v%s\n", config.Name, config.Version)
	fmt.Printf("Supported languages: %v\n", config.SupportedLangs)
	fmt.Printf("Vulnerability categories: %v\n", config.Categories)

	// Configure a scan
	scanConfig := agent.ScanConfig{
		RepoURL:   "https://github.com/OWASP/NodeGoat.git", // Example vulnerable app
		Branch:    "master",
		Languages: []string{"javascript"}, // Focus on JavaScript
		Timeout:   5 * time.Minute,
	}

	fmt.Printf("\nStarting scan of %s...\n", scanConfig.RepoURL)
	startTime := time.Now()

	// Execute the scan
	result, err := semgrepAgent.Scan(ctx, scanConfig)
	if err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	// Display results
	fmt.Printf("\n✓ Scan completed in %v\n", time.Since(startTime))
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Tool version: %s\n", result.Metadata.ToolVersion)
	fmt.Printf("Files scanned: %d\n", result.Metadata.FilesScanned)
	fmt.Printf("Lines scanned: %d\n", result.Metadata.LinesScanned)

	// Display findings
	fmt.Printf("\nFound %d security issues:\n", len(result.Findings))
	
	severityCounts := make(map[agent.Severity]int)
	for _, finding := range result.Findings {
		severityCounts[finding.Severity]++
	}

	for severity, count := range severityCounts {
		fmt.Printf("  %s: %d\n", severity, count)
	}

	// Display top 5 findings
	fmt.Println("\nTop findings:")
	maxDisplay := 5
	if len(result.Findings) < maxDisplay {
		maxDisplay = len(result.Findings)
	}

	for i := 0; i < maxDisplay; i++ {
		finding := result.Findings[i]
		fmt.Printf("\n%d. %s (%s)\n", i+1, finding.Title, finding.Severity)
		fmt.Printf("   File: %s:%d\n", finding.File, finding.Line)
		fmt.Printf("   Rule: %s\n", finding.RuleID)
		fmt.Printf("   Confidence: %.1f\n", finding.Confidence)
		if finding.Code != "" {
			fmt.Printf("   Code: %s\n", finding.Code)
		}
	}

	if len(result.Findings) > maxDisplay {
		fmt.Printf("\n... and %d more findings\n", len(result.Findings)-maxDisplay)
	}

	fmt.Println("\nScan completed successfully!")
}