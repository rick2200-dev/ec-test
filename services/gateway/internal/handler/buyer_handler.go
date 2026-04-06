package handler

import (
	"net/http"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

// BuyerHandler handles buyer-facing routes.
type BuyerHandler struct {
	// TODO: add service clients for catalog, order, search, etc.
}

// NewBuyerHandler creates a new BuyerHandler.
func NewBuyerHandler() *BuyerHandler {
	return &BuyerHandler{}
}

// ListProducts proxies to the catalog service to list products.
// GET /products
func (h *BuyerHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"products": []any{},
		"message":  "stub: proxy to catalog service",
	})
}

// GetProduct proxies to the catalog service to get a single product by slug.
// GET /products/{slug}
func (h *BuyerHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"slug":    slug,
		"message": "stub: proxy to catalog service",
	})
}

// SearchProducts proxies to the search service.
// GET /search
func (h *BuyerHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"results": []any{},
		"message": "stub: proxy to search service",
	})
}

// CreateOrder proxies to the order service.
// POST /orders
func (h *BuyerHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusAccepted, map[string]any{
		"message": "stub: proxy to order service",
	})
}

// ListOrders proxies to the order service to list the buyer's orders.
// GET /orders
func (h *BuyerHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"orders":  []any{},
		"message": "stub: proxy to order service",
	})
}
