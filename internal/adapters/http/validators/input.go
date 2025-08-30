package validators

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/microcosm-cc/bluemonday"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
)

// InputValidator provides comprehensive input validation and sanitization
type InputValidator struct {
	validator *validator.Validate
	sanitizer *bluemonday.Policy
}

// ValidationError represents a validation error with field details
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// NewInputValidator creates a new input validator with security policies
func NewInputValidator() *InputValidator {
	v := validator.New()
	
	// Register custom validation tags
	v.RegisterValidation("no_sql_injection", validateNoSQLInjection)
	v.RegisterValidation("safe_string", validateSafeString)
	v.RegisterValidation("repository_url", validateRepositoryURL)
	v.RegisterValidation("scan_type", validateScanType)
	v.RegisterValidation("severity", validateSeverity)
	
	// Create strict sanitization policy
	sanitizer := bluemonday.StrictPolicy()
	
	return &InputValidator{
		validator: v,
		sanitizer: sanitizer,
	}
}

// ValidateAndSanitize validates and sanitizes input struct
func (iv *InputValidator) ValidateAndSanitize(input interface{}) error {
	// First sanitize all string fields
	if err := iv.sanitizeStruct(input); err != nil {
		return errors.NewValidationError("sanitization failed").WithCause(err)
	}
	
	// Then validate the sanitized input
	if err := iv.validator.Struct(input); err != nil {
		return iv.formatValidationErrors(err)
	}
	
	return nil
}

// SanitizeString sanitizes a single string input
func (iv *InputValidator) SanitizeString(input string) string {
	// Remove HTML tags and potentially dangerous content
	sanitized := iv.sanitizer.Sanitize(input)
	
	// Additional sanitization for common attack vectors
	sanitized = strings.TrimSpace(sanitized)
	sanitized = iv.removeControlCharacters(sanitized)
	
	return sanitized
}

// ValidateStruct validates a struct without sanitization
func (iv *InputValidator) ValidateStruct(input interface{}) error {
	if err := iv.validator.Struct(input); err != nil {
		return iv.formatValidationErrors(err)
	}
	return nil
}

// sanitizeStruct recursively sanitizes all string fields in a struct
func (iv *InputValidator) sanitizeStruct(input interface{}) error {
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return nil
	}
	
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// Skip unexported fields
		if !field.CanSet() {
			continue
		}
		
		switch field.Kind() {
		case reflect.String:
			if field.String() != "" {
				sanitized := iv.SanitizeString(field.String())
				field.SetString(sanitized)
			}
		case reflect.Ptr:
			if !field.IsNil() && field.Elem().Kind() == reflect.String {
				original := field.Elem().String()
				if original != "" {
					sanitized := iv.SanitizeString(original)
					field.Elem().SetString(sanitized)
				}
			}
		case reflect.Struct:
			if err := iv.sanitizeStruct(field.Addr().Interface()); err != nil {
				return fmt.Errorf("failed to sanitize field %s: %w", fieldType.Name, err)
			}
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					elem := field.Index(j)
					if elem.String() != "" {
						sanitized := iv.SanitizeString(elem.String())
						elem.SetString(sanitized)
					}
				}
			}
		}
	}
	
	return nil
}

// formatValidationErrors converts validator errors to our error format
func (iv *InputValidator) formatValidationErrors(err error) error {
	var validationErrors []ValidationError
	
	if validatorErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validatorErrors {
			validationError := ValidationError{
				Field:   fieldError.Field(),
				Tag:     fieldError.Tag(),
				Value:   fmt.Sprintf("%v", fieldError.Value()),
				Message: iv.getErrorMessage(fieldError),
			}
			validationErrors = append(validationErrors, validationError)
		}
	}
	
	details := make(map[string]interface{})
	details["validation_errors"] = validationErrors
	
	return errors.NewValidationError("input validation failed").WithDetails(details)
}

// getErrorMessage returns user-friendly error messages for validation tags
func (iv *InputValidator) getErrorMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fieldError.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fieldError.Field())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fieldError.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", fieldError.Field(), fieldError.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", fieldError.Field(), fieldError.Param())
	case "no_sql_injection":
		return fmt.Sprintf("%s contains potentially dangerous content", fieldError.Field())
	case "safe_string":
		return fmt.Sprintf("%s contains invalid characters", fieldError.Field())
	case "repository_url":
		return fmt.Sprintf("%s must be a valid repository URL", fieldError.Field())
	case "scan_type":
		return fmt.Sprintf("%s must be one of: full, incremental, ide", fieldError.Field())
	case "severity":
		return fmt.Sprintf("%s must be one of: critical, high, medium, low, info", fieldError.Field())
	default:
		return fmt.Sprintf("%s is invalid", fieldError.Field())
	}
}

// removeControlCharacters removes control characters that could be used in attacks
func (iv *InputValidator) removeControlCharacters(input string) string {
	// Remove control characters except tab, newline, and carriage return
	controlChars := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	return controlChars.ReplaceAllString(input, "")
}

// Custom validation functions

// validateNoSQLInjection checks for common SQL injection patterns
func validateNoSQLInjection(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	
	// Common SQL injection patterns
	sqlInjectionPatterns := []string{
		`(?i)(union\s+select)`,
		`(?i)(drop\s+table)`,
		`(?i)(delete\s+from)`,
		`(?i)(insert\s+into)`,
		`(?i)(update\s+\w+\s+set)`,
		`(?i)(exec\s*\()`,
		`(?i)(execute\s*\()`,
		`(?i)(sp_\w+)`,
		`(?i)(xp_\w+)`,
		`(?i)(\bor\s+1\s*=\s*1)`,
		`(?i)(\band\s+1\s*=\s*1)`,
		`(?i)(--\s*$)`,
		`(?i)(/\*.*\*/)`,
		`[\x00\x1a]`, // Null bytes and substitute characters
	}
	
	for _, pattern := range sqlInjectionPatterns {
		matched, _ := regexp.MatchString(pattern, value)
		if matched {
			return false
		}
	}
	
	return true
}

// validateSafeString checks for safe string content
func validateSafeString(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	
	// Allow alphanumeric, spaces, and common safe punctuation
	safePattern := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,!?@#$%^&*()+=\[\]{}|\\:";'<>?/~` + "`" + `]*$`)
	return safePattern.MatchString(value)
}

// validateRepositoryURL validates repository URL format
func validateRepositoryURL(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	
	// Valid repository URL patterns
	repoPatterns := []string{
		`^https://github\.com/[\w\-\.]+/[\w\-\.]+(?:\.git)?/?$`,
		`^https://gitlab\.com/[\w\-\.]+/[\w\-\.]+(?:\.git)?/?$`,
		`^https://bitbucket\.org/[\w\-\.]+/[\w\-\.]+(?:\.git)?/?$`,
		`^git@github\.com:[\w\-\.]+/[\w\-\.]+\.git$`,
		`^git@gitlab\.com:[\w\-\.]+/[\w\-\.]+\.git$`,
	}
	
	for _, pattern := range repoPatterns {
		matched, _ := regexp.MatchString(pattern, value)
		if matched {
			return true
		}
	}
	
	return false
}

// validateScanType validates scan type values
func validateScanType(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validTypes := []string{"full", "incremental", "ide"}
	
	for _, validType := range validTypes {
		if value == validType {
			return true
		}
	}
	
	return false
}

// validateSeverity validates severity values
func validateSeverity(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	validSeverities := []string{"critical", "high", "medium", "low", "info"}
	
	for _, validSeverity := range validSeverities {
		if value == validSeverity {
			return true
		}
	}
	
	return false
}

// ValidationMiddleware creates a Gin middleware for input validation
func ValidationMiddleware(validator *InputValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store validator in context for use in handlers
		c.Set("validator", validator)
		c.Next()
	}
}

// GetValidator retrieves the validator from Gin context
func GetValidator(c *gin.Context) *InputValidator {
	if validator, exists := c.Get("validator"); exists {
		if v, ok := validator.(*InputValidator); ok {
			return v
		}
	}
	// Fallback to new validator if not found in context
	return NewInputValidator()
}