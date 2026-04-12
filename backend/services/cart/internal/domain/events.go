package domain

// CartCheckedOutEvent is published on the "cart-events" topic when a buyer
// successfully completes checkout. Downstream consumers (recommend service)
// use this event to record user purchase behaviour.
type CartCheckedOutEvent struct {
	BuyerAuth0ID          string   `json:"buyer_auth0_id"`
	OrderIDs              []string `json:"order_ids"`
	StripePaymentIntentID string   `json:"stripe_payment_intent_id"`
	TotalAmount           int64    `json:"total_amount"`
	Currency              string   `json:"currency"`
}
