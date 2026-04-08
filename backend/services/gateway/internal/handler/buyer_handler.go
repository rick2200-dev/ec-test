package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// BuyerHandler handles buyer-facing routes.
type BuyerHandler struct {
	catalog   *proxy.ServiceClient
	order     *proxy.ServiceClient
	recommend *proxy.ServiceClient
}

// NewBuyerHandler creates a new BuyerHandler.
func NewBuyerHandler(svc *proxy.Services) *BuyerHandler {
	return &BuyerHandler{
		catalog:   svc.Catalog,
		order:     svc.Order,
		recommend: svc.Recommend,
	}
}

// ListProducts proxies to the catalog service to list products.
// GET /products
func (h *BuyerHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	q.Set("status", "active")
	body, status, err := h.catalog.Get(r.Context(), "/products", q.Encode())
	if err != nil {
		slog.Error("proxy to catalog failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "catalog service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetProduct proxies to the catalog service to get a single product by slug.
// GET /products/{slug}
func (h *BuyerHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	body, status, err := h.catalog.Get(r.Context(), "/products/"+url.PathEscape(slug), "")
	if err != nil {
		slog.Error("proxy to catalog failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "catalog service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// SearchProducts proxies to the search service.
// GET /search
func (h *BuyerHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	// TODO: proxy to search service once it is ready
	q := r.URL.Query().Get("q")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"results": []any{},
		"message": "stub: search service not ready",
	})
}

// CreateOrder proxies to the order service.
// POST /orders
func (h *BuyerHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.order.Post(r.Context(), "/orders", r.Body)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// ListOrders proxies to the order service to list the buyer's orders.
// GET /orders
func (h *BuyerHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.order.Get(r.Context(), "/orders/buyer", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to order failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "order service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// TrackEvent proxies to the recommend service to record user behavior events.
// POST /events
func (h *BuyerHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.recommend.Post(r.Context(), "/events", r.Body)
	if err != nil {
		slog.Error("proxy to recommend failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "recommend service unavailable"})
		return
	}
	writeRaw(w, status, body)
}

// GetRecommendations proxies to the recommend service to fetch recommendations.
// GET /recommendations
func (h *BuyerHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	body, status, err := h.recommend.Get(r.Context(), "/recommendations", r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy to recommend failed", "error", err)
		httputil.JSON(w, http.StatusBadGateway, map[string]string{"error": "recommend service unavailable"})
		return
	}
	writeRaw(w, status, body)
}
