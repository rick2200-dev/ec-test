package handler

import (
	"net/http"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

// HealthHandler provides health check endpoints.
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Liveness returns 200 if the service is running.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness returns 200 if the service is ready to serve traffic.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	// TODO: check downstream service connectivity
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
