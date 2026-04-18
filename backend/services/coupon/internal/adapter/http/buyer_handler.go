package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// BuyerHandler exposes the buyer-facing coupon endpoints. The gateway
// proxies /api/v1/buyer/coupons/* here; the authenticated
// buyer_auth0_id arrives as the X-User-ID header.
type BuyerHandler struct {
	svc port.CouponUseCase
}

func NewBuyerHandler(svc port.CouponUseCase) *BuyerHandler {
	return &BuyerHandler{svc: svc}
}

func (h *BuyerHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/preview", h.Preview)
	r.Get("/redemptions", h.ListRedemptions)
	return r
}

type sellerSubtotalReq struct {
	SellerID string `json:"seller_id"`
	Subtotal int64  `json:"subtotal"`
}

type previewRequest struct {
	Code            string              `json:"code"`
	Currency        string              `json:"currency"`
	SellerSubtotals []sellerSubtotalReq `json:"seller_subtotals"`
}

func (h *BuyerHandler) Preview(w http.ResponseWriter, r *http.Request) {
	buyer := r.Header.Get("X-User-ID")
	if buyer == "" {
		httputil.Error(w, apperrors.Unauthorized("missing X-User-ID"))
		return
	}
	var req previewRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	seller := make([]domain.SellerSubtotal, 0, len(req.SellerSubtotals))
	for _, s := range req.SellerSubtotals {
		id, err := uuid.Parse(s.SellerID)
		if err != nil {
			httputil.Error(w, apperrors.BadRequest("seller_id is not a valid UUID"))
			return
		}
		seller = append(seller, domain.SellerSubtotal{SellerID: id, Subtotal: s.Subtotal})
	}
	result, err := h.svc.PreviewCoupon(r.Context(), req.Code, buyer, seller, req.Currency)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if result == nil {
		// Unknown code — surface as 404 with a stable error code so the
		// UI can render the "not a valid coupon" message directly.
		httputil.Error(w, apperrors.NotFound("coupon not found").WithCode("COUPON_NOT_FOUND"))
		return
	}
	httputil.JSON(w, http.StatusOK, result)
}

type redemptionListResponse struct {
	Redemptions []domain.CouponRedemption `json:"redemptions"`
	Total       int                       `json:"total"`
}

func (h *BuyerHandler) ListRedemptions(w http.ResponseWriter, r *http.Request) {
	buyer := r.Header.Get("X-User-ID")
	if buyer == "" {
		httputil.Error(w, apperrors.Unauthorized("missing X-User-ID"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rows, total, err := h.svc.ListMyRedemptions(r.Context(), buyer, limit, offset)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if rows == nil {
		rows = []domain.CouponRedemption{}
	}
	httputil.JSON(w, http.StatusOK, redemptionListResponse{Redemptions: rows, Total: total})
}
