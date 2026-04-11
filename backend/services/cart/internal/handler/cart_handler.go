package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/cart/internal/service"
)

// CartHandler exposes the cart REST API under /cart.
type CartHandler struct {
	svc *service.CartService
}

// NewCartHandler creates a CartHandler.
func NewCartHandler(svc *service.CartService) *CartHandler {
	return &CartHandler{svc: svc}
}

// Routes returns the chi router for cart endpoints.
func (h *CartHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	r.Delete("/", h.Clear)
	r.Post("/items", h.AddItem)
	r.Put("/items/{skuId}", h.UpdateItem)
	r.Delete("/items/{skuId}", h.RemoveItem)
	r.Post("/checkout", h.Checkout)
	return r
}

// Get handles GET /cart.
func (h *CartHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	cart, err := h.svc.GetCart(r.Context(), tc.TenantID, tc.UserID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, cart)
}

// Clear handles DELETE /cart.
func (h *CartHandler) Clear(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	cart, err := h.svc.ClearCart(r.Context(), tc.TenantID, tc.UserID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, cart)
}

// addItemRequest is the payload for POST /cart/items.
type addItemRequest struct {
	SKUID    uuid.UUID `json:"sku_id"`
	Quantity int       `json:"quantity"`
}

// AddItem handles POST /cart/items.
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	var req addItemRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	cart, err := h.svc.AddItem(r.Context(), tc.TenantID, tc.UserID, req.SKUID, req.Quantity)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, cart)
}

// updateItemRequest is the payload for PUT /cart/items/{skuId}.
type updateItemRequest struct {
	Quantity int `json:"quantity"`
}

// UpdateItem handles PUT /cart/items/{skuId}.
func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	skuID, err := uuid.Parse(chi.URLParam(r, "skuId"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid sku id"))
		return
	}

	var req updateItemRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	cart, err := h.svc.UpdateItemQuantity(r.Context(), tc.TenantID, tc.UserID, skuID, req.Quantity)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, cart)
}

// RemoveItem handles DELETE /cart/items/{skuId}.
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	skuID, err := uuid.Parse(chi.URLParam(r, "skuId"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid sku id"))
		return
	}

	cart, err := h.svc.RemoveItem(r.Context(), tc.TenantID, tc.UserID, skuID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, cart)
}

// checkoutRequest is the payload for POST /cart/checkout.
type checkoutRequest struct {
	ShippingAddress json.RawMessage `json:"shipping_address"`
	Currency        string          `json:"currency"`
}

// Checkout handles POST /cart/checkout.
func (h *CartHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	var req checkoutRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	if len(req.ShippingAddress) == 0 {
		httputil.Error(w, apperrors.BadRequest("shipping_address is required"))
		return
	}

	result, err := h.svc.Checkout(r.Context(), tc.TenantID, tc.UserID, req.ShippingAddress, req.Currency)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}
