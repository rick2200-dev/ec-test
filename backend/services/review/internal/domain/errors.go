package domain

import "errors"

// Domain-level sentinel errors for the review service.
// These are transport-agnostic; the handler layer maps them to HTTP status codes.
var (
	// ErrReviewNotFound is returned when the requested review does not exist.
	ErrReviewNotFound = errors.New("review not found")

	// ErrReplyNotFound is returned when the requested reply does not exist.
	ErrReplyNotFound = errors.New("reply not found")

	// ErrAlreadyReviewed is returned when a buyer tries to review a product
	// they have already reviewed.
	ErrAlreadyReviewed = errors.New("you have already reviewed this product")

	// ErrAlreadyReplied is returned when a seller tries to reply to a review
	// that already has a reply.
	ErrAlreadyReplied = errors.New("this review already has a reply")

	// ErrPurchaseRequired is returned when a buyer tries to review a product
	// they have not purchased.
	ErrPurchaseRequired = errors.New("you can only review products you have purchased")

	// ErrNotReviewOwner is returned when a buyer tries to modify a review
	// that does not belong to them.
	ErrNotReviewOwner = errors.New("you can only modify your own review")

	// ErrNotSellerOfProduct is returned when a seller tries to reply to a
	// review on a product that does not belong to them.
	ErrNotSellerOfProduct = errors.New("you can only reply to reviews on your products")

	// ErrInvalidRating is returned when the rating value is outside 1-5.
	ErrInvalidRating = errors.New("rating must be between 1 and 5")

	// ErrProductNotFound is returned when the specified product does not exist
	// in the catalog.
	ErrProductNotFound = errors.New("product not found")
)
