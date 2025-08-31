package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
)

// CreateRepositoryRequest represents a request to create a repository
type CreateRepositoryRequest struct {
	Name          string `json:"name" validate:"required,min=1,max=255"`
	URL           string `json:"url" validate:"required,url"`
	DefaultBranch string `json:"default_branch" validate:"omitempty,min=1,max=100"`
	Language      string `json:"language" validate:"omitempty,min=1,max=100"`
	Description   string `json:"description" validate:"omitempty,max=1000"`
}

// UpdateRepositoryRequest represents a request to update a repository
type UpdateRepositoryRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Language    string `json:"language" validate:"omitempty,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=1000"`
}

// UpdateRepositorySettingsRequest represents a request to update repository settings
type UpdateRepositorySettingsRequest struct {
	Settings map[string]interface{} `json:"settings" validate:"required"`
}

// RepositoryResponse represents a repository in API responses
type RepositoryResponse struct {
	ID             uuid.UUID              `json:"id"`
	OrganizationID uuid.UUID              `json:"organization_id"`
	Name           string                 `json:"name"`
	URL            string                 `json:"url"`
	Provider       string                 `json:"provider"`
	ProviderID     string                 `json:"provider_id"`
	DefaultBranch  string                 `json:"default_branch"`
	Language       string                 `json:"language"`
	Description    string                 `json:"description"`
	Languages      []string               `json:"languages"`
	Settings       map[string]interface{} `json:"settings"`
	IsActive       bool                   `json:"is_active"`
	LastScanAt     *time.Time             `json:"last_scan_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// RepositoryListResponse represents a paginated list of repositories
type RepositoryListResponse struct {
	Repositories []RepositoryResponse `json:"repositories"`
	Pagination   Pagination           `json:"pagination"`
}

// ToRepositoryResponse converts a domain repository entity to response DTO
func ToRepositoryResponse(repo *entities.Repository) RepositoryResponse {
	return RepositoryResponse{
		ID:             repo.ID,
		OrganizationID: repo.OrganizationID,
		Name:           repo.Name,
		URL:            repo.URL,
		Provider:       repo.Provider,
		ProviderID:     repo.ProviderID,
		DefaultBranch:  repo.DefaultBranch,
		Language:       repo.Language,
		Description:    repo.Description,
		Languages:      repo.Languages,
		Settings:       repo.Settings,
		IsActive:       repo.IsActive,
		LastScanAt:     repo.LastScanAt,
		CreatedAt:      repo.CreatedAt,
		UpdatedAt:      repo.UpdatedAt,
	}
}

// ToRepositoryListResponse converts a list of domain repository entities to response DTO
func ToRepositoryListResponse(repos []*entities.Repository, pagination Pagination) RepositoryListResponse {
	repoResponses := make([]RepositoryResponse, len(repos))
	for i, repo := range repos {
		repoResponses[i] = ToRepositoryResponse(repo)
	}
	
	return RepositoryListResponse{
		Repositories: repoResponses,
		Pagination:   pagination,
	}
}