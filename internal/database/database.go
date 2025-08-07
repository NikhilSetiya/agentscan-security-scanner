package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/errors"
)

// DB wraps the database connection with additional functionality
type DB struct {
	*sqlx.DB
	config *config.DatabaseConfig
}

// New creates a new database connection
func New(cfg *config.DatabaseConfig) (*DB, error) {
	if cfg == nil {
		return nil, errors.NewValidationError("database configuration is required")
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	// Open database connection
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, errors.NewInternalError("failed to connect to database").WithCause(err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.NewInternalError("failed to ping database").WithCause(err)
	}

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// Health checks the database connection health
func (db *DB) Health(ctx context.Context) error {
	if db.DB == nil {
		return errors.NewInternalError("database connection is nil")
	}

	if err := db.PingContext(ctx); err != nil {
		return errors.NewInternalError("database health check failed").WithCause(err)
	}

	return nil
}

// BeginTx starts a new transaction with the given options
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	tx, err := db.DB.BeginTxx(ctx, opts)
	if err != nil {
		return nil, errors.NewInternalError("failed to begin transaction").WithCause(err)
	}
	return tx, nil
}

// WithTransaction executes a function within a database transaction
func (db *DB) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.NewInternalError("failed to rollback transaction").
				WithCause(fmt.Errorf("original error: %v, rollback error: %v", err, rbErr))
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.NewInternalError("failed to commit transaction").WithCause(err)
	}

	return nil
}

// Stats returns database connection statistics
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// Config returns the database configuration
func (db *DB) Config() *config.DatabaseConfig {
	return db.config
}