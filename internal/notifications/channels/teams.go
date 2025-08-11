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

// TeamsHandler implements notification sending to Microsoft Teams
type TeamsHandler struct {
	logger     *zap.Logger
	httpClient *http.Client
}

// TeamsMessage represents a Microsoft Teams message payload
type TeamsMessage struct {
	Type        string                `json:"@type"`
	Context     string                `json:"@context"`
	Summary     string                `json:"summary"`
	ThemeColor  string                `json:"themeColor,omitempty"`
	Title       string                `json:"title,omitempty"`
	Text        string                `json:"text,omitempty"`
	Sections    []TeamsSection        `json:"sections,omitempty"`
	Actions     []TeamsAction         `json:"potentialAction,omitempty"`
}

// TeamsSection represents a section in a Teams message
type TeamsSection struct {
	ActivityTitle    string       `json:"activityTitle,omitempty"`
	ActivitySubtitle string       `json:"activitySubtitle,omitempty"`
	ActivityImage    string       `json:"activityImage,omitempty"`
	Facts            []TeamsFact  `json:"facts,omitempty"`
	Text             string       `json:"text,omitempty"`
	Markdown         bool         `json:"markdown,omitempty"`
}

// TeamsFact represents a fact in a Teams section
type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TeamsAction represents an action button in a Teams message
type TeamsAction struct {
	Type    string           `json:"@type"`
	Name    string           `json:"name"`
	Targets []TeamsTarget    `json:"targets,omitempty"`
}

// TeamsTarget represents a target for a Teams action
type TeamsTarget struct {
	OS  string `json:"os"`
	URI string `json:"uri"`
}

// NewTeamsHandler creates a new Microsoft Teams notification handler
func NewTeamsHandler(logger *zap.Logger) *TeamsHandler {
	return &TeamsHandler{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send sends a notification to Microsoft Teams
func (h *TeamsHandler) Send(ctx context.Context, channel notifications.NotificationChannel, message notifications.NotificationMessage) error {
	if channel.Config.TeamsWebhookURL == "" {
		return fmt.Errorf("teams webhook URL not configured")
	}

	teamsMessage := h.buildTeamsMessage(channel, message)

	payload, err := json.Marshal(teamsMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal teams message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", channel.Config.TeamsWebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send teams message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("teams API returned status %d", resp.StatusCode)
	}

	h.logger.Info("Successfully sent Teams notification",
		zap.String("channel_id", channel.ID.String()),
		zap.String("webhook_url", maskWebhookURL(channel.Config.TeamsWebhookURL)))

	return nil
}

// Test tests the Microsoft Teams channel connectivity
func (h *TeamsHandler) Test(ctx context.Context, channel notifications.NotificationChannel) error {
	if channel.Config.TeamsWebhookURL == "" {
		return fmt.Errorf("teams webhook URL not configured")
	}

	testMessage := notifications.NotificationMessage{
		Subject: "AgentScan Test Notification",
		Body:    "This is a test notification from AgentScan. If you receive this, your Microsoft Teams integration is working correctly!",
		Format:  "markdown",
	}

	return h.Send(ctx, channel, testMessage)
}

// GetChannelType returns the channel type
func (h *TeamsHandler) GetChannelType() notifications.NotificationChannelType {
	return notifications.ChannelTypeTeams
}

// buildTeamsMessage converts a generic notification message to Teams format
func (h *TeamsHandler) buildTeamsMessage(channel notifications.NotificationChannel, message notifications.NotificationMessage) TeamsMessage {
	teamsMessage := TeamsMessage{
		Type:    "MessageCard",
		Context: "https://schema.org/extensions",
		Summary: message.Subject,
		Title:   message.Subject,
		Text:    message.Body,
	}

	// Set theme color based on message type or severity
	if eventType, exists := message.Metadata["event_type"]; exists {
		switch eventType {
		case "scan_completed":
			if status, exists := message.Metadata["status"]; exists && status == "completed" {
				teamsMessage.ThemeColor = "00FF00" // Green
			} else {
				teamsMessage.ThemeColor = "FFA500" // Orange
			}
		case "critical_finding":
			teamsMessage.ThemeColor = "FF0000" // Red
		case "scan_failed":
			teamsMessage.ThemeColor = "FF0000" // Red
		default:
			teamsMessage.ThemeColor = "0078D4" // Microsoft Blue
		}
	} else {
		teamsMessage.ThemeColor = "0078D4" // Microsoft Blue
	}

	// Create section with facts
	section := TeamsSection{
		ActivityTitle: "AgentScan Security Scanner",
		Markdown:      true,
	}

	// Add facts from metadata
	var facts []TeamsFact

	if repository, exists := message.Metadata["repository"]; exists {
		facts = append(facts, TeamsFact{
			Name:  "Repository",
			Value: fmt.Sprintf("%v", repository),
		})
	}

	if branch, exists := message.Metadata["branch"]; exists {
		facts = append(facts, TeamsFact{
			Name:  "Branch",
			Value: fmt.Sprintf("%v", branch),
		})
	}

	if commit, exists := message.Metadata["commit"]; exists {
		commitStr := fmt.Sprintf("%v", commit)
		if len(commitStr) > 8 {
			commitStr = commitStr[:8]
		}
		facts = append(facts, TeamsFact{
			Name:  "Commit",
			Value: commitStr,
		})
	}

	if findingsCount, exists := message.Metadata["findings_count"]; exists {
		if count, ok := findingsCount.(map[string]interface{}); ok {
			var countText string
			if high, exists := count["high"]; exists {
				countText += fmt.Sprintf("🔴 High: %v  ", high)
			}
			if medium, exists := count["medium"]; exists {
				countText += fmt.Sprintf("🟡 Medium: %v  ", medium)
			}
			if low, exists := count["low"]; exists {
				countText += fmt.Sprintf("🟢 Low: %v", low)
			}
			if countText != "" {
				facts = append(facts, TeamsFact{
					Name:  "Findings",
					Value: countText,
				})
			}
		}
	}

	if duration, exists := message.Metadata["duration"]; exists {
		facts = append(facts, TeamsFact{
			Name:  "Duration",
			Value: fmt.Sprintf("%v", duration),
		})
	}

	if len(facts) > 0 {
		section.Facts = facts
	}

	teamsMessage.Sections = []TeamsSection{section}

	// Add action button if dashboard URL is available
	if dashboardURL, exists := message.Metadata["dashboard_url"]; exists {
		action := TeamsAction{
			Type: "OpenUri",
			Name: "View in Dashboard",
			Targets: []TeamsTarget{
				{
					OS:  "default",
					URI: fmt.Sprintf("%v", dashboardURL),
				},
			},
		}
		teamsMessage.Actions = []TeamsAction{action}
	}

	return teamsMessage
}