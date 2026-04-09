package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// SellerHandler handles seller-facing routes.
type SellerHandler struct {
	catalog   *proxy.ServiceClient
	order     *proxy.ServiceClient
	inventory *proxy.ServiceClient
	auth      *proxy.ServiceClient
}

// NewSellerHandler creates a new SellerHandler.
func NewSellerHandler(svc *proxy.Services) *SellerHandler {
	return &SellerHandler{
		catalog:   svc.Catalog,
		order:     svc.Order,
		inventory: svc.Inventory,
		auth:      svc.Auth,
	}
}

// ListProducts lists the seller's own products.
// GET /products
func (h *SellerHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant context"})
		return
	}
	q := r.URL.Query()
	if tc.SellerID != nil {
		q.Set("seller_id", tc.SellerID.String())
	}
	body, status, pErr := h.catalog.Get(r.Context(), "/products", q.Encode())
	if pErr != nil {
		slog.Error("proxy to catalog failed", "error", pErr)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "catalog service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// CreateProduct creates a new product for the seller.
// POST /products
func (h *SellerHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.catalog.Post(r.Context(), "/products", r.Body)
	if err != nil {
		slog.Error("proxy to catalog failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "catalog service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// UpdateProduct updates a seller's product.
// PUT /products/{id}
func (h *SellerHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, status, err := h.catalog.Put(r.Context(), "/products/"+url.PathEscape(id), r.Body)
	if err != nil {
		slog.Error("proxy to catalog failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "catalog service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListOrders lists orders for the seller.
// GET /orders
func (h *SellerHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.order.Get(r.Context(), "/orders/seller", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// UpdateOrderStatus updates the status of an order.
// PUT /orders/{id}/status
func (h *SellerHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, status, err := h.order.Put(r.Context(), "/orders/"+url.PathEscape(id)+"/status", r.Body)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListInventory lists inventory for the seller.
// GET /inventory
func (h *SellerHandler) ListInventory(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing tenant context"})
		return
	}
	q := r.URL.Query()
	if tc.SellerID != nil {
		q.Set("seller_id", tc.SellerID.String())
	}
	body, status, pErr := h.inventory.Get(r.Context(), "/inventory", q.Encode())
	if pErr != nil {
		slog.Error("proxy to inventory failed", "error", pErr)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inventory service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// UpdateStock updates stock for a specific SKU.
// PUT /inventory/{skuID}
func (h *SellerHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	skuID := r.PathValue("skuID")
	body, status, err := h.inventory.Put(r.Context(), "/inventory/"+url.PathEscape(skuID), r.Body)
	if err != nil {
		slog.Error("proxy to inventory failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "inventory service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListPlans lists all available subscription plans.
// GET /plans
func (h *SellerHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	body, status, pErr := h.auth.Get(r.Context(), "/plans", r.URL.RawQuery)
	if pErr != nil {
		slog.Error("proxy to auth failed", "error", pErr)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetSubscription retrieves the current subscription for the seller.
// GET /subscription
func (h *SellerHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil || tc.SellerID == nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	body, status, pErr := h.auth.Get(r.Context(), "/subscriptions/sellers/"+tc.SellerID.String(), "")
	if pErr != nil {
		slog.Error("proxy to auth failed", "error", pErr)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Subscribe subscribes the seller to a plan.
// POST /subscription
func (h *SellerHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil || tc.SellerID == nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
		return
	}
	body, status, pErr := h.auth.Post(r.Context(), "/subscriptions/sellers/"+tc.SellerID.String(), r.Body)
	if pErr != nil {
		slog.Error("proxy to auth failed", "error", pErr)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "auth service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
