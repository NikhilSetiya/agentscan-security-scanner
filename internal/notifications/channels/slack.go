package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/agentscan/agentscan/internal/notifications"
)

// SlackHandler implements notification sending to Slack
type SlackHandler struct {
	logger     *zap.Logger
	httpClient *http.Client
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Username    string            `json:"username,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackAttachment represents a Slack message attachment
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackBlock represents a Slack block element
type SlackBlock struct {
	Type string      `json:"type"`
	Text *SlackText  `json:"text,omitempty"`
}

// SlackText represents text in a Slack block
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewSlackHandler creates a new Slack notification handler
func NewSlackHandler(logger *zap.Logger) *SlackHandler {
	return &SlackHandler{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send sends a notification to Slack
func (h *SlackHandler) Send(ctx context.Context, channel notifications.NotificationChannel, message notifications.NotificationMessage) error {
	if channel.Config.SlackWebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	slackMessage := h.buildSlackMessage(channel, message)

	payload, err := json.Marshal(slackMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", channel.Config.SlackWebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	h.logger.Info("Successfully sent Slack notification",
		zap.String("channel_id", channel.ID.String()),
		zap.String("webhook_url", maskWebhookURL(channel.Config.SlackWebhookURL)))

	return nil
}

// Test tests the Slack channel connectivity
func (h *SlackHandler) Test(ctx context.Context, channel notifications.NotificationChannel) error {
	if channel.Config.SlackWebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	testMessage := notifications.NotificationMessage{
		Subject: "AgentScan Test Notification",
		Body:    "This is a test notification from AgentScan. If you receive this, your Slack integration is working correctly!",
		Format:  "markdown",
	}

	return h.Send(ctx, channel, testMessage)
}

// GetChannelType returns the channel type
func (h *SlackHandler) GetChannelType() notifications.NotificationChannelType {
	return notifications.ChannelTypeSlack
}

// buildSlackMessage converts a generic notification message to Slack format
func (h *SlackHandler) buildSlackMessage(channel notifications.NotificationChannel, message notifications.NotificationMessage) SlackMessage {
	slackMessage := SlackMessage{
		Text:     message.Subject,
		Username: channel.Config.SlackUsername,
		Channel:  channel.Config.SlackChannel,
	}

	// Set icon based on message metadata
	if severity, exists := message.Metadata["severity"]; exists {
		switch severity {
		case "high":
			slackMessage.IconEmoji = ":rotating_light:"
		case "medium":
			slackMessage.IconEmoji = ":warning:"
		case "low":
			slackMessage.IconEmoji = ":information_source:"
		default:
			slackMessage.IconEmoji = ":shield:"
		}
	} else {
		slackMessage.IconEmoji = ":shield:"
	}

	// Create attachment for rich formatting
	attachment := SlackAttachment{
		Text:      message.Body,
		Footer:    "AgentScan Security Scanner",
		Timestamp: time.Now().Unix(),
	}

	// Set color based on message type or severity
	if eventType, exists := message.Metadata["event_type"]; exists {
		switch eventType {
		case "scan_completed":
			if status, exists := message.Metadata["status"]; exists && status == "completed" {
				attachment.Color = "good" // Green
			} else {
				attachment.Color = "warning" // Yellow
			}
		case "critical_finding":
			attachment.Color = "danger" // Red
		case "scan_failed":
			attachment.Color = "danger" // Red
		default:
			attachment.Color = "#36a64f" // Default green
		}
	}

	// Add fields from metadata
	if repository, exists := message.Metadata["repository"]; exists {
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Repository",
			Value: fmt.Sprintf("%v", repository),
			Short: true,
		})
	}

	if branch, exists := message.Metadata["branch"]; exists {
		attachment.Fields = append(attachment.Fields, SlackField{
			Title: "Branch",
			Value: fmt.Sprintf("%v", branch),
			Short: true,
		})
	}

	if findingsCount, exists := message.Metadata["findings_count"]; exists {
		if count, ok := findingsCount.(map[string]interface{}); ok {
			var countText string
			if high, exists := count["high"]; exists {
				countText += fmt.Sprintf("High: %v ", high)
			}
			if medium, exists := count["medium"]; exists {
				countText += fmt.Sprintf("Medium: %v ", medium)
			}
			if low, exists := count["low"]; exists {
				countText += fmt.Sprintf("Low: %v", low)
			}
			if countText != "" {
				attachment.Fields = append(attachment.Fields, SlackField{
					Title: "Findings",
					Value: countText,
					Short: true,
				})
			}
		}
	}

	if dashboardURL, exists := message.Metadata["dashboard_url"]; exists {
		attachment.TitleLink = fmt.Sprintf("%v", dashboardURL)
		attachment.Title = "View in Dashboard"
	}

	slackMessage.Attachments = []SlackAttachment{attachment}

	return slackMessage
}

// maskWebhookURL masks the webhook URL for logging
func maskWebhookURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:20] + "***"
}