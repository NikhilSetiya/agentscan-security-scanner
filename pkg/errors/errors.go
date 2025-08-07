package errors

import (
	"fmt"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeAuthorization  ErrorType = "authorization"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeInternal       ErrorType = "internal"
	ErrorTypeExternal       ErrorType = "external"
	ErrorTypeTimeout        ErrorType = "timeout"
)

// AppError represents an application error with context
type AppError struct {
	Type      ErrorType         `json:"type"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id"`
	Timestamp time.Time         `json:"timestamp"`
	Cause     error             `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError creates a new application error
func NewAppError(errorType ErrorType, code, message string) *AppError {
	return &AppError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Details:   make(map[string]string),
		Timestamp: time.Now(),
	}
}

// WithCause adds a cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key, value string) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// Common error constructors
func NewValidationError(message string) *AppError {
	return NewAppError(ErrorTypeValidation, "VALIDATION_ERROR", message)
}

func NewAuthenticationError(message string) *AppError {
	return NewAppError(ErrorTypeAuthentication, "AUTHENTICATION_ERROR", message)
}

func NewAuthorizationError(message string) *AppError {
	return NewAppError(ErrorTypeAuthorization, "AUTHORIZATION_ERROR", message)
}

func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrorTypeNotFound, "NOT_FOUND", fmt.Sprintf("%s not found", resource))
}

func NewConflictError(message string) *AppError {
	return NewAppError(ErrorTypeConflict, "CONFLICT", message)
}

func NewRateLimitError(message string) *AppError {
	return NewAppError(ErrorTypeRateLimit, "RATE_LIMIT_EXCEEDED", message)
}

func NewInternalError(message string) *AppError {
	return NewAppError(ErrorTypeInternal, "INTERNAL_ERROR", message)
}

func NewExternalError(service, message string) *AppError {
	return NewAppError(ErrorTypeExternal, "EXTERNAL_SERVICE_ERROR", message).
		WithDetail("service", service)
}

func NewTimeoutError(operation string) *AppError {
	return NewAppError(ErrorTypeTimeout, "TIMEOUT", fmt.Sprintf("%s timed out", operation))
}

// Agent-specific errors
func NewAgentError(agentName, message string) *AppError {
	return NewAppError(ErrorTypeInternal, "AGENT_ERROR", message).
		WithDetail("agent", agentName)
}

func NewScanError(scanID, message string) *AppError {
	return NewAppError(ErrorTypeInternal, "SCAN_ERROR", message).
		WithDetail("scan_id", scanID)
}

func NewConsensusError(message string) *AppError {
	return NewAppError(ErrorTypeInternal, "CONSENSUS_ERROR", message)
}

// IsType checks if the error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == errorType
	}
	return false
}

// GetCode returns the error code if it's an AppError
func GetCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return "UNKNOWN_ERROR"
}

// GetType returns the error type if it's an AppError
func GetType(err error) ErrorType {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type
	}
	return ErrorTypeInternal
}