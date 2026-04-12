package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/domain"
	"github.com/Riku-KANO/ec-test/services/inquiry/internal/port"
)

// SellerHandler exposes the seller-facing inquiry REST API.
type SellerHandler struct {
	svc port.InquiryUseCase
}

// NewSellerHandler constructs a SellerHandler.
func NewSellerHandler(svc port.InquiryUseCase) *SellerHandler {
	return &SellerHandler{svc: svc}
}

// Routes returns the chi router for /seller/inquiries.
func (h *SellerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/messages", h.PostMessage)
	r.Post("/{id}/read", h.MarkRead)
	r.Post("/{id}/close", h.Close)
	return r
}

// List handles GET /seller/inquiries.
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
	status := r.URL.Query().Get("status")

	items, total, err := h.svc.ListForSeller(r.Context(), tc.TenantID, *tc.SellerID, status, limit, offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, listResponse{Items: items, Total: total, Limit: limit, Offset: offset})
}

// Get handles GET /seller/inquiries/{id}.
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
		httputil.Error(w, apperrors.BadRequest("invalid inquiry id"))
		return
	}
	result, err := h.svc.GetInquiry(r.Context(), tc.TenantID, id, tc.UserID, tc.SellerID)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}

// PostMessage handles POST /seller/inquiries/{id}/messages.
func (h *SellerHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
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
		httputil.Error(w, apperrors.BadRequest("invalid inquiry id"))
		return
	}
	var req postMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	msg, err := h.svc.PostMessage(r.Context(), tc.TenantID, tc.UserID, tc.SellerID, domain.PostMessageInput{
		InquiryID:  id,
		SenderType: domain.SenderTypeSeller,
		Body:       req.Body,
	})
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusCreated, msg)
}

// MarkRead handles POST /seller/inquiries/{id}/read.
func (h *SellerHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
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
		httputil.Error(w, apperrors.BadRequest("invalid inquiry id"))
		return
	}
	if err := h.svc.MarkRead(r.Context(), tc.TenantID, id, domain.SenderTypeSeller, tc.UserID, tc.SellerID); err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Close handles POST /seller/inquiries/{id}/close.
func (h *SellerHandler) Close(w http.ResponseWriter, r *http.Request) {
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
		httputil.Error(w, apperrors.BadRequest("invalid inquiry id"))
		return
	}
	if err := h.svc.CloseInquiry(r.Context(), tc.TenantID, id, tc.SellerID); err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
