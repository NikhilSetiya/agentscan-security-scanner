package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/types"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	config *config.Config
	repos  *database.Repositories
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(cfg *config.Config, repos *database.Repositories) *AuthHandler {
	return &AuthHandler{
		config: cfg,
		repos:  repos,
	}
}

// GitHubUser represents a GitHub user from the API
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubTokenResponse represents GitHub OAuth token response
type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// LoginWithGitHub handles GitHub OAuth login
func (h *AuthHandler) LoginWithGitHub(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		BadRequestResponse(c, "Missing authorization code")
		return
	}

	_ = c.Query("state") // TODO: Validate state parameter for CSRF protection

	// Exchange code for access token
	accessToken, err := h.exchangeGitHubCode(code)
	if err != nil {
		InternalErrorResponse(c, "Failed to exchange authorization code")
		return
	}

	// Get user info from GitHub
	githubUser, err := h.getGitHubUser(accessToken)
	if err != nil {
		InternalErrorResponse(c, "Failed to get user information from GitHub")
		return
	}

	// Find or create user in database
	user, err := h.findOrCreateUser(c.Request.Context(), githubUser)
	if err != nil {
		InternalErrorResponse(c, "Failed to create or update user")
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.generateJWTToken(user)
	if err != nil {
		InternalErrorResponse(c, "Failed to generate authentication token")
		return
	}

	response := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      ToUserDTO(user),
	}

	SuccessResponse(c, response)
}

// GetAuthURL returns the GitHub OAuth authorization URL
func (h *AuthHandler) GetAuthURL(c *gin.Context) {
	state := uuid.New().String() // Generate random state for CSRF protection
	// TODO: Store state in session/cache for validation

	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		h.config.Auth.GitHubClientID,
		url.QueryEscape("http://localhost:3000/auth/callback"), // TODO: Make configurable
		url.QueryEscape("user:email"),
		state,
	)

	SuccessResponse(c, map[string]string{
		"auth_url": authURL,
		"state":    state,
	})
}

// RefreshToken refreshes an existing JWT token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		UnauthorizedResponse(c, "User not found in context")
		return
	}

	// Generate new JWT token
	token, expiresAt, err := h.generateJWTToken(user)
	if err != nil {
		InternalErrorResponse(c, "Failed to generate authentication token")
		return
	}

	response := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      ToUserDTO(user),
	}

	SuccessResponse(c, response)
}

// GetCurrentUserInfo returns information about the current user
func (h *AuthHandler) GetCurrentUserInfo(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		UnauthorizedResponse(c, "User not found in context")
		return
	}

	SuccessResponse(c, ToUserDTO(user))
}

// Logout handles user logout (client-side token invalidation)
func (h *AuthHandler) Logout(c *gin.Context) {
	// Since we're using stateless JWT tokens, logout is handled client-side
	// by removing the token from storage
	SuccessResponse(c, map[string]string{
		"message": "Logged out successfully",
	})
}

// exchangeGitHubCode exchanges authorization code for access token
func (h *AuthHandler) exchangeGitHubCode(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", h.config.Auth.GitHubClientID)
	data.Set("client_secret", h.config.Auth.GitHubSecret)
	data.Set("code", code)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp GitHubTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// getGitHubUser gets user information from GitHub API
func (h *AuthHandler) getGitHubUser(accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user GitHubUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	// Get user email if not public
	if user.Email == "" {
		email, err := h.getGitHubUserEmail(accessToken)
		if err == nil {
			user.Email = email
		}
	}

	return &user, nil
}

// getGitHubUserEmail gets user's primary email from GitHub API
func (h *AuthHandler) getGitHubUserEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}

	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}

	// Find primary email
	for _, email := range emails {
		if email.Primary {
			return email.Email, nil
		}
	}

	// Return first email if no primary found
	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("no email found")
}

// findOrCreateUser finds an existing user or creates a new one
func (h *AuthHandler) findOrCreateUser(ctx context.Context, githubUser *GitHubUser) (*types.User, error) {
	// Try to find user by email first
	user, err := h.repos.Users.GetByEmail(ctx, githubUser.Email)
	if err == nil {
		// Update GitHub ID and other info if user exists
		githubID := githubUser.ID
		user.GitHubID = &githubID
		user.Name = githubUser.Name
		user.AvatarURL = githubUser.AvatarURL
		
		if err := h.repos.Users.Update(ctx, user); err != nil {
			return nil, err
		}
		
		return user, nil
	}

	// Create new user if not found
	user = &types.User{
		ID:        uuid.New(),
		Email:     githubUser.Email,
		Name:      githubUser.Name,
		AvatarURL: githubUser.AvatarURL,
		GitHubID:  &githubUser.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.repos.Users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// generateJWTToken generates a JWT token for the user
func (h *AuthHandler) generateJWTToken(user *types.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(h.config.Auth.JWTExpiration)

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
	tokenString, err := token.SignedString([]byte(h.config.Auth.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}