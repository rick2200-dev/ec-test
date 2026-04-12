package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/port"
)

// OrderHandler handles HTTP requests for order operations.
type OrderHandler struct {
	svc port.OrderUseCase
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(svc port.OrderUseCase) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// Routes returns the chi router for order endpoints.
//
// Note: POST /orders is intentionally not registered here. All buyer purchases
// now flow through the cart service (POST /api/v1/buyer/cart/checkout), which
// calls the order service's internal POST /internal/checkouts endpoint. The
// legacy single-seller Create handler is retained on this struct only so the
// deprecated gRPC CreateOrder RPC has an implementation to delegate to.
func (h *OrderHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/buyer", h.ListBuyerOrders)
	r.Get("/seller", h.ListSellerOrders)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}/status", h.UpdateStatus)
	return r
}

// createOrderRequest is the request body for creating an order.
type createOrderRequest struct {
	SellerID        uuid.UUID              `json:"seller_id"`
	Lines           []createOrderLineReq   `json:"lines"`
	ShippingAddress map[string]interface{} `json:"shipping_address"`
	Currency        string                 `json:"currency"`
}

type createOrderLineReq struct {
	SKUID       uuid.UUID `json:"sku_id"`
	ProductName string    `json:"product_name"`
	SKUCode     string    `json:"sku_code"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
}

// createOrderResponse is the response body for order creation.
type createOrderResponse struct {
	Order              *domain.OrderWithLines `json:"order"`
	StripeClientSecret string                 `json:"stripe_client_secret"`
}

// Create handles POST /orders.
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant context required"})
		return
	}

	var req createOrderRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	// Marshal shipping address to JSON.
	shippingJSON, err := marshalJSON(req.ShippingAddress)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid shipping address"})
		return
	}

	var lines []domain.OrderLineInput
	for _, l := range req.Lines {
		lines = append(lines, domain.OrderLineInput{
			SKUID:       l.SKUID,
			ProductName: l.ProductName,
			SKUCode:     l.SKUCode,
			Quantity:    l.Quantity,
			UnitPrice:   l.UnitPrice,
		})
	}

	input := domain.CreateOrderInput{
		SellerID:        req.SellerID,
		BuyerAuth0ID:    tc.UserID,
		Lines:           lines,
		ShippingAddress: shippingJSON,
		Currency:        req.Currency,
	}

	order, clientSecret, err := h.svc.CreateOrder(r.Context(), tc.TenantID, input)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusCreated, createOrderResponse{
		Order:              order,
		StripeClientSecret: clientSecret,
	})
}

// GetByID handles GET /orders/{id}.
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid order id"})
		return
	}

	order, err := h.svc.GetOrder(r.Context(), tenantID, id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, order)
}

// ListBuyerOrders handles GET /orders/buyer.
func (h *OrderHandler) ListBuyerOrders(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant context required"})
		return
	}

	p := pagination.FromRequest(r)

	orders, total, err := h.svc.ListBuyerOrders(r.Context(), tc.TenantID, tc.UserID, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	resp := pagination.Response[domain.Order]{
		Items:  orders,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// ListSellerOrders handles GET /orders/seller.
func (h *OrderHandler) ListSellerOrders(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant context required"})
		return
	}

	if tc.SellerID == nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "seller context required"})
		return
	}

	p := pagination.FromRequest(r)
	status := r.URL.Query().Get("status")

	orders, total, err := h.svc.ListSellerOrders(r.Context(), tc.TenantID, *tc.SellerID, status, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	resp := pagination.Response[domain.Order]{
		Items:  orders,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// updateStatusRequest is the request body for updating order status.
type updateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus handles PUT /orders/{id}/status.
//
// Seller-only endpoint: the caller must have seller context set in the
// tenant context (SellerID != nil), and the service layer additionally
// enforces that the seller owns the target order before mutating it.
// Prior to this handler being tightened, any authenticated tenant
// caller could advance any order's status — see the parallel fix in
// service.UpdateOrderStatus.
func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.Forbidden("seller context required"))
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid order id"})
		return
	}

	var req updateStatusRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	if err := h.svc.UpdateOrderStatus(r.Context(), tc.TenantID, *tc.SellerID, id, req.Status); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

// marshalJSON is a helper to convert a map to json.RawMessage.
func marshalJSON(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}
