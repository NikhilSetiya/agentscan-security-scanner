package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/agentscan/agentscan/pkg/errors"
)

// StandardResponse represents the standard API response format
type StandardResponse struct {
	Success       bool        `json:"success"`
	Data          interface{} `json:"data,omitempty"`
	Error         *ErrorInfo  `json:"error,omitempty"`
	Meta          *MetaInfo   `json:"meta,omitempty"`
	RequestID     string      `json:"request_id,omitempty"`
	CorrelationID string      `json:"correlation_id,omitempty"`
	Timestamp     time.Time   `json:"timestamp"`
	Version       string      `json:"version"`
}

// ErrorInfo represents error information in responses
type ErrorInfo struct {
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Type          string                 `json:"type,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// MetaInfo represents metadata in responses
type MetaInfo struct {
	Pagination *PaginationResponse `json:"pagination,omitempty"`
	Timestamp  time.Time           `json:"timestamp"`
}

// ResponseHelper provides helper methods for creating standardized responses
type ResponseHelper struct{}

// NewResponseHelper creates a new response helper
func NewResponseHelper() *ResponseHelper {
	return &ResponseHelper{}
}

// getContextInfo extracts request and correlation IDs from context
func (rh *ResponseHelper) getContextInfo(c *gin.Context) (requestID, correlationID string) {
	if id, exists := c.Get("request_id"); exists {
		if idStr, ok := id.(string); ok {
			requestID = idStr
		}
	}
	
	if id, exists := c.Get("correlation_id"); exists {
		if idStr, ok := id.(string); ok {
			correlationID = idStr
		}
	}
	
	return requestID, correlationID
}

// Success sends a successful response
func (rh *ResponseHelper) Success(c *gin.Context, data interface{}) {
	requestID, correlationID := rh.getContextInfo(c)
	
	response := StandardResponse{
		Success:       true,
		Data:          data,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Version:       "v1",
	}
	
	c.JSON(http.StatusOK, response)
}

// SuccessWithMeta sends a successful response with metadata
func (rh *ResponseHelper) SuccessWithMeta(c *gin.Context, data interface{}, meta *MetaInfo) {
	requestID, correlationID := rh.getContextInfo(c)
	
	if meta != nil {
		meta.Timestamp = time.Now()
	}
	
	response := StandardResponse{
		Success:       true,
		Data:          data,
		Meta:          meta,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Version:       "v1",
	}
	
	c.JSON(http.StatusOK, response)
}

// Created sends a 201 Created response
func (rh *ResponseHelper) Created(c *gin.Context, data interface{}) {
	requestID, correlationID := rh.getContextInfo(c)
	
	response := StandardResponse{
		Success:       true,
		Data:          data,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Version:       "v1",
	}
	
	c.JSON(http.StatusCreated, response)
}

// Paginated sends a paginated response
func (rh *ResponseHelper) Paginated(c *gin.Context, data interface{}, pagination PaginationResponse) {
	meta := &MetaInfo{
		Pagination: &pagination,
		Timestamp:  time.Now(),
	}
	
	rh.SuccessWithMeta(c, data, meta)
}

// Error sends an error response
func (rh *ResponseHelper) Error(c *gin.Context, err error) {
	requestID, correlationID := rh.getContextInfo(c)
	
	var statusCode int
	var errorInfo *ErrorInfo
	
	if appErr, ok := err.(*errors.AppError); ok {
		statusCode = rh.mapErrorTypeToHTTPStatus(appErr.Type)
		
		errorInfo = &ErrorInfo{
			Code:          appErr.Code,
			Message:       appErr.Message,
			Type:          string(appErr.Type),
			Details:       appErr.Details,
			RequestID:     requestID,
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		}
	} else {
		statusCode = http.StatusInternalServerError
		errorInfo = &ErrorInfo{
			Code:          "INTERNAL_ERROR",
			Message:       "An internal error occurred",
			Type:          string(errors.ErrorTypeInternal),
			RequestID:     requestID,
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		}
	}
	
	response := StandardResponse{
		Success:       false,
		Error:         errorInfo,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Version:       "v1",
	}
	
	c.JSON(statusCode, response)
}

// BadRequest sends a 400 Bad Request response
func (rh *ResponseHelper) BadRequest(c *gin.Context, message string, details map[string]interface{}) {
	err := errors.NewValidationError(message).WithDetails(details)
	rh.Error(c, err)
}

// Unauthorized sends a 401 Unauthorized response
func (rh *ResponseHelper) Unauthorized(c *gin.Context, message string) {
	err := errors.NewUnauthorizedError(message)
	rh.Error(c, err)
}

// Forbidden sends a 403 Forbidden response
func (rh *ResponseHelper) Forbidden(c *gin.Context, message string) {
	err := errors.NewForbiddenError(message)
	rh.Error(c, err)
}

// NotFound sends a 404 Not Found response
func (rh *ResponseHelper) NotFound(c *gin.Context, message string) {
	err := errors.NewNotFoundError(message)
	rh.Error(c, err)
}

// Conflict sends a 409 Conflict response
func (rh *ResponseHelper) Conflict(c *gin.Context, message string) {
	err := errors.NewConflictError(message)
	rh.Error(c, err)
}

// InternalError sends a 500 Internal Server Error response
func (rh *ResponseHelper) InternalError(c *gin.Context, message string) {
	err := errors.NewInternalError(message)
	rh.Error(c, err)
}

// ValidationError sends a validation error response
func (rh *ResponseHelper) ValidationError(c *gin.Context, message string, details map[string]interface{}) {
	err := errors.NewValidationError(message).WithDetails(details)
	rh.Error(c, err)
}

// mapErrorTypeToHTTPStatus maps error types to HTTP status codes
func (rh *ResponseHelper) mapErrorTypeToHTTPStatus(errorType errors.ErrorType) int {
	switch errorType {
	case errors.ErrorTypeValidation:
		return http.StatusBadRequest
	case errors.ErrorTypeAuthentication:
		return http.StatusUnauthorized
	case errors.ErrorTypeAuthorization:
		return http.StatusForbidden
	case errors.ErrorTypeNotFound:
		return http.StatusNotFound
	case errors.ErrorTypeConflict:
		return http.StatusConflict
	case errors.ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case errors.ErrorTypeTimeout:
		return http.StatusRequestTimeout
	case errors.ErrorTypeExternal:
		return http.StatusBadGateway
	case errors.ErrorTypeSecurity:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// Global response helper instance
var GlobalResponseHelper = NewResponseHelper()

// Convenience functions using the global helper

// SuccessResponse sends a successful response using the global helper
func SuccessResponse(c *gin.Context, data interface{}) {
	GlobalResponseHelper.Success(c, data)
}

// CreatedResponse sends a 201 Created response using the global helper
func CreatedResponse(c *gin.Context, data interface{}) {
	GlobalResponseHelper.Created(c, data)
}

// PaginatedResponse sends a paginated response using the global helper
func PaginatedResponse(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	pagination := CreatePaginationResponse(page, pageSize, total)
	GlobalResponseHelper.Paginated(c, data, pagination)
}

// ErrorResponse sends an error response using the global helper
func ErrorResponse(c *gin.Context, err error) {
	GlobalResponseHelper.Error(c, err)
}

// BadRequestResponse sends a 400 Bad Request response using the global helper
func BadRequestResponse(c *gin.Context, message string) {
	GlobalResponseHelper.BadRequest(c, message, nil)
}

// ValidationErrorResponse sends a validation error response using the global helper
func ValidationErrorResponse(c *gin.Context, message string, details map[string]interface{}) {
	GlobalResponseHelper.ValidationError(c, message, details)
}

// UnauthorizedResponse sends a 401 Unauthorized response using the global helper
func UnauthorizedResponse(c *gin.Context, message string) {
	GlobalResponseHelper.Unauthorized(c, message)
}

// ForbiddenResponse sends a 403 Forbidden response using the global helper
func ForbiddenResponse(c *gin.Context, message string) {
	GlobalResponseHelper.Forbidden(c, message)
}

// NotFoundResponse sends a 404 Not Found response using the global helper
func NotFoundResponse(c *gin.Context, message string) {
	GlobalResponseHelper.NotFound(c, message)
}

// ConflictResponse sends a 409 Conflict response using the global helper
func ConflictResponse(c *gin.Context, message string) {
	GlobalResponseHelper.Conflict(c, message)
}

// InternalErrorResponse sends a 500 Internal Server Error response using the global helper
func InternalErrorResponse(c *gin.Context, message string) {
	GlobalResponseHelper.InternalError(c, message)
}