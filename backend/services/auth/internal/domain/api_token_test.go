package domain_test

import (
	"testing"
	"time"

	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

func TestAPITokenScope_Valid(t *testing.T) {
	cases := []struct {
		scope domain.APITokenScope
		want  bool
	}{
		{domain.ScopeProductsRead, true},
		{domain.ScopeProductsWrite, true},
		{domain.ScopeOrdersRead, true},
		{domain.ScopeOrdersWrite, true},
		{domain.ScopeInventoryRead, true},
		{domain.ScopeInventoryWrite, true},
		{"", false},
		{"products:admin", false},
		{"unknown", false},
	}
	for _, c := range cases {
		if got := c.scope.Valid(); got != c.want {
			t.Errorf("Valid(%q) = %v, want %v", c.scope, got, c.want)
		}
	}
}

func TestAllAPITokenScopes_MatchConstants(t *testing.T) {
	got := domain.AllAPITokenScopes()
	if len(got) != 6 {
		t.Fatalf("AllAPITokenScopes() returned %d scopes, want 6", len(got))
	}
	for _, s := range got {
		if !s.Valid() {
			t.Errorf("AllAPITokenScopes() contains invalid scope %q", s)
		}
	}
}

func TestScopesForSellerRole(t *testing.T) {
	cases := []struct {
		role     domain.SellerUserRole
		wantSize int
	}{
		{domain.SellerUserRoleOwner, 6},
		{domain.SellerUserRoleAdmin, 0},
		{domain.SellerUserRoleMember, 0},
		{"", 0},
		{"unknown", 0},
	}
	for _, c := range cases {
		if got := len(domain.ScopesForSellerRole(c.role)); got != c.wantSize {
			t.Errorf("ScopesForSellerRole(%q) returned %d scopes, want %d", c.role, got, c.wantSize)
		}
	}
}

func TestSellerAPIToken_IsActive(t *testing.T) {
	now := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name  string
		token domain.SellerAPIToken
		want  bool
	}{
		{
			name:  "active no expiry",
			token: domain.SellerAPIToken{},
			want:  true,
		},
		{
			name:  "active with future expiry",
			token: domain.SellerAPIToken{ExpiresAt: &future},
			want:  true,
		},
		{
			name:  "expired",
			token: domain.SellerAPIToken{ExpiresAt: &past},
			want:  false,
		},
		{
			name:  "expired exactly now",
			token: domain.SellerAPIToken{ExpiresAt: &now},
			want:  false,
		},
		{
			name:  "revoked",
			token: domain.SellerAPIToken{RevokedAt: &past},
			want:  false,
		},
		{
			name:  "revoked but not yet expired",
			token: domain.SellerAPIToken{RevokedAt: &past, ExpiresAt: &future},
			want:  false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.token.IsActive(now); got != c.want {
				t.Errorf("IsActive() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestSellerAPIToken_Status(t *testing.T) {
	now := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name  string
		token domain.SellerAPIToken
		want  string
	}{
		{"active", domain.SellerAPIToken{}, "active"},
		{"active future expiry", domain.SellerAPIToken{ExpiresAt: &future}, "active"},
		{"expired", domain.SellerAPIToken{ExpiresAt: &past}, "expired"},
		{"revoked", domain.SellerAPIToken{RevokedAt: &past}, "revoked"},
		// Revoked takes precedence over expired.
		{"revoked and expired", domain.SellerAPIToken{RevokedAt: &past, ExpiresAt: &past}, "revoked"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.token.Status(now); got != c.want {
				t.Errorf("Status() = %q, want %q", got, c.want)
			}
		})
	}
}
