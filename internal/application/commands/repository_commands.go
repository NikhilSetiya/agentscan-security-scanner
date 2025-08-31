package commands

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/dto"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/services"
)

// CreateRepositoryCommand represents a command to create a repository
type CreateRepositoryCommand struct {
	OrganizationID uuid.UUID
	Name           string
	URL            string
	DefaultBranch  string
}

// UpdateRepositoryCommand represents a command to update a repository
type UpdateRepositoryCommand struct {
	RepositoryID uuid.UUID
	Name         string
	Description  string
	Language     string
}

// SetDefaultBranchCommand represents a command to set default branch
type SetDefaultBranchCommand struct {
	RepositoryID uuid.UUID
	Branch       string
}

// AddLanguageCommand represents a command to add a language
type AddLanguageCommand struct {
	RepositoryID uuid.UUID
	Language     string
}

// UpdateSettingsCommand represents a command to update repository settings
type UpdateSettingsCommand struct {
	RepositoryID uuid.UUID
	Settings     map[string]interface{}
}

// DeactivateRepositoryCommand represents a command to deactivate a repository
type DeactivateRepositoryCommand struct {
	RepositoryID uuid.UUID
}

// ActivateRepositoryCommand represents a command to activate a repository
type ActivateRepositoryCommand struct {
	RepositoryID uuid.UUID
}

// RepositoryCommandHandler handles repository-related commands
type RepositoryCommandHandler struct {
	repositoryService *services.RepositoryService
}

// NewRepositoryCommandHandler creates a new repository command handler
func NewRepositoryCommandHandler(repositoryService *services.RepositoryService) *RepositoryCommandHandler {
	return &RepositoryCommandHandler{
		repositoryService: repositoryService,
	}
}

// CreateRepository handles the create repository command
func (h *RepositoryCommandHandler) CreateRepository(ctx context.Context, cmd CreateRepositoryCommand) (*dto.RepositoryResponse, error) {
	repository, err := h.repositoryService.CreateRepository(ctx, cmd.OrganizationID, cmd.Name, cmd.URL, cmd.DefaultBranch)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToRepositoryResponse(repository)
	return &response, nil
}

// UpdateRepository handles the update repository command
func (h *RepositoryCommandHandler) UpdateRepository(ctx context.Context, cmd UpdateRepositoryCommand) error {
	return h.repositoryService.UpdateRepository(ctx, cmd.RepositoryID, cmd.Name, cmd.Description, cmd.Language)
}

// SetDefaultBranch handles the set default branch command
func (h *RepositoryCommandHandler) SetDefaultBranch(ctx context.Context, cmd SetDefaultBranchCommand) error {
	return h.repositoryService.SetDefaultBranch(ctx, cmd.RepositoryID, cmd.Branch)
}

// AddLanguage handles the add language command
func (h *RepositoryCommandHandler) AddLanguage(ctx context.Context, cmd AddLanguageCommand) error {
	return h.repositoryService.AddLanguage(ctx, cmd.RepositoryID, cmd.Language)
}

// UpdateSettings handles the update settings command
func (h *RepositoryCommandHandler) UpdateSettings(ctx context.Context, cmd UpdateSettingsCommand) error {
	return h.repositoryService.UpdateSettings(ctx, cmd.RepositoryID, cmd.Settings)
}

// DeactivateRepository handles the deactivate repository command
func (h *RepositoryCommandHandler) DeactivateRepository(ctx context.Context, cmd DeactivateRepositoryCommand) error {
	return h.repositoryService.DeactivateRepository(ctx, cmd.RepositoryID)
}

// ActivateRepository handles the activate repository command
func (h *RepositoryCommandHandler) ActivateRepository(ctx context.Context, cmd ActivateRepositoryCommand) error {
	return h.repositoryService.ActivateRepository(ctx, cmd.RepositoryID)
}