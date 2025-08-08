package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/database"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	// Authentication events
	AuditEventLogin          AuditEventType = "auth.login"
	AuditEventLoginFailed    AuditEventType = "auth.login_failed"
	AuditEventLogout         AuditEventType = "auth.logout"
	AuditEventTokenRefresh   AuditEventType = "auth.token_refresh"
	AuditEventPasswordChange AuditEventType = "auth.password_change"

	// Authorization events
	AuditEventPermissionDenied AuditEventType = "authz.permission_denied"
	AuditEventRoleChanged      AuditEventType = "authz.role_changed"
	AuditEventAccessGranted    AuditEventType = "authz.access_granted"

	// User management events
	AuditEventUserCreated AuditEventType = "user.created"
	AuditEventUserUpdated AuditEventType = "user.updated"
	AuditEventUserDeleted AuditEventType = "user.deleted"

	// Organization events
	AuditEventOrgCreated      AuditEventType = "org.created"
	AuditEventOrgUpdated      AuditEventType = "org.updated"
	AuditEventOrgDeleted      AuditEventType = "org.deleted"
	AuditEventOrgMemberAdded  AuditEventType = "org.member_added"
	AuditEventOrgMemberRemoved AuditEventType = "org.member_removed"

	// Repository events
	AuditEventRepoCreated AuditEventType = "repo.created"
	AuditEventRepoUpdated AuditEventType = "repo.updated"
	AuditEventRepoDeleted AuditEventType = "repo.deleted"

	// Scan events
	AuditEventScanCreated   AuditEventType = "scan.created"
	AuditEventScanCancelled AuditEventType = "scan.cancelled"
	AuditEventScanCompleted AuditEventType = "scan.completed"

	// Finding events
	AuditEventFindingStatusChanged AuditEventType = "finding.status_changed"
	AuditEventFindingExported      AuditEventType = "finding.exported"
)

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	EventType      AuditEventType         `json:"event_type" db:"event_type"`
	UserID         *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty" db:"organization_id"`
	ResourceType   string                 `json:"resource_type,omitempty" db:"resource_type"`
	ResourceID     *uuid.UUID             `json:"resource_id,omitempty" db:"resource_id"`
	IPAddress      string                 `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      string                 `json:"user_agent,omitempty" db:"user_agent"`
	RequestID      string                 `json:"request_id,omitempty" db:"request_id"`
	Details        map[string]interface{} `json:"details,omitempty" db:"details"`
	Success        bool                   `json:"success" db:"success"`
	ErrorMessage   string                 `json:"error_message,omitempty" db:"error_message"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// AuditLogger handles audit logging
type AuditLogger struct {
	repos *database.Repositories
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(repos *database.Repositories) *AuditLogger {
	return &AuditLogger{
		repos: repos,
	}
}

// LogEvent logs an audit event
func (a *AuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.CreatedAt = time.Now()

	// TODO: Implement audit event storage in database
	// For now, we'll just log to stdout/structured logging
	_, _ = json.Marshal(event)
	// log.Info("Audit Event", "event", string(eventJSON))

	return nil
}

// LogAuthEvent logs an authentication event
func (a *AuditLogger) LogAuthEvent(ctx context.Context, eventType AuditEventType, userID *uuid.UUID, success bool, details map[string]interface{}, c *gin.Context) error {
	userAgent := ""
	if c != nil {
		userAgent = c.GetHeader("User-Agent")
	}
	
	event := &AuditEvent{
		EventType:    eventType,
		UserID:       userID,
		IPAddress:    a.getClientIP(c),
		UserAgent:    userAgent,
		RequestID:    a.getRequestID(c),
		Details:      details,
		Success:      success,
		ResourceType: "user",
	}

	if userID != nil {
		event.ResourceID = userID
	}

	return a.LogEvent(ctx, event)
}

// LogAuthzEvent logs an authorization event
func (a *AuditLogger) LogAuthzEvent(ctx context.Context, eventType AuditEventType, userID, orgID *uuid.UUID, resourceType string, resourceID *uuid.UUID, success bool, details map[string]interface{}, c *gin.Context) error {
	userAgent := ""
	if c != nil {
		userAgent = c.GetHeader("User-Agent")
	}
	
	event := &AuditEvent{
		EventType:      eventType,
		UserID:         userID,
		OrganizationID: orgID,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		IPAddress:      a.getClientIP(c),
		UserAgent:      userAgent,
		RequestID:      a.getRequestID(c),
		Details:        details,
		Success:        success,
	}

	return a.LogEvent(ctx, event)
}

// LogResourceEvent logs a resource-related event
func (a *AuditLogger) LogResourceEvent(ctx context.Context, eventType AuditEventType, userID, orgID *uuid.UUID, resourceType string, resourceID *uuid.UUID, details map[string]interface{}, c *gin.Context) error {
	userAgent := ""
	if c != nil {
		userAgent = c.GetHeader("User-Agent")
	}
	
	event := &AuditEvent{
		EventType:      eventType,
		UserID:         userID,
		OrganizationID: orgID,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		IPAddress:      a.getClientIP(c),
		UserAgent:      userAgent,
		RequestID:      a.getRequestID(c),
		Details:        details,
		Success:        true,
	}

	return a.LogEvent(ctx, event)
}

// AuditMiddleware creates middleware that logs requests
func (a *AuditLogger) AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log the request
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		userID, _ := GetCurrentUserID(c)
		
		details := map[string]interface{}{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
		}

		// Determine if this was successful
		success := statusCode < 400

		// Log different types of events based on the path
		var eventType AuditEventType
		switch {
		case c.Request.URL.Path == "/api/v1/auth/github/callback" || c.Request.URL.Path == "/api/v1/auth/gitlab/callback":
			if success {
				eventType = AuditEventLogin
			} else {
				eventType = AuditEventLoginFailed
			}
		case c.Request.URL.Path == "/api/v1/auth/logout":
			eventType = AuditEventLogout
		case c.Request.URL.Path == "/api/v1/user/refresh":
			eventType = AuditEventTokenRefresh
		default:
			// For other endpoints, we might not want to log everything
			return
		}

		a.LogAuthEvent(c.Request.Context(), eventType, &userID, success, details, c)
	}
}

// getClientIP extracts the client IP address from the request
func (a *AuditLogger) getClientIP(c *gin.Context) string {
	if c == nil {
		return "unknown"
	}
	
	// Check X-Forwarded-For header first
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}

// getRequestID extracts the request ID from the context
func (a *AuditLogger) getRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// AuditFilter represents filters for audit log queries
type AuditFilter struct {
	EventType      AuditEventType `json:"event_type,omitempty"`
	UserID         *uuid.UUID     `json:"user_id,omitempty"`
	OrganizationID *uuid.UUID     `json:"organization_id,omitempty"`
	ResourceType   string         `json:"resource_type,omitempty"`
	ResourceID     *uuid.UUID     `json:"resource_id,omitempty"`
	Success        *bool          `json:"success,omitempty"`
	Since          time.Time      `json:"since,omitempty"`
	Until          time.Time      `json:"until,omitempty"`
}

// GetAuditLogs retrieves audit logs with filtering
func (a *AuditLogger) GetAuditLogs(ctx context.Context, filter *AuditFilter, pagination *database.Pagination) ([]*AuditEvent, int64, error) {
	// TODO: Implement audit log retrieval from database
	// For now, return empty results
	return []*AuditEvent{}, 0, nil
}

// GetUserAuditLogs retrieves audit logs for a specific user
func (a *AuditLogger) GetUserAuditLogs(ctx context.Context, userID uuid.UUID, pagination *database.Pagination) ([]*AuditEvent, int64, error) {
	filter := &AuditFilter{
		UserID: &userID,
	}
	return a.GetAuditLogs(ctx, filter, pagination)
}

// GetOrganizationAuditLogs retrieves audit logs for a specific organization
func (a *AuditLogger) GetOrganizationAuditLogs(ctx context.Context, orgID uuid.UUID, pagination *database.Pagination) ([]*AuditEvent, int64, error) {
	filter := &AuditFilter{
		OrganizationID: &orgID,
	}
	return a.GetAuditLogs(ctx, filter, pagination)
}

// CleanupOldAuditLogs removes audit logs older than the specified duration
func (a *AuditLogger) CleanupOldAuditLogs(ctx context.Context, olderThan time.Duration) error {
	// TODO: Implement audit log cleanup
	return nil
}

// ExportAuditLogs exports audit logs in various formats
func (a *AuditLogger) ExportAuditLogs(ctx context.Context, filter *AuditFilter, format string) ([]byte, error) {
	logs, _, err := a.GetAuditLogs(ctx, filter, &database.Pagination{Page: 1, PageSize: 10000})
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.Marshal(logs)
	case "csv":
		// TODO: Implement CSV export
		return nil, fmt.Errorf("CSV export not implemented")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}