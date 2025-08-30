package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/adapters/http/validators"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/errors"
)

// ValidationMiddleware creates a middleware that adds input validation to the context
func ValidationMiddleware() gin.HandlerFunc {
	validator := validators.NewInputValidator()
	
	return func(c *gin.Context) {
		c.Set("validator", validator)
		c.Next()
	}
}

// ValidateJSON validates JSON request body using the validator from context
func ValidateJSON(obj interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bind JSON to struct
		if err := c.ShouldBindJSON(obj); err != nil {
			c.Error(errors.NewValidationError("invalid JSON format").WithCause(err))
			c.Abort()
			return
		}
		
		// Get validator from context
		validator := validators.GetValidator(c)
		
		// Validate and sanitize the input
		if err := validator.ValidateAndSanitize(obj); err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		
		// Store validated object in context
		c.Set("validated_input", obj)
		c.Next()
	}
}

// ValidateUUID validates UUID parameters and adds them to context
func ValidateUUID(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		paramValue := c.Param(paramName)
		if paramValue == "" {
			c.Error(errors.NewValidationError("missing required parameter: " + paramName))
			c.Abort()
			return
		}
		
		parsedUUID, err := uuid.Parse(paramValue)
		if err != nil {
			c.Error(errors.NewValidationError("invalid UUID format for parameter: " + paramName))
			c.Abort()
			return
		}
		
		// Store parsed UUID in context with a standardized key
		c.Set(paramName+"_uuid", parsedUUID)
		c.Next()
	}
}

// ValidateQuery validates query parameters
func ValidateQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		validator := validators.GetValidator(c)
		
		// Validate and sanitize query parameters
		for key, values := range c.Request.URL.Query() {
			for i, value := range values {
				sanitized := validator.SanitizeString(value)
				c.Request.URL.Query()[key][i] = sanitized
			}
		}
		
		c.Next()
	}
}

// GetValidatedInput retrieves the validated input from context
func GetValidatedInput(c *gin.Context) interface{} {
	if input, exists := c.Get("validated_input"); exists {
		return input
	}
	return nil
}

// GetUUIDParam retrieves a validated UUID parameter from context
func GetUUIDParam(c *gin.Context, paramName string) (uuid.UUID, bool) {
	if value, exists := c.Get(paramName + "_uuid"); exists {
		if uuid, ok := value.(uuid.UUID); ok {
			return uuid, true
		}
	}
	return uuid.Nil, false
}

// ValidateContentType validates that the request has the correct content type
func ValidateContentType(expectedType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		contentType := c.GetHeader("Content-Type")
		if contentType != expectedType {
			c.Error(errors.NewValidationError("invalid content type, expected: " + expectedType))
			c.Abort()
			return
		}
		c.Next()
	}
}

// ValidateRequestSize validates that the request body size is within limits
func ValidateRequestSize(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.Error(errors.NewValidationError("request body too large"))
			c.Abort()
			return
		}
		c.Next()
	}
}