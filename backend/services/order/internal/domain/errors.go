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
)
