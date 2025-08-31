package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/commands"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/queries"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/services"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// RepositoryHandler handles HTTP requests for repository operations
type RepositoryHandler struct {
	appService *services.ApplicationService
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(appService *services.ApplicationService) *RepositoryHandler {
	return &RepositoryHandler{
		appService: appService,
	}
}

// CreateRepositoryRequest represents the request to create a repository
type CreateRepositoryRequest struct {
	OrganizationID uuid.UUID `json:"organization_id" binding:"required"`
	Name           string    `json:"name" binding:"required,min=1,max=255"`
	URL            string    `json:"url" binding:"required,url"`
	DefaultBranch  string    `json:"default_branch" binding:"required,min=1,max=255"`
	Description    string    `json:"description" binding:"omitempty,max=1000"`
	Language       string    `json:"language" binding:"omitempty,max=50"`
}

// UpdateRepositoryRequest represents the request to update a repository
type UpdateRepositoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"omitempty,max=1000"`
	Language    string `json:"language" binding:"omitempty,max=50"`
}

// SetDefaultBranchRequest represents the request to set default branch
type SetDefaultBranchRequest struct {
	Branch string `json:"branch" binding:"required,min=1,max=255"`
}

// AddLanguageRequest represents the request to add a language
type AddLanguageRequest struct {
	Language string `json:"language" binding:"required,min=1,max=50"`
}

// UpdateRepositorySettingsRequest represents the request to update repository settings
type UpdateRepositorySettingsRequest struct {
	Settings map[string]interface{} `json:"settings" binding:"required"`
}

// CreateRepository handles POST /repositories
func (h *RepositoryHandler) CreateRepository(c *gin.Context) {
	var req CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.CreateRepositoryCommand{
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		URL:            req.URL,
		DefaultBranch:  req.DefaultBranch,
		Description:    req.Description,
		Language:       req.Language,
	}

	repository, err := h.appService.CreateRepository(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    repository,
	})
}

// GetRepository handles GET /repositories/:id
func (h *RepositoryHandler) GetRepository(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	query := queries.GetRepositoryByIDQuery{
		RepositoryID: repositoryID,
	}

	repository, err := h.appService.GetRepositoryByID(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    repository,
	})
}

// GetRepositoryByURL handles GET /repositories/by-url
func (h *RepositoryHandler) GetRepositoryByURL(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}

	query := queries.GetRepositoryByURLQuery{
		URL: url,
	}

	repository, err := h.appService.GetRepositoryByURL(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    repository,
	})
}

// ListRepositories handles GET /repositories
func (h *RepositoryHandler) ListRepositories(c *gin.Context) {
	orgID, err := parseUUID(c.Query("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid organization ID",
		})
		return
	}

	// Parse pagination parameters
	page, pageSize := parsePagination(c)
	
	// Parse filter parameters
	filter := repositories.Filter{
		Search:    c.Query("search"),
		Status:    c.Query("status"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	pagination := repositories.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	query := queries.ListRepositoriesQuery{
		OrganizationID: orgID,
		Filter:         filter,
		Pagination:     pagination,
	}

	repositories, total, err := h.appService.ListRepositories(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    repositories,
		"meta": gin.H{
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
			},
		},
	})
}

// ListActiveRepositories handles GET /repositories/active
func (h *RepositoryHandler) ListActiveRepositories(c *gin.Context) {
	// Parse pagination parameters
	page, pageSize := parsePagination(c)
	
	// Parse filter parameters
	filter := repositories.Filter{
		Search:    c.Query("search"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	pagination := repositories.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	query := queries.ListActiveRepositoriesQuery{
		Filter:     filter,
		Pagination: pagination,
	}

	repositories, total, err := h.appService.ListActiveRepositories(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    repositories,
		"meta": gin.H{
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
			},
		},
	})
}

// UpdateRepository handles PUT /repositories/:id
func (h *RepositoryHandler) UpdateRepository(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	var req UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.UpdateRepositoryCommand{
		RepositoryID: repositoryID,
		Name:         req.Name,
		Description:  req.Description,
		Language:     req.Language,
	}

	err = h.appService.UpdateRepository(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Repository updated successfully",
	})
}

// SetDefaultBranch handles PUT /repositories/:id/default-branch
func (h *RepositoryHandler) SetDefaultBranch(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	var req SetDefaultBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.SetDefaultBranchCommand{
		RepositoryID: repositoryID,
		Branch:       req.Branch,
	}

	err = h.appService.SetDefaultBranch(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Default branch updated successfully",
	})
}

// AddLanguage handles POST /repositories/:id/languages
func (h *RepositoryHandler) AddLanguage(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	var req AddLanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.AddLanguageCommand{
		RepositoryID: repositoryID,
		Language:     req.Language,
	}

	err = h.appService.AddLanguage(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Language added successfully",
	})
}

// UpdateRepositorySettings handles PUT /repositories/:id/settings
func (h *RepositoryHandler) UpdateRepositorySettings(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	var req UpdateRepositorySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.UpdateRepositorySettingsCommand{
		RepositoryID: repositoryID,
		Settings:     req.Settings,
	}

	err = h.appService.UpdateRepositorySettings(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Repository settings updated successfully",
	})
}

// GetRepositorySettings handles GET /repositories/:id/settings
func (h *RepositoryHandler) GetRepositorySettings(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	query := queries.GetRepositorySettingsQuery{
		RepositoryID: repositoryID,
	}

	settings, err := h.appService.GetRepositorySettings(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// DeactivateRepository handles POST /repositories/:id/deactivate
func (h *RepositoryHandler) DeactivateRepository(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	cmd := commands.DeactivateRepositoryCommand{
		RepositoryID: repositoryID,
	}

	err = h.appService.DeactivateRepository(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Repository deactivated successfully",
	})
}

// ActivateRepository handles POST /repositories/:id/activate
func (h *RepositoryHandler) ActivateRepository(c *gin.Context) {
	repositoryID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid repository ID",
		})
		return
	}

	cmd := commands.ActivateRepositoryCommand{
		RepositoryID: repositoryID,
	}

	err = h.appService.ActivateRepository(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Repository activated successfully",
	})
}