package domain

import (
	"time"

	"github.com/google/uuid"
)

// PlanFeatures describes the capabilities granted by a subscription plan.
type PlanFeatures struct {
	MaxProducts     int     `json:"max_products"`
	SearchBoost     float64 `json:"search_boost"`
	FeaturedSlots   int     `json:"featured_slots"`
	PromotedResults int     `json:"promoted_results"`
}

// SubscriptionPlan represents a plan tier available within a tenant.
type SubscriptionPlan struct {
	ID            uuid.UUID    `json:"id"`
	TenantID      uuid.UUID    `json:"tenant_id"`
	Name          string       `json:"name"`
	Slug          string       `json:"slug"`
	Tier          int          `json:"tier"`
	PriceAmount   int64        `json:"price_amount"`
	PriceCurrency string       `json:"price_currency"`
	Features      PlanFeatures `json:"features"`
	StripePriceID string       `json:"stripe_price_id,omitempty"`
	Status        string       `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// SubscriptionStatus represents the lifecycle state of a seller subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
)

// SellerSubscription tracks which plan a seller is currently on.
type SellerSubscription struct {
	ID                   uuid.UUID          `json:"id"`
	TenantID             uuid.UUID          `json:"tenant_id"`
	SellerID             uuid.UUID          `json:"seller_id"`
	PlanID               uuid.UUID          `json:"plan_id"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty"`
	StripeCustomerID     string             `json:"stripe_customer_id,omitempty"`
	Status               SubscriptionStatus `json:"status"`
	CurrentPeriodStart   *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end,omitempty"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// SellerSubscriptionWithPlan combines a subscription with its plan details.
type SellerSubscriptionWithPlan struct {
	SellerSubscription
	PlanName string `json:"plan_name"`
	PlanSlug string `json:"plan_slug"`
	PlanTier int    `json:"plan_tier"`
}

// --- Buyer Subscription Types ---

// BuyerPlanFeatures describes the capabilities granted by a buyer subscription plan.
type BuyerPlanFeatures struct {
	FreeShipping bool `json:"free_shipping"`
}

// BuyerPlan represents a buyer subscription plan tier available within a tenant.
type BuyerPlan struct {
	ID            uuid.UUID         `json:"id"`
	TenantID      uuid.UUID         `json:"tenant_id"`
	Name          string            `json:"name"`
	Slug          string            `json:"slug"`
	PriceAmount   int64             `json:"price_amount"`
	PriceCurrency string            `json:"price_currency"`
	Features      BuyerPlanFeatures `json:"features"`
	StripePriceID string            `json:"stripe_price_id,omitempty"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// BuyerSubscription tracks which plan a buyer is currently on.
type BuyerSubscription struct {
	ID                   uuid.UUID          `json:"id"`
	TenantID             uuid.UUID          `json:"tenant_id"`
	BuyerAuth0ID         string             `json:"buyer_auth0_id"`
	PlanID               uuid.UUID          `json:"plan_id"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty"`
	StripeCustomerID     string             `json:"stripe_customer_id,omitempty"`
	Status               SubscriptionStatus `json:"status"`
	CurrentPeriodStart   *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end,omitempty"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// BuyerSubscriptionWithPlan combines a buyer subscription with its plan details.
type BuyerSubscriptionWithPlan struct {
	BuyerSubscription
	PlanName string            `json:"plan_name"`
	PlanSlug string            `json:"plan_slug"`
	Features BuyerPlanFeatures `json:"features"`
}
