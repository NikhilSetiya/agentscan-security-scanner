package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/types"
)

// UserService provides user management functionality
type UserService struct {
	repos *database.Repositories
}

// NewUserService creates a new user service
func NewUserService(repos *database.Repositories) *UserService {
	return &UserService{
		repos: repos,
	}
}

// UserProfile represents a user profile for API responses
type UserProfile struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	GitHubID  *int      `json:"github_id,omitempty"`
	GitLabID  *int      `json:"gitlab_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateProfileRequest represents a request to update user profile
type UpdateProfileRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=255"`
	AvatarURL string `json:"avatar_url" binding:"omitempty,url"`
}

// ChangeEmailRequest represents a request to change user email
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
}

// GetProfile returns the user profile
func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	user, err := s.repos.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		GitHubID:  user.GitHubID,
		GitLabID:  user.GitLabID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// UpdateProfile updates the user profile
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) (*UserProfile, error) {
	user, err := s.repos.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update fields
	user.Name = req.Name
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}
	user.UpdatedAt = time.Now()

	if err := s.repos.Users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		GitHubID:  user.GitHubID,
		GitLabID:  user.GitLabID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// ChangeEmail changes the user's email address
func (s *UserService) ChangeEmail(ctx context.Context, userID uuid.UUID, req *ChangeEmailRequest) error {
	// Check if email is already in use
	existingUser, err := s.repos.Users.GetByEmail(ctx, req.NewEmail)
	if err == nil && existingUser.ID != userID {
		return fmt.Errorf("email already in use")
	}

	user, err := s.repos.Users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.Email = req.NewEmail
	user.UpdatedAt = time.Now()

	if err := s.repos.Users.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user email: %w", err)
	}

	return nil
}

// DeleteAccount deletes a user account
func (s *UserService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	// TODO: Implement proper account deletion with data cleanup
	// This should include:
	// - Removing user from organizations
	// - Cleaning up scan jobs and findings
	// - Invalidating all sessions
	// - Audit logging

	if err := s.repos.Users.Delete(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// GetUserByEmail returns a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	return s.repos.Users.GetByEmail(ctx, email)
}

// GetUserByID returns a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*types.User, error) {
	return s.repos.Users.GetByID(ctx, userID)
}

// ListUsers returns a paginated list of users (admin only)
func (s *UserService) ListUsers(ctx context.Context, pagination *database.Pagination) ([]*types.User, int64, error) {
	return s.repos.Users.List(ctx, pagination)
}

// UserStats represents user statistics
type UserStats struct {
	TotalUsers   int64 `json:"total_users"`
	ActiveUsers  int64 `json:"active_users"`
	NewUsers     int64 `json:"new_users"`
	GitHubUsers  int64 `json:"github_users"`
	GitLabUsers  int64 `json:"gitlab_users"`
}

// GetUserStats returns user statistics (admin only)
func (s *UserService) GetUserStats(ctx context.Context) (*UserStats, error) {
	// TODO: Implement user statistics queries
	return &UserStats{
		TotalUsers:   0,
		ActiveUsers:  0,
		NewUsers:     0,
		GitHubUsers:  0,
		GitLabUsers:  0,
	}, nil
}

// UserActivity represents user activity information
type UserActivity struct {
	UserID       uuid.UUID `json:"user_id"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	LastActiveAt *time.Time `json:"last_active_at"`
	LoginCount   int64     `json:"login_count"`
	ScanCount    int64     `json:"scan_count"`
}

// GetUserActivity returns user activity information
func (s *UserService) GetUserActivity(ctx context.Context, userID uuid.UUID) (*UserActivity, error) {
	// TODO: Implement user activity tracking
	return &UserActivity{
		UserID:       userID,
		LastLoginAt:  nil,
		LastActiveAt: nil,
		LoginCount:   0,
		ScanCount:    0,
	}, nil
}

// UpdateLastActive updates the user's last active timestamp
func (s *UserService) UpdateLastActive(ctx context.Context, userID uuid.UUID) error {
	// TODO: Implement last active tracking
	// This could be called from middleware to track user activity
	return nil
}