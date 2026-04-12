package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/pagination"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/port"
)

// SellerHandler handles HTTP requests for seller operations.
type SellerHandler struct {
	svc       port.AuthUseCase
	team      *SellerTeamHandler
	apiTokens *APITokenHandler
}

// NewSellerHandler creates a new SellerHandler. The team and API token
// handlers are mounted as nested subroutes under /{sellerID} so the
// /sellers prefix can own the entire seller subtree. apiTokens may be nil
// in tests that don't exercise the token surface.
func NewSellerHandler(svc port.AuthUseCase, team *SellerTeamHandler, apiTokens *APITokenHandler) *SellerHandler {
	return &SellerHandler{svc: svc, team: team, apiTokens: apiTokens}
}

// Routes returns the chi router for seller endpoints.
func (h *SellerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}/approve", h.Approve)
	// Seller team management lives under /{sellerID}/team. The team handler's
	// Mount registers GET/POST/PUT/DELETE on the subrouter so the {sellerID}
	// URL parameter is available to its handlers via chi.URLParam.
	r.Route("/{sellerID}/team", h.team.Mount)
	// Seller API access tokens live under /{sellerID}/api-tokens, parallel
	// to /team. The handler is optional so tests that only care about the
	// team surface can skip wiring it.
	if h.apiTokens != nil {
		r.Route("/{sellerID}/api-tokens", h.apiTokens.Mount)
	}
	return r
}

// createSellerRequest is the request body for creating a seller.
type createSellerRequest struct {
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	Auth0OrgID        string `json:"auth0_org_id"`
	StripeAccountID   string `json:"stripe_account_id"`
	CommissionRateBPS int    `json:"commission_rate_bps"`
}

// Create handles POST /sellers (tenant-scoped).
func (h *SellerHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	var req createSellerRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	s := &domain.Seller{
		Name:              req.Name,
		Slug:              req.Slug,
		Auth0OrgID:        req.Auth0OrgID,
		StripeAccountID:   req.StripeAccountID,
		CommissionRateBPS: req.CommissionRateBPS,
	}

	if err := h.svc.CreateSeller(r.Context(), tenantID, s); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, s)
}

// GetByID handles GET /sellers/{id}.
func (h *SellerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seller id"})
		return
	}

	s, err := h.svc.GetSeller(r.Context(), tenantID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, s)
}

// List handles GET /sellers.
func (h *SellerHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	p := pagination.FromRequest(r)

	sellers, total, err := h.svc.ListSellers(r.Context(), tenantID, p.Limit, p.Offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	resp := pagination.Response[domain.Seller]{
		Items:  sellers,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// Approve handles PUT /sellers/{id}/approve.
func (h *SellerHandler) Approve(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seller id"})
		return
	}

	if err := h.svc.ApproveSeller(r.Context(), tenantID, id); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "approved"})
}
