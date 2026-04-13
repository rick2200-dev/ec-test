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
	CreatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error
	ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error)
	GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.SubscriptionPlan, error)
	UpdatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error
	GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	SubscribeSeller(ctx context.Context, tenantID, sellerID, planID uuid.UUID) (*domain.SellerSubscription, error)

	// Buyer plans
	CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error
	ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error)
	GetBuyerPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.BuyerPlan, error)
	UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error
	GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	SubscribeBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, planID uuid.UUID) (*domain.BuyerSubscription, error)
}

// SellerSubscriptionStore abstracts the seller plan / subscription persistence.
type SellerSubscriptionStore interface {
	CreatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error
	GetPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionPlan, error)
	ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error)
	UpdatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error
	GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	UpsertSellerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.SellerSubscription) error
	// RefreshPlanBoostView refreshes the catalog_svc.seller_plan_boost
	// materialized view. The view lives in catalog_svc for locality with
	// the search engine but its source tables are now in subscription_svc.
	RefreshPlanBoostView(ctx context.Context) error
}

// BuyerSubscriptionStore abstracts the buyer plan / subscription persistence.
type BuyerSubscriptionStore interface {
	CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error
	GetBuyerPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BuyerPlan, error)
	ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error)
	UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error
	GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	UpsertBuyerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.BuyerSubscription) error
}
