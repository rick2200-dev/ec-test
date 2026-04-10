package domain_test

import (
	"testing"

	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

func TestSellerUserRole_Rank(t *testing.T) {
	cases := []struct {
		role domain.SellerUserRole
		want int
	}{
		{domain.SellerUserRoleOwner, 3},
		{domain.SellerUserRoleAdmin, 2},
		{domain.SellerUserRoleMember, 1},
		{"", 0},
		{"unknown", 0},
	}
	for _, c := range cases {
		if got := c.role.Rank(); got != c.want {
			t.Errorf("Rank(%q) = %d, want %d", c.role, got, c.want)
		}
	}
}

func TestSellerUserRole_AtLeast(t *testing.T) {
	cases := []struct {
		role domain.SellerUserRole
		min  domain.SellerUserRole
		want bool
	}{
		{domain.SellerUserRoleOwner, domain.SellerUserRoleMember, true},
		{domain.SellerUserRoleOwner, domain.SellerUserRoleOwner, true},
		{domain.SellerUserRoleAdmin, domain.SellerUserRoleOwner, false},
		{domain.SellerUserRoleAdmin, domain.SellerUserRoleAdmin, true},
		{domain.SellerUserRoleMember, domain.SellerUserRoleAdmin, false},
		{"", domain.SellerUserRoleMember, false},
		{"unknown", domain.SellerUserRoleMember, false},
	}
	for _, c := range cases {
		if got := c.role.AtLeast(c.min); got != c.want {
			t.Errorf("%q.AtLeast(%q) = %v, want %v", c.role, c.min, got, c.want)
		}
	}
}

func TestSellerUserRole_Valid(t *testing.T) {
	if !domain.SellerUserRoleOwner.Valid() {
		t.Error("owner should be valid")
	}
	if !domain.SellerUserRoleMember.Valid() {
		t.Error("member should be valid")
	}
	if domain.SellerUserRole("").Valid() {
		t.Error("empty should be invalid")
	}
	if domain.SellerUserRole("god").Valid() {
		t.Error("unknown role should be invalid")
	}
}

func TestPlatformAdminRole_Rank(t *testing.T) {
	cases := []struct {
		role domain.PlatformAdminRole
		want int
	}{
		{domain.PlatformAdminRoleSuperAdmin, 3},
		{domain.PlatformAdminRoleAdmin, 2},
		{domain.PlatformAdminRoleSupport, 1},
		{"", 0},
		{"unknown", 0},
	}
	for _, c := range cases {
		if got := c.role.Rank(); got != c.want {
			t.Errorf("Rank(%q) = %d, want %d", c.role, got, c.want)
		}
	}
}

func TestPlatformAdminRole_AtLeast(t *testing.T) {
	if !domain.PlatformAdminRoleSuperAdmin.AtLeast(domain.PlatformAdminRoleSupport) {
		t.Error("super_admin should satisfy >= support")
	}
	if domain.PlatformAdminRoleSupport.AtLeast(domain.PlatformAdminRoleAdmin) {
		t.Error("support should not satisfy >= admin")
	}
	if !domain.PlatformAdminRoleAdmin.AtLeast(domain.PlatformAdminRoleAdmin) {
		t.Error("admin should satisfy >= admin")
	}
}

func TestPlatformAdminRole_Valid(t *testing.T) {
	if !domain.PlatformAdminRoleSuperAdmin.Valid() {
		t.Error("super_admin should be valid")
	}
	if domain.PlatformAdminRole("").Valid() {
		t.Error("empty should be invalid")
	}
	if domain.PlatformAdminRole("ceo").Valid() {
		t.Error("unknown role should be invalid")
	}
}
