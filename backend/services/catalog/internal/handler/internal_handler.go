package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/service"
)

// InternalHandler exposes intra-cluster endpoints that bypass the API
// gateway (no JWT, relies on cluster network isolation). Currently used
// by the cart service to snapshot SKU details at add-to-cart time.
type InternalHandler struct {
	svc *service.CatalogService
}

// NewInternalHandler creates a new InternalHandler.
func NewInternalHandler(svc *service.CatalogService) *InternalHandler {
	return &InternalHandler{svc: svc}
}

// Routes returns the chi router for /internal endpoints.
func (h *InternalHandler) Routes() chi.Router {
	r := chi.NewRouter()
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
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, sku)
}
