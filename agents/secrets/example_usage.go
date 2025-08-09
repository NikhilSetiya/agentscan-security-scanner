package secrets

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/agentscan/agentscan/pkg/agent"
)

// ExampleBasicUsage demonstrates basic usage of secret scanning agents
func ExampleBasicUsage() {
	factory := NewFactory()
	
	// Create both agents with default configuration
	truffleAgent, err := factory.CreateAgent(TruffleHogAgent)
	if err != nil {
		log.Fatalf("Failed to create TruffleHog agent: %v", err)
	}
	
	gitSecretsAgent, err := factory.CreateAgent(GitSecretsAgent)
	if err != nil {
		log.Fatalf("Failed to create git-secrets agent: %v", err)
	}
	
	// Scan configuration
	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/user/repo",
		Branch:  "main",
		Timeout: 10 * time.Minute,
	}
	
	ctx := context.Background()
	
	// Run TruffleHog scan
	fmt.Println("Running TruffleHog scan...")
	truffleResult, err := truffleAgent.Scan(ctx, scanConfig)
	if err != nil {
		log.Printf("TruffleHog scan failed: %v", err)
	} else {
		fmt.Printf("TruffleHog found %d secrets\n", len(truffleResult.Findings))
		for _, finding := range truffleResult.Findings {
			fmt.Printf("- %s: %s (confidence: %.2f)\n", 
				finding.RuleID, finding.Title, finding.Confidence)
		}
	}
	
	// Run git-secrets scan
	fmt.Println("\nRunning git-secrets scan...")
	gitSecretsResult, err := gitSecretsAgent.Scan(ctx, scanConfig)
	if err != nil {
		log.Printf("git-secrets scan failed: %v", err)
	} else {
		fmt.Printf("git-secrets found %d secrets\n", len(gitSecretsResult.Findings))
		for _, finding := range gitSecretsResult.Findings {
			fmt.Printf("- %s: %s (confidence: %.2f)\n", 
				finding.RuleID, finding.Title, finding.Confidence)
		}
	}
}

// ExampleSecureConfiguration demonstrates using secure default configurations
func ExampleSecureConfiguration() {
	factory := NewFactory()
	
	// Create agents with secure configurations
	truffleAgent := factory.CreateTruffleHogAgent(SecureDefaultTruffleHogConfig())
	gitSecretsAgent := factory.CreateGitSecretsAgent(SecureDefaultGitSecretsConfig())
	
	// Scan configuration for a production repository
	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/company/production-app",
		Branch:  "main",
		Timeout: 15 * time.Minute,
	}
	
	ctx := context.Background()
	
	// Run both scans in parallel
	type ScanResult struct {
		AgentName string
		Result    *agent.ScanResult
		Error     error
	}
	
	results := make(chan ScanResult, 2)
	
	// Start TruffleHog scan
	go func() {
		result, err := truffleAgent.Scan(ctx, scanConfig)
		results <- ScanResult{
			AgentName: "TruffleHog",
			Result:    result,
			Error:     err,
		}
	}()
	
	// Start git-secrets scan
	go func() {
		result, err := gitSecretsAgent.Scan(ctx, scanConfig)
		results <- ScanResult{
			AgentName: "git-secrets",
			Result:    result,
			Error:     err,
		}
	}()
	
	// Collect results
	var allFindings []agent.Finding
	for i := 0; i < 2; i++ {
		scanResult := <-results
		if scanResult.Error != nil {
			log.Printf("%s scan failed: %v", scanResult.AgentName, scanResult.Error)
			continue
		}
		
		fmt.Printf("%s scan completed in %v\n", 
			scanResult.AgentName, scanResult.Result.Duration)
		fmt.Printf("Found %d secrets\n", len(scanResult.Result.Findings))
		
		allFindings = append(allFindings, scanResult.Result.Findings...)
	}
	
	// Deduplicate and analyze findings
	uniqueFindings := deduplicateFindings(allFindings)
	fmt.Printf("\nTotal unique secrets found: %d\n", len(uniqueFindings))
	
	// Group by severity
	severityCount := make(map[agent.Severity]int)
	for _, finding := range uniqueFindings {
		severityCount[finding.Severity]++
	}
	
	for severity, count := range severityCount {
		fmt.Printf("- %s: %d\n", severity, count)
	}
}

// ExampleIncrementalScan demonstrates scanning only specific files
func ExampleIncrementalScan() {
	factory := NewFactory()
	
	// Create TruffleHog agent for incremental scanning
	truffleAgent, err := factory.CreateAgent(TruffleHogAgent)
	if err != nil {
		log.Fatalf("Failed to create TruffleHog agent: %v", err)
	}
	
	// Scan only specific files (e.g., changed files in a PR)
	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/user/repo",
		Branch:  "feature-branch",
		Files: []string{
			"src/config.js",
			"config/database.yml",
			".env.example",
		},
		Timeout: 5 * time.Minute,
	}
	
	ctx := context.Background()
	
	fmt.Println("Running incremental scan on specific files...")
	result, err := truffleAgent.Scan(ctx, scanConfig)
	if err != nil {
		log.Fatalf("Incremental scan failed: %v", err)
	}
	
	fmt.Printf("Scanned %d files in %v\n", len(scanConfig.Files), result.Duration)
	fmt.Printf("Found %d secrets\n", len(result.Findings))
	
	for _, finding := range result.Findings {
		fmt.Printf("Secret in %s:%d - %s\n", 
			finding.File, finding.Line, finding.Title)
		fmt.Printf("  Fix: %s\n", finding.Fix.Suggestion)
	}
}

// ExampleCustomConfiguration demonstrates custom agent configuration
func ExampleCustomConfiguration() {
	factory := NewFactory()
	
	// Custom TruffleHog configuration for a specific use case
	truffleConfig := SecureDefaultTruffleHogConfig()
	truffleConfig.MaxDepth = 50 // Only scan last 50 commits
	truffleConfig.IncludeDetectors = []string{
		"aws", "github", "slack", "stripe", // Only scan for these types
	}
	truffleConfig.Whitelist = append(truffleConfig.Whitelist,
		`vendor/.*`,     // Ignore vendor directory
		`node_modules/.*`, // Ignore node_modules
		`.*\.min\.js$`,  // Ignore minified files
	)
	
	// Custom git-secrets configuration
	gitSecretsConfig := SecureDefaultGitSecretsConfig()
	gitSecretsConfig.CustomPatterns = append(gitSecretsConfig.CustomPatterns,
		// Company-specific patterns
		`COMPANY_API_KEY_[A-Za-z0-9]{32}`,
		`INTERNAL_SECRET_[A-Za-z0-9]+`,
	)
	gitSecretsConfig.ScanCommits = false // Only scan working directory
	
	// Create agents with custom configurations
	truffleAgent := factory.CreateTruffleHogAgent(truffleConfig)
	gitSecretsAgent := factory.CreateGitSecretsAgent(gitSecretsConfig)
	
	// Health check before scanning
	ctx := context.Background()
	
	fmt.Println("Performing health checks...")
	if err := truffleAgent.HealthCheck(ctx); err != nil {
		log.Printf("TruffleHog health check failed: %v", err)
	} else {
		fmt.Println("TruffleHog is healthy")
	}
	
	if err := gitSecretsAgent.HealthCheck(ctx); err != nil {
		log.Printf("git-secrets health check failed: %v", err)
	} else {
		fmt.Println("git-secrets is healthy")
	}
	
	// Get agent information
	truffleInfo := truffleAgent.GetConfig()
	gitSecretsInfo := gitSecretsAgent.GetConfig()
	
	fmt.Printf("\nTruffleHog supports %d languages\n", len(truffleInfo.SupportedLangs))
	fmt.Printf("git-secrets supports %d languages\n", len(gitSecretsInfo.SupportedLangs))
}

// ExampleErrorHandling demonstrates proper error handling
func ExampleErrorHandling() {
	factory := NewFactory()
	
	// Create agent
	agent, err := factory.CreateAgent(TruffleHogAgent)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	
	// Invalid scan configuration
	scanConfig := agent.ScanConfig{
		RepoURL: "", // Empty URL will cause error
		Branch:  "main",
		Timeout: 1 * time.Minute,
	}
	
	ctx := context.Background()
	
	result, err := agent.Scan(ctx, scanConfig)
	if err != nil {
		fmt.Printf("Scan failed as expected: %v\n", err)
		
		// Check if we got a partial result
		if result != nil {
			fmt.Printf("Scan status: %s\n", result.Status)
			if result.Error != "" {
				fmt.Printf("Error details: %s\n", result.Error)
			}
		}
		return
	}
	
	// Handle successful scan
	fmt.Printf("Scan completed successfully with %d findings\n", len(result.Findings))
}

// deduplicateFindings removes duplicate findings based on file, line, and rule ID
func deduplicateFindings(findings []agent.Finding) []agent.Finding {
	seen := make(map[string]bool)
	var unique []agent.Finding
	
	for _, finding := range findings {
		// Create a key based on file, line, and rule ID
		key := fmt.Sprintf("%s:%d:%s", finding.File, finding.Line, finding.RuleID)
		
		if !seen[key] {
			seen[key] = true
			unique = append(unique, finding)
		}
	}
	
	return unique
}

// ExampleBatchScanning demonstrates scanning multiple repositories
func ExampleBatchScanning() {
	factory := NewFactory()
	
	// Create agents
	agents := factory.CreateAllAgents()
	
	// List of repositories to scan
	repositories := []string{
		"https://github.com/user/repo1",
		"https://github.com/user/repo2",
		"https://github.com/user/repo3",
	}
	
	ctx := context.Background()
	
	for _, repoURL := range repositories {
		fmt.Printf("\nScanning repository: %s\n", repoURL)
		
		scanConfig := agent.ScanConfig{
			RepoURL: repoURL,
			Branch:  "main",
			Timeout: 10 * time.Minute,
		}
		
		var totalFindings int
		
		for _, securityAgent := range agents {
			agentConfig := securityAgent.GetConfig()
			fmt.Printf("Running %s...", agentConfig.Name)
			
			result, err := securityAgent.Scan(ctx, scanConfig)
			if err != nil {
				fmt.Printf(" failed: %v\n", err)
				continue
			}
			
			fmt.Printf(" found %d secrets\n", len(result.Findings))
			totalFindings += len(result.Findings)
		}
		
		fmt.Printf("Total secrets found in %s: %d\n", repoURL, totalFindings)
	}
}