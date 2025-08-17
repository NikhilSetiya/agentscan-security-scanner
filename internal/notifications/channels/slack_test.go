package channels

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/notifications"
)

func TestSlackHandler_Send(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewSlackHandler(logger)

	// Create test server
	var receivedMessage SlackMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&receivedMessage)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeSlack,
		Config: notifications.ChannelConfig{
			SlackWebhookURL: server.URL,
			SlackChannel:    "#security",
			SlackUsername:   "AgentScan",
		},
	}

	message := notifications.NotificationMessage{
		Subject: "Test Notification",
		Body:    "This is a test message",
		Format:  "markdown",
		Metadata: map[string]interface{}{
			"event_type":  "scan_completed",
			"repository":  "test/repo",
			"branch":      "main",
			"status":      "completed",
			"dashboard_url": "https://dashboard.example.com",
		},
	}

	// Execute
	err := handler.Send(ctx, channel, message)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Test Notification", receivedMessage.Text)
	assert.Equal(t, "#security", receivedMessage.Channel)
	assert.Equal(t, "AgentScan", receivedMessage.Username)
	assert.Equal(t, ":shield:", receivedMessage.IconEmoji)
	assert.Len(t, receivedMessage.Attachments, 1)

	attachment := receivedMessage.Attachments[0]
	assert.Equal(t, "This is a test message", attachment.Text)
	assert.Equal(t, "AgentScan Security Scanner", attachment.Footer)
	assert.Equal(t, "good", attachment.Color)
	assert.Contains(t, attachment.Fields, SlackField{
		Title: "Repository",
		Value: "test/repo",
		Short: true,
	})
}

func TestSlackHandler_Send_CriticalFinding(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewSlackHandler(logger)

	// Create test server
	var receivedMessage SlackMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedMessage)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeSlack,
		Config: notifications.ChannelConfig{
			SlackWebhookURL: server.URL,
		},
	}

	message := notifications.NotificationMessage{
		Subject: "Critical Security Finding",
		Body:    "SQL injection detected",
		Format:  "markdown",
		Metadata: map[string]interface{}{
			"event_type": "critical_finding",
			"severity":   "high",
		},
	}

	// Execute
	err := handler.Send(ctx, channel, message)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, ":rotating_light:", receivedMessage.IconEmoji)
	assert.Equal(t, "danger", receivedMessage.Attachments[0].Color)
}

func TestSlackHandler_Send_NoWebhookURL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewSlackHandler(logger)

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeSlack,
		Config: notifications.ChannelConfig{
			// No webhook URL
		},
	}

	message := notifications.NotificationMessage{
		Subject: "Test Notification",
		Body:    "This is a test message",
		Format:  "markdown",
	}

	// Execute
	err := handler.Send(ctx, channel, message)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slack webhook URL not configured")
}

func TestSlackHandler_Send_ServerError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewSlackHandler(logger)

	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeSlack,
		Config: notifications.ChannelConfig{
			SlackWebhookURL: server.URL,
		},
	}

	message := notifications.NotificationMessage{
		Subject: "Test Notification",
		Body:    "This is a test message",
		Format:  "markdown",
	}

	// Execute
	err := handler.Send(ctx, channel, message)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slack API returned status 500")
}

func TestSlackHandler_Test(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewSlackHandler(logger)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeSlack,
		Config: notifications.ChannelConfig{
			SlackWebhookURL: server.URL,
		},
	}

	// Execute
	err := handler.Test(ctx, channel)

	// Assert
	require.NoError(t, err)
}

func TestSlackHandler_GetChannelType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewSlackHandler(logger)

	channelType := handler.GetChannelType()
	assert.Equal(t, notifications.ChannelTypeSlack, channelType)
}

func TestSlackHandler_BuildSlackMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewSlackHandler(logger)

	channel := notifications.NotificationChannel{
		Config: notifications.ChannelConfig{
			SlackChannel:  "#security",
			SlackUsername: "AgentScan",
		},
	}

	message := notifications.NotificationMessage{
		Subject: "Test Subject",
		Body:    "Test Body",
		Metadata: map[string]interface{}{
			"event_type":  "scan_completed",
			"repository":  "test/repo",
			"branch":      "main",
			"status":      "completed",
			"findings_count": map[string]interface{}{
				"high":   2,
				"medium": 3,
				"low":    1,
			},
			"dashboard_url": "https://dashboard.example.com",
		},
	}

	slackMessage := handler.buildSlackMessage(channel, message)

	assert.Equal(t, "Test Subject", slackMessage.Text)
	assert.Equal(t, "#security", slackMessage.Channel)
	assert.Equal(t, "AgentScan", slackMessage.Username)
	assert.Equal(t, ":shield:", slackMessage.IconEmoji)
	assert.Len(t, slackMessage.Attachments, 1)

	attachment := slackMessage.Attachments[0]
	assert.Equal(t, "Test Body", attachment.Text)
	assert.Equal(t, "good", attachment.Color)
	assert.Equal(t, "AgentScan Security Scanner", attachment.Footer)
	assert.Equal(t, "View in Dashboard", attachment.Title)
	assert.Equal(t, "https://dashboard.example.com", attachment.TitleLink)

	// Check fields
	expectedFields := []SlackField{
		{Title: "Repository", Value: "test/repo", Short: true},
		{Title: "Branch", Value: "main", Short: true},
		{Title: "Findings", Value: "High: 2 Medium: 3 Low: 1", Short: true},
	}

	for _, expectedField := range expectedFields {
		assert.Contains(t, attachment.Fields, expectedField)
	}
}

func TestMaskWebhookURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "normal URL",
			url:      "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX",
			expected: "https://hooks.slack.***",
		},
		{
			name:     "short URL",
			url:      "short",
			expected: "***",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskWebhookURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}