package domain

import (
	"time"

	"github.com/google/uuid"
)

// RecommendationType defines the kind of recommendation to generate.
type RecommendationType string

const (
	Popular                  RecommendationType = "popular"
	Similar                  RecommendationType = "similar"
	PersonalizedForYou       RecommendationType = "for_you"
	FrequentlyBoughtTogether RecommendationType = "frequently_bought_together"
)

// UserEventType defines the kind of user behavior event.
type UserEventType string

const (
	ProductViewed UserEventType = "product_viewed"
	AddedToCart   UserEventType = "added_to_cart"
	Purchased     UserEventType = "purchased"
)

// RecommendRequest holds parameters for generating recommendations.
type RecommendRequest struct {
	TenantID  uuid.UUID
	UserID    string
	ProductID *uuid.UUID // required for Similar and FrequentlyBoughtTogether
	Type      RecommendationType
	Limit     int
}

// RecommendResponse holds the result of a recommendation query.
type RecommendResponse struct {
	Products []RecommendedProduct `json:"products"`
}

// RecommendedProduct is a single product recommendation with scoring metadata.
type RecommendedProduct struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	SellerID      uuid.UUID `json:"seller_id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	PriceAmount   int64     `json:"price_amount"`
	PriceCurrency string    `json:"price_currency"`
	Score         float64   `json:"score"`
	Reason        string    `json:"reason"`
}

// UserEvent represents a user behavior event for recommendation tracking.
type UserEvent struct {
	ID        uuid.UUID     `json:"id"`
	TenantID  uuid.UUID     `json:"tenant_id"`
	UserID    string        `json:"user_id"`
	EventType UserEventType `json:"event_type"`
	ProductID uuid.UUID     `json:"product_id"`
	CreatedAt time.Time     `json:"created_at"`
}
