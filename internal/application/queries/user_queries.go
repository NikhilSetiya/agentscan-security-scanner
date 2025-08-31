package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/dto"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/services"
)

// GetUserByIDQuery represents a query to get user by ID
type GetUserByIDQuery struct {
	UserID uuid.UUID
}

// GetUserByEmailQuery represents a query to get user by email
type GetUserByEmailQuery struct {
	Email string
}

// GetUserBySupabaseIDQuery represents a query to get user by Supabase ID
type GetUserBySupabaseIDQuery struct {
	SupabaseID string
}

// ListUsersQuery represents a query to list users
type ListUsersQuery struct {
	OrganizationID uuid.UUID
	Filter         repositories.Filter
	Pagination     repositories.Pagination
}

// UserQueryHandler handles user-related queries
type UserQueryHandler struct {
	userService *services.UserService
}

// NewUserQueryHandler creates a new user query handler
func NewUserQueryHandler(userService *services.UserService) *UserQueryHandler {
	return &UserQueryHandler{
		userService: userService,
	}
}

// GetUserByID handles the get user by ID query
func (h *UserQueryHandler) GetUserByID(ctx context.Context, query GetUserByIDQuery) (*dto.UserResponse, error) {
	user, err := h.userService.GetUserByID(ctx, query.UserID)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToUserResponse(user)
	return &response, nil
}

// GetUserByEmail handles the get user by email query
func (h *UserQueryHandler) GetUserByEmail(ctx context.Context, query GetUserByEmailQuery) (*dto.UserResponse, error) {
	user, err := h.userService.GetUserByEmail(ctx, query.Email)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToUserResponse(user)
	return &response, nil
}

// GetUserBySupabaseID handles the get user by Supabase ID query
func (h *UserQueryHandler) GetUserBySupabaseID(ctx context.Context, query GetUserBySupabaseIDQuery) (*dto.UserResponse, error) {
	user, err := h.userService.GetUserBySupabaseID(ctx, query.SupabaseID)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToUserResponse(user)
	return &response, nil
}

// ListUsers handles the list users query
func (h *UserQueryHandler) ListUsers(ctx context.Context, query ListUsersQuery) (*dto.UserListResponse, error) {
	users, total, err := h.userService.ListUsers(ctx, query.OrganizationID, query.Filter, query.Pagination)
	if err != nil {
		return nil, err
	}
	
	pagination := dto.CreatePagination(query.Pagination.Page, query.Pagination.PageSize, total)
	response := dto.ToUserListResponse(users, pagination)
	return &response, nil
}