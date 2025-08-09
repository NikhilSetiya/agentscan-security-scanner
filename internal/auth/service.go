package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/types"
)

// Service provides authentication functionality
type Service struct {
	config *config.Config
	repos  *database.Repositories
}

// NewService creates a new authentication service
func NewService(cfg *config.Config, repos *database.Repositories) *Service {
	return &Service{
		config: cfg,
		repos:  repos,
	}
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Name   string    `json:"name"`
	jwt.RegisteredClaims
}

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// UserSession represents a user session
type UserSession struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// GenerateTokenPair generates a new JWT access token and refresh token
func (s *Service) GenerateTokenPair(ctx context.Context, user *types.User, ipAddress, userAgent string) (*TokenPair, error) {
	// Generate access token
	accessToken, expiresAt, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in database
	session := &UserSession{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour), // 30 days
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.storeSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *Service) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (*TokenPair, error) {
	// Validate refresh token
	session, err := s.validateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user
	user, err := s.repos.Users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Generate new token pair
	tokenPair, err := s.GenerateTokenPair(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token pair: %w", err)
	}

	// Invalidate old refresh token
	if err := s.invalidateSession(ctx, session.ID); err != nil {
		// Log error but don't fail the refresh
		// TODO: Add proper logging
	}

	return tokenPair, nil
}

// ValidateAccessToken validates a JWT access token and returns the claims
func (s *Service) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}

// RevokeToken revokes a refresh token
func (s *Service) RevokeToken(ctx context.Context, refreshToken string) error {
	session, err := s.validateRefreshToken(ctx, refreshToken)
	if err != nil {
		return fmt.Errorf("invalid refresh token: %w", err)
	}

	return s.invalidateSession(ctx, session.ID)
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (s *Service) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return s.invalidateAllUserSessions(ctx, userID)
}

// CleanupExpiredSessions removes expired sessions from the database
func (s *Service) CleanupExpiredSessions(ctx context.Context) error {
	return s.cleanupExpiredSessions(ctx)
}

// GetUserSessions returns all active sessions for a user
func (s *Service) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*UserSession, error) {
	return s.getUserSessions(ctx, userID)
}

// generateAccessToken generates a JWT access token
func (s *Service) generateAccessToken(user *types.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.config.Auth.JWTExpiration)

	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "agentscan",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Auth.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// generateRefreshToken generates a cryptographically secure refresh token
func (s *Service) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// storeSession stores a user session in the database
func (s *Service) storeSession(ctx context.Context, session *UserSession) error {
	// TODO: Implement session storage in database
	// For now, we'll use a simple in-memory approach or Redis
	return nil
}

// validateRefreshToken validates a refresh token and returns the session
func (s *Service) validateRefreshToken(ctx context.Context, refreshToken string) (*UserSession, error) {
	// TODO: Implement refresh token validation from database
	// For now, return a mock session
	return &UserSession{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// invalidateSession invalidates a session by ID
func (s *Service) invalidateSession(ctx context.Context, sessionID uuid.UUID) error {
	// TODO: Implement session invalidation in database
	return nil
}

// invalidateAllUserSessions invalidates all sessions for a user
func (s *Service) invalidateAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	// TODO: Implement bulk session invalidation in database
	return nil
}

// cleanupExpiredSessions removes expired sessions from the database
func (s *Service) cleanupExpiredSessions(ctx context.Context) error {
	// TODO: Implement expired session cleanup in database
	return nil
}

// getUserSessions returns all active sessions for a user
func (s *Service) getUserSessions(ctx context.Context, userID uuid.UUID) ([]*UserSession, error) {
	// TODO: Implement session retrieval from database
	return []*UserSession{}, nil
}