package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/service"
)

// InternalHandler exposes intra-cluster endpoints that bypass the API
// gateway. Currently used by the cart service to create multi-seller
// checkouts (one PaymentIntent covering N orders across N sellers).
type InternalHandler struct {
	svc *service.OrderService
}

// NewInternalHandler creates a new InternalHandler.
func NewInternalHandler(svc *service.OrderService) *InternalHandler {
	return &InternalHandler{svc: svc}
}

// Routes returns the chi router for /internal endpoints.
func (h *InternalHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/checkouts", h.CreateCheckout)
	return r
}

// checkoutRequest is the request body for POST /internal/checkouts.
type checkoutRequest struct {
	BuyerAuth0ID    string            `json:"buyer_auth0_id"`
	Lines           []checkoutLineReq `json:"lines"`
	ShippingAddress map[string]any    `json:"shipping_address"`
	Currency        string            `json:"currency"`
}

type checkoutLineReq struct {
	SKUID       uuid.UUID `json:"sku_id"`
	SellerID    uuid.UUID `json:"seller_id"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
	ProductName string    `json:"product_name"`
	SKUCode     string    `json:"sku_code"`
}

// checkoutResponse is the response body for POST /internal/checkouts.
type checkoutResponse struct {
	OrderIDs              []string `json:"order_ids"`
	StripeClientSecret    string   `json:"stripe_client_secret"`
	StripePaymentIntentID string   `json:"stripe_payment_intent_id"`
	TotalAmount           int64    `json:"total_amount"`
	Currency              string   `json:"currency"`
}

// CreateCheckout handles POST /internal/checkouts.
func (h *InternalHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	var req checkoutRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	if req.BuyerAuth0ID == "" {
		httputil.Error(w, apperrors.BadRequest("buyer_auth0_id is required"))
		return
	}

	shippingJSON, err := json.Marshal(req.ShippingAddress)
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid shipping address"))
		return
	}

	lines := make([]domain.CheckoutLineInput, 0, len(req.Lines))
	for _, l := range req.Lines {
		lines = append(lines, domain.CheckoutLineInput{
			SKUID:       l.SKUID,
			SellerID:    l.SellerID,
			Quantity:    l.Quantity,
			UnitPrice:   l.UnitPrice,
			ProductName: l.ProductName,
			SKUCode:     l.SKUCode,
		})
	}

	result, err := h.svc.CreateCheckout(r.Context(), tenantID, domain.CheckoutInput{
		BuyerAuth0ID:    req.BuyerAuth0ID,
		Lines:           lines,
		ShippingAddress: shippingJSON,
		Currency:        req.Currency,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}

	orderIDs := make([]string, 0, len(result.Orders))
	for i := range result.Orders {
		orderIDs = append(orderIDs, result.Orders[i].Order.ID.String())
	}

	httputil.JSON(w, http.StatusCreated, checkoutResponse{
		OrderIDs:              orderIDs,
		StripeClientSecret:    result.StripeClientSecret,
		StripePaymentIntentID: result.StripePaymentIntentID,
		TotalAmount:           result.TotalAmount,
		Currency:              result.Currency,
	})
}
