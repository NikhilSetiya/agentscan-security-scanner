package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Offset   int `json:"offset"`
}

// PaginationMiddleware extracts and validates pagination parameters
func PaginationMiddleware(defaultPageSize, maxPageSize int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageSize)))

		// Validate and normalize pagination parameters
		if page < 1 {
			page = 1
		}
		
		if pageSize < 1 {
			pageSize = defaultPageSize
		}
		
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}

		// Calculate offset
		offset := (page - 1) * pageSize

		// Store pagination parameters in context
		pagination := PaginationParams{
			Page:     page,
			PageSize: pageSize,
			Offset:   offset,
		}

		c.Set("pagination", pagination)
		c.Next()
	}
}

// GetPaginationFromContext extracts pagination parameters from context
func GetPaginationFromContext(c *gin.Context) PaginationParams {
	if pagination, exists := c.Get("pagination"); exists {
		if p, ok := pagination.(PaginationParams); ok {
			return p
		}
	}
	
	// Return default if not found
	return PaginationParams{
		Page:     1,
		PageSize: 20,
		Offset:   0,
	}
}

// FilterParams represents common filtering parameters
type FilterParams struct {
	Search    string `json:"search,omitempty"`
	Status    string `json:"status,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// FilterMiddleware extracts and validates common filter parameters
func FilterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		filters := FilterParams{
			Search:    c.Query("search"),
			Status:    c.Query("status"),
			SortBy:    c.DefaultQuery("sort_by", "created_at"),
			SortOrder: c.DefaultQuery("sort_order", "desc"),
		}

		// Validate sort order
		if filters.SortOrder != "asc" && filters.SortOrder != "desc" {
			filters.SortOrder = "desc"
		}

		// Store filter parameters in context
		c.Set("filters", filters)
		c.Next()
	}
}

// GetFiltersFromContext extracts filter parameters from context
func GetFiltersFromContext(c *gin.Context) FilterParams {
	if filters, exists := c.Get("filters"); exists {
		if f, ok := filters.(FilterParams); ok {
			return f
		}
	}
	
	// Return default if not found
	return FilterParams{
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}