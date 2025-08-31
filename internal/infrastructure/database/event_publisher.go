package database

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/agentscan/agentscan/pkg/errors"
)

// EventPublisher publishes repository events
type EventPublisher struct {
	eventBusURL string
	listeners   map[string][]EventHandler
	mu          sync.RWMutex
	enabled     bool
}

// EventHandler handles repository events
type EventHandler interface {
	Handle(ctx context.Context, event *RepositoryEvent[interface{}]) error
	EventTypes() []string
	ID() string
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(eventBusURL string) *EventPublisher {
	return &EventPublisher{
		eventBusURL: eventBusURL,
		listeners:   make(map[string][]EventHandler),
		enabled:     true,
	}
}

// Publish publishes a repository event
func (ep *EventPublisher) Publish(ctx context.Context, event *RepositoryEvent[interface{}]) error {
	if !ep.enabled {
		return nil
	}

	ep.mu.RLock()
	handlers := ep.listeners[event.Type]
	ep.mu.RUnlock()

	// Publish to local handlers
	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h.Handle(ctx, event); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Event handler error: %v\n", err)
			}
		}(handler)
	}

	// Publish to external event bus if configured
	if ep.eventBusURL != "" {
		return ep.publishToEventBus(ctx, event)
	}

	return nil
}

// AddListener adds an event listener
func (ep *EventPublisher) AddListener(handler EventHandler) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	for _, eventType := range handler.EventTypes() {
		ep.listeners[eventType] = append(ep.listeners[eventType], handler)
	}

	return nil
}

// RemoveListener removes an event listener
func (ep *EventPublisher) RemoveListener(handlerID string) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	for eventType, handlers := range ep.listeners {
		var newHandlers []EventHandler
		for _, handler := range handlers {
			if handler.ID() != handlerID {
				newHandlers = append(newHandlers, handler)
			}
		}
		ep.listeners[eventType] = newHandlers
	}

	return nil
}

// Enable enables or disables event publishing
func (ep *EventPublisher) Enable(enabled bool) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.enabled = enabled
}

// HealthCheck checks if the event publisher is healthy
func (ep *EventPublisher) HealthCheck(ctx context.Context) error {
	if !ep.enabled {
		return nil
	}

	// If external event bus is configured, check its health
	if ep.eventBusURL != "" {
		// Implement health check for external event bus
		// This is a placeholder - implement based on your event bus
		return nil
	}

	return nil
}

// publishToEventBus publishes event to external event bus
func (ep *EventPublisher) publishToEventBus(ctx context.Context, event *RepositoryEvent[interface{}]) error {
	// Serialize event
	eventData, err := json.Marshal(event)
	if err != nil {
		return errors.NewInternalError("failed to serialize event").WithCause(err)
	}

	// Publish to external event bus
	// This is a placeholder - implement based on your event bus (Kafka, RabbitMQ, etc.)
	fmt.Printf("Publishing event to %s: %s\n", ep.eventBusURL, string(eventData))

	return nil
}

// Simple event handler implementations

// LoggingEventHandler logs all repository events
type LoggingEventHandler struct {
	id string
}

// NewLoggingEventHandler creates a new logging event handler
func NewLoggingEventHandler() *LoggingEventHandler {
	return &LoggingEventHandler{
		id: "logging_handler",
	}
}

// Handle handles repository events by logging them
func (leh *LoggingEventHandler) Handle(ctx context.Context, event *RepositoryEvent[interface{}]) error {
	eventData, _ := json.Marshal(event)
	fmt.Printf("Repository Event: %s\n", string(eventData))
	return nil
}

// EventTypes returns the event types this handler is interested in
func (leh *LoggingEventHandler) EventTypes() []string {
	return []string{"created", "updated", "deleted"}
}

// ID returns the handler ID
func (leh *LoggingEventHandler) ID() string {
	return leh.id
}

// MetricsEventHandler updates metrics based on repository events
type MetricsEventHandler struct {
	id      string
	metrics *RepositoryMetrics
}

// NewMetricsEventHandler creates a new metrics event handler
func NewMetricsEventHandler(metrics *RepositoryMetrics) *MetricsEventHandler {
	return &MetricsEventHandler{
		id:      "metrics_handler",
		metrics: metrics,
	}
}

// Handle handles repository events by updating metrics
func (meh *MetricsEventHandler) Handle(ctx context.Context, event *RepositoryEvent[interface{}]) error {
	// Update metrics based on event type
	switch event.Type {
	case "created":
		// Could track creation events separately
	case "updated":
		// Could track update events separately
	case "deleted":
		// Could track deletion events separately
	}
	return nil
}

// EventTypes returns the event types this handler is interested in
func (meh *MetricsEventHandler) EventTypes() []string {
	return []string{"created", "updated", "deleted"}
}

// ID returns the handler ID
func (meh *MetricsEventHandler) ID() string {
	return meh.id
}