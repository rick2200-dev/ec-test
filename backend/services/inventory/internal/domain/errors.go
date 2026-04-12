package domain

import "errors"

// Domain-level sentinel errors for the inventory service.
// These are transport-agnostic; the handler layer maps them to HTTP status codes.
var (
	// ErrInventoryNotFound is returned when no inventory record exists for
	// the requested SKU.
	ErrInventoryNotFound = errors.New("inventory not found")

	// ErrInvalidQuantity is returned when a quantity is not positive.
	ErrInvalidQuantity = errors.New("quantity must be positive")
)
