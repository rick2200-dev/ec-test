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
	// RunTenantTx executes fn within a tenant-scoped database transaction.
	// The transaction is embedded in the returned context so repository methods
	// can extract it via database.TxFromContext without leaking pgx.Tx into signatures.
	RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error
}

// TenantStore is the driven port for tenant persistence.
// *repository.TenantRepository satisfies this interface.
type TenantStore interface {
	// Create persists a new tenant.
	Create(ctx context.Context, t *domain.Tenant) error
	// GetByID retrieves a tenant by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	// GetBySlug retrieves a tenant by its URL-friendly slug.
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	// List returns a paginated list of all tenants.
	List(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error)
}

// SellerStore is the driven port for seller persistence.
// *repository.SellerRepository satisfies this interface.
type SellerStore interface {
	// GetByID retrieves a seller by its UUID within the tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error)
	// GetBySlug retrieves a seller by its URL-friendly slug within the tenant.
	GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Seller, error)
	// List returns a paginated list of sellers within the tenant.
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error)
	// UpdateStatus changes the approval status of a seller (e.g. "pending" → "active").
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.SellerStatus) error
	// Create persists a new seller within the tenant (initial status: pending).
	Create(ctx context.Context, tenantID uuid.UUID, s *domain.Seller) error
}

// SellerUserStore is the driven port for seller_user persistence.
// *repository.SellerUserRepository satisfies this interface.
type SellerUserStore interface {
	// GetByID retrieves a seller team member by their UUID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerUser, error)
	// GetByAuth0ID retrieves a seller team member by their Auth0 user ID within the seller.
	GetByAuth0ID(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (*domain.SellerUser, error)
	// ListBySeller returns all team members for the given seller.
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID) ([]domain.SellerUser, error)
	// Create adds a user to a seller team with the given role.
	Create(ctx context.Context, su *domain.SellerUser) error
	// UpdateRole changes the role of an existing seller team member.
	UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.SellerUserRole) error
	// Delete removes a user from a seller team.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// CountByRole returns the number of seller team members with the given role.
	CountByRole(ctx context.Context, tenantID, sellerID uuid.UUID, role domain.SellerUserRole) (int, error)
	// CheckRole returns the actor's role within a seller org. Returns ("", nil)
	// when the user is not a member of the seller.
	CheckRole(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error)
}

// PlatformAdminStore is the driven port for platform_admin persistence.
// *repository.PlatformAdminRepository satisfies this interface.
type PlatformAdminStore interface {
	// GetByID retrieves a platform admin by their UUID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformAdmin, error)
	// GetByAuth0ID retrieves a platform admin by their Auth0 user ID.
	GetByAuth0ID(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (*domain.PlatformAdmin, error)
	// List returns all platform admins for the tenant.
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.PlatformAdmin, error)
	// CountByRole returns the number of platform admins with the given role.
	CountByRole(ctx context.Context, tenantID uuid.UUID, role domain.PlatformAdminRole) (int, error)
	// Create grants platform admin status to a user.
	Create(ctx context.Context, pa *domain.PlatformAdmin) error
	// UpdateRole changes the role of a platform admin.
	UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.PlatformAdminRole) error
	// Delete revokes platform admin status from a user.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// CheckRole returns the actor's role as a platform admin in the tenant.
	// Returns ("", nil) when the user is not an admin.
	CheckRole(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (domain.PlatformAdminRole, error)
}

// RBACAuditStore is the driven port for rbac_audit_log persistence.
// *repository.RBACAuditRepository satisfies this interface.
type RBACAuditStore interface {
	// Append appends a new RBAC audit log entry.
	Append(ctx context.Context, e *domain.RBACAuditEntry) error
	// ListByTenant returns a paginated list of RBAC audit entries for the tenant.
	ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error)
}

// SubscriptionStore is the driven port for seller subscription persistence.
// *repository.SubscriptionRepository satisfies this interface.
type SubscriptionStore interface {
	// CreatePlan persists a new seller subscription plan.
	CreatePlan(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error
	// GetPlanByID retrieves a seller subscription plan by its UUID.
	GetPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionPlan, error)
	// ListPlans returns all seller subscription plans for the tenant.
	ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error)
	// UpdatePlan persists changes to an existing seller subscription plan.
	UpdatePlan(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error
	// GetSellerSubscription retrieves a seller's active subscription together with its plan details.
	GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	// UpsertSellerSubscription inserts or updates a seller's subscription record.
	UpsertSellerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.SellerSubscription) error
	// RefreshPlanBoostView refreshes the materialized view used to compute search-ranking boost scores from seller plans.
	RefreshPlanBoostView(ctx context.Context) error
}

// BuyerSubscriptionStore is the driven port for buyer subscription persistence.
// *repository.BuyerSubscriptionRepository satisfies this interface.
type BuyerSubscriptionStore interface {
	// CreateBuyerPlan persists a new buyer subscription plan.
	CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, p *domain.BuyerPlan) error
	// GetBuyerPlanByID retrieves a buyer subscription plan by its UUID.
	GetBuyerPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BuyerPlan, error)
	// ListBuyerPlans returns all buyer subscription plans for the tenant.
	ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error)
	// UpdateBuyerPlan persists changes to an existing buyer subscription plan.
	UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, p *domain.BuyerPlan) error
	// GetBuyerSubscription retrieves a buyer's active subscription together with its plan details.
	GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	// UpsertBuyerSubscription inserts or updates a buyer's subscription record.
	UpsertBuyerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.BuyerSubscription) error
}

// APITokenStore is the driven port for seller API token persistence.
// *repository.APITokenRepository satisfies this interface.
type APITokenStore interface {
	// Create persists a new seller API token (stores the hashed secret, not the plaintext).
	Create(ctx context.Context, t *domain.SellerAPIToken) error
	// GetByID retrieves an API token record by its UUID.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerAPIToken, error)
	// ListBySeller returns a paginated list of API tokens for the seller.
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error)
	// Revoke marks an API token as revoked; the token cannot be used after this call.
	Revoke(ctx context.Context, tenantID, id uuid.UUID, actorAuth0UserID string) error
	// GetByLookup retrieves a token record by its prefix and lookup hash for authentication.
	GetByLookup(ctx context.Context, prefix, lookup string) (*domain.SellerAPIToken, error)
	// TouchLastUsedAt updates the last-used timestamp of the token without blocking the request path.
	TouchLastUsedAt(ctx context.Context, id uuid.UUID) error
}
