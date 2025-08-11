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

	"github.com/agentscan/agentscan/internal/notifications"
)

func TestTeamsHandler_Send(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewTeamsHandler(logger)

	// Create test server
	var receivedMessage TeamsMessage
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
		Type: notifications.ChannelTypeTeams,
		Config: notifications.ChannelConfig{
			TeamsWebhookURL: server.URL,
		},
	}

	message := notifications.NotificationMessage{
		Subject: "Test Notification",
		Body:    "This is a test message",
		Format:  "markdown",
		Metadata: map[string]interface{}{
			"event_type":    "scan_completed",
			"repository":    "test/repo",
			"branch":        "main",
			"commit":        "abc123def456",
			"status":        "completed",
			"dashboard_url": "https://dashboard.example.com",
			"duration":      "2m30s",
		},
	}

	// Execute
	err := handler.Send(ctx, channel, message)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "MessageCard", receivedMessage.Type)
	assert.Equal(t, "https://schema.org/extensions", receivedMessage.Context)
	assert.Equal(t, "Test Notification", receivedMessage.Summary)
	assert.Equal(t, "Test Notification", receivedMessage.Title)
	assert.Equal(t, "This is a test message", receivedMessage.Text)
	assert.Equal(t, "00FF00", receivedMessage.ThemeColor) // Green for completed

	assert.Len(t, receivedMessage.Sections, 1)
	section := receivedMessage.Sections[0]
	assert.Equal(t, "AgentScan Security Scanner", section.ActivityTitle)
	assert.True(t, section.Markdown)

	// Check facts
	expectedFacts := []TeamsFact{
		{Name: "Repository", Value: "test/repo"},
		{Name: "Branch", Value: "main"},
		{Name: "Commit", Value: "abc123de"},
		{Name: "Duration", Value: "2m30s"},
	}

	for _, expectedFact := range expectedFacts {
		assert.Contains(t, section.Facts, expectedFact)
	}

	// Check action
	assert.Len(t, receivedMessage.Actions, 1)
	action := receivedMessage.Actions[0]
	assert.Equal(t, "OpenUri", action.Type)
	assert.Equal(t, "View in Dashboard", action.Name)
	assert.Len(t, action.Targets, 1)
	assert.Equal(t, "default", action.Targets[0].OS)
	assert.Equal(t, "https://dashboard.example.com", action.Targets[0].URI)
}

func TestTeamsHandler_Send_CriticalFinding(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewTeamsHandler(logger)

	// Create test server
	var receivedMessage TeamsMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedMessage)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeTeams,
		Config: notifications.ChannelConfig{
			TeamsWebhookURL: server.URL,
		},
	}

	message := notifications.NotificationMessage{
		Subject: "Critical Security Finding",
		Body:    "SQL injection detected",
		Format:  "markdown",
		Metadata: map[string]interface{}{
			"event_type": "critical_finding",
		},
	}

	// Execute
	err := handler.Send(ctx, channel, message)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "FF0000", receivedMessage.ThemeColor) // Red for critical finding
}

func TestTeamsHandler_Send_ScanFailed(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewTeamsHandler(logger)

	// Create test server
	var receivedMessage TeamsMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedMessage)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeTeams,
		Config: notifications.ChannelConfig{
			TeamsWebhookURL: server.URL,
		},
	}

	message := notifications.NotificationMessage{
		Subject: "Scan Failed",
		Body:    "Docker container failed",
		Format:  "markdown",
		Metadata: map[string]interface{}{
			"event_type": "scan_failed",
		},
	}

	// Execute
	err := handler.Send(ctx, channel, message)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "FF0000", receivedMessage.ThemeColor) // Red for failed scan
}

func TestTeamsHandler_Send_NoWebhookURL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewTeamsHandler(logger)

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeTeams,
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
	assert.Contains(t, err.Error(), "teams webhook URL not configured")
}

func TestTeamsHandler_Send_ServerError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewTeamsHandler(logger)

	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeTeams,
		Config: notifications.ChannelConfig{
			TeamsWebhookURL: server.URL,
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
	assert.Contains(t, err.Error(), "teams API returned status 400")
}

func TestTeamsHandler_Test(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewTeamsHandler(logger)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channel := notifications.NotificationChannel{
		ID:   uuid.New(),
		Type: notifications.ChannelTypeTeams,
		Config: notifications.ChannelConfig{
			TeamsWebhookURL: server.URL,
		},
	}

	// Execute
	err := handler.Test(ctx, channel)

	// Assert
	require.NoError(t, err)
}

func TestTeamsHandler_GetChannelType(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewTeamsHandler(logger)

	channelType := handler.GetChannelType()
	assert.Equal(t, notifications.ChannelTypeTeams, channelType)
}

func TestTeamsHandler_BuildTeamsMessage(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := NewTeamsHandler(logger)

	channel := notifications.NotificationChannel{}

	message := notifications.NotificationMessage{
		Subject: "Test Subject",
		Body:    "Test Body",
		Metadata: map[string]interface{}{
			"event_type":  "scan_completed",
			"repository":  "test/repo",
			"branch":      "main",
			"commit":      "abc123def456789",
			"status":      "completed",
			"findings_count": map[string]interface{}{
				"high":   2,
				"medium": 3,
				"low":    1,
			},
			"duration":      "2m30s",
			"dashboard_url": "https://dashboard.example.com",
		},
	}

	teamsMessage := handler.buildTeamsMessage(channel, message)

	assert.Equal(t, "MessageCard", teamsMessage.Type)
	assert.Equal(t, "https://schema.org/extensions", teamsMessage.Context)
	assert.Equal(t, "Test Subject", teamsMessage.Summary)
	assert.Equal(t, "Test Subject", teamsMessage.Title)
	assert.Equal(t, "Test Body", teamsMessage.Text)
	assert.Equal(t, "00FF00", teamsMessage.ThemeColor) // Green for completed

	assert.Len(t, teamsMessage.Sections, 1)
	section := teamsMessage.Sections[0]
	assert.Equal(t, "AgentScan Security Scanner", section.ActivityTitle)
	assert.True(t, section.Markdown)

	// Check facts
	expectedFacts := []TeamsFact{
		{Name: "Repository", Value: "test/repo"},
		{Name: "Branch", Value: "main"},
		{Name: "Commit", Value: "abc123de"}, // Truncated to 8 chars
		{Name: "Findings", Value: "🔴 High: 2  🟡 Medium: 3  🟢 Low: 1"},
		{Name: "Duration", Value: "2m30s"},
	}

	for _, expectedFact := range expectedFacts {
		assert.Contains(t, section.Facts, expectedFact)
	}

	// Check action
	assert.Len(t, teamsMessage.Actions, 1)
	action := teamsMessage.Actions[0]
	assert.Equal(t, "OpenUri", action.Type)
	assert.Equal(t, "View in Dashboard", action.Name)
	assert.Len(t, action.Targets, 1)
	assert.Equal(t, "default", action.Targets[0].OS)
	assert.Equal(t, "https://dashboard.example.com", action.Targets[0].URI)
}