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
	ErrorTypeSecurity       ErrorType = "security"
)

// AppError represents an application error with context
type AppError struct {
	Type          ErrorType              `json:"type"`
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details,omitempty"`
	RequestID     string                 `json:"request_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Cause         error                  `json:"-"`
	Stack         []string               `json:"stack,omitempty"`
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
		Details:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// WithCause adds a cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithDetails adds multiple details to the error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// WithCorrelationID adds a correlation ID to the error
func (e *AppError) WithCorrelationID(correlationID string) *AppError {
	e.CorrelationID = correlationID
	return e
}

// WithStack adds stack trace information to the error
func (e *AppError) WithStack(stack []string) *AppError {
	e.Stack = stack
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

func NewSecurityError(message string) *AppError {
	return NewAppError(ErrorTypeSecurity, "SECURITY_ERROR", message)
}

// Additional error constructors for better error handling

func NewUnauthorizedError(message string) *AppError {
	return NewAppError(ErrorTypeAuthentication, "UNAUTHORIZED", message)
}

func NewForbiddenError(message string) *AppError {
	return NewAppError(ErrorTypeAuthorization, "FORBIDDEN", message)
}

func NewBadRequestError(message string) *AppError {
	return NewAppError(ErrorTypeValidation, "BAD_REQUEST", message)
}

func NewServiceUnavailableError(service string) *AppError {
	return NewAppError(ErrorTypeExternal, "SERVICE_UNAVAILABLE", fmt.Sprintf("%s service is unavailable", service)).
		WithDetail("service", service)
}

func NewDatabaseError(operation, message string) *AppError {
	return NewAppError(ErrorTypeInternal, "DATABASE_ERROR", message).
		WithDetail("operation", operation)
}

func NewNetworkError(message string) *AppError {
	return NewAppError(ErrorTypeExternal, "NETWORK_ERROR", message)
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

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	return IsType(err, ErrorTypeNotFound)
}

// IsValidation checks if the error is a validation error
func IsValidation(err error) bool {
	return IsType(err, ErrorTypeValidation)
}

// IsAuthentication checks if the error is an authentication error
func IsAuthentication(err error) bool {
	return IsType(err, ErrorTypeAuthentication)
}

// IsAuthorization checks if the error is an authorization error
func IsAuthorization(err error) bool {
	return IsType(err, ErrorTypeAuthorization)
}

// IsTimeout checks if the error is a timeout error
func IsTimeout(err error) bool {
	return IsType(err, ErrorTypeTimeout)
}

// IsRateLimit checks if the error is a rate limit error
func IsRateLimit(err error) bool {
	return IsType(err, ErrorTypeRateLimit)
}

// Error wrapping and unwrapping utilities

// WrapError wraps an existing error with additional context
func WrapError(err error, message string) *AppError {
	if appErr, ok := err.(*AppError); ok {
		// If it's already an AppError, preserve the original type and add context
		return &AppError{
			Type:          appErr.Type,
			Code:          appErr.Code,
			Message:       fmt.Sprintf("%s: %s", message, appErr.Message),
			Details:       appErr.Details,
			RequestID:     appErr.RequestID,
			CorrelationID: appErr.CorrelationID,
			Timestamp:     time.Now(),
			Cause:         appErr,
			Stack:         appErr.Stack,
		}
	}
	
	// If it's a regular error, wrap it as an internal error
	return NewInternalError(message).WithCause(err)
}

// WrapWithType wraps an error with a specific type
func WrapWithType(err error, errorType ErrorType, code, message string) *AppError {
	appErr := NewAppError(errorType, code, message).WithCause(err)
	
	// If the original error is an AppError, preserve some context
	if originalAppErr, ok := err.(*AppError); ok {
		if originalAppErr.RequestID != "" {
			appErr.RequestID = originalAppErr.RequestID
		}
		if originalAppErr.CorrelationID != "" {
			appErr.CorrelationID = originalAppErr.CorrelationID
		}
		// Merge details
		for k, v := range originalAppErr.Details {
			appErr.WithDetail(k, v)
		}
	}
	
	return appErr
}

// UnwrapAll returns all errors in the chain
func UnwrapAll(err error) []error {
	var errors []error
	for err != nil {
		errors = append(errors, err)
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return errors
}

// GetRootCause returns the root cause of an error chain
func GetRootCause(err error) error {
	for {
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			if cause := unwrapper.Unwrap(); cause != nil {
				err = cause
				continue
			}
		}
		break
	}
	return err
}

// HasType checks if any error in the chain has the specified type
func HasType(err error, errorType ErrorType) bool {
	for err != nil {
		if appErr, ok := err.(*AppError); ok {
			if appErr.Type == errorType {
				return true
			}
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}

// GetFirstOfType returns the first error of the specified type in the chain
func GetFirstOfType(err error, errorType ErrorType) *AppError {
	for err != nil {
		if appErr, ok := err.(*AppError); ok {
			if appErr.Type == errorType {
				return appErr
			}
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return nil
}

// ErrorChain represents a chain of errors for detailed logging
type ErrorChain struct {
	Errors []ErrorInfo `json:"errors"`
}

// ErrorInfo represents information about an error in the chain
type ErrorInfo struct {
	Type      string                 `json:"type"`
	Code      string                 `json:"code,omitempty"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp,omitempty"`
}

// GetErrorChain returns detailed information about the error chain
func GetErrorChain(err error) *ErrorChain {
	chain := &ErrorChain{
		Errors: make([]ErrorInfo, 0),
	}
	
	for err != nil {
		if appErr, ok := err.(*AppError); ok {
			chain.Errors = append(chain.Errors, ErrorInfo{
				Type:      string(appErr.Type),
				Code:      appErr.Code,
				Message:   appErr.Message,
				Details:   appErr.Details,
				Timestamp: appErr.Timestamp,
			})
		} else {
			chain.Errors = append(chain.Errors, ErrorInfo{
				Type:    "unknown",
				Message: err.Error(),
			})
		}
		
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	
	return chain
}

// Context helpers for error correlation

// FromContext extracts error context from request context
func FromContext(ctx interface{}) (requestID, correlationID string) {
	// This would be implemented based on your context structure
	// For now, return empty strings
	return "", ""
}

// WithContext adds context information to an error
func WithContext(err *AppError, ctx interface{}) *AppError {
	requestID, correlationID := FromContext(ctx)
	if requestID != "" {
		err.WithRequestID(requestID)
	}
	if correlationID != "" {
		err.WithCorrelationID(correlationID)
	}
	return err
}