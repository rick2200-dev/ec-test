package middleware

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

const (
	claimsNamespace = "https://ecmarket.example.com"
)

// JWTConfig holds configuration for JWT validation.
type JWTConfig struct {
	Issuer   string // Auth0 issuer URL
	Audience string // API audience identifier
	JWKSURL  string // JWKS endpoint
}

// JWTMiddleware validates Auth0 JWT tokens and extracts tenant context.
// TODO: Replace payload-only decode with proper JWKS signature verification for production.
type JWTMiddleware struct {
	config JWTConfig
}

// NewJWTMiddleware creates a new JWT middleware.
func NewJWTMiddleware(cfg JWTConfig) *JWTMiddleware {
	return &JWTMiddleware{config: cfg}
}

// VerifyJWT is the HTTP middleware that validates the JWT and injects tenant context.
func (m *JWTMiddleware) VerifyJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		claims, err := extractClaims(parts[1])
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		tc, err := claimsToTenantContext(claims)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		ctx := tenant.WithContext(r.Context(), tc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns middleware that checks if the user has a specific role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !tenant.HasRole(r.Context(), role) {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractClaims decodes the JWT payload without signature verification.
func extractClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	return claims, nil
}

func claimsToTenantContext(claims map[string]any) (tenant.Context, error) {
	tc := tenant.Context{}

	tenantIDStr, ok := claims[claimsNamespace+"/tenant_id"].(string)
	if !ok {
		return tc, fmt.Errorf("missing tenant_id claim")
	}
	tid, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return tc, fmt.Errorf("invalid tenant_id: %w", err)
	}
	tc.TenantID = tid

	sub, ok := claims["sub"].(string)
	if !ok {
		return tc, fmt.Errorf("missing sub claim")
	}
	tc.UserID = sub

	if sellerIDStr, ok := claims[claimsNamespace+"/seller_id"].(string); ok && sellerIDStr != "" {
		sid, err := uuid.Parse(sellerIDStr)
		if err == nil {
			tc.SellerID = &sid
		}
	}

	if rolesRaw, ok := claims[claimsNamespace+"/roles"].([]any); ok {
		for _, r := range rolesRaw {
			if role, ok := r.(string); ok {
				tc.Roles = append(tc.Roles, role)
			}
		}
	}

	return tc, nil
}
