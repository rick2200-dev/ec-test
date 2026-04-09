package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/service"
)

// BuyerSubscriptionHandler handles HTTP requests for buyer plan and subscription operations.
type BuyerSubscriptionHandler struct {
	svc *service.AuthService
}

// NewBuyerSubscriptionHandler creates a new BuyerSubscriptionHandler.
func NewBuyerSubscriptionHandler(svc *service.AuthService) *BuyerSubscriptionHandler {
	return &BuyerSubscriptionHandler{svc: svc}
}

// BuyerPlanRoutes returns the chi router for buyer plan management endpoints.
func (h *BuyerSubscriptionHandler) BuyerPlanRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListBuyerPlans)
	r.Post("/", h.CreateBuyerPlan)
	r.Get("/{id}", h.GetBuyerPlan)
	r.Put("/{id}", h.UpdateBuyerPlan)
	return r
}

// BuyerSubscriptionRoutes returns the chi router for buyer subscription endpoints.
func (h *BuyerSubscriptionHandler) BuyerSubscriptionRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/buyers/{buyerAuth0ID}", h.GetBuyerSubscription)
	r.Post("/buyers/{buyerAuth0ID}", h.SubscribeBuyer)
	return r
}

// ListBuyerPlans handles GET /buyer-plans.
func (h *BuyerSubscriptionHandler) ListBuyerPlans(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	plans, err := h.svc.ListBuyerPlans(r.Context(), tenantID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, plans)
}

type createBuyerPlanRequest struct {
	Name          string                  `json:"name"`
	Slug          string                  `json:"slug"`
	PriceAmount   int64                   `json:"price_amount"`
	PriceCurrency string                  `json:"price_currency"`
	Features      domain.BuyerPlanFeatures `json:"features"`
	StripePriceID string                  `json:"stripe_price_id"`
}

// CreateBuyerPlan handles POST /buyer-plans.
func (h *BuyerSubscriptionHandler) CreateBuyerPlan(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	var req createBuyerPlanRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	plan := &domain.BuyerPlan{
		Name:          req.Name,
		Slug:          req.Slug,
		PriceAmount:   req.PriceAmount,
		PriceCurrency: req.PriceCurrency,
		Features:      req.Features,
		StripePriceID: req.StripePriceID,
	}

	if err := h.svc.CreateBuyerPlan(r.Context(), tenantID, plan); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, plan)
}

// GetBuyerPlan handles GET /buyer-plans/{id}.
func (h *BuyerSubscriptionHandler) GetBuyerPlan(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid plan id"})
		return
	}

	plan, err := h.svc.GetBuyerPlan(r.Context(), tenantID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, plan)
}

type updateBuyerPlanRequest struct {
	Name          string                  `json:"name"`
	Slug          string                  `json:"slug"`
	PriceAmount   int64                   `json:"price_amount"`
	PriceCurrency string                  `json:"price_currency"`
	Features      domain.BuyerPlanFeatures `json:"features"`
	StripePriceID string                  `json:"stripe_price_id"`
	Status        string                  `json:"status"`
}

// UpdateBuyerPlan handles PUT /buyer-plans/{id}.
func (h *BuyerSubscriptionHandler) UpdateBuyerPlan(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid plan id"})
		return
	}

	var req updateBuyerPlanRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	plan := &domain.BuyerPlan{
		ID:            id,
		Name:          req.Name,
		Slug:          req.Slug,
		PriceAmount:   req.PriceAmount,
		PriceCurrency: req.PriceCurrency,
		Features:      req.Features,
		StripePriceID: req.StripePriceID,
		Status:        req.Status,
	}

	if err := h.svc.UpdateBuyerPlan(r.Context(), tenantID, plan); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, plan)
}

// GetBuyerSubscription handles GET /buyer-subscriptions/buyers/{buyerAuth0ID}.
func (h *BuyerSubscriptionHandler) GetBuyerSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	buyerAuth0ID := chi.URLParam(r, "buyerAuth0ID")
	if buyerAuth0ID == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "buyer_auth0_id required"})
		return
	}

	sub, err := h.svc.GetBuyerSubscription(r.Context(), tenantID, buyerAuth0ID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, sub)
}

type subscribeBuyerRequest struct {
	PlanID string `json:"plan_id"`
}

// SubscribeBuyer handles POST /buyer-subscriptions/buyers/{buyerAuth0ID}.
func (h *BuyerSubscriptionHandler) SubscribeBuyer(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	buyerAuth0ID := chi.URLParam(r, "buyerAuth0ID")
	if buyerAuth0ID == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "buyer_auth0_id required"})
		return
	}

	var req subscribeBuyerRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid plan id"})
		return
	}

	sub, err := h.svc.SubscribeBuyer(r.Context(), tenantID, buyerAuth0ID, planID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, sub)
}
