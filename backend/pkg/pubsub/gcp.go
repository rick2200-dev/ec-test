package pubsub

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Riku-KANO/ec-test/pkg/tracing"
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

// Publish encodes the event to JSON and publishes it to the named topic. A
// PRODUCER span wraps the call and the active trace context is injected into
// the outgoing message Attributes so downstream subscribers can continue the
// trace.
func (p *GCPPublisher) Publish(ctx context.Context, topic string, event Event) error {
	tracer := otel.Tracer("github.com/Riku-KANO/ec-test/pkg/pubsub")
	ctx, span := tracer.Start(ctx, "pubsub.publish "+topic,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "gcp_pubsub"),
			attribute.String("messaging.destination.name", topic),
			attribute.String("messaging.operation", "publish"),
			attribute.String("messaging.message.id", event.ID),
			attribute.String("messaging.message.type", event.Type),
		),
	)
	defer span.End()

	data, err := Encode(event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "encode failed")
		return fmt.Errorf("encode event: %w", err)
	}

	attrs := map[string]string{"event_type": event.Type}
	tracing.InjectAttrs(ctx, attrs)

	// In v2 the Topic API was replaced by Publisher. We create a Publisher
	// per call and Stop() it once the synchronous Get below returns so the
	// background flushing goroutine is released.
	publisher := p.client.Publisher(topic)
	defer publisher.Stop()

	result := publisher.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attrs,
	})

	if _, err := result.Get(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish failed")
		return fmt.Errorf("publish event to %s: %w", topic, err)
	}
	return nil
}

// Close closes the underlying Pub/Sub client.
func (p *GCPPublisher) Close() error {
	return p.client.Close()
}
