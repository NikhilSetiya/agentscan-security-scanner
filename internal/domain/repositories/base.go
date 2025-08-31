package repositories

import (
	"context"
)

// Filter represents common filtering parameters
type Filter struct {
	Search    string
	Status    string
	SortBy    string
	SortOrder string
}

// Pagination represents pagination parameters
type Pagination struct {
	Page       int   `json:"page" validate:"min=1"`
	PageSize   int   `json:"page_size" validate:"min=1,max=100"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// BaseRepository defines common repository operations
type BaseRepository[T any, ID comparable] interface {
	// Create creates a new entity
	Create(ctx context.Context, entity T) error
	
	// GetByID retrieves an entity by its ID
	GetByID(ctx context.Context, id ID) (T, error)
	
	// Update updates an existing entity
	Update(ctx context.Context, entity T) error
	
	// Delete deletes an entity by its ID
	Delete(ctx context.Context, id ID) error
	
	// List retrieves entities with filtering and pagination
	List(ctx context.Context, filter Filter, pagination Pagination) ([]T, int64, error)
	
	// Exists checks if an entity exists by its ID
	Exists(ctx context.Context, id ID) (bool, error)
	
	// Count returns the total count of entities matching the filter
	Count(ctx context.Context, filter Filter) (int64, error)
}