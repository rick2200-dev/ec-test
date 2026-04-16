// Package app contains the subscription service's application layer.
package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/domain"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/port"

	subscriptionv1 "github.com/Riku-KANO/ec-test/services/subscription/api/gen/go/subscription/v1"
)

const planEventsTopic = "subscription-events"

// Service implements SubscriptionUseCase. It deliberately does not verify
// that the seller or buyer referenced in a subscribe call exists — that
// belonged to the auth-coupled predecessor. Seller/buyer identity is
// authoritative on the caller side; a mismatched seller_id is caught by the
// buffer of server-side authorization the gateway already applies before
// forwarding the request.
type Service struct {
	seller    port.SellerSubscriptionStore
	buyer     port.BuyerSubscriptionStore
	publisher pubsub.Publisher
}

// NewService constructs a Service. publisher may be nil — events are dropped
// silently in that case (useful for tests and for local dev without Pub/Sub).
func NewService(seller port.SellerSubscriptionStore, buyer port.BuyerSubscriptionStore, publisher pubsub.Publisher) *Service {
	return &Service{seller: seller, buyer: buyer, publisher: publisher}
}

// --- Seller Plan Methods ---

func (s *Service) CreatePlan(ctx context.Context, plan *domain.SubscriptionPlan) error {
	if plan.Status == "" {
		plan.Status = "active"
	}
	if plan.PriceCurrency == "" {
		plan.PriceCurrency = "JPY"
	}
	if err := s.seller.CreatePlan(ctx, plan); err != nil {
		return apperrors.Internal("failed to create plan", err)
	}
	slog.Info("subscription plan created", "id", plan.ID, "slug", plan.Slug)
	return nil
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.SubscriptionPlan, error) {
	plans, err := s.seller.ListPlans(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list plans", err)
	}
	return plans, nil
}

func (s *Service) GetPlan(ctx context.Context, planID uuid.UUID) (*domain.SubscriptionPlan, error) {
	plan, err := s.seller.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("plan not found")
	}
	return plan, nil
}

func (s *Service) UpdatePlan(ctx context.Context, plan *domain.SubscriptionPlan) error {
	existing, err := s.seller.GetPlanByID(ctx, plan.ID)
	if err != nil {
		return apperrors.Internal("failed to get plan", err)
	}
	if existing == nil {
		return apperrors.NotFound("plan not found")
	}
	if err := s.seller.UpdatePlan(ctx, plan); err != nil {
		return apperrors.Internal("failed to update plan", err)
	}
	// Plan changes affect every seller already subscribed to it; subscribers
	// resolve the affected cohort on their side.
	pubsub.PublishProtoEvent(ctx, s.publisher, "subscription_plan.updated", planEventsTopic, &subscriptionv1.SubscriptionPlanUpdated{
		PlanId:          plan.ID.String(),
		PlanTier:        int32(plan.Tier),
		SearchBoost:     plan.Features.SearchBoost,
		PromotedResults: int32(plan.Features.PromotedResults),
	})
	slog.Info("subscription plan updated", "id", plan.ID)
	return nil
}

func (s *Service) GetSellerSubscription(ctx context.Context, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error) {
	sub, err := s.seller.GetSellerSubscription(ctx, sellerID)
	if err != nil {
		return nil, apperrors.Internal("failed to get seller subscription", err)
	}
	if sub == nil {
		return nil, apperrors.NotFound("seller subscription not found")
	}
	return sub, nil
}

func (s *Service) SubscribeSeller(ctx context.Context, sellerID, planID uuid.UUID) (*domain.SellerSubscription, error) {
	plan, err := s.seller.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("plan not found")
	}

	sub := &domain.SellerSubscription{
		ID:       uuid.New(),
		SellerID: sellerID,
		PlanID:   planID,
		Status:   domain.SubscriptionStatusActive,
	}
	if err := s.seller.UpsertSellerSubscription(ctx, sub); err != nil {
		return nil, apperrors.Internal("failed to subscribe seller", err)
	}
	// Phase 2.2 completion: publish the seller's new plan so search (and
	// any future consumer) can maintain its own seller_plan_boost
	// projection. The former call into catalog_svc.refresh_seller_plan_boost
	// went away with this event-driven replacement.
	pubsub.PublishProtoEvent(ctx, s.publisher, "seller.subscribed", planEventsTopic, &subscriptionv1.SellerSubscribed{
		SellerId:        sellerID.String(),
		PlanId:          planID.String(),
		PlanSlug:        plan.Slug,
		PlanTier:        int32(plan.Tier),
		SearchBoost:     plan.Features.SearchBoost,
		PromotedResults: int32(plan.Features.PromotedResults),
	})
	slog.Info("seller subscribed", "seller_id", sellerID, "plan_id", planID)
	return sub, nil
}

// --- Buyer Plan Methods ---

func (s *Service) CreateBuyerPlan(ctx context.Context, plan *domain.BuyerPlan) error {
	if plan.Status == "" {
		plan.Status = "active"
	}
	if plan.PriceCurrency == "" {
		plan.PriceCurrency = "JPY"
	}
	if err := s.buyer.CreateBuyerPlan(ctx, plan); err != nil {
		return apperrors.Internal("failed to create buyer plan", err)
	}
	slog.Info("buyer plan created", "id", plan.ID, "slug", plan.Slug)
	return nil
}

func (s *Service) ListBuyerPlans(ctx context.Context) ([]domain.BuyerPlan, error) {
	plans, err := s.buyer.ListBuyerPlans(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list buyer plans", err)
	}
	return plans, nil
}

func (s *Service) GetBuyerPlan(ctx context.Context, planID uuid.UUID) (*domain.BuyerPlan, error) {
	plan, err := s.buyer.GetBuyerPlanByID(ctx, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get buyer plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("buyer plan not found")
	}
	return plan, nil
}

func (s *Service) UpdateBuyerPlan(ctx context.Context, plan *domain.BuyerPlan) error {
	existing, err := s.buyer.GetBuyerPlanByID(ctx, plan.ID)
	if err != nil {
		return apperrors.Internal("failed to get buyer plan", err)
	}
	if existing == nil {
		return apperrors.NotFound("buyer plan not found")
	}
	if err := s.buyer.UpdateBuyerPlan(ctx, plan); err != nil {
		return apperrors.Internal("failed to update buyer plan", err)
	}
	slog.Info("buyer plan updated", "id", plan.ID)
	return nil
}

func (s *Service) GetBuyerSubscription(ctx context.Context, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error) {
	sub, err := s.buyer.GetBuyerSubscription(ctx, buyerAuth0ID)
	if err != nil {
		return nil, apperrors.Internal("failed to get buyer subscription", err)
	}
	if sub == nil {
		return nil, apperrors.NotFound("buyer subscription not found")
	}
	return sub, nil
}

func (s *Service) SubscribeBuyer(ctx context.Context, buyerAuth0ID string, planID uuid.UUID) (*domain.BuyerSubscription, error) {
	plan, err := s.buyer.GetBuyerPlanByID(ctx, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get buyer plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("buyer plan not found")
	}

	sub := &domain.BuyerSubscription{
		ID:           uuid.New(),
		BuyerAuth0ID: buyerAuth0ID,
		PlanID:       planID,
		Status:       domain.SubscriptionStatusActive,
	}
	if err := s.buyer.UpsertBuyerSubscription(ctx, sub); err != nil {
		return nil, apperrors.Internal("failed to subscribe buyer", err)
	}
	slog.Info("buyer subscribed", "buyer_auth0_id", buyerAuth0ID, "plan_id", planID)
	return sub, nil
}
