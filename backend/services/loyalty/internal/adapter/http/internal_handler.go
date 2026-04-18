package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// InternalHandler exposes the in-cluster endpoints order-svc calls
// synchronously from its webhook path. Gateway never forwards here;
// the X-Internal-Token middleware gates all routes.
type InternalHandler struct {
	svc port.LoyaltyUseCase
}

func NewInternalHandler(svc port.LoyaltyUseCase) *InternalHandler {
	return &InternalHandler{svc: svc}
}

func (h *InternalHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/earn", h.AwardEarn)
	r.Post("/reservations", h.Reserve)
	r.Post("/reservations/{id}/commit", h.Commit)
	r.Post("/reservations/{id}/release", h.Release)
	return r
}

type awardEarnRequest struct {
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	OrderID      string `json:"order_id"`
	PaidSubtotal int64  `json:"paid_subtotal"`
	Currency     string `json:"currency"`
}

type awardEarnResponse struct {
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	BalanceAfter  int64  `json:"balance_after"`
}

// AwardEarn is POST /internal/earn. order-svc calls it from
// HandlePaymentSuccess after a Stripe payment_intent.succeeded.
func (h *InternalHandler) AwardEarn(w http.ResponseWriter, r *http.Request) {
	var req awardEarnRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	if req.BuyerAuth0ID == "" || req.OrderID == "" {
		httputil.Error(w, apperrors.BadRequest("buyer_auth0_id and order_id are required"))
		return
	}
	tx, err := h.svc.AwardEarn(r.Context(), req.BuyerAuth0ID, req.OrderID, req.PaidSubtotal, req.Currency)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	if tx == nil {
		// Zero-earn (subtotal too small or earn rate 0). Return 204.
		httputil.JSON(w, http.StatusNoContent, nil)
		return
	}
	// The service layer reports replay-safe idempotent behavior by
	// returning the existing row on a duplicate; the handler stays
	// simple here and leaves replay introspection for future work.
	httputil.JSON(w, http.StatusOK, awardEarnResponse{
		TransactionID: tx.ID.String(),
		Amount:        tx.Amount,
		BalanceAfter:  tx.BalanceAfter,
	})
}

type reserveRequest struct {
	BuyerAuth0ID string `json:"buyer_auth0_id"`
	Amount       int64  `json:"amount"`
}

type reserveResponse struct {
	ReservationID string    `json:"reservation_id"`
	Amount        int64     `json:"amount"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// Reserve POST /internal/reservations — order-svc calls this during
// CreateCheckout when the buyer opts to spend points.
func (h *InternalHandler) Reserve(w http.ResponseWriter, r *http.Request) {
	var req reserveRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}
	if req.BuyerAuth0ID == "" || req.Amount <= 0 {
		httputil.Error(w, apperrors.BadRequest("buyer_auth0_id and positive amount are required"))
		return
	}
	reservation, err := h.svc.ReservePoints(r.Context(), req.BuyerAuth0ID, req.Amount)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, reserveResponse{
		ReservationID: reservation.ID.String(),
		Amount:        reservation.Amount,
		ExpiresAt:     reservation.ExpiresAt,
	})
}

type commitRequest struct {
	OrderID               string `json:"order_id"`
	StripePaymentIntentID string `json:"stripe_payment_intent_id"`
	PaidSubtotal          int64  `json:"paid_subtotal"`
	Currency              string `json:"currency"`
}

type commitResponse struct {
	Redeemed         int64 `json:"redeemed"`
	Earned           int64 `json:"earned"`
	NewBalance       int64 `json:"new_balance"`
	AlreadyCommitted bool  `json:"already_committed"`
}

// Commit POST /internal/reservations/{id}/commit — order-svc calls
// this from HandlePaymentSuccess once Stripe confirms payment.
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
	if req.OrderID == "" {
		httputil.Error(w, apperrors.BadRequest("order_id is required"))
		return
	}
	result, err := h.svc.CommitPointsReservation(r.Context(), id, req.OrderID, req.StripePaymentIntentID, req.PaidSubtotal, req.Currency)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, commitResponse{
		Redeemed:         result.Redeemed,
		Earned:           result.Earned,
		NewBalance:       result.NewBalance,
		AlreadyCommitted: result.AlreadyCommitted,
	})
}

type releaseRequest struct {
	Reason string `json:"reason"`
}

type releaseResponse struct {
	Released bool `json:"released"`
}

// Release POST /internal/reservations/{id}/release — called on
// checkout abandonment or PaymentIntent failure.
func (h *InternalHandler) Release(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperrors.BadRequest("reservation id is not a valid UUID"))
		return
	}
	var req releaseRequest
	if err := httputil.Decode(r, &req); err != nil {
		// Body is optional — reason is informational.
		req = releaseRequest{}
	}
	released, err := h.svc.ReleasePointsReservation(r.Context(), id, req.Reason)
	if err != nil {
		httputil.Error(w, mapError(err))
		return
	}
	httputil.JSON(w, http.StatusOK, releaseResponse{Released: released})
}
