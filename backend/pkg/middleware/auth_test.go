package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

// --- claimsToTenantContext (internal function) ---

func TestClaimsToTenantContext_FullClaims(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	claims := map[string]any{
		claimsNamespace + "/tenant_id": tid.String(),
		"sub":                          "auth0|user1",
		claimsNamespace + "/seller_id": sid.String(),
		claimsNamespace + "/roles":     []any{"seller", "admin"},
	}

	tc, err := claimsToTenantContext(claims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.TenantID != tid {
		t.Errorf("TenantID = %v, want %v", tc.TenantID, tid)
	}
	if tc.UserID != "auth0|user1" {
		t.Errorf("UserID = %q, want %q", tc.UserID, "auth0|user1")
	}
	if tc.SellerID == nil || *tc.SellerID != sid {
		t.Errorf("SellerID = %v, want %v", tc.SellerID, sid)
	}
	if len(tc.Roles) != 2 || tc.Roles[0] != "seller" || tc.Roles[1] != "admin" {
		t.Errorf("Roles = %v, want [seller admin]", tc.Roles)
	}
}

func TestClaimsToTenantContext_MinimalClaims(t *testing.T) {
	tid := uuid.New()
	claims := map[string]any{
		claimsNamespace + "/tenant_id": tid.String(),
		"sub":                          "auth0|buyer1",
	}

	tc, err := claimsToTenantContext(claims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.SellerID != nil {
		t.Errorf("SellerID = %v, want nil", tc.SellerID)
	}
	if tc.Roles != nil {
		t.Errorf("Roles = %v, want nil", tc.Roles)
	}
}

func TestClaimsToTenantContext_MissingTenantID(t *testing.T) {
	claims := map[string]any{
		"sub": "auth0|user1",
	}
	_, err := claimsToTenantContext(claims)
	if err == nil {
		t.Fatal("expected error for missing tenant_id")
	}
}

func TestClaimsToTenantContext_InvalidTenantID(t *testing.T) {
	claims := map[string]any{
		claimsNamespace + "/tenant_id": "not-a-uuid",
		"sub":                          "auth0|user1",
	}
	_, err := claimsToTenantContext(claims)
	if err == nil {
		t.Fatal("expected error for invalid tenant_id")
	}
}

func TestClaimsToTenantContext_MissingSub(t *testing.T) {
	claims := map[string]any{
		claimsNamespace + "/tenant_id": uuid.New().String(),
	}
	_, err := claimsToTenantContext(claims)
	if err == nil {
		t.Fatal("expected error for missing sub")
	}
}

func TestClaimsToTenantContext_InvalidSellerID(t *testing.T) {
	claims := map[string]any{
		claimsNamespace + "/tenant_id": uuid.New().String(),
		"sub":                          "auth0|user1",
		claimsNamespace + "/seller_id": "bad-uuid",
	}
	tc, err := claimsToTenantContext(claims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid seller_id is silently ignored.
	if tc.SellerID != nil {
		t.Errorf("SellerID = %v, want nil for invalid UUID", tc.SellerID)
	}
}

func TestClaimsToTenantContext_EmptySellerID(t *testing.T) {
	claims := map[string]any{
		claimsNamespace + "/tenant_id": uuid.New().String(),
		"sub":                          "auth0|user1",
		claimsNamespace + "/seller_id": "",
	}
	tc, err := claimsToTenantContext(claims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.SellerID != nil {
		t.Errorf("SellerID = %v, want nil for empty string", tc.SellerID)
	}
}

func TestClaimsToTenantContext_NonStringRoles(t *testing.T) {
	claims := map[string]any{
		claimsNamespace + "/tenant_id": uuid.New().String(),
		"sub":                          "auth0|user1",
		claimsNamespace + "/roles":     []any{"seller", 42, true},
	}
	tc, err := claimsToTenantContext(claims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only string roles should be extracted.
	if len(tc.Roles) != 1 || tc.Roles[0] != "seller" {
		t.Errorf("Roles = %v, want [seller]", tc.Roles)
	}
}

// --- RequireRole ---

func TestRequireRole_Allowed(t *testing.T) {
	tc := tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|u",
		Roles:    []string{"admin"},
	}
	ctx := tenant.WithContext(context.Background(), tc)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireRole("admin")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	tc := tenant.Context{
		TenantID: uuid.New(),
		UserID:   "auth0|u",
		Roles:    []string{"buyer"},
	}
	ctx := tenant.WithContext(context.Background(), tc)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := RequireRole("admin")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Error("inner handler should not have been called")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireRole_NoRolesInContext(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	handler := RequireRole("admin")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

// --- VerifyJWT (header validation paths — no real JWKS) ---

func TestVerifyJWT_HealthzBypass(t *testing.T) {
	m := NewJWTMiddleware(context.Background(), JWTConfig{})

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	for _, path := range []string{"/healthz", "/readyz"} {
		called = false
		handler := m.VerifyJWT(inner)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if !called {
			t.Errorf("%s: inner handler was not called", path)
		}
	}
}

func TestVerifyJWT_MissingAuthHeader(t *testing.T) {
	m := NewJWTMiddleware(context.Background(), JWTConfig{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	handler := m.VerifyJWT(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestVerifyJWT_InvalidAuthHeaderFormat(t *testing.T) {
	m := NewJWTMiddleware(context.Background(), JWTConfig{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	cases := []string{
		"Basic abc123",
		"Bearer",
		"justtoken",
	}
	for _, auth := range cases {
		handler := m.VerifyJWT(inner)
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", auth)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("auth=%q: expected %d, got %d", auth, http.StatusUnauthorized, rec.Code)
		}
	}
}

func TestVerifyJWT_NoJWKSURL(t *testing.T) {
	m := NewJWTMiddleware(context.Background(), JWTConfig{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	handler := m.VerifyJWT(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer some.fake.token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
