package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/repository"
)

// AuthService implements business logic for auth operations.
type AuthService struct {
	tenants       *repository.TenantRepository
	sellers       *repository.SellerRepository
	subscriptions *repository.SubscriptionRepository
}

// NewAuthService creates a new AuthService.
func NewAuthService(tenants *repository.TenantRepository, sellers *repository.SellerRepository, subscriptions *repository.SubscriptionRepository) *AuthService {
	return &AuthService{
		tenants:       tenants,
		sellers:       sellers,
		subscriptions: subscriptions,
	}
}

// CreateTenant creates a new tenant.
func (s *AuthService) CreateTenant(ctx context.Context, t *domain.Tenant) error {
	// Check for slug uniqueness.
	existing, err := s.tenants.GetBySlug(ctx, t.Slug)
	if err != nil {
		return apperrors.Internal("failed to check tenant slug", err)
	}
	if existing != nil {
		return apperrors.Conflict("tenant slug already exists")
	}

	t.Status = domain.TenantStatusActive
	if err := s.tenants.Create(ctx, t); err != nil {
		return apperrors.Internal("failed to create tenant", err)
	}

	slog.Info("tenant created", "id", t.ID, "slug", t.Slug)
	return nil
}

// GetTenant retrieves a tenant by ID.
func (s *AuthService) GetTenant(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t, err := s.tenants.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get tenant", err)
	}
	if t == nil {
		return nil, apperrors.NotFound("tenant not found")
	}
	return t, nil
}

// ListTenants returns a paginated list of tenants.
func (s *AuthService) ListTenants(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error) {
	tenants, total, err := s.tenants.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list tenants", err)
	}
	return tenants, total, nil
}

// CreateSeller creates a new seller within a tenant.
func (s *AuthService) CreateSeller(ctx context.Context, tenantID uuid.UUID, seller *domain.Seller) error {
	// Verify tenant exists.
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return apperrors.Internal("failed to verify tenant", err)
	}
	if t == nil {
		return apperrors.NotFound("tenant not found")
	}

	// Check slug uniqueness within tenant.
	existing, err := s.sellers.GetBySlug(ctx, tenantID, seller.Slug)
	if err != nil {
		return apperrors.Internal("failed to check seller slug", err)
	}
	if existing != nil {
		return apperrors.Conflict("seller slug already exists in this tenant")
	}

	seller.Status = domain.SellerStatusPending
	if err := s.sellers.Create(ctx, tenantID, seller); err != nil {
		return apperrors.Internal("failed to create seller", err)
	}

	slog.Info("seller created", "id", seller.ID, "tenant_id", tenantID, "slug", seller.Slug)
	return nil
}

// GetSeller retrieves a seller by ID within a tenant.
func (s *AuthService) GetSeller(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error) {
	seller, err := s.sellers.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get seller", err)
	}
	if seller == nil {
		return nil, apperrors.NotFound("seller not found")
	}
	return seller, nil
}

// ListSellers returns a paginated list of sellers for a tenant.
func (s *AuthService) ListSellers(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error) {
	sellers, total, err := s.sellers.List(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list sellers", err)
	}
	return sellers, total, nil
}

// ApproveSeller transitions a seller to the approved status.
func (s *AuthService) ApproveSeller(ctx context.Context, tenantID, id uuid.UUID) error {
	seller, err := s.sellers.GetByID(ctx, tenantID, id)
	if err != nil {
		return apperrors.Internal("failed to get seller", err)
	}
	if seller == nil {
		return apperrors.NotFound("seller not found")
	}
	if seller.Status != domain.SellerStatusPending {
		return apperrors.BadRequest("seller is not in pending status")
	}

	if err := s.sellers.UpdateStatus(ctx, tenantID, id, domain.SellerStatusApproved); err != nil {
		return apperrors.Internal("failed to approve seller", err)
	}

	slog.Info("seller approved", "id", id, "tenant_id", tenantID)
	return nil
}

// --- Subscription Plan Methods ---

// CreatePlan creates a new subscription plan within a tenant.
func (s *AuthService) CreatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error {
	if plan.Status == "" {
		plan.Status = "active"
	}
	if plan.PriceCurrency == "" {
		plan.PriceCurrency = "JPY"
	}

	if err := s.subscriptions.CreatePlan(ctx, tenantID, plan); err != nil {
		return apperrors.Internal("failed to create plan", err)
	}

	slog.Info("subscription plan created", "id", plan.ID, "tenant_id", tenantID, "slug", plan.Slug)
	return nil
}

// ListPlans returns all active subscription plans for a tenant.
func (s *AuthService) ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error) {
	plans, err := s.subscriptions.ListPlans(ctx, tenantID)
	if err != nil {
		return nil, apperrors.Internal("failed to list plans", err)
	}
	return plans, nil
}

// GetPlan retrieves a subscription plan by ID.
func (s *AuthService) GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.SubscriptionPlan, error) {
	plan, err := s.subscriptions.GetPlanByID(ctx, tenantID, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("plan not found")
	}
	return plan, nil
}

// UpdatePlan modifies an existing subscription plan.
func (s *AuthService) UpdatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error {
	existing, err := s.subscriptions.GetPlanByID(ctx, tenantID, plan.ID)
	if err != nil {
		return apperrors.Internal("failed to get plan", err)
	}
	if existing == nil {
		return apperrors.NotFound("plan not found")
	}

	if err := s.subscriptions.UpdatePlan(ctx, tenantID, plan); err != nil {
		return apperrors.Internal("failed to update plan", err)
	}

	slog.Info("subscription plan updated", "id", plan.ID, "tenant_id", tenantID)
	return nil
}

// GetSellerSubscription retrieves the current subscription for a seller.
func (s *AuthService) GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error) {
	sub, err := s.subscriptions.GetSellerSubscription(ctx, tenantID, sellerID)
	if err != nil {
		return nil, apperrors.Internal("failed to get seller subscription", err)
	}
	if sub == nil {
		return nil, apperrors.NotFound("seller subscription not found")
	}
	return sub, nil
}

// SubscribeSeller subscribes a seller to a plan. Returns a Stripe Checkout URL for paid plans.
func (s *AuthService) SubscribeSeller(ctx context.Context, tenantID, sellerID, planID uuid.UUID) (*domain.SellerSubscription, error) {
	// Verify seller exists.
	seller, err := s.sellers.GetByID(ctx, tenantID, sellerID)
	if err != nil {
		return nil, apperrors.Internal("failed to get seller", err)
	}
	if seller == nil {
		return nil, apperrors.NotFound("seller not found")
	}

	// Verify plan exists.
	plan, err := s.subscriptions.GetPlanByID(ctx, tenantID, planID)
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

	if err := s.subscriptions.UpsertSellerSubscription(ctx, tenantID, sub); err != nil {
		return nil, apperrors.Internal("failed to subscribe seller", err)
	}

	// Refresh the materialized view so search picks up the change.
	if err := s.subscriptions.RefreshPlanBoostView(ctx); err != nil {
		slog.Warn("failed to refresh plan boost view", "error", err)
	}

	slog.Info("seller subscribed", "seller_id", sellerID, "plan_id", planID, "tenant_id", tenantID)
	return sub, nil
}
