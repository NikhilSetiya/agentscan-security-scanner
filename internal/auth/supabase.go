package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"
	"github.com/supabase-community/gotrue-go"
	"github.com/supabase-community/gotrue-go/types"
)

// SupabaseClient wraps the Supabase client for authentication operations
type SupabaseClient struct {
	client     *supabase.Client
	authClient gotrue.Client
	jwtSecret  string
}

// NewSupabaseClient creates a new Supabase client for authentication
func NewSupabaseClient() (*SupabaseClient, error) {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseServiceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	supabaseJWTSecret := os.Getenv("SUPABASE_JWT_SECRET")

	if supabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL environment variable is required")
	}

	if supabaseServiceKey == "" {
		return nil, fmt.Errorf("SUPABASE_SERVICE_ROLE_KEY environment variable is required")
	}

	if supabaseJWTSecret == "" {
		return nil, fmt.Errorf("SUPABASE_JWT_SECRET environment variable is required")
	}

	// Create main Supabase client
	client, err := supabase.NewClient(supabaseURL, supabaseServiceKey, &supabase.ClientOptions{
		Headers: map[string]string{
			"X-Client-Info": "agentscan-backend@1.0.0",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Supabase client: %w", err)
	}

	// Extract project reference from URL (e.g., https://abc123.supabase.co -> abc123)
	projectRef := ""
	if strings.Contains(supabaseURL, ".supabase.co") {
		parts := strings.Split(supabaseURL, ".")
		if len(parts) > 0 {
			urlParts := strings.Split(parts[0], "//")
			if len(urlParts) > 1 {
				projectRef = urlParts[1]
			}
		}
	}

	// Create GoTrue client for auth operations
	authClient := gotrue.New(projectRef, supabaseServiceKey)

	return &SupabaseClient{
		client:     client,
		authClient: authClient,
		jwtSecret:  supabaseJWTSecret,
	}, nil
}

// SupabaseUser represents a user from Supabase Auth
type SupabaseUser struct {
	ID       string                 `json:"id"`
	Email    string                 `json:"email"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"user_metadata,omitempty"`
	AppMetadata map[string]interface{} `json:"app_metadata,omitempty"`
}

// SupabaseJWTClaims represents the JWT token claims from Supabase
type SupabaseJWTClaims struct {
	Email        string                 `json:"email"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
	AppMetadata  map[string]interface{} `json:"app_metadata"`
	Role         string                 `json:"role"`
	jwt.RegisteredClaims
}

// ValidateToken validates a Supabase JWT token and returns user information
func (sc *SupabaseClient) ValidateToken(ctx context.Context, tokenString string) (*SupabaseUser, error) {
	// Parse and validate Supabase JWT token
	token, err := jwt.ParseWithClaims(tokenString, &SupabaseJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(sc.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse Supabase token: %w", err)
	}

	claims, ok := token.Claims.(*SupabaseJWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid Supabase token claims")
	}

	// Check token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("Supabase token has expired")
	}

	// Extract user information from claims
	email := claims.Email
	name := ""

	// Extract name from user metadata
	if claims.UserMetadata != nil {
		if nameVal, ok := claims.UserMetadata["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
			}
		}
	}

	// If no name in metadata, try to extract from email
	if name == "" && email != "" {
		if atIndex := strings.Index(email, "@"); atIndex > 0 {
			name = email[:atIndex]
		}
	}

	return &SupabaseUser{
		ID:          claims.Subject,
		Email:       email,
		Name:        name,
		Metadata:    claims.UserMetadata,
		AppMetadata: claims.AppMetadata,
	}, nil
}

// GetUserByID retrieves a user by their Supabase ID
func (sc *SupabaseClient) GetUserByID(ctx context.Context, userID string) (*SupabaseUser, error) {
	// Parse user ID as UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	// Use admin API to get user by ID
	resp, err := sc.authClient.AdminGetUser(types.AdminGetUserRequest{
		UserID: userUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	user := resp.User

	// Extract name from user metadata
	name := ""
	if user.UserMetadata != nil {
		if nameVal, ok := user.UserMetadata["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
			}
		}
	}

	// If no name in metadata, try to extract from email
	if name == "" && user.Email != "" {
		if atIndex := strings.Index(user.Email, "@"); atIndex > 0 {
			name = user.Email[:atIndex]
		}
	}

	return &SupabaseUser{
		ID:          user.ID.String(),
		Email:       user.Email,
		Name:        name,
		Metadata:    user.UserMetadata,
		AppMetadata: user.AppMetadata,
	}, nil
}

// CreateUser creates a new user in Supabase (admin operation)
func (sc *SupabaseClient) CreateUser(ctx context.Context, email, password, name string) (*SupabaseUser, error) {
	req := types.AdminCreateUserRequest{
		Email:    email,
		Password: &password,
	}

	if name != "" {
		req.UserMetadata = map[string]interface{}{
			"name": name,
		}
	}

	resp, err := sc.authClient.AdminCreateUser(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user := resp.User

	return &SupabaseUser{
		ID:          user.ID.String(),
		Email:       user.Email,
		Name:        name,
		Metadata:    user.UserMetadata,
		AppMetadata: user.AppMetadata,
	}, nil
}

// UpdateUser updates a user in Supabase (admin operation)
func (sc *SupabaseClient) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) (*SupabaseUser, error) {
	// Parse user ID as UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	req := types.AdminUpdateUserRequest{
		UserID: userUUID,
	}

	// Map updates to request fields
	if email, ok := updates["email"].(string); ok {
		req.Email = email
	}
	if userMetadata, ok := updates["user_metadata"].(map[string]interface{}); ok {
		req.UserMetadata = userMetadata
	}
	if appMetadata, ok := updates["app_metadata"].(map[string]interface{}); ok {
		req.AppMetadata = appMetadata
	}

	resp, err := sc.authClient.AdminUpdateUser(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	user := resp.User

	// Extract name from user metadata
	name := ""
	if user.UserMetadata != nil {
		if nameVal, ok := user.UserMetadata["name"]; ok {
			if nameStr, ok := nameVal.(string); ok {
				name = nameStr
			}
		}
	}

	return &SupabaseUser{
		ID:          user.ID.String(),
		Email:       user.Email,
		Name:        name,
		Metadata:    user.UserMetadata,
		AppMetadata: user.AppMetadata,
	}, nil
}

// DeleteUser deletes a user from Supabase (admin operation)
func (sc *SupabaseClient) DeleteUser(ctx context.Context, userID string) error {
	// Parse user ID as UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	err = sc.authClient.AdminDeleteUser(types.AdminDeleteUserRequest{
		UserID: userUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}