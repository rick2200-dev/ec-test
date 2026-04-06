package handler

import (
	"net/http"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

// SellerHandler handles seller-facing routes.
type SellerHandler struct {
	// TODO: add service clients for catalog, order, inventory, etc.
}

// NewSellerHandler creates a new SellerHandler.
func NewSellerHandler() *SellerHandler {
	return &SellerHandler{}
}

// ListProducts lists the seller's own products.
// GET /products
func (h *SellerHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"products": []any{},
		"message":  "stub: proxy to catalog service",
	})
}

// CreateProduct creates a new product for the seller.
// POST /products
func (h *SellerHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusCreated, map[string]any{
		"message": "stub: proxy to catalog service",
	})
}

// UpdateProduct updates a seller's product.
// PUT /products/{id}
func (h *SellerHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"message": "stub: proxy to catalog service",
	})
}

// ListOrders lists orders for the seller.
// GET /orders
func (h *SellerHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"orders":  []any{},
		"message": "stub: proxy to order service",
	})
}

// UpdateOrderStatus updates the status of an order.
// PUT /orders/{id}/status
func (h *SellerHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"message": "stub: proxy to order service",
	})
}

// ListInventory lists inventory for the seller.
// GET /inventory
func (h *SellerHandler) ListInventory(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]any{
		"inventory": []any{},
		"message":   "stub: proxy to inventory service",
	})
}

// UpdateStock updates stock for a specific SKU.
// PUT /inventory/{skuID}
func (h *SellerHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	skuID := r.PathValue("skuID")
	httputil.JSON(w, http.StatusOK, map[string]any{
		"sku_id":  skuID,
		"message": "stub: proxy to inventory service",
	})
}
