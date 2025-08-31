package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
)

// RepositoryRepository defines the interface for repository data operations
type RepositoryRepository interface {
	BaseRepository[*entities.Repository, uuid.UUID]
	
	// GetByURL retrieves a repository by its URL
	GetByURL(ctx context.Context, url string) (*entities.Repository, error)
	
	// GetByProviderID retrieves a repository by provider and provider ID
	GetByProviderID(ctx context.Context, provider, providerID string) (*entities.Repository, error)
	
	// ListByOrganization retrieves repositories by organization
	ListByOrganization(ctx context.Context, orgID uuid.UUID, filter Filter, pagination Pagination) ([]*entities.Repository, int64, error)
	
	// ListActive retrieves only active repositories
	ListActive(ctx context.Context, filter Filter, pagination Pagination) ([]*entities.Repository, int64, error)
	
	// UpdateLastScanTime updates the last scan timestamp
	UpdateLastScanTime(ctx context.Context, repoID uuid.UUID) error
	
	// Deactivate marks a repository as inactive
	Deactivate(ctx context.Context, repoID uuid.UUID) error
	
	// Activate marks a repository as active
	Activate(ctx context.Context, repoID uuid.UUID) error
	
	// UpdateSettings updates repository settings
	UpdateSettings(ctx context.Context, repoID uuid.UUID, settings map[string]interface{}) error
	
	// GetSettings retrieves repository settings
	GetSettings(ctx context.Context, repoID uuid.UUID) (map[string]interface{}, error)
}