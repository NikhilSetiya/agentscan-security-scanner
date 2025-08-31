package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/commands"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/queries"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/services"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	appService *services.ApplicationService
}

// NewUserHandler creates a new user handler
func NewUserHandler(appService *services.ApplicationService) *UserHandler {
	return &UserHandler{
		appService: appService,
	}
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email string             `json:"email" binding:"required,email"`
	Name  string             `json:"name" binding:"required,min=1,max=255"`
	Role  entities.UserRole  `json:"role" binding:"required"`
}

// UpdateUserProfileRequest represents the request to update user profile
type UpdateUserProfileRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=255"`
	AvatarURL string `json:"avatar_url" binding:"omitempty,url"`
}

// UpdateUserRoleRequest represents the request to update user role
type UpdateUserRoleRequest struct {
	Role entities.UserRole `json:"role" binding:"required"`
}

// CreateUser handles POST /users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.CreateUserCommand{
		Email: req.Email,
		Name:  req.Name,
		Role:  req.Role,
	}

	user, err := h.appService.CreateUser(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
	})
}

// GetUser handles GET /users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	query := queries.GetUserByIDQuery{
		UserID: userID,
	}

	user, err := h.appService.GetUserByID(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// GetUserByEmail handles GET /users/email/:email
func (h *UserHandler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email is required",
		})
		return
	}

	query := queries.GetUserByEmailQuery{
		Email: email,
	}

	user, err := h.appService.GetUserByEmail(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// ListUsers handles GET /users
func (h *UserHandler) ListUsers(c *gin.Context) {
	orgID, err := parseUUID(c.Query("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid organization ID",
		})
		return
	}

	// Parse pagination parameters
	page, pageSize := parsePagination(c)
	
	// Parse filter parameters
	filter := repositories.Filter{
		Search:    c.Query("search"),
		Status:    c.Query("status"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	pagination := repositories.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	query := queries.ListUsersQuery{
		OrganizationID: orgID,
		Filter:         filter,
		Pagination:     pagination,
	}

	users, total, err := h.appService.ListUsers(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
		"meta": gin.H{
			"pagination": gin.H{
				"page":       page,
				"page_size":  pageSize,
				"total":      total,
				"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
			},
		},
	})
}

// UpdateUserProfile handles PUT /users/:id/profile
func (h *UserHandler) UpdateUserProfile(c *gin.Context) {
	userID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var req UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.UpdateUserProfileCommand{
		UserID:    userID,
		Name:      req.Name,
		AvatarURL: req.AvatarURL,
	}

	err = h.appService.UpdateUserProfile(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User profile updated successfully",
	})
}

// UpdateUserRole handles PUT /users/:id/role
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	targetUserID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Get admin user ID from context (set by auth middleware)
	adminUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	adminID, ok := adminUserID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user context",
		})
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	cmd := commands.UpdateUserRoleCommand{
		AdminUserID:  adminID,
		TargetUserID: targetUserID,
		NewRole:      req.Role,
	}

	err = h.appService.UpdateUserRole(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User role updated successfully",
	})
}

// DeactivateUser handles POST /users/:id/deactivate
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	targetUserID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Get admin user ID from context
	adminUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	adminID, ok := adminUserID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user context",
		})
		return
	}

	cmd := commands.DeactivateUserCommand{
		AdminUserID:  adminID,
		TargetUserID: targetUserID,
	}

	err = h.appService.DeactivateUser(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deactivated successfully",
	})
}

// ActivateUser handles POST /users/:id/activate
func (h *UserHandler) ActivateUser(c *gin.Context) {
	targetUserID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Get admin user ID from context
	adminUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	adminID, ok := adminUserID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user context",
		})
		return
	}

	cmd := commands.ActivateUserCommand{
		AdminUserID:  adminID,
		TargetUserID: targetUserID,
	}

	err = h.appService.ActivateUser(c.Request.Context(), cmd)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User activated successfully",
	})
}