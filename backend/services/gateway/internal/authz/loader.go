// Package authz provides gateway-side implementations of the loader
// interfaces declared in pkg/authz. The loaders call the auth service's
// /internal/authz/* endpoints (protected by a shared secret) and cache
// results in a short-TTL in-memory cache.
package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"

	pkgauthz "github.com/Riku-KANO/ec-test/pkg/authz"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

const (
	defaultCacheTTL  = 30 * time.Second
	defaultCacheSize = 10000
)

// roleResponse matches the JSON shape returned by auth service
// /internal/authz/seller-role and /internal/authz/platform-admin-role.
type roleResponse struct {
	Role string `json:"role"`
}

// Loader implements both pkg/authz.SellerRoleLoader and
// pkg/authz.PlatformAdminRoleLoader by calling the auth service.
type Loader struct {
	client *proxy.ServiceClient // already configured with X-Internal-Token
	cache  *pkgauthz.TTLCache
}

// NewLoader builds a Loader. The provided client should already be wrapped
// with the X-Internal-Token header (use proxy.ServiceClient.WithHeader).
func NewLoader(client *proxy.ServiceClient) *Loader {
	return &Loader{
		client: client,
		cache:  pkgauthz.NewTTLCache(defaultCacheTTL, defaultCacheSize),
	}
}

// LoadSellerRole implements pkgauthz.SellerRoleLoader.
func (l *Loader) LoadSellerRole(ctx context.Context, tenantID, sellerID uuid.UUID, sub string) (pkgauthz.SellerRole, error) {
	key := pkgauthz.SellerCacheKey(tenantID, sellerID, sub)
	if v, err := l.cache.Get(key); err == nil {
		return pkgauthz.SellerRole(v), nil
	}

	q := url.Values{}
	q.Set("seller_id", sellerID.String())
	q.Set("sub", sub)
	body, status, err := l.client.Get(ctx, "/internal/authz/seller-role", q.Encode())
	if err != nil {
		return pkgauthz.SellerRoleNone, fmt.Errorf("authz: seller role lookup: %w", err)
	}
	if status != 200 {
		return pkgauthz.SellerRoleNone, fmt.Errorf("authz: seller role lookup status %d: %s", status, string(body))
	}
	var resp roleResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return pkgauthz.SellerRoleNone, fmt.Errorf("authz: decode seller role: %w", err)
	}
	role := pkgauthz.SellerRole(resp.Role)
	l.cache.Set(key, string(role))
	return role, nil
}

// EvictSellerRole implements pkgauthz.SellerRoleLoader.
func (l *Loader) EvictSellerRole(tenantID, sellerID uuid.UUID, sub string) {
	l.cache.Delete(pkgauthz.SellerCacheKey(tenantID, sellerID, sub))
}

// LoadPlatformAdminRole implements pkgauthz.PlatformAdminRoleLoader.
func (l *Loader) LoadPlatformAdminRole(ctx context.Context, tenantID uuid.UUID, sub string) (pkgauthz.PlatformAdminRole, error) {
	key := pkgauthz.PlatformAdminCacheKey(tenantID, sub)
	if v, err := l.cache.Get(key); err == nil {
		return pkgauthz.PlatformAdminRole(v), nil
	}

	q := url.Values{}
	q.Set("sub", sub)
	body, status, err := l.client.Get(ctx, "/internal/authz/platform-admin-role", q.Encode())
	if err != nil {
		return pkgauthz.PlatformAdminRoleNone, fmt.Errorf("authz: platform admin role lookup: %w", err)
	}
	if status != 200 {
		return pkgauthz.PlatformAdminRoleNone, fmt.Errorf("authz: platform admin role lookup status %d: %s", status, string(body))
	}
	var resp roleResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return pkgauthz.PlatformAdminRoleNone, fmt.Errorf("authz: decode platform admin role: %w", err)
	}
	role := pkgauthz.PlatformAdminRole(resp.Role)
	l.cache.Set(key, string(role))
	return role, nil
}

// EvictPlatformAdminRole implements pkgauthz.PlatformAdminRoleLoader.
func (l *Loader) EvictPlatformAdminRole(tenantID uuid.UUID, sub string) {
	l.cache.Delete(pkgauthz.PlatformAdminCacheKey(tenantID, sub))
}
