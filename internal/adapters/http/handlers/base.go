package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/shared/utils"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/adapters/http/middleware"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/adapters/http/validators"
)

// BaseHandler provides common functionality for all API handlers
type BaseHandler struct {
	logger    *zap.Logger
	validator *validators.InputValidator
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(logger *zap.Logger) *BaseHandler {
	return &BaseHandler{
		logger:    logger,
		validator: validators.NewInputValidator(),
	}
}

// GetAuthenticatedUser extracts authenticated user information from context
func (h *BaseHandler) GetAuthenticatedUser(c *gin.Context) (uuid.UUID, string, string, error) {
	return middleware.GetUserFromContext(c)
}

// RequireAuthentication ensures user is authenticated
func (h *BaseHandler) RequireAuthentication(c *gin.Context) (uuid.UUID, string, string, bool) {
	userID, email, role, err := h.GetAuthenticatedUser(c)
	if err != nil {
		utils.UnauthorizedResponse(c, "User authentication required")
		return uuid.Nil, "", "", false
	}
	return userID, email, role, true
}

// ParseUUIDParam extracts and validates UUID parameter
func (h *BaseHandler) ParseUUIDParam(c *gin.Context, paramName string) (uuid.UUID, bool) {
	id, err := utils.ParseUUIDParam(c, paramName)
	if err != nil {
		utils.ErrorResponse(c, err)
		return uuid.Nil, false
	}
	return id, true
}

// ParsePagination extracts pagination parameters
func (h *BaseHandler) ParsePagination(c *gin.Context, defaultPageSize, maxPageSize int) utils.PaginationRequest {
	return utils.ParsePaginationFromQuery(c, defaultPageSize, maxPageSize)
}

// ParseFilters extracts common filter parameters
func (h *BaseHandler) ParseFilters(c *gin.Context) utils.FilterParams {
	return utils.ParseFilterParams(c)
}

// ValidateAndBind validates and binds JSON request body
func (h *BaseHandler) ValidateAndBind(c *gin.Context, target interface{}) bool {
	if err := utils.ValidateAndBindJSON(c, target); err != nil {
		utils.ErrorResponse(c, err)
		return false
	}
	return true
}

// HandleError logs and responds with error
func (h *BaseHandler) HandleError(c *gin.Context, err error, context string) {
	// Log error with context
	h.logger.Error("handler error",
		zap.String("context", context),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.Error(err),
	)
	
	utils.ErrorResponse(c, err)
}

// Success sends a successful response
func (h *BaseHandler) Success(c *gin.Context, data interface{}) {
	utils.SuccessResponse(c, data)
}

// Created sends a 201 Created response
func (h *BaseHandler) Created(c *gin.Context, data interface{}) {
	utils.CreatedResponse(c, data)
}

// Paginated sends a paginated response
func (h *BaseHandler) Paginated(c *gin.Context, data interface{}, pagination utils.PaginationRequest, total int64) {
	utils.PaginatedResponse(c, data, pagination.Page, pagination.PageSize, total)
}

// BadRequest sends a 400 Bad Request response
func (h *BaseHandler) BadRequest(c *gin.Context, message string) {
	utils.BadRequestResponse(c, message)
}

// Unauthorized sends a 401 Unauthorized response
func (h *BaseHandler) Unauthorized(c *gin.Context, message string) {
	utils.UnauthorizedResponse(c, message)
}

// Forbidden sends a 403 Forbidden response
func (h *BaseHandler) Forbidden(c *gin.Context, message string) {
	utils.ForbiddenResponse(c, message)
}

// NotFound sends a 404 Not Found response
func (h *BaseHandler) NotFound(c *gin.Context, message string) {
	utils.NotFoundResponse(c, message)
}

// Conflict sends a 409 Conflict response
func (h *BaseHandler) Conflict(c *gin.Context, message string) {
	utils.ConflictResponse(c, message)
}

// InternalError sends a 500 Internal Server Error response
func (h *BaseHandler) InternalError(c *gin.Context, message string) {
	utils.InternalErrorResponse(c, message)
}

// ValidationError sends a validation error response
func (h *BaseHandler) ValidationError(c *gin.Context, message string, details map[string]interface{}) {
	utils.ValidationErrorResponse(c, message, details)
}

// LogRequest logs incoming request for debugging
func (h *BaseHandler) LogRequest(c *gin.Context, handlerName string) {
	h.logger.Info("handling request",
		zap.String("handler", handlerName),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("user_agent", c.Request.UserAgent()),
		zap.String("remote_addr", c.ClientIP()),
	)
}

// LogResponse logs response for debugging
func (h *BaseHandler) LogResponse(c *gin.Context, handlerName string, statusCode int) {
	h.logger.Info("response sent",
		zap.String("handler", handlerName),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.Int("status_code", statusCode),
	)
}