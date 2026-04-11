package apitoken

import (
	"net/http"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
)

// Block is middleware that hard-rejects any request coming in with an
// API token. Applied to UI-only subtrees (billing, plans, team, token
// management) to prevent a privilege-escalation class: a token must never
// be able to grant itself broader scopes, add new team members, or issue
// more tokens from itself.
//
// JWT requests have no apitoken.Context and fall through untouched.
func Block(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); ok {
			httputil.JSON(w, http.StatusForbidden, map[string]string{
				"error": "api tokens cannot access this endpoint; use the dashboard",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
