package utils

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/agentscan/agentscan/internal/adapters/http/validators"
	"github.com/agentscan/agentscan/pkg/errors"
)

// ValidationHelper provides request validation utilities
type ValidationHelper struct {
	validator *validators.InputValidator
}

// NewValidationHelper creates a new validation helper
func NewValidationHelper() *ValidationHelper {
	return &ValidationHelper{
		validator: validators.NewInputValidator(),
	}
}

// ValidateAndBind validates and binds JSON request body
func (vh *ValidationHelper) ValidateAndBind(c *gin.Context, target interface{}) error {
	// Bind JSON to target struct
	if err := c.ShouldBindJSON(target); err != nil {
		return errors.NewValidationError("invalid JSON format").WithCause(err)
	}
	
	// Validate and sanitize the input
	if err := vh.validator.ValidateAndSanitize(target); err != nil {
		return err
	}
	
	return nil
}

// ValidateStruct validates a struct using the validator
func (vh *ValidationHelper) ValidateStruct(target interface{}) error {
	return vh.validator.ValidateAndSanitize(target)
}

// ValidateJSON validates JSON data without binding to Gin context
func (vh *ValidationHelper) ValidateJSON(jsonData []byte, target interface{}) error {
	// Unmarshal JSON
	if err := json.Unmarshal(jsonData, target); err != nil {
		return errors.NewValidationError("invalid JSON format").WithCause(err)
	}
	
	// Validate and sanitize
	return vh.validator.ValidateAndSanitize(target)
}

// ExtractValidationErrors extracts validation errors from validator errors
func (vh *ValidationHelper) ExtractValidationErrors(err error) map[string]interface{} {
	details := make(map[string]interface{})
	
	if validatorErrors, ok := err.(validator.ValidationErrors); ok {
		var fieldErrors []map[string]interface{}
		
		for _, fieldError := range validatorErrors {
			fieldError := map[string]interface{}{
				"field":   fieldError.Field(),
				"tag":     fieldError.Tag(),
				"value":   fieldError.Value(),
				"message": getValidationErrorMessage(fieldError),
			}
			fieldErrors = append(fieldErrors, fieldError)
		}
		
		details["validation_errors"] = fieldErrors
	} else {
		details["error"] = err.Error()
	}
	
	return details
}

// getValidationErrorMessage returns user-friendly error messages for validation failures
func getValidationErrorMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return fieldError.Field() + " is required"
	case "email":
		return fieldError.Field() + " must be a valid email address"
	case "min":
		return fieldError.Field() + " must be at least " + fieldError.Param() + " characters long"
	case "max":
		return fieldError.Field() + " must be at most " + fieldError.Param() + " characters long"
	case "url":
		return fieldError.Field() + " must be a valid URL"
	case "uuid":
		return fieldError.Field() + " must be a valid UUID"
	case "oneof":
		return fieldError.Field() + " must be one of: " + fieldError.Param()
	case "no_sql_injection":
		return fieldError.Field() + " contains potentially dangerous SQL patterns"
	case "no_xss":
		return fieldError.Field() + " contains potentially dangerous script content"
	case "safe_filename":
		return fieldError.Field() + " contains invalid filename characters"
	case "safe_url":
		return fieldError.Field() + " is not a safe URL"
	default:
		return fieldError.Field() + " is invalid"
	}
}

// ValidateRequiredFields validates that required fields are present and not empty
func ValidateRequiredFields(data map[string]interface{}, requiredFields []string) error {
	var missingFields []string
	
	for _, field := range requiredFields {
		value, exists := data[field]
		if !exists {
			missingFields = append(missingFields, field)
			continue
		}
		
		// Check if value is empty
		if isEmpty(value) {
			missingFields = append(missingFields, field)
		}
	}
	
	if len(missingFields) > 0 {
		return errors.NewValidationError("missing required fields").WithDetails(map[string]interface{}{
			"missing_fields": missingFields,
		})
	}
	
	return nil
}

// isEmpty checks if a value is considered empty
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return strings.TrimSpace(v.String()) == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// SanitizeString sanitizes a string input
func (vh *ValidationHelper) SanitizeString(input string) string {
	return vh.validator.SanitizeString(input)
}

// ValidateEnum validates that a value is in a list of allowed values
func ValidateEnum(value string, allowedValues []string, fieldName string) error {
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}
	
	return errors.NewValidationError("invalid enum value").WithDetails(map[string]interface{}{
		"field":          fieldName,
		"value":          value,
		"allowed_values": allowedValues,
	})
}

// ValidateStringLength validates string length constraints
func ValidateStringLength(value, fieldName string, minLength, maxLength int) error {
	length := len(value)
	
	if minLength > 0 && length < minLength {
		return errors.NewValidationError("string too short").WithDetails(map[string]interface{}{
			"field":      fieldName,
			"value":      value,
			"min_length": minLength,
			"actual":     length,
		})
	}
	
	if maxLength > 0 && length > maxLength {
		return errors.NewValidationError("string too long").WithDetails(map[string]interface{}{
			"field":      fieldName,
			"value":      value,
			"max_length": maxLength,
			"actual":     length,
		})
	}
	
	return nil
}

// ValidateRange validates that a numeric value is within a range
func ValidateRange(value int, fieldName string, min, max int) error {
	if min != 0 && value < min {
		return errors.NewValidationError("value too small").WithDetails(map[string]interface{}{
			"field":   fieldName,
			"value":   value,
			"minimum": min,
		})
	}
	
	if max != 0 && value > max {
		return errors.NewValidationError("value too large").WithDetails(map[string]interface{}{
			"field":   fieldName,
			"value":   value,
			"maximum": max,
		})
	}
	
	return nil
}

// Global validation helper instance
var GlobalValidationHelper = NewValidationHelper()

// Convenience functions using the global helper

// ValidateAndBindJSON validates and binds JSON request body using the global helper
func ValidateAndBindJSON(c *gin.Context, target interface{}) error {
	return GlobalValidationHelper.ValidateAndBind(c, target)
}

// ValidateStruct validates a struct using the global helper
func ValidateStruct(target interface{}) error {
	return GlobalValidationHelper.ValidateStruct(target)
}

// SanitizeString sanitizes a string using the global helper
func SanitizeString(input string) string {
	return GlobalValidationHelper.SanitizeString(input)
}