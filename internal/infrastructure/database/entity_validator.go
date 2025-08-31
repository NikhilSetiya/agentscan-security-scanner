package database

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/pkg/errors"
)

// EntityValidator provides validation functionality for repository entities
type EntityValidator[T any] struct {
	strict bool
	rules  map[string]repositories.ValidationRule[T]
	mu     sync.RWMutex
}

// NewEntityValidator creates a new entity validator
func NewEntityValidator[T any](strict bool) *EntityValidator[T] {
	validator := &EntityValidator[T]{
		strict: strict,
		rules:  make(map[string]repositories.ValidationRule[T]),
	}
	
	// Add default validation rules
	validator.addDefaultRules()
	
	return validator
}

// Validate validates an entity against all registered rules
func (ev *EntityValidator[T]) Validate(ctx context.Context, entity *T) error {
	ev.mu.RLock()
	defer ev.mu.RUnlock()
	
	var validationErrors []string
	
	// Run all validation rules
	for name, rule := range ev.rules {
		if err := rule.Validate(ctx, entity); err != nil {
			if ev.strict {
				return fmt.Errorf("validation rule '%s' failed: %w", name, err)
			}
			validationErrors = append(validationErrors, fmt.Sprintf("%s: %s", name, err.Error()))
		}
	}
	
	// Return aggregated validation errors
	if len(validationErrors) > 0 {
		return repositories.NewValidationError(
			reflect.TypeOf(*entity).Name(),
			map[string]interface{}{
				"errors": validationErrors,
			},
		)
	}
	
	return nil
}

// ValidateBatch validates multiple entities
func (ev *EntityValidator[T]) ValidateBatch(ctx context.Context, entities []*T) ([]error, error) {
	var errors []error
	
	for i, entity := range entities {
		if err := ev.Validate(ctx, entity); err != nil {
			errors = append(errors, fmt.Errorf("entity %d: %w", i, err))
		} else {
			errors = append(errors, nil)
		}
	}
	
	// Check if any validation failed
	hasErrors := false
	for _, err := range errors {
		if err != nil {
			hasErrors = true
			break
		}
	}
	
	if hasErrors {
		return errors, fmt.Errorf("batch validation failed")
	}
	
	return errors, nil
}

// AddValidationRule adds a custom validation rule
func (ev *EntityValidator[T]) AddValidationRule(name string, rule repositories.ValidationRule[T]) error {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	
	ev.rules[name] = rule
	return nil
}

// RemoveValidationRule removes a validation rule
func (ev *EntityValidator[T]) RemoveValidationRule(name string) error {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	
	delete(ev.rules, name)
	return nil
}

// ListValidationRules returns the names of all registered validation rules
func (ev *EntityValidator[T]) ListValidationRules() []string {
	ev.mu.RLock()
	defer ev.mu.RUnlock()
	
	var names []string
	for name := range ev.rules {
		names = append(names, name)
	}
	
	return names
}

// addDefaultRules adds default validation rules
func (ev *EntityValidator[T]) addDefaultRules() {
	// Add common validation rules
	ev.rules["required_fields"] = &RequiredFieldsRule[T]{}
	ev.rules["field_lengths"] = &FieldLengthRule[T]{}
	ev.rules["email_format"] = &EmailFormatRule[T]{}
	ev.rules["url_format"] = &URLFormatRule[T]{}
}

// Default validation rule implementations

// RequiredFieldsRule validates that required fields are not empty
type RequiredFieldsRule[T any] struct{}

func (r *RequiredFieldsRule[T]) Validate(ctx context.Context, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// Check for required tag
		if tag := fieldType.Tag.Get("validate"); strings.Contains(tag, "required") {
			if r.isEmpty(field) {
				return fmt.Errorf("field '%s' is required", fieldType.Name)
			}
		}
	}
	
	return nil
}

func (r *RequiredFieldsRule[T]) Name() string {
	return "required_fields"
}

func (r *RequiredFieldsRule[T]) Description() string {
	return "Validates that required fields are not empty"
}

func (r *RequiredFieldsRule[T]) isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	default:
		return false
	}
}

// FieldLengthRule validates field lengths
type FieldLengthRule[T any] struct{}

func (r *FieldLengthRule[T]) Validate(ctx context.Context, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		if field.Kind() == reflect.String {
			tag := fieldType.Tag.Get("validate")
			if strings.Contains(tag, "max=") {
				// Extract max length from tag
				parts := strings.Split(tag, ",")
				for _, part := range parts {
					if strings.HasPrefix(part, "max=") {
						var maxLen int
						if _, err := fmt.Sscanf(part, "max=%d", &maxLen); err == nil {
							if field.Len() > maxLen {
								return fmt.Errorf("field '%s' exceeds maximum length of %d", fieldType.Name, maxLen)
							}
						}
					}
				}
			}
		}
	}
	
	return nil
}

func (r *FieldLengthRule[T]) Name() string {
	return "field_lengths"
}

func (r *FieldLengthRule[T]) Description() string {
	return "Validates field lengths against maximum constraints"
}

// EmailFormatRule validates email format
type EmailFormatRule[T any] struct{}

func (r *EmailFormatRule[T]) Validate(ctx context.Context, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()
	
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		if field.Kind() == reflect.String {
			tag := fieldType.Tag.Get("validate")
			if strings.Contains(tag, "email") {
				email := field.String()
				if email != "" && !emailRegex.MatchString(email) {
					return fmt.Errorf("field '%s' must be a valid email address", fieldType.Name)
				}
			}
		}
	}
	
	return nil
}

func (r *EmailFormatRule[T]) Name() string {
	return "email_format"
}

func (r *EmailFormatRule[T]) Description() string {
	return "Validates email address format"
}

// URLFormatRule validates URL format
type URLFormatRule[T any] struct{}

func (r *URLFormatRule[T]) Validate(ctx context.Context, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()
	
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		if field.Kind() == reflect.String {
			tag := fieldType.Tag.Get("validate")
			if strings.Contains(tag, "url") {
				url := field.String()
				if url != "" && !urlRegex.MatchString(url) {
					return fmt.Errorf("field '%s' must be a valid URL", fieldType.Name)
				}
			}
		}
	}
	
	return nil
}

func (r *URLFormatRule[T]) Name() string {
	return "url_format"
}

func (r *URLFormatRule[T]) Description() string {
	return "Validates URL format"
}

// Custom validation rule for business logic
type BusinessLogicRule[T any] struct {
	name        string
	description string
	validator   func(ctx context.Context, entity *T) error
}

// NewBusinessLogicRule creates a new business logic validation rule
func NewBusinessLogicRule[T any](name, description string, validator func(ctx context.Context, entity *T) error) *BusinessLogicRule[T] {
	return &BusinessLogicRule[T]{
		name:        name,
		description: description,
		validator:   validator,
	}
}

func (r *BusinessLogicRule[T]) Validate(ctx context.Context, entity *T) error {
	return r.validator(ctx, entity)
}

func (r *BusinessLogicRule[T]) Name() string {
	return r.name
}

func (r *BusinessLogicRule[T]) Description() string {
	return r.description
}