package domain

// Event type name constants for order lifecycle events published to the
// "order-events" topic. These strings are part of the public event contract —
// renaming is a breaking change for all subscribers. The payload schemas are
// defined in services/order/api/proto/order/v1/events.proto.
const (
	EventTypeOrderCreated = "order.created"
	EventTypeOrderPaid    = "order.paid"
	EventTypeOrderShipped = "order.shipped"
)

// Event type name constants for payout lifecycle events published to the
// "payout-events" topic.
const (
	EventTypePayoutFailed    = "payout.failed"
	EventTypePayoutCompleted = "payout.completed"
)
