package apitoken

import (
	"net/http"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

// RequireScope returns middleware that enforces a fine-grained scope on
// API-token requests. It is a no-op for JWT-authenticated callers: the
// UI path is already gated by the seller-role RBAC middleware, so adding
// a second check here would both duplicate logic and force us to invent a
// JWT → scope mapping that does not exist in the role model.
//
// Behaviour matrix:
//
//	API token + scope present     → pass through
//	API token + scope missing     → 403 "scope not granted"
//	JWT (no apitoken.Context)     → pass through
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac, ok := FromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if !ac.HasScope(scope) {
				httputil.JSON(w, http.StatusForbidden, map[string]string{
					"error": "api token is missing required scope: " + scope,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
