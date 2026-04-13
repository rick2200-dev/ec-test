package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/domain"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/port"
)

// SubscriptionHandler handles HTTP requests for seller plan and subscription operations.
type SubscriptionHandler struct {
	svc port.SubscriptionUseCase
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(svc port.SubscriptionUseCase) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

// PlanRoutes returns the chi router for plan management endpoints.
func (h *SubscriptionHandler) PlanRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListPlans)
	r.Post("/", h.CreatePlan)
	r.Get("/{id}", h.GetPlan)
	r.Put("/{id}", h.UpdatePlan)
	return r
}

// SubscriptionRoutes returns the chi router for seller subscription endpoints.
func (h *SubscriptionHandler) SubscriptionRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/sellers/{sellerID}", h.GetSellerSubscription)
	r.Post("/sellers/{sellerID}", h.SubscribeSeller)
	return r
}

func (h *SubscriptionHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	plans, err := h.svc.ListPlans(r.Context(), tenantID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, plans)
}

type createPlanRequest struct {
	Name          string              `json:"name"`
	Slug          string              `json:"slug"`
	Tier          int                 `json:"tier"`
	PriceAmount   int64               `json:"price_amount"`
	PriceCurrency string              `json:"price_currency"`
	Features      domain.PlanFeatures `json:"features"`
	StripePriceID string              `json:"stripe_price_id"`
}

func (h *SubscriptionHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	var req createPlanRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	plan := &domain.SubscriptionPlan{
		Name:          req.Name,
		Slug:          req.Slug,
		Tier:          req.Tier,
		PriceAmount:   req.PriceAmount,
		PriceCurrency: req.PriceCurrency,
		Features:      req.Features,
		StripePriceID: req.StripePriceID,
	}

	if err := h.svc.CreatePlan(r.Context(), tenantID, plan); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, plan)
}

func (h *SubscriptionHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
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

	plan, err := h.svc.GetPlan(r.Context(), tenantID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, plan)
}

type updatePlanRequest struct {
	Name          string              `json:"name"`
	Slug          string              `json:"slug"`
	Tier          int                 `json:"tier"`
	PriceAmount   int64               `json:"price_amount"`
	PriceCurrency string              `json:"price_currency"`
	Features      domain.PlanFeatures `json:"features"`
	StripePriceID string              `json:"stripe_price_id"`
	Status        string              `json:"status"`
}

func (h *SubscriptionHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
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

	var req updatePlanRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	plan := &domain.SubscriptionPlan{
		ID:            id,
		Name:          req.Name,
		Slug:          req.Slug,
		Tier:          req.Tier,
		PriceAmount:   req.PriceAmount,
		PriceCurrency: req.PriceCurrency,
		Features:      req.Features,
		StripePriceID: req.StripePriceID,
		Status:        req.Status,
	}

	if err := h.svc.UpdatePlan(r.Context(), tenantID, plan); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, plan)
}

func (h *SubscriptionHandler) GetSellerSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	sellerID, err := uuid.Parse(chi.URLParam(r, "sellerID"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seller id"})
		return
	}

	sub, err := h.svc.GetSellerSubscription(r.Context(), tenantID, sellerID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, sub)
}

type subscribeRequest struct {
	PlanID string `json:"plan_id"`
}

func (h *SubscriptionHandler) SubscribeSeller(w http.ResponseWriter, r *http.Request) {
	tenantID, err := tenant.TenantID(r.Context())
	if err != nil {
		httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant context required"})
		return
	}

	sellerID, err := uuid.Parse(chi.URLParam(r, "sellerID"))
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid seller id"})
		return
	}

	var req subscribeRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid plan id"})
		return
	}

	sub, err := h.svc.SubscribeSeller(r.Context(), tenantID, sellerID, planID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, sub)
}
