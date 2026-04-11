package middleware

import (
	"crypto/subtle"
	"net/http"
)

// RequireInternalToken returns HTTP middleware that rejects any request whose
// X-Internal-Token header does not match the configured shared secret. It is
// intended to gate `/internal/*` routes on services that would otherwise rely
// solely on cluster-internal network isolation.
//
// If `secret` is empty the middleware fails closed with 503 Service
// Unavailable so a misconfigured deployment cannot accidentally expose the
// protected routes. Callers should fail fast at startup if they require this
// middleware to be functional.
//
// The comparison uses crypto/subtle to avoid leaking the secret via a timing
// side channel.
func RequireInternalToken(secret string) func(http.Handler) http.Handler {
	secretBytes := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(secretBytes) == 0 {
				http.Error(w, `{"error":"internal endpoint not configured"}`, http.StatusServiceUnavailable)
				return
			}
			got := []byte(r.Header.Get("X-Internal-Token"))
			if subtle.ConstantTimeEq(int32(len(got)), int32(len(secretBytes))) != 1 ||
				subtle.ConstantTimeCompare(got, secretBytes) != 1 {
				http.Error(w, `{"error":"invalid internal token"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
