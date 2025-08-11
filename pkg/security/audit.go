package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/agentscan/agentscan/pkg/logging"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	// Authentication events
	EventTypeLogin          AuditEventType = "auth.login"
	EventTypeLogout         AuditEventType = "auth.logout"
	EventTypeLoginFailed    AuditEventType = "auth.login_failed"
	EventTypeTokenRefresh   AuditEventType = "auth.token_refresh"
	EventTypePasswordChange AuditEventType = "auth.password_change"

	// Authorization events
	EventTypeAccessGranted AuditEventType = "authz.access_granted"
	EventTypeAccessDenied  AuditEventType = "authz.access_denied"
	EventTypeRoleChanged   AuditEventType = "authz.role_changed"

	// Data access events
	EventTypeDataRead   AuditEventType = "data.read"
	EventTypeDataWrite  AuditEventType = "data.write"
	EventTypeDataDelete AuditEventType = "data.delete"
	EventTypeDataExport AuditEventType = "data.export"

	// Scan events
	EventTypeScanStarted   AuditEventType = "scan.started"
	EventTypeScanCompleted AuditEventType = "scan.completed"
	EventTypeScanFailed    AuditEventType = "scan.failed"
	EventTypeScanCancelled AuditEventType = "scan.cancelled"

	// Configuration events
	EventTypeConfigChanged AuditEventType = "config.changed"
	EventTypeUserCreated   AuditEventType = "user.created"
	EventTypeUserDeleted   AuditEventType = "user.deleted"
	EventTypeUserModified  AuditEventType = "user.modified"

	// Security events
	EventTypeSecurityViolation AuditEventType = "security.violation"
	EventTypeRateLimitExceeded AuditEventType = "security.rate_limit_exceeded"
	EventTypeSuspiciousActivity AuditEventType = "security.suspicious_activity"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   AuditEventType         `json:"event_type"`
	UserID      string                 `json:"user_id,omitempty"`
	Username    string                 `json:"username,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Result      string                 `json:"result"` // success, failure, denied
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	ServiceName string                 `json:"service_name"`
	Version     string                 `json:"version"`
}

// AuditLogger handles audit logging with compliance requirements
type AuditLogger struct {
	logger          *logging.Logger
	encryptionSvc   *EncryptionService
	serviceName     string
	version         string
	retentionPeriod time.Duration
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(serviceName, version string, encryptionSvc *EncryptionService) *AuditLogger {
	return &AuditLogger{
		logger:          logging.GetLogger(),
		encryptionSvc:   encryptionSvc,
		serviceName:     serviceName,
		version:         version,
		retentionPeriod: 7 * 365 * 24 * time.Hour, // 7 years for compliance
	}
}

// LogEvent logs an audit event
func (a *AuditLogger) LogEvent(ctx context.Context, event AuditEvent) error {
	// Set default fields
	if event.ID == "" {
		token, err := a.encryptionSvc.GenerateSecureToken(16)
		if err != nil {
			return fmt.Errorf("failed to generate event ID: %w", err)
		}
		event.ID = token
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	event.ServiceName = a.serviceName
	event.Version = a.version

	// Extract request ID from context if available
	if requestID := logging.GetCorrelationID(ctx); requestID != "" {
		event.RequestID = requestID
	}

	// Encrypt sensitive details
	if event.Details != nil {
		encryptedDetails, err := a.encryptionSvc.EncryptSensitiveFields(event.Details)
		if err != nil {
			a.logger.Error("Failed to encrypt audit event details",
				"error", err,
				"event_id", event.ID,
				"event_type", event.EventType,
			)
			// Continue without encryption rather than failing
		} else {
			event.Details = encryptedDetails
		}
	}

	// Log the audit event
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	a.logger.Info("AUDIT_EVENT",
		"audit_event", string(eventJSON),
		"event_id", event.ID,
		"event_type", string(event.EventType),
		"user_id", event.UserID,
		"result", event.Result,
		"resource", event.Resource,
		"action", event.Action,
	)

	return nil
}

// LogAuthenticationEvent logs authentication-related events
func (a *AuditLogger) LogAuthenticationEvent(ctx context.Context, eventType AuditEventType, userID, username string, success bool, details map[string]interface{}) error {
	result := "success"
	if !success {
		result = "failure"
	}

	event := AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Username:  username,
		Result:    result,
		Details:   details,
	}

	// Extract IP and User-Agent from HTTP request if available
	if req := getHTTPRequestFromContext(ctx); req != nil {
		event.IPAddress = getClientIP(req)
		event.UserAgent = req.UserAgent()
	}

	return a.LogEvent(ctx, event)
}

// LogAuthorizationEvent logs authorization-related events
func (a *AuditLogger) LogAuthorizationEvent(ctx context.Context, userID, username, resource, action string, granted bool, details map[string]interface{}) error {
	result := "granted"
	eventType := EventTypeAccessGranted
	if !granted {
		result = "denied"
		eventType = EventTypeAccessDenied
	}

	event := AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Username:  username,
		Resource:  resource,
		Action:    action,
		Result:    result,
		Details:   details,
	}

	// Extract IP from HTTP request if available
	if req := getHTTPRequestFromContext(ctx); req != nil {
		event.IPAddress = getClientIP(req)
		event.UserAgent = req.UserAgent()
	}

	return a.LogEvent(ctx, event)
}

// LogDataAccessEvent logs data access events
func (a *AuditLogger) LogDataAccessEvent(ctx context.Context, eventType AuditEventType, userID, username, resource string, details map[string]interface{}) error {
	event := AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Username:  username,
		Resource:  resource,
		Result:    "success",
		Details:   details,
	}

	// Extract IP from HTTP request if available
	if req := getHTTPRequestFromContext(ctx); req != nil {
		event.IPAddress = getClientIP(req)
		event.UserAgent = req.UserAgent()
	}

	return a.LogEvent(ctx, event)
}

// LogScanEvent logs scan-related events
func (a *AuditLogger) LogScanEvent(ctx context.Context, eventType AuditEventType, userID, username, scanID, repoURL string, details map[string]interface{}) error {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["scan_id"] = scanID
	details["repository_url"] = repoURL

	event := AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Username:  username,
		Resource:  fmt.Sprintf("scan:%s", scanID),
		Action:    string(eventType),
		Result:    "success",
		Details:   details,
	}

	return a.LogEvent(ctx, event)
}

// LogSecurityEvent logs security-related events
func (a *AuditLogger) LogSecurityEvent(ctx context.Context, eventType AuditEventType, message string, details map[string]interface{}) error {
	event := AuditEvent{
		EventType: eventType,
		Message:   message,
		Result:    "violation",
		Details:   details,
	}

	// Extract IP from HTTP request if available
	if req := getHTTPRequestFromContext(ctx); req != nil {
		event.IPAddress = getClientIP(req)
		event.UserAgent = req.UserAgent()
	}

	return a.LogEvent(ctx, event)
}

// LogConfigurationEvent logs configuration change events
func (a *AuditLogger) LogConfigurationEvent(ctx context.Context, eventType AuditEventType, userID, username, configKey string, oldValue, newValue interface{}) error {
	details := map[string]interface{}{
		"config_key": configKey,
		"old_value":  oldValue,
		"new_value":  newValue,
	}

	event := AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Username:  username,
		Resource:  fmt.Sprintf("config:%s", configKey),
		Action:    "modify",
		Result:    "success",
		Details:   details,
	}

	return a.LogEvent(ctx, event)
}

// getHTTPRequestFromContext extracts HTTP request from context
func getHTTPRequestFromContext(ctx context.Context) *http.Request {
	if req, ok := ctx.Value("http_request").(*http.Request); ok {
		return req
	}
	return nil
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := len(xff); idx > 0 {
			if commaIdx := 0; commaIdx < idx {
				for i, c := range xff {
					if c == ',' {
						commaIdx = i
						break
					}
				}
				if commaIdx > 0 {
					return xff[:commaIdx]
				}
			}
			return xff
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// ComplianceReport generates a compliance report for audit events
type ComplianceReport struct {
	StartDate    time.Time              `json:"start_date"`
	EndDate      time.Time              `json:"end_date"`
	TotalEvents  int                    `json:"total_events"`
	EventsByType map[AuditEventType]int `json:"events_by_type"`
	UserActivity map[string]int         `json:"user_activity"`
	SecurityEvents []AuditEvent         `json:"security_events"`
	GeneratedAt  time.Time              `json:"generated_at"`
}

// GenerateComplianceReport generates a compliance report (placeholder for database integration)
func (a *AuditLogger) GenerateComplianceReport(startDate, endDate time.Time) (*ComplianceReport, error) {
	// This would typically query a database for audit events
	// For now, return a placeholder structure
	report := &ComplianceReport{
		StartDate:      startDate,
		EndDate:        endDate,
		EventsByType:   make(map[AuditEventType]int),
		UserActivity:   make(map[string]int),
		SecurityEvents: make([]AuditEvent, 0),
		GeneratedAt:    time.Now().UTC(),
	}

	return report, nil
}