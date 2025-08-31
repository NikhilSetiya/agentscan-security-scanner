package entities

import (
	"fmt"
)

// DomainError represents a domain-specific error
type DomainError struct {
	Type    string
	Message string
	Field   string
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field: %s)", e.Type, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *DomainError {
	return &DomainError{
		Type:    "ValidationError",
		Message: message,
	}
}

// NewValidationErrorWithField creates a new validation error with field information
func NewValidationErrorWithField(message, field string) *DomainError {
	return &DomainError{
		Type:    "ValidationError",
		Message: message,
		Field:   field,
	}
}

// NewBusinessRuleError creates a new business rule error
func NewBusinessRuleError(message string) *DomainError {
	return &DomainError{
		Type:    "BusinessRuleError",
		Message: message,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string) *DomainError {
	return &DomainError{
		Type:    "NotFoundError",
		Message: message,
	}
}

// NewConflictError creates a new conflict error
func NewConflictError(message string) *DomainError {
	return &DomainError{
		Type:    "ConflictError",
		Message: message,
	}
}