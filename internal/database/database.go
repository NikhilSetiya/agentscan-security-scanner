package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
)

// DB wraps the database connection with additional functionality
type DB struct {
	*sqlx.DB
	config *config.DatabaseConfig
	stmtCache map[string]*sqlx.Stmt
	stmtMutex sync.RWMutex
}

// New creates a new database connection with optimized settings
func New(cfg *config.DatabaseConfig) (*DB, error) {
	if cfg == nil {
		return nil, errors.NewValidationError("database configuration is required")
	}

	// Build basic connection string compatible with PgBouncer
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	// Open database connection
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, errors.NewInternalError("failed to connect to database").WithCause(err)
	}

	// Configure optimized connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	
	// Set connection max idle time to prevent stale connections
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.NewInternalError("failed to ping database").WithCause(err)
	}

	return &DB{
		DB:        db,
		config:    cfg,
		stmtCache: make(map[string]*sqlx.Stmt),
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

// PrepareStatement prepares and caches a SQL statement
func (db *DB) PrepareStatement(ctx context.Context, name, query string) (*sqlx.Stmt, error) {
	db.stmtMutex.RLock()
	if stmt, exists := db.stmtCache[name]; exists {
		db.stmtMutex.RUnlock()
		return stmt, nil
	}
	db.stmtMutex.RUnlock()

	db.stmtMutex.Lock()
	defer db.stmtMutex.Unlock()

	// Double-check after acquiring write lock
	if stmt, exists := db.stmtCache[name]; exists {
		return stmt, nil
	}

	stmt, err := db.PreparexContext(ctx, query)
	if err != nil {
		return nil, errors.NewInternalError("failed to prepare statement").WithCause(err)
	}

	db.stmtCache[name] = stmt
	return stmt, nil
}

// GetCachedStatement retrieves a cached prepared statement
func (db *DB) GetCachedStatement(name string) (*sqlx.Stmt, bool) {
	db.stmtMutex.RLock()
	defer db.stmtMutex.RUnlock()
	stmt, exists := db.stmtCache[name]
	return stmt, exists
}

// ClearStatementCache clears all cached prepared statements
func (db *DB) ClearStatementCache() error {
	db.stmtMutex.Lock()
	defer db.stmtMutex.Unlock()

	var errs []error
	for name, stmt := range db.stmtCache {
		if err := stmt.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close statement %s: %w", name, err))
		}
	}

	db.stmtCache = make(map[string]*sqlx.Stmt)

	if len(errs) > 0 {
		return errors.NewInternalError("failed to clear statement cache").WithCause(fmt.Errorf("%v", errs))
	}

	return nil
}

// QueryWithTimeout executes a query with a timeout
func (db *DB) QueryWithTimeout(ctx context.Context, timeout time.Duration, query string, args ...interface{}) (*sqlx.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rows, err := db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewInternalError("query execution failed").WithCause(err)
	}

	return rows, nil
}

// ExecWithTimeout executes a statement with a timeout
func (db *DB) ExecWithTimeout(ctx context.Context, timeout time.Duration, query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewInternalError("statement execution failed").WithCause(err)
	}

	return result, nil
}

// BatchInsert performs optimized batch insert operations
func (db *DB) BatchInsert(ctx context.Context, table string, columns []string, values [][]interface{}, batchSize int) error {
	if len(values) == 0 {
		return nil
	}

	if batchSize <= 0 {
		batchSize = 1000 // Default batch size
	}

	// Process in batches
	for i := 0; i < len(values); i += batchSize {
		end := i + batchSize
		if end > len(values) {
			end = len(values)
		}

		batch := values[i:end]
		if err := db.executeBatch(ctx, table, columns, batch); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) executeBatch(ctx context.Context, table string, columns []string, batch [][]interface{}) error {
	if len(batch) == 0 {
		return nil
	}

	// Build column list
	columnList := ""
	for i, col := range columns {
		if i > 0 {
			columnList += ", "
		}
		columnList += col
	}

	// Build multi-row insert query
	valueStrings := make([]string, len(batch))
	args := make([]interface{}, 0, len(batch)*len(columns))
	
	for i, row := range batch {
		placeholders := make([]string, len(columns))
		for j := range columns {
			placeholders[j] = fmt.Sprintf("$%d", len(args)+j+1)
		}
		
		valueString := "("
		for j, placeholder := range placeholders {
			if j > 0 {
				valueString += ", "
			}
			valueString += placeholder
		}
		valueString += ")"
		
		valueStrings[i] = valueString
		args = append(args, row...)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", 
		table, 
		columnList,
		valueStrings[0])
	
	for i := 1; i < len(valueStrings); i++ {
		query += ", " + valueStrings[i]
	}

	_, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.NewInternalError("batch insert failed").WithCause(err)
	}

	return nil
}