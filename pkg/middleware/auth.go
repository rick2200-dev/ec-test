package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"

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
type JWTMiddleware struct {
	config  JWTConfig
	keyCache *jwk.Cache
}

// NewJWTMiddleware creates a new JWT middleware and initializes the JWKS key cache.
func NewJWTMiddleware(cfg JWTConfig) *JWTMiddleware {
	cache := jwk.NewCache(context.Background())
	if cfg.JWKSURL != "" {
		// Register the JWKS URL with a 15-minute refresh interval.
		if err := cache.Register(cfg.JWKSURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
			// Non-fatal: the cache will retry on first use.
			_ = err
		}
	}
	return &JWTMiddleware{config: cfg, keyCache: cache}
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

		claims, err := m.verifyAndExtractClaims(r.Context(), parts[1])
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

// verifyAndExtractClaims verifies the JWT signature using JWKS and returns the claims.
func (m *JWTMiddleware) verifyAndExtractClaims(ctx context.Context, rawToken string) (map[string]any, error) {
	if m.config.JWKSURL == "" {
		return nil, fmt.Errorf("JWKS URL not configured")
	}

	keySet, err := m.keyCache.Get(ctx, m.config.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}

	parseOpts := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	}
	if m.config.Issuer != "" {
		parseOpts = append(parseOpts, jwt.WithIssuer(m.config.Issuer))
	}
	if m.config.Audience != "" {
		parseOpts = append(parseOpts, jwt.WithAudience(m.config.Audience))
	}

	token, err := jwt.ParseString(rawToken, parseOpts...)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	// Convert jwt.Token to map[string]any for downstream processing.
	claims, err := token.AsMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("extract claims: %w", err)
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
