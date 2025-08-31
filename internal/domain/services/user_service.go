package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// UserService provides business logic for user operations
type UserService struct {
	userRepo repositories.UserRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo repositories.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// CreateUser creates a new user with validation
func (s *UserService) CreateUser(ctx context.Context, email, name string, role entities.UserRole) (*entities.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, entities.NewConflictError("user with this email already exists")
	}

	// Create new user
	user := entities.NewUser(email, name, role)
	
	// Validate user
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Save user
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*entities.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*entities.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// GetUserBySupabaseID retrieves a user by Supabase ID
func (s *UserService) GetUserBySupabaseID(ctx context.Context, supabaseID string) (*entities.User, error) {
	return s.userRepo.GetBySupabaseID(ctx, supabaseID)
}

// UpdateUserProfile updates user profile information
func (s *UserService) UpdateUserProfile(ctx context.Context, userID uuid.UUID, name, avatarURL string) error {
	// Get existing user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Update profile
	user.UpdateProfile(name, avatarURL)

	// Validate updated user
	if err := user.Validate(); err != nil {
		return err
	}

	// Save changes
	return s.userRepo.Update(ctx, user)
}

// UpdateUserRole updates user role (admin only operation)
func (s *UserService) UpdateUserRole(ctx context.Context, adminUserID, targetUserID uuid.UUID, newRole entities.UserRole) error {
	// Verify admin user has permission
	adminUser, err := s.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return err
	}

	if !adminUser.IsAdmin() {
		return entities.NewBusinessRuleError("only admin users can update roles")
	}

	// Get target user
	targetUser, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}

	// Prevent admin from demoting themselves
	if adminUserID == targetUserID && newRole != entities.UserRoleAdmin {
		return entities.NewBusinessRuleError("admin users cannot demote themselves")
	}

	// Update role
	targetUser.Role = newRole
	targetUser.UpdatedAt = time.Now()

	// Save changes
	return s.userRepo.Update(ctx, targetUser)
}

// SetSupabaseID sets the Supabase ID for a user
func (s *UserService) SetSupabaseID(ctx context.Context, userID uuid.UUID, supabaseID string) error {
	// Get existing user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Set Supabase ID
	user.SetSupabaseID(supabaseID)

	// Save changes
	return s.userRepo.Update(ctx, user)
}

// ValidateUserAccess validates if a user can access a resource
func (s *UserService) ValidateUserAccess(ctx context.Context, userID uuid.UUID, requiredRole entities.UserRole) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.CanAccessResource(requiredRole) {
		return entities.NewBusinessRuleError("insufficient permissions")
	}

	return nil
}

// ListUsers lists users with filtering and pagination
func (s *UserService) ListUsers(ctx context.Context, orgID uuid.UUID, filter repositories.Filter, pagination repositories.Pagination) ([]*entities.User, int64, error) {
	return s.userRepo.ListByOrganization(ctx, orgID, filter, pagination)
}

// DeactivateUser deactivates a user account
func (s *UserService) DeactivateUser(ctx context.Context, adminUserID, targetUserID uuid.UUID) error {
	// Verify admin user has permission
	adminUser, err := s.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return err
	}

	if !adminUser.IsAdmin() {
		return entities.NewBusinessRuleError("only admin users can deactivate accounts")
	}

	// Prevent admin from deactivating themselves
	if adminUserID == targetUserID {
		return entities.NewBusinessRuleError("admin users cannot deactivate themselves")
	}

	return s.userRepo.DeactivateUser(ctx, targetUserID)
}

// ActivateUser activates a user account
func (s *UserService) ActivateUser(ctx context.Context, adminUserID, targetUserID uuid.UUID) error {
	// Verify admin user has permission
	adminUser, err := s.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return err
	}

	if !adminUser.IsAdmin() {
		return entities.NewBusinessRuleError("only admin users can activate accounts")
	}

	return s.userRepo.ActivateUser(ctx, targetUserID)
}