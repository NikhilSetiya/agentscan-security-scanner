package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
)

// parseUUID parses a string to UUID
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// parseIntQuery parses a string to int
func parseIntQuery(s string) (int, error) {
	return strconv.Atoi(s)
}

// parsePagination extracts pagination parameters from query string
func parsePagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := c.Query("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	return page, pageSize
}

// handleError handles different types of errors and returns appropriate HTTP responses
func handleError(c *gin.Context, err error) {
	switch e := err.(type) {
	case *entities.ValidationError:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": e.Message,
			"details": e.Details,
		})
	case *entities.NotFoundError:
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Resource not found",
			"message": e.Message,
		})
	case *entities.ConflictError:
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": e.Message,
		})
	case *entities.BusinessRuleError:
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "Business rule violation",
			"message": e.Message,
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "An unexpected error occurred",
		})
	}
}

// getUserFromContext extracts user ID from gin context
func getUserFromContext(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, entities.NewValidationError("user not authenticated")
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, entities.NewValidationError("invalid user context")
	}

	return uid, nil
}

// getOrganizationFromContext extracts organization ID from gin context
func getOrganizationFromContext(c *gin.Context) (uuid.UUID, error) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		return uuid.Nil, entities.NewValidationError("organization not found in context")
	}

	oid, ok := orgID.(uuid.UUID)
	if !ok {
		return uuid.Nil, entities.NewValidationError("invalid organization context")
	}

	return oid, nil
}

// validateRequiredParam validates that a required parameter is present
func validateRequiredParam(value, paramName string) error {
	if value == "" {
		return entities.NewValidationError("missing required parameter: " + paramName)
	}
	return nil
}

// successResponse creates a standardized success response
func successResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// createdResponse creates a standardized created response
func createdResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}

// messageResponse creates a standardized message response
func messageResponse(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// paginatedResponse creates a standardized paginated response
func paginatedResponse(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"meta": gin.H{
			"pagination": gin.H{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": totalPages,
				"has_next":    int64(page) < totalPages,
				"has_prev":    page > 1,
			},
		},
	})
}