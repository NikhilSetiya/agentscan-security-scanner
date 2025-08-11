package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service implements the NotificationService interface
type Service struct {
	logger    *zap.Logger
	channels  map[NotificationChannelType]ChannelHandler
	templates TemplateManager
	repo      NotificationRepository
	mu        sync.RWMutex
}

// ChannelHandler defines the interface for channel-specific notification handlers
type ChannelHandler interface {
	Send(ctx context.Context, channel NotificationChannel, message NotificationMessage) error
	Test(ctx context.Context, channel NotificationChannel) error
	GetChannelType() NotificationChannelType
}

// NotificationMessage represents a formatted notification message
type NotificationMessage struct {
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	Format      string                 `json:"format"`
	Attachments []NotificationAttachment `json:"attachments,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationAttachment represents a file attachment
type NotificationAttachment struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
	URL         string `json:"url,omitempty"`
}

// TemplateManager handles notification templates
type TemplateManager interface {
	RenderScanCompleted(notification ScanCompletedNotification, format string) (NotificationMessage, error)
	RenderCriticalFinding(notification CriticalFindingNotification, format string) (NotificationMessage, error)
	RenderScanFailed(notification ScanFailedNotification, format string) (NotificationMessage, error)
}

// NotificationRepository handles persistence of notification data
type NotificationRepository interface {
	CreateChannel(ctx context.Context, channel *NotificationChannel) error
	GetChannel(ctx context.Context, id uuid.UUID) (*NotificationChannel, error)
	GetChannelsByUser(ctx context.Context, userID uuid.UUID) ([]NotificationChannel, error)
	GetChannelsByOrg(ctx context.Context, orgID uuid.UUID) ([]NotificationChannel, error)
	UpdateChannel(ctx context.Context, channel *NotificationChannel) error
	DeleteChannel(ctx context.Context, id uuid.UUID) error
	
	LogEvent(ctx context.Context, event *NotificationEvent) error
	GetEvents(ctx context.Context, filter NotificationFilter, limit, offset int) ([]NotificationEvent, int64, error)
	GetStats(ctx context.Context, filter NotificationFilter) (*NotificationStats, error)
}

// NewService creates a new notification service
func NewService(logger *zap.Logger, repo NotificationRepository, templates TemplateManager) *Service {
	return &Service{
		logger:    logger,
		channels:  make(map[NotificationChannelType]ChannelHandler),
		templates: templates,
		repo:      repo,
	}
}

// RegisterChannelHandler registers a handler for a specific channel type
func (s *Service) RegisterChannelHandler(handler ChannelHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels[handler.GetChannelType()] = handler
}

// SendScanCompleted sends notification when a scan is completed
func (s *Service) SendScanCompleted(ctx context.Context, notification ScanCompletedNotification) error {
	s.logger.Info("Sending scan completed notification",
		zap.String("scan_id", notification.ScanID.String()),
		zap.String("repository", notification.Repository),
		zap.String("status", notification.Status))

	// Get notification channels for the user/org
	channels, err := s.getNotificationChannels(ctx, notification.UserID, notification.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get notification channels: %w", err)
	}

	// Filter channels based on preferences
	filteredChannels := s.filterChannelsForScanCompleted(channels, notification)

	// Send notifications to all applicable channels
	var errors []error
	for _, channel := range filteredChannels {
		if err := s.sendScanCompletedToChannel(ctx, channel, notification); err != nil {
			s.logger.Error("Failed to send scan completed notification",
				zap.String("channel_id", channel.ID.String()),
				zap.String("channel_type", string(channel.Type)),
				zap.Error(err))
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to %d channels: %v", len(errors), errors)
	}

	return nil
}

// SendCriticalFinding sends notification for critical security findings
func (s *Service) SendCriticalFinding(ctx context.Context, notification CriticalFindingNotification) error {
	s.logger.Info("Sending critical finding notification",
		zap.String("scan_id", notification.ScanID.String()),
		zap.String("repository", notification.Repository),
		zap.String("finding_id", notification.Finding.ID))

	// Get notification channels for the user/org
	channels, err := s.getNotificationChannels(ctx, notification.UserID, notification.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get notification channels: %w", err)
	}

	// Filter channels based on preferences
	filteredChannels := s.filterChannelsForCriticalFinding(channels, notification)

	// Send notifications to all applicable channels
	var errors []error
	for _, channel := range filteredChannels {
		if err := s.sendCriticalFindingToChannel(ctx, channel, notification); err != nil {
			s.logger.Error("Failed to send critical finding notification",
				zap.String("channel_id", channel.ID.String()),
				zap.String("channel_type", string(channel.Type)),
				zap.Error(err))
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to %d channels: %v", len(errors), errors)
	}

	return nil
}

// SendScanFailed sends notification when a scan fails
func (s *Service) SendScanFailed(ctx context.Context, notification ScanFailedNotification) error {
	s.logger.Info("Sending scan failed notification",
		zap.String("scan_id", notification.ScanID.String()),
		zap.String("repository", notification.Repository),
		zap.String("error", notification.Error))

	// Get notification channels for the user/org
	channels, err := s.getNotificationChannels(ctx, notification.UserID, notification.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get notification channels: %w", err)
	}

	// Filter channels based on preferences
	filteredChannels := s.filterChannelsForScanFailed(channels, notification)

	// Send notifications to all applicable channels
	var errors []error
	for _, channel := range filteredChannels {
		if err := s.sendScanFailedToChannel(ctx, channel, notification); err != nil {
			s.logger.Error("Failed to send scan failed notification",
				zap.String("channel_id", channel.ID.String()),
				zap.String("channel_type", string(channel.Type)),
				zap.Error(err))
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to %d channels: %v", len(errors), errors)
	}

	return nil
}

// TestConnection tests the notification channel connectivity
func (s *Service) TestConnection(ctx context.Context, channel NotificationChannel) error {
	s.mu.RLock()
	handler, exists := s.channels[channel.Type]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for channel type: %s", channel.Type)
	}

	// Log test event
	event := &NotificationEvent{
		ID:        uuid.New(),
		ChannelID: channel.ID,
		Type:      EventTypeTest,
		Status:    StatusPending,
		Message:   "Testing connection",
		CreatedAt: time.Now(),
	}

	if err := s.repo.LogEvent(ctx, event); err != nil {
		s.logger.Warn("Failed to log test event", zap.Error(err))
	}

	// Test the connection
	err := handler.Test(ctx, channel)
	
	// Update event status
	if err != nil {
		event.Status = StatusFailed
		event.Error = err.Error()
	} else {
		event.Status = StatusSent
	}

	if updateErr := s.repo.LogEvent(ctx, event); updateErr != nil {
		s.logger.Warn("Failed to update test event", zap.Error(updateErr))
	}

	return err
}

// GetSupportedChannels returns list of supported notification channels
func (s *Service) GetSupportedChannels() []NotificationChannelType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channels := make([]NotificationChannelType, 0, len(s.channels))
	for channelType := range s.channels {
		channels = append(channels, channelType)
	}

	return channels
}

// Helper methods

func (s *Service) getNotificationChannels(ctx context.Context, userID uuid.UUID, orgID *uuid.UUID) ([]NotificationChannel, error) {
	var channels []NotificationChannel
	var err error

	// Get user channels
	userChannels, err := s.repo.GetChannelsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user channels: %w", err)
	}
	channels = append(channels, userChannels...)

	// Get org channels if orgID is provided
	if orgID != nil {
		orgChannels, err := s.repo.GetChannelsByOrg(ctx, *orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get org channels: %w", err)
		}
		channels = append(channels, orgChannels...)
	}

	// Filter enabled channels
	var enabledChannels []NotificationChannel
	for _, channel := range channels {
		if channel.Enabled {
			enabledChannels = append(enabledChannels, channel)
		}
	}

	return enabledChannels, nil
}

func (s *Service) filterChannelsForScanCompleted(channels []NotificationChannel, notification ScanCompletedNotification) []NotificationChannel {
	var filtered []NotificationChannel

	for _, channel := range channels {
		if !channel.Preferences.ScanCompleted {
			continue
		}

		// Check repository filter
		if len(channel.Preferences.Repositories) > 0 {
			found := false
			for _, repo := range channel.Preferences.Repositories {
				if repo == notification.Repository {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check time window
		if !s.isWithinTimeWindow(channel.Preferences.TimeWindow) {
			continue
		}

		filtered = append(filtered, channel)
	}

	return filtered
}

func (s *Service) filterChannelsForCriticalFinding(channels []NotificationChannel, notification CriticalFindingNotification) []NotificationChannel {
	var filtered []NotificationChannel

	for _, channel := range channels {
		if !channel.Preferences.CriticalFindings {
			continue
		}

		// Check minimum severity
		if !s.meetsSeverityThreshold(notification.Finding.Severity, channel.Preferences.MinSeverity) {
			continue
		}

		// Check repository filter
		if len(channel.Preferences.Repositories) > 0 {
			found := false
			for _, repo := range channel.Preferences.Repositories {
				if repo == notification.Repository {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check time window
		if !s.isWithinTimeWindow(channel.Preferences.TimeWindow) {
			continue
		}

		filtered = append(filtered, channel)
	}

	return filtered
}

func (s *Service) filterChannelsForScanFailed(channels []NotificationChannel, notification ScanFailedNotification) []NotificationChannel {
	var filtered []NotificationChannel

	for _, channel := range channels {
		if !channel.Preferences.ScanFailed {
			continue
		}

		// Check repository filter
		if len(channel.Preferences.Repositories) > 0 {
			found := false
			for _, repo := range channel.Preferences.Repositories {
				if repo == notification.Repository {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check time window
		if !s.isWithinTimeWindow(channel.Preferences.TimeWindow) {
			continue
		}

		filtered = append(filtered, channel)
	}

	return filtered
}

func (s *Service) isWithinTimeWindow(window TimeWindow) bool {
	if !window.Enabled {
		return true
	}

	now := time.Now()
	
	// Load timezone
	loc, err := time.LoadLocation(window.Timezone)
	if err != nil {
		s.logger.Warn("Invalid timezone, using UTC", zap.String("timezone", window.Timezone))
		loc = time.UTC
	}
	
	nowInTz := now.In(loc)
	
	// Check weekday
	if len(window.Weekdays) > 0 {
		currentWeekday := int(nowInTz.Weekday())
		found := false
		for _, day := range window.Weekdays {
			if day == currentWeekday {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check time range
	if window.StartTime != "" && window.EndTime != "" {
		currentTime := nowInTz.Format("15:04")
		if currentTime < window.StartTime || currentTime > window.EndTime {
			return false
		}
	}

	return true
}

func (s *Service) meetsSeverityThreshold(findingSeverity, minSeverity string) bool {
	severityLevels := map[string]int{
		"low":    1,
		"medium": 2,
		"high":   3,
	}

	findingLevel, exists := severityLevels[findingSeverity]
	if !exists {
		return false
	}

	minLevel, exists := severityLevels[minSeverity]
	if !exists {
		return true // If min severity is not set or invalid, allow all
	}

	return findingLevel >= minLevel
}

func (s *Service) sendScanCompletedToChannel(ctx context.Context, channel NotificationChannel, notification ScanCompletedNotification) error {
	return s.sendToChannel(ctx, channel, func() (NotificationMessage, error) {
		return s.templates.RenderScanCompleted(notification, s.getFormatForChannel(channel.Type))
	}, EventTypeScanCompleted, map[string]interface{}{
		"scan_id":    notification.ScanID.String(),
		"repository": notification.Repository,
		"status":     notification.Status,
	})
}

func (s *Service) sendCriticalFindingToChannel(ctx context.Context, channel NotificationChannel, notification CriticalFindingNotification) error {
	return s.sendToChannel(ctx, channel, func() (NotificationMessage, error) {
		return s.templates.RenderCriticalFinding(notification, s.getFormatForChannel(channel.Type))
	}, EventTypeCriticalFinding, map[string]interface{}{
		"scan_id":    notification.ScanID.String(),
		"repository": notification.Repository,
		"finding_id": notification.Finding.ID,
		"severity":   notification.Finding.Severity,
	})
}

func (s *Service) sendScanFailedToChannel(ctx context.Context, channel NotificationChannel, notification ScanFailedNotification) error {
	return s.sendToChannel(ctx, channel, func() (NotificationMessage, error) {
		return s.templates.RenderScanFailed(notification, s.getFormatForChannel(channel.Type))
	}, EventTypeScanFailed, map[string]interface{}{
		"scan_id":    notification.ScanID.String(),
		"repository": notification.Repository,
		"error":      notification.Error,
	})
}

func (s *Service) sendToChannel(ctx context.Context, channel NotificationChannel, messageFunc func() (NotificationMessage, error), eventType NotificationEventType, metadata map[string]interface{}) error {
	s.mu.RLock()
	handler, exists := s.channels[channel.Type]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for channel type: %s", channel.Type)
	}

	// Create event
	event := &NotificationEvent{
		ID:        uuid.New(),
		ChannelID: channel.ID,
		Type:      eventType,
		Status:    StatusPending,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	// Log event
	if err := s.repo.LogEvent(ctx, event); err != nil {
		s.logger.Warn("Failed to log notification event", zap.Error(err))
	}

	// Render message
	message, err := messageFunc()
	if err != nil {
		event.Status = StatusFailed
		event.Error = fmt.Sprintf("Failed to render message: %v", err)
		s.repo.LogEvent(ctx, event)
		return fmt.Errorf("failed to render message: %w", err)
	}

	event.Message = message.Subject

	// Send message
	err = handler.Send(ctx, channel, message)
	if err != nil {
		event.Status = StatusFailed
		event.Error = err.Error()
		s.repo.LogEvent(ctx, event)
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Update event status
	event.Status = StatusSent
	if updateErr := s.repo.LogEvent(ctx, event); updateErr != nil {
		s.logger.Warn("Failed to update notification event", zap.Error(updateErr))
	}

	return nil
}

func (s *Service) getFormatForChannel(channelType NotificationChannelType) string {
	switch channelType {
	case ChannelTypeSlack, ChannelTypeTeams:
		return "markdown"
	case ChannelTypeEmail:
		return "html"
	default:
		return "text"
	}
}