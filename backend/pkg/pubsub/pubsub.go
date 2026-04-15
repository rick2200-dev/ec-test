package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Event is the standard envelope for all Pub/Sub messages.
type Event struct {
	ID        string    `json:"event_id"`
	Type      string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// NewEvent creates a new event with a generated ID and current timestamp.
func NewEvent(eventType string, data any) Event {
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
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

// PublishProtoEvent publishes an event whose payload is defined by a proto
// contract (services/*/api/proto/*/events.proto). The payload is encoded as
// protojson with snake_case field names so the wire format stays compatible
// with existing JSON-based consumers. It is a no-op if publisher is nil.
func PublishProtoEvent(ctx context.Context, publisher Publisher, eventType, topic string, payload proto.Message) {
	if publisher == nil {
		return
	}
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(payload)
	if err != nil {
		slog.Warn("failed to marshal proto event payload", "event_type", eventType, "error", err)
		return
	}
	event := NewEvent(eventType, json.RawMessage(raw))
	if err := publisher.Publish(ctx, topic, event); err != nil {
		slog.Warn("failed to publish event", "event_type", eventType, "topic", topic, "error", err)
	}
}

// PublishEvent publishes an event, logging a warning on failure.
// It is a no-op if publisher is nil, making it safe for services
// that run without a Pub/Sub backend (e.g., in tests).
func PublishEvent(ctx context.Context, publisher Publisher, eventType, topic string, data any) {
	if publisher == nil {
		return
	}
	event := NewEvent(eventType, data)
	if err := publisher.Publish(ctx, topic, event); err != nil {
		slog.Warn("failed to publish event", "event_type", eventType, "topic", topic, "error", err)
	}
}
