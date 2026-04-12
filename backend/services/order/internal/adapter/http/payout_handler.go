package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/port"
)

// PayoutHandler handles HTTP requests for payout operations.
type PayoutHandler struct {
	svc port.OrderUseCase
}

// NewPayoutHandler creates a new PayoutHandler.
func NewPayoutHandler(svc port.OrderUseCase) *PayoutHandler {
	return &PayoutHandler{svc: svc}
}

// Routes returns the chi router for payout endpoints.
func (h *PayoutHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	return r
}

// List handles GET /payouts - lists payouts for the authenticated seller.
func (h *PayoutHandler) List(w http.ResponseWriter, r *http.Request) {
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

	payouts, total, err := h.svc.ListPayouts(r.Context(), tc.TenantID, *tc.SellerID, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	resp := pagination.Response[domain.Payout]{
		Items:  payouts,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}
