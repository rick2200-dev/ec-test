package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

func TestInternalContext_PopulatesFromHeaders(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()

	var gotTC tenant.Context
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, err := tenant.FromContext(r.Context())
		if err != nil {
			t.Fatalf("FromContext: %v", err)
		}
		gotTC = tc
	})

	handler := middleware.InternalContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", tid.String())
	req.Header.Set("X-User-ID", "auth0|user1")
	req.Header.Set("X-Seller-ID", sid.String())
	req.Header.Set("X-Roles", "seller,admin")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if gotTC.TenantID != tid {
		t.Errorf("TenantID = %v, want %v", gotTC.TenantID, tid)
	}
	if gotTC.UserID != "auth0|user1" {
		t.Errorf("UserID = %q, want %q", gotTC.UserID, "auth0|user1")
	}
	if gotTC.SellerID == nil || *gotTC.SellerID != sid {
		t.Errorf("SellerID = %v, want %v", gotTC.SellerID, sid)
	}
	if len(gotTC.Roles) != 2 || gotTC.Roles[0] != "seller" || gotTC.Roles[1] != "admin" {
		t.Errorf("Roles = %v, want [seller admin]", gotTC.Roles)
	}
}

func TestInternalContext_MinimalHeaders(t *testing.T) {
	tid := uuid.New()

	var gotTC tenant.Context
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, err := tenant.FromContext(r.Context())
		if err != nil {
			t.Fatalf("FromContext: %v", err)
		}
		gotTC = tc
	})

	handler := middleware.InternalContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", tid.String())
	req.Header.Set("X-User-ID", "auth0|buyer1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if gotTC.TenantID != tid {
		t.Errorf("TenantID = %v, want %v", gotTC.TenantID, tid)
	}
	if gotTC.SellerID != nil {
		t.Errorf("SellerID = %v, want nil", gotTC.SellerID)
	}
}

func TestInternalContext_NoTenantIDHeader(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Should not have tenant context.
		if _, err := tenant.FromContext(r.Context()); err == nil {
			t.Error("expected no tenant context")
		}
	})

	handler := middleware.InternalContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}
}

func TestInternalContext_InvalidTenantID(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, err := tenant.FromContext(r.Context()); err == nil {
			t.Error("expected no tenant context for invalid UUID")
		}
	})

	handler := middleware.InternalContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}
}

func TestInternalContext_InvalidSellerID(t *testing.T) {
	tid := uuid.New()
	var gotTC tenant.Context
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, err := tenant.FromContext(r.Context())
		if err != nil {
			t.Fatalf("FromContext: %v", err)
		}
		gotTC = tc
	})

	handler := middleware.InternalContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", tid.String())
	req.Header.Set("X-User-ID", "auth0|u")
	req.Header.Set("X-Seller-ID", "bad-uuid")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Invalid seller_id should be silently ignored.
	if gotTC.SellerID != nil {
		t.Errorf("SellerID = %v, want nil", gotTC.SellerID)
	}
}

func TestInternalContext_SkipsWhenTenantContextExists(t *testing.T) {
	existingTID := uuid.New()
	headerTID := uuid.New()

	tc := tenant.Context{
		TenantID: existingTID,
		UserID:   "auth0|existing",
	}

	var gotTC tenant.Context
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := tenant.FromContext(r.Context())
		if err != nil {
			t.Fatalf("FromContext: %v", err)
		}
		gotTC = got
	})

	handler := middleware.InternalContext(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(tenant.WithContext(req.Context(), tc))
	req.Header.Set("X-Tenant-ID", headerTID.String())
	req.Header.Set("X-User-ID", "auth0|new")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should keep the existing context, not the header values.
	if gotTC.TenantID != existingTID {
		t.Errorf("TenantID = %v, want existing %v", gotTC.TenantID, existingTID)
	}
	if gotTC.UserID != "auth0|existing" {
		t.Errorf("UserID = %q, want %q", gotTC.UserID, "auth0|existing")
	}
}
