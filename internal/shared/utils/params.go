package utils

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/pkg/errors"
)

// ParseUUIDParam extracts and validates a UUID parameter from the URL
func ParseUUIDParam(c *gin.Context, paramName string) (uuid.UUID, error) {
	paramValue := c.Param(paramName)
	if paramValue == "" {
		return uuid.Nil, errors.NewValidationError("missing required parameter").WithDetails(map[string]interface{}{
			"parameter": paramName,
		})
	}

	parsedUUID, err := uuid.Parse(paramValue)
	if err != nil {
		return uuid.Nil, errors.NewValidationError("invalid UUID format").WithDetails(map[string]interface{}{
			"parameter": paramName,
			"value":     paramValue,
		})
	}

	return parsedUUID, nil
}

// ParseOptionalUUIDParam extracts and validates an optional UUID parameter
func ParseOptionalUUIDParam(c *gin.Context, paramName string) (*uuid.UUID, error) {
	paramValue := c.Param(paramName)
	if paramValue == "" {
		return nil, nil
	}

	parsedUUID, err := uuid.Parse(paramValue)
	if err != nil {
		return nil, errors.NewValidationError("invalid UUID format").WithDetails(map[string]interface{}{
			"parameter": paramName,
			"value":     paramValue,
		})
	}

	return &parsedUUID, nil
}

// ParseUUIDQuery extracts and validates a UUID from query parameters
func ParseUUIDQuery(c *gin.Context, queryName string) (*uuid.UUID, error) {
	queryValue := c.Query(queryName)
	if queryValue == "" {
		return nil, nil
	}

	parsedUUID, err := uuid.Parse(queryValue)
	if err != nil {
		return nil, errors.NewValidationError("invalid UUID format").WithDetails(map[string]interface{}{
			"query":     queryName,
			"value":     queryValue,
		})
	}

	return &parsedUUID, nil
}

// ParseIntParam extracts and validates an integer parameter
func ParseIntParam(c *gin.Context, paramName string) (int, error) {
	paramValue := c.Param(paramName)
	if paramValue == "" {
		return 0, errors.NewValidationError("missing required parameter").WithDetails(map[string]interface{}{
			"parameter": paramName,
		})
	}

	intValue, err := strconv.Atoi(paramValue)
	if err != nil {
		return 0, errors.NewValidationError("invalid integer format").WithDetails(map[string]interface{}{
			"parameter": paramName,
			"value":     paramValue,
		})
	}

	return intValue, nil
}

// ParseOptionalIntQuery extracts and validates an optional integer from query parameters
func ParseOptionalIntQuery(c *gin.Context, queryName string, defaultValue int) int {
	queryValue := c.Query(queryName)
	if queryValue == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(queryValue)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// ParseBoolQuery extracts and validates a boolean from query parameters
func ParseBoolQuery(c *gin.Context, queryName string, defaultValue bool) bool {
	queryValue := c.Query(queryName)
	if queryValue == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(queryValue)
	if err != nil {
		return defaultValue
	}

	return boolValue
}

// ParseStringQuery extracts and sanitizes a string from query parameters
func ParseStringQuery(c *gin.Context, queryName string, maxLength int) string {
	queryValue := c.Query(queryName)
	if queryValue == "" {
		return ""
	}

	// Trim whitespace
	queryValue = strings.TrimSpace(queryValue)

	// Limit length
	if maxLength > 0 && len(queryValue) > maxLength {
		queryValue = queryValue[:maxLength]
	}

	return queryValue
}

// ParseEnumQuery extracts and validates an enum value from query parameters
func ParseEnumQuery(c *gin.Context, queryName string, validValues []string, defaultValue string) string {
	queryValue := c.Query(queryName)
	if queryValue == "" {
		return defaultValue
	}

	// Check if value is in valid list
	for _, validValue := range validValues {
		if queryValue == validValue {
			return queryValue
		}
	}

	return defaultValue
}

// FilterParams represents common filtering parameters
type FilterParams struct {
	Search    string `json:"search,omitempty"`
	Status    string `json:"status,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// ParseFilterParams extracts common filter parameters from query
func ParseFilterParams(c *gin.Context) FilterParams {
	return FilterParams{
		Search:    ParseStringQuery(c, "search", 255),
		Status:    ParseStringQuery(c, "status", 50),
		SortBy:    ParseEnumQuery(c, "sort_by", []string{"created_at", "updated_at", "name", "status"}, "created_at"),
		SortOrder: ParseEnumQuery(c, "sort_order", []string{"asc", "desc"}, "desc"),
	}
}

// ValidateRequiredParams validates that all required parameters are present
func ValidateRequiredParams(c *gin.Context, paramNames []string) error {
	var missingParams []string

	for _, paramName := range paramNames {
		if c.Param(paramName) == "" {
			missingParams = append(missingParams, paramName)
		}
	}

	if len(missingParams) > 0 {
		return errors.NewValidationError("missing required parameters").WithDetails(map[string]interface{}{
			"missing_parameters": missingParams,
		})
	}

	return nil
}

// ValidateRequiredQueries validates that all required query parameters are present
func ValidateRequiredQueries(c *gin.Context, queryNames []string) error {
	var missingQueries []string

	for _, queryName := range queryNames {
		if c.Query(queryName) == "" {
			missingQueries = append(missingQueries, queryName)
		}
	}

	if len(missingQueries) > 0 {
		return errors.NewValidationError("missing required query parameters").WithDetails(map[string]interface{}{
			"missing_queries": missingQueries,
		})
	}

	return nil
}

// ParseMultipleUUIDs parses a comma-separated list of UUIDs from query parameter
func ParseMultipleUUIDs(c *gin.Context, queryName string) ([]uuid.UUID, error) {
	queryValue := c.Query(queryName)
	if queryValue == "" {
		return nil, nil
	}

	uuidStrings := strings.Split(queryValue, ",")
	var uuids []uuid.UUID

	for _, uuidStr := range uuidStrings {
		uuidStr = strings.TrimSpace(uuidStr)
		if uuidStr == "" {
			continue
		}

		parsedUUID, err := uuid.Parse(uuidStr)
		if err != nil {
			return nil, errors.NewValidationError("invalid UUID in list").WithDetails(map[string]interface{}{
				"query":        queryName,
				"invalid_uuid": uuidStr,
			})
		}

		uuids = append(uuids, parsedUUID)
	}

	return uuids, nil
}

// ParseMultipleStrings parses a comma-separated list of strings from query parameter
func ParseMultipleStrings(c *gin.Context, queryName string, maxItems int) []string {
	queryValue := c.Query(queryName)
	if queryValue == "" {
		return nil
	}

	strings := strings.Split(queryValue, ",")
	var result []string

	for i, str := range strings {
		if maxItems > 0 && i >= maxItems {
			break
		}

		str = strings.TrimSpace(str)
		if str != "" {
			result = append(result, str)
		}
	}

	return result
}