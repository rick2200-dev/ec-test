package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Event is the standard envelope for all Pub/Sub messages.
type Event struct {
	ID        string    `json:"event_id"`
	Type      string    `json:"event_type"`
	TenantID  string    `json:"tenant_id"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// NewEvent creates a new event with a generated ID and current timestamp.
func NewEvent(eventType string, tenantID uuid.UUID, data any) Event {
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		TenantID:  tenantID.String(),
		Timestamp: time.Now().UTC(),
		Data:      data,
	}
}

// Publisher publishes events to a topic.
type Publisher interface {
	Publish(ctx context.Context, topic string, event Event) error
	Close() error
}

// Handler processes a received event.
type Handler func(ctx context.Context, event Event) error

// Subscriber subscribes to events from a subscription.
type Subscriber interface {
	Subscribe(ctx context.Context, subscription string, handler Handler) error
	Close() error
}

// Encode serializes an event to JSON bytes.
func Encode(event Event) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}
	return data, nil
}

// Decode deserializes JSON bytes into an event.
func Decode(data []byte) (Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return Event{}, fmt.Errorf("unmarshal event: %w", err)
	}
	return event, nil
}

// PublishEvent publishes an event, logging a warning on failure.
// It is a no-op if publisher is nil, making it safe for services
// that run without a Pub/Sub backend (e.g., in tests).
func PublishEvent(ctx context.Context, publisher Publisher, tenantID uuid.UUID, eventType, topic string, data any) {
	if publisher == nil {
		return
	}
	event := NewEvent(eventType, tenantID, data)
	if err := publisher.Publish(ctx, topic, event); err != nil {
		slog.Warn("failed to publish event", "event_type", eventType, "topic", topic, "error", err)
	}
}
