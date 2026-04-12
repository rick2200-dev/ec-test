package authz_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/authz"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

func TestSellerRole_Rank(t *testing.T) {
	cases := []struct {
		role authz.SellerRole
		want int
	}{
		{authz.SellerRoleOwner, 3},
		{authz.SellerRoleAdmin, 2},
		{authz.SellerRoleMember, 1},
		{authz.SellerRoleNone, 0},
		{"unknown", 0},
	}
	for _, c := range cases {
		if got := c.role.Rank(); got != c.want {
			t.Errorf("Rank(%q) = %d, want %d", c.role, got, c.want)
		}
	}
}

func TestSellerRole_AtLeast(t *testing.T) {
	if !authz.SellerRoleOwner.AtLeast(authz.SellerRoleMember) {
		t.Error("owner should >= member")
	}
	if authz.SellerRoleMember.AtLeast(authz.SellerRoleOwner) {
		t.Error("member should not >= owner")
	}
	if authz.SellerRoleNone.AtLeast(authz.SellerRoleMember) {
		t.Error("none should not satisfy any minimum role")
	}
}

func TestPlatformAdminRole_Rank(t *testing.T) {
	cases := []struct {
		role authz.PlatformAdminRole
		want int
	}{
		{authz.PlatformAdminRoleSuperAdmin, 3},
		{authz.PlatformAdminRoleAdmin, 2},
		{authz.PlatformAdminRoleSupport, 1},
		{authz.PlatformAdminRoleNone, 0},
	}
	for _, c := range cases {
		if got := c.role.Rank(); got != c.want {
			t.Errorf("Rank(%q) = %d, want %d", c.role, got, c.want)
		}
	}
}

func TestPlatformAdminRole_AtLeast(t *testing.T) {
	if !authz.PlatformAdminRoleSuperAdmin.AtLeast(authz.PlatformAdminRoleSupport) {
		t.Error("super_admin should >= support")
	}
	if authz.PlatformAdminRoleAdmin.AtLeast(authz.PlatformAdminRoleSuperAdmin) {
		t.Error("admin should not >= super_admin")
	}
	if authz.PlatformAdminRoleNone.AtLeast(authz.PlatformAdminRoleSupport) {
		t.Error("none should not satisfy any minimum")
	}
}

func TestTTLCache_GetSetDelete(t *testing.T) {
	c := authz.NewTTLCache(1*time.Second, 100)

	if _, err := c.Get("missing"); !errors.Is(err, authz.ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}

	c.Set("k", "v")
	v, err := c.Get("k")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if v != "v" {
		t.Errorf("Get returned %q, want %q", v, "v")
	}

	c.Delete("k")
	if _, err := c.Get("k"); !errors.Is(err, authz.ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after Delete, got %v", err)
	}
}

func TestTTLCache_Expiry(t *testing.T) {
	c := authz.NewTTLCache(20*time.Millisecond, 100)
	c.Set("k", "v")

	// Within TTL — should hit.
	if _, err := c.Get("k"); err != nil {
		t.Fatalf("expected hit before expiry, got %v", err)
	}

	time.Sleep(40 * time.Millisecond)

	// After TTL — should miss.
	if _, err := c.Get("k"); !errors.Is(err, authz.ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after expiry, got %v", err)
	}
}

func TestCacheKeys(t *testing.T) {
	tid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	sid := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	sub := "auth0|abc"

	if got := authz.SellerCacheKey(tid, sid, sub); got == "" {
		t.Error("SellerCacheKey returned empty string")
	}
	// Different subs must produce different keys.
	a := authz.SellerCacheKey(tid, sid, "auth0|a")
	b := authz.SellerCacheKey(tid, sid, "auth0|b")
	if a == b {
		t.Error("expected distinct keys for distinct subs")
	}

	if got := authz.PlatformAdminCacheKey(tid, sub); got == "" {
		t.Error("PlatformAdminCacheKey returned empty string")
	}
}

// ---------------------------------------------------------------------------
// TTLCache capacity / eviction
// ---------------------------------------------------------------------------

func TestTTLCache_CapacityEviction(t *testing.T) {
	const maxSize = 5
	c := authz.NewTTLCache(10*time.Second, maxSize)

	// Fill to capacity.
	for i := 0; i < maxSize; i++ {
		c.Set(fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i))
	}

	// Add a 6th entry — should trigger eviction.
	c.Set("k5", "v5")

	// Count how many of k0..k5 are still retrievable.
	hits := 0
	for i := 0; i <= maxSize; i++ {
		if _, err := c.Get(fmt.Sprintf("k%d", i)); err == nil {
			hits++
		}
	}
	if hits > maxSize {
		t.Errorf("expected at most %d entries after eviction, but found %d", maxSize, hits)
	}
}

func TestTTLCache_SweepExpired(t *testing.T) {
	const maxSize = 5
	c := authz.NewTTLCache(20*time.Millisecond, maxSize)

	// Fill with entries that will expire quickly.
	for i := 0; i < maxSize; i++ {
		c.Set(fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i))
	}

	// Wait for all entries to expire.
	time.Sleep(40 * time.Millisecond)

	// Add a new entry — sweep should clear all expired entries.
	c.Set("fresh", "value")

	// The fresh entry must be present.
	v, err := c.Get("fresh")
	if err != nil {
		t.Fatalf("expected fresh entry to be present, got %v", err)
	}
	if v != "value" {
		t.Errorf("fresh entry = %q, want %q", v, "value")
	}

	// All old entries should have been swept.
	for i := 0; i < maxSize; i++ {
		if _, err := c.Get(fmt.Sprintf("k%d", i)); !errors.Is(err, authz.ErrCacheMiss) {
			t.Errorf("expected ErrCacheMiss for expired k%d, got %v", i, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Mock loaders
// ---------------------------------------------------------------------------

type mockSellerRoleLoader struct {
	role authz.SellerRole
	err  error
}

func (m *mockSellerRoleLoader) LoadSellerRole(_ context.Context, _, _ uuid.UUID, _ string) (authz.SellerRole, error) {
	return m.role, m.err
}
func (m *mockSellerRoleLoader) EvictSellerRole(_, _ uuid.UUID, _ string) {}

type mockPlatformAdminRoleLoader struct {
	role authz.PlatformAdminRole
	err  error
}

func (m *mockPlatformAdminRoleLoader) LoadPlatformAdminRole(_ context.Context, _ uuid.UUID, _ string) (authz.PlatformAdminRole, error) {
	return m.role, m.err
}
func (m *mockPlatformAdminRoleLoader) EvictPlatformAdminRole(_ uuid.UUID, _ string) {}

// helper to build an *http.Request with tenant context injected.
func requestWithTenant(tc tenant.Context) *http.Request {
	ctx := tenant.WithContext(context.Background(), tc)
	return httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
}

// noopHandler records whether it was called.
type noopHandler struct{ called bool }

func (h *noopHandler) ServeHTTP(http.ResponseWriter, *http.Request) { h.called = true }

// ---------------------------------------------------------------------------
// RequireSellerRole middleware
// ---------------------------------------------------------------------------

func TestRequireSellerRole_NoTenantContext(t *testing.T) {
	loader := &mockSellerRoleLoader{role: authz.SellerRoleOwner}
	mw := authz.RequireSellerRole(loader, authz.SellerRoleMember)

	next := &noopHandler{}
	handler := mw(next)

	// Request with no tenant context at all.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if next.called {
		t.Error("next handler should not have been called")
	}
}

func TestRequireSellerRole_NoSellerID(t *testing.T) {
	loader := &mockSellerRoleLoader{role: authz.SellerRoleOwner}
	mw := authz.RequireSellerRole(loader, authz.SellerRoleMember)

	next := &noopHandler{}
	handler := mw(next)

	// Tenant context present but SellerID is nil (e.g. a buyer).
	req := requestWithTenant(tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|user1",
		SellerID: nil,
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if next.called {
		t.Error("next handler should not have been called")
	}
}

func TestRequireSellerRole_InsufficientRole(t *testing.T) {
	loader := &mockSellerRoleLoader{role: authz.SellerRoleMember}
	mw := authz.RequireSellerRole(loader, authz.SellerRoleAdmin) // require admin

	next := &noopHandler{}
	handler := mw(next)

	sid := uuid.New()
	req := requestWithTenant(tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|user1",
		SellerID: &sid,
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if next.called {
		t.Error("next handler should not have been called")
	}
}

func TestRequireSellerRole_SufficientRole(t *testing.T) {
	loader := &mockSellerRoleLoader{role: authz.SellerRoleAdmin}
	mw := authz.RequireSellerRole(loader, authz.SellerRoleMember)

	next := &noopHandler{}
	handler := mw(next)

	sid := uuid.New()
	req := requestWithTenant(tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|user1",
		SellerID: &sid,
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !next.called {
		t.Error("next handler should have been called")
	}
}

func TestRequireSellerRole_LoaderError(t *testing.T) {
	loader := &mockSellerRoleLoader{err: errors.New("db down")}
	mw := authz.RequireSellerRole(loader, authz.SellerRoleMember)

	next := &noopHandler{}
	handler := mw(next)

	sid := uuid.New()
	req := requestWithTenant(tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|user1",
		SellerID: &sid,
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if next.called {
		t.Error("next handler should not have been called")
	}
}

// ---------------------------------------------------------------------------
// RequirePlatformAdminRole middleware
// ---------------------------------------------------------------------------

func TestRequirePlatformAdminRole_NoTenantContext(t *testing.T) {
	loader := &mockPlatformAdminRoleLoader{role: authz.PlatformAdminRoleSuperAdmin}
	mw := authz.RequirePlatformAdminRole(loader, authz.PlatformAdminRoleSupport)

	next := &noopHandler{}
	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if next.called {
		t.Error("next handler should not have been called")
	}
}

func TestRequirePlatformAdminRole_InsufficientRole(t *testing.T) {
	loader := &mockPlatformAdminRoleLoader{role: authz.PlatformAdminRoleSupport}
	mw := authz.RequirePlatformAdminRole(loader, authz.PlatformAdminRoleAdmin) // require admin

	next := &noopHandler{}
	handler := mw(next)

	req := requestWithTenant(tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|user1",
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if next.called {
		t.Error("next handler should not have been called")
	}
}

func TestRequirePlatformAdminRole_SufficientRole(t *testing.T) {
	loader := &mockPlatformAdminRoleLoader{role: authz.PlatformAdminRoleSuperAdmin}
	mw := authz.RequirePlatformAdminRole(loader, authz.PlatformAdminRoleSupport)

	next := &noopHandler{}
	handler := mw(next)

	req := requestWithTenant(tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|user1",
	})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !next.called {
		t.Error("next handler should have been called")
	}
}

// ---------------------------------------------------------------------------
// CurrentSellerRole / CurrentPlatformAdminRole context helpers
// ---------------------------------------------------------------------------

func TestCurrentSellerRole_Set(t *testing.T) {
	// Simulate what the middleware does: set up tenant context, then run
	// the middleware so it stores the role in context, and verify inside
	// the next handler.
	loader := &mockSellerRoleLoader{role: authz.SellerRoleOwner}
	mw := authz.RequireSellerRole(loader, authz.SellerRoleMember)

	sid := uuid.New()
	req := requestWithTenant(tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|user1",
		SellerID: &sid,
	})

	var gotRole authz.SellerRole
	var gotOK bool
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotRole, gotOK = authz.CurrentSellerRole(r.Context())
	})

	rec := httptest.NewRecorder()
	mw(inner).ServeHTTP(rec, req)

	if !gotOK {
		t.Fatal("CurrentSellerRole returned ok=false, want true")
	}
	if gotRole != authz.SellerRoleOwner {
		t.Errorf("CurrentSellerRole = %q, want %q", gotRole, authz.SellerRoleOwner)
	}
}

func TestCurrentSellerRole_NotSet(t *testing.T) {
	_, ok := authz.CurrentSellerRole(context.Background())
	if ok {
		t.Error("CurrentSellerRole on empty context should return ok=false")
	}
}

func TestCurrentPlatformAdminRole_Set(t *testing.T) {
	loader := &mockPlatformAdminRoleLoader{role: authz.PlatformAdminRoleAdmin}
	mw := authz.RequirePlatformAdminRole(loader, authz.PlatformAdminRoleSupport)

	req := requestWithTenant(tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|user1",
	})

	var gotRole authz.PlatformAdminRole
	var gotOK bool
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotRole, gotOK = authz.CurrentPlatformAdminRole(r.Context())
	})

	rec := httptest.NewRecorder()
	mw(inner).ServeHTTP(rec, req)

	if !gotOK {
		t.Fatal("CurrentPlatformAdminRole returned ok=false, want true")
	}
	if gotRole != authz.PlatformAdminRoleAdmin {
		t.Errorf("CurrentPlatformAdminRole = %q, want %q", gotRole, authz.PlatformAdminRoleAdmin)
	}
}

func TestCurrentPlatformAdminRole_NotSet(t *testing.T) {
	_, ok := authz.CurrentPlatformAdminRole(context.Background())
	if ok {
		t.Error("CurrentPlatformAdminRole on empty context should return ok=false")
	}
}
