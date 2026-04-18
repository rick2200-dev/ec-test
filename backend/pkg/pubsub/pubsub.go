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
//
// This is the fire-and-forget variant: publish failures are logged but the
// caller never sees them. Suitable for best-effort notifications where a
// lost event is tolerable. For cases that need publish-success-or-retry
// (e.g. webhook handlers that gate idempotency markers on publish success),
// use PublishProtoEventWithErr instead.
func PublishProtoEvent(ctx context.Context, publisher Publisher, eventType, topic string, payload proto.Message) {
	if err := PublishProtoEventWithErr(ctx, publisher, eventType, topic, payload); err != nil {
		slog.Warn("failed to publish event", "event_type", eventType, "topic", topic, "error", err)
	}
}

// PublishProtoEventWithErr is the error-returning variant of
// PublishProtoEvent. Callers use this when publish success drives a
// downstream state transition — e.g. stamping a
// `paid_event_published_at` column only after the event actually lands,
// so a Pub/Sub outage during the first webhook delivery doesn't leave
// subscribers permanently unaware that the order was paid.
//
// A nil publisher is treated as success (matches the fire-and-forget
// variant's "safe for tests without a broker" behavior). A marshal
// failure is returned as an error because a malformed payload is a
// caller bug, not an external broker hiccup.
func PublishProtoEventWithErr(ctx context.Context, publisher Publisher, eventType, topic string, payload proto.Message) error {
	if publisher == nil {
		return nil
	}
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal proto event: %w", err)
	}
	event := NewEvent(eventType, json.RawMessage(raw))
	return publisher.Publish(ctx, topic, event)
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
