package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// InjectAttrs writes the active trace context into the given Pub/Sub message
// Attributes map using the globally configured propagator (W3C TraceContext +
// Baggage). The caller is responsible for setting the mutated map on the
// outgoing message.
func InjectAttrs(ctx context.Context, attrs map[string]string) {
	if attrs == nil {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(attrs))
}

// ExtractAttrs reads trace context from a Pub/Sub message's Attributes and
// returns a context with the upstream span as parent. Safe to call with a nil
// or empty map — the returned context is unchanged in that case.
func ExtractAttrs(ctx context.Context, attrs map[string]string) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(attrs))
}
