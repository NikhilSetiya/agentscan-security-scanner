package database

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/errors"
)

// Migrator handles database migrations
type Migrator struct {
	migrate *migrate.Migrate
	db      *sql.DB
}

// NewMigrator creates a new database migrator
func NewMigrator(cfg *config.DatabaseConfig, migrationsPath string) (*Migrator, error) {
	if cfg == nil {
		return nil, errors.NewValidationError("database configuration is required")
	}

	if migrationsPath == "" {
		migrationsPath = "migrations"
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, errors.NewInternalError("failed to open database connection").WithCause(err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, errors.NewInternalError("failed to ping database").WithCause(err)
	}

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		db.Close()
		return nil, errors.NewInternalError("failed to create postgres driver").WithCause(err)
	}

	// Get absolute path for migrations
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		db.Close()
		return nil, errors.NewInternalError("failed to get absolute path for migrations").WithCause(err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", absPath),
		"postgres",
		driver,
	)
	if err != nil {
		db.Close()
		return nil, errors.NewInternalError("failed to create migrate instance").WithCause(err)
	}

	return &Migrator{
		migrate: m,
		db:      db,
	}, nil
}

// Close closes the migrator and database connection
func (m *Migrator) Close() error {
	var err error
	if m.migrate != nil {
		if sourceErr, dbErr := m.migrate.Close(); sourceErr != nil || dbErr != nil {
			err = fmt.Errorf("source error: %v, db error: %v", sourceErr, dbErr)
		}
	}
	if m.db != nil {
		if dbErr := m.db.Close(); dbErr != nil {
			if err != nil {
				err = fmt.Errorf("%v, close error: %v", err, dbErr)
			} else {
				err = dbErr
			}
		}
	}
	return err
}

// Up runs all available migrations
func (m *Migrator) Up() error {
	if err := m.migrate.Up(); err != nil {
		if err == migrate.ErrNoChange {
			return nil // No migrations to run
		}
		return errors.NewInternalError("failed to run migrations").WithCause(err)
	}
	return nil
}

// Down rolls back all migrations
func (m *Migrator) Down() error {
	if err := m.migrate.Down(); err != nil {
		if err == migrate.ErrNoChange {
			return nil // No migrations to rollback
		}
		return errors.NewInternalError("failed to rollback migrations").WithCause(err)
	}
	return nil
}

// Steps runs n migrations up (positive) or down (negative)
func (m *Migrator) Steps(n int) error {
	if err := m.migrate.Steps(n); err != nil {
		if err == migrate.ErrNoChange {
			return nil // No migrations to run
		}
		return errors.NewInternalError("failed to run migration steps").WithCause(err)
	}
	return nil
}

// Migrate to a specific version
func (m *Migrator) Migrate(version uint) error {
	if err := m.migrate.Migrate(version); err != nil {
		if err == migrate.ErrNoChange {
			return nil // Already at target version
		}
		return errors.NewInternalError("failed to migrate to version").WithCause(err)
	}
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			return 0, false, nil // No migrations have been run
		}
		return 0, false, errors.NewInternalError("failed to get migration version").WithCause(err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(version int) error {
	if err := m.migrate.Force(version); err != nil {
		return errors.NewInternalError("failed to force migration version").WithCause(err)
	}
	return nil
}

// Drop drops the entire database schema
func (m *Migrator) Drop() error {
	if err := m.migrate.Drop(); err != nil {
		return errors.NewInternalError("failed to drop database schema").WithCause(err)
	}
	return nil
}