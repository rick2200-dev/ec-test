package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

// InternalContext is HTTP middleware that populates tenant.Context from
// forwarded headers (X-Tenant-ID, X-User-ID, X-Seller-ID, X-Roles).
// Use this on services that receive requests from the API gateway or
// other internal services rather than directly from end-users with JWTs.
//
// If the context already contains tenant information (e.g. set by
// VerifyJWT), this middleware is a no-op.
func InternalContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if tenant context is already present.
		if _, err := tenant.FromContext(r.Context()); err == nil {
			next.ServeHTTP(w, r)
			return
		}

		tenantIDStr := r.Header.Get("X-Tenant-ID")
		userID := r.Header.Get("X-User-ID")

		if tenantIDStr == "" {
			next.ServeHTTP(w, r)
			return
		}

		tid, err := uuid.Parse(tenantIDStr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		tc := tenant.Context{
			TenantID: tid,
			UserID:   userID,
		}

		if sellerIDStr := r.Header.Get("X-Seller-ID"); sellerIDStr != "" {
			if sid, err := uuid.Parse(sellerIDStr); err == nil {
				tc.SellerID = &sid
			}
		}

		if rolesStr := r.Header.Get("X-Roles"); rolesStr != "" {
			tc.Roles = strings.Split(rolesStr, ",")
		}

		ctx := tenant.WithContext(r.Context(), tc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
