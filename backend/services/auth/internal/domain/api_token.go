package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// APITokenScope is a resource×action permission string granted to a seller
// API access token. The set is closed; the same list is enforced by the
// CHECK constraint on auth_svc.seller_api_tokens.
type APITokenScope string

const (
	ScopeProductsRead   APITokenScope = "products:read"
	ScopeProductsWrite  APITokenScope = "products:write"
	ScopeOrdersRead     APITokenScope = "orders:read"
	ScopeOrdersWrite    APITokenScope = "orders:write"
	ScopeInventoryRead  APITokenScope = "inventory:read"
	ScopeInventoryWrite APITokenScope = "inventory:write"
)

// Valid reports whether the scope is one of the known values.
func (s APITokenScope) Valid() bool {
	switch s {
	case ScopeProductsRead, ScopeProductsWrite,
		ScopeOrdersRead, ScopeOrdersWrite,
		ScopeInventoryRead, ScopeInventoryWrite:
		return true
	}
	return false
}

// AllAPITokenScopes returns the full set of known scopes. Useful for UI
// listing and for "grant everything my role allows" helpers.
func AllAPITokenScopes() []APITokenScope {
	return []APITokenScope{
		ScopeProductsRead, ScopeProductsWrite,
		ScopeOrdersRead, ScopeOrdersWrite,
		ScopeInventoryRead, ScopeInventoryWrite,
	}
}

// ScopesForSellerRole returns the scopes a seller user with the given RBAC
// role is allowed to grant to a new API token. Today only owners can issue
// tokens, so admins/members return an empty slice — handlers additionally
// enforce the owner requirement via pkg/authz middleware, but this keeps
// the rule in a single source of truth.
func ScopesForSellerRole(role SellerUserRole) []APITokenScope {
	if role == SellerUserRoleOwner {
		return AllAPITokenScopes()
	}
	return nil
}

// SellerAPIToken is a persisted access token that grants scoped access to
// the /api/v1/seller/* surface. TokenHash is never serialized.
type SellerAPIToken struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	SellerID    uuid.UUID       `json:"seller_id"`
	Name        string          `json:"name"`
	TokenPrefix string          `json:"token_prefix"`
	TokenLookup string          `json:"token_lookup"`
	TokenHash   []byte          `json:"-"`
	Scopes      []APITokenScope `json:"scopes"`

	RateLimitRPS   *int `json:"rate_limit_rps,omitempty"`
	RateLimitBurst *int `json:"rate_limit_burst,omitempty"`

	IssuedByAuth0UserID  string     `json:"issued_by_auth0_user_id"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	RevokedAt            *time.Time `json:"revoked_at,omitempty"`
	RevokedByAuth0UserID string     `json:"revoked_by_auth0_user_id,omitempty"`
	LastUsedAt           *time.Time `json:"last_used_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// IsActive reports whether the token is neither revoked nor expired at the
// given instant. Use for status display and reject-if-inactive guards.
func (t *SellerAPIToken) IsActive(now time.Time) bool {
	if t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && !now.Before(*t.ExpiresAt) {
		return false
	}
	return true
}

// Status returns a short string label used for dashboard listing and logs.
func (t *SellerAPIToken) Status(now time.Time) string {
	switch {
	case t.RevokedAt != nil:
		return "revoked"
	case t.ExpiresAt != nil && !now.Before(*t.ExpiresAt):
		return "expired"
	default:
		return "active"
	}
}

// Sentinel errors returned by the api-token service layer. Handlers map
// these into HTTP status codes via mapAPITokenError (see service package).
var (
	ErrAPITokenNotFound        = errors.New("api token not found")
	ErrAPITokenRevoked         = errors.New("api token revoked")
	ErrAPITokenExpired         = errors.New("api token expired")
	ErrAPITokenInvalidFormat   = errors.New("api token malformed")
	ErrAPITokenScopeNotGranted = errors.New("scope not granted to issuer")
)
