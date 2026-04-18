package handler

import (
	"errors"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// mapError converts domain sentinel errors to HTTP-aware AppErrors.
// Infrastructure errors already wrapped as AppError pass through unchanged.
// Any unrecognised error becomes a generic 500.
func mapError(err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrEmptyOrder),
		errors.Is(err, domain.ErrBuyerRequired),
		errors.Is(err, domain.ErrInvalidQuantity),
		errors.Is(err, domain.ErrInvalidOrderStatus):
		return apperrors.BadRequest(err.Error())
	case errors.Is(err, domain.ErrOrderNotPending):
		return apperrors.Conflict(err.Error())
	// ─── Discount / coupon mapping ───────────────────────────────
	// Stable Code values match coupon-svc so the frontend's existing
	// switch (e.g. "COUPON_EXPIRED" → "This code has expired") works
	// for buyers regardless of which service produced the error.
	case errors.Is(err, domain.ErrCouponNotFound):
		return apperrors.NotFound(err.Error()).WithCode("COUPON_NOT_FOUND")
	case errors.Is(err, domain.ErrCouponExpired):
		return apperrors.BadRequest(err.Error()).WithCode("COUPON_EXPIRED")
	case errors.Is(err, domain.ErrCouponRevoked):
		return apperrors.BadRequest(err.Error()).WithCode("COUPON_REVOKED")
	case errors.Is(err, domain.ErrCouponNotStarted):
		return apperrors.BadRequest(err.Error()).WithCode("COUPON_NOT_STARTED")
	case errors.Is(err, domain.ErrCouponUsageLimitHit):
		return apperrors.Conflict(err.Error()).WithCode("COUPON_USAGE_LIMIT")
	case errors.Is(err, domain.ErrCouponPerUserLimit):
		return apperrors.Conflict(err.Error()).WithCode("COUPON_PER_USER_LIMIT")
	case errors.Is(err, domain.ErrCouponMinOrderNotMet):
		return apperrors.BadRequest(err.Error()).WithCode("COUPON_MIN_ORDER")
	case errors.Is(err, domain.ErrCouponCurrencyMismatch):
		return apperrors.BadRequest(err.Error()).WithCode("COUPON_CURRENCY")
	case errors.Is(err, domain.ErrInsufficientPoints):
		return apperrors.BadRequest(err.Error()).WithCode("INSUFFICIENT_POINTS")
	case errors.Is(err, domain.ErrFeatureDisabled):
		return apperrors.BadRequest(err.Error()).WithCode("FEATURE_DISABLED")
	default:
		return apperrors.Internal("internal error", err)
	}
}
