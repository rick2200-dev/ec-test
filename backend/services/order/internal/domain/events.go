package domain

// Event type name constants for order lifecycle events published to the
// "order-events" topic. These strings are part of the public event
// contract — renaming is a breaking change for all subscribers.
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

// OrderCreatedEvent is published when a new order is persisted.
type OrderCreatedEvent struct {
	OrderID               string `json:"order_id"`
	SellerID              string `json:"seller_id"`
	BuyerAuth0ID          string `json:"buyer_auth0_id"`
	TotalAmount           int64  `json:"total_amount"`
	Currency              string `json:"currency"`
	StripePaymentIntentID string `json:"stripe_payment_intent_id,omitempty"`
}

// OrderPaidEvent is published when a Stripe payment_intent.succeeded
// webhook is processed and the order is marked paid.
type OrderPaidEvent struct {
	OrderID               string `json:"order_id"`
	SellerID              string `json:"seller_id"`
	BuyerAuth0ID          string `json:"buyer_auth0_id"`
	TotalAmount           int64  `json:"total_amount"`
	StripePaymentIntentID string `json:"stripe_payment_intent_id"`
}

// OrderShippedEvent is published when a seller marks an order as shipped.
type OrderShippedEvent struct {
	OrderID string `json:"order_id"`
}

// PayoutFailedEvent is published when a Stripe Transfer fails during
// payment success handling.
type PayoutFailedEvent struct {
	PayoutID string `json:"payout_id"`
	OrderID  string `json:"order_id"`
	SellerID string `json:"seller_id"`
	Error    string `json:"error"`
}

// PayoutCompletedEvent is published when a Stripe Transfer is created
// successfully and the payout row is marked completed.
type PayoutCompletedEvent struct {
	PayoutID         string `json:"payout_id"`
	OrderID          string `json:"order_id"`
	SellerID         string `json:"seller_id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	StripeTransferID string `json:"stripe_transfer_id"`
}
