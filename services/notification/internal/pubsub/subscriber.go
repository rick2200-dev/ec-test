package pubsub

import (
	"context"
	"fmt"
	"log/slog"

	gcppubsub "cloud.google.com/go/pubsub"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
)

// GCPSubscriber implements pubsub.Subscriber using Google Cloud Pub/Sub.
type GCPSubscriber struct {
	client *gcppubsub.Client
}

// NewGCPSubscriber creates a new GCPSubscriber.
// When PUBSUB_EMULATOR_HOST is set, the client automatically connects to the emulator.
func NewGCPSubscriber(ctx context.Context, projectID string) (*GCPSubscriber, error) {
	client, err := gcppubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}
	return &GCPSubscriber{client: client}, nil
}

// Subscribe starts receiving messages from the given subscription and dispatching them to the handler.
// This method blocks until the context is cancelled.
func (s *GCPSubscriber) Subscribe(ctx context.Context, subscription string, handler pubsub.Handler) error {
	sub := s.client.Subscription(subscription)

	slog.Info("subscribing to pubsub", "subscription", subscription)

	return sub.Receive(ctx, func(ctx context.Context, msg *gcppubsub.Message) {
		event, err := pubsub.Decode(msg.Data)
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
		slog.Info("event processed successfully",
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
