package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	pkgmiddleware "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/port"
)

// InternalHandler exposes intra-cluster endpoints that bypass the API
// gateway. Routes are gated by a shared-secret X-Internal-Token header in
// addition to any cluster-level network isolation, so an accidental
// exposure of this port does not leak catalog data to the open Internet.
// Currently used by the cart service to snapshot SKU details at
// add-to-cart time.
type InternalHandler struct {
	svc    port.CatalogUseCase
	secret string
}

// NewInternalHandler creates a new InternalHandler. The secret is checked
// against the X-Internal-Token header on every request; an empty secret
// causes the middleware to fail closed with 503.
func NewInternalHandler(svc port.CatalogUseCase, secret string) *InternalHandler {
	return &InternalHandler{svc: svc, secret: secret}
}

// Routes returns the chi router for /internal endpoints.
func (h *InternalHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(pkgmiddleware.RequireInternalToken(h.secret))
	r.Get("/skus/{id}", h.GetSKU)
	r.Get("/products/{id}", h.GetProduct)
	return r
}

// productLookupResponse is the shape returned by GET /internal/products/{id}.
// Used by the review service to resolve product details and SKU IDs for
// purchase verification.
type productLookupResponse struct {
	ProductID   uuid.UUID   `json:"product_id"`
	SellerID    uuid.UUID   `json:"seller_id"`
	ProductName string      `json:"product_name"`
	SKUIDs      []uuid.UUID `json:"sku_ids"`
}

// GetProduct handles GET /internal/products/{id}. Returns product metadata
// and the list of SKU IDs so callers can perform purchase verification
// against the order service without needing catalog domain knowledge.
func (h *InternalHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid product id"))
		return
	}

	product, err := h.svc.GetProductByID(r.Context(), tenantID, productID)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	skus, err := h.svc.ListSKUs(r.Context(), tenantID, productID)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	skuIDs := make([]uuid.UUID, 0, len(skus))
	for _, s := range skus {
		skuIDs = append(skuIDs, s.ID)
	}

	httputil.JSON(w, http.StatusOK, productLookupResponse{
		ProductID:   product.ID,
		SellerID:    product.SellerID,
		ProductName: product.Name,
		SKUIDs:      skuIDs,
	})
}

// GetSKU handles GET /internal/skus/{id}. Requires X-Tenant-ID header.
func (h *InternalHandler) GetSKU(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	skuID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid sku id"))
		return
	}

	sku, err := h.svc.GetSKUWithProductName(r.Context(), tenantID, skuID)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, sku)
}
