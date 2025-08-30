package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/internal/api"
	"github.com/agentscan/agentscan/internal/application/dto"
	"github.com/agentscan/agentscan/internal/application/services"
	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/pkg/types"
)

// RepositoryHandler handles repository-related HTTP requests
type RepositoryHandler struct {
	*BaseHandler
	repositoryService *services.RepositoryService
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(repos repositories.Repositories, repositoryService *services.RepositoryService) *RepositoryHandler {
	return &RepositoryHandler{
		BaseHandler:       NewBaseHandler(repos),
		repositoryService: repositoryService,
	}
}

// CreateRepository creates a new repository
func (h *RepositoryHandler) CreateRepository(c *gin.Context) {
	var request dto.CreateRepositoryRequest

	h.HandleCreate[types.Repository](
		c,
		&request,
		func(ctx context.Context, req interface{}) (*types.Repository, error) {
			createReq := req.(*dto.CreateRepositoryRequest)
			return h.repositoryService.CreateRepository(ctx, createReq)
		},
		"repository",
	)
}

// GetRepository retrieves a repository by ID
func (h *RepositoryHandler) GetRepository(c *gin.Context) {
	h.HandleGet[types.Repository](
		c,
		func(ctx context.Context, id uuid.UUID) (*types.Repository, error) {
			return h.repositoryService.GetRepository(ctx, id)
		},
		"repository",
	)
}

// ListRepositories lists repositories with filtering and pagination
func (h *RepositoryHandler) ListRepositories(c *gin.Context) {
	h.HandleList[types.Repository](
		c,
		func(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*types.Repository, int, error) {
			return h.repositoryService.ListRepositories(ctx, filters, limit, offset)
		},
		h.buildRepositoryFilters,
		"repository",
	)
}

// UpdateRepository updates a repository
func (h *RepositoryHandler) UpdateRepository(c *gin.Context) {
	var request dto.UpdateRepositoryRequest

	h.HandleUpdate[types.Repository](
		c,
		&request,
		func(ctx context.Context, id uuid.UUID, req interface{}) (*types.Repository, error) {
			updateReq := req.(*dto.UpdateRepositoryRequest)
			return h.repositoryService.UpdateRepository(ctx, id, updateReq)
		},
		"repository",
	)
}

// DeleteRepository deletes a repository
func (h *RepositoryHandler) DeleteRepository(c *gin.Context) {
	h.HandleDelete(
		c,
		func(ctx context.Context, id uuid.UUID) error {
			return h.repositoryService.DeleteRepository(ctx, id)
		},
		"repository",
	)
}

// GetRepositoryByURL retrieves a repository by URL
func (h *RepositoryHandler) GetRepositoryByURL(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		h.BadRequest(c, "URL parameter is required")
		return
	}

	repo, err := h.repositoryService.GetRepositoryByURL(c.Request.Context(), url)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, repo)
}

// GetOrganizationRepositories retrieves repositories for an organization
func (h *RepositoryHandler) GetOrganizationRepositories(c *gin.Context) {
	orgID, err := h.GetUUIDParam(c, "orgId")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Check organization access
	if err := h.RequireOrganizationAccess(c, orgID); err != nil {
		h.Error(c, err)
		return
	}

	// Get pagination parameters
	limit, offset, err := h.GetPaginationParams(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Build filters
	filters := h.buildRepositoryFilters(c)
	filters["organization_id"] = orgID

	repos, total, err := h.repositoryService.ListRepositories(c.Request.Context(), filters, limit, offset)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Calculate pagination metadata
	page := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit

	meta := &api.Meta{
		Pagination: &api.Pagination{
			Page:       page,
			PageSize:   limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	}

	h.SuccessWithMeta(c, repos, meta)
}

// SyncRepository synchronizes a repository with its remote source
func (h *RepositoryHandler) SyncRepository(c *gin.Context) {
	id, err := h.GetUUIDParam(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Check permissions
	if err := h.RequireRole(c, "user"); err != nil {
		h.Error(c, err)
		return
	}

	err = h.repositoryService.SyncRepository(c.Request.Context(), id)
	if err != nil {
		h.LogError(c, err, "Failed to sync repository", map[string]interface{}{
			"repository_id": id,
		})
		h.Error(c, err)
		return
	}

	h.LogAction(c, "sync", "repository", map[string]interface{}{
		"repository_id": id,
	})

	h.Success(c, gin.H{"message": "Repository sync initiated successfully"})
}

// GetRepositoryStatistics retrieves repository statistics
func (h *RepositoryHandler) GetRepositoryStatistics(c *gin.Context) {
	id, err := h.GetUUIDParam(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	stats, err := h.repositoryService.GetRepositoryStatistics(c.Request.Context(), id)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, stats)
}

// GetActiveRepositories retrieves all active repositories
func (h *RepositoryHandler) GetActiveRepositories(c *gin.Context) {
	repos, err := h.repositoryService.GetActiveRepositories(c.Request.Context())
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, repos)
}

// GetRepositoriesByLanguage retrieves repositories by programming language
func (h *RepositoryHandler) GetRepositoriesByLanguage(c *gin.Context) {
	language := c.Query("language")
	if language == "" {
		h.BadRequest(c, "Language parameter is required")
		return
	}

	repos, err := h.repositoryService.GetRepositoriesByLanguage(c.Request.Context(), language)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, repos)
}

// buildRepositoryFilters builds filters from query parameters
func (h *RepositoryHandler) buildRepositoryFilters(c *gin.Context) map[string]interface{} {
	filters := make(map[string]interface{})

	// String filters
	if name := c.Query("name"); name != "" {
		filters["name"] = "%" + name + "%"
	}

	if url := c.Query("url"); url != "" {
		filters["url"] = "%" + url + "%"
	}

	if language := c.Query("language"); language != "" {
		filters["language"] = language
	}

	if provider := c.Query("provider"); provider != "" {
		filters["provider"] = provider
	}

	// Boolean filters
	if isActive := c.Query("is_active"); isActive != "" {
		if isActive == "true" {
			filters["is_active"] = true
		} else if isActive == "false" {
			filters["is_active"] = false
		}
	}

	if isPrivate := c.Query("is_private"); isPrivate != "" {
		if isPrivate == "true" {
			filters["is_private"] = true
		} else if isPrivate == "false" {
			filters["is_private"] = false
		}
	}

	// UUID filters
	if orgID := c.Query("organization_id"); orgID != "" {
		if parsedUUID, err := uuid.Parse(orgID); err == nil {
			filters["organization_id"] = parsedUUID
		}
	}

	// Date range filters
	if createdAfter := c.Query("created_after"); createdAfter != "" {
		filters["created_at >="] = createdAfter
	}

	if createdBefore := c.Query("created_before"); createdBefore != "" {
		filters["created_at <="] = createdBefore
	}

	if lastScanAfter := c.Query("last_scan_after"); lastScanAfter != "" {
		filters["last_scan_at >="] = lastScanAfter
	}

	if lastScanBefore := c.Query("last_scan_before"); lastScanBefore != "" {
		filters["last_scan_at <="] = lastScanBefore
	}

	return filters
}