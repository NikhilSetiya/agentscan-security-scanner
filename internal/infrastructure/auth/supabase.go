package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
)

// SupabaseClient handles Supabase authentication operations
type SupabaseClient struct {
	config     *config.SupabaseConfig
	httpClient *http.Client
	jwtSecret  string
}

// SupabaseUser represents a user from Supabase
type SupabaseUser struct {
	ID                 string                 `json:"id"`
	Email              string                 `json:"email"`
	EmailConfirmed     bool                   `json:"email_confirmed_at"`
	Phone              string                 `json:"phone"`
	PhoneConfirmed     bool                   `json:"phone_confirmed_at"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	LastSignInAt       *time.Time             `json:"last_sign_in_at"`
	Role               string                 `json:"role"`
	UserMetadata       map[string]interface{} `json:"user_metadata"`
	AppMetadata        map[string]interface{} `json:"app_metadata"`
	Identities         []SupabaseIdentity     `json:"identities"`
	Aud                string                 `json:"aud"`
	ConfirmationSentAt *time.Time             `json:"confirmation_sent_at"`
	RecoverySentAt     *time.Time             `json:"recovery_sent_at"`
	EmailChangeSentAt  *time.Time             `json:"email_change_sent_at"`
	NewEmail           string                 `json:"new_email"`
}

// SupabaseIdentity represents a user identity from Supabase
type SupabaseIdentity struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	IdentityData map[string]interface{} `json:"identity_data"`
	Provider     string                 `json:"provider"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// SupabaseSession represents a Supabase session
type SupabaseSession struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
	ExpiresAt    int64        `json:"expires_at"`
	TokenType    string       `json:"token_type"`
	User         SupabaseUser `json:"user"`
}

// SupabaseAuthResponse represents the response from Supabase auth endpoints
type SupabaseAuthResponse struct {
	User    *SupabaseUser    `json:"user,omitempty"`
	Session *SupabaseSession `json:"session,omitempty"`
	Error   *SupabaseError   `json:"error,omitempty"`
}

// SupabaseError represents an error from Supabase
type SupabaseError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// JWTClaims represents the claims in a Supabase JWT token
type JWTClaims struct {
	Aud                  string                   `json:"aud"`
	Exp                  int64                    `json:"exp"`
	Iat                  int64                    `json:"iat"`
	Iss                  string                   `json:"iss"`
	Sub                  string                   `json:"sub"`
	Email                string                   `json:"email"`
	Phone                string                   `json:"phone"`
	AppMetadata          map[string]interface{}   `json:"app_metadata"`
	UserMetadata         map[string]interface{}   `json:"user_metadata"`
	Role                 string                   `json:"role"`
	AAL                  string                   `json:"aal"`
	AMR                  []map[string]interface{} `json:"amr"`
	SessionID            string                   `json:"session_id"`
	jwt.RegisteredClaims
}

// NewSupabaseClient creates a new Supabase client
func NewSupabaseClient(cfg *config.SupabaseConfig, jwtSecret string) *SupabaseClient {
	return &SupabaseClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		jwtSecret: jwtSecret,
	}
}

// ValidateJWT validates a Supabase JWT token
func (sc *SupabaseClient) ValidateJWT(tokenString string) (*JWTClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(sc.jwtSecret), nil
	})

	if err != nil {
		return nil, errors.NewAuthenticationError("invalid JWT token").WithCause(err)
	}

	// Validate the token and extract claims
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		// Additional validation
		if claims.Aud != "authenticated" {
			return nil, errors.NewAuthenticationError("invalid token audience")
		}

		if claims.Iss != sc.config.URL+"/auth/v1" {
			return nil, errors.NewAuthenticationError("invalid token issuer")
		}

		// Check expiration
		if time.Now().Unix() > claims.Exp {
			return nil, errors.NewAuthenticationError("token expired")
		}

		return claims, nil
	}

	return nil, errors.NewAuthenticationError("invalid token claims")
}

// GetUser retrieves user information from Supabase
func (sc *SupabaseClient) GetUser(ctx context.Context, accessToken string) (*SupabaseUser, error) {
	url := fmt.Sprintf("%s/auth/v1/user", sc.config.URL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request").WithCause(err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", sc.config.AnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewInternalError("failed to get user from Supabase").WithCause(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewAuthenticationError("failed to get user from Supabase")
	}

	var user SupabaseUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, errors.NewInternalError("failed to decode user response").WithCause(err)
	}

	return &user, nil
}

// RefreshToken refreshes an access token using a refresh token
func (sc *SupabaseClient) RefreshToken(ctx context.Context, refreshToken string) (*SupabaseSession, error) {
	url := fmt.Sprintf("%s/auth/v1/token?grant_type=refresh_token", sc.config.URL)

	payload := map[string]string{
		"refresh_token": refreshToken,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.NewInternalError("failed to marshal refresh request").WithCause(err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, errors.NewInternalError("failed to create refresh request").WithCause(err)
	}

	// Set headers
	req.Header.Set("apikey", sc.config.AnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewInternalError("failed to refresh token").WithCause(err)
	}
	defer resp.Body.Close()

	var authResp SupabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, errors.NewInternalError("failed to decode refresh response").WithCause(err)
	}

	if authResp.Error != nil {
		return nil, errors.NewAuthenticationError(authResp.Error.Message)
	}

	if authResp.Session == nil {
		return nil, errors.NewAuthenticationError("no session in refresh response")
	}

	return authResp.Session, nil
}

// GetUserRole extracts the user role from JWT claims or user metadata
func (sc *SupabaseClient) GetUserRole(claims *JWTClaims) string {
	// Check app_metadata first (set by admin)
	if claims.AppMetadata != nil {
		if role, ok := claims.AppMetadata["role"].(string); ok && role != "" {
			return role
		}
	}

	// Check user_metadata (set by user)
	if claims.UserMetadata != nil {
		if role, ok := claims.UserMetadata["role"].(string); ok && role != "" {
			return role
		}
	}

	// Check direct role claim
	if claims.Role != "" {
		return claims.Role
	}

	// Default role
	return "user"
}

// IsTokenExpired checks if a JWT token is expired
func (sc *SupabaseClient) IsTokenExpired(tokenString string) bool {
	claims, err := sc.ValidateJWT(tokenString)
	if err != nil {
		return true
	}

	return time.Now().Unix() > claims.Exp
}

// SignOut signs out a user by invalidating their session
func (sc *SupabaseClient) SignOut(ctx context.Context, accessToken string) error {
	url := fmt.Sprintf("%s/auth/v1/logout", sc.config.URL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return errors.NewInternalError("failed to create signout request").WithCause(err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", sc.config.AnonKey)

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return errors.NewInternalError("failed to sign out user").WithCause(err)
	}
	defer resp.Body.Close()

	// Supabase returns 204 No Content on successful logout
	if resp.StatusCode != http.StatusNoContent {
		return errors.NewInternalError("failed to sign out from Supabase")
	}

	return nil
}

// UpdateUser updates user metadata in Supabase
func (sc *SupabaseClient) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) (*SupabaseUser, error) {
	url := fmt.Sprintf("%s/auth/v1/admin/users/%s", sc.config.URL, userID)

	payloadBytes, err := json.Marshal(updates)
	if err != nil {
		return nil, errors.NewInternalError("failed to marshal update user request").WithCause(err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, errors.NewInternalError("failed to create update user request").WithCause(err)
	}

	// Set headers (requires service role key for admin operations)
	req.Header.Set("Authorization", "Bearer "+sc.config.ServiceRoleKey)
	req.Header.Set("apikey", sc.config.ServiceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewInternalError("failed to update user").WithCause(err)
	}
	defer resp.Body.Close()

	var authResp SupabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, errors.NewInternalError("failed to decode update user response").WithCause(err)
	}

	if authResp.Error != nil {
		return nil, errors.NewValidationError(authResp.Error.Message)
	}

	if authResp.User == nil {
		return nil, errors.NewInternalError("no user in update response")
	}

	return authResp.User, nil
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.NewAuthenticationError("authorization header is required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.NewAuthenticationError("invalid authorization header format")
	}

	return parts[1], nil
}