package main

import (
	"context"
	"fmt"

	//nolint:staticcheck // SA1019: pubsub v2 migration is tracked separately; using v1 consistently across services
	gcppubsub "cloud.google.com/go/pubsub"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
)

// gcpSub implements pubsub.Subscriber using Google Cloud Pub/Sub.
type gcpSub struct {
	client *gcppubsub.Client
}

// newGCPSub creates a new GCP Pub/Sub subscriber.
func newGCPSub(ctx context.Context, projectID string) (*gcpSub, error) {
	client, err := gcppubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}
	return &gcpSub{client: client}, nil
}

// Subscribe starts receiving messages from the named subscription and passes
// each decoded event to the handler.
func (s *gcpSub) Subscribe(ctx context.Context, subscription string, handler pubsub.Handler) error {
	sub := s.client.Subscription(subscription)

	return sub.Receive(ctx, func(_ context.Context, msg *gcppubsub.Message) {
		event, err := pubsub.Decode(msg.Data)
		if err != nil {
			msg.Nack()
			return
		}

		if err := handler(ctx, event); err != nil {
			msg.Nack()
			return
		}

		msg.Ack()
	})
}

// Close closes the underlying Pub/Sub client.
func (s *gcpSub) Close() error {
	return s.client.Close()
}
