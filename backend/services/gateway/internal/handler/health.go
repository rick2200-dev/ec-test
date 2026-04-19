package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// HealthHandler provides health check endpoints.
type HealthHandler struct {
	svc *proxy.Services
}

// NewHealthHandler creates a new HealthHandler. svc may be nil for the
// bare liveness endpoint; readiness needs it to fan out to downstreams.
func NewHealthHandler(svc *proxy.Services) *HealthHandler {
	return &HealthHandler{svc: svc}
}

// Liveness returns 200 if the gateway process is running. It intentionally
// does not probe downstreams — kubernetes uses this to decide whether to
// restart the pod, which is orthogonal to downstream availability.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness pings every downstream service's /healthz concurrently and
// reports 503 if any of them is unreachable. This tells orchestrators
// the gateway should be pulled out of rotation when a downstream is
// flapping — upstream load balancers can then retry against another
// replica that might have fresher connections.
//
// We intentionally probe /healthz (liveness) rather than /readyz: readyz
// can flap on a single slow dep (DB, Redis) and that cascade is too noisy
// for the gateway's own probe. What we care about here is TCP reachability.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
		return
	}

	targets := map[string]*proxy.ServiceClient{
		"auth":         h.svc.Auth,
		"catalog":      h.svc.Catalog,
		"order":        h.svc.Order,
		"inventory":    h.svc.Inventory,
		"search":       h.svc.Search,
		"recommend":    h.svc.Recommend,
		"cart":         h.svc.Cart,
		"inquiry":      h.svc.Inquiry,
		"review":       h.svc.Review,
		"subscription": h.svc.Subscription,
		"shipping":     h.svc.Shipping,
		"coupon":       h.svc.Coupon,
		"loyalty":      h.svc.Loyalty,
		"notification": h.svc.Notification,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	type result struct {
		name string
		ok   bool
		err  string
	}
	resultsCh := make(chan result, len(targets))
	var wg sync.WaitGroup
	for name, client := range targets {
		if client == nil {
			continue
		}
		wg.Add(1)
		go func(name string, c *proxy.ServiceClient) {
			defer wg.Done()
			_, status, err := c.Get(ctx, "/healthz", "")
			r := result{name: name}
			switch {
			case err != nil:
				r.err = err.Error()
			case status < 200 || status >= 300:
				// Anything outside 2xx is unhealthy: 4xx means the
				// gateway is hitting the wrong URL or the downstream
				// has rejected the probe (missing auth, 404 for bad
				// base URL, etc.). 5xx is the downstream crashing.
				// Either way, traffic won't survive — fail readiness
				// so upstream load balancers drain us.
				r.err = http.StatusText(status)
			default:
				r.ok = true
			}
			resultsCh <- r
		}(name, client)
	}
	wg.Wait()
	close(resultsCh)

	services := make(map[string]string, len(targets))
	allOK := true
	for res := range resultsCh {
		if res.ok {
			services[res.name] = "ok"
			continue
		}
		allOK = false
		if res.err == "" {
			services[res.name] = "unavailable"
		} else {
			services[res.name] = "unavailable: " + res.err
		}
	}

	status := http.StatusOK
	overall := "ready"
	if !allOK {
		status = http.StatusServiceUnavailable
		overall = "degraded"
	}
	httputil.JSON(w, status, map[string]any{
		"status":   overall,
		"services": services,
	})
}
