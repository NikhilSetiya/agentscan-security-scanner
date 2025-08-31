package services

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// RepositoryService provides business logic for repository operations
type RepositoryService struct {
	repoRepo repositories.RepositoryRepository
}

// NewRepositoryService creates a new repository service
func NewRepositoryService(repoRepo repositories.RepositoryRepository) *RepositoryService {
	return &RepositoryService{
		repoRepo: repoRepo,
	}
}

// CreateRepository creates a new repository with validation
func (s *RepositoryService) CreateRepository(ctx context.Context, orgID uuid.UUID, name, repoURL, defaultBranch string) (*entities.Repository, error) {
	// Parse and validate URL
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return nil, entities.NewValidationError("invalid repository URL")
	}

	// Extract provider information
	provider, providerID, err := s.parseRepositoryURL(parsedURL)
	if err != nil {
		return nil, err
	}

	// Check if repository already exists
	existingRepo, err := s.repoRepo.GetByURL(ctx, repoURL)
	if err == nil && existingRepo != nil {
		return nil, entities.NewConflictError("repository with this URL already exists")
	}

	// Set default branch if not provided
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	// Create new repository
	repo := entities.NewRepository(orgID, name, repoURL, provider, providerID, defaultBranch)

	// Validate repository
	if err := repo.Validate(); err != nil {
		return nil, err
	}

	// Save repository
	if err := s.repoRepo.Create(ctx, repo); err != nil {
		return nil, err
	}

	return repo, nil
}

// GetRepositoryByID retrieves a repository by ID
func (s *RepositoryService) GetRepositoryByID(ctx context.Context, repoID uuid.UUID) (*entities.Repository, error) {
	return s.repoRepo.GetByID(ctx, repoID)
}

// GetRepositoryByURL retrieves a repository by URL
func (s *RepositoryService) GetRepositoryByURL(ctx context.Context, repoURL string) (*entities.Repository, error) {
	return s.repoRepo.GetByURL(ctx, repoURL)
}

// UpdateRepository updates repository information
func (s *RepositoryService) UpdateRepository(ctx context.Context, repoID uuid.UUID, name, description, language string) error {
	// Get existing repository
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return err
	}

	// Update details
	repo.UpdateDetails(name, description, language)

	// Validate updated repository
	if err := repo.Validate(); err != nil {
		return err
	}

	// Save changes
	return s.repoRepo.Update(ctx, repo)
}

// SetDefaultBranch sets the default branch for a repository
func (s *RepositoryService) SetDefaultBranch(ctx context.Context, repoID uuid.UUID, branch string) error {
	// Get existing repository
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return err
	}

	// Set default branch
	repo.SetDefaultBranch(branch)

	// Save changes
	return s.repoRepo.Update(ctx, repo)
}

// AddLanguage adds a programming language to the repository
func (s *RepositoryService) AddLanguage(ctx context.Context, repoID uuid.UUID, language string) error {
	// Get existing repository
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return err
	}

	// Add language
	repo.AddLanguage(language)

	// Save changes
	return s.repoRepo.Update(ctx, repo)
}

// UpdateLastScanTime updates the last scan timestamp
func (s *RepositoryService) UpdateLastScanTime(ctx context.Context, repoID uuid.UUID) error {
	return s.repoRepo.UpdateLastScanTime(ctx, repoID)
}

// DeactivateRepository deactivates a repository
func (s *RepositoryService) DeactivateRepository(ctx context.Context, repoID uuid.UUID) error {
	return s.repoRepo.Deactivate(ctx, repoID)
}

// ActivateRepository activates a repository
func (s *RepositoryService) ActivateRepository(ctx context.Context, repoID uuid.UUID) error {
	return s.repoRepo.Activate(ctx, repoID)
}

// UpdateSettings updates repository settings
func (s *RepositoryService) UpdateSettings(ctx context.Context, repoID uuid.UUID, settings map[string]interface{}) error {
	return s.repoRepo.UpdateSettings(ctx, repoID, settings)
}

// GetSettings retrieves repository settings
func (s *RepositoryService) GetSettings(ctx context.Context, repoID uuid.UUID) (map[string]interface{}, error) {
	return s.repoRepo.GetSettings(ctx, repoID)
}

// ListRepositories lists repositories with filtering and pagination
func (s *RepositoryService) ListRepositories(ctx context.Context, orgID uuid.UUID, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.Repository, int64, error) {
	return s.repoRepo.ListByOrganization(ctx, orgID, filter, pagination)
}

// ListActiveRepositories lists only active repositories
func (s *RepositoryService) ListActiveRepositories(ctx context.Context, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.Repository, int64, error) {
	return s.repoRepo.ListActive(ctx, filter, pagination)
}

// ValidateRepositoryAccess validates if a user can access a repository
func (s *RepositoryService) ValidateRepositoryAccess(ctx context.Context, userID, repoID uuid.UUID) error {
	// Get repository
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		return err
	}

	// For now, we'll allow access to all repositories within the same organization
	// In a more complex system, you might have repository-specific permissions
	_ = repo // TODO: Implement organization-based access control

	return nil
}

// parseRepositoryURL extracts provider and provider ID from repository URL
func (s *RepositoryService) parseRepositoryURL(parsedURL *url.URL) (string, string, error) {
	host := strings.ToLower(parsedURL.Host)
	path := strings.Trim(parsedURL.Path, "/")

	switch {
	case strings.Contains(host, "github.com"):
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", entities.NewValidationError("invalid GitHub repository URL format")
		}
		providerID := strings.Join(parts[:2], "/")
		// Remove .git suffix if present
		providerID = strings.TrimSuffix(providerID, ".git")
		return "github", providerID, nil

	case strings.Contains(host, "gitlab.com"):
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", entities.NewValidationError("invalid GitLab repository URL format")
		}
		providerID := strings.Join(parts[:2], "/")
		// Remove .git suffix if present
		providerID = strings.TrimSuffix(providerID, ".git")
		return "gitlab", providerID, nil

	case strings.Contains(host, "bitbucket.org"):
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "", "", entities.NewValidationError("invalid Bitbucket repository URL format")
		}
		providerID := strings.Join(parts[:2], "/")
		// Remove .git suffix if present
		providerID = strings.TrimSuffix(providerID, ".git")
		return "bitbucket", providerID, nil

	default:
		return "git", path, nil // Generic git repository
	}
}