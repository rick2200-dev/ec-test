package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// ShippingHandler proxies shipment requests to the shipping service.
//
// Seller routes sit in the UI-only subtree (apitoken.Block) because
// registering a shipment triggers event publishing — too risky to expose
// to API tokens in v1. Buyer routes sit under the JWT-only /buyer group.
type ShippingHandler struct {
	shipping *proxy.ServiceClient
}

// NewShippingHandler creates a new ShippingHandler.
func NewShippingHandler(svc *proxy.Services) *ShippingHandler {
	return &ShippingHandler{shipping: svc.Shipping}
}

// ListShipments handles GET /seller/shipments
func (h *ShippingHandler) ListShipments(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.shipping.Get(r.Context(), "/seller/shipments", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to shipping failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "shipping service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetShipment handles GET /seller/shipments/{id}
func (h *ShippingHandler) GetShipment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.shipping.Get(r.Context(), "/seller/shipments/"+url.PathEscape(id), "")
	if err != nil {
		slog.Error("proxy to shipping failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "shipping service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// RegisterShipment handles POST /seller/shipments/{id}/register
func (h *ShippingHandler) RegisterShipment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.shipping.Post(r.Context(), "/seller/shipments/"+url.PathEscape(id)+"/register", r.Body)
	if err != nil {
		slog.Error("proxy to shipping failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "shipping service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// MarkShipmentDelivered handles POST /seller/shipments/{id}/deliver
func (h *ShippingHandler) MarkShipmentDelivered(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	body, status, err := h.shipping.Post(r.Context(), "/seller/shipments/"+url.PathEscape(id)+"/deliver", r.Body)
	if err != nil {
		slog.Error("proxy to shipping failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "shipping service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetSellerOrderShipment handles GET /seller/orders/{order_id}/shipment
func (h *ShippingHandler) GetSellerOrderShipment(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "order_id")
	body, status, err := h.shipping.Get(r.Context(), "/seller/orders/"+url.PathEscape(orderID)+"/shipment", "")
	if err != nil {
		slog.Error("proxy to shipping failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "shipping service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetBuyerOrderShipment handles GET /buyer/orders/{order_id}/shipment
func (h *ShippingHandler) GetBuyerOrderShipment(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "order_id")
	body, status, err := h.shipping.Get(r.Context(), "/buyer/orders/"+url.PathEscape(orderID)+"/shipment", "")
	if err != nil {
		slog.Error("proxy to shipping failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "shipping service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
