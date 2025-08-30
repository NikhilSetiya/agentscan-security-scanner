package database

import (
	"fmt"
	"strings"
)

// QueryBuilder provides secure SQL query building with parameterized queries
type QueryBuilder struct {
	baseQuery string
	conditions []string
	args map[string]interface{}
	orderBy string
	limit int
	offset int
}

// NewQueryBuilder creates a new secure query builder
func NewQueryBuilder(baseQuery string) *QueryBuilder {
	return &QueryBuilder{
		baseQuery: baseQuery,
		conditions: []string{},
		args: make(map[string]interface{}),
	}
}

// Where adds a WHERE condition with parameterized values
func (qb *QueryBuilder) Where(condition string, key string, value interface{}) *QueryBuilder {
	if value != nil {
		// Only add condition if value is not nil/empty
		switch v := value.(type) {
		case string:
			if v != "" {
				qb.conditions = append(qb.conditions, condition)
				qb.args[key] = value
			}
		case *string:
			if v != nil && *v != "" {
				qb.conditions = append(qb.conditions, condition)
				qb.args[key] = *v
			}
		default:
			qb.conditions = append(qb.conditions, condition)
			qb.args[key] = value
		}
	}
	return qb
}

// WhereUUID adds a WHERE condition for UUID values
func (qb *QueryBuilder) WhereUUID(condition string, key string, value interface{}) *QueryBuilder {
	if value != nil {
		qb.conditions = append(qb.conditions, condition)
		qb.args[key] = value
	}
	return qb
}

// WhereLike adds a WHERE LIKE condition with proper escaping
func (qb *QueryBuilder) WhereLike(condition string, key string, value string) *QueryBuilder {
	if value != "" {
		// Escape special characters in LIKE patterns
		escapedValue := strings.ReplaceAll(value, "%", "\\%")
		escapedValue = strings.ReplaceAll(escapedValue, "_", "\\_")
		qb.conditions = append(qb.conditions, condition)
		qb.args[key] = "%" + escapedValue + "%"
	}
	return qb
}

// OrderBy sets the ORDER BY clause
func (qb *QueryBuilder) OrderBy(orderBy string) *QueryBuilder {
	// Validate ORDER BY to prevent injection
	if isValidOrderBy(orderBy) {
		qb.orderBy = orderBy
	}
	return qb
}

// Limit sets the LIMIT clause
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	if limit > 0 {
		qb.limit = limit
	}
	return qb
}

// Offset sets the OFFSET clause
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	if offset >= 0 {
		qb.offset = offset
	}
	return qb
}

// Build constructs the final SQL query with all conditions
func (qb *QueryBuilder) Build() (string, map[string]interface{}) {
	query := qb.baseQuery
	
	// Add WHERE conditions
	if len(qb.conditions) > 0 {
		query += " WHERE " + strings.Join(qb.conditions, " AND ")
	}
	
	// Add ORDER BY
	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}
	
	// Add LIMIT and OFFSET
	if qb.limit > 0 {
		query += " LIMIT :limit"
		qb.args["limit"] = qb.limit
	}
	
	if qb.offset > 0 {
		query += " OFFSET :offset"
		qb.args["offset"] = qb.offset
	}
	
	return query, qb.args
}

// BuildCount constructs a COUNT query
func (qb *QueryBuilder) BuildCount() (string, map[string]interface{}) {
	// Extract table name from base query for count
	countQuery := convertToCountQuery(qb.baseQuery)
	
	// Add WHERE conditions
	if len(qb.conditions) > 0 {
		countQuery += " WHERE " + strings.Join(qb.conditions, " AND ")
	}
	
	return countQuery, qb.args
}

// convertToCountQuery converts a SELECT query to a COUNT query
func convertToCountQuery(baseQuery string) string {
	// Simple conversion - in production you might want more sophisticated parsing
	if strings.Contains(strings.ToUpper(baseQuery), "SELECT") && strings.Contains(strings.ToUpper(baseQuery), "FROM") {
		// Find the FROM clause and build count query
		upperQuery := strings.ToUpper(baseQuery)
		fromIndex := strings.Index(upperQuery, "FROM")
		if fromIndex != -1 {
			fromClause := baseQuery[fromIndex:]
			return "SELECT COUNT(*) " + fromClause
		}
	}
	
	// Fallback - just replace SELECT part
	return strings.Replace(baseQuery, "SELECT *", "SELECT COUNT(*)", 1)
}

// isValidOrderBy validates ORDER BY clauses to prevent injection
func isValidOrderBy(orderBy string) bool {
	// Allow only safe ORDER BY patterns
	allowedPatterns := []string{
		"created_at DESC",
		"created_at ASC",
		"updated_at DESC",
		"updated_at ASC",
		"name ASC",
		"name DESC",
		"severity DESC",
		"severity ASC",
		"id ASC",
		"id DESC",
	}
	
	for _, pattern := range allowedPatterns {
		if orderBy == pattern {
			return true
		}
	}
	
	return false
}