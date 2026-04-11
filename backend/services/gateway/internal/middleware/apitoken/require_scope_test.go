package apitoken

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// RequireScope is deliberately inert for JWT callers (no apitoken.Context)
// and enforces scope set membership for API-token callers. Both branches
// need direct coverage because a future editor could easily break one.

func TestRequireScope_NoContextPassesThrough(t *testing.T) {
	var reached bool
	h := RequireScope("products:read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rec, req)
	if !reached {
		t.Error("handler should have been reached (no API token context → pass)")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestRequireScope_WithScopeAllows(t *testing.T) {
	ac := Context{
		ID: uuid.New(),
		Scopes: map[string]struct{}{
			"products:read":  {},
			"products:write": {},
		},
	}
	var reached bool
	h := RequireScope("products:read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil).WithContext(WithContext(httptest.NewRequest("GET", "/", nil).Context(), ac))
	h.ServeHTTP(rec, req)
	if !reached {
		t.Error("handler should have been reached (scope present)")
	}
}

func TestRequireScope_MissingScopeForbids(t *testing.T) {
	ac := Context{
		ID: uuid.New(),
		Scopes: map[string]struct{}{
			"products:read": {},
		},
	}
	h := RequireScope("orders:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached")
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil).WithContext(WithContext(httptest.NewRequest("GET", "/", nil).Context(), ac))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestBlock_WithTokenForbids(t *testing.T) {
	ac := Context{ID: uuid.New()}
	h := Block(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached for API-token requests")
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil).WithContext(WithContext(httptest.NewRequest("GET", "/", nil).Context(), ac))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestBlock_WithoutTokenPassesThrough(t *testing.T) {
	var reached bool
	h := Block(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rec, req)
	if !reached {
		t.Error("JWT requests should pass through Block")
	}
}
