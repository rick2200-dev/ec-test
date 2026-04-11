package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/service"
)

// BuyerHandler exposes the buyer-facing inquiry REST API.
type BuyerHandler struct {
	svc *service.InquiryService
}

// NewBuyerHandler constructs a BuyerHandler.
func NewBuyerHandler(svc *service.InquiryService) *BuyerHandler {
	return &BuyerHandler{svc: svc}
}

// Routes returns the chi router for /inquiries (buyer scope).
func (h *BuyerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/messages", h.PostMessage)
	r.Post("/{id}/read", h.MarkRead)
	return r
}

// List handles GET /inquiries.
func (h *BuyerHandler) List(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	limit, offset := parsePagination(r)

	items, total, err := h.svc.ListForBuyer(r.Context(), tc.TenantID, tc.UserID, limit, offset)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, listResponse{Items: items, Total: total, Limit: limit, Offset: offset})
}

// createInquiryRequest is the POST /inquiries payload.
type createInquiryRequest struct {
	SellerID    uuid.UUID `json:"seller_id"`
	SKUID       uuid.UUID `json:"sku_id"`
	Subject     string    `json:"subject"`
	InitialBody string    `json:"initial_body"`
}

// Create handles POST /inquiries.
func (h *BuyerHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	var req createInquiryRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	result, err := h.svc.CreateInquiry(r.Context(), tc.TenantID, tc.UserID, domain.CreateInquiryInput{
		SellerID:    req.SellerID,
		SKUID:       req.SKUID,
		Subject:     req.Subject,
		InitialBody: req.InitialBody,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, result)
}

// Get handles GET /inquiries/{id}.
func (h *BuyerHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid inquiry id"))
		return
	}
	result, err := h.svc.GetInquiry(r.Context(), tc.TenantID, id, tc.UserID, tc.SellerID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}

// postMessageRequest is the POST /inquiries/{id}/messages payload.
type postMessageRequest struct {
	Body string `json:"body"`
}

// PostMessage handles POST /inquiries/{id}/messages.
func (h *BuyerHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid inquiry id"))
		return
	}
	var req postMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	msg, err := h.svc.PostMessage(r.Context(), tc.TenantID, tc.UserID, tc.SellerID, domain.PostMessageInput{
		InquiryID:  id,
		SenderType: domain.SenderTypeBuyer,
		Body:       req.Body,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, msg)
}

// MarkRead handles POST /inquiries/{id}/read.
func (h *BuyerHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid inquiry id"))
		return
	}
	if err := h.svc.MarkRead(r.Context(), tc.TenantID, id, domain.SenderTypeBuyer, tc.UserID, tc.SellerID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// listResponse is the standard paginated list envelope.
type listResponse struct {
	Items  []domain.Inquiry `json:"items"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// parsePagination reads limit/offset from query params with sensible defaults.
func parsePagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
