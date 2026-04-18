package port

import "errors"

// Adapter-layer sentinels. App translates these to domain errors
// before crossing into handler code.
var (
	// ErrUsageLimitExceeded means IncrementUsageIfBelowLimit found
	// usage_count >= usage_limit_total. App returns
	// domain.ErrCouponUsageLimitHit.
	ErrUsageLimitExceeded = errors.New("coupon usage limit exceeded")

	// ErrDuplicateRedemption: (coupon_id, order_id) UNIQUE violation on
	// Insert. App returns the existing redemption row and reports
	// already_committed=true to the caller.
	ErrDuplicateRedemption = errors.New("duplicate redemption")
)
