package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentscan/agentscan/pkg/errors"
)

// PaginationStrategy defines different pagination approaches
type PaginationStrategy string

const (
	// OffsetPagination uses LIMIT/OFFSET (good for small datasets)
	OffsetPagination PaginationStrategy = "offset"
	// CursorPagination uses cursor-based pagination (good for large datasets)
	CursorPagination PaginationStrategy = "cursor"
	// KeysetPagination uses keyset pagination (best for very large datasets)
	KeysetPagination PaginationStrategy = "keyset"
)

// PaginationConfig configures pagination behavior
type PaginationConfig struct {
	Strategy    PaginationStrategy `json:"strategy"`
	PageSize    int               `json:"page_size"`
	MaxPageSize int               `json:"max_page_size"`
	DefaultSize int               `json:"default_size"`
}

// DefaultPaginationConfig returns sensible defaults
func DefaultPaginationConfig() *PaginationConfig {
	return &PaginationConfig{
		Strategy:    OffsetPagination,
		PageSize:    20,
		MaxPageSize: 100,
		DefaultSize: 20,
	}
}

// PaginationRequest represents a pagination request
type PaginationRequest struct {
	Strategy   PaginationStrategy `json:"strategy,omitempty"`
	Page       int               `json:"page,omitempty"`        // For offset pagination
	PageSize   int               `json:"page_size,omitempty"`
	Cursor     string            `json:"cursor,omitempty"`      // For cursor pagination
	After      interface{}       `json:"after,omitempty"`      // For keyset pagination
	Before     interface{}       `json:"before,omitempty"`     // For keyset pagination
	SortField  string            `json:"sort_field,omitempty"` // Field to sort by
	SortOrder  string            `json:"sort_order,omitempty"` // ASC or DESC
}

// PaginationResponse represents a pagination response
type PaginationResponse struct {
	Page         int         `json:"page,omitempty"`
	PageSize     int         `json:"page_size"`
	Total        int         `json:"total,omitempty"`        // Only for offset pagination
	TotalPages   int         `json:"total_pages,omitempty"`  // Only for offset pagination
	HasNext      bool        `json:"has_next"`
	HasPrev      bool        `json:"has_prev"`
	NextCursor   string      `json:"next_cursor,omitempty"`   // For cursor pagination
	PrevCursor   string      `json:"prev_cursor,omitempty"`   // For cursor pagination
	NextAfter    interface{} `json:"next_after,omitempty"`    // For keyset pagination
	PrevBefore   interface{} `json:"prev_before,omitempty"`   // For keyset pagination
}

// PaginatedResult represents a paginated result set
type PaginatedResult[T any] struct {
	Data       []T                 `json:"data"`
	Pagination *PaginationResponse `json:"pagination"`
}

// PaginationOptimizer provides optimized pagination implementations
type PaginationOptimizer struct {
	db     *sqlx.DB
	config *PaginationConfig
}

// NewPaginationOptimizer creates a new pagination optimizer
func NewPaginationOptimizer(db *sqlx.DB, config *PaginationConfig) *PaginationOptimizer {
	if config == nil {
		config = DefaultPaginationConfig()
	}
	return &PaginationOptimizer{
		db:     db,
		config: config,
	}
}

// ValidateRequest validates and normalizes a pagination request
func (po *PaginationOptimizer) ValidateRequest(req *PaginationRequest) error {
	// Set defaults
	if req.Strategy == "" {
		req.Strategy = po.config.Strategy
	}
	if req.PageSize <= 0 {
		req.PageSize = po.config.DefaultSize
	}
	if req.PageSize > po.config.MaxPageSize {
		req.PageSize = po.config.MaxPageSize
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.SortOrder == "" {
		req.SortOrder = "DESC"
	}

	// Validate sort order
	if req.SortOrder != "ASC" && req.SortOrder != "DESC" {
		return errors.NewValidationError("invalid sort order, must be ASC or DESC")
	}

	return nil
}

// PaginateQuery applies pagination to a query based on the strategy
func (po *PaginationOptimizer) PaginateQuery(ctx context.Context, baseQuery string, req *PaginationRequest, args []interface{}) (*PaginatedResult[map[string]interface{}], error) {
	if err := po.ValidateRequest(req); err != nil {
		return nil, err
	}

	switch req.Strategy {
	case OffsetPagination:
		return po.paginateWithOffset(ctx, baseQuery, req, args)
	case CursorPagination:
		return po.paginateWithCursor(ctx, baseQuery, req, args)
	case KeysetPagination:
		return po.paginateWithKeyset(ctx, baseQuery, req, args)
	default:
		return nil, errors.NewValidationError("unsupported pagination strategy")
	}
}

// paginateWithOffset implements traditional LIMIT/OFFSET pagination
func (po *PaginationOptimizer) paginateWithOffset(ctx context.Context, baseQuery string, req *PaginationRequest, args []interface{}) (*PaginatedResult[map[string]interface{}], error) {
	// Calculate offset
	offset := (req.Page - 1) * req.PageSize

	// Add ORDER BY if not present
	if !strings.Contains(strings.ToUpper(baseQuery), "ORDER BY") && req.SortField != "" {
		baseQuery += fmt.Sprintf(" ORDER BY %s %s", req.SortField, req.SortOrder)
	}

	// Add LIMIT and OFFSET
	paginatedQuery := fmt.Sprintf("%s LIMIT $%d OFFSET $%d", baseQuery, len(args)+1, len(args)+2)
	paginatedArgs := append(args, req.PageSize, offset)

	// Execute paginated query
	rows, err := po.db.QueryContext(ctx, paginatedQuery, paginatedArgs...)
	if err != nil {
		return nil, errors.NewDatabaseError("paginate_offset", "failed to execute paginated query").WithCause(err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.NewDatabaseError("paginate_offset", "failed to get columns").WithCause(err)
	}

	// Scan results
	var data []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, errors.NewDatabaseError("paginate_offset", "failed to scan row").WithCause(err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		data = append(data, row)
	}

	// Get total count for offset pagination
	countQuery := po.buildCountQuery(baseQuery)
	var total int
	err = po.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, errors.NewDatabaseError("paginate_offset", "failed to get total count").WithCause(err)
	}

	// Calculate pagination metadata
	totalPages := (total + req.PageSize - 1) / req.PageSize
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	pagination := &PaginationResponse{
		Page:       req.Page,
		PageSize:   req.PageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	return &PaginatedResult[map[string]interface{}]{
		Data:       data,
		Pagination: pagination,
	}, nil
}

// paginateWithCursor implements cursor-based pagination
func (po *PaginationOptimizer) paginateWithCursor(ctx context.Context, baseQuery string, req *PaginationRequest, args []interface{}) (*PaginatedResult[map[string]interface{}], error) {
	if req.SortField == "" {
		return nil, errors.NewValidationError("sort_field is required for cursor pagination")
	}

	// Parse cursor if provided
	var cursorValue interface{}
	if req.Cursor != "" {
		// Decode cursor (in production, you'd use proper encoding/decoding)
		cursorValue = req.Cursor
	}

	// Build cursor condition
	var cursorCondition string
	if cursorValue != nil {
		operator := ">"
		if req.SortOrder == "DESC" {
			operator = "<"
		}
		cursorCondition = fmt.Sprintf(" AND %s %s $%d", req.SortField, operator, len(args)+1)
		args = append(args, cursorValue)
	}

	// Add WHERE clause for cursor
	if strings.Contains(strings.ToUpper(baseQuery), "WHERE") {
		baseQuery += cursorCondition
	} else if cursorCondition != "" {
		baseQuery += " WHERE 1=1" + cursorCondition
	}

	// Add ORDER BY
	baseQuery += fmt.Sprintf(" ORDER BY %s %s", req.SortField, req.SortOrder)

	// Add LIMIT (fetch one extra to check if there's a next page)
	paginatedQuery := fmt.Sprintf("%s LIMIT $%d", baseQuery, len(args)+1)
	paginatedArgs := append(args, req.PageSize+1)

	// Execute query
	rows, err := po.db.QueryContext(ctx, paginatedQuery, paginatedArgs...)
	if err != nil {
		return nil, errors.NewDatabaseError("paginate_cursor", "failed to execute paginated query").WithCause(err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.NewDatabaseError("paginate_cursor", "failed to get columns").WithCause(err)
	}

	// Scan results
	var data []map[string]interface{}
	var nextCursor string
	rowCount := 0

	for rows.Next() {
		rowCount++
		if rowCount > req.PageSize {
			// We have a next page, but don't include this row in results
			break
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, errors.NewDatabaseError("paginate_cursor", "failed to scan row").WithCause(err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
			// Set next cursor to the sort field value of the last row
			if col == req.SortField && rowCount == req.PageSize {
				nextCursor = fmt.Sprintf("%v", values[i])
			}
		}
		data = append(data, row)
	}

	hasNext := rowCount > req.PageSize
	hasPrev := req.Cursor != ""

	pagination := &PaginationResponse{
		PageSize:   req.PageSize,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		NextCursor: nextCursor,
	}

	return &PaginatedResult[map[string]interface{}]{
		Data:       data,
		Pagination: pagination,
	}, nil
}

// paginateWithKeyset implements keyset pagination (most efficient for large datasets)
func (po *PaginationOptimizer) paginateWithKeyset(ctx context.Context, baseQuery string, req *PaginationRequest, args []interface{}) (*PaginatedResult[map[string]interface{}], error) {
	if req.SortField == "" {
		return nil, errors.NewValidationError("sort_field is required for keyset pagination")
	}

	// Build keyset condition
	var keysetCondition string
	if req.After != nil {
		operator := ">"
		if req.SortOrder == "DESC" {
			operator = "<"
		}
		keysetCondition = fmt.Sprintf(" AND %s %s $%d", req.SortField, operator, len(args)+1)
		args = append(args, req.After)
	}

	// Add WHERE clause for keyset
	if strings.Contains(strings.ToUpper(baseQuery), "WHERE") {
		baseQuery += keysetCondition
	} else if keysetCondition != "" {
		baseQuery += " WHERE 1=1" + keysetCondition
	}

	// Add ORDER BY
	baseQuery += fmt.Sprintf(" ORDER BY %s %s", req.SortField, req.SortOrder)

	// Add LIMIT (fetch one extra to check if there's a next page)
	paginatedQuery := fmt.Sprintf("%s LIMIT $%d", baseQuery, len(args)+1)
	paginatedArgs := append(args, req.PageSize+1)

	// Execute query
	rows, err := po.db.QueryContext(ctx, paginatedQuery, paginatedArgs...)
	if err != nil {
		return nil, errors.NewDatabaseError("paginate_keyset", "failed to execute paginated query").WithCause(err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.NewDatabaseError("paginate_keyset", "failed to get columns").WithCause(err)
	}

	// Scan results
	var data []map[string]interface{}
	var nextAfter interface{}
	rowCount := 0

	for rows.Next() {
		rowCount++
		if rowCount > req.PageSize {
			// We have a next page, but don't include this row in results
			break
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, errors.NewDatabaseError("paginate_keyset", "failed to scan row").WithCause(err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
			// Set next after to the sort field value of the last row
			if col == req.SortField && rowCount == req.PageSize {
				nextAfter = values[i]
			}
		}
		data = append(data, row)
	}

	hasNext := rowCount > req.PageSize
	hasPrev := req.After != nil

	pagination := &PaginationResponse{
		PageSize:  req.PageSize,
		HasNext:   hasNext,
		HasPrev:   hasPrev,
		NextAfter: nextAfter,
	}

	return &PaginatedResult[map[string]interface{}]{
		Data:       data,
		Pagination: pagination,
	}, nil
}

// buildCountQuery builds a count query from a base query
func (po *PaginationOptimizer) buildCountQuery(baseQuery string) string {
	// Remove ORDER BY clause for count query
	upperQuery := strings.ToUpper(baseQuery)
	orderByIndex := strings.Index(upperQuery, "ORDER BY")
	if orderByIndex != -1 {
		baseQuery = baseQuery[:orderByIndex]
	}

	// Remove LIMIT and OFFSET clauses
	limitIndex := strings.Index(upperQuery, "LIMIT")
	if limitIndex != -1 {
		baseQuery = baseQuery[:limitIndex]
	}

	// Wrap in count query
	return fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS count_query", baseQuery)
}

// OptimizedPaginator provides high-level pagination methods for common entities
type OptimizedPaginator struct {
	optimizer *PaginationOptimizer
}

// NewOptimizedPaginator creates a new optimized paginator
func NewOptimizedPaginator(db *sqlx.DB) *OptimizedPaginator {
	return &OptimizedPaginator{
		optimizer: NewPaginationOptimizer(db, DefaultPaginationConfig()),
	}
}

// PaginateScanJobs paginates scan jobs with optimized queries
func (op *OptimizedPaginator) PaginateScanJobs(ctx context.Context, filters map[string]interface{}, req *PaginationRequest) (*PaginatedResult[map[string]interface{}], error) {
	baseQuery := `
		SELECT 
			sj.id, sj.repository_id, sj.user_id, sj.branch, sj.commit_sha,
			sj.scan_type, sj.priority, sj.status, sj.started_at, sj.completed_at,
			sj.created_at, sj.updated_at,
			r.name as repository_name, r.url as repository_url,
			u.email as user_email, u.name as user_name
		FROM scan_jobs sj
		INNER JOIN repositories r ON sj.repository_id = r.id
		LEFT JOIN users u ON sj.user_id = u.id
		WHERE 1=1
	`

	var args []interface{}
	argIndex := 1

	// Add filters
	for field, value := range filters {
		switch field {
		case "repository_id":
			baseQuery += fmt.Sprintf(" AND sj.repository_id = $%d", argIndex)
			args = append(args, value)
			argIndex++
		case "user_id":
			baseQuery += fmt.Sprintf(" AND sj.user_id = $%d", argIndex)
			args = append(args, value)
			argIndex++
		case "status":
			baseQuery += fmt.Sprintf(" AND sj.status = $%d", argIndex)
			args = append(args, value)
			argIndex++
		case "organization_id":
			baseQuery += fmt.Sprintf(" AND r.organization_id = $%d", argIndex)
			args = append(args, value)
			argIndex++
		}
	}

	// Set default sort field for scan jobs
	if req.SortField == "" {
		req.SortField = "sj.created_at"
	}

	return op.optimizer.PaginateQuery(ctx, baseQuery, req, args)
}

// PaginateRepositories paginates repositories with optimized queries
func (op *OptimizedPaginator) PaginateRepositories(ctx context.Context, orgID uuid.UUID, req *PaginationRequest) (*PaginatedResult[map[string]interface{}], error) {
	baseQuery := `
		SELECT 
			r.id, r.organization_id, r.name, r.url, r.provider, r.provider_id,
			r.default_branch, r.last_scan_at, r.created_at, r.updated_at,
			COALESCE(scan_stats.scan_count, 0) as scan_count,
			COALESCE(scan_stats.last_scan_status, '') as last_scan_status
		FROM repositories r
		LEFT JOIN (
			SELECT 
				repository_id,
				COUNT(*) as scan_count,
				MAX(created_at) as last_scan_date,
				(ARRAY_AGG(status ORDER BY created_at DESC))[1] as last_scan_status
			FROM scan_jobs
			WHERE created_at >= NOW() - INTERVAL '30 days'
			GROUP BY repository_id
		) scan_stats ON r.id = scan_stats.repository_id
		WHERE r.organization_id = $1
	`

	args := []interface{}{orgID}

	// Set default sort field for repositories
	if req.SortField == "" {
		req.SortField = "r.last_scan_at"
		req.SortOrder = "DESC"
	}

	return op.optimizer.PaginateQuery(ctx, baseQuery, req, args)
}

// PaginateFindings paginates findings with optimized queries
func (op *OptimizedPaginator) PaginateFindings(ctx context.Context, scanJobID uuid.UUID, req *PaginationRequest) (*PaginatedResult[map[string]interface{}], error) {
	baseQuery := `
		SELECT 
			f.id, f.scan_job_id, f.tool, f.rule_id, f.severity, f.category,
			f.title, f.description, f.file_path, f.line_number, f.column_number,
			f.confidence, f.consensus_score, f.status, f.created_at, f.updated_at
		FROM findings f
		WHERE f.scan_job_id = $1
	`

	args := []interface{}{scanJobID}

	// Set default sort field for findings
	if req.SortField == "" {
		req.SortField = "f.severity"
		req.SortOrder = "DESC"
	}

	return op.optimizer.PaginateQuery(ctx, baseQuery, req, args)
}