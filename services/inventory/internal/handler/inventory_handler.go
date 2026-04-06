package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/service"
)

// InventoryHandler handles HTTP requests for inventory operations.
type InventoryHandler struct {
	svc *service.InventoryService
}

// NewInventoryHandler creates a new InventoryHandler.
func NewInventoryHandler(svc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

// Routes returns the chi router for inventory endpoints.
func (h *InventoryHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{skuID}", h.GetBySKUID)
	r.Put("/{skuID}", h.UpdateStock)
	r.Post("/{skuID}/reserve", h.Reserve)
	r.Post("/{skuID}/release", h.Release)
	r.Post("/{skuID}/confirm", h.ConfirmSold)
	return r
}

// List handles GET /inventory?seller_id=...
func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	sellerIDStr := r.URL.Query().Get("seller_id")
	if sellerIDStr == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "seller_id query parameter is required"})
		return
	}
	sellerID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seller_id"})
		return
	}

	p := pagination.FromRequest(r)

	items, total, err := h.svc.ListInventory(r.Context(), tenantID, sellerID, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	resp := pagination.Response[domain.Inventory]{
		Items:  items,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// GetBySKUID handles GET /inventory/{skuID}.
func (h *InventoryHandler) GetBySKUID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	skuID, err := uuid.Parse(chi.URLParam(r, "skuID"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sku id"})
		return
	}

	inv, err := h.svc.GetInventory(r.Context(), tenantID, skuID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, inv)
}

// updateStockRequest is the request body for PUT /inventory/{skuID}.
type updateStockRequest struct {
	SellerID          uuid.UUID `json:"seller_id"`
	QuantityAvailable int       `json:"quantity_available"`
	LowStockThreshold int       `json:"low_stock_threshold"`
}

// UpdateStock handles PUT /inventory/{skuID}.
func (h *InventoryHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	skuID, err := uuid.Parse(chi.URLParam(r, "skuID"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sku id"})
		return
	}

	var req updateStockRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	inv := &domain.Inventory{
		SKUID:             skuID,
		SellerID:          req.SellerID,
		QuantityAvailable: req.QuantityAvailable,
		LowStockThreshold: req.LowStockThreshold,
	}

	if err := h.svc.UpdateStock(r.Context(), tenantID, inv); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, inv)
}

// quantityRequest is the request body for reserve/release/confirm endpoints.
type quantityRequest struct {
	Quantity int `json:"quantity"`
}

// Reserve handles POST /inventory/{skuID}/reserve.
func (h *InventoryHandler) Reserve(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	skuID, err := uuid.Parse(chi.URLParam(r, "skuID"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sku id"})
		return
	}

	var req quantityRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.svc.ReserveStock(r.Context(), tenantID, skuID, req.Quantity); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "reserved"})
}

// Release handles POST /inventory/{skuID}/release.
func (h *InventoryHandler) Release(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	skuID, err := uuid.Parse(chi.URLParam(r, "skuID"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sku id"})
		return
	}

	var req quantityRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.svc.ReleaseStock(r.Context(), tenantID, skuID, req.Quantity); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "released"})
}

// ConfirmSold handles POST /inventory/{skuID}/confirm.
func (h *InventoryHandler) ConfirmSold(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	skuID, err := uuid.Parse(chi.URLParam(r, "skuID"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sku id"})
		return
	}

	var req quantityRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.svc.ConfirmSold(r.Context(), tenantID, skuID, req.Quantity); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "sold_confirmed"})
}
