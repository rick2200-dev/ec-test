// Package app contains the subscription service's application layer.
package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/domain"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/port"
)

// Service implements SubscriptionUseCase. It deliberately does not verify
// that the seller or buyer referenced in a subscribe call exists — that
// belonged to the auth-coupled predecessor. Seller/buyer identity is
// authoritative on the caller side; a mismatched seller_id is caught by the
// buffer of server-side authorization the gateway already applies before
// forwarding the request.
type Service struct {
	seller port.SellerSubscriptionStore
	buyer  port.BuyerSubscriptionStore
}

// NewService constructs a Service.
func NewService(seller port.SellerSubscriptionStore, buyer port.BuyerSubscriptionStore) *Service {
	return &Service{seller: seller, buyer: buyer}
}

// --- Seller Plan Methods ---

func (s *Service) CreatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error {
	if plan.Status == "" {
		plan.Status = "active"
	}
	if plan.PriceCurrency == "" {
		plan.PriceCurrency = "JPY"
	}
	if err := s.seller.CreatePlan(ctx, tenantID, plan); err != nil {
		return apperrors.Internal("failed to create plan", err)
	}
	slog.Info("subscription plan created", "id", plan.ID, "tenant_id", tenantID, "slug", plan.Slug)
	return nil
}

func (s *Service) ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error) {
	plans, err := s.seller.ListPlans(ctx, tenantID)
	if err != nil {
		return nil, apperrors.Internal("failed to list plans", err)
	}
	return plans, nil
}

func (s *Service) GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.SubscriptionPlan, error) {
	plan, err := s.seller.GetPlanByID(ctx, tenantID, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("plan not found")
	}
	return plan, nil
}

func (s *Service) UpdatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error {
	existing, err := s.seller.GetPlanByID(ctx, tenantID, plan.ID)
	if err != nil {
		return apperrors.Internal("failed to get plan", err)
	}
	if existing == nil {
		return apperrors.NotFound("plan not found")
	}
	if err := s.seller.UpdatePlan(ctx, tenantID, plan); err != nil {
		return apperrors.Internal("failed to update plan", err)
	}
	slog.Info("subscription plan updated", "id", plan.ID, "tenant_id", tenantID)
	return nil
}

func (s *Service) GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error) {
	sub, err := s.seller.GetSellerSubscription(ctx, tenantID, sellerID)
	if err != nil {
		return nil, apperrors.Internal("failed to get seller subscription", err)
	}
	if sub == nil {
		return nil, apperrors.NotFound("seller subscription not found")
	}
	return sub, nil
}

func (s *Service) SubscribeSeller(ctx context.Context, tenantID, sellerID, planID uuid.UUID) (*domain.SellerSubscription, error) {
	plan, err := s.seller.GetPlanByID(ctx, tenantID, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("plan not found")
	}

	sub := &domain.SellerSubscription{
		ID:       uuid.New(),
		TenantID: tenantID,
		SellerID: sellerID,
		PlanID:   planID,
		Status:   domain.SubscriptionStatusActive,
	}
	if err := s.seller.UpsertSellerSubscription(ctx, tenantID, sub); err != nil {
		return nil, apperrors.Internal("failed to subscribe seller", err)
	}
	// Refresh the materialized view so search ranking picks up the new tier.
	// Best-effort: a failed refresh is logged but not fatal — the view is
	// also refreshed on a schedule by search.
	if err := s.seller.RefreshPlanBoostView(ctx); err != nil {
		slog.Warn("failed to refresh plan boost view", "error", err)
	}
	slog.Info("seller subscribed", "seller_id", sellerID, "plan_id", planID, "tenant_id", tenantID)
	return sub, nil
}

// --- Buyer Plan Methods ---

func (s *Service) CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error {
	if plan.Status == "" {
		plan.Status = "active"
	}
	if plan.PriceCurrency == "" {
		plan.PriceCurrency = "JPY"
	}
	if err := s.buyer.CreateBuyerPlan(ctx, tenantID, plan); err != nil {
		return apperrors.Internal("failed to create buyer plan", err)
	}
	slog.Info("buyer plan created", "id", plan.ID, "tenant_id", tenantID, "slug", plan.Slug)
	return nil
}

func (s *Service) ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error) {
	plans, err := s.buyer.ListBuyerPlans(ctx, tenantID)
	if err != nil {
		return nil, apperrors.Internal("failed to list buyer plans", err)
	}
	return plans, nil
}

func (s *Service) GetBuyerPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.BuyerPlan, error) {
	plan, err := s.buyer.GetBuyerPlanByID(ctx, tenantID, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get buyer plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("buyer plan not found")
	}
	return plan, nil
}

func (s *Service) UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error {
	existing, err := s.buyer.GetBuyerPlanByID(ctx, tenantID, plan.ID)
	if err != nil {
		return apperrors.Internal("failed to get buyer plan", err)
	}
	if existing == nil {
		return apperrors.NotFound("buyer plan not found")
	}
	if err := s.buyer.UpdateBuyerPlan(ctx, tenantID, plan); err != nil {
		return apperrors.Internal("failed to update buyer plan", err)
	}
	slog.Info("buyer plan updated", "id", plan.ID, "tenant_id", tenantID)
	return nil
}

func (s *Service) GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error) {
	sub, err := s.buyer.GetBuyerSubscription(ctx, tenantID, buyerAuth0ID)
	if err != nil {
		return nil, apperrors.Internal("failed to get buyer subscription", err)
	}
	if sub == nil {
		return nil, apperrors.NotFound("buyer subscription not found")
	}
	return sub, nil
}

func (s *Service) SubscribeBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, planID uuid.UUID) (*domain.BuyerSubscription, error) {
	plan, err := s.buyer.GetBuyerPlanByID(ctx, tenantID, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get buyer plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("buyer plan not found")
	}

	sub := &domain.BuyerSubscription{
		ID:           uuid.New(),
		TenantID:     tenantID,
		BuyerAuth0ID: buyerAuth0ID,
		PlanID:       planID,
		Status:       domain.SubscriptionStatusActive,
	}
	if err := s.buyer.UpsertBuyerSubscription(ctx, tenantID, sub); err != nil {
		return nil, apperrors.Internal("failed to subscribe buyer", err)
	}
	slog.Info("buyer subscribed", "buyer_auth0_id", buyerAuth0ID, "plan_id", planID, "tenant_id", tenantID)
	return sub, nil
}
