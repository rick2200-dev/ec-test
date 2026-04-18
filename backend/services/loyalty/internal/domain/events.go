package domain

// Loyalty event type constants — the Type field on the event envelope
// so subscribers can filter without unmarshalling the payload.
const (
	EventTypePointsEarned    = "points.earned"
	EventTypePointsRedeemed  = "points.redeemed"
	EventTypePointsRefunded  = "points.refunded"
	EventTypePointsAdjusted  = "points.adjusted"
)

// TopicLoyaltyEvents is the GCP Pub/Sub topic name for all loyalty
// events. Consumers (notification) subscribe here.
const TopicLoyaltyEvents = "loyalty-events"
