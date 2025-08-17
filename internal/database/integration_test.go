// +build integration

package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// TestDatabaseIntegration tests the complete database functionality
// Run with: go test -tags=integration ./internal/database
func TestDatabaseIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run.")
	}

	// Load test configuration
	cfg := &config.DatabaseConfig{
		Host:            getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            5432,
		Name:            getEnvOrDefault("TEST_DB_NAME", "agentscan_test"),
		User:            getEnvOrDefault("TEST_DB_USER", "agentscan"),
		Password:        getEnvOrDefault("TEST_DB_PASSWORD", "agentscan_dev_password"),
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// Create database connection
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.Close()

	// Test database health
	ctx := context.Background()
	if err := db.Health(ctx); err != nil {
		t.Fatalf("Database health check failed: %v", err)
	}

	// Run migration tests
	t.Run("Migrations", func(t *testing.T) {
		testMigrations(t, cfg)
	})

	// Run repository tests
	t.Run("UserRepository", func(t *testing.T) {
		testUserRepository(t, db)
	})

	t.Run("ScanJobRepository", func(t *testing.T) {
		testScanJobRepository(t, db)
	})

	t.Run("FindingRepository", func(t *testing.T) {
		testFindingRepository(t, db)
	})
}

func testMigrations(t *testing.T, cfg *config.DatabaseConfig) {
	migrator, err := NewMigrator(cfg, "../../migrations")
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	// Test migration up
	if err := migrator.Up(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test getting version
	version, dirty, err := migrator.Version()
	if err != nil {
		t.Fatalf("Failed to get migration version: %v", err)
	}

	if dirty {
		t.Error("Database is in dirty state after migration")
	}

	if version == 0 {
		t.Error("Expected migration version > 0")
	}

	t.Logf("Migration version: %d", version)
}

func testUserRepository(t *testing.T, db *DB) {
	repo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &types.User{
		Email: "test@example.com",
		Name:  "Test User",
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user was created with ID
	if user.ID == uuid.Nil {
		t.Error("User ID should be set after creation")
	}

	// Get user by ID
	retrieved, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}

	// Get user by email
	byEmail, err := repo.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}

	if byEmail.ID != user.ID {
		t.Error("User retrieved by email should have same ID")
	}

	// Update user
	retrieved.Name = "Updated Name"
	if err := repo.Update(ctx, retrieved); err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify update
	updated, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", updated.Name)
	}

	// Delete user
	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify deletion
	_, err = repo.GetByID(ctx, user.ID)
	if err == nil {
		t.Error("Expected error when getting deleted user")
	}
}

func testScanJobRepository(t *testing.T, db *DB) {
	repo := NewScanJobRepository(db)
	ctx := context.Background()

	// Create test scan job
	job := &types.ScanJob{
		RepositoryID:    uuid.New(),
		Branch:          "main",
		CommitSHA:       "abc123",
		ScanType:        types.ScanTypeFull,
		Priority:        types.PriorityMedium,
		Status:          types.ScanJobStatusQueued,
		AgentsRequested: []string{"semgrep", "eslint"},
	}

	if err := repo.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create scan job: %v", err)
	}

	// Get scan job by ID
	retrieved, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to get scan job by ID: %v", err)
	}

	if retrieved.Branch != job.Branch {
		t.Errorf("Expected branch %s, got %s", job.Branch, retrieved.Branch)
	}

	// Update status
	if err := repo.UpdateStatus(ctx, job.ID, types.ScanJobStatusRunning); err != nil {
		t.Fatalf("Failed to update scan job status: %v", err)
	}

	// Set started
	if err := repo.SetStarted(ctx, job.ID); err != nil {
		t.Fatalf("Failed to set scan job as started: %v", err)
	}

	// Set completed
	if err := repo.SetCompleted(ctx, job.ID); err != nil {
		t.Fatalf("Failed to set scan job as completed: %v", err)
	}

	// Verify final status
	final, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to get final scan job: %v", err)
	}

	if final.Status != types.ScanJobStatusCompleted {
		t.Errorf("Expected status %s, got %s", types.ScanJobStatusCompleted, final.Status)
	}

	if final.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func testFindingRepository(t *testing.T, db *DB) {
	repo := NewFindingRepository(db)
	ctx := context.Background()

	// Create test finding
	finding := &types.Finding{
		ScanResultID: uuid.New(),
		ScanJobID:    uuid.New(),
		Tool:         "semgrep",
		RuleID:       "javascript.lang.security.audit.xss.direct-response-write",
		Severity:     "high",
		Category:     "xss",
		Title:        "Potential XSS vulnerability",
		Description:  "User input is directly written to HTTP response",
		FilePath:     "src/app.js",
		LineNumber:   42,
		Confidence:   0.9,
		Status:       types.FindingStatusOpen,
	}

	if err := repo.Create(ctx, finding); err != nil {
		t.Fatalf("Failed to create finding: %v", err)
	}

	// Get finding by ID
	retrieved, err := repo.GetByID(ctx, finding.ID)
	if err != nil {
		t.Fatalf("Failed to get finding by ID: %v", err)
	}

	if retrieved.Tool != finding.Tool {
		t.Errorf("Expected tool %s, got %s", finding.Tool, retrieved.Tool)
	}

	// Update status
	if err := repo.UpdateStatus(ctx, finding.ID, types.FindingStatusFixed); err != nil {
		t.Fatalf("Failed to update finding status: %v", err)
	}

	// Verify status update
	updated, err := repo.GetByID(ctx, finding.ID)
	if err != nil {
		t.Fatalf("Failed to get updated finding: %v", err)
	}

	if updated.Status != types.FindingStatusFixed {
		t.Errorf("Expected status %s, got %s", types.FindingStatusFixed, updated.Status)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}