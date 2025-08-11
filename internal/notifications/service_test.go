package notifications

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// MockChannelHandler is a mock implementation of ChannelHandler
type MockChannelHandler struct {
	mock.Mock
}

func (m *MockChannelHandler) Send(ctx context.Context, channel NotificationChannel, message NotificationMessage) error {
	args := m.Called(ctx, channel, message)
	return args.Error(0)
}

func (m *MockChannelHandler) Test(ctx context.Context, channel NotificationChannel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelHandler) GetChannelType() NotificationChannelType {
	args := m.Called()
	return args.Get(0).(NotificationChannelType)
}

// MockTemplateManager is a mock implementation of TemplateManager
type MockTemplateManager struct {
	mock.Mock
}

func (m *MockTemplateManager) RenderScanCompleted(notification ScanCompletedNotification, format string) (NotificationMessage, error) {
	args := m.Called(notification, format)
	return args.Get(0).(NotificationMessage), args.Error(1)
}

func (m *MockTemplateManager) RenderCriticalFinding(notification CriticalFindingNotification, format string) (NotificationMessage, error) {
	args := m.Called(notification, format)
	return args.Get(0).(NotificationMessage), args.Error(1)
}

func (m *MockTemplateManager) RenderScanFailed(notification ScanFailedNotification, format string) (NotificationMessage, error) {
	args := m.Called(notification, format)
	return args.Get(0).(NotificationMessage), args.Error(1)
}

// MockNotificationRepository is a mock implementation of NotificationRepository
type MockNotificationRepository struct {
	mock.Mock
}

func (m *MockNotificationRepository) CreateChannel(ctx context.Context, channel *NotificationChannel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockNotificationRepository) GetChannel(ctx context.Context, id uuid.UUID) (*NotificationChannel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*NotificationChannel), args.Error(1)
}

func (m *MockNotificationRepository) GetChannelsByUser(ctx context.Context, userID uuid.UUID) ([]NotificationChannel, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]NotificationChannel), args.Error(1)
}

func (m *MockNotificationRepository) GetChannelsByOrg(ctx context.Context, orgID uuid.UUID) ([]NotificationChannel, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]NotificationChannel), args.Error(1)
}

func (m *MockNotificationRepository) UpdateChannel(ctx context.Context, channel *NotificationChannel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockNotificationRepository) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationRepository) LogEvent(ctx context.Context, event *NotificationEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockNotificationRepository) GetEvents(ctx context.Context, filter NotificationFilter, limit, offset int) ([]NotificationEvent, int64, error) {
	args := m.Called(ctx, filter, limit, offset)
	return args.Get(0).([]NotificationEvent), args.Get(1).(int64), args.Error(2)
}

func (m *MockNotificationRepository) GetStats(ctx context.Context, filter NotificationFilter) (*NotificationStats, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*NotificationStats), args.Error(1)
}

func TestService_SendScanCompleted(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	mockHandler := &MockChannelHandler{}

	service := NewService(logger, mockRepo, mockTemplates)
	
	// Setup handler mock for registration
	mockHandler.On("GetChannelType").Return(ChannelTypeSlack)
	service.RegisterChannelHandler(mockHandler)

	ctx := context.Background()
	userID := uuid.New()
	notification := ScanCompletedNotification{
		ScanID:     uuid.New(),
		Repository: "test/repo",
		Branch:     "main",
		Commit:     "abc123def456",
		Status:     "completed",
		Duration:   2 * time.Minute,
		FindingsCount: FindingsCount{
			High:   2,
			Medium: 5,
			Low:    3,
			Total:  10,
		},
		DashboardURL: "https://dashboard.example.com/scan/123",
		UserID:       userID,
	}

	// Setup mocks
	channels := []NotificationChannel{
		{
			ID:     uuid.New(),
			UserID: userID,
			Type:   ChannelTypeSlack,
			Name:   "Test Slack Channel",
			Config: ChannelConfig{
				SlackWebhookURL: "https://hooks.slack.com/test",
			},
			Enabled: true,
			Preferences: NotificationPreferences{
				ScanCompleted: true,
			},
		},
	}

	expectedMessage := NotificationMessage{
		Subject: "Scan completed for test/repo",
		Body:    "Scan completed successfully",
		Format:  "markdown",
	}

	mockRepo.On("GetChannelsByUser", ctx, userID).Return(channels, nil)
	mockRepo.On("GetChannelsByOrg", ctx, mock.Anything).Return([]NotificationChannel{}, nil)
	mockTemplates.On("RenderScanCompleted", notification, "markdown").Return(expectedMessage, nil)
	mockHandler.On("GetChannelType").Return(ChannelTypeSlack)
	mockHandler.On("Send", ctx, channels[0], expectedMessage).Return(nil)
	mockRepo.On("LogEvent", ctx, mock.AnythingOfType("*notifications.NotificationEvent")).Return(nil)

	// Execute
	err := service.SendScanCompleted(ctx, notification)

	// Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockTemplates.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}

func TestService_SendCriticalFinding(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	mockHandler := &MockChannelHandler{}

	service := NewService(logger, mockRepo, mockTemplates)
	
	// Setup handler mock for registration
	mockHandler.On("GetChannelType").Return(ChannelTypeEmail)
	service.RegisterChannelHandler(mockHandler)

	ctx := context.Background()
	userID := uuid.New()
	notification := CriticalFindingNotification{
		ScanID:     uuid.New(),
		Repository: "test/repo",
		Branch:     "main",
		Commit:     "abc123def456",
		Finding: CriticalFinding{
			ID:          "finding-123",
			Title:       "SQL Injection vulnerability",
			Severity:    "high",
			Category:    "sql_injection",
			File:        "app.js",
			Line:        42,
			Tool:        "semgrep",
			Description: "Potential SQL injection detected",
		},
		DashboardURL: "https://dashboard.example.com/finding/123",
		UserID:       userID,
	}

	// Setup mocks
	channels := []NotificationChannel{
		{
			ID:     uuid.New(),
			UserID: userID,
			Type:   ChannelTypeEmail,
			Name:   "Test Email Channel",
			Config: ChannelConfig{
				EmailAddress: "test@example.com",
			},
			Enabled: true,
			Preferences: NotificationPreferences{
				CriticalFindings: true,
				MinSeverity:      "high",
			},
		},
	}

	expectedMessage := NotificationMessage{
		Subject: "Critical security finding in test/repo",
		Body:    "Critical finding detected",
		Format:  "html",
	}

	mockRepo.On("GetChannelsByUser", ctx, userID).Return(channels, nil)
	mockRepo.On("GetChannelsByOrg", ctx, mock.Anything).Return([]NotificationChannel{}, nil)
	mockTemplates.On("RenderCriticalFinding", notification, "html").Return(expectedMessage, nil)
	mockHandler.On("GetChannelType").Return(ChannelTypeEmail)
	mockHandler.On("Send", ctx, channels[0], expectedMessage).Return(nil)
	mockRepo.On("LogEvent", ctx, mock.AnythingOfType("*notifications.NotificationEvent")).Return(nil)

	// Execute
	err := service.SendCriticalFinding(ctx, notification)

	// Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockTemplates.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}

func TestService_SendScanFailed(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	mockHandler := &MockChannelHandler{}

	service := NewService(logger, mockRepo, mockTemplates)
	
	// Setup handler mock for registration
	mockHandler.On("GetChannelType").Return(ChannelTypeTeams)
	service.RegisterChannelHandler(mockHandler)

	ctx := context.Background()
	userID := uuid.New()
	notification := ScanFailedNotification{
		ScanID:       uuid.New(),
		Repository:   "test/repo",
		Branch:       "main",
		Commit:       "abc123def456",
		Error:        "Docker container failed to start",
		Duration:     30 * time.Second,
		DashboardURL: "https://dashboard.example.com/scan/123",
		UserID:       userID,
	}

	// Setup mocks
	channels := []NotificationChannel{
		{
			ID:     uuid.New(),
			UserID: userID,
			Type:   ChannelTypeTeams,
			Name:   "Test Teams Channel",
			Config: ChannelConfig{
				TeamsWebhookURL: "https://outlook.office.com/webhook/test",
			},
			Enabled: true,
			Preferences: NotificationPreferences{
				ScanFailed: true,
			},
		},
	}

	expectedMessage := NotificationMessage{
		Subject: "Scan failed for test/repo",
		Body:    "Scan failed with error",
		Format:  "markdown",
	}

	mockRepo.On("GetChannelsByUser", ctx, userID).Return(channels, nil)
	mockRepo.On("GetChannelsByOrg", ctx, mock.Anything).Return([]NotificationChannel{}, nil)
	mockTemplates.On("RenderScanFailed", notification, "markdown").Return(expectedMessage, nil)
	mockHandler.On("GetChannelType").Return(ChannelTypeTeams)
	mockHandler.On("Send", ctx, channels[0], expectedMessage).Return(nil)
	mockRepo.On("LogEvent", ctx, mock.AnythingOfType("*notifications.NotificationEvent")).Return(nil)

	// Execute
	err := service.SendScanFailed(ctx, notification)

	// Assert
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockTemplates.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}

func TestService_TestConnection(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	mockHandler := &MockChannelHandler{}

	service := NewService(logger, mockRepo, mockTemplates)
	
	// Setup handler mock for registration
	mockHandler.On("GetChannelType").Return(ChannelTypeSlack)
	service.RegisterChannelHandler(mockHandler)

	ctx := context.Background()
	channel := NotificationChannel{
		ID:   uuid.New(),
		Type: ChannelTypeSlack,
		Config: ChannelConfig{
			SlackWebhookURL: "https://hooks.slack.com/test",
		},
	}

	// Setup mocks
	mockHandler.On("GetChannelType").Return(ChannelTypeSlack)
	mockHandler.On("Test", ctx, channel).Return(nil)
	mockRepo.On("LogEvent", ctx, mock.AnythingOfType("*notifications.NotificationEvent")).Return(nil)

	// Execute
	err := service.TestConnection(ctx, channel)

	// Assert
	require.NoError(t, err)
	mockHandler.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestService_GetSupportedChannels(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	
	service := NewService(logger, mockRepo, mockTemplates)

	// Register multiple handlers
	slackHandler := &MockChannelHandler{}
	slackHandler.On("GetChannelType").Return(ChannelTypeSlack)
	service.RegisterChannelHandler(slackHandler)

	teamsHandler := &MockChannelHandler{}
	teamsHandler.On("GetChannelType").Return(ChannelTypeTeams)
	service.RegisterChannelHandler(teamsHandler)

	emailHandler := &MockChannelHandler{}
	emailHandler.On("GetChannelType").Return(ChannelTypeEmail)
	service.RegisterChannelHandler(emailHandler)

	// Execute
	channels := service.GetSupportedChannels()

	// Assert
	assert.Len(t, channels, 3)
	assert.Contains(t, channels, ChannelTypeSlack)
	assert.Contains(t, channels, ChannelTypeTeams)
	assert.Contains(t, channels, ChannelTypeEmail)
}

func TestService_FilterChannelsForScanCompleted(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	
	service := NewService(logger, mockRepo, mockTemplates)

	notification := ScanCompletedNotification{
		Repository: "test/repo",
	}

	channels := []NotificationChannel{
		{
			ID:   uuid.New(),
			Name: "Enabled for scan completed",
			Preferences: NotificationPreferences{
				ScanCompleted: true,
			},
		},
		{
			ID:   uuid.New(),
			Name: "Disabled for scan completed",
			Preferences: NotificationPreferences{
				ScanCompleted: false,
			},
		},
		{
			ID:   uuid.New(),
			Name: "Repository filter match",
			Preferences: NotificationPreferences{
				ScanCompleted: true,
				Repositories:  []string{"test/repo", "other/repo"},
			},
		},
		{
			ID:   uuid.New(),
			Name: "Repository filter no match",
			Preferences: NotificationPreferences{
				ScanCompleted: true,
				Repositories:  []string{"other/repo"},
			},
		},
	}

	filtered := service.filterChannelsForScanCompleted(channels, notification)

	assert.Len(t, filtered, 2)
	assert.Equal(t, "Enabled for scan completed", filtered[0].Name)
	assert.Equal(t, "Repository filter match", filtered[1].Name)
}

func TestService_FilterChannelsForCriticalFinding(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	
	service := NewService(logger, mockRepo, mockTemplates)

	notification := CriticalFindingNotification{
		Repository: "test/repo",
		Finding: CriticalFinding{
			Severity: "high",
		},
	}

	channels := []NotificationChannel{
		{
			ID:   uuid.New(),
			Name: "Enabled for critical findings",
			Preferences: NotificationPreferences{
				CriticalFindings: true,
				MinSeverity:      "medium",
			},
		},
		{
			ID:   uuid.New(),
			Name: "Disabled for critical findings",
			Preferences: NotificationPreferences{
				CriticalFindings: false,
			},
		},
		{
			ID:   uuid.New(),
			Name: "Severity too low",
			Preferences: NotificationPreferences{
				CriticalFindings: true,
				MinSeverity:      "high",
			},
		},
	}

	// Change finding severity to medium to test filtering
	notification.Finding.Severity = "medium"
	filtered := service.filterChannelsForCriticalFinding(channels, notification)

	assert.Len(t, filtered, 1)
	assert.Equal(t, "Enabled for critical findings", filtered[0].Name)
}

func TestService_MeetsSeverityThreshold(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	
	service := NewService(logger, mockRepo, mockTemplates)

	tests := []struct {
		findingSeverity string
		minSeverity     string
		expected        bool
	}{
		{"high", "low", true},
		{"high", "medium", true},
		{"high", "high", true},
		{"medium", "low", true},
		{"medium", "medium", true},
		{"medium", "high", false},
		{"low", "low", true},
		{"low", "medium", false},
		{"low", "high", false},
		{"invalid", "medium", false},
		{"medium", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.findingSeverity, tt.minSeverity), func(t *testing.T) {
			result := service.meetsSeverityThreshold(tt.findingSeverity, tt.minSeverity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_IsWithinTimeWindow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockRepo := &MockNotificationRepository{}
	mockTemplates := &MockTemplateManager{}
	
	service := NewService(logger, mockRepo, mockTemplates)

	tests := []struct {
		name     string
		window   TimeWindow
		expected bool
	}{
		{
			name: "disabled window",
			window: TimeWindow{
				Enabled: false,
			},
			expected: true,
		},
		{
			name: "enabled window with no restrictions",
			window: TimeWindow{
				Enabled: true,
			},
			expected: true,
		},
		{
			name: "weekday restriction - current day included",
			window: TimeWindow{
				Enabled:  true,
				Weekdays: []int{int(time.Now().Weekday())},
			},
			expected: true,
		},
		{
			name: "weekday restriction - current day excluded",
			window: TimeWindow{
				Enabled:  true,
				Weekdays: []int{(int(time.Now().Weekday()) + 1) % 7},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isWithinTimeWindow(tt.window)
			assert.Equal(t, tt.expected, result)
		})
	}
}