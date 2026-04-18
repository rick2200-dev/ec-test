package handler

import (
	"errors"
	"net/http"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
)

// mapError translates app/domain errors to AppError with HTTP
// statuses + stable Code values. Anything we don't recognize collapses
// to a generic 500 so internal details never leak.
func mapError(err error) *apperrors.AppError {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, domain.ErrCouponNotFound):
		return apperrors.NotFound("coupon not found").WithCode("COUPON_NOT_FOUND")
	case errors.Is(err, domain.ErrCouponExpired):
		return apperrors.BadRequest("coupon has expired").WithCode("COUPON_EXPIRED")
	case errors.Is(err, domain.ErrCouponRevoked):
		return apperrors.BadRequest("coupon is no longer available").WithCode("COUPON_REVOKED")
	case errors.Is(err, domain.ErrCouponNotStarted):
		return apperrors.BadRequest("coupon is not yet valid").WithCode("COUPON_NOT_STARTED")
	case errors.Is(err, domain.ErrCouponUsageLimitHit):
		return apperrors.Conflict("coupon usage limit reached").WithCode("COUPON_USAGE_LIMIT")
	case errors.Is(err, domain.ErrCouponPerUserLimit):
		return apperrors.Conflict("you have already used this coupon").WithCode("COUPON_PER_USER_LIMIT")
	case errors.Is(err, domain.ErrMinOrderNotMet):
		return apperrors.BadRequest("order amount is below the coupon minimum").WithCode("COUPON_MIN_ORDER")
	case errors.Is(err, domain.ErrCurrencyMismatch):
		return apperrors.BadRequest("coupon currency does not match order currency").WithCode("COUPON_CURRENCY")
	case errors.Is(err, domain.ErrInvalidCouponInput):
		return apperrors.BadRequest("invalid coupon application")
	case errors.Is(err, domain.ErrReservationNotFound):
		return apperrors.NotFound("coupon reservation not found")
	case errors.Is(err, domain.ErrReservationAlreadyTerminal):
		return apperrors.Conflict("coupon reservation is no longer pending").WithCode("RESERVATION_TERMINAL")
	}
	return &apperrors.AppError{Status: http.StatusInternalServerError, Message: "internal error", Err: err}
}
