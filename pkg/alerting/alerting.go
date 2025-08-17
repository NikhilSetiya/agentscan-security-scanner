package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/logging"
)

// Severity represents alert severity levels
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeverityFatal    Severity = "fatal"
)

// Alert represents an alert
type Alert struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    Severity          `json:"severity"`
	Component   string            `json:"component"`
	Timestamp   time.Time         `json:"timestamp"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Resolved    bool              `json:"resolved"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
}

// AlertRule represents an alerting rule
type AlertRule struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Condition   AlertCondition    `json:"condition"`
	Severity    Severity          `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Enabled     bool              `json:"enabled"`
}

// AlertCondition represents conditions for triggering alerts
type AlertCondition struct {
	MetricName  string  `json:"metric_name"`
	Operator    string  `json:"operator"` // >, <, >=, <=, ==, !=
	Threshold   float64 `json:"threshold"`
	Duration    string  `json:"duration"`
	Aggregation string  `json:"aggregation"` // avg, sum, min, max, count
}

// NotificationChannel represents a notification channel
type NotificationChannel interface {
	Send(ctx context.Context, alert *Alert) error
	Name() string
}

// Service provides alerting functionality
type Service struct {
	channels    []NotificationChannel
	rules       map[string]*AlertRule
	activeAlerts map[string]*Alert
	logger      *logging.Logger
	mutex       sync.RWMutex
	config      *Config
}

// Config holds alerting configuration
type Config struct {
	Enabled           bool          `json:"enabled"`
	DefaultSeverity   Severity      `json:"default_severity"`
	AlertTimeout      time.Duration `json:"alert_timeout"`
	ResolutionTimeout time.Duration `json:"resolution_timeout"`
	MaxAlerts         int           `json:"max_alerts"`
}

// DefaultConfig returns default alerting configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:           true,
		DefaultSeverity:   SeverityWarning,
		AlertTimeout:      5 * time.Minute,
		ResolutionTimeout: 15 * time.Minute,
		MaxAlerts:         1000,
	}
}

// NewService creates a new alerting service
func NewService(logger *logging.Logger, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		channels:     make([]NotificationChannel, 0),
		rules:        make(map[string]*AlertRule),
		activeAlerts: make(map[string]*Alert),
		logger:       logger,
		config:       config,
	}
}

// AddChannel adds a notification channel
func (s *Service) AddChannel(channel NotificationChannel) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.channels = append(s.channels, channel)
}

// AddRule adds an alerting rule
func (s *Service) AddRule(rule *AlertRule) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.rules[rule.Name] = rule
}

// RemoveRule removes an alerting rule
func (s *Service) RemoveRule(name string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.rules, name)
}

// TriggerAlert triggers an alert
func (s *Service) TriggerAlert(ctx context.Context, alert *Alert) error {
	if !s.config.Enabled {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if we've reached the maximum number of alerts
	if len(s.activeAlerts) >= s.config.MaxAlerts {
		s.logger.WithContext(ctx).Warn("Maximum number of active alerts reached, dropping alert")
		return fmt.Errorf("maximum number of active alerts reached")
	}

	// Generate ID if not provided
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("%s-%d", alert.Component, time.Now().Unix())
	}

	// Set timestamp if not provided
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Check if alert is already active
	if existingAlert, exists := s.activeAlerts[alert.ID]; exists {
		// Update existing alert
		existingAlert.Description = alert.Description
		existingAlert.Timestamp = alert.Timestamp
		existingAlert.Labels = alert.Labels
		existingAlert.Annotations = alert.Annotations
		return nil
	}

	// Add to active alerts
	s.activeAlerts[alert.ID] = alert

	// Log the alert
	s.logger.WithContext(ctx).WithFields(logging.Fields{
		"alert_id":    alert.ID,
		"title":       alert.Title,
		"severity":    alert.Severity,
		"component":   alert.Component,
	}).Warn("Alert triggered")

	// Send notifications
	go s.sendNotifications(ctx, alert)

	return nil
}

// ResolveAlert resolves an active alert
func (s *Service) ResolveAlert(ctx context.Context, alertID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	alert, exists := s.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	// Mark as resolved
	now := time.Now()
	alert.Resolved = true
	alert.ResolvedAt = &now

	// Log resolution
	s.logger.WithContext(ctx).WithFields(logging.Fields{
		"alert_id":  alert.ID,
		"title":     alert.Title,
		"component": alert.Component,
		"duration":  now.Sub(alert.Timestamp).String(),
	}).Info("Alert resolved")

	// Remove from active alerts
	delete(s.activeAlerts, alertID)

	// Send resolution notifications
	go s.sendNotifications(ctx, alert)

	return nil
}

// GetActiveAlerts returns all active alerts
func (s *Service) GetActiveAlerts() []*Alert {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	alerts := make([]*Alert, 0, len(s.activeAlerts))
	for _, alert := range s.activeAlerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// GetAlert returns a specific alert
func (s *Service) GetAlert(alertID string) (*Alert, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	alert, exists := s.activeAlerts[alertID]
	return alert, exists
}

// sendNotifications sends alert notifications to all channels
func (s *Service) sendNotifications(ctx context.Context, alert *Alert) {
	s.mutex.RLock()
	channels := make([]NotificationChannel, len(s.channels))
	copy(channels, s.channels)
	s.mutex.RUnlock()

	for _, channel := range channels {
		go func(ch NotificationChannel) {
			if err := ch.Send(ctx, alert); err != nil {
				s.logger.WithContext(ctx).WithError(err).WithFields(logging.Fields{
					"channel":   ch.Name(),
					"alert_id":  alert.ID,
				}).Error("Failed to send alert notification")
			}
		}(channel)
	}
}

// SlackChannel implements Slack notifications
type SlackChannel struct {
	webhookURL string
	channel    string
	username   string
	iconEmoji  string
	client     *http.Client
}

// NewSlackChannel creates a new Slack notification channel
func NewSlackChannel(webhookURL, channel, username, iconEmoji string) *SlackChannel {
	return &SlackChannel{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
		iconEmoji:  iconEmoji,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the channel name
func (sc *SlackChannel) Name() string {
	return "slack"
}

// Send sends an alert to Slack
func (sc *SlackChannel) Send(ctx context.Context, alert *Alert) error {
	color := sc.getColorForSeverity(alert.Severity)
	status := "FIRING"
	if alert.Resolved {
		status = "RESOLVED"
		color = "good"
	}

	payload := map[string]interface{}{
		"channel":   sc.channel,
		"username":  sc.username,
		"icon_emoji": sc.iconEmoji,
		"attachments": []map[string]interface{}{
			{
				"color":      color,
				"title":      fmt.Sprintf("[%s] %s", status, alert.Title),
				"text":       alert.Description,
				"timestamp":  alert.Timestamp.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "Severity",
						"value": string(alert.Severity),
						"short": true,
					},
					{
						"title": "Component",
						"value": alert.Component,
						"short": true,
					},
				},
			},
		},
	}

	// Add labels as fields
	if len(alert.Labels) > 0 {
		attachment := payload["attachments"].([]map[string]interface{})[0]
		fields := attachment["fields"].([]map[string]interface{})
		
		for key, value := range alert.Labels {
			fields = append(fields, map[string]interface{}{
				"title": key,
				"value": value,
				"short": true,
			})
		}
		
		attachment["fields"] = fields
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sc.webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status %d", resp.StatusCode)
	}

	return nil
}

// getColorForSeverity returns the appropriate color for alert severity
func (sc *SlackChannel) getColorForSeverity(severity Severity) string {
	switch severity {
	case SeverityInfo:
		return "#36a64f" // green
	case SeverityWarning:
		return "#ff9500" // orange
	case SeverityCritical:
		return "#ff0000" // red
	case SeverityFatal:
		return "#8b0000" // dark red
	default:
		return "#808080" // gray
	}
}

// EmailChannel implements email notifications
type EmailChannel struct {
	smtpHost     string
	smtpPort     int
	username     string
	password     string
	fromAddress  string
	toAddresses  []string
	client       *http.Client
}

// NewEmailChannel creates a new email notification channel
func NewEmailChannel(smtpHost string, smtpPort int, username, password, fromAddress string, toAddresses []string) *EmailChannel {
	return &EmailChannel{
		smtpHost:    smtpHost,
		smtpPort:    smtpPort,
		username:    username,
		password:    password,
		fromAddress: fromAddress,
		toAddresses: toAddresses,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the channel name
func (ec *EmailChannel) Name() string {
	return "email"
}

// Send sends an alert via email
func (ec *EmailChannel) Send(ctx context.Context, alert *Alert) error {
	// This is a simplified implementation
	// In a real implementation, you would use an SMTP library to send emails
	
	subject := fmt.Sprintf("[AgentScan Alert] %s - %s", alert.Severity, alert.Title)
	body := fmt.Sprintf(`
Alert Details:
- Title: %s
- Description: %s
- Severity: %s
- Component: %s
- Timestamp: %s
- Status: %s

Labels:
%s

Annotations:
%s
`,
		alert.Title,
		alert.Description,
		alert.Severity,
		alert.Component,
		alert.Timestamp.Format(time.RFC3339),
		func() string {
			if alert.Resolved {
				return "RESOLVED"
			}
			return "FIRING"
		}(),
		formatMap(alert.Labels),
		formatMap(alert.Annotations),
	)

	// Log the email (in a real implementation, send actual email)
	fmt.Printf("Email Alert:\nTo: %v\nSubject: %s\nBody: %s\n", ec.toAddresses, subject, body)

	return nil
}

// WebhookChannel implements webhook notifications
type WebhookChannel struct {
	url     string
	headers map[string]string
	client  *http.Client
}

// NewWebhookChannel creates a new webhook notification channel
func NewWebhookChannel(url string, headers map[string]string) *WebhookChannel {
	return &WebhookChannel{
		url:     url,
		headers: headers,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the channel name
func (wc *WebhookChannel) Name() string {
	return "webhook"
}

// Send sends an alert via webhook
func (wc *WebhookChannel) Send(ctx context.Context, alert *Alert) error {
	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", wc.url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range wc.headers {
		req.Header.Set(key, value)
	}

	resp, err := wc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// formatMap formats a map for display
func formatMap(m map[string]string) string {
	if len(m) == 0 {
		return "  (none)"
	}

	result := ""
	for key, value := range m {
		result += fmt.Sprintf("  %s: %s\n", key, value)
	}
	return result
}

// PredefinedAlerts contains common alert definitions
var PredefinedAlerts = map[string]*AlertRule{
	"high_cpu_usage": {
		Name:        "high_cpu_usage",
		Description: "CPU usage is above threshold",
		Condition: AlertCondition{
			MetricName:  "cpu_usage_percent",
			Operator:    ">",
			Threshold:   80.0,
			Duration:    "5m",
			Aggregation: "avg",
		},
		Severity: SeverityWarning,
		Labels: map[string]string{
			"category": "performance",
		},
		Annotations: map[string]string{
			"summary":     "High CPU usage detected",
			"description": "CPU usage has been above 80% for more than 5 minutes",
		},
		Enabled: true,
	},
	"high_memory_usage": {
		Name:        "high_memory_usage",
		Description: "Memory usage is above threshold",
		Condition: AlertCondition{
			MetricName:  "memory_usage_percent",
			Operator:    ">",
			Threshold:   85.0,
			Duration:    "5m",
			Aggregation: "avg",
		},
		Severity: SeverityWarning,
		Labels: map[string]string{
			"category": "performance",
		},
		Annotations: map[string]string{
			"summary":     "High memory usage detected",
			"description": "Memory usage has been above 85% for more than 5 minutes",
		},
		Enabled: true,
	},
	"database_connection_pool_exhausted": {
		Name:        "database_connection_pool_exhausted",
		Description: "Database connection pool is nearly exhausted",
		Condition: AlertCondition{
			MetricName:  "database_connections_usage_percent",
			Operator:    ">",
			Threshold:   90.0,
			Duration:    "2m",
			Aggregation: "avg",
		},
		Severity: SeverityCritical,
		Labels: map[string]string{
			"category": "database",
		},
		Annotations: map[string]string{
			"summary":     "Database connection pool nearly exhausted",
			"description": "Database connection pool usage has been above 90% for more than 2 minutes",
		},
		Enabled: true,
	},
	"scan_failure_rate_high": {
		Name:        "scan_failure_rate_high",
		Description: "Scan failure rate is above threshold",
		Condition: AlertCondition{
			MetricName:  "scan_failure_rate",
			Operator:    ">",
			Threshold:   10.0,
			Duration:    "10m",
			Aggregation: "avg",
		},
		Severity: SeverityWarning,
		Labels: map[string]string{
			"category": "business",
		},
		Annotations: map[string]string{
			"summary":     "High scan failure rate detected",
			"description": "Scan failure rate has been above 10% for more than 10 minutes",
		},
		Enabled: true,
	},
	"queue_backlog_high": {
		Name:        "queue_backlog_high",
		Description: "Job queue backlog is above threshold",
		Condition: AlertCondition{
			MetricName:  "queue_size",
			Operator:    ">",
			Threshold:   1000.0,
			Duration:    "5m",
			Aggregation: "sum",
		},
		Severity: SeverityWarning,
		Labels: map[string]string{
			"category": "performance",
		},
		Annotations: map[string]string{
			"summary":     "High queue backlog detected",
			"description": "Job queue backlog has been above 1000 items for more than 5 minutes",
		},
		Enabled: true,
	},
}