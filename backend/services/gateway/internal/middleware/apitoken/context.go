// Package apitoken holds all API-token-related middleware and helpers
// for the gateway. The package is intentionally self-contained — its
// only dependencies on the rest of the gateway are config, the proxy
// ServiceClient, and pkg/redis — so it can later be lifted out into a
// dedicated public-api-gateway binary without touching the UI BFF.
//
// The public surface is:
//
//   - APITokenOrJWT: the entry-point middleware. On Bearer headers with
//     the configured prefix it resolves the token, injects tenant +
//     APITokenContext, and lets the request through. Otherwise it
//     delegates to the JWT middleware.
//   - RequireScope: scope-level authorization. No-op for JWT callers
//     (whose authorization is handled by RBAC middleware); 403s for
//     API-token callers lacking the scope.
//   - BlockAPIToken: hard 403 for API-token callers on UI-only routes
//     (billing, team, token management) to prevent privilege escalation.
//   - APITokenRateLimit: per-token Redis sliding-window rate limiting.
//
// See plan §3.3 for the full extraction rationale.
package apitoken

import (
	"context"

	"github.com/google/uuid"
)

// ctxKey is the unexported context-key type used for API token state
// stored on request contexts by APITokenOrJWT.
type ctxKey string

const (
	apiTokenCtxKey ctxKey = "apitoken_ctx"
)

// Context is the per-request API token state. It is present only when
// the request was authenticated via an API token — JWT requests will
// have FromContext return ok=false, which is exactly what RequireScope
// and BlockAPIToken check to short-circuit.
//
// Scopes is a map rather than a slice so RequireScope can O(1) test
// membership on the hot path.
type Context struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	SellerID       uuid.UUID
	Scopes         map[string]struct{}
	RateLimitRPS   int // 0 = use gateway default
	RateLimitBurst int // 0 = use gateway default
	IssuedBy       string
}

// WithContext stores an API token Context into a Go context.
func WithContext(ctx context.Context, ac Context) context.Context {
	return context.WithValue(ctx, apiTokenCtxKey, ac)
}

// FromContext returns the API token Context if present. The bool is
// false for JWT-authenticated requests.
func FromContext(ctx context.Context) (Context, bool) {
	ac, ok := ctx.Value(apiTokenCtxKey).(Context)
	return ac, ok
}

// HasScope reports whether the token grants the given scope. Used by
// RequireScope.
func (c Context) HasScope(scope string) bool {
	_, ok := c.Scopes[scope]
	return ok
}
