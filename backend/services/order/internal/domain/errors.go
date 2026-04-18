package domain

import "errors"

// Domain-level sentinel errors for the order service.
// These are transport-agnostic; the handler layer maps them to HTTP status codes.
var (
	// ErrOrderNotFound is returned when an order does not exist or does not
	// belong to the requested tenant/seller/buyer.
	ErrOrderNotFound = errors.New("order not found")

	// ErrEmptyOrder is returned when an order or checkout has no line items.
	ErrEmptyOrder = errors.New("order must have at least one line item")

	// ErrBuyerRequired is returned when buyer_auth0_id is missing.
	ErrBuyerRequired = errors.New("buyer_auth0_id is required")

	// ErrInvalidQuantity is returned when a line item quantity is not positive.
	ErrInvalidQuantity = errors.New("quantity must be positive")

	// ErrInvalidOrderStatus is returned when an unsupported status value is used.
	ErrInvalidOrderStatus = errors.New("invalid order status")

	// ErrOrderNotPending is returned when SetPaid targets an order that is not
	// in `pending` status. This is intentionally idempotent: a redelivered
	// payment_intent.succeeded for an already-paid or cancelled order must not
	// trigger a second Stripe Transfer.
	ErrOrderNotPending = errors.New("order is not in pending status")

	// ─── Discount / coupon sentinels ──────────────────────────────
	// These mirror coupon-svc's error codes so a 400 from Reserve can
	// be surfaced to the buyer without swallowing the reason.

	ErrCouponNotFound         = errors.New("coupon not found")
	ErrCouponExpired          = errors.New("coupon has expired")
	ErrCouponRevoked          = errors.New("coupon is no longer available")
	ErrCouponNotStarted       = errors.New("coupon is not yet valid")
	ErrCouponUsageLimitHit    = errors.New("coupon usage limit reached")
	ErrCouponPerUserLimit     = errors.New("coupon already used by this buyer")
	ErrCouponMinOrderNotMet   = errors.New("order subtotal is below the coupon minimum")
	ErrCouponCurrencyMismatch = errors.New("coupon currency does not match order currency")

	// ErrInsufficientPoints is returned when the buyer asked to redeem
	// more points than their account has spendable. Mirrors loyalty-svc.
	ErrInsufficientPoints = errors.New("insufficient point balance")

	// ErrFeatureDisabled is returned when the client supplied coupon_code
	// or points_to_redeem but the feature flag is off in this environment.
	ErrFeatureDisabled = errors.New("coupon / loyalty feature is disabled")
)
