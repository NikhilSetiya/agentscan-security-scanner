package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/agentscan/internal/config"
	"github.com/your-org/agentscan/internal/shared/logging"
)

// DatabaseIntegrationTest provides comprehensive database integration testing
type DatabaseIntegrationTest struct {
	db     *sql.DB
	config *config.DatabaseConfig
	logger logging.Logger
}

// NewDatabaseIntegrationTest creates a new database integration test suite
func NewDatabaseIntegrationTest(db *sql.DB, config *config.DatabaseConfig) *DatabaseIntegrationTest {
	return &DatabaseIntegrationTest{
		db:     db,
		config: config,
		logger: logging.GetLogger(),
	}
}

// TestDatabaseConnectivity tests basic database connectivity
func (dit *DatabaseIntegrationTest) TestDatabaseConnectivity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dit.logger.Info("Starting database connectivity test")

	// Test basic ping
	err := dit.db.PingContext(ctx)
	require.NoError(t, err, "Database ping should succeed")

	// Test simple query
	var result int
	err = dit.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	require.NoError(t, err, "Simple query should succeed")
	assert.Equal(t, 1, result, "Query result should be 1")

	dit.logger.Info("Database connectivity test completed successfully")
}

// TestConnectionPooling tests database connection pool behavior
func (dit *DatabaseIntegrationTest) TestConnectionPooling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dit.logger.Info("Starting connection pool test")

	// Get initial stats
	initialStats := dit.db.Stats()
	dit.logger.Info("Initial connection pool stats",
		"open_connections", initialStats.OpenConnections,
		"in_use", initialStats.InUse,
		"idle", initialStats.Idle,
	)

	// Test concurrent connections
	concurrency := 20
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Simulate work with database connection
			conn, err := dit.db.Conn(ctx)
			if err != nil {
				errors <- fmt.Errorf("worker %d: failed to get connection: %w", id, err)
				return
			}
			defer conn.Close()

			// Hold connection for a short time
			time.Sleep(100 * time.Millisecond)

			// Execute a query
			var result int
			err = conn.QueryRowContext(ctx, "SELECT $1", id).Scan(&result)
			if err != nil {
				errors <- fmt.Errorf("worker %d: query failed: %w", id, err)
				return
			}

			if result != id {
				errors <- fmt.Errorf("worker %d: unexpected result %d", id, result)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Connection pool error: %v", err)
		errorCount++
	}

	// Get final stats
	finalStats := dit.db.Stats()
	dit.logger.Info("Final connection pool stats",
		"open_connections", finalStats.OpenConnections,
		"in_use", finalStats.InUse,
		"idle", finalStats.Idle,
		"wait_count", finalStats.WaitCount,
		"wait_duration", finalStats.WaitDuration,
	)

	assert.Equal(t, 0, errorCount, "No connection pool errors should occur")
	assert.LessOrEqual(t, finalStats.OpenConnections, dit.config.MaxOpenConns, 
		"Open connections should not exceed max")

	dit.logger.Info("Connection pool test completed successfully")
}

// TestTransactionPerformance tests transaction performance and rollback behavior
func (dit *DatabaseIntegrationTest) TestTransactionPerformance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dit.logger.Info("Starting transaction performance test")

	// Create test table
	_, err := dit.db.ExecContext(ctx, `
		CREATE TEMPORARY TABLE test_transactions (
			id SERIAL PRIMARY KEY,
			data TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err, "Test table creation should succeed")

	// Test successful transaction
	start := time.Now()
	tx, err := dit.db.BeginTx(ctx, nil)
	require.NoError(t, err, "Transaction begin should succeed")

	for i := 0; i < 100; i++ {
		_, err = tx.ExecContext(ctx, 
			"INSERT INTO test_transactions (data) VALUES ($1)", 
			fmt.Sprintf("test_data_%d", i))
		require.NoError(t, err, "Insert should succeed")
	}

	err = tx.Commit()
	require.NoError(t, err, "Transaction commit should succeed")
	
	commitDuration := time.Since(start)
	dit.logger.Info("Transaction commit performance",
		"duration", commitDuration,
		"operations", 100,
	)

	// Verify data was inserted
	var count int
	err = dit.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_transactions").Scan(&count)
	require.NoError(t, err, "Count query should succeed")
	assert.Equal(t, 100, count, "All records should be inserted")

	// Test transaction rollback
	start = time.Now()
	tx, err = dit.db.BeginTx(ctx, nil)
	require.NoError(t, err, "Transaction begin should succeed")

	for i := 0; i < 50; i++ {
		_, err = tx.ExecContext(ctx, 
			"INSERT INTO test_transactions (data) VALUES ($1)", 
			fmt.Sprintf("rollback_data_%d", i))
		require.NoError(t, err, "Insert should succeed")
	}

	err = tx.Rollback()
	require.NoError(t, err, "Transaction rollback should succeed")
	
	rollbackDuration := time.Since(start)
	dit.logger.Info("Transaction rollback performance",
		"duration", rollbackDuration,
		"operations", 50,
	)

	// Verify data was not inserted
	err = dit.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_transactions").Scan(&count)
	require.NoError(t, err, "Count query should succeed")
	assert.Equal(t, 100, count, "Rollback should not affect record count")

	dit.logger.Info("Transaction performance test completed successfully")
}

// TestQueryPerformance tests query performance with various scenarios
func (dit *DatabaseIntegrationTest) TestQueryPerformance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dit.logger.Info("Starting query performance test")

	// Create test table with indexes
	_, err := dit.db.ExecContext(ctx, `
		CREATE TEMPORARY TABLE test_performance (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(255),
			status VARCHAR(20),
			score INTEGER,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err, "Test table creation should succeed")

	// Create indexes
	_, err = dit.db.ExecContext(ctx, "CREATE INDEX idx_test_email ON test_performance(email)")
	require.NoError(t, err, "Email index creation should succeed")

	_, err = dit.db.ExecContext(ctx, "CREATE INDEX idx_test_status_score ON test_performance(status, score)")
	require.NoError(t, err, "Composite index creation should succeed")

	// Insert test data
	dit.logger.Info("Inserting test data for performance testing")
	start := time.Now()
	
	tx, err := dit.db.BeginTx(ctx, nil)
	require.NoError(t, err, "Transaction begin should succeed")

	stmt, err := tx.PrepareContext(ctx, 
		"INSERT INTO test_performance (name, email, status, score) VALUES ($1, $2, $3, $4)")
	require.NoError(t, err, "Prepared statement should succeed")

	for i := 0; i < 10000; i++ {
		_, err = stmt.ExecContext(ctx,
			fmt.Sprintf("User %d", i),
			fmt.Sprintf("user%d@example.com", i),
			[]string{"active", "inactive", "pending"}[i%3],
			rand.Intn(1000),
		)
		require.NoError(t, err, "Insert should succeed")
	}

	err = stmt.Close()
	require.NoError(t, err, "Statement close should succeed")

	err = tx.Commit()
	require.NoError(t, err, "Transaction commit should succeed")

	insertDuration := time.Since(start)
	dit.logger.Info("Bulk insert performance",
		"duration", insertDuration,
		"records", 10000,
		"records_per_second", float64(10000)/insertDuration.Seconds(),
	)

	// Test various query patterns
	testCases := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{
			name:  "Primary key lookup",
			query: "SELECT * FROM test_performance WHERE id = $1",
			args:  []interface{}{5000},
		},
		{
			name:  "Indexed email lookup",
			query: "SELECT * FROM test_performance WHERE email = $1",
			args:  []interface{}{"user5000@example.com"},
		},
		{
			name:  "Composite index query",
			query: "SELECT * FROM test_performance WHERE status = $1 AND score > $2",
			args:  []interface{}{"active", 500},
		},
		{
			name:  "Range query with limit",
			query: "SELECT * FROM test_performance WHERE score BETWEEN $1 AND $2 ORDER BY score LIMIT 100",
			args:  []interface{}{400, 600},
		},
		{
			name:  "Aggregation query",
			query: "SELECT status, COUNT(*), AVG(score) FROM test_performance GROUP BY status",
			args:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			
			rows, err := dit.db.QueryContext(ctx, tc.query, tc.args...)
			require.NoError(t, err, "Query should succeed")
			
			var rowCount int
			for rows.Next() {
				rowCount++
			}
			rows.Close()
			
			duration := time.Since(start)
			dit.logger.Info("Query performance",
				"test", tc.name,
				"duration", duration,
				"rows_returned", rowCount,
			)
			
			// Performance assertions
			assert.Less(t, duration, 1*time.Second, "Query should complete within 1 second")
		})
	}

	dit.logger.Info("Query performance test completed successfully")
}

// TestDatabaseResilience tests database resilience and error handling
func (dit *DatabaseIntegrationTest) TestDatabaseResilience(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dit.logger.Info("Starting database resilience test")

	// Test invalid query handling
	_, err := dit.db.QueryContext(ctx, "SELECT * FROM non_existent_table")
	assert.Error(t, err, "Invalid query should return error")
	dit.logger.Info("Invalid query error handling verified")

	// Test constraint violation
	_, err = dit.db.ExecContext(ctx, `
		CREATE TEMPORARY TABLE test_constraints (
			id SERIAL PRIMARY KEY,
			unique_field VARCHAR(50) UNIQUE NOT NULL
		)
	`)
	require.NoError(t, err, "Constraint table creation should succeed")

	// Insert valid record
	_, err = dit.db.ExecContext(ctx, 
		"INSERT INTO test_constraints (unique_field) VALUES ($1)", "unique_value")
	require.NoError(t, err, "First insert should succeed")

	// Try to insert duplicate
	_, err = dit.db.ExecContext(ctx, 
		"INSERT INTO test_constraints (unique_field) VALUES ($1)", "unique_value")
	assert.Error(t, err, "Duplicate insert should fail")
	dit.logger.Info("Constraint violation handling verified")

	// Test transaction timeout (if supported)
	shortCtx, shortCancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer shortCancel()

	time.Sleep(2 * time.Millisecond) // Ensure context is expired
	_, err = dit.db.QueryContext(shortCtx, "SELECT 1")
	assert.Error(t, err, "Expired context should cause error")
	dit.logger.Info("Context timeout handling verified")

	dit.logger.Info("Database resilience test completed successfully")
}

// TestConcurrentOperations tests concurrent database operations
func (dit *DatabaseIntegrationTest) TestConcurrentOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dit.logger.Info("Starting concurrent operations test")

	// Create test table
	_, err := dit.db.ExecContext(ctx, `
		CREATE TEMPORARY TABLE test_concurrent (
			id SERIAL PRIMARY KEY,
			counter INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err, "Test table creation should succeed")

	// Insert initial record
	_, err = dit.db.ExecContext(ctx, "INSERT INTO test_concurrent (counter) VALUES (0)")
	require.NoError(t, err, "Initial insert should succeed")

	// Test concurrent updates
	concurrency := 50
	iterations := 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				// Simulate concurrent counter increment
				tx, err := dit.db.BeginTx(ctx, nil)
				if err != nil {
					errors <- fmt.Errorf("worker %d: begin tx failed: %w", workerID, err)
					continue
				}

				var currentCounter int
				err = tx.QueryRowContext(ctx, 
					"SELECT counter FROM test_concurrent WHERE id = 1 FOR UPDATE").Scan(&currentCounter)
				if err != nil {
					tx.Rollback()
					errors <- fmt.Errorf("worker %d: select failed: %w", workerID, err)
					continue
				}

				// Simulate some processing time
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)

				_, err = tx.ExecContext(ctx, 
					"UPDATE test_concurrent SET counter = $1, updated_at = NOW() WHERE id = 1", 
					currentCounter+1)
				if err != nil {
					tx.Rollback()
					errors <- fmt.Errorf("worker %d: update failed: %w", workerID, err)
					continue
				}

				err = tx.Commit()
				if err != nil {
					errors <- fmt.Errorf("worker %d: commit failed: %w", workerID, err)
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Logf("Concurrent operation error: %v", err)
		errorCount++
	}

	// Verify final counter value
	var finalCounter int
	err = dit.db.QueryRowContext(ctx, "SELECT counter FROM test_concurrent WHERE id = 1").Scan(&finalCounter)
	require.NoError(t, err, "Final counter query should succeed")

	expectedCounter := concurrency * iterations
	dit.logger.Info("Concurrent operations test results",
		"duration", duration,
		"expected_counter", expectedCounter,
		"actual_counter", finalCounter,
		"error_count", errorCount,
		"operations_per_second", float64(expectedCounter)/duration.Seconds(),
	)

	// Allow for some errors due to concurrency, but final result should be correct
	assert.Equal(t, expectedCounter, finalCounter, "Final counter should match expected value")
	assert.Less(t, errorCount, expectedCounter/10, "Error rate should be less than 10%")

	dit.logger.Info("Concurrent operations test completed successfully")
}

// TestDatabaseMigrations tests database migration functionality
func (dit *DatabaseIntegrationTest) TestDatabaseMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dit.logger.Info("Starting database migrations test")

	// Create migration tracking table
	_, err := dit.db.ExecContext(ctx, `
		CREATE TEMPORARY TABLE test_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err, "Migration table creation should succeed")

	// Simulate migration steps
	migrations := []struct {
		version int
		sql     string
	}{
		{1, "CREATE TEMPORARY TABLE test_users (id SERIAL PRIMARY KEY, name VARCHAR(100))"},
		{2, "ALTER TABLE test_users ADD COLUMN email VARCHAR(255)"},
		{3, "CREATE INDEX idx_test_users_email ON test_users(email)"},
		{4, "ALTER TABLE test_users ADD COLUMN created_at TIMESTAMP DEFAULT NOW()"},
	}

	for _, migration := range migrations {
		start := time.Now()

		// Check if migration already applied
		var count int
		err = dit.db.QueryRowContext(ctx, 
			"SELECT COUNT(*) FROM test_migrations WHERE version = $1", migration.version).Scan(&count)
		require.NoError(t, err, "Migration check should succeed")

		if count > 0 {
			dit.logger.Info("Migration already applied", "version", migration.version)
			continue
		}

		// Apply migration in transaction
		tx, err := dit.db.BeginTx(ctx, nil)
		require.NoError(t, err, "Migration transaction should begin")

		_, err = tx.ExecContext(ctx, migration.sql)
		if err != nil {
			tx.Rollback()
			require.NoError(t, err, "Migration should succeed")
		}

		_, err = tx.ExecContext(ctx, 
			"INSERT INTO test_migrations (version) VALUES ($1)", migration.version)
		require.NoError(t, err, "Migration tracking should succeed")

		err = tx.Commit()
		require.NoError(t, err, "Migration commit should succeed")

		duration := time.Since(start)
		dit.logger.Info("Migration applied successfully",
			"version", migration.version,
			"duration", duration,
		)
	}

	// Verify all migrations were applied
	var appliedCount int
	err = dit.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_migrations").Scan(&appliedCount)
	require.NoError(t, err, "Applied migrations count should succeed")
	assert.Equal(t, len(migrations), appliedCount, "All migrations should be applied")

	dit.logger.Info("Database migrations test completed successfully")
}

// RunAllTests runs all database integration tests
func (dit *DatabaseIntegrationTest) RunAllTests(t *testing.T) {
	dit.logger.Info("Starting comprehensive database integration test suite")

	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"Connectivity", dit.TestDatabaseConnectivity},
		{"ConnectionPooling", dit.TestConnectionPooling},
		{"TransactionPerformance", dit.TestTransactionPerformance},
		{"QueryPerformance", dit.TestQueryPerformance},
		{"DatabaseResilience", dit.TestDatabaseResilience},
		{"ConcurrentOperations", dit.TestConcurrentOperations},
		{"DatabaseMigrations", dit.TestDatabaseMigrations},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := time.Now()
			test.test(t)
			duration := time.Since(start)
			dit.logger.Info("Test completed",
				"test_name", test.name,
				"duration", duration,
			)
		})
	}

	dit.logger.Info("Database integration test suite completed successfully")
}