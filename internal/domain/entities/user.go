package entities

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents user roles in the system
type UserRole string

const (
	UserRoleAdmin     UserRole = "admin"
	UserRoleDeveloper UserRole = "developer"
	UserRoleViewer    UserRole = "viewer"
)

// User represents a user entity in the domain
type User struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email" validate:"required,email"`
	Name       string     `json:"name" validate:"required,min=1,max=255"`
	AvatarURL  string     `json:"avatar_url" validate:"omitempty,url"`
	SupabaseID *string    `json:"supabase_id,omitempty"`
	Role       UserRole   `json:"role" validate:"required"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// NewUser creates a new user entity
func NewUser(email, name string, role UserRole) *User {
	return &User{
		ID:        uuid.New(),
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// UpdateProfile updates user profile information
func (u *User) UpdateProfile(name, avatarURL string) {
	u.Name = name
	u.AvatarURL = avatarURL
	u.UpdatedAt = time.Now()
}

// SetSupabaseID sets the Supabase user ID
func (u *User) SetSupabaseID(supabaseID string) {
	u.SupabaseID = &supabaseID
	u.UpdatedAt = time.Now()
}

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// IsDeveloper checks if user has developer role or higher
func (u *User) IsDeveloper() bool {
	return u.Role == UserRoleAdmin || u.Role == UserRoleDeveloper
}

// CanAccessResource checks if user can access a resource based on role
func (u *User) CanAccessResource(requiredRole UserRole) bool {
	roleHierarchy := map[UserRole]int{
		UserRoleViewer:    1,
		UserRoleDeveloper: 2,
		UserRoleAdmin:     3,
	}

	userLevel := roleHierarchy[u.Role]
	requiredLevel := roleHierarchy[requiredRole]

	return userLevel >= requiredLevel
}

// Validate validates the user entity
func (u *User) Validate() error {
	if u.Email == "" {
		return NewValidationError("email is required")
	}
	if u.Name == "" {
		return NewValidationError("name is required")
	}
	if u.Role == "" {
		return NewValidationError("role is required")
	}
	return nil
}