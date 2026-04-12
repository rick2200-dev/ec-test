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

	// CreateTenant creates a new tenant and provisions its schema.
	CreateTenant(ctx context.Context, t *domain.Tenant) error
	// GetTenant retrieves a tenant by its UUID.
	GetTenant(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	// ListTenants returns a paginated list of all tenants.
	ListTenants(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error)

	// Seller operations

	// CreateSeller registers a new seller within the tenant (initial status: pending).
	CreateSeller(ctx context.Context, tenantID uuid.UUID, seller *domain.Seller) error
	// GetSeller retrieves a seller by its UUID within the tenant.
	GetSeller(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error)
	// ListSellers returns a paginated list of sellers within the tenant.
	ListSellers(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error)
	// ApproveSeller transitions a seller's status from pending to active.
	ApproveSeller(ctx context.Context, tenantID, id uuid.UUID) error

	// Seller subscription / plan operations

	// CreatePlan creates a new seller subscription plan.
	CreatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error
	// ListPlans returns all seller subscription plans for the tenant.
	ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error)
	// GetPlan retrieves a seller subscription plan by its UUID.
	GetPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.SubscriptionPlan, error)
	// UpdatePlan persists changes to an existing seller subscription plan.
	UpdatePlan(ctx context.Context, tenantID uuid.UUID, plan *domain.SubscriptionPlan) error
	// GetSellerSubscription retrieves a seller's current subscription and plan details.
	GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	// SubscribeSeller assigns a subscription plan to a seller.
	SubscribeSeller(ctx context.Context, tenantID, sellerID, planID uuid.UUID) (*domain.SellerSubscription, error)

	// Buyer subscription / plan operations

	// CreateBuyerPlan creates a new buyer subscription plan.
	CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error
	// ListBuyerPlans returns all buyer subscription plans for the tenant.
	ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error)
	// GetBuyerPlan retrieves a buyer subscription plan by its UUID.
	GetBuyerPlan(ctx context.Context, tenantID, planID uuid.UUID) (*domain.BuyerPlan, error)
	// UpdateBuyerPlan persists changes to an existing buyer subscription plan.
	UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, plan *domain.BuyerPlan) error
	// GetBuyerSubscription retrieves a buyer's current subscription and plan details.
	GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	// SubscribeBuyer assigns a subscription plan to a buyer.
	SubscribeBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, planID uuid.UUID) (*domain.BuyerSubscription, error)

	// RBAC — seller team

	// LookupSellerRole returns the actor's role within the seller org,
	// or "" if the actor is not a member of the seller.
	LookupSellerRole(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error)
	// ListSellerTeam returns all team members of the given seller.
	ListSellerTeam(ctx context.Context, tenantID, sellerID uuid.UUID) ([]domain.SellerUser, error)
	// AddSellerUser adds a new member to the seller team with the specified role.
	AddSellerUser(ctx context.Context, tenantID, sellerID uuid.UUID, newAuth0UserID string, role domain.SellerUserRole) (*domain.SellerUser, error)
	// UpdateSellerUserRole changes the role of an existing seller team member.
	UpdateSellerUserRole(ctx context.Context, tenantID, sellerID, targetID uuid.UUID, newRole domain.SellerUserRole) error
	// RemoveSellerUser removes a member from the seller team.
	RemoveSellerUser(ctx context.Context, tenantID, sellerID, targetID uuid.UUID) error
	// TransferSellerOwnership transfers seller ownership to a new owner;
	// the current owner is downgraded to admin.
	TransferSellerOwnership(ctx context.Context, tenantID, sellerID, newOwnerID uuid.UUID) error

	// RBAC — platform admins

	// LookupPlatformAdminRole returns the actor's platform admin role,
	// or "" if the actor is not an admin.
	LookupPlatformAdminRole(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (domain.PlatformAdminRole, error)
	// ListPlatformAdmins returns all platform admins for the tenant.
	ListPlatformAdmins(ctx context.Context, tenantID uuid.UUID) ([]domain.PlatformAdmin, error)
	// GrantPlatformAdmin assigns a platform admin role to a user.
	GrantPlatformAdmin(ctx context.Context, tenantID uuid.UUID, newAuth0UserID string, role domain.PlatformAdminRole) (*domain.PlatformAdmin, error)
	// UpdatePlatformAdminRole changes the role of a platform admin.
	UpdatePlatformAdminRole(ctx context.Context, tenantID, targetID uuid.UUID, newRole domain.PlatformAdminRole) error
	// RevokePlatformAdmin removes platform admin status from a user.
	RevokePlatformAdmin(ctx context.Context, tenantID, targetID uuid.UUID) error
	// ListRBACAuditLog returns a paginated list of RBAC audit events for the tenant.
	ListRBACAuditLog(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error)
	// BootstrapSuperAdmin grants super-admin role to the given user;
	// intended for initial tenant setup only.
	BootstrapSuperAdmin(ctx context.Context, tenantID uuid.UUID, auth0UserID string) error

	// API tokens

	// IssueAPIToken generates a new seller API token with the given scopes and optional expiry.
	// Returns the token record and the plaintext secret — the secret is shown only once and not stored.
	IssueAPIToken(
		ctx context.Context,
		tenantID, sellerID uuid.UUID,
		name string,
		scopes []domain.APITokenScope,
		rateLimitRPS, rateLimitBurst *int,
		expiresAt *time.Time,
		tokenPrefix string,
	) (*domain.SellerAPIToken, string, error)
	// ListAPITokens returns a paginated list of API tokens for the seller.
	ListAPITokens(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error)
	// GetAPIToken retrieves an API token by its UUID, ensuring it belongs to the seller.
	GetAPIToken(ctx context.Context, tenantID, sellerID, id uuid.UUID) (*domain.SellerAPIToken, error)
	// RevokeAPIToken revokes an API token and returns the prefix and lookup hash for cache invalidation.
	RevokeAPIToken(ctx context.Context, tenantID, sellerID, id uuid.UUID) (prefix, lookup string, err error)
	// LookupAPIToken authenticates an incoming API request by validating the secret against the stored hash.
	LookupAPIToken(ctx context.Context, prefix, lookup, secret string) (*domain.SellerAPIToken, error)
}
