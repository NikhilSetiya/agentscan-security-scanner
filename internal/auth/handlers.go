package auth

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/api"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
)

// Handlers provides HTTP handlers for authentication
type Handlers struct {
	authService  *Service
	oauthService *OAuthService
	userService  *UserService
	auditLogger  *api.AuditLogger
	config       *config.Config
}

// NewHandlers creates new authentication handlers
func NewHandlers(cfg *config.Config, repos *database.Repositories, auditLogger *api.AuditLogger) *Handlers {
	authService := NewService(cfg, repos)
	userService := NewUserService(repos)
	oauthService := NewOAuthService(cfg, repos, authService)

	return &Handlers{
		authService:  authService,
		oauthService: oauthService,
		userService:  userService,
		auditLogger:  auditLogger,
		config:       cfg,
	}
}

// GetGitHubAuthURL returns the GitHub OAuth authorization URL
func (h *Handlers) GetGitHubAuthURL(c *gin.Context) {
	state, err := h.generateState()
	if err != nil {
		api.InternalErrorResponse(c, "Failed to generate state")
		return
	}

	// TODO: Store state in session/cache for validation
	authURL, err := h.oauthService.GetAuthURL("github", state)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to generate auth URL")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"auth_url": authURL,
		"state":    state,
	})
}

// GetGitLabAuthURL returns the GitLab OAuth authorization URL
func (h *Handlers) GetGitLabAuthURL(c *gin.Context) {
	state, err := h.generateState()
	if err != nil {
		api.InternalErrorResponse(c, "Failed to generate state")
		return
	}

	// TODO: Store state in session/cache for validation
	authURL, err := h.oauthService.GetAuthURL("gitlab", state)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to generate auth URL")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"auth_url": authURL,
		"state":    state,
	})
}

// HandleGitHubCallback handles GitHub OAuth callback
func (h *Handlers) HandleGitHubCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		api.BadRequestResponse(c, "Missing authorization code")
		return
	}

	_ = c.Query("state") // TODO: Validate state parameter for CSRF protection

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	tokenPair, user, err := h.oauthService.HandleCallback(c.Request.Context(), "github", code, ipAddress, userAgent)
	if err != nil {
		// Log failed login
		h.auditLogger.LogAuthEvent(c.Request.Context(), api.AuditEventLoginFailed, nil, false, map[string]interface{}{
			"provider": "github",
			"error":    err.Error(),
		}, c)
		api.InternalErrorResponse(c, "Authentication failed")
		return
	}

	// Log successful login
	h.auditLogger.LogAuthEvent(c.Request.Context(), api.AuditEventLogin, &user.ID, true, map[string]interface{}{
		"provider":   "github",
		"login_type": "oauth",
	}, c)

	response := api.LoginResponse{
		Token:     tokenPair.AccessToken,
		ExpiresAt: tokenPair.ExpiresAt,
		User:      api.ToUserDTO(user),
	}

	api.SuccessResponse(c, response)
}

// HandleGitLabCallback handles GitLab OAuth callback
func (h *Handlers) HandleGitLabCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		api.BadRequestResponse(c, "Missing authorization code")
		return
	}

	_ = c.Query("state") // TODO: Validate state parameter for CSRF protection

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	tokenPair, user, err := h.oauthService.HandleCallback(c.Request.Context(), "gitlab", code, ipAddress, userAgent)
	if err != nil {
		// Log failed login
		h.auditLogger.LogAuthEvent(c.Request.Context(), api.AuditEventLoginFailed, nil, false, map[string]interface{}{
			"provider": "gitlab",
			"error":    err.Error(),
		}, c)
		api.InternalErrorResponse(c, "Authentication failed")
		return
	}

	// Log successful login
	h.auditLogger.LogAuthEvent(c.Request.Context(), api.AuditEventLogin, &user.ID, true, map[string]interface{}{
		"provider":   "gitlab",
		"login_type": "oauth",
	}, c)

	response := api.LoginResponse{
		Token:     tokenPair.AccessToken,
		ExpiresAt: tokenPair.ExpiresAt,
		User:      api.ToUserDTO(user),
	}

	api.SuccessResponse(c, response)
}

// RefreshToken refreshes an access token using a refresh token
func (h *Handlers) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, "Invalid request body")
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken, ipAddress, userAgent)
	if err != nil {
		api.UnauthorizedResponse(c, "Invalid refresh token")
		return
	}

	// Get user for response
	claims, err := h.authService.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to validate new token")
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to get user")
		return
	}

	response := api.LoginResponse{
		Token:     tokenPair.AccessToken,
		ExpiresAt: tokenPair.ExpiresAt,
		User:      api.ToUserDTO(user),
	}

	api.SuccessResponse(c, response)
}

// GetCurrentUser returns information about the current user
func (h *Handlers) GetCurrentUser(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	profile, err := h.userService.GetProfile(c.Request.Context(), user.ID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to get user profile")
		return
	}

	api.SuccessResponse(c, profile)
}

// UpdateProfile updates the current user's profile
func (h *Handlers) UpdateProfile(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, "Invalid request body")
		return
	}

	profile, err := h.userService.UpdateProfile(c.Request.Context(), user.ID, &req)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to update profile")
		return
	}

	api.SuccessResponse(c, profile)
}

// ChangeEmail changes the current user's email
func (h *Handlers) ChangeEmail(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	var req ChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, "Invalid request body")
		return
	}

	if err := h.userService.ChangeEmail(c.Request.Context(), user.ID, &req); err != nil {
		if err.Error() == "email already in use" {
			api.BadRequestResponse(c, "Email already in use")
			return
		}
		api.InternalErrorResponse(c, "Failed to change email")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"message": "Email changed successfully",
	})
}

// Logout handles user logout
func (h *Handlers) Logout(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if exists {
		// Log successful logout
		h.auditLogger.LogAuthEvent(c.Request.Context(), api.AuditEventLogout, &userID, true, map[string]interface{}{
			"logout_type": "manual",
		}, c)
	}

	// For JWT tokens, logout is handled client-side by removing the token
	// If we had refresh tokens in the request, we would revoke them here
	api.SuccessResponse(c, map[string]string{
		"message": "Logged out successfully",
	})
}

// RevokeToken revokes a refresh token
func (h *Handlers) RevokeToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, "Invalid request body")
		return
	}

	if err := h.authService.RevokeToken(c.Request.Context(), req.RefreshToken); err != nil {
		api.BadRequestResponse(c, "Invalid refresh token")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"message": "Token revoked successfully",
	})
}

// RevokeAllTokens revokes all refresh tokens for the current user
func (h *Handlers) RevokeAllTokens(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	if err := h.authService.RevokeAllUserTokens(c.Request.Context(), userID); err != nil {
		api.InternalErrorResponse(c, "Failed to revoke tokens")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"message": "All tokens revoked successfully",
	})
}

// GetSessions returns all active sessions for the current user
func (h *Handlers) GetSessions(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	sessions, err := h.authService.GetUserSessions(c.Request.Context(), userID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to get sessions")
		return
	}

	api.SuccessResponse(c, sessions)
}

// DeleteAccount deletes the current user's account
func (h *Handlers) DeleteAccount(c *gin.Context) {
	user, exists := GetCurrentUser(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	// TODO: Add confirmation mechanism (email verification, password confirmation, etc.)
	
	if err := h.userService.DeleteAccount(c.Request.Context(), user.ID); err != nil {
		api.InternalErrorResponse(c, "Failed to delete account")
		return
	}

	// Log account deletion
	h.auditLogger.LogAuthEvent(c.Request.Context(), api.AuditEventUserDeleted, &user.ID, true, map[string]interface{}{
		"deletion_type": "self_service",
	}, c)

	api.SuccessResponse(c, map[string]string{
		"message": "Account deleted successfully",
	})
}

// generateState generates a random state string for OAuth
func (h *Handlers) generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HealthCheck returns the health status of the auth service
func (h *Handlers) HealthCheck(c *gin.Context) {
	// TODO: Add actual health checks (database connectivity, etc.)
	api.SuccessResponse(c, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"services": map[string]string{
			"auth":  "ok",
			"oauth": "ok",
		},
	})
}