package authz_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/authz"
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
