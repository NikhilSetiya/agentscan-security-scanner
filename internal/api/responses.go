package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success       bool        `json:"success"`
	Data          interface{} `json:"data,omitempty"`
	Error         *APIError   `json:"error,omitempty"`
	Meta          *Meta       `json:"meta,omitempty"`
	RequestID     string      `json:"request_id,omitempty"`
	CorrelationID string      `json:"correlation_id,omitempty"`
	Timestamp     time.Time   `json:"timestamp"`
	Version       string      `json:"version,omitempty"`
}

// APIError represents an API error with enhanced details support
type APIError struct {
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details,omitempty"`
	Type          string                 `json:"type,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Stack         []string               `json:"stack,omitempty"`
}

// Meta represents response metadata with enhanced pagination support
type Meta struct {
	Pagination *Pagination `json:"pagination,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// ErrorResponse represents a simple error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// getContextInfo extracts request and correlation IDs from context
func getContextInfo(c *gin.Context) (requestID, correlationID string) {
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

// SuccessResponse sends a successful response
func SuccessResponse(c *gin.Context, data interface{}) {
	requestID, correlationID := getContextInfo(c)
	
	response := APIResponse{
		Success:       true,
		Data:          data,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Version:       "v1",
	}
	
	c.JSON(http.StatusOK, response)
}

// SuccessResponseWithMeta sends a successful response with metadata
func SuccessResponseWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	requestID, correlationID := getContextInfo(c)
	
	// Ensure meta has timestamp
	if meta != nil {
		meta.Timestamp = time.Now()
	}
	
	response := APIResponse{
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

// CreatedResponse sends a 201 Created response
func CreatedResponse(c *gin.Context, data interface{}) {
	requestID, correlationID := getContextInfo(c)
	
	response := APIResponse{
		Success:       true,
		Data:          data,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Version:       "v1",
	}
	
	c.JSON(http.StatusCreated, response)
}

// ErrorResponseFromError sends an error response based on the error type
func ErrorResponseFromError(c *gin.Context, err error) {
	requestID, correlationID := getContextInfo(c)
	
	var statusCode int
	var apiError *APIError
	
	switch e := err.(type) {
	case *errors.AppError:
		statusCode = mapErrorTypeToHTTPStatus(e.Type)
		
		apiError = &APIError{
			Code:          e.Code,
			Message:       e.Message,
			Type:          string(e.Type),
			RequestID:     requestID,
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		}
		
		// Add details if available
		if len(e.Details) > 0 {
			apiError.Details = make(map[string]interface{})
			for k, v := range e.Details {
				apiError.Details[k] = v
			}
		}
		
		// Add stack trace for internal errors in debug mode
		if e.Type == errors.ErrorTypeInternal && len(e.Stack) > 0 {
			apiError.Stack = e.Stack
		}
		
	default:
		statusCode = http.StatusInternalServerError
		apiError = &APIError{
			Code:          "UNKNOWN_ERROR",
			Message:       "An unknown error occurred",
			Type:          string(errors.ErrorTypeInternal),
			RequestID:     requestID,
			CorrelationID: correlationID,
			Timestamp:     time.Now(),
		}
	}
	
	response := APIResponse{
		Success:       false,
		Error:         apiError,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Version:       "v1",
	}
	
	c.JSON(statusCode, response)
}

// mapErrorTypeToHTTPStatus maps error types to HTTP status codes
func mapErrorTypeToHTTPStatus(errorType errors.ErrorType) int {
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

// createErrorResponse creates a standardized error response
func createErrorResponse(c *gin.Context, statusCode int, code, message string, errorType errors.ErrorType, details map[string]interface{}) {
	requestID, correlationID := getContextInfo(c)
	
	apiError := &APIError{
		Code:          code,
		Message:       message,
		Type:          string(errorType),
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Details:       details,
	}
	
	response := APIResponse{
		Success:       false,
		Error:         apiError,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
		Version:       "v1",
	}
	
	c.JSON(statusCode, response)
}

// BadRequestResponse sends a 400 Bad Request response
func BadRequestResponse(c *gin.Context, message string) {
	createErrorResponse(c, http.StatusBadRequest, "BAD_REQUEST", message, errors.ErrorTypeValidation, nil)
}

// UnauthorizedResponse sends a 401 Unauthorized response
func UnauthorizedResponse(c *gin.Context, message string) {
	createErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", message, errors.ErrorTypeAuthentication, nil)
}

// ForbiddenResponse sends a 403 Forbidden response
func ForbiddenResponse(c *gin.Context, message string) {
	createErrorResponse(c, http.StatusForbidden, "FORBIDDEN", message, errors.ErrorTypeAuthorization, nil)
}

// NotFoundResponse sends a 404 Not Found response
func NotFoundResponse(c *gin.Context, message string) {
	createErrorResponse(c, http.StatusNotFound, "NOT_FOUND", message, errors.ErrorTypeNotFound, nil)
}

// InternalErrorResponse sends a 500 Internal Server Error response
func InternalErrorResponse(c *gin.Context, message string) {
	createErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", message, errors.ErrorTypeInternal, nil)
}

// ValidationErrorResponse sends a 400 Bad Request response with validation details
func ValidationErrorResponse(c *gin.Context, message string, details map[string]interface{}) {
	createErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", message, errors.ErrorTypeValidation, details)
}

// ConflictResponse sends a 409 Conflict response
func ConflictResponse(c *gin.Context, message string) {
	createErrorResponse(c, http.StatusConflict, "CONFLICT", message, errors.ErrorTypeConflict, nil)
}

// TooManyRequestsResponse sends a 429 Too Many Requests response
func TooManyRequestsResponse(c *gin.Context, message string) {
	createErrorResponse(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", message, errors.ErrorTypeRateLimit, nil)
}

// Helper functions for pagination

// NewPagination creates a new pagination metadata object
func NewPagination(page, pageSize int, total int64) *Pagination {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	
	return &Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// NewMetaWithPagination creates a new Meta object with pagination
func NewMetaWithPagination(page, pageSize int, total int64) *Meta {
	return &Meta{
		Pagination: NewPagination(page, pageSize, total),
		Timestamp:  time.Now(),
	}
}

// PaginatedResponse sends a successful response with pagination metadata
func PaginatedResponse(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	meta := NewMetaWithPagination(page, pageSize, total)
	SuccessResponseWithMeta(c, data, meta)
}

// DTO types for API requests and responses

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	User      *UserDTO   `json:"user"`
}

// UserDTO represents a user in API responses
type UserDTO struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// ScanJobDTO represents a scan job in API responses
type ScanJobDTO struct {
	ID               uuid.UUID              `json:"id"`
	RepositoryID     uuid.UUID              `json:"repository_id"`
	Branch           string                 `json:"branch"`
	CommitSHA        string                 `json:"commit_sha"`
	ScanType         string                 `json:"scan_type"`
	Priority         int                    `json:"priority"`
	Status           string                 `json:"status"`
	AgentsRequested  []string               `json:"agents_requested"`
	AgentsCompleted  []string               `json:"agents_completed"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// FindingDTO represents a finding in API responses
type FindingDTO struct {
	ID             uuid.UUID              `json:"id"`
	Tool           string                 `json:"tool"`
	RuleID         string                 `json:"rule_id"`
	Severity       string                 `json:"severity"`
	Category       string                 `json:"category"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	FilePath       string                 `json:"file_path"`
	LineNumber     int                    `json:"line_number"`
	ColumnNumber   int                    `json:"column_number,omitempty"`
	CodeSnippet    string                 `json:"code_snippet,omitempty"`
	Confidence     float64                `json:"confidence"`
	ConsensusScore *float64               `json:"consensus_score,omitempty"`
	Status         string                 `json:"status"`
	FixSuggestion  map[string]interface{} `json:"fix_suggestion,omitempty"`
	References     []string               `json:"references,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// CreateScanJobRequest represents a request to create a scan job
type CreateScanJobRequest struct {
	RepositoryURL   string   `json:"repository_url" binding:"required,url" validate:"required,repository_url"`
	Branch          string   `json:"branch" validate:"omitempty,min=1,max=100,safe_string,no_sql_injection"`
	CommitSHA       string   `json:"commit_sha" validate:"omitempty,min=1,max=100,safe_string,no_sql_injection"`
	ScanType        string   `json:"scan_type" binding:"required,oneof=full incremental ide" validate:"required,scan_type"`
	Priority        int      `json:"priority" binding:"min=1,max=10" validate:"min=1,max=10"`
	AgentsRequested []string `json:"agents_requested" validate:"omitempty,dive,safe_string,no_sql_injection"`
}

// UpdateScanJobStatusRequest represents a request to update scan job status
type UpdateScanJobStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=queued running completed failed cancelled" validate:"required,oneof=queued running completed failed cancelled"`
}

// UpdateFindingStatusRequest represents a request to update finding status
type UpdateFindingStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=open fixed ignored false_positive" validate:"required,oneof=open fixed ignored false_positive"`
}

// Conversion functions

// ToUserDTO converts a User to UserDTO
func ToUserDTO(user *types.User) *UserDTO {
	// Extract username from email if not available
	username := user.Name
	if username == "" {
		if atIndex := strings.Index(user.Email, "@"); atIndex > 0 {
			username = user.Email[:atIndex]
		} else {
			username = user.Email
		}
	}

	// Default role - in production this should come from database
	role := "developer" // matches frontend expectation: 'admin' | 'developer' | 'viewer'
	
	return &UserDTO{
		ID:        user.ID,
		Username:  username,
		Email:     user.Email,
		Name:      user.Name,
		Role:      role,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
	}
}

// ToScanJobDTO converts a ScanJob to ScanJobDTO
func ToScanJobDTO(job *types.ScanJob) *ScanJobDTO {
	return &ScanJobDTO{
		ID:               job.ID,
		RepositoryID:     job.RepositoryID,
		Branch:           job.Branch,
		CommitSHA:        job.CommitSHA,
		ScanType:         job.ScanType,
		Priority:         job.Priority,
		Status:           job.Status,
		AgentsRequested:  job.AgentsRequested,
		AgentsCompleted:  job.AgentsCompleted,
		StartedAt:        job.StartedAt,
		CompletedAt:      job.CompletedAt,
		ErrorMessage:     job.ErrorMessage,
		Metadata:         job.Metadata,
		CreatedAt:        job.CreatedAt,
		UpdatedAt:        job.UpdatedAt,
	}
}

// ToFindingDTO converts a Finding to FindingDTO
func ToFindingDTO(finding *types.Finding) *FindingDTO {
	return &FindingDTO{
		ID:             finding.ID,
		Tool:           finding.Tool,
		RuleID:         finding.RuleID,
		Severity:       finding.Severity,
		Category:       finding.Category,
		Title:          finding.Title,
		Description:    finding.Description,
		FilePath:       finding.FilePath,
		LineNumber:     finding.LineNumber,
		ColumnNumber:   finding.ColumnNumber,
		CodeSnippet:    finding.CodeSnippet,
		Confidence:     finding.Confidence,
		ConsensusScore: finding.ConsensusScore,
		Status:         finding.Status,
		FixSuggestion:  finding.FixSuggestion,
		References:     finding.References,
		CreatedAt:      finding.CreatedAt,
		UpdatedAt:      finding.UpdatedAt,
	}
}