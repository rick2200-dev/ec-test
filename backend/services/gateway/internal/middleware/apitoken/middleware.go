package apitoken

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	pkgmw "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

// ErrInvalidTokenFormat is a package-level sentinel returned by ParseToken
// when the wire format is structurally invalid. It is never meant to flow
// outside this package — the caller translates it to a 401.
var ErrInvalidTokenFormat = errors.New("apitoken: invalid token format")

// ParseToken splits "<prefix><lookup>_<secret>" into (lookup, secret).
// Returns ErrInvalidTokenFormat on any structural issue. Duplicated from
// services/auth/internal/service.ParseAPIToken because that package is in
// a different module and importing it would pull the whole auth service
// internal into the gateway — this is a four-line function with zero
// drift risk (format is locked in the wire protocol).
func ParseToken(raw, prefix string) (lookup, secret string, err error) {
	if prefix == "" || !strings.HasPrefix(raw, prefix) {
		return "", "", ErrInvalidTokenFormat
	}
	rest := raw[len(prefix):]
	sep := strings.IndexByte(rest, '_')
	if sep <= 0 || sep == len(rest)-1 {
		return "", "", ErrInvalidTokenFormat
	}
	return rest[:sep], rest[sep+1:], nil
}

// OrJWT returns a middleware that decides, per request, between API token
// authentication and JWT. The prefix check is intentionally the first
// thing we look at — any Bearer token that does not start with the
// configured prefix (e.g. "sk_live_") falls through to JWT verification
// unchanged, so UI traffic and API-token traffic can share routes.
//
// Flow for a token with the matching prefix:
//  1. Parse the wire format. Malformed → 401.
//  2. Resolve via the loader. Non-active status → 401. Transport error → 503.
//  3. Inject tenant.Context (TenantID, UserID = issuer sub, SellerID, Roles
//     = ["seller"]) plus the apitoken.Context so downstream scope and
//     rate-limit middlewares can pick it up.
//  4. Log the request with the token id so every call is attributable.
//
// The synthetic "seller" role is what lets pkgmw.RequireRole("seller") on
// the /seller subtree accept this request — no other change to the
// existing role gate is needed.
func OrJWT(jwt *pkgmw.JWTMiddleware, loader *Loader, prefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Let the JWT middleware produce the standard 401.
				jwt.VerifyJWT(next).ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				jwt.VerifyJWT(next).ServeHTTP(w, r)
				return
			}
			raw := parts[1]

			// Not one of our API tokens → JWT path.
			if prefix == "" || !strings.HasPrefix(raw, prefix) {
				jwt.VerifyJWT(next).ServeHTTP(w, r)
				return
			}

			lookup, secret, err := ParseToken(raw, prefix)
			if err != nil {
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api token"})
				return
			}

			info, status, err := loader.Load(r.Context(), prefix, lookup, secret)
			if err != nil {
				// Transport-class failure. Don't surface details to the
				// caller, but log enough to diagnose server-side.
				slog.Error("api token lookup failed",
					"error", err,
					"prefix", prefix,
					"lookup", lookup,
					"path", r.URL.Path,
				)
				httputil.JSON(w, http.StatusServiceUnavailable, map[string]string{"error": "api token lookup unavailable"})
				return
			}
			if status != StatusActive {
				// Uniform 401 for revoked/expired/not_found/invalid so an
				// attacker cannot distinguish "this token used to work" from
				// "this token never existed". Still log the disposition so
				// operators can see patterns.
				slog.Info("api token rejected",
					"status", string(status),
					"prefix", prefix,
					"lookup", lookup,
					"path", r.URL.Path,
				)
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api token"})
				return
			}

			// Build a tenant.Context that downstream handlers can use the
			// same way they use JWT-derived contexts. The UserID field
			// carries the issuer's Auth0 sub — this is what audit logs in
			// auth_service_rbac_audit pick up as the actor on writes, so a
			// token-triggered update shows the human who issued the token.
			sellerID := info.SellerID
			tc := tenant.Context{
				TenantID: info.TenantID,
				SellerID: &sellerID,
				UserID:   info.IssuedByAuth0UserID,
				Roles:    []string{"seller"},
			}

			// Build the apitoken.Context that RequireScope and
			// APITokenRateLimit consume.
			scopeSet := make(map[string]struct{}, len(info.Scopes))
			for _, sc := range info.Scopes {
				scopeSet[sc] = struct{}{}
			}
			ac := Context{
				ID:             info.ID,
				TenantID:       info.TenantID,
				SellerID:       info.SellerID,
				Scopes:         scopeSet,
				RateLimitRPS:   info.RateLimitRPS,
				RateLimitBurst: info.RateLimitBurst,
				IssuedBy:       info.IssuedByAuth0UserID,
			}

			ctx := tenant.WithContext(r.Context(), tc)
			ctx = WithContext(ctx, ac)

			slog.Info("api token request",
				"token_id", info.ID,
				"seller_id", info.SellerID,
				"path", r.URL.Path,
				"method", r.Method,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
