package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// AdminHandler handles platform admin routes.
type AdminHandler struct {
	auth         *proxy.ServiceClient
	subscription *proxy.ServiceClient
	order        *proxy.ServiceClient
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(svc *proxy.Services) *AdminHandler {
	return &AdminHandler{
		auth:         svc.Auth,
		subscription: svc.Subscription,
		order:        svc.Order,
	}
}

// ListTenants lists all tenants.
// GET /tenants
func (h *AdminHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.auth.Get(r.Context(), "/tenants", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// CreateTenant creates a new tenant.
// POST /tenants
func (h *AdminHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.auth.Post(r.Context(), "/tenants", r.Body)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListSellers lists all sellers.
// GET /sellers
func (h *AdminHandler) ListSellers(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.auth.Get(r.Context(), "/sellers", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ApproveSeller approves a seller.
// PUT /sellers/{id}/approve
func (h *AdminHandler) ApproveSeller(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, status, err := h.auth.Put(r.Context(), "/sellers/"+url.PathEscape(id)+"/approve", r.Body)
	if err != nil {
		slog.Error("proxy to auth failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListCommissions proxies the commission-rule catalog owned by order-svc.
// GET /commissions
func (h *AdminHandler) ListCommissions(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.order.Get(r.Context(), "/commissions", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// CreateCommission forwards rule creation to order-svc.
// POST /commissions
func (h *AdminHandler) CreateCommission(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.order.Post(r.Context(), "/commissions", r.Body)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListPlans lists all subscription plans.
// GET /plans
func (h *AdminHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.subscription.Get(r.Context(), "/plans", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to subscription failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "subscription service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// CreatePlan creates a new subscription plan.
// POST /plans
func (h *AdminHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.subscription.Post(r.Context(), "/plans", r.Body)
	if err != nil {
		slog.Error("proxy to subscription failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "subscription service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// UpdatePlan updates a subscription plan.
// PUT /plans/{id}
func (h *AdminHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, status, err := h.subscription.Put(r.Context(), "/plans/"+url.PathEscape(id), r.Body)
	if err != nil {
		slog.Error("proxy to subscription failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "subscription service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListBuyerPlans lists all buyer subscription plans.
// GET /buyer-plans
func (h *AdminHandler) ListBuyerPlans(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.subscription.Get(r.Context(), "/buyer-plans", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to subscription failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "subscription service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// CreateBuyerPlan creates a new buyer subscription plan.
// POST /buyer-plans
func (h *AdminHandler) CreateBuyerPlan(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.subscription.Post(r.Context(), "/buyer-plans", r.Body)
	if err != nil {
		slog.Error("proxy to subscription failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "subscription service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// UpdateBuyerPlan updates a buyer subscription plan.
// PUT /buyer-plans/{id}
func (h *AdminHandler) UpdateBuyerPlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, status, err := h.subscription.Put(r.Context(), "/buyer-plans/"+url.PathEscape(id), r.Body)
	if err != nil {
		slog.Error("proxy to subscription failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "subscription service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
