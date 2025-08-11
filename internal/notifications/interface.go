package notifications

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// NotificationService defines the interface for sending notifications
type NotificationService interface {
	// SendScanCompleted sends notification when a scan is completed
	SendScanCompleted(ctx context.Context, notification ScanCompletedNotification) error
	
	// SendCriticalFinding sends notification for critical security findings
	SendCriticalFinding(ctx context.Context, notification CriticalFindingNotification) error
	
	// SendScanFailed sends notification when a scan fails
	SendScanFailed(ctx context.Context, notification ScanFailedNotification) error
	
	// TestConnection tests the notification channel connectivity
	TestConnection(ctx context.Context, channel NotificationChannel) error
	
	// GetSupportedChannels returns list of supported notification channels
	GetSupportedChannels() []NotificationChannelType
}

// NotificationChannel represents a notification destination
type NotificationChannel struct {
	ID          uuid.UUID               `json:"id" db:"id"`
	UserID      uuid.UUID               `json:"user_id" db:"user_id"`
	OrgID       *uuid.UUID              `json:"org_id,omitempty" db:"org_id"`
	Type        NotificationChannelType `json:"type" db:"type"`
	Name        string                  `json:"name" db:"name"`
	Config      ChannelConfig           `json:"config" db:"config"`
	Enabled     bool                    `json:"enabled" db:"enabled"`
	Preferences NotificationPreferences `json:"preferences" db:"preferences"`
	CreatedAt   time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at" db:"updated_at"`
}

// NotificationChannelType represents the type of notification channel
type NotificationChannelType string

const (
	ChannelTypeSlack  NotificationChannelType = "slack"
	ChannelTypeTeams  NotificationChannelType = "teams"
	ChannelTypeEmail  NotificationChannelType = "email"
	ChannelTypeWebhook NotificationChannelType = "webhook"
)

// ChannelConfig contains channel-specific configuration
type ChannelConfig struct {
	// Slack configuration
	SlackWebhookURL string `json:"slack_webhook_url,omitempty"`
	SlackChannel    string `json:"slack_channel,omitempty"`
	SlackUsername   string `json:"slack_username,omitempty"`
	
	// Teams configuration
	TeamsWebhookURL string `json:"teams_webhook_url,omitempty"`
	
	// Email configuration
	EmailAddress string `json:"email_address,omitempty"`
	SMTPServer   string `json:"smtp_server,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUsername string `json:"smtp_username,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	
	// Webhook configuration
	WebhookURL    string            `json:"webhook_url,omitempty"`
	WebhookSecret string            `json:"webhook_secret,omitempty"`
	WebhookHeaders map[string]string `json:"webhook_headers,omitempty"`
}

// NotificationPreferences defines when to send notifications
type NotificationPreferences struct {
	ScanCompleted    bool     `json:"scan_completed"`
	ScanFailed       bool     `json:"scan_failed"`
	CriticalFindings bool     `json:"critical_findings"`
	MinSeverity      string   `json:"min_severity"`      // high, medium, low
	Repositories     []string `json:"repositories"`      // specific repos to notify for
	TimeWindow       TimeWindow `json:"time_window"`     // when to send notifications
}

// TimeWindow defines when notifications should be sent
type TimeWindow struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"` // HH:MM format
	EndTime   string `json:"end_time"`   // HH:MM format
	Timezone  string `json:"timezone"`   // IANA timezone
	Weekdays  []int  `json:"weekdays"`   // 0=Sunday, 1=Monday, etc.
}

// ScanCompletedNotification contains data for scan completion notifications
type ScanCompletedNotification struct {
	ScanID       uuid.UUID `json:"scan_id"`
	Repository   string    `json:"repository"`
	Branch       string    `json:"branch"`
	Commit       string    `json:"commit"`
	Status       string    `json:"status"`
	Duration     time.Duration `json:"duration"`
	FindingsCount FindingsCount `json:"findings_count"`
	DashboardURL string    `json:"dashboard_url"`
	UserID       uuid.UUID `json:"user_id"`
	OrgID        *uuid.UUID `json:"org_id,omitempty"`
}

// CriticalFindingNotification contains data for critical finding notifications
type CriticalFindingNotification struct {
	ScanID       uuid.UUID `json:"scan_id"`
	Repository   string    `json:"repository"`
	Branch       string    `json:"branch"`
	Commit       string    `json:"commit"`
	Finding      CriticalFinding `json:"finding"`
	DashboardURL string    `json:"dashboard_url"`
	UserID       uuid.UUID `json:"user_id"`
	OrgID        *uuid.UUID `json:"org_id,omitempty"`
}

// ScanFailedNotification contains data for scan failure notifications
type ScanFailedNotification struct {
	ScanID       uuid.UUID `json:"scan_id"`
	Repository   string    `json:"repository"`
	Branch       string    `json:"branch"`
	Commit       string    `json:"commit"`
	Error        string    `json:"error"`
	Duration     time.Duration `json:"duration"`
	DashboardURL string    `json:"dashboard_url"`
	UserID       uuid.UUID `json:"user_id"`
	OrgID        *uuid.UUID `json:"org_id,omitempty"`
}

// FindingsCount represents the count of findings by severity
type FindingsCount struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
	Total  int `json:"total"`
}

// CriticalFinding represents a critical security finding
type CriticalFinding struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Tool        string `json:"tool"`
	Description string `json:"description"`
}

// NotificationTemplate represents a notification message template
type NotificationTemplate struct {
	Subject  string `json:"subject"`
	Body     string `json:"body"`
	Format   string `json:"format"` // text, html, markdown
}

// NotificationEvent represents a notification event for audit logging
type NotificationEvent struct {
	ID          uuid.UUID               `json:"id"`
	ChannelID   uuid.UUID               `json:"channel_id"`
	Type        NotificationEventType   `json:"type"`
	Status      NotificationStatus      `json:"status"`
	Message     string                  `json:"message"`
	Error       string                  `json:"error,omitempty"`
	Metadata    MetadataMap             `json:"metadata,omitempty"`
	CreatedAt   time.Time               `json:"created_at"`
}

// NotificationEventType represents the type of notification event
type NotificationEventType string

const (
	EventTypeScanCompleted    NotificationEventType = "scan_completed"
	EventTypeCriticalFinding  NotificationEventType = "critical_finding"
	EventTypeScanFailed       NotificationEventType = "scan_failed"
	EventTypeTest             NotificationEventType = "test"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSent      NotificationStatus = "sent"
	StatusFailed    NotificationStatus = "failed"
	StatusSkipped   NotificationStatus = "skipped"
)

// NotificationFilter defines filtering criteria for notifications
type NotificationFilter struct {
	UserID       *uuid.UUID              `json:"user_id,omitempty"`
	OrgID        *uuid.UUID              `json:"org_id,omitempty"`
	ChannelType  *NotificationChannelType `json:"channel_type,omitempty"`
	EventType    *NotificationEventType   `json:"event_type,omitempty"`
	Status       *NotificationStatus      `json:"status,omitempty"`
	Repository   *string                  `json:"repository,omitempty"`
	MinSeverity  *string                  `json:"min_severity,omitempty"`
	DateFrom     *time.Time               `json:"date_from,omitempty"`
	DateTo       *time.Time               `json:"date_to,omitempty"`
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	TotalSent     int64                              `json:"total_sent"`
	TotalFailed   int64                              `json:"total_failed"`
	ByChannel     map[NotificationChannelType]int64  `json:"by_channel"`
	ByEventType   map[NotificationEventType]int64    `json:"by_event_type"`
	RecentEvents  []NotificationEvent                `json:"recent_events"`
	LastUpdated   time.Time                          `json:"last_updated"`
}

// MetadataMap is a wrapper type for JSON serialization
type MetadataMap map[string]interface{}