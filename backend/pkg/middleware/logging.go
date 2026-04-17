package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger is HTTP middleware that logs request details.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		// Use InfoContext so tracing.NewSlogHandler can read the active
		// span from r.Context() and tag the record with trace_id /
		// span_id — otherwise access logs cannot be cross-referenced
		// back to the Jaeger trace the request produced.
		slog.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start).String(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
