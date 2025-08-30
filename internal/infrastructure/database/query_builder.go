package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/agentscan/agentscan/pkg/errors"
)

// SecureQueryBuilder provides secure query building utilities
type SecureQueryBuilder struct {
	db sqlx.Ext // Can be *sqlx.DB or *sqlx.Tx
}

// NewSecureQueryBuilder creates a new secure query builder
func NewSecureQueryBuilder(db sqlx.Ext) *SecureQueryBuilder {
	return &SecureQueryBuilder{
		db: db,
	}
}

// WhereClause represents a WHERE clause with parameters
type WhereClause struct {
	Condition string
	Args      []interface{}
}

// CreateWhereClause creates a new WHERE clause
func CreateWhereClause(field, operator string, value interface{}) WhereClause {
	switch operator {
	case "IN":
		// Handle IN clauses with array values
		if values, ok := value.([]interface{}); ok {
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
			}
			condition := fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", "))
			return WhereClause{
				Condition: condition,
				Args:      values,
			}
		}
		// Single value IN clause
		return WhereClause{
			Condition: fmt.Sprintf("%s IN ($1)", field),
			Args:      []interface{}{value},
		}
	case "LIKE":
		return WhereClause{
			Condition: fmt.Sprintf("%s LIKE $1", field),
			Args:      []interface{}{value},
		}
	case ">=":
		return WhereClause{
			Condition: fmt.Sprintf("%s >= $1", field),
			Args:      []interface{}{value},
		}
	case "<=":
		return WhereClause{
			Condition: fmt.Sprintf("%s <= $1", field),
			Args:      []interface{}{value},
		}
	case ">":
		return WhereClause{
			Condition: fmt.Sprintf("%s > $1", field),
			Args:      []interface{}{value},
		}
	case "<":
		return WhereClause{
			Condition: fmt.Sprintf("%s < $1", field),
			Args:      []interface{}{value},
		}
	case "!=", "<>":
		return WhereClause{
			Condition: fmt.Sprintf("%s != $1", field),
			Args:      []interface{}{value},
		}
	default: // "="
		return WhereClause{
			Condition: fmt.Sprintf("%s = $1", field),
			Args:      []interface{}{value},
		}
	}
}

// QueryFilter represents query filtering options
type QueryFilter struct {
	WhereClauses []WhereClause
	OrderBy      string
	Limit        int
	Offset       int
}

// ExecuteSecureQuery executes a query with proper parameter binding
func (qb *SecureQueryBuilder) ExecuteSecureQuery(ctx context.Context, query string, args []interface{}, dest interface{}) error {
	// Validate the query for potential SQL injection
	if err := qb.validateQuery(query); err != nil {
		return err
	}

	// Execute the query based on the destination type
	switch v := dest.(type) {
	case *[]interface{}:
		// For slice results, use Select
		return qb.db.SelectContext(ctx, dest, query, args...)
	default:
		// For single results, use Get
		return qb.db.GetContext(ctx, dest, query, args...)
	}
}

// validateQuery performs basic SQL injection validation
func (qb *SecureQueryBuilder) validateQuery(query string) error {
	// Convert to lowercase for checking
	lowerQuery := strings.ToLower(query)

	// Check for dangerous patterns
	dangerousPatterns := []string{
		"drop table",
		"drop database",
		"truncate",
		"delete from",
		"insert into",
		"update ",
		"alter table",
		"create table",
		"grant ",
		"revoke ",
		"exec ",
		"execute ",
		"sp_",
		"xp_",
		"--",
		"/*",
		"*/",
		";",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerQuery, pattern) {
			// Allow certain patterns in specific contexts
			if pattern == "delete from" && strings.HasPrefix(strings.TrimSpace(lowerQuery), "delete from") {
				continue // Allow DELETE queries
			}
			if pattern == "insert into" && strings.HasPrefix(strings.TrimSpace(lowerQuery), "insert into") {
				continue // Allow INSERT queries
			}
			if pattern == "update " && strings.HasPrefix(strings.TrimSpace(lowerQuery), "update ") {
				continue // Allow UPDATE queries
			}
			if pattern == ";" && strings.Count(lowerQuery, ";") == 1 && strings.HasSuffix(strings.TrimSpace(lowerQuery), ";") {
				continue // Allow single trailing semicolon
			}

			return errors.NewValidationError("potentially dangerous SQL pattern detected").WithDetails(map[string]interface{}{
				"pattern": pattern,
				"query":   query,
			})
		}
	}

	return nil
}

// BuildSelectQuery builds a secure SELECT query with filters
func (qb *SecureQueryBuilder) BuildSelectQuery(tableName string, filter *QueryFilter) (string, []interface{}) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", tableName)
	
	var args []interface{}
	var conditions []string
	argIndex := 1
	
	// Add WHERE clauses
	for _, whereClause := range filter.WhereClauses {
		condition := whereClause.Condition
		for i, arg := range whereClause.Args {
			placeholder := fmt.Sprintf("$%d", i+1)
			actualPlaceholder := fmt.Sprintf("$%d", argIndex)
			condition = strings.Replace(condition, placeholder, actualPlaceholder, 1)
			args = append(args, arg)
			argIndex++
		}
		conditions = append(conditions, condition)
	}
	
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Add ORDER BY
	if filter.OrderBy != "" {
		baseQuery += " ORDER BY " + filter.OrderBy
	}
	
	// Add LIMIT and OFFSET
	if filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}
	
	if filter.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}
	
	return baseQuery, args
}

// BuildCountQuery builds a secure COUNT query with filters
func (qb *SecureQueryBuilder) BuildCountQuery(tableName string, whereClauses []WhereClause) (string, []interface{}) {
	baseQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	
	var args []interface{}
	var conditions []string
	argIndex := 1
	
	for _, whereClause := range whereClauses {
		condition := whereClause.Condition
		for i, arg := range whereClause.Args {
			placeholder := fmt.Sprintf("$%d", i+1)
			actualPlaceholder := fmt.Sprintf("$%d", argIndex)
			condition = strings.Replace(condition, placeholder, actualPlaceholder, 1)
			args = append(args, arg)
			argIndex++
		}
		conditions = append(conditions, condition)
	}
	
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	return baseQuery, args
}

// BuildInsertQuery builds a secure INSERT query
func (qb *SecureQueryBuilder) BuildInsertQuery(tableName string, fields []string, values []interface{}) (string, []interface{}) {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)
	
	return query, values
}

// BuildUpdateQuery builds a secure UPDATE query
func (qb *SecureQueryBuilder) BuildUpdateQuery(tableName string, updates map[string]interface{}, whereClause WhereClause) (string, []interface{}) {
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+len(whereClause.Args))
	argIndex := 1
	
	// Build SET clause
	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}
	
	// Build WHERE clause
	condition := whereClause.Condition
	for i, arg := range whereClause.Args {
		placeholder := fmt.Sprintf("$%d", i+1)
		actualPlaceholder := fmt.Sprintf("$%d", argIndex)
		condition = strings.Replace(condition, placeholder, actualPlaceholder, 1)
		args = append(args, arg)
		argIndex++
	}
	
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setParts, ", "),
		condition,
	)
	
	return query, args
}

// BuildDeleteQuery builds a secure DELETE query
func (qb *SecureQueryBuilder) BuildDeleteQuery(tableName string, whereClause WhereClause) (string, []interface{}) {
	condition := whereClause.Condition
	args := make([]interface{}, len(whereClause.Args))
	copy(args, whereClause.Args)
	
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, condition)
	
	return query, args
}