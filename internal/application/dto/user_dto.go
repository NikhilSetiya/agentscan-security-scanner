package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
)

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Email string             `json:"email" validate:"required,email"`
	Name  string             `json:"name" validate:"required,min=1,max=255"`
	Role  entities.UserRole  `json:"role" validate:"required"`
}

// UpdateUserProfileRequest represents a request to update user profile
type UpdateUserProfileRequest struct {
	Name      string `json:"name" validate:"required,min=1,max=255"`
	AvatarURL string `json:"avatar_url" validate:"omitempty,url"`
}

// UpdateUserRoleRequest represents a request to update user role
type UpdateUserRoleRequest struct {
	Role entities.UserRole `json:"role" validate:"required"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID        uuid.UUID         `json:"id"`
	Email     string            `json:"email"`
	Name      string            `json:"name"`
	AvatarURL string            `json:"avatar_url"`
	Role      entities.UserRole `json:"role"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// UserListResponse represents a paginated list of users
type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	Pagination Pagination     `json:"pagination"`
}

// ToUserResponse converts a domain user entity to response DTO
func ToUserResponse(user *entities.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// ToUserListResponse converts a list of domain user entities to response DTO
func ToUserListResponse(users []*entities.User, pagination Pagination) UserListResponse {
	userResponses := make([]UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = ToUserResponse(user)
	}
	
	return UserListResponse{
		Users:      userResponses,
		Pagination: pagination,
	}
}