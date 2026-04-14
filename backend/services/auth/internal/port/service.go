package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// The driving ports (inbound use cases) are split along domain boundaries so
// handlers depend only on the surface they actually need. A later extraction
// (e.g. moving subscription/credential out of this service) can change one
// interface without touching the others.
//
// *app.IdentityService, *app.RBACService, and *app.CredentialService each
// satisfy the corresponding interface here. *app.AuthService (a composite
// facade) satisfies all three, which is handy for main.go and tests that
// want a single wiring point.

// IdentityUseCase covers seller registration/approval.
// Buyer profile upsert lives on a separate local interface in the handler
// package because Auth0 is the source of truth for buyer identity and the
// repo only caches display fields.
type IdentityUseCase interface {
	// Seller operations

	// CreateSeller registers a new seller (initial status: pending).
	CreateSeller(ctx context.Context, seller *domain.Seller) error
	// GetSeller retrieves a seller by its UUID.
	GetSeller(ctx context.Context, id uuid.UUID) (*domain.Seller, error)
	// ListSellers returns a paginated list of sellers.
	ListSellers(ctx context.Context, limit, offset int) ([]domain.Seller, int, error)
	// ApproveSeller transitions a seller's status from pending to active.
	ApproveSeller(ctx context.Context, id uuid.UUID) error
}

// RBACUseCase covers seller-team and platform-admin role management plus the
// audit trail. The read-only role lookups are also here because the gateway's
// authorization middleware consults them on every request.
type RBACUseCase interface {
	// Seller team

	// LookupSellerRole returns the actor's role within the seller org,
	// or "" if the actor is not a member of the seller.
	LookupSellerRole(ctx context.Context, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error)
	// ListSellerTeam returns all team members of the given seller.
	ListSellerTeam(ctx context.Context, sellerID uuid.UUID) ([]domain.SellerUser, error)
	// AddSellerUser adds a new member to the seller team with the specified role.
	AddSellerUser(ctx context.Context, sellerID uuid.UUID, newAuth0UserID string, role domain.SellerUserRole) (*domain.SellerUser, error)
	// UpdateSellerUserRole changes the role of an existing seller team member.
	UpdateSellerUserRole(ctx context.Context, sellerID, targetID uuid.UUID, newRole domain.SellerUserRole) error
	// RemoveSellerUser removes a member from the seller team.
	RemoveSellerUser(ctx context.Context, sellerID, targetID uuid.UUID) error
	// TransferSellerOwnership transfers seller ownership to a new owner;
	// the current owner is downgraded to admin.
	TransferSellerOwnership(ctx context.Context, sellerID, newOwnerID uuid.UUID) error

	// Platform admins

	// LookupPlatformAdminRole returns the actor's platform admin role,
	// or "" if the actor is not an admin.
	LookupPlatformAdminRole(ctx context.Context, auth0UserID string) (domain.PlatformAdminRole, error)
	// ListPlatformAdmins returns all platform admins.
	ListPlatformAdmins(ctx context.Context) ([]domain.PlatformAdmin, error)
	// GrantPlatformAdmin assigns a platform admin role to a user.
	GrantPlatformAdmin(ctx context.Context, newAuth0UserID string, role domain.PlatformAdminRole) (*domain.PlatformAdmin, error)
	// UpdatePlatformAdminRole changes the role of a platform admin.
	UpdatePlatformAdminRole(ctx context.Context, targetID uuid.UUID, newRole domain.PlatformAdminRole) error
	// RevokePlatformAdmin removes platform admin status from a user.
	RevokePlatformAdmin(ctx context.Context, targetID uuid.UUID) error

	// Audit log

	// ListRBACAuditLog returns a paginated list of RBAC audit events.
	ListRBACAuditLog(ctx context.Context, limit, offset int) ([]domain.RBACAuditEntry, int, error)
}

// CredentialUseCase covers seller API access token lifecycle. The gateway
// calls LookupAPIToken on every uncached API-token request, so this
// interface is part of the request hot path.
type CredentialUseCase interface {
	// IssueAPIToken generates a new seller API token with the given scopes and optional expiry.
	// Returns the token record and the plaintext secret — the secret is shown only once and not stored.
	IssueAPIToken(
		ctx context.Context,
		sellerID uuid.UUID,
		name string,
		scopes []domain.APITokenScope,
		rateLimitRPS, rateLimitBurst *int,
		expiresAt *time.Time,
		tokenPrefix string,
	) (*domain.SellerAPIToken, string, error)
	// ListAPITokens returns a paginated list of API tokens for the seller.
	ListAPITokens(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error)
	// GetAPIToken retrieves an API token by its UUID, ensuring it belongs to the seller.
	GetAPIToken(ctx context.Context, sellerID, id uuid.UUID) (*domain.SellerAPIToken, error)
	// RevokeAPIToken revokes an API token and returns the prefix and lookup hash for cache invalidation.
	RevokeAPIToken(ctx context.Context, sellerID, id uuid.UUID) (prefix, lookup string, err error)
	// LookupAPIToken authenticates an incoming API request by validating the secret against the stored hash.
	LookupAPIToken(ctx context.Context, prefix, lookup, secret string) (*domain.SellerAPIToken, error)
}
