package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// RepositoryFactory creates and manages repository implementations
type RepositoryFactory struct {
	db *sqlx.DB
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(config DatabaseConfig) (*RepositoryFactory, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)
	
	// Open database connection
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	
	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	return &RepositoryFactory{
		db: db,
	}, nil
}

// GetDB returns the database connection
func (f *RepositoryFactory) GetDB() *sqlx.DB {
	return f.db
}

// Close closes the database connection
func (f *RepositoryFactory) Close() error {
	return f.db.Close()
}

// Health checks database health
func (f *RepositoryFactory) Health() error {
	return f.db.Ping()
}

// CreateUserRepository creates a user repository implementation
func (f *RepositoryFactory) CreateUserRepository() repositories.UserRepository {
	return NewUserRepository(f.db)
}

// CreateRepositoryRepository creates a repository repository implementation
func (f *RepositoryFactory) CreateRepositoryRepository() repositories.RepositoryRepository {
	return NewRepositoryRepository(f.db)
}

// CreateScanJobRepository creates a scan job repository implementation
func (f *RepositoryFactory) CreateScanJobRepository() repositories.ScanJobRepository {
	// TODO: Implement ScanJobRepositoryImpl
	return nil
}

// CreateFindingRepository creates a finding repository implementation
func (f *RepositoryFactory) CreateFindingRepository() repositories.FindingRepository {
	// TODO: Implement FindingRepositoryImpl
	return nil
}

// Transaction executes a function within a database transaction
func (f *RepositoryFactory) Transaction(fn func(*sqlx.Tx) error) error {
	tx, err := f.db.Beginx()
	if err != nil {
		return err
	}
	
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	
	err = fn(tx)
	return err
}

// Migrate runs database migrations
func (f *RepositoryFactory) Migrate() error {
	// TODO: Implement database migrations
	// This would typically use a migration library like golang-migrate
	return nil
}

// DefaultDatabaseConfig returns default database configuration
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "",
		Database:        "agentscan",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 15,
	}
}

// NewOptimizedDB creates an optimized database connection
func NewOptimizedDB(config DatabaseConfig) (*sqlx.DB, error) {
	factory, err := NewRepositoryFactory(config)
	if err != nil {
		return nil, err
	}
	
	return factory.GetDB(), nil
}