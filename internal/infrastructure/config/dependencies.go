package config

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/handlers"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/services"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
	domainServices "github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/services"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/infrastructure/cache"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/infrastructure/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/infrastructure/external"
)

// Dependencies holds all application dependencies
type Dependencies struct {
	// Database
	DB *sqlx.DB
	
	// Cache
	Cache cache.Cache
	
	// Repositories
	UserRepository       repositories.UserRepository
	RepositoryRepository repositories.RepositoryRepository
	ScanJobRepository    repositories.ScanJobRepository
	FindingRepository    repositories.FindingRepository
	
	// Domain Services
	UserService       *domainServices.UserService
	RepositoryService *domainServices.RepositoryService
	ScanService       *domainServices.ScanService
	
	// Application Handlers
	UserCommandHandler       *handlers.UserCommandHandler
	UserQueryHandler         *handlers.UserQueryHandler
	RepositoryCommandHandler *handlers.RepositoryCommandHandler
	RepositoryQueryHandler   *handlers.RepositoryQueryHandler
	ScanCommandHandler       *handlers.ScanCommandHandler
	ScanQueryHandler         *handlers.ScanQueryHandler
	
	// Application Service
	ApplicationService *services.ApplicationService
	
	// External Clients
	GitHubClient *external.GitHubClient
}\n\n// Config holds configuration for dependency injection\ntype Config struct {\n\tDatabaseURL    string\n\tRedisURL       string\n\tGitHubToken    string\n\tCacheEnabled   bool\n\tCacheDefaultTTL time.Duration\n}\n\n// NewDependencies creates and wires all application dependencies
func NewDependencies(config *Config) (*Dependencies, error) {
	deps := &Dependencies{}
	
	// Initialize database
	db, err := initDatabase(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	deps.DB = db
	
	// Initialize cache
	cacheInstance, err := initCache(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}
	deps.Cache = cacheInstance
	
	// Initialize repositories
	deps.UserRepository = database.NewUserRepository(db)
	deps.RepositoryRepository = database.NewRepositoryRepository(db)
	deps.ScanJobRepository = database.NewScanJobRepository(db)
	// deps.FindingRepository = database.NewFindingRepository(db) // TODO: Implement
	
	// Initialize domain services
	deps.UserService = domainServices.NewUserService(deps.UserRepository)
	deps.RepositoryService = domainServices.NewRepositoryService(deps.RepositoryRepository)
	deps.ScanService = domainServices.NewScanService(deps.ScanJobRepository, deps.RepositoryRepository, deps.UserRepository)
	
	// Initialize application handlers
	deps.UserCommandHandler = handlers.NewUserCommandHandler(deps.UserService)
	deps.UserQueryHandler = handlers.NewUserQueryHandler(deps.UserService)
	deps.RepositoryCommandHandler = handlers.NewRepositoryCommandHandler(deps.RepositoryService)
	deps.RepositoryQueryHandler = handlers.NewRepositoryQueryHandler(deps.RepositoryService)
	deps.ScanCommandHandler = handlers.NewScanCommandHandler(deps.ScanService)
	deps.ScanQueryHandler = handlers.NewScanQueryHandler(deps.ScanService)
	
	// Initialize application service
	deps.ApplicationService = services.NewApplicationService(
		deps.UserCommandHandler,
		deps.UserQueryHandler,
		deps.RepositoryCommandHandler,
		deps.RepositoryQueryHandler,
		deps.ScanCommandHandler,
		deps.ScanQueryHandler,
	)
	
	// Initialize external clients
	if config.GitHubToken != "" {
		deps.GitHubClient = external.NewGitHubClient(config.GitHubToken)
	}
	
	return deps, nil
}\n\n// Close closes all resources\nfunc (d *Dependencies) Close() error {\n\tif d.DB != nil {\n\t\treturn d.DB.Close()\n\t}\n\treturn nil\n}\n\n// initDatabase initializes the database connection
func initDatabase(databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	return db, nil
}

// initCache initializes the cache
func initCache(config *Config) (cache.Cache, error) {
	if !config.CacheEnabled {
		return cache.NewMemoryCache(config.CacheDefaultTTL), nil
	}
	
	if config.RedisURL == "" {
		return cache.NewMemoryCache(config.CacheDefaultTTL), nil
	}
	
	// Parse Redis URL and create client
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	
	redisClient := redis.NewClient(opt)
	
	// Test Redis connection
	if err := redisClient.Ping(redisClient.Context()).Err(); err != nil {
		// Fall back to memory cache if Redis is not available
		return cache.NewMemoryCache(config.CacheDefaultTTL), nil
	}
	
	return cache.NewRedisCache(redisClient, config.CacheDefaultTTL, "agentscan:"), nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		CacheEnabled:    true,
		CacheDefaultTTL: 15 * time.Minute,
	}
}"