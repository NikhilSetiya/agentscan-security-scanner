package database

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/pkg/config"
)

func setupTestDB(t *testing.T) *DB {
	cfg := &config.DatabaseConfig{
		Host:               "localhost",
		Port:               5432,
		Name:               "agentscan_test",
		User:               "agentscan",
		Password:           "agentscan_password",
		SSLMode:            "disable",
		MaxOpenConns:       25,
		MaxIdleConns:       5,
		ConnMaxLifetime:    5 * time.Minute,
		ConnMaxIdleTime:    10 * time.Minute,
		QueryTimeout:       30 * time.Second,
		StatementTimeout:   30 * time.Second,
		EnablePreparedStmt: true,
	}

	db, err := New(cfg)
	require.NoError(t, err)

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_performance (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			value INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	return db
}

func TestDB_PreparedStatements(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test prepared statement creation
	stmt, err := db.PrepareStatement(ctx, "insert_test", "INSERT INTO test_performance (name, value) VALUES ($1, $2)")
	assert.NoError(t, err)
	assert.NotNil(t, stmt)

	// Test cached statement retrieval
	cachedStmt, exists := db.GetCachedStatement("insert_test")
	assert.True(t, exists)
	assert.Equal(t, stmt, cachedStmt)

	// Test statement execution
	_, err = stmt.ExecContext(ctx, "test", 42)
	assert.NoError(t, err)

	// Test statement cache clearing
	err = db.ClearStatementCache()
	assert.NoError(t, err)

	_, exists = db.GetCachedStatement("insert_test")
	assert.False(t, exists)
}

func TestDB_QueryWithTimeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test successful query with timeout
	rows, err := db.QueryWithTimeout(ctx, 5*time.Second, "SELECT 1 as test")
	assert.NoError(t, err)
	assert.NotNil(t, rows)
	rows.Close()

	// Test query timeout (this would need a slow query to properly test)
	// For now, just test that the method works with a fast query
	rows, err = db.QueryWithTimeout(ctx, 1*time.Millisecond, "SELECT 1")
	// This might or might not timeout depending on system speed
	if err == nil {
		rows.Close()
	}
}

func TestDB_ExecWithTimeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test successful execution with timeout
	result, err := db.ExecWithTimeout(ctx, 5*time.Second, "INSERT INTO test_performance (name, value) VALUES ($1, $2)", "timeout_test", 123)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	rowsAffected, err := result.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
}

func TestDB_BatchInsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Prepare test data
	columns := []string{"name", "value"}
	values := [][]interface{}{
		{"batch1", 1},
		{"batch2", 2},
		{"batch3", 3},
		{"batch4", 4},
		{"batch5", 5},
	}

	// Test batch insert
	err := db.BatchInsert(ctx, "test_performance", columns, values, 2)
	assert.NoError(t, err)

	// Verify data was inserted
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM test_performance WHERE name LIKE 'batch%'")
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestDB_ConnectionPooling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Test connection pool stats
	stats := db.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1)
	assert.GreaterOrEqual(t, stats.OpenConnections, 0)
}

func TestDB_TransactionPerformance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test transaction performance
	start := time.Now()
	err := db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		for i := 0; i < 100; i++ {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_performance (name, value) VALUES ($1, $2)", fmt.Sprintf("tx_test_%d", i), i)
			if err != nil {
				return err
			}
		}
		return nil
	})
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, 5*time.Second) // Should complete within 5 seconds

	// Verify all records were inserted
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM test_performance WHERE name LIKE 'tx_test_%'")
	assert.NoError(t, err)
	assert.Equal(t, 100, count)
}

func BenchmarkDB_SimpleQuery(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer db.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := db.QueryxContext(ctx, "SELECT 1")
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkDB_PreparedStatement(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer db.Close()

	ctx := context.Background()

	// Prepare statement once
	stmt, err := db.PrepareStatement(ctx, "bench_select", "SELECT $1 as value")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := stmt.QueryxContext(ctx, i)
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

func BenchmarkDB_BatchInsert(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer db.Close()

	ctx := context.Background()
	columns := []string{"name", "value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		values := [][]interface{}{
			{fmt.Sprintf("bench_%d", i), i},
		}
		err := db.BatchInsert(ctx, "test_performance", columns, values, 1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDB_ConcurrentQueries(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer db.Close()

	ctx := context.Background()
	concurrency := 10

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg sync.WaitGroup
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					rows, err := db.QueryxContext(ctx, "SELECT 1")
					if err != nil {
						b.Error(err)
						return
					}
					rows.Close()
				}()
			}
			wg.Wait()
		}
	})
}

func BenchmarkDB_Transaction(b *testing.B) {
	db := setupTestDB(&testing.T{})
	defer db.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_performance (name, value) VALUES ($1, $2)", fmt.Sprintf("bench_tx_%d", i), i)
			return err
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Cleanup function to run after tests
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup test database
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "agentscan_test",
		User:     "agentscan",
		Password: "agentscan_password",
		SSLMode:  "disable",
	}

	if db, err := New(cfg); err == nil {
		db.Exec("DROP TABLE IF EXISTS test_performance")
		db.Close()
	}

	// Exit with test result code
	panic(code)
}