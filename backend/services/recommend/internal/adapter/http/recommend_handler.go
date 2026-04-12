package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/port"
)

// RecommendHandler handles HTTP requests for recommendations.
type RecommendHandler struct {
	svc port.RecommendUseCase
}

// NewRecommendHandler creates a new RecommendHandler.
func NewRecommendHandler(svc port.RecommendUseCase) *RecommendHandler {
	return &RecommendHandler{svc: svc}
}

// Routes returns the chi router for recommendation endpoints.
func (h *RecommendHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetRecommendations)
	r.Post("/events", h.RecordEvent)
	return r
}

// GetRecommendations handles GET /recommendations?type=popular&limit=10&product_id=...
func (h *RecommendHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	recType := domain.RecommendationType(r.URL.Query().Get("type"))
	if recType == "" {
		recType = domain.Popular
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	req := domain.RecommendRequest{
		TenantID: tc.TenantID,
		UserID:   tc.UserID,
		Type:     recType,
		Limit:    limit,
	}

	if pid := r.URL.Query().Get("product_id"); pid != "" {
		parsed, err := uuid.Parse(pid)
		if err != nil {
			httputil.Error(w, apperrors.BadRequest("invalid product_id"))
			return
		}
		req.ProductID = &parsed
	}

	resp, err := h.svc.GetRecommendations(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

// recordEventRequest is the JSON body for recording a user event.
type recordEventRequest struct {
	EventType string `json:"event_type"`
	ProductID string `json:"product_id"`
}

// RecordEvent handles POST /recommendations/events
func (h *RecommendHandler) RecordEvent(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}

	var body recordEventRequest
	if err := httputil.Decode(r, &body); err != nil {
		httputil.Error(w, err)
		return
	}

	productID, err := uuid.Parse(body.ProductID)
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid product_id"))
		return
	}

	event := domain.UserEvent{
		TenantID:  tc.TenantID,
		UserID:    tc.UserID,
		EventType: domain.UserEventType(body.EventType),
		ProductID: productID,
	}

	if err := h.svc.RecordUserEvent(r.Context(), event); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}
