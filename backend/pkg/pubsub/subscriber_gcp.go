package pubsub

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	//nolint:staticcheck // SA1019: pubsub v2 migration is tracked separately; using v1 consistently across services
	gcppubsub "cloud.google.com/go/pubsub"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Riku-KANO/ec-test/pkg/tracing"
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
//
// A CONSUMER span wraps each message. The trace context is extracted from the
// message Attributes so the span links back to the upstream producer.
func (s *GCPSubscriber) Subscribe(ctx context.Context, subscription string, handler Handler) error {
	sub := s.client.Subscription(subscription)

	slog.Info("subscribing to pubsub", "subscription", subscription)

	tracer := otel.Tracer("github.com/Riku-KANO/ec-test/pkg/pubsub")

	return sub.Receive(ctx, func(ctx context.Context, msg *gcppubsub.Message) {
		ctx = tracing.ExtractAttrs(ctx, msg.Attributes)
		ctx, span := tracer.Start(ctx, "pubsub.process "+subscription,
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				attribute.String("messaging.system", "gcp_pubsub"),
				attribute.String("messaging.source.name", subscription),
				attribute.String("messaging.operation", "process"),
				attribute.String("messaging.message.id", msg.ID),
			),
		)
		defer span.End()

		event, err := Decode(msg.Data)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "decode failed")
			slog.ErrorContext(ctx, "failed to decode event",
				"subscription", subscription,
				"error", err,
			)
			msg.Nack()
			return
		}

		span.SetAttributes(
			attribute.String("messaging.message.event_id", event.ID),
			attribute.String("messaging.message.event_type", event.Type),
		)

		if err := handler(ctx, event); err != nil {
			// If the handler returns a RetryAfterError, sleep for the requested
			// delay before nacking. The GCP client library automatically extends
			// the ack deadline while this callback is running, so the message is
			// not redelivered during the sleep. This avoids tight nack/redeliver
			// loops (e.g., while a concurrent processing claim is still active).
			var retryErr *RetryAfterError
			if errors.As(err, &retryErr) && retryErr.Delay > 0 {
				slog.InfoContext(ctx, "handler requested delayed retry, sleeping before nack",
					"subscription", subscription,
					"event_id", event.ID,
					"delay", retryErr.Delay,
				)
				select {
				case <-time.After(retryErr.Delay):
				case <-ctx.Done():
				}
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, "handler failed")
			slog.ErrorContext(ctx, "failed to handle event",
				"subscription", subscription,
				"event_id", event.ID,
				"event_type", event.Type,
				"error", err,
			)
			msg.Nack()
			return
		}

		msg.Ack()
		slog.DebugContext(ctx, "event processed successfully",
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
