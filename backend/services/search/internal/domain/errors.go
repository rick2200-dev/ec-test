package domain

import "errors"

// Domain-level sentinel errors for the search service.
// These are transport-agnostic; the handler layer maps them to HTTP status codes.
var (
	// ErrMissingTenantID is returned when the request carries no tenant context.
	ErrMissingTenantID = errors.New("tenant_id is required")
)
