// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the auth service.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// TxRunner starts a transaction and embeds it in the context.
// Repository methods extract the transaction via database.TxFromContext so
// pgx.Tx never appears in service or port signatures.
// *database.PoolTxRunner satisfies this interface.
type TxRunner interface {
	// RunTx executes fn within a database transaction.
	// The transaction is embedded in the returned context so repository methods
	// can extract it via database.TxFromContext without leaking pgx.Tx into signatures.
	RunTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// SellerStore is the driven port for seller persistence.
// *repository.SellerRepository satisfies this interface.
type SellerStore interface {
	// GetByID retrieves a seller by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Seller, error)
	// GetBySlug retrieves a seller by its URL-friendly slug.
	GetBySlug(ctx context.Context, slug string) (*domain.Seller, error)
	// List returns a paginated list of sellers.
	List(ctx context.Context, limit, offset int) ([]domain.Seller, int, error)
	// UpdateStatus changes the approval status of a seller (e.g. "pending" → "active").
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SellerStatus) error
	// Create persists a new seller (initial status: pending).
	Create(ctx context.Context, s *domain.Seller) error
	// BatchGetByIDs returns sellers for the given ids. Unknown ids are
	// silently omitted. Used by the internal batch-get endpoint so
	// service-to-service callers (order) can snapshot seller_name without
	// reading auth_svc directly.
	BatchGetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Seller, error)
}

// SellerUserStore is the driven port for seller_user persistence.
// *repository.SellerUserRepository satisfies this interface.
type SellerUserStore interface {
	// GetByID retrieves a seller team member by their UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SellerUser, error)
	// GetByAuth0ID retrieves a seller team member by their Auth0 user ID within the seller.
	GetByAuth0ID(ctx context.Context, sellerID uuid.UUID, auth0UserID string) (*domain.SellerUser, error)
	// ListBySeller returns all team members for the given seller.
	ListBySeller(ctx context.Context, sellerID uuid.UUID) ([]domain.SellerUser, error)
	// Create adds a user to a seller team with the given role.
	Create(ctx context.Context, su *domain.SellerUser) error
	// UpdateRole changes the role of an existing seller team member.
	UpdateRole(ctx context.Context, id uuid.UUID, role domain.SellerUserRole) error
	// Delete removes a user from a seller team.
	Delete(ctx context.Context, id uuid.UUID) error
	// CountByRole returns the number of seller team members with the given role.
	CountByRole(ctx context.Context, sellerID uuid.UUID, role domain.SellerUserRole) (int, error)
	// CheckRole returns the actor's role within a seller org. Returns ("", nil)
	// when the user is not a member of the seller.
	CheckRole(ctx context.Context, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error)
}

// PlatformAdminStore is the driven port for platform_admin persistence.
// *repository.PlatformAdminRepository satisfies this interface.
type PlatformAdminStore interface {
	// GetByID retrieves a platform admin by their UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error)
	// GetByAuth0ID retrieves a platform admin by their Auth0 user ID.
	GetByAuth0ID(ctx context.Context, auth0UserID string) (*domain.PlatformAdmin, error)
	// List returns all platform admins.
	List(ctx context.Context) ([]domain.PlatformAdmin, error)
	// CountByRole returns the number of platform admins with the given role.
	CountByRole(ctx context.Context, role domain.PlatformAdminRole) (int, error)
	// Create grants platform admin status to a user.
	Create(ctx context.Context, pa *domain.PlatformAdmin) error
	// UpdateRole changes the role of a platform admin.
	UpdateRole(ctx context.Context, id uuid.UUID, role domain.PlatformAdminRole) error
	// Delete revokes platform admin status from a user.
	Delete(ctx context.Context, id uuid.UUID) error
	// CheckRole returns the actor's role as a platform admin.
	// Returns ("", nil) when the user is not an admin.
	CheckRole(ctx context.Context, auth0UserID string) (domain.PlatformAdminRole, error)
}

// RBACAuditStore is the driven port for rbac_audit_log persistence.
// *repository.RBACAuditRepository satisfies this interface.
type RBACAuditStore interface {
	// Append appends a new RBAC audit log entry.
	Append(ctx context.Context, e *domain.RBACAuditEntry) error
	// List returns a paginated list of RBAC audit entries.
	List(ctx context.Context, limit, offset int) ([]domain.RBACAuditEntry, int, error)
}

// APITokenStore is the driven port for seller API token persistence.
// *repository.APITokenRepository satisfies this interface.
type APITokenStore interface {
	// Create persists a new seller API token (stores the hashed secret, not the plaintext).
	Create(ctx context.Context, t *domain.SellerAPIToken) error
	// GetByID retrieves an API token record by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SellerAPIToken, error)
	// ListBySeller returns a paginated list of API tokens for the seller.
	ListBySeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error)
	// Revoke marks an API token as revoked; the token cannot be used after this call.
	Revoke(ctx context.Context, id uuid.UUID, actorAuth0UserID string) error
	// GetByLookup retrieves a token record by its prefix and lookup hash for authentication.
	GetByLookup(ctx context.Context, prefix, lookup string) (*domain.SellerAPIToken, error)
	// TouchLastUsedAt updates the last-used timestamp of the token without blocking the request path.
	TouchLastUsedAt(ctx context.Context, id uuid.UUID) error
}
