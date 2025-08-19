package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/secrets"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
)

func main() {
	var (
		supabaseURL    = flag.String("supabase-url", "", "Supabase project URL")
		serviceRoleKey = flag.String("service-role-key", "", "Supabase service role key")
		dryRun         = flag.Bool("dry-run", false, "Show what would be migrated without actually doing it")
		listSecrets    = flag.Bool("list", false, "List existing secrets")
	)
	flag.Parse()

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Get Supabase configuration from environment if not provided
	if *supabaseURL == "" {
		*supabaseURL = os.Getenv("SUPABASE_URL")
	}
	if *serviceRoleKey == "" {
		*serviceRoleKey = os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	}

	if *supabaseURL == "" || *serviceRoleKey == "" {
		log.Fatal("Supabase URL and service role key are required")
	}

	// Create secrets manager
	secretsManager := secrets.NewSupabaseSecretsManager(*supabaseURL, *serviceRoleKey, logger)

	ctx := context.Background()

	// List existing secrets if requested
	if *listSecrets {
		secretNames, err := secretsManager.ListSecrets(ctx)
		if err != nil {
			log.Fatalf("Failed to list secrets: %v", err)
		}

		fmt.Println("Existing secrets:")
		for _, name := range secretNames {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	// Load current configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Show what would be migrated
	secretsToMigrate := map[string]string{
		"JWT_SECRET":        cfg.Auth.JWTSecret,
		"GITHUB_CLIENT_ID":  cfg.Auth.GitHubClientID,
		"GITHUB_SECRET":     cfg.Auth.GitHubSecret,
		"GITLAB_CLIENT_ID":  cfg.Auth.GitLabClientID,
		"GITLAB_SECRET":     cfg.Auth.GitLabSecret,
		"DB_PASSWORD":       cfg.Database.Password,
		"REDIS_PASSWORD":    cfg.Redis.Password,
	}

	fmt.Println("Secrets to migrate:")
	for name, value := range secretsToMigrate {
		if value != "" {
			if *dryRun {
				fmt.Printf("  - %s: [REDACTED] (length: %d)\n", name, len(value))
			} else {
				fmt.Printf("  - %s\n", name)
			}
		} else {
			fmt.Printf("  - %s: [EMPTY - SKIPPED]\n", name)
		}
	}

	if *dryRun {
		fmt.Println("\nDry run mode - no secrets were actually migrated")
		return
	}

	// Confirm migration
	fmt.Print("\nProceed with migration? (y/N): ")
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Migration cancelled")
		return
	}

	// Perform migration
	fmt.Println("\nMigrating secrets...")
	if err := secretsManager.MigrateFromEnv(ctx, cfg); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("✅ Migration completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Update your application to load secrets from Supabase")
	fmt.Println("2. Remove sensitive values from environment variables")
	fmt.Println("3. Test the application with the new secrets configuration")
}