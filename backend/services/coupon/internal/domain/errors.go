package domain

import "errors"

// Domain sentinel errors — transport-agnostic. Adapters map these to
// HTTP statuses or gRPC codes.
var (
	ErrCouponNotFound      = errors.New("coupon not found")
	ErrCouponExpired       = errors.New("coupon has expired")
	ErrCouponRevoked       = errors.New("coupon is revoked")
	ErrCouponNotStarted    = errors.New("coupon is not yet valid")
	ErrCouponUsageLimitHit = errors.New("coupon usage limit reached")
	ErrCouponPerUserLimit  = errors.New("per-user usage limit reached for this coupon")
	ErrMinOrderNotMet      = errors.New("order amount is below the coupon minimum")
	ErrCurrencyMismatch    = errors.New("coupon currency does not match order currency")
	ErrInvalidCouponInput  = errors.New("invalid coupon input")

	ErrReservationNotFound         = errors.New("coupon reservation not found")
	ErrReservationAlreadyTerminal  = errors.New("coupon reservation no longer pending")
)
