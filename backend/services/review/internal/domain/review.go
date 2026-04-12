package domain

import (
	"time"

	"github.com/google/uuid"
)

// Review represents a buyer's review of a product they have purchased.
// One review per buyer per product is enforced at the database level.
type Review struct {
	ID           uuid.UUID    `json:"id"`
	TenantID     uuid.UUID    `json:"tenant_id"`
	BuyerAuth0ID string       `json:"buyer_auth0_id"`
	ProductID    uuid.UUID    `json:"product_id"`
	SellerID     uuid.UUID    `json:"seller_id"`
	ProductName  string       `json:"product_name"`
	Rating       int          `json:"rating"`
	Title        string       `json:"title"`
	Body         string       `json:"body"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Reply        *ReviewReply `json:"reply,omitempty"`
}

// ReviewReply represents a seller's reply to a review.
// One reply per review is enforced at the database level.
type ReviewReply struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	ReviewID      uuid.UUID `json:"review_id"`
	SellerAuth0ID string    `json:"seller_auth0_id"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ProductRating is the denormalized aggregate rating for a product.
type ProductRating struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	ProductID     uuid.UUID `json:"product_id"`
	AverageRating float64   `json:"average_rating"`
	ReviewCount   int       `json:"review_count"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateReviewInput is the input for creating a new review.
// SellerID, ProductName are resolved from the catalog service;
// the caller only provides the product_id and review content.
type CreateReviewInput struct {
	ProductID uuid.UUID `json:"product_id"`
	Rating    int       `json:"rating"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
}

// UpdateReviewInput is the input for updating an existing review.
// All fields are optional; only non-nil fields are applied.
type UpdateReviewInput struct {
	Rating *int    `json:"rating,omitempty"`
	Title  *string `json:"title,omitempty"`
	Body   *string `json:"body,omitempty"`
}

// CreateReplyInput is the input for creating a seller reply.
type CreateReplyInput struct {
	Body string `json:"body"`
}

// UpdateReplyInput is the input for updating a seller reply.
type UpdateReplyInput struct {
	Body string `json:"body"`
}
