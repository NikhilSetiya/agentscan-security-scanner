package commands

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/dto"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/services"
)

// CreateUserCommand represents a command to create a user
type CreateUserCommand struct {
	Email string
	Name  string
	Role  entities.UserRole
}

// UpdateUserProfileCommand represents a command to update user profile
type UpdateUserProfileCommand struct {
	UserID    uuid.UUID
	Name      string
	AvatarURL string
}

// UpdateUserRoleCommand represents a command to update user role
type UpdateUserRoleCommand struct {
	AdminUserID  uuid.UUID
	TargetUserID uuid.UUID
	NewRole      entities.UserRole
}

// DeactivateUserCommand represents a command to deactivate a user
type DeactivateUserCommand struct {
	AdminUserID  uuid.UUID
	TargetUserID uuid.UUID
}

// ActivateUserCommand represents a command to activate a user
type ActivateUserCommand struct {
	AdminUserID  uuid.UUID
	TargetUserID uuid.UUID
}

// UserCommandHandler handles user-related commands
type UserCommandHandler struct {
	userService *services.UserService
}

// NewUserCommandHandler creates a new user command handler
func NewUserCommandHandler(userService *services.UserService) *UserCommandHandler {
	return &UserCommandHandler{
		userService: userService,
	}
}

// CreateUser handles the create user command
func (h *UserCommandHandler) CreateUser(ctx context.Context, cmd CreateUserCommand) (*dto.UserResponse, error) {
	user, err := h.userService.CreateUser(ctx, cmd.Email, cmd.Name, cmd.Role)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToUserResponse(user)
	return &response, nil
}

// UpdateUserProfile handles the update user profile command
func (h *UserCommandHandler) UpdateUserProfile(ctx context.Context, cmd UpdateUserProfileCommand) error {
	return h.userService.UpdateUserProfile(ctx, cmd.UserID, cmd.Name, cmd.AvatarURL)
}

// UpdateUserRole handles the update user role command
func (h *UserCommandHandler) UpdateUserRole(ctx context.Context, cmd UpdateUserRoleCommand) error {
	return h.userService.UpdateUserRole(ctx, cmd.AdminUserID, cmd.TargetUserID, cmd.NewRole)
}

// DeactivateUser handles the deactivate user command
func (h *UserCommandHandler) DeactivateUser(ctx context.Context, cmd DeactivateUserCommand) error {
	return h.userService.DeactivateUser(ctx, cmd.AdminUserID, cmd.TargetUserID)
}

// ActivateUser handles the activate user command
func (h *UserCommandHandler) ActivateUser(ctx context.Context, cmd ActivateUserCommand) error {
	return h.userService.ActivateUser(ctx, cmd.AdminUserID, cmd.TargetUserID)
}