// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the auth service.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// TxRunner starts a tenant-scoped transaction and embeds it in the context.
// Repository methods extract the transaction via database.TxFromContext so
// pgx.Tx never appears in service or port signatures.
// *database.PoolTxRunner satisfies this interface.
type TxRunner interface {
	RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error
}

// TenantStore is the driven port for tenant persistence.
// *repository.TenantRepository satisfies this interface.
type TenantStore interface {
	Create(ctx context.Context, t *domain.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	List(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error)
}

// SellerStore is the driven port for seller persistence.
// *repository.SellerRepository satisfies this interface.
type SellerStore interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error)
	GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Seller, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.SellerStatus) error
	Create(ctx context.Context, tenantID uuid.UUID, s *domain.Seller) error
}

// SellerUserStore is the driven port for seller_user persistence.
// *repository.SellerUserRepository satisfies this interface.
type SellerUserStore interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerUser, error)
	GetByAuth0ID(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (*domain.SellerUser, error)
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID) ([]domain.SellerUser, error)
	Create(ctx context.Context, su *domain.SellerUser) error
	UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.SellerUserRole) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	CountByRole(ctx context.Context, tenantID, sellerID uuid.UUID, role domain.SellerUserRole) (int, error)
	// CheckRole returns the actor's role within a seller org. Returns ("", nil)
	// when the user is not a member of the seller.
	CheckRole(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error)
}

// PlatformAdminStore is the driven port for platform_admin persistence.
// *repository.PlatformAdminRepository satisfies this interface.
type PlatformAdminStore interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformAdmin, error)
	GetByAuth0ID(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (*domain.PlatformAdmin, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.PlatformAdmin, error)
	CountByRole(ctx context.Context, tenantID uuid.UUID, role domain.PlatformAdminRole) (int, error)
	Create(ctx context.Context, pa *domain.PlatformAdmin) error
	UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.PlatformAdminRole) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// CheckRole returns the actor's role as a platform admin in the tenant.
	// Returns ("", nil) when the user is not an admin.
	CheckRole(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (domain.PlatformAdminRole, error)
}

// RBACAuditStore is the driven port for rbac_audit_log persistence.
// *repository.RBACAuditRepository satisfies this interface.
type RBACAuditStore interface {
	Append(ctx context.Context, e *domain.RBACAuditEntry) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error)
}

// SubscriptionStore is the driven port for seller subscription persistence.
// *repository.SubscriptionRepository satisfies this interface.
type SubscriptionStore interface {
	CreatePlan(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error
	GetPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionPlan, error)
	ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error)
	UpdatePlan(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error
	GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	UpsertSellerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.SellerSubscription) error
	RefreshPlanBoostView(ctx context.Context) error
}

// BuyerSubscriptionStore is the driven port for buyer subscription persistence.
// *repository.BuyerSubscriptionRepository satisfies this interface.
type BuyerSubscriptionStore interface {
	CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, p *domain.BuyerPlan) error
	GetBuyerPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BuyerPlan, error)
	ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error)
	UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, p *domain.BuyerPlan) error
	GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	UpsertBuyerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.BuyerSubscription) error
}

// APITokenStore is the driven port for seller API token persistence.
// *repository.APITokenRepository satisfies this interface.
type APITokenStore interface {
	Create(ctx context.Context, t *domain.SellerAPIToken) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerAPIToken, error)
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error)
	Revoke(ctx context.Context, tenantID, id uuid.UUID, actorAuth0UserID string) error
	GetByLookup(ctx context.Context, prefix, lookup string) (*domain.SellerAPIToken, error)
	TouchLastUsedAt(ctx context.Context, id uuid.UUID) error
}

