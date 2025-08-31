package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	BaseRepository[*entities.User, uuid.UUID]
	
	// GetByEmail retrieves a user by email address
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	
	// GetBySupabaseID retrieves a user by Supabase ID
	GetBySupabaseID(ctx context.Context, supabaseID string) (*entities.User, error)
	
	// UpdateProfile updates user profile information
	UpdateProfile(ctx context.Context, userID uuid.UUID, name, avatarURL string) error
	
	// UpdateRole updates user role
	UpdateRole(ctx context.Context, userID uuid.UUID, role entities.UserRole) error
	
	// ListByOrganization retrieves users by organization
	ListByOrganization(ctx context.Context, orgID uuid.UUID, filter Filter, pagination Pagination) ([]*entities.User, int64, error)
	
	// DeactivateUser deactivates a user account
	DeactivateUser(ctx context.Context, userID uuid.UUID) error
	
	// ActivateUser activates a user account
	ActivateUser(ctx context.Context, userID uuid.UUID) error
}