package pubsub

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
)

// GCPPublisher implements the Publisher interface using Google Cloud Pub/Sub.
type GCPPublisher struct {
	client *pubsub.Client
}

// NewGCPPublisher creates a new GCPPublisher.
// If PUBSUB_EMULATOR_HOST is set, the underlying GCP library connects to the
// emulator automatically.
func NewGCPPublisher(ctx context.Context, projectID string) (*GCPPublisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}
	return &GCPPublisher{client: client}, nil
}

// Publish encodes the event to JSON and publishes it to the named topic.
func (p *GCPPublisher) Publish(ctx context.Context, topic string, event Event) error {
	data, err := Encode(event)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}

	// In v2 the Topic API was replaced by Publisher. We create a Publisher
	// per call and Stop() it once the synchronous Get below returns so the
	// background flushing goroutine is released.
	publisher := p.client.Publisher(topic)
	defer publisher.Stop()

	result := publisher.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"event_type": event.Type,
			"tenant_id":  event.TenantID,
		},
	})

	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish event to %s: %w", topic, err)
	}
	return nil
}

// Close closes the underlying Pub/Sub client.
func (p *GCPPublisher) Close() error {
	return p.client.Close()
}
