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
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// APIError represents an API error with enhanced details support
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
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

// SuccessResponse sends a successful response
func SuccessResponse(c *gin.Context, data interface{}) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success:   true,
		Data:      data,
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusOK, response)
}

// SuccessResponseWithMeta sends a successful response with metadata
func SuccessResponseWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	// Ensure meta has timestamp
	if meta != nil {
		meta.Timestamp = time.Now()
	}
	
	response := APIResponse{
		Success:   true,
		Data:      data,
		Meta:      meta,
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusOK, response)
}

// CreatedResponse sends a 201 Created response
func CreatedResponse(c *gin.Context, data interface{}) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success:   true,
		Data:      data,
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusCreated, response)
}

// ErrorResponseFromError sends an error response based on the error type
func ErrorResponseFromError(c *gin.Context, err error) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	var statusCode int
	var apiError *APIError
	
	switch e := err.(type) {
	case *errors.AppError:
		switch e.Type {
		case errors.ErrorTypeValidation:
			statusCode = http.StatusBadRequest
		case errors.ErrorTypeAuthentication:
			statusCode = http.StatusUnauthorized
		case errors.ErrorTypeAuthorization:
			statusCode = http.StatusForbidden
		case errors.ErrorTypeNotFound:
			statusCode = http.StatusNotFound
		case errors.ErrorTypeConflict:
			statusCode = http.StatusConflict
		case errors.ErrorTypeRateLimit:
			statusCode = http.StatusTooManyRequests
		case errors.ErrorTypeTimeout:
			statusCode = http.StatusRequestTimeout
		default:
			statusCode = http.StatusInternalServerError
		}
		
		apiError = &APIError{
			Code:    e.Code,
			Message: e.Message,
		}
		
		// Add details if available - now using map[string]interface{}
		if len(e.Details) > 0 {
			apiError.Details = make(map[string]interface{})
			for k, v := range e.Details {
				apiError.Details[k] = v
			}
		}
	default:
		statusCode = http.StatusInternalServerError
		apiError = &APIError{
			Code:    "UNKNOWN_ERROR",
			Message: "An unknown error occurred",
		}
	}
	
	response := APIResponse{
		Success:   false,
		Error:     apiError,
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(statusCode, response)
}

// BadRequestResponse sends a 400 Bad Request response
func BadRequestResponse(c *gin.Context, message string) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "BAD_REQUEST",
			Message: message,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusBadRequest, response)
}

// UnauthorizedResponse sends a 401 Unauthorized response
func UnauthorizedResponse(c *gin.Context, message string) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusUnauthorized, response)
}

// ForbiddenResponse sends a 403 Forbidden response
func ForbiddenResponse(c *gin.Context, message string) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "FORBIDDEN",
			Message: message,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusForbidden, response)
}

// NotFoundResponse sends a 404 Not Found response
func NotFoundResponse(c *gin.Context, message string) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "NOT_FOUND",
			Message: message,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusNotFound, response)
}

// InternalErrorResponse sends a 500 Internal Server Error response
func InternalErrorResponse(c *gin.Context, message string) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "INTERNAL_ERROR",
			Message: message,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusInternalServerError, response)
}

// ValidationErrorResponse sends a 400 Bad Request response with validation details
func ValidationErrorResponse(c *gin.Context, message string, details map[string]interface{}) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "VALIDATION_ERROR",
			Message: message,
			Details: details,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusBadRequest, response)
}

// ConflictResponse sends a 409 Conflict response
func ConflictResponse(c *gin.Context, message string) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "CONFLICT",
			Message: message,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusConflict, response)
}

// TooManyRequestsResponse sends a 429 Too Many Requests response
func TooManyRequestsResponse(c *gin.Context, message string) {
	requestID, exists := c.Get("request_id")
	requestIDStr := ""
	if exists {
		if id, ok := requestID.(string); ok {
			requestIDStr = id
		}
	}
	
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "RATE_LIMIT_EXCEEDED",
			Message: message,
		},
		RequestID: requestIDStr,
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusTooManyRequests, response)
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
	RepositoryURL   string   `json:"repository_url" binding:"required,url"`
	Branch          string   `json:"branch"`
	CommitSHA       string   `json:"commit_sha"`
	ScanType        string   `json:"scan_type" binding:"required,oneof=full incremental ide"`
	Priority        int      `json:"priority" binding:"min=1,max=10"`
	AgentsRequested []string `json:"agents_requested"`
}

// UpdateScanJobStatusRequest represents a request to update scan job status
type UpdateScanJobStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=queued running completed failed cancelled"`
}

// UpdateFindingStatusRequest represents a request to update finding status
type UpdateFindingStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=open fixed ignored false_positive"`
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