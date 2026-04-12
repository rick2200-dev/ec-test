package domain

import "errors"

// Domain-level sentinel errors for the catalog service.
// These are transport-agnostic; the handler layer maps them to HTTP status codes.
var (
	// ErrCategoryNotFound is returned when a category does not exist.
	ErrCategoryNotFound = errors.New("category not found")

	// ErrCategorySlugConflict is returned when a category slug already exists.
	ErrCategorySlugConflict = errors.New("category slug already exists")

	// ErrProductNotFound is returned when a product does not exist.
	ErrProductNotFound = errors.New("product not found")

	// ErrProductSlugConflict is returned when a product slug already exists.
	ErrProductSlugConflict = errors.New("product slug already exists")

	// ErrSKUNotFound is returned when a SKU does not exist.
	ErrSKUNotFound = errors.New("sku not found")

	// ErrSellerRequired is returned when seller_id is missing on a write operation.
	ErrSellerRequired = errors.New("seller_id is required")

	// ErrNotProductOwner is returned when the caller does not own the product.
	ErrNotProductOwner = errors.New("not authorized to update this product")
)
