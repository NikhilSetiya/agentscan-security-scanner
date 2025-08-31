package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/your-org/agentscan/internal/config"
)

func main() {
	var (
		envFile     = flag.String("env", "", "Environment file to load")
		validate    = flag.Bool("validate", false, "Validate environment variables")
		generate    = flag.Bool("generate", false, "Generate environment template")
		docs        = flag.Bool("docs", false, "Generate environment documentation")
		output      = flag.String("output", "", "Output file for generated content")
		verbose     = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	if *generate {
		if err := generateTemplate(*output, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating template: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *docs {
		if err := generateDocs(*output, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating docs: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *validate {
		if err := validateEnv(*envFile, *verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Environment validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Environment validation passed")
		return
	}

	// Default: show environment status
	if err := showEnvStatus(*envFile, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func generateTemplate(output string, verbose bool) error {
	if output == "" {
		output = ".env.template"
	}

	envConfig := &config.EnvConfig{}
	
	if err := config.SaveEnvTemplate(output, envConfig); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	if verbose {
		fmt.Printf("✅ Environment template generated: %s\n", output)
	} else {
		fmt.Printf("Generated: %s\n", output)
	}

	return nil
}

func generateDocs(output string, verbose bool) error {
	if output == "" {
		output = "ENVIRONMENT.md"
	}

	envConfig := &config.EnvConfig{}
	generator := config.NewEnvDocGenerator(envConfig)
	
	docs, err := generator.GenerateMarkdown()
	if err != nil {
		return fmt.Errorf("failed to generate docs: %w", err)
	}

	if err := os.WriteFile(output, []byte(docs), 0644); err != nil {
		return fmt.Errorf("failed to write docs: %w", err)
	}

	if verbose {
		fmt.Printf("✅ Environment documentation generated: %s\n", output)
	} else {
		fmt.Printf("Generated: %s\n", output)
	}

	return nil
}

func validateEnv(envFile string, verbose bool) error {
	// Load environment manager
	envManager := config.NewEnvManager()
	
	// Load environment files
	if envFile != "" {
		if err := envManager.LoadEnvFiles(envFile); err != nil {
			return fmt.Errorf("failed to load env file: %w", err)
		}
	} else {
		if err := envManager.LoadEnvFiles(); err != nil {
			return fmt.Errorf("failed to load env files: %w", err)
		}
	}

	if verbose {
		loadedFiles := envManager.GetLoadedFiles()
		if len(loadedFiles) > 0 {
			fmt.Printf("📁 Loaded environment files: %v\n", loadedFiles)
		} else {
			fmt.Println("📁 No environment files loaded (using system environment)")
		}
	}

	// Load and validate configuration
	envConfig, err := config.LoadEnvConfig()
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("🔧 Environment: %s\n", envConfig.Environment)
		fmt.Printf("🚀 Application: %s v%s\n", envConfig.AppName, envConfig.AppVersion)
		fmt.Printf("🌐 Server: %s:%d\n", envConfig.Host, envConfig.Port)
		
		if envConfig.HTTPSEnabled {
			fmt.Println("🔒 HTTPS: Enabled")
		} else {
			fmt.Println("🔓 HTTPS: Disabled")
		}
		
		if envConfig.MetricsEnabled {
			fmt.Printf("📊 Metrics: Enabled (port %d)\n", envConfig.MetricsPort)
		} else {
			fmt.Println("📊 Metrics: Disabled")
		}
		
		fmt.Printf("📝 Log Level: %s\n", envConfig.LogLevel)
	}

	return nil
}

func showEnvStatus(envFile string, verbose bool) error {
	fmt.Println("🔍 AgentScan Environment Status")
	fmt.Println("================================")

	// Check current environment
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}
	fmt.Printf("Environment: %s\n", env)

	// Check for environment files
	envFiles := []string{".env.local", ".env." + env, ".env"}
	fmt.Println("\nEnvironment Files:")
	
	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("  ✅ %s (exists)\n", file)
		} else {
			fmt.Printf("  ❌ %s (not found)\n", file)
		}
	}

	// Check critical environment variables
	fmt.Println("\nCritical Variables:")
	criticalVars := []string{
		"DATABASE_URL",
		"SUPABASE_URL",
		"SUPABASE_SERVICE_ROLE_KEY",
		"JWT_SECRET",
	}

	allSet := true
	for _, varName := range criticalVars {
		value := os.Getenv(varName)
		if value == "" {
			fmt.Printf("  ❌ %s (not set)\n", varName)
			allSet = false
		} else {
			if verbose {
				// Show partial value for security
				displayValue := value
				if len(value) > 10 {
					displayValue = value[:6] + "..." + value[len(value)-4:]
				}
				fmt.Printf("  ✅ %s (%s)\n", varName, displayValue)
			} else {
				fmt.Printf("  ✅ %s (set)\n", varName)
			}
		}
	}

	if !allSet {
		fmt.Println("\n⚠️  Some critical environment variables are missing.")
		fmt.Println("   Run with -generate to create a template file.")
		return fmt.Errorf("missing critical environment variables")
	}

	fmt.Println("\n✅ Environment appears to be configured correctly.")
	fmt.Println("   Run with -validate to perform full validation.")

	return nil
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "AgentScan Environment Configuration Tool\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                          # Show environment status\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s -validate                # Validate environment\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s -generate                # Generate .env template\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s -docs                    # Generate documentation\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s -env .env.production     # Use specific env file\n", filepath.Base(os.Args[0]))
	}
}