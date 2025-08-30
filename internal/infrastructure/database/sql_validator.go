package database

import (
	"regexp"
	"strings"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
)

// SQLValidator provides SQL injection detection and prevention
type SQLValidator struct {
	dangerousPatterns []*regexp.Regexp
}

// NewSQLValidator creates a new SQL validator
func NewSQLValidator() *SQLValidator {
	// Compile dangerous SQL patterns
	patterns := []string{
		// SQL injection patterns
		`(?i)(union\s+select)`,
		`(?i)(drop\s+table)`,
		`(?i)(delete\s+from)`,
		`(?i)(insert\s+into)`,
		`(?i)(update\s+\w+\s+set)`,
		`(?i)(exec\s*\()`,
		`(?i)(execute\s*\()`,
		`(?i)(sp_\w+)`,
		`(?i)(xp_\w+)`,
		`(?i)(\bor\s+1\s*=\s*1)`,
		`(?i)(\band\s+1\s*=\s*1)`,
		`(?i)(--\s*$)`,
		`(?i)(/\*.*\*/)`,
		`[\x00\x1a]`, // Null bytes and substitute characters
		// Additional dangerous patterns
		`(?i)(alter\s+table)`,
		`(?i)(create\s+table)`,
		`(?i)(truncate\s+table)`,
		`(?i)(grant\s+)`,
		`(?i)(revoke\s+)`,
		`(?i)(shutdown)`,
		`(?i)(waitfor\s+delay)`,
		`(?i)(benchmark\s*\()`,
		`(?i)(sleep\s*\()`,
		`(?i)(pg_sleep\s*\()`,
		`(?i)(information_schema)`,
		`(?i)(sys\.)`,
		`(?i)(master\.)`,
		`(?i)(msdb\.)`,
		`(?i)(tempdb\.)`,
	}

	var compiledPatterns []*regexp.Regexp
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			compiledPatterns = append(compiledPatterns, compiled)
		}
	}

	return &SQLValidator{
		dangerousPatterns: compiledPatterns,
	}
}

// ValidateInput checks if input contains SQL injection patterns
func (v *SQLValidator) ValidateInput(input string) error {
	if input == "" {
		return nil
	}

	// Check against dangerous patterns
	for _, pattern := range v.dangerousPatterns {
		if pattern.MatchString(input) {
			return errors.NewSecurityError("potential SQL injection detected").
				WithDetail("pattern", pattern.String()).
				WithDetail("input", input)
		}
	}

	return nil
}

// ValidateQuery validates a complete SQL query for safety
func (v *SQLValidator) ValidateQuery(query string) error {
	if query == "" {
		return errors.NewValidationError("empty query")
	}

	// Normalize query for analysis
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	// Check if query starts with allowed operations
	allowedOperations := []string{"select", "insert", "update", "delete"}
	hasAllowedStart := false
	for _, op := range allowedOperations {
		if strings.HasPrefix(normalizedQuery, op) {
			hasAllowedStart = true
			break
		}
	}

	if !hasAllowedStart {
		return errors.NewSecurityError("query must start with allowed operation").
			WithDetail("allowed_operations", strings.Join(allowedOperations, ", "))
	}

	// Check for dangerous patterns in the query
	return v.ValidateInput(query)
}

// SanitizeOrderBy validates and sanitizes ORDER BY clauses
func (v *SQLValidator) SanitizeOrderBy(orderBy string) (string, error) {
	if orderBy == "" {
		return "", nil
	}

	// Remove extra whitespace
	orderBy = strings.TrimSpace(orderBy)

	// Validate ORDER BY format
	orderByPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\s+(ASC|DESC))?$`)
	if !orderByPattern.MatchString(orderBy) {
		return "", errors.NewValidationError("invalid ORDER BY format")
	}

	// Check against injection patterns
	if err := v.ValidateInput(orderBy); err != nil {
		return "", err
	}

	return orderBy, nil
}

// SanitizeTableName validates and sanitizes table names
func (v *SQLValidator) SanitizeTableName(tableName string) (string, error) {
	if tableName == "" {
		return "", errors.NewValidationError("empty table name")
	}

	// Remove extra whitespace
	tableName = strings.TrimSpace(tableName)

	// Validate table name format (alphanumeric and underscores only)
	tableNamePattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !tableNamePattern.MatchString(tableName) {
		return "", errors.NewValidationError("invalid table name format")
	}

	// Check against injection patterns
	if err := v.ValidateInput(tableName); err != nil {
		return "", err
	}

	return tableName, nil
}

// SanitizeColumnName validates and sanitizes column names
func (v *SQLValidator) SanitizeColumnName(columnName string) (string, error) {
	if columnName == "" {
		return "", errors.NewValidationError("empty column name")
	}

	// Remove extra whitespace
	columnName = strings.TrimSpace(columnName)

	// Validate column name format (alphanumeric and underscores only)
	columnNamePattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !columnNamePattern.MatchString(columnName) {
		return "", errors.NewValidationError("invalid column name format")
	}

	// Check against injection patterns
	if err := v.ValidateInput(columnName); err != nil {
		return "", err
	}

	return columnName, nil
}

// ValidateLimit validates LIMIT values
func (v *SQLValidator) ValidateLimit(limit int) error {
	if limit < 0 {
		return errors.NewValidationError("limit cannot be negative")
	}
	if limit > 10000 {
		return errors.NewValidationError("limit too large (max 10000)")
	}
	return nil
}

// ValidateOffset validates OFFSET values
func (v *SQLValidator) ValidateOffset(offset int) error {
	if offset < 0 {
		return errors.NewValidationError("offset cannot be negative")
	}
	return nil
}

// Global validator instance
var globalSQLValidator = NewSQLValidator()

// ValidateSQLInput is a convenience function for validating SQL input
func ValidateSQLInput(input string) error {
	return globalSQLValidator.ValidateInput(input)
}

// ValidateSQLQuery is a convenience function for validating SQL queries
func ValidateSQLQuery(query string) error {
	return globalSQLValidator.ValidateQuery(query)
}