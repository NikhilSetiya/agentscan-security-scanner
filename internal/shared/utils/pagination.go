package utils

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page     int `json:"page" form:"page" validate:"min=1"`
	PageSize int `json:"page_size" form:"limit" validate:"min=1,max=100"`
}

// PaginationResponse represents pagination metadata in responses
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// ParsePaginationFromQuery extracts pagination parameters from query string
func ParsePaginationFromQuery(c *gin.Context, defaultPageSize, maxPageSize int) PaginationRequest {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageSize)))

	// Validate and normalize
	if page < 1 {
		page = 1
	}
	
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return PaginationRequest{
		Page:     page,
		PageSize: pageSize,
	}
}

// CalculateOffset calculates the database offset from page and page size
func (p PaginationRequest) CalculateOffset() int {
	return (p.Page - 1) * p.PageSize
}

// CreatePaginationResponse creates pagination metadata for responses
func CreatePaginationResponse(page, pageSize int, total int64) PaginationResponse {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	
	if totalPages == 0 {
		totalPages = 1
	}

	return PaginationResponse{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// ValidatePagination validates pagination parameters
func ValidatePagination(page, pageSize, maxPageSize int) (int, int, error) {
	if page < 1 {
		page = 1
	}
	
	if pageSize < 1 {
		pageSize = 20 // default
	}
	
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize, nil
}

// PaginateSlice paginates a slice in memory (for small datasets)
func PaginateSlice[T any](items []T, page, pageSize int) ([]T, PaginationResponse) {
	total := int64(len(items))
	
	// Calculate pagination
	pagination := CreatePaginationResponse(page, pageSize, total)
	
	// Calculate slice bounds
	start := (page - 1) * pageSize
	end := start + pageSize
	
	// Handle bounds
	if start >= len(items) {
		return []T{}, pagination
	}
	
	if end > len(items) {
		end = len(items)
	}
	
	return items[start:end], pagination
}

// DatabasePagination represents database-specific pagination parameters
type DatabasePagination struct {
	Limit  int
	Offset int
}

// ToDatabase converts pagination request to database parameters
func (p PaginationRequest) ToDatabase() DatabasePagination {
	return DatabasePagination{
		Limit:  p.PageSize,
		Offset: p.CalculateOffset(),
	}
}

// PaginatedResult represents a paginated result set
type PaginatedResult[T any] struct {
	Items      []T                `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

// NewPaginatedResult creates a new paginated result
func NewPaginatedResult[T any](items []T, page, pageSize int, total int64) PaginatedResult[T] {
	return PaginatedResult[T]{
		Items:      items,
		Pagination: CreatePaginationResponse(page, pageSize, total),
	}
}