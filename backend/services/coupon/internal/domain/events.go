package domain

// Coupon event type constants — the Type field on the event envelope
// so subscribers filter without unmarshalling the payload.
const (
	EventTypeCouponIssued   = "coupon.issued"
	EventTypeCouponRedeemed = "coupon.redeemed"
	EventTypeCouponRevoked  = "coupon.revoked"
)

// TopicCouponEvents is the GCP Pub/Sub topic for all coupon events.
const TopicCouponEvents = "coupon-events"
