package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// AuthUseCase is the driving port (inbound) for auth operations.
// Handlers depend on this interface; *service.AuthService satisfies it.
type AuthUseCase interface {
	// Tenant operations
	CreateTenant(ctx context.Context, t *domain.Tenant) error
	GetTenant(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	ListTenants(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error)

	// Seller operations
	CreateSeller(ctx context.Context, tenantID uuid.UUID, seller *domain.Seller) error
	GetSeller(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error)
	ListSellers(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error)
	ApproveSeller(ctx context.Context, tenantID, id uuid.UUID) error

	// Seller subscription / plan operations
	CreatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error
	ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error)
	GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.SubscriptionPlan, error)
	UpdatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error
	GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	SubscribeSeller(ctx context.Context, tenantID, sellerID, planID uuid.UUID) (*domain.SellerSubscription, error)

	// Buyer subscription / plan operations
	CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error
	ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error)
	GetBuyerPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.BuyerPlan, error)
	UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error
	GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	SubscribeBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, planID uuid.UUID) (*domain.BuyerSubscription, error)

	// RBAC — seller team
	LookupSellerRole(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error)
	ListSellerTeam(ctx context.Context, tenantID, sellerID uuid.UUID) ([]domain.SellerUser, error)
	AddSellerUser(ctx context.Context, tenantID, sellerID uuid.UUID, newAuth0UserID string, role domain.SellerUserRole) (*domain.SellerUser, error)
	UpdateSellerUserRole(ctx context.Context, tenantID, sellerID, targetID uuid.UUID, newRole domain.SellerUserRole) error
	RemoveSellerUser(ctx context.Context, tenantID, sellerID, targetID uuid.UUID) error
	TransferSellerOwnership(ctx context.Context, tenantID, sellerID, newOwnerID uuid.UUID) error

	// RBAC — platform admins
	LookupPlatformAdminRole(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (domain.PlatformAdminRole, error)
	ListPlatformAdmins(ctx context.Context, tenantID uuid.UUID) ([]domain.PlatformAdmin, error)
	GrantPlatformAdmin(ctx context.Context, tenantID uuid.UUID, newAuth0UserID string, role domain.PlatformAdminRole) (*domain.PlatformAdmin, error)
	UpdatePlatformAdminRole(ctx context.Context, tenantID, targetID uuid.UUID, newRole domain.PlatformAdminRole) error
	RevokePlatformAdmin(ctx context.Context, tenantID, targetID uuid.UUID) error
	ListRBACAuditLog(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error)
	BootstrapSuperAdmin(ctx context.Context, tenantID uuid.UUID, auth0UserID string) error

	// API tokens
	IssueAPIToken(
		ctx context.Context,
		tenantID, sellerID uuid.UUID,
		name string,
		scopes []domain.APITokenScope,
		rateLimitRPS, rateLimitBurst *int,
		expiresAt *time.Time,
		tokenPrefix string,
	) (*domain.SellerAPIToken, string, error)
	ListAPITokens(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error)
	GetAPIToken(ctx context.Context, tenantID, sellerID, id uuid.UUID) (*domain.SellerAPIToken, error)
	RevokeAPIToken(ctx context.Context, tenantID, sellerID, id uuid.UUID) (prefix, lookup string, err error)
	LookupAPIToken(ctx context.Context, prefix, lookup, secret string) (*domain.SellerAPIToken, error)
}
