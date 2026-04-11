package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// CartHandler proxies buyer cart requests to the cart micro-service. The
// cart service owns the Redis-backed cart and orchestrates multi-seller
// checkout through the order service.
type CartHandler struct {
	cart *proxy.ServiceClient
}

// NewCartHandler creates a new CartHandler.
func NewCartHandler(svc *proxy.Services) *CartHandler {
	return &CartHandler{cart: svc.Cart}
}

// Get proxies GET /cart.
func (h *CartHandler) Get(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.cart.Get(r.Context(), "/cart", "")
	if err != nil {
		slog.Error("proxy to cart failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "cart service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// AddItem proxies POST /cart/items.
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.cart.Post(r.Context(), "/cart/items", r.Body)
	if err != nil {
		slog.Error("proxy to cart failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "cart service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// UpdateItem proxies PUT /cart/items/{skuId}.
func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	skuID := chi.URLParam(r, "skuId")
	body, status, err := h.cart.Put(r.Context(), "/cart/items/"+url.PathEscape(skuID), r.Body)
	if err != nil {
		slog.Error("proxy to cart failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "cart service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// RemoveItem proxies DELETE /cart/items/{skuId}.
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	skuID := chi.URLParam(r, "skuId")
	body, status, err := h.cart.Delete(r.Context(), "/cart/items/"+url.PathEscape(skuID))
	if err != nil {
		slog.Error("proxy to cart failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "cart service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Clear proxies DELETE /cart.
func (h *CartHandler) Clear(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.cart.Delete(r.Context(), "/cart")
	if err != nil {
		slog.Error("proxy to cart failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "cart service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// Checkout proxies POST /cart/checkout.
func (h *CartHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.cart.Post(r.Context(), "/cart/checkout", r.Body)
	if err != nil {
		slog.Error("proxy to cart failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "cart service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
