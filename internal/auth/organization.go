package auth

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/types"
)

// OrganizationRepository defines the interface for organization database operations
type OrganizationRepository interface {
	CreateOrganization(ctx context.Context, org *types.Organization) error
	GetOrganization(ctx context.Context, orgID uuid.UUID) (*types.Organization, error)
	UpdateOrganization(ctx context.Context, org *types.Organization) error
	DeleteOrganization(ctx context.Context, orgID uuid.UUID) error
	GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*types.Organization, error)
	GetOrganizationMembers(ctx context.Context, orgID uuid.UUID) ([]*types.OrganizationMember, error)
	GetOrganizationMember(ctx context.Context, userID, orgID uuid.UUID) (*types.OrganizationMember, error)
	AddOrganizationMember(ctx context.Context, member *types.OrganizationMember) error
	RemoveOrganizationMember(ctx context.Context, orgID, userID uuid.UUID) error
	UpdateOrganizationMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
	SlugExists(ctx context.Context, slug string) (bool, error)
}

// OrganizationService provides organization management functionality
type OrganizationService struct {
	repos       *database.Repositories
	rbacService *RBACService
	orgRepo     OrganizationRepository
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(repos *database.Repositories, rbacService *RBACService, orgRepo OrganizationRepository) *OrganizationService {
	return &OrganizationService{
		repos:       repos,
		rbacService: rbacService,
		orgRepo:     orgRepo,
	}
}

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Slug string `json:"slug" binding:"required,min=1,max=50,alphanum"`
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// OrganizationDTO represents an organization in API responses
type OrganizationDTO struct {
	ID        uuid.UUID              `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	Settings  map[string]interface{} `json:"settings"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// OrganizationMemberDTO represents an organization member in API responses
type OrganizationMemberDTO struct {
	ID           uuid.UUID    `json:"id"`
	UserID       uuid.UUID    `json:"user_id"`
	Role         string       `json:"role"`
	User         *UserProfile `json:"user,omitempty"`
	JoinedAt     time.Time    `json:"joined_at"`
}

// InviteMemberRequest represents a request to invite a member to an organization
type InviteMemberRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=owner admin member"`
}

// UpdateMemberRoleRequest represents a request to update a member's role
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=owner admin member"`
}

// CreateOrganization creates a new organization
func (s *OrganizationService) CreateOrganization(ctx context.Context, userID uuid.UUID, req *CreateOrganizationRequest) (*OrganizationDTO, error) {
	// Validate slug format
	if !isValidSlug(req.Slug) {
		return nil, fmt.Errorf("invalid slug format: must contain only lowercase letters, numbers, and hyphens")
	}

	// Check if slug is already taken
	exists, err := s.orgRepo.SlugExists(ctx, req.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug availability: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("slug '%s' is already taken", req.Slug)
	}

	// Create organization
	org := &types.Organization{
		ID:        uuid.New(),
		Name:      req.Name,
		Slug:      req.Slug,
		Settings:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.orgRepo.CreateOrganization(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	// Add creator as owner
	member := &types.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           types.RoleOwner,
		CreatedAt:      time.Now(),
	}

	if err := s.orgRepo.AddOrganizationMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add organization owner: %w", err)
	}

	return toOrganizationDTO(org), nil
}

// GetOrganization gets an organization by ID
func (s *OrganizationService) GetOrganization(ctx context.Context, userID, orgID uuid.UUID) (*OrganizationDTO, error) {
	// Check if user has permission to read the organization
	hasPermission, err := s.rbacService.CheckPermission(ctx, userID, PermissionOrgRead, "organization", &orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, fmt.Errorf("insufficient permissions to read organization")
	}

	org, err := s.orgRepo.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return toOrganizationDTO(org), nil
}

// UpdateOrganization updates an organization
func (s *OrganizationService) UpdateOrganization(ctx context.Context, userID, orgID uuid.UUID, req *UpdateOrganizationRequest) (*OrganizationDTO, error) {
	// Check if user has permission to update the organization
	hasPermission, err := s.rbacService.CheckPermission(ctx, userID, PermissionOrgWrite, "organization", &orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, fmt.Errorf("insufficient permissions to update organization")
	}

	org, err := s.orgRepo.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Update fields
	org.Name = req.Name
	org.UpdatedAt = time.Now()

	if err := s.orgRepo.UpdateOrganization(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return toOrganizationDTO(org), nil
}

// DeleteOrganization deletes an organization
func (s *OrganizationService) DeleteOrganization(ctx context.Context, userID, orgID uuid.UUID) error {
	// Check if user has permission to delete the organization
	hasPermission, err := s.rbacService.CheckPermission(ctx, userID, PermissionOrgDelete, "organization", &orgID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to delete organization")
	}

	// TODO: Check if organization has repositories or other dependencies
	// TODO: Implement soft delete or cascade delete

	if err := s.orgRepo.DeleteOrganization(ctx, orgID); err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return nil
}

// ListUserOrganizations lists all organizations a user is a member of
func (s *OrganizationService) ListUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*OrganizationDTO, error) {
	orgs, err := s.orgRepo.GetUserOrganizations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	result := make([]*OrganizationDTO, len(orgs))
	for i, org := range orgs {
		result[i] = toOrganizationDTO(org)
	}

	return result, nil
}

// GetOrganizationMembers gets all members of an organization
func (s *OrganizationService) GetOrganizationMembers(ctx context.Context, userID, orgID uuid.UUID) ([]*OrganizationMemberDTO, error) {
	// Check if user has permission to read organization members
	hasPermission, err := s.rbacService.CheckPermission(ctx, userID, PermissionUserRead, "organization", &orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, fmt.Errorf("insufficient permissions to read organization members")
	}

	members, err := s.orgRepo.GetOrganizationMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization members: %w", err)
	}

	result := make([]*OrganizationMemberDTO, len(members))
	for i, member := range members {
		memberDTO := &OrganizationMemberDTO{
			ID:       member.ID,
			UserID:   member.UserID,
			Role:     member.Role,
			JoinedAt: member.CreatedAt,
		}

		// Optionally load user details
		user, err := s.repos.Users.GetByID(ctx, member.UserID)
		if err == nil {
			memberDTO.User = &UserProfile{
				ID:        user.ID,
				Email:     user.Email,
				Name:      user.Name,
				AvatarURL: user.AvatarURL,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			}
		}

		result[i] = memberDTO
	}

	return result, nil
}

// InviteMember invites a user to join an organization
func (s *OrganizationService) InviteMember(ctx context.Context, userID, orgID uuid.UUID, req *InviteMemberRequest) error {
	// Check if user has permission to invite members
	hasPermission, err := s.rbacService.CheckPermission(ctx, userID, PermissionUserInvite, "organization", &orgID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to invite members")
	}

	// Find user by email
	invitedUser, err := s.repos.Users.GetByEmail(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("user with email %s not found", req.Email)
	}

	// Check if user is already a member
	existingMember, err := s.orgRepo.GetOrganizationMember(ctx, invitedUser.ID, orgID)
	if err != nil {
		return fmt.Errorf("failed to check existing membership: %w", err)
	}
	if existingMember != nil {
		return fmt.Errorf("user is already a member of this organization")
	}

	// Add user as member
	member := &types.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         invitedUser.ID,
		Role:           req.Role,
		CreatedAt:      time.Now(),
	}

	if err := s.orgRepo.AddOrganizationMember(ctx, member); err != nil {
		return fmt.Errorf("failed to add organization member: %w", err)
	}

	// TODO: Send invitation email or notification

	return nil
}

// RemoveMember removes a member from an organization
func (s *OrganizationService) RemoveMember(ctx context.Context, userID, orgID, memberUserID uuid.UUID) error {
	// Check if user has permission to remove members
	hasPermission, err := s.rbacService.CheckPermission(ctx, userID, PermissionUserWrite, "organization", &orgID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to remove members")
	}

	// Don't allow removing the last owner
	if err := s.validateOwnerRemoval(ctx, orgID, memberUserID); err != nil {
		return err
	}

	if err := s.orgRepo.RemoveOrganizationMember(ctx, orgID, memberUserID); err != nil {
		return fmt.Errorf("failed to remove organization member: %w", err)
	}

	return nil
}

// UpdateMemberRole updates a member's role in an organization
func (s *OrganizationService) UpdateMemberRole(ctx context.Context, userID, orgID, memberUserID uuid.UUID, req *UpdateMemberRoleRequest) error {
	// Check if user has permission to update member roles
	hasPermission, err := s.rbacService.CheckPermission(ctx, userID, PermissionUserWrite, "organization", &orgID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return fmt.Errorf("insufficient permissions to update member roles")
	}

	// Don't allow changing the last owner's role
	if req.Role != types.RoleOwner {
		if err := s.validateOwnerRemoval(ctx, orgID, memberUserID); err != nil {
			return err
		}
	}

	if err := s.orgRepo.UpdateOrganizationMemberRole(ctx, orgID, memberUserID, req.Role); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	return nil
}

func (s *OrganizationService) validateOwnerRemoval(ctx context.Context, orgID, userID uuid.UUID) error {
	// TODO: Check if this is the last owner
	// If so, return an error
	return nil
}

// Utility functions

func isValidSlug(slug string) bool {
	// Slug should contain only lowercase letters, numbers, and hyphens
	// Should not start or end with a hyphen
	matched, _ := regexp.MatchString(`^[a-z0-9]+(-[a-z0-9]+)*$`, slug)
	return matched && len(slug) >= 1 && len(slug) <= 50
}

func toOrganizationDTO(org *types.Organization) *OrganizationDTO {
	return &OrganizationDTO{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Settings:  org.Settings,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
}