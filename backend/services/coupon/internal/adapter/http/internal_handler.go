package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// InternalHandler exposes the in-cluster Reserve/Commit/Release
// endpoints that order-svc calls synchronously. Gateway never forwards
// here.
type InternalHandler struct {
	svc port.CouponUseCase
}

func NewInternalHandler(svc port.CouponUseCase) *InternalHandler {
	return &InternalHandler{svc: svc}
}

func (h *InternalHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/reservations", h.Reserve)
	r.Post("/reservations/{id}/commit", h.Commit)
	r.Post("/reservations/{id}/release", h.Release)
	return r
}

type sellerSubtotalIn struct {
	SellerID string `json:"seller_id"`
	Subtotal int64  `json:"subtotal"`
}

type reserveRequest struct {
	Code            string             `json:"code"`
	BuyerAuth0ID    string             `json:"buyer_auth0_id"`
	Currency        string             `json:"currency"`
	SellerSubtotals []sellerSubtotalIn `json:"seller_subtotals"`
}

type reserveResponse struct {
	ReservationID      string    `json:"reservation_id"`
	CouponID           string    `json:"coupon_id"`
	IssuerType         string    `json:"issuer_type"`
	ApplicableSellerID string    `json:"applicable_seller_id,omitempty"`
	DiscountAmount     int64     `json:"discount_amount"`
	Currency           string    `json:"currency"`
	ExpiresAt          time.Time `json:"expires_at"`
}

func (h *InternalHandler) Reserve(w http.ResponseWriter, r *http.Request) {
	var req reserveRequest
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
	reservation, err := h.svc.ReserveCoupon(r.Context(), req.Code, req.BuyerAuth0ID, seller, req.Currency)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	issuerType := domain.IssuerTypePlatform
	applicableSellerID := ""
	if reservation.ApplicableSellerID != nil {
		issuerType = domain.IssuerTypeSeller
		applicableSellerID = reservation.ApplicableSellerID.String()
	}
	httputil.JSON(w, http.StatusOK, reserveResponse{
		ReservationID:      reservation.ID.String(),
		CouponID:           reservation.CouponID.String(),
		IssuerType:         string(issuerType),
		ApplicableSellerID: applicableSellerID,
		DiscountAmount:     reservation.DiscountAmount,
		Currency:           reservation.Currency,
		ExpiresAt:          reservation.ExpiresAt,
	})
}

type commitRequest struct {
	OrderID               string `json:"order_id"`
	StripePaymentIntentID string `json:"stripe_payment_intent_id"`
}

type commitResponse struct {
	RedemptionID     string `json:"redemption_id"`
	AlreadyCommitted bool   `json:"already_committed"`
}

func (h *InternalHandler) Commit(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("reservation id is not a valid UUID"))
		return
	}
	var req commitRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("order_id is not a valid UUID"))
		return
	}
	result, err := h.svc.CommitReservation(r.Context(), id, orderID, req.StripePaymentIntentID)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, commitResponse{
		RedemptionID:     result.RedemptionID.String(),
		AlreadyCommitted: result.AlreadyCommitted,
	})
}

type releaseRequest struct {
	Reason string `json:"reason"`
}

type releaseResponse struct {
	Released bool `json:"released"`
}

func (h *InternalHandler) Release(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("reservation id is not a valid UUID"))
		return
	}
	var req releaseRequest
	if err := httputil.Decode(r, &req); err != nil {
		// Release with no body is allowed — the reason is informational.
		req = releaseRequest{}
	}
	released, err := h.svc.ReleaseReservation(r.Context(), id, req.Reason)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, releaseResponse{Released: released})
}
