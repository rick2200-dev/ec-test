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
	return r
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
