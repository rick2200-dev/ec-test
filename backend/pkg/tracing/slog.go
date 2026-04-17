package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// NewSlogHandler wraps an existing slog.Handler so every record emitted with a
// ctx carrying an active span gains trace_id and span_id attributes. This is
// the single integration point that links structured logs to Jaeger spans —
// operators can click a trace_id from a log line and see the full trace.
func NewSlogHandler(inner slog.Handler) slog.Handler {
	if inner == nil {
		return nil
	}
	return &slogHandler{inner: inner}
}

type slogHandler struct {
	inner slog.Handler
}

func (h *slogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *slogHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	return &slogHandler{inner: h.inner.WithGroup(name)}
}
