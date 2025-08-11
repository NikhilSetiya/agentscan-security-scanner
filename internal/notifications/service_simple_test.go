package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestService_Simple(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// Create a simple in-memory repository mock
	repo := &SimpleInMemoryRepo{
		channels: make(map[uuid.UUID][]NotificationChannel),
		events:   make([]NotificationEvent, 0),
	}
	
	templates := NewDefaultTemplateManager()
	service := NewService(logger, repo, templates)

	// Test GetSupportedChannels with no handlers
	channels := service.GetSupportedChannels()
	assert.Empty(t, channels)

	// Test registering a handler
	handler := &SimpleSlackHandler{}
	service.RegisterChannelHandler(handler)

	channels = service.GetSupportedChannels()
	assert.Len(t, channels, 1)
	assert.Equal(t, ChannelTypeSlack, channels[0])
}

// SimpleInMemoryRepo is a simple in-memory implementation for testing
type SimpleInMemoryRepo struct {
	channels map[uuid.UUID][]NotificationChannel
	events   []NotificationEvent
}

func (r *SimpleInMemoryRepo) CreateChannel(ctx context.Context, channel *NotificationChannel) error {
	if channel.ID == uuid.Nil {
		channel.ID = uuid.New()
	}
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()
	
	if _, exists := r.channels[channel.UserID]; !exists {
		r.channels[channel.UserID] = make([]NotificationChannel, 0)
	}
	r.channels[channel.UserID] = append(r.channels[channel.UserID], *channel)
	return nil
}

func (r *SimpleInMemoryRepo) GetChannel(ctx context.Context, id uuid.UUID) (*NotificationChannel, error) {
	for _, userChannels := range r.channels {
		for _, channel := range userChannels {
			if channel.ID == id {
				return &channel, nil
			}
		}
	}
	return nil, nil
}

func (r *SimpleInMemoryRepo) GetChannelsByUser(ctx context.Context, userID uuid.UUID) ([]NotificationChannel, error) {
	if channels, exists := r.channels[userID]; exists {
		return channels, nil
	}
	return []NotificationChannel{}, nil
}

func (r *SimpleInMemoryRepo) GetChannelsByOrg(ctx context.Context, orgID uuid.UUID) ([]NotificationChannel, error) {
	return []NotificationChannel{}, nil
}

func (r *SimpleInMemoryRepo) UpdateChannel(ctx context.Context, channel *NotificationChannel) error {
	return nil
}

func (r *SimpleInMemoryRepo) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (r *SimpleInMemoryRepo) LogEvent(ctx context.Context, event *NotificationEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	r.events = append(r.events, *event)
	return nil
}

func (r *SimpleInMemoryRepo) GetEvents(ctx context.Context, filter NotificationFilter, limit, offset int) ([]NotificationEvent, int64, error) {
	return r.events, int64(len(r.events)), nil
}

func (r *SimpleInMemoryRepo) GetStats(ctx context.Context, filter NotificationFilter) (*NotificationStats, error) {
	return &NotificationStats{
		TotalSent:   int64(len(r.events)),
		ByChannel:   make(map[NotificationChannelType]int64),
		ByEventType: make(map[NotificationEventType]int64),
		LastUpdated: time.Now(),
	}, nil
}

// SimpleSlackHandler is a simple handler for testing
type SimpleSlackHandler struct{}

func (h *SimpleSlackHandler) Send(ctx context.Context, channel NotificationChannel, message NotificationMessage) error {
	return nil
}

func (h *SimpleSlackHandler) Test(ctx context.Context, channel NotificationChannel) error {
	return nil
}

func (h *SimpleSlackHandler) GetChannelType() NotificationChannelType {
	return ChannelTypeSlack
}