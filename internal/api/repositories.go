package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// RepositoryHandler handles repository-related endpoints
type RepositoryHandler struct {
	repos *database.Repositories
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(repos *database.Repositories) *RepositoryHandler {
	return &RepositoryHandler{
		repos: repos,
	}
}

// CreateRepositoryRequest represents a request to create a repository
type CreateRepositoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	URL         string `json:"url" binding:"required,url"`
	Language    string `json:"language" binding:"required,min=1,max=100"`
	Branch      string `json:"branch"`
	Description string `json:"description"`
}

// UpdateRepositoryRequest represents a request to update a repository
type UpdateRepositoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Language    string `json:"language" binding:"required,min=1,max=100"`
	Branch      string `json:"branch"`
	Description string `json:"description"`
}

// ListRepositories retrieves a paginated list of repositories
func (h *RepositoryHandler) ListRepositories(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get current user for organization context
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// For now, use nil orgID to get all repositories
	// In a real implementation, you'd get the user's organization
	var orgID *uuid.UUID = nil

	pagination := &database.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	repoList, total, err := h.repos.Repositories.List(c.Request.Context(), orgID, pagination)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Filter by search term if provided
	if search != "" {
		var filteredRepos []*types.Repository
		searchLower := strings.ToLower(search)
		
		for _, repo := range repoList {
			if strings.Contains(strings.ToLower(repo.Name), searchLower) ||
			   strings.Contains(strings.ToLower(repo.URL), searchLower) ||
			   strings.Contains(strings.ToLower(repo.Language), searchLower) {
				filteredRepos = append(filteredRepos, repo)
			}
		}
		
		repoList = filteredRepos
		total = int64(len(filteredRepos))
	}

	// Convert to API format
	var repositories []map[string]interface{}
	for _, repo := range repoList {
		repoData := map[string]interface{}{
			"id":           repo.ID.String(),
			"name":         repo.Name,
			"url":          repo.URL,
			"language":     repo.Language,
			"branch":       repo.DefaultBranch,
			"description":  repo.Description,
			"created_at":   repo.CreatedAt.Format(time.RFC3339),
		}

		if repo.LastScanAt != nil {
			repoData["last_scan_at"] = repo.LastScanAt.Format(time.RFC3339)
		}

		repositories = append(repositories, repoData)
	}

	responseData := map[string]interface{}{
		"repositories": repositories,
	}

	PaginatedResponse(c, responseData, page, pageSize, total)
}

// GetRepository retrieves a single repository by ID
func (h *RepositoryHandler) GetRepository(c *gin.Context) {
	repoIDStr := c.Param("id")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		BadRequestResponse(c, "Invalid repository ID")
		return
	}

	// Get current user for authorization
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	repo, err := h.repos.Repositories.GetByID(c.Request.Context(), repoID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Convert to API format
	repoData := map[string]interface{}{
		"id":           repo.ID.String(),
		"name":         repo.Name,
		"url":          repo.URL,
		"language":     repo.Language,
		"branch":       repo.DefaultBranch,
		"description":  repo.Description,
		"created_at":   repo.CreatedAt.Format(time.RFC3339),
		"updated_at":   repo.UpdatedAt.Format(time.RFC3339),
	}

	if repo.LastScanAt != nil {
		repoData["last_scan_at"] = repo.LastScanAt.Format(time.RFC3339)
	}

	SuccessResponse(c, repoData)
}

// CreateRepository creates a new repository
func (h *RepositoryHandler) CreateRepository(c *gin.Context) {
	var req CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, "Invalid repository data", map[string]interface{}{
			"validation_errors": err.Error(),
		})
		return
	}

	// Get current user for organization context
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Validate and parse repository URL
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		BadRequestResponse(c, "Invalid repository URL")
		return
	}

	// Extract provider and provider ID from URL
	provider, providerID, err := h.parseRepositoryURL(parsedURL)
	if err != nil {
		BadRequestResponse(c, err.Error())
		return
	}

	// Check if repository already exists
	existingRepo, err := h.repos.Repositories.GetByURL(c.Request.Context(), req.URL)
	if err == nil && existingRepo != nil {
		ConflictResponse(c, "Repository with this URL already exists")
		return
	}

	// Set default branch if not provided
	branch := req.Branch
	if branch == "" {
		branch = "main"
	}

	// Create repository
	repo := &types.Repository{
		ID:             uuid.New(),
		OrganizationID: uuid.New(), // TODO: Get from user's organization
		Name:           req.Name,
		URL:            req.URL,
		Provider:       provider,
		ProviderID:     providerID,
		DefaultBranch:  branch,
		Language:       req.Language,
		Description:    req.Description,
		Languages:      []string{req.Language}, // Initialize with primary language
		Settings:       make(map[string]interface{}),
		IsActive:       true,
	}

	if err := h.repos.Repositories.Create(c.Request.Context(), repo); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Convert to API format
	repoData := map[string]interface{}{
		"id":          repo.ID.String(),
		"name":        repo.Name,
		"url":         repo.URL,
		"language":    repo.Language,
		"branch":      repo.DefaultBranch,
		"description": repo.Description,
		"created_at":  repo.CreatedAt.Format(time.RFC3339),
	}

	CreatedResponse(c, repoData)
}

// UpdateRepository updates an existing repository
func (h *RepositoryHandler) UpdateRepository(c *gin.Context) {
	repoIDStr := c.Param("id")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		BadRequestResponse(c, "Invalid repository ID")
		return
	}

	var req UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationErrorResponse(c, "Invalid repository data", map[string]interface{}{
			"validation_errors": err.Error(),
		})
		return
	}

	// Get current user for authorization
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Get existing repository
	repo, err := h.repos.Repositories.GetByID(c.Request.Context(), repoID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Update fields
	repo.Name = req.Name
	repo.Language = req.Language
	repo.Description = req.Description
	
	if req.Branch != "" {
		repo.DefaultBranch = req.Branch
	}

	// Update languages array if language changed
	if len(repo.Languages) == 0 || repo.Languages[0] != req.Language {
		repo.Languages = []string{req.Language}
	}

	if err := h.repos.Repositories.Update(c.Request.Context(), repo); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Convert to API format
	repoData := map[string]interface{}{
		"id":          repo.ID.String(),
		"name":        repo.Name,
		"url":         repo.URL,
		"language":    repo.Language,
		"branch":      repo.DefaultBranch,
		"description": repo.Description,
		"created_at":  repo.CreatedAt.Format(time.RFC3339),
		"updated_at":  repo.UpdatedAt.Format(time.RFC3339),
	}

	if repo.LastScanAt != nil {
		repoData["last_scan_at"] = repo.LastScanAt.Format(time.RFC3339)
	}

	SuccessResponse(c, repoData)
}

// DeleteRepository soft deletes a repository
func (h *RepositoryHandler) DeleteRepository(c *gin.Context) {
	repoIDStr := c.Param("id")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		BadRequestResponse(c, "Invalid repository ID")
		return
	}

	// Get current user for authorization
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Check if repository exists
	_, err = h.repos.Repositories.GetByID(c.Request.Context(), repoID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Soft delete the repository
	if err := h.repos.Repositories.Delete(c.Request.Context(), repoID); err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	SuccessResponse(c, map[string]string{
		"message": "Repository deleted successfully",
	})
}

// GetRepositoryScans retrieves scans for a specific repository
func (h *RepositoryHandler) GetRepositoryScans(c *gin.Context) {
	repoIDStr := c.Param("id")
	repoID, err := uuid.Parse(repoIDStr)
	if err != nil {
		BadRequestResponse(c, "Invalid repository ID")
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get current user for authorization
	_, exists := GetCurrentUserID(c)
	if !exists {
		UnauthorizedResponse(c, "User authentication required")
		return
	}

	// Check if repository exists
	_, err = h.repos.Repositories.GetByID(c.Request.Context(), repoID)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Get scans for this repository
	filter := &database.ScanJobFilter{
		RepositoryID: &repoID,
	}
	
	if status != "" {
		filter.Status = status
	}

	pagination := &database.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	scanJobs, total, err := h.repos.ScanJobs.List(c.Request.Context(), filter, pagination)
	if err != nil {
		ErrorResponseFromError(c, err)
		return
	}

	// Convert to API format
	var scans []map[string]interface{}
	for _, job := range scanJobs {
		// Calculate progress based on status
		progress := 0
		if job.Status == "completed" {
			progress = 100
		} else if job.Status == "running" {
			progress = 50 // Assume 50% for running scans
		}

		// Calculate duration
		duration := ""
		if job.StartedAt != nil && job.CompletedAt != nil {
			durationSeconds := int(job.CompletedAt.Sub(*job.StartedAt).Seconds())
			if durationSeconds >= 60 {
				minutes := durationSeconds / 60
				seconds := durationSeconds % 60
				duration = fmt.Sprintf("%dm %ds", minutes, seconds)
			} else {
				duration = fmt.Sprintf("%ds", durationSeconds)
			}
		}

		// Get findings count for this scan
		findings, err := h.repos.Findings.ListByScanJob(c.Request.Context(), job.ID)
		findingsCount := 0
		if err == nil {
			findingsCount = len(findings)
		}

		scanData := map[string]interface{}{
			"id":              job.ID.String(),
			"repository_id":   job.RepositoryID.String(),
			"status":          job.Status,
			"progress":        progress,
			"findings_count":  findingsCount,
			"branch":          job.Branch,
			"commit":          job.CommitSHA,
			"scan_type":       job.ScanType,
			"created_at":      job.CreatedAt.Format(time.RFC3339),
		}

		if job.StartedAt != nil {
			scanData["started_at"] = job.StartedAt.Format(time.RFC3339)
		}

		if job.CompletedAt != nil {
			scanData["completed_at"] = job.CompletedAt.Format(time.RFC3339)
		}

		if duration != "" {
			scanData["duration"] = duration
		}

		if job.ErrorMessage != "" {
			scanData["error_message"] = job.ErrorMessage
		}

		scans = append(scans, scanData)
	}

	responseData := map[string]interface{}{
		"scans": scans,
	}

	PaginatedResponse(c, responseData, page, pageSize, total)
}

// parseRepositoryURL extracts provider and provider ID from repository URL
func (h *RepositoryHandler) parseRepositoryURL(parsedURL *url.URL) (string, string, error) {
	host := strings.ToLower(parsedURL.Host)
	path := strings.Trim(parsedURL.Path, "/")

	switch {
	case strings.Contains(host, "github.com"):
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", errors.NewValidationError("Invalid GitHub repository URL format")
		}
		providerID := strings.Join(parts[:2], "/")
		// Remove .git suffix if present
		providerID = strings.TrimSuffix(providerID, ".git")
		return "github", providerID, nil

	case strings.Contains(host, "gitlab.com"):
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", errors.NewValidationError("Invalid GitLab repository URL format")
		}
		providerID := strings.Join(parts[:2], "/")
		// Remove .git suffix if present
		providerID = strings.TrimSuffix(providerID, ".git")
		return "gitlab", providerID, nil

	case strings.Contains(host, "bitbucket.org"):
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", errors.NewValidationError("Invalid Bitbucket repository URL format")
		}
		providerID := strings.Join(parts[:2], "/")
		// Remove .git suffix if present
		providerID = strings.TrimSuffix(providerID, ".git")
		return "bitbucket", providerID, nil

	default:
		return "git", path, nil // Generic git repository
	}
}