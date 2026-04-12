package pubsub

import (
	"context"
	"fmt"
	"log/slog"

	//nolint:staticcheck // SA1019: pubsub v2 migration is tracked separately; using v1 consistently across services
	gcppubsub "cloud.google.com/go/pubsub"
)

// GCPSubscriber implements the Subscriber interface using Google Cloud Pub/Sub.
//
// It is the shared implementation that the notification, inventory, recommend,
// and any future subscriber-owning services use directly instead of each
// re-implementing the same decode/ack/nack/log dance. The publisher side uses
// pubsub v2 (see gcp.go); the subscriber side intentionally stays on v1 until
// the codebase-wide migration happens, so all consumers see the same API shape.
type GCPSubscriber struct {
	client *gcppubsub.Client
}

// NewGCPSubscriber creates a new GCPSubscriber.
//
// When PUBSUB_EMULATOR_HOST is set, the GCP client library automatically
// connects to the emulator, which is how local docker-compose environments
// are exercised.
func NewGCPSubscriber(ctx context.Context, projectID string) (*GCPSubscriber, error) {
	client, err := gcppubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}
	return &GCPSubscriber{client: client}, nil
}

// Subscribe starts receiving messages from the given subscription and
// dispatching them to handler. It blocks until ctx is cancelled.
//
// Decode errors and handler errors both Nack the message so the broker
// redelivers. Successful handling Acks. Structured slog entries carry the
// subscription name and event metadata so operators can trace what each
// service actually processed.
func (s *GCPSubscriber) Subscribe(ctx context.Context, subscription string, handler Handler) error {
	sub := s.client.Subscription(subscription)

	slog.Info("subscribing to pubsub", "subscription", subscription)

	return sub.Receive(ctx, func(ctx context.Context, msg *gcppubsub.Message) {
		event, err := Decode(msg.Data)
		if err != nil {
			slog.Error("failed to decode event",
				"subscription", subscription,
				"error", err,
			)
			msg.Nack()
			return
		}

		if err := handler(ctx, event); err != nil {
			slog.Error("failed to handle event",
				"subscription", subscription,
				"event_id", event.ID,
				"event_type", event.Type,
				"error", err,
			)
			msg.Nack()
			return
		}

		msg.Ack()
		slog.Debug("event processed successfully",
			"subscription", subscription,
			"event_id", event.ID,
			"event_type", event.Type,
		)
	})
}

// Close closes the underlying Pub/Sub client.
func (s *GCPSubscriber) Close() error {
	return s.client.Close()
}
