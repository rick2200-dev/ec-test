package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/subscription/internal/domain"
)

// SubscriptionUseCase is the inbound application-level interface. Both the
// gRPC and HTTP adapters depend on it, so handler code has no knowledge of
// the underlying store implementation.
type SubscriptionUseCase interface {
	// Seller plans
	CreatePlan(ctx context.Context, plan *domain.SubscriptionPlan) error
	ListPlans(ctx context.Context) ([]domain.SubscriptionPlan, error)
	GetPlan(ctx context.Context, planID uuid.UUID) (*domain.SubscriptionPlan, error)
	UpdatePlan(ctx context.Context, plan *domain.SubscriptionPlan) error
	GetSellerSubscription(ctx context.Context, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	SubscribeSeller(ctx context.Context, sellerID, planID uuid.UUID) (*domain.SellerSubscription, error)

	// Buyer plans
	CreateBuyerPlan(ctx context.Context, plan *domain.BuyerPlan) error
	ListBuyerPlans(ctx context.Context) ([]domain.BuyerPlan, error)
	GetBuyerPlan(ctx context.Context, planID uuid.UUID) (*domain.BuyerPlan, error)
	UpdateBuyerPlan(ctx context.Context, plan *domain.BuyerPlan) error
	GetBuyerSubscription(ctx context.Context, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	SubscribeBuyer(ctx context.Context, buyerAuth0ID string, planID uuid.UUID) (*domain.BuyerSubscription, error)
}

// SellerSubscriptionStore abstracts the seller plan / subscription persistence.
type SellerSubscriptionStore interface {
	CreatePlan(ctx context.Context, plan *domain.SubscriptionPlan) error
	GetPlanByID(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error)
	ListPlans(ctx context.Context) ([]domain.SubscriptionPlan, error)
	UpdatePlan(ctx context.Context, plan *domain.SubscriptionPlan) error
	GetSellerSubscription(ctx context.Context, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	UpsertSellerSubscription(ctx context.Context, sub *domain.SellerSubscription) error
	// RefreshPlanBoostView refreshes the catalog_svc.seller_plan_boost
	// materialized view. The view lives in catalog_svc for locality with
	// the search engine but its source tables are now in subscription_svc.
	RefreshPlanBoostView(ctx context.Context) error
}

// BuyerSubscriptionStore abstracts the buyer plan / subscription persistence.
type BuyerSubscriptionStore interface {
	CreateBuyerPlan(ctx context.Context, plan *domain.BuyerPlan) error
	GetBuyerPlanByID(ctx context.Context, id uuid.UUID) (*domain.BuyerPlan, error)
	ListBuyerPlans(ctx context.Context) ([]domain.BuyerPlan, error)
	UpdateBuyerPlan(ctx context.Context, plan *domain.BuyerPlan) error
	GetBuyerSubscription(ctx context.Context, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	UpsertBuyerSubscription(ctx context.Context, sub *domain.BuyerSubscription) error
}
