package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/port"
)

// AuthService implements business logic for auth operations.
type AuthService struct {
	db                 port.TxRunner
	tenants            port.TenantStore
	sellers            port.SellerStore
	sellerUsers        port.SellerUserStore
	platformAdmins     port.PlatformAdminStore
	rbacAudit          port.RBACAuditStore
	subscriptions      port.SubscriptionStore
	buyerSubscriptions port.BuyerSubscriptionStore
	apiTokens          port.APITokenStore
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	db port.TxRunner,
	tenants port.TenantStore,
	sellers port.SellerStore,
	sellerUsers port.SellerUserStore,
	platformAdmins port.PlatformAdminStore,
	rbacAudit port.RBACAuditStore,
	subscriptions port.SubscriptionStore,
	buyerSubscriptions port.BuyerSubscriptionStore,
	apiTokens port.APITokenStore,
) *AuthService {
	return &AuthService{
		db:                 db,
		tenants:            tenants,
		sellers:            sellers,
		sellerUsers:        sellerUsers,
		platformAdmins:     platformAdmins,
		rbacAudit:          rbacAudit,
		subscriptions:      subscriptions,
		buyerSubscriptions: buyerSubscriptions,
		apiTokens:          apiTokens,
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

// CreateSeller creates a new seller within a tenant and records the calling
// user as the initial owner of that seller organization. Both inserts happen
// in a single transaction so the seller is never left without an owner.
func (s *AuthService) CreateSeller(ctx context.Context, tenantID uuid.UUID, seller *domain.Seller) error {
	// Verify tenant exists.
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return apperrors.Internal("failed to verify tenant", err)
	}
	if t == nil {
		return apperrors.NotFound("tenant not found")
	}

	// The caller becomes the initial owner. Without an identified caller we
	// cannot set up the owning user_role, so fail loudly.
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required to create seller")
	}

	// Check slug uniqueness within tenant (non-TX; UNIQUE constraint is the
	// source of truth for concurrent races).
	existing, err := s.sellers.GetBySlug(ctx, tenantID, seller.Slug)
	if err != nil {
		return apperrors.Internal("failed to check seller slug", err)
	}
	if existing != nil {
		return apperrors.Conflict("seller slug already exists in this tenant")
	}

	seller.Status = domain.SellerStatusPending

	err = s.db.RunTenantTx(ctx, tenantID, func(txCtx context.Context) error {
		if err := s.sellers.Create(txCtx, tenantID, seller); err != nil {
			return err
		}
		owner := &domain.SellerUser{
			TenantID:    tenantID,
			SellerID:    seller.ID,
			Auth0UserID: tc.UserID,
			Role:        domain.SellerUserRoleOwner,
		}
		return s.sellerUsers.Create(txCtx, owner)
	})
	if err != nil {
		return apperrors.Internal("failed to create seller", err)
	}

	slog.Info("seller created", "id", seller.ID, "tenant_id", tenantID, "slug", seller.Slug, "owner", tc.UserID)
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

// --- Buyer Plan Methods ---

// CreateBuyerPlan creates a new buyer subscription plan within a tenant.
func (s *AuthService) CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error {
	if plan.Status == "" {
		plan.Status = "active"
	}
	if plan.PriceCurrency == "" {
		plan.PriceCurrency = "JPY"
	}

	if err := s.buyerSubscriptions.CreateBuyerPlan(ctx, tenantID, plan); err != nil {
		return apperrors.Internal("failed to create buyer plan", err)
	}

	slog.Info("buyer plan created", "id", plan.ID, "tenant_id", tenantID, "slug", plan.Slug)
	return nil
}

// ListBuyerPlans returns all active buyer plans for a tenant.
func (s *AuthService) ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error) {
	plans, err := s.buyerSubscriptions.ListBuyerPlans(ctx, tenantID)
	if err != nil {
		return nil, apperrors.Internal("failed to list buyer plans", err)
	}
	return plans, nil
}

// GetBuyerPlan retrieves a buyer plan by ID.
func (s *AuthService) GetBuyerPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.BuyerPlan, error) {
	plan, err := s.buyerSubscriptions.GetBuyerPlanByID(ctx, tenantID, planID)
	if err != nil {
		return nil, apperrors.Internal("failed to get buyer plan", err)
	}
	if plan == nil {
		return nil, apperrors.NotFound("buyer plan not found")
	}
	return plan, nil
}

// UpdateBuyerPlan modifies an existing buyer plan.
func (s *AuthService) UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error {
	existing, err := s.buyerSubscriptions.GetBuyerPlanByID(ctx, tenantID, plan.ID)
	if err != nil {
		return apperrors.Internal("failed to get buyer plan", err)
	}
	if existing == nil {
		return apperrors.NotFound("buyer plan not found")
	}

	if err := s.buyerSubscriptions.UpdateBuyerPlan(ctx, tenantID, plan); err != nil {
		return apperrors.Internal("failed to update buyer plan", err)
	}

	slog.Info("buyer plan updated", "id", plan.ID, "tenant_id", tenantID)
	return nil
}

// GetBuyerSubscription retrieves the current subscription for a buyer.
func (s *AuthService) GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error) {
	sub, err := s.buyerSubscriptions.GetBuyerSubscription(ctx, tenantID, buyerAuth0ID)
	if err != nil {
		return nil, apperrors.Internal("failed to get buyer subscription", err)
	}
	if sub == nil {
		return nil, apperrors.NotFound("buyer subscription not found")
	}
	return sub, nil
}

// SubscribeBuyer subscribes a buyer to a plan.
func (s *AuthService) SubscribeBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, planID uuid.UUID) (*domain.BuyerSubscription, error) {
	// Verify plan exists.
	plan, err := s.buyerSubscriptions.GetBuyerPlanByID(ctx, tenantID, planID)
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

	if err := s.buyerSubscriptions.UpsertBuyerSubscription(ctx, tenantID, sub); err != nil {
		return nil, apperrors.Internal("failed to subscribe buyer", err)
	}

	slog.Info("buyer subscribed", "buyer_auth0_id", buyerAuth0ID, "plan_id", planID, "tenant_id", tenantID)
	return sub, nil
}
