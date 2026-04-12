package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
	"github.com/Riku-KANO/ec-test/services/review/internal/port"
)

// SellerHandler exposes the seller-facing review REST API.
type SellerHandler struct {
	svc port.ReviewUseCase
}

// NewSellerHandler constructs a SellerHandler.
func NewSellerHandler(svc port.ReviewUseCase) *SellerHandler {
	return &SellerHandler{svc: svc}
}

// Routes returns the chi router for /seller/reviews.
func (h *SellerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/reply", h.CreateReply)
	r.Put("/{id}/reply", h.UpdateReply)
	r.Delete("/{id}/reply", h.DeleteReply)
	return r
}

// List handles GET /seller/reviews.
func (h *SellerHandler) List(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.Forbidden("seller context required"))
		return
	}
	limit, offset := parsePagination(r)

	items, total, err := h.svc.ListForSeller(r.Context(), tc.TenantID, *tc.SellerID, limit, offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, reviewListResponse{Items: items, Total: total, Limit: limit, Offset: offset})
}

// Get handles GET /seller/reviews/{id}.
func (h *SellerHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.Forbidden("seller context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid review id"))
		return
	}

	review, err := h.svc.GetReview(r.Context(), tc.TenantID, id)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if review.SellerID != *tc.SellerID {
		httputil.Error(w, apperrors.Forbidden("review does not belong to your seller account"))
		return
	}
	httputil.JSON(w, http.StatusOK, review)
}

// replyRequest is the POST/PUT /seller/reviews/{id}/reply payload.
type replyRequest struct {
	Body string `json:"body"`
}

// CreateReply handles POST /seller/reviews/{id}/reply.
func (h *SellerHandler) CreateReply(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.Forbidden("seller context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid review id"))
		return
	}
	var req replyRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	reply, err := h.svc.CreateReply(r.Context(), tc.TenantID, id, tc.UserID, *tc.SellerID, domain.CreateReplyInput{
		Body: req.Body,
	})
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusCreated, reply)
}

// UpdateReply handles PUT /seller/reviews/{id}/reply.
func (h *SellerHandler) UpdateReply(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.Forbidden("seller context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid review id"))
		return
	}
	var req replyRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, mapError(err))
		return
	}

	reply, err := h.svc.UpdateReply(r.Context(), tc.TenantID, id, tc.UserID, *tc.SellerID, domain.UpdateReplyInput{
		Body: req.Body,
	})
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, reply)
}

// DeleteReply handles DELETE /seller/reviews/{id}/reply.
func (h *SellerHandler) DeleteReply(w http.ResponseWriter, r *http.Request) {
	tc, err := tenant.FromContext(r.Context())
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("tenant context required"))
		return
	}
	if tc.SellerID == nil {
		httputil.Error(w, apperrors.Forbidden("seller context required"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("invalid review id"))
		return
	}

	if err := h.svc.DeleteReply(r.Context(), tc.TenantID, id, *tc.SellerID); err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
