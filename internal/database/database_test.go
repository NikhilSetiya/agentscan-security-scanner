package database

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/errors"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.DatabaseConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &config.DatabaseConfig{
				Host:            "localhost",
				Port:            5432,
				Name:            "test_db",
				User:            "test_user",
				Password:        "test_password",
				SSLMode:         "disable",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
			},
			wantErr: true, // Will fail without actual DB, which is expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if db != nil {
				db.Close()
			}
		})
	}
}

func TestDB_Health(t *testing.T) {
	// This test requires a real database connection
	// In a real test environment, you would set up a test database
	t.Skip("Skipping database health test - requires real database")

	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		Name:            "test_db",
		User:            "test_user",
		Password:        "test_password",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Health(ctx); err != nil {
		t.Errorf("Health() error = %v", err)
	}
}

func TestDB_WithTransaction(t *testing.T) {
	// This test requires a real database connection
	t.Skip("Skipping transaction test - requires real database")

	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		Name:            "test_db",
		User:            "test_user",
		Password:        "test_password",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test successful transaction
	err = db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// Perform some database operations
		return nil
	})
	if err != nil {
		t.Errorf("WithTransaction() error = %v", err)
	}

	// Test transaction rollback
	testErr := errors.NewInternalError("test error")
	err = db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return testErr
	})
	if err != testErr {
		t.Errorf("WithTransaction() should return the original error, got %v", err)
	}
}