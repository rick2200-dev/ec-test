package handler

import (
	"context"
	"net/http"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

// HealthHandler handles liveness and readiness probes.
type HealthHandler struct {
	redis *goredis.Client
}

// NewHealthHandler creates a HealthHandler that pings Redis for readiness.
func NewHealthHandler(redis *goredis.Client) *HealthHandler {
	return &HealthHandler{redis: redis}
}

// Liveness handles GET /healthz.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness handles GET /readyz. Reports unavailable if Redis cannot be
// pinged within 2 seconds.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.redis.Ping(ctx).Err(); err != nil {
		httputil.JSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"error":  err.Error(),
		})
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
