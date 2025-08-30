package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/internal/adapters/http/validators"
	"github.com/agentscan/agentscan/internal/api"
	"github.com/agentscan/agentscan/internal/application/dto"
	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/internal/shared/logging"
	"github.com/agentscan/agentscan/pkg/errors"
)

// BaseHandler provides common functionality for all handlers
type BaseHandler struct {
	repos     repositories.Repositories
	validator *validators.InputValidator
	logger    *logging.Logger
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(repos repositories.Repositories) *BaseHandler {
	return &BaseHandler{
		repos:     repos,
		validator: validators.NewInputValidator(),
		logger:    logging.GetLogger(),
	}
}

// Common response helpers

// Success sends a successful response
func (h *BaseHandler) Success(c *gin.Context, data interface{}) {
	api.SuccessResponse(c, data)
}

// SuccessWithMeta sends a successful response with metadata
func (h *BaseHandler) SuccessWithMeta(c *gin.Context, data interface{}, meta *api.Meta) {
	api.SuccessResponseWithMeta(c, data, meta)
}

// Created sends a 201 Created response
func (h *BaseHandler) Created(c *gin.Context, data interface{}) {
	api.CreatedResponse(c, data)
}

// Error sends an error response
func (h *BaseHandler) Error(c *gin.Context, err error) {
	api.ErrorResponseFromError(c, err)
}

// BadRequest sends a 400 Bad Request response
func (h *BaseHandler) BadRequest(c *gin.Context, message string) {
	api.BadRequestResponse(c, message)
}

// NotFound sends a 404 Not Found response
func (h *BaseHandler) NotFound(c *gin.Context, message string) {
	api.NotFoundResponse(c, message)
}

// Unauthorized sends a 401 Unauthorized response
func (h *BaseHandler) Unauthorized(c *gin.Context, message string) {
	api.UnauthorizedResponse(c, message)
}

// Forbidden sends a 403 Forbidden response
func (h *BaseHandler) Forbidden(c *gin.Context, message string) {
	api.ForbiddenResponse(c, message)
}

// Conflict sends a 409 Conflict response
func (h *BaseHandler) Conflict(c *gin.Context, message string) {
	api.ConflictResponse(c, message)
}

// ValidationError sends a validation error response
func (h *BaseHandler) ValidationError(c *gin.Context, message string, details map[string]interface{}) {
	api.ValidationErrorResponse(c, message, details)
}

// Common parameter extraction helpers

// GetUUIDParam extracts and validates a UUID parameter
func (h *BaseHandler) GetUUIDParam(c *gin.Context, paramName string) (uuid.UUID, error) {
	paramValue := c.Param(paramName)
	if paramValue == "" {
		return uuid.Nil, errors.NewValidationError("missing required parameter").WithDetails(map[string]interface{}{
			"parameter": paramName,
		})
	}

	parsedUUID, err := uuid.Parse(paramValue)
	if err != nil {
		return uuid.Nil, errors.NewValidationError("invalid UUID format").WithDetails(map[string]interface{}{
			"parameter": paramName,
			"value":     paramValue,
		})
	}

	return parsedUUID, nil
}

// GetIntQuery extracts and validates an integer query parameter
func (h *BaseHandler) GetIntQuery(c *gin.Context, paramName string, defaultValue int) (int, error) {
	paramValue := c.Query(paramName)
	if paramValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(paramValue)
	if err != nil {
		return 0, errors.NewValidationError("invalid integer parameter").WithDetails(map[string]interface{}{
			"parameter": paramName,
			"value":     paramValue,
		})
	}

	return value, nil
}

// GetPaginationParams extracts pagination parameters
func (h *BaseHandler) GetPaginationParams(c *gin.Context) (int, int, error) {
	page, err := h.GetIntQuery(c, "page", 1)
	if err != nil {
		return 0, 0, err
	}

	if page < 1 {
		page = 1
	}

	pageSize, err := h.GetIntQuery(c, "limit", 20)
	if err != nil {
		return 0, 0, err
	}

	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	return pageSize, offset, nil
}

// GetUserContext extracts user information from context
func (h *BaseHandler) GetUserContext(c *gin.Context) (uuid.UUID, string, string, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, "", "", errors.NewUnauthorizedError("user not authenticated")
	}

	email, exists := c.Get("user_email")
	if !exists {
		return uuid.Nil, "", "", errors.NewUnauthorizedError("user email not found")
	}

	role, exists := c.Get("user_role")
	if !exists {
		return uuid.Nil, "", "", errors.NewUnauthorizedError("user role not found")
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, "", "", errors.NewInternalError("invalid user ID format")
	}

	userEmail, ok := email.(string)
	if !ok {
		return uuid.Nil, "", "", errors.NewInternalError("invalid email format")
	}

	userRole, ok := role.(string)
	if !ok {
		return uuid.Nil, "", "", errors.NewInternalError("invalid role format")
	}

	return uid, userEmail, userRole, nil
}

// GetOrganizationContext extracts organization information from context
func (h *BaseHandler) GetOrganizationContext(c *gin.Context) (uuid.UUID, error) {
	orgID, exists := c.Get("org_id")
	if !exists {
		return uuid.Nil, errors.NewForbiddenError("organization access required")
	}

	oid, ok := orgID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.NewForbiddenError("invalid organization ID")
	}

	return oid, nil
}

// Validation helpers

// ValidateAndBind validates and binds JSON request body
func (h *BaseHandler) ValidateAndBind(c *gin.Context, target interface{}) error {
	// Bind JSON to target struct
	if err := c.ShouldBindJSON(target); err != nil {
		return errors.NewValidationError("invalid JSON format").WithCause(err)
	}

	// Validate and sanitize the input
	if err := h.validator.ValidateAndSanitize(target); err != nil {
		return err
	}

	return nil
}

// Logging helpers

// LogAction logs an action with context
func (h *BaseHandler) LogAction(c *gin.Context, action, resource string, details map[string]interface{}) {
	h.logger.LogAuditEvent(c, action, resource, details)
}

// LogError logs an error with context
func (h *BaseHandler) LogError(c *gin.Context, err error, message string, details map[string]interface{}) {
	h.logger.LogError(c, err, message, details)
}

// Common CRUD operations

// HandleCreate provides a generic create operation
func (h *BaseHandler) HandleCreate[T any](
	c *gin.Context,
	request interface{},
	createFunc func(context.Context, interface{}) (*T, error),
	resourceName string,
) {
	// Validate request
	if err := h.ValidateAndBind(c, request); err != nil {
		h.Error(c, err)
		return
	}

	// Get user context
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Create entity
	entity, err := createFunc(c.Request.Context(), request)
	if err != nil {
		h.LogError(c, err, fmt.Sprintf("Failed to create %s", resourceName), map[string]interface{}{
			"user_id": userID,
			"request": request,
		})
		h.Error(c, err)
		return
	}

	// Log success
	h.LogAction(c, "create", resourceName, map[string]interface{}{
		"user_id": userID,
		"entity":  entity,
	})

	h.Created(c, entity)
}

// HandleGet provides a generic get operation
func (h *BaseHandler) HandleGet[T any](
	c *gin.Context,
	getFunc func(context.Context, uuid.UUID) (*T, error),
	resourceName string,
) {
	// Get ID parameter
	id, err := h.GetUUIDParam(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get entity
	entity, err := getFunc(c.Request.Context(), id)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, entity)
}

// HandleList provides a generic list operation
func (h *BaseHandler) HandleList[T any](
	c *gin.Context,
	listFunc func(context.Context, map[string]interface{}, int, int) ([]*T, int, error),
	filterFunc func(*gin.Context) map[string]interface{},
	resourceName string,
) {
	// Get pagination parameters
	limit, offset, err := h.GetPaginationParams(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get filters
	filters := filterFunc(c)

	// List entities
	entities, total, err := listFunc(c.Request.Context(), filters, limit, offset)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Calculate pagination metadata
	page := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit

	meta := &api.Meta{
		Pagination: &api.Pagination{
			Page:       page,
			PageSize:   limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	}

	h.SuccessWithMeta(c, entities, meta)
}

// HandleUpdate provides a generic update operation
func (h *BaseHandler) HandleUpdate[T any](
	c *gin.Context,
	request interface{},
	updateFunc func(context.Context, uuid.UUID, interface{}) (*T, error),
	resourceName string,
) {
	// Get ID parameter
	id, err := h.GetUUIDParam(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Validate request
	if err := h.ValidateAndBind(c, request); err != nil {
		h.Error(c, err)
		return
	}

	// Get user context
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Update entity
	entity, err := updateFunc(c.Request.Context(), id, request)
	if err != nil {
		h.LogError(c, err, fmt.Sprintf("Failed to update %s", resourceName), map[string]interface{}{
			"user_id": userID,
			"id":      id,
			"request": request,
		})
		h.Error(c, err)
		return
	}

	// Log success
	h.LogAction(c, "update", resourceName, map[string]interface{}{
		"user_id": userID,
		"id":      id,
		"entity":  entity,
	})

	h.Success(c, entity)
}

// HandleDelete provides a generic delete operation
func (h *BaseHandler) HandleDelete(
	c *gin.Context,
	deleteFunc func(context.Context, uuid.UUID) error,
	resourceName string,
) {
	// Get ID parameter
	id, err := h.GetUUIDParam(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	// Get user context
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	// Delete entity
	err = deleteFunc(c.Request.Context(), id)
	if err != nil {
		h.LogError(c, err, fmt.Sprintf("Failed to delete %s", resourceName), map[string]interface{}{
			"user_id": userID,
			"id":      id,
		})
		h.Error(c, err)
		return
	}

	// Log success
	h.LogAction(c, "delete", resourceName, map[string]interface{}{
		"user_id": userID,
		"id":      id,
	})

	h.Success(c, gin.H{"message": fmt.Sprintf("%s deleted successfully", resourceName)})
}

// Security helpers

// RequireRole checks if user has required role
func (h *BaseHandler) RequireRole(c *gin.Context, requiredRole string) error {
	_, _, userRole, err := h.GetUserContext(c)
	if err != nil {
		return err
	}

	if !h.hasRole(userRole, requiredRole) {
		return errors.NewForbiddenError("insufficient permissions")
	}

	return nil
}

// RequireOrganizationAccess checks if user has access to organization
func (h *BaseHandler) RequireOrganizationAccess(c *gin.Context, orgID uuid.UUID) error {
	userID, _, _, err := h.GetUserContext(c)
	if err != nil {
		return err
	}

	// Check if user has access to organization
	hasAccess, err := h.repos.Organizations().GetUserOrganizations(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	for _, org := range hasAccess {
		if org.ID == orgID {
			return nil
		}
	}

	return errors.NewForbiddenError("organization access denied")
}

// hasRole checks if user role has required permissions
func (h *BaseHandler) hasRole(userRole, requiredRole string) bool {
	roleHierarchy := map[string]int{
		"viewer": 1,
		"user":   2,
		"admin":  3,
		"owner":  4,
	}

	userLevel, exists := roleHierarchy[userRole]
	if !exists {
		return false
	}

	requiredLevel, exists := roleHierarchy[requiredRole]
	if !exists {
		return false
	}

	return userLevel >= requiredLevel
}

// Rate limiting helpers

// CheckRateLimit checks if request is within rate limits
func (h *BaseHandler) CheckRateLimit(c *gin.Context, key string, limit int, window time.Duration) error {
	// This would integrate with a rate limiting service
	// For now, we'll just log the check
	h.logger.LogInfo(c, "Rate limit check", map[string]interface{}{
		"key":    key,
		"limit":  limit,
		"window": window,
	})
	return nil
}

// Cache helpers

// GetFromCache retrieves data from cache
func (h *BaseHandler) GetFromCache(ctx context.Context, key string, target interface{}) error {
	// This would integrate with a caching service
	// For now, we'll return cache miss
	return errors.NewNotFoundError("cache miss")
}

// SetCache stores data in cache
func (h *BaseHandler) SetCache(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	// This would integrate with a caching service
	// For now, we'll just log the operation
	h.logger.LogInfo(ctx, "Cache set", map[string]interface{}{
		"key": key,
		"ttl": ttl,
	})
	return nil
}