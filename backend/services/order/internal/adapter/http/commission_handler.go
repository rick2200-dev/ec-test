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

// CommissionHandler handles HTTP requests for commission rule operations.
type CommissionHandler struct {
	svc port.OrderUseCase
}

// NewCommissionHandler creates a new CommissionHandler.
func NewCommissionHandler(svc port.OrderUseCase) *CommissionHandler {
	return &CommissionHandler{svc: svc}
}

// Routes returns the chi router for commission endpoints.
func (h *CommissionHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	return r
}

// List handles GET /commissions.
func (h *CommissionHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	p := pagination.FromRequest(r)

	rules, total, err := h.svc.ListCommissionRules(r.Context(), tenantID, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	resp := pagination.Response[domain.CommissionRule]{
		Items:  rules,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// Create handles POST /commissions.
func (h *CommissionHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	var rule domain.CommissionRule
	if err := httputil.Decode(r, &rule); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	if err := h.svc.CreateCommissionRule(r.Context(), tenantID, &rule); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	httputil.JSON(w, http.StatusCreated, rule)
}
