package domain

import "errors"

// Domain-level sentinel errors for the recommend service.
// These are transport-agnostic; the handler layer maps them to HTTP status codes.
var (
	// ErrMissingTenantID is returned when the request carries no tenant context.
	ErrMissingTenantID = errors.New("tenant_id is required")

	// ErrMissingUserID is returned when a user event has no user_id.
	ErrMissingUserID = errors.New("user_id is required")

	// ErrMissingProductID is returned when a user event or type-specific request
	// requires a product_id but none was supplied.
	ErrMissingProductID = errors.New("product_id is required")

	// ErrInvalidRecommendationType is returned when the requested recommendation
	// type is not one of the recognised values.
	ErrInvalidRecommendationType = errors.New("invalid recommendation type")

	// ErrInvalidEventType is returned when the user event type is not recognised.
	ErrInvalidEventType = errors.New("invalid event type")
)
