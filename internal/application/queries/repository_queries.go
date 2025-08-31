package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/dto"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/services"
)

// GetRepositoryByIDQuery represents a query to get repository by ID
type GetRepositoryByIDQuery struct {
	RepositoryID uuid.UUID
}

// GetRepositoryByURLQuery represents a query to get repository by URL
type GetRepositoryByURLQuery struct {
	URL string
}

// ListRepositoriesQuery represents a query to list repositories
type ListRepositoriesQuery struct {
	OrganizationID uuid.UUID
	Filter         repositories.Filter
	Pagination     repositories.Pagination
}

// ListActiveRepositoriesQuery represents a query to list active repositories
type ListActiveRepositoriesQuery struct {
	Filter     repositories.Filter
	Pagination repositories.Pagination
}

// GetRepositorySettingsQuery represents a query to get repository settings
type GetRepositorySettingsQuery struct {
	RepositoryID uuid.UUID
}

// RepositoryQueryHandler handles repository-related queries
type RepositoryQueryHandler struct {
	repositoryService *services.RepositoryService
}

// NewRepositoryQueryHandler creates a new repository query handler
func NewRepositoryQueryHandler(repositoryService *services.RepositoryService) *RepositoryQueryHandler {
	return &RepositoryQueryHandler{
		repositoryService: repositoryService,
	}
}

// GetRepositoryByID handles the get repository by ID query
func (h *RepositoryQueryHandler) GetRepositoryByID(ctx context.Context, query GetRepositoryByIDQuery) (*dto.RepositoryResponse, error) {
	repository, err := h.repositoryService.GetRepositoryByID(ctx, query.RepositoryID)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToRepositoryResponse(repository)
	return &response, nil
}

// GetRepositoryByURL handles the get repository by URL query
func (h *RepositoryQueryHandler) GetRepositoryByURL(ctx context.Context, query GetRepositoryByURLQuery) (*dto.RepositoryResponse, error) {
	repository, err := h.repositoryService.GetRepositoryByURL(ctx, query.URL)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToRepositoryResponse(repository)
	return &response, nil
}

// ListRepositories handles the list repositories query
func (h *RepositoryQueryHandler) ListRepositories(ctx context.Context, query ListRepositoriesQuery) (*dto.RepositoryListResponse, error) {
	repositories, total, err := h.repositoryService.ListRepositories(ctx, query.OrganizationID, query.Filter, query.Pagination)
	if err != nil {
		return nil, err
	}
	
	pagination := dto.CreatePagination(query.Pagination.Page, query.Pagination.PageSize, total)
	response := dto.ToRepositoryListResponse(repositories, pagination)
	return &response, nil
}

// ListActiveRepositories handles the list active repositories query
func (h *RepositoryQueryHandler) ListActiveRepositories(ctx context.Context, query ListActiveRepositoriesQuery) (*dto.RepositoryListResponse, error) {
	repositories, total, err := h.repositoryService.ListActiveRepositories(ctx, query.Filter, query.Pagination)
	if err != nil {
		return nil, err
	}
	
	pagination := dto.CreatePagination(query.Pagination.Page, query.Pagination.PageSize, total)
	response := dto.ToRepositoryListResponse(repositories, pagination)
	return &response, nil
}

// GetRepositorySettings handles the get repository settings query
func (h *RepositoryQueryHandler) GetRepositorySettings(ctx context.Context, query GetRepositorySettingsQuery) (map[string]interface{}, error) {
	return h.repositoryService.GetSettings(ctx, query.RepositoryID)
}