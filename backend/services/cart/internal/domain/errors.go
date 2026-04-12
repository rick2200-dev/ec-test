package domain

import "errors"

// Domain-level sentinel errors for the cart service.
// These are transport-agnostic; the handler layer maps them to HTTP status codes.
var (
	// ErrEmptyCart is returned when an operation requires a non-empty cart.
	ErrEmptyCart = errors.New("cart is empty")

	// ErrSKUNotInCart is returned when the requested SKU is not in the cart.
	ErrSKUNotInCart = errors.New("sku not in cart")

	// ErrInvalidQuantity is returned when a quantity is not valid (≤ 0, or < 0
	// for operations where zero is disallowed).
	ErrInvalidQuantity = errors.New("quantity must be positive")

	// ErrNonNegativeQuantity is returned when a quantity is negative.
	ErrNonNegativeQuantity = errors.New("quantity must be non-negative")
)
