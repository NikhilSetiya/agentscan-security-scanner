package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// OAuthProvider represents an OAuth provider
type OAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCode(code string) (string, error)
	GetUser(accessToken string) (*OAuthUser, error)
}

// OAuthUser represents a user from an OAuth provider
type OAuthUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Provider  string `json:"provider"`
}

// GitHubProvider implements OAuth for GitHub
type GitHubProvider struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

// NewGitHubProvider creates a new GitHub OAuth provider
func NewGitHubProvider(cfg *config.Config) *GitHubProvider {
	return &GitHubProvider{
		clientID:     cfg.Auth.GitHubClientID,
		clientSecret: cfg.Auth.GitHubSecret,
		redirectURI:  "http://localhost:3000/auth/callback", // TODO: Make configurable
	}
}

// GetAuthURL returns the GitHub OAuth authorization URL
func (p *GitHubProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		p.clientID,
		url.QueryEscape(p.redirectURI),
		url.QueryEscape("user:email"),
		state,
	)
}

// ExchangeCode exchanges authorization code for access token
func (p *GitHubProvider) ExchangeCode(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", code)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("GitHub OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return tokenResp.AccessToken, nil
}

// GetUser gets user information from GitHub API
func (p *GitHubProvider) GetUser(accessToken string) (*OAuthUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var githubUser struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Get user email if not public
	if githubUser.Email == "" {
		email, err := p.getUserEmail(accessToken)
		if err == nil {
			githubUser.Email = email
		}
	}

	return &OAuthUser{
		ID:        fmt.Sprintf("%d", githubUser.ID),
		Email:     githubUser.Email,
		Name:      githubUser.Name,
		Username:  githubUser.Login,
		AvatarURL: githubUser.AvatarURL,
		Provider:  "github",
	}, nil
}

// getUserEmail gets user's primary email from GitHub API
func (p *GitHubProvider) getUserEmail(accessToken string) (string, error) {
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

// GitLabProvider implements OAuth for GitLab
type GitLabProvider struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

// NewGitLabProvider creates a new GitLab OAuth provider
func NewGitLabProvider(cfg *config.Config) *GitLabProvider {
	return &GitLabProvider{
		clientID:     cfg.Auth.GitLabClientID,
		clientSecret: cfg.Auth.GitLabSecret,
		redirectURI:  "http://localhost:3000/auth/gitlab/callback", // TODO: Make configurable
	}
}

// GetAuthURL returns the GitLab OAuth authorization URL
func (p *GitLabProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://gitlab.com/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		p.clientID,
		url.QueryEscape(p.redirectURI),
		url.QueryEscape("read_user"),
		state,
	)
}

// ExchangeCode exchanges authorization code for access token
func (p *GitLabProvider) ExchangeCode(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", p.redirectURI)

	req, err := http.NewRequest("POST", "https://gitlab.com/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("GitLab OAuth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return tokenResp.AccessToken, nil
}

// GetUser gets user information from GitLab API
func (p *GitLabProvider) GetUser(accessToken string) (*OAuthUser, error) {
	req, err := http.NewRequest("GET", "https://gitlab.com/api/v4/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var gitlabUser struct {
		ID        int    `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.Unmarshal(body, &gitlabUser); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &OAuthUser{
		ID:        fmt.Sprintf("%d", gitlabUser.ID),
		Email:     gitlabUser.Email,
		Name:      gitlabUser.Name,
		Username:  gitlabUser.Username,
		AvatarURL: gitlabUser.AvatarURL,
		Provider:  "gitlab",
	}, nil
}

// OAuthService manages OAuth providers and user authentication
type OAuthService struct {
	config    *config.Config
	repos     *database.Repositories
	authSvc   *Service
	providers map[string]OAuthProvider
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(cfg *config.Config, repos *database.Repositories, authSvc *Service) *OAuthService {
	providers := make(map[string]OAuthProvider)
	
	if cfg.Auth.GitHubClientID != "" && cfg.Auth.GitHubSecret != "" {
		providers["github"] = NewGitHubProvider(cfg)
	}
	
	if cfg.Auth.GitLabClientID != "" && cfg.Auth.GitLabSecret != "" {
		providers["gitlab"] = NewGitLabProvider(cfg)
	}

	return &OAuthService{
		config:    cfg,
		repos:     repos,
		authSvc:   authSvc,
		providers: providers,
	}
}

// GetAuthURL returns the OAuth authorization URL for a provider
func (s *OAuthService) GetAuthURL(provider, state string) (string, error) {
	p, exists := s.providers[provider]
	if !exists {
		return "", fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	return p.GetAuthURL(state), nil
}

// HandleCallback handles OAuth callback and returns a token pair
func (s *OAuthService) HandleCallback(ctx context.Context, provider, code, ipAddress, userAgent string) (*TokenPair, *types.User, error) {
	p, exists := s.providers[provider]
	if !exists {
		return nil, nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}

	// Exchange code for access token
	accessToken, err := p.ExchangeCode(code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from provider
	oauthUser, err := p.GetUser(accessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Find or create user in database
	user, err := s.findOrCreateUser(ctx, oauthUser)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// Generate token pair
	tokenPair, err := s.authSvc.GenerateTokenPair(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	return tokenPair, user, nil
}

// findOrCreateUser finds an existing user or creates a new one
func (s *OAuthService) findOrCreateUser(ctx context.Context, oauthUser *OAuthUser) (*types.User, error) {
	// Try to find user by email first
	user, err := s.repos.Users.GetByEmail(ctx, oauthUser.Email)
	if err == nil {
		// Update provider-specific info if user exists
		updated := false
		
		switch oauthUser.Provider {
		case "github":
			if githubID, err := parseProviderID(oauthUser.ID); err == nil {
				user.GitHubID = &githubID
				updated = true
			}
		case "gitlab":
			if gitlabID, err := parseProviderID(oauthUser.ID); err == nil {
				user.GitLabID = &gitlabID
				updated = true
			}
		}
		
		// Update other fields
		if user.Name != oauthUser.Name {
			user.Name = oauthUser.Name
			updated = true
		}
		if user.AvatarURL != oauthUser.AvatarURL {
			user.AvatarURL = oauthUser.AvatarURL
			updated = true
		}
		
		if updated {
			user.UpdatedAt = time.Now()
			if err := s.repos.Users.Update(ctx, user); err != nil {
				return nil, fmt.Errorf("failed to update user: %w", err)
			}
		}
		
		return user, nil
	}

	// Create new user if not found
	user = &types.User{
		ID:        uuid.New(),
		Email:     oauthUser.Email,
		Name:      oauthUser.Name,
		AvatarURL: oauthUser.AvatarURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set provider-specific ID
	switch oauthUser.Provider {
	case "github":
		if githubID, err := parseProviderID(oauthUser.ID); err == nil {
			user.GitHubID = &githubID
		}
	case "gitlab":
		if gitlabID, err := parseProviderID(oauthUser.ID); err == nil {
			user.GitLabID = &gitlabID
		}
	}

	if err := s.repos.Users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// parseProviderID parses a provider ID string to int
func parseProviderID(idStr string) (int, error) {
	var id int
	n, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, fmt.Errorf("invalid provider ID format")
	}
	// Check if the entire string was consumed (no extra characters)
	var remainder string
	if _, err := fmt.Sscanf(idStr, "%d%s", &id, &remainder); err == nil && remainder != "" {
		return 0, fmt.Errorf("invalid provider ID format")
	}
	return id, nil
}