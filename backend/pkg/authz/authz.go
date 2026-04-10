// Package authz provides fine-grained role-based authorization that
// complements the coarse JWT role check in pkg/middleware. Roles are loaded
// from the auth service via a Loader interface (so the gateway can implement
// it as an HTTP call to /internal/authz/*).
//
// This package lives in the shared pkg module because it must be importable
// by the gateway. It therefore declares its own role types rather than
// depending on services/auth/internal/domain (which is a private internal
// package of a different module). The wire format on the auth service side
// is plain strings, so the two type families stay in sync via constants.
package authz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/httputil"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

// SellerRole is a seller-organization-scoped role. Mirrors
// services/auth/internal/domain.SellerUserRole on the wire.
type SellerRole string

const (
	SellerRoleNone   SellerRole = ""
	SellerRoleMember SellerRole = "member"
	SellerRoleAdmin  SellerRole = "admin"
	SellerRoleOwner  SellerRole = "owner"
)

// Rank returns a numeric rank for comparison. Higher = more privileged.
// SellerRoleNone has rank 0 so it never satisfies AtLeast.
func (r SellerRole) Rank() int {
	switch r {
	case SellerRoleOwner:
		return 3
	case SellerRoleAdmin:
		return 2
	case SellerRoleMember:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether r is at least as privileged as min.
func (r SellerRole) AtLeast(min SellerRole) bool { return r.Rank() >= min.Rank() }

// PlatformAdminRole is a tenant-scoped platform admin role. Mirrors
// services/auth/internal/domain.PlatformAdminRole on the wire.
type PlatformAdminRole string

const (
	PlatformAdminRoleNone       PlatformAdminRole = ""
	PlatformAdminRoleSupport    PlatformAdminRole = "support"
	PlatformAdminRoleAdmin      PlatformAdminRole = "admin"
	PlatformAdminRoleSuperAdmin PlatformAdminRole = "super_admin"
)

// Rank returns a numeric rank for comparison. Higher = more privileged.
func (r PlatformAdminRole) Rank() int {
	switch r {
	case PlatformAdminRoleSuperAdmin:
		return 3
	case PlatformAdminRoleAdmin:
		return 2
	case PlatformAdminRoleSupport:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether r is at least as privileged as min.
func (r PlatformAdminRole) AtLeast(min PlatformAdminRole) bool { return r.Rank() >= min.Rank() }

// SellerRoleLoader resolves a user's role within a seller organization.
// Implementations should be safe for concurrent use.
type SellerRoleLoader interface {
	LoadSellerRole(ctx context.Context, tenantID, sellerID uuid.UUID, sub string) (SellerRole, error)
	// EvictSellerRole drops any cached value for (tenantID, sellerID, sub).
	// Mutation handlers should call this after modifying seller team
	// membership so the change is visible immediately within this process.
	EvictSellerRole(tenantID, sellerID uuid.UUID, sub string)
}

// PlatformAdminRoleLoader resolves a user's platform admin role for a tenant.
type PlatformAdminRoleLoader interface {
	LoadPlatformAdminRole(ctx context.Context, tenantID uuid.UUID, sub string) (PlatformAdminRole, error)
	EvictPlatformAdminRole(tenantID uuid.UUID, sub string)
}

// Per-request context keys. The middleware stores resolved roles in the
// request context so downstream handlers can read them without re-querying.
type ctxKey string

const (
	ctxSellerRole        ctxKey = "authz_seller_role"
	ctxPlatformAdminRole ctxKey = "authz_platform_admin_role"
)

// CurrentSellerRole returns the seller role resolved by RequireSellerRole
// for the current request, if any.
func CurrentSellerRole(ctx context.Context) (SellerRole, bool) {
	r, ok := ctx.Value(ctxSellerRole).(SellerRole)
	return r, ok
}

// CurrentPlatformAdminRole returns the platform admin role resolved by
// RequirePlatformAdminRole for the current request, if any.
func CurrentPlatformAdminRole(ctx context.Context) (PlatformAdminRole, bool) {
	r, ok := ctx.Value(ctxPlatformAdminRole).(PlatformAdminRole)
	return r, ok
}

// RequireSellerRole returns middleware that loads the caller's seller role
// from loader and rejects the request unless it is at least min. The seller
// id is taken from tenant.Context.SellerID — callers without one get 401.
func RequireSellerRole(loader SellerRoleLoader, min SellerRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tc, err := tenant.FromContext(r.Context())
			if err != nil || tc.UserID == "" || tc.SellerID == nil {
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "seller context required"})
				return
			}
			role, err := loader.LoadSellerRole(r.Context(), tc.TenantID, *tc.SellerID, tc.UserID)
			if err != nil {
				httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "role lookup failed"})
				return
			}
			if !role.AtLeast(min) {
				httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "insufficient seller role"})
				return
			}
			ctx := context.WithValue(r.Context(), ctxSellerRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePlatformAdminRole returns middleware that loads the caller's
// platform admin role and rejects the request unless it is at least min.
func RequirePlatformAdminRole(loader PlatformAdminRoleLoader, min PlatformAdminRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tc, err := tenant.FromContext(r.Context())
			if err != nil || tc.UserID == "" {
				httputil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "caller identity required"})
				return
			}
			role, err := loader.LoadPlatformAdminRole(r.Context(), tc.TenantID, tc.UserID)
			if err != nil {
				httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "role lookup failed"})
				return
			}
			if !role.AtLeast(min) {
				httputil.JSON(w, http.StatusForbidden, map[string]string{"error": "insufficient admin role"})
				return
			}
			ctx := context.WithValue(r.Context(), ctxPlatformAdminRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ----- TTL cache used by Loader implementations -----

// ErrCacheMiss is returned by TTLCache.Get when no entry exists or it has
// expired. Loader implementations use this to decide whether to fetch.
var ErrCacheMiss = errors.New("authz: cache miss")

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

// TTLCache is a tiny string-keyed TTL cache designed for role lookups. It
// is safe for concurrent use and is intentionally dependency-free (no LRU
// library) — eviction happens lazily on read and via explicit Delete.
type TTLCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	maxSize int
}

// NewTTLCache creates a TTLCache with the given TTL and max size. When the
// cache hits maxSize, the next write triggers a sweep that drops expired
// entries; if the cache is still full afterward, the oldest-keyed entries
// (by expiry) are removed until it is below maxSize. This is not a true LRU
// but it is sufficient for short-lived role caches with small key cardinality.
func NewTTLCache(ttl time.Duration, maxSize int) *TTLCache {
	return &TTLCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get returns the cached value or ErrCacheMiss.
func (c *TTLCache) Get(key string) (string, error) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return "", ErrCacheMiss
	}
	if time.Now().After(e.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return "", ErrCacheMiss
	}
	return e.value, nil
}

// Set stores value under key with the configured TTL.
func (c *TTLCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		c.sweepLocked()
		if len(c.entries) >= c.maxSize {
			// still full — drop a few entries with the soonest expiry
			c.evictOldestLocked(c.maxSize / 10)
		}
	}
	c.entries[key] = cacheEntry{value: value, expiresAt: time.Now().Add(c.ttl)}
}

// Delete removes a single key.
func (c *TTLCache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

func (c *TTLCache) sweepLocked() {
	now := time.Now()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
}

func (c *TTLCache) evictOldestLocked(n int) {
	if n <= 0 {
		n = 1
	}
	type kv struct {
		k string
		t time.Time
	}
	oldest := make([]kv, 0, n)
	for k, e := range c.entries {
		if len(oldest) < n {
			oldest = append(oldest, kv{k, e.expiresAt})
			continue
		}
		// find the largest in oldest and replace if e is older
		maxIdx := 0
		for i := 1; i < len(oldest); i++ {
			if oldest[i].t.After(oldest[maxIdx].t) {
				maxIdx = i
			}
		}
		if e.expiresAt.Before(oldest[maxIdx].t) {
			oldest[maxIdx] = kv{k, e.expiresAt}
		}
	}
	for _, kv := range oldest {
		delete(c.entries, kv.k)
	}
}

// SellerCacheKey returns the canonical cache key for a seller role lookup.
func SellerCacheKey(tenantID, sellerID uuid.UUID, sub string) string {
	return fmt.Sprintf("seller:%s:%s:%s", tenantID, sellerID, sub)
}

// PlatformAdminCacheKey returns the canonical cache key for a platform
// admin role lookup.
func PlatformAdminCacheKey(tenantID uuid.UUID, sub string) string {
	return fmt.Sprintf("padmin:%s:%s", tenantID, sub)
}
