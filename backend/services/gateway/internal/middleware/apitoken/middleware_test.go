package apitoken

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	pkgmw "github.com/Riku-KANO/ec-test/pkg/middleware"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// Why these tests exist: OrJWT is the critical hand-off between API token
// and JWT paths. A bug here either silently accepts traffic that should
// be rejected or routes UI traffic through the token decoder. The tests
// below cover the four decision branches that matter: no auth, wrong
// scheme, non-matching prefix (→ JWT), matching prefix (→ token loader).

func TestParseToken(t *testing.T) {
	cases := []struct {
		name       string
		raw        string
		prefix     string
		wantLookup string
		wantSecret string
		wantErr    bool
	}{
		{"happy path", "sk_live_abc_secret", "sk_live_", "abc", "secret", false},
		{"empty prefix", "sk_live_abc_secret", "", "", "", true},
		{"wrong prefix", "sk_test_abc_secret", "sk_live_", "", "", true},
		{"no separator", "sk_live_absecret", "sk_live_", "", "", true},
		{"empty lookup", "sk_live__secret", "sk_live_", "", "", true},
		{"empty secret", "sk_live_abc_", "sk_live_", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lookup, secret, err := ParseToken(c.raw, c.prefix)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error, got lookup=%q secret=%q", lookup, secret)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if lookup != c.wantLookup {
				t.Errorf("lookup = %q, want %q", lookup, c.wantLookup)
			}
			if secret != c.wantSecret {
				t.Errorf("secret = %q, want %q", secret, c.wantSecret)
			}
		})
	}
}

// newTestLoader spins up an httptest server that plays the auth service's
// /internal/authz/api-token endpoint and returns a Loader pointed at it.
// The handler is supplied by the caller so each test can shape the
// response independently.
func newTestLoader(t *testing.T, handler http.HandlerFunc) *Loader {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := proxy.NewServiceClient(srv.URL).WithHeader("X-Internal-Token", "test-secret")
	return NewLoader(client, 0)
}

// OrJWT with no Authorization header should delegate to the JWT
// middleware entirely. We stub JWT with a handler that always fails so
// "delegation happened" is observable as a 401 from the stub, not from
// our middleware.
func TestOrJWT_NoAuthHeader_DelegatesToJWT(t *testing.T) {
	loader := newTestLoader(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("loader should not be called when no bearer is present")
	})

	// A real JWTMiddleware with no JWKS URL will always fail verification,
	// which is exactly the behaviour we want to observe.
	jwt := pkgmw.NewJWTMiddleware(context.Background(), pkgmw.JWTConfig{})

	var reached bool
	mw := OrJWT(jwt, loader, "sk_live_")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/seller/products", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if reached {
		t.Error("inner handler should not have been reached")
	}
}

// A Bearer token that does NOT match our configured prefix must fall
// through to JWT so existing Auth0-authenticated traffic keeps working.
func TestOrJWT_NonMatchingPrefix_DelegatesToJWT(t *testing.T) {
	loader := newTestLoader(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("loader should not be called for non-matching prefix")
	})
	jwt := pkgmw.NewJWTMiddleware(context.Background(), pkgmw.JWTConfig{})

	mw := OrJWT(jwt, loader, "sk_live_")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("inner handler should not run — JWT stub should 401 first")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/seller/products", nil)
	req.Header.Set("Authorization", "Bearer eyJ.some.jwt")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 from JWT stub", rec.Code)
	}
}

// A valid API token must be decoded, resolved against the loader, and
// populate both tenant.Context and apitoken.Context. This is the happy
// path: everything downstream depends on it.
func TestOrJWT_ValidTokenInjectsContexts(t *testing.T) {
	tokenID := uuid.New()
	tenantID := uuid.New()
	sellerID := uuid.New()

	loader := newTestLoader(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/internal/authz/api-token" {
			t.Errorf("path = %s, want /internal/authz/api-token", r.URL.Path)
		}
		if r.Header.Get("X-Internal-Token") != "test-secret" {
			t.Errorf("missing internal token header")
		}

		var req lookupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Prefix != "sk_live_" || req.Lookup != "abc123" || req.Secret != "shh" {
			t.Errorf("req = %+v, want prefix=sk_live_ lookup=abc123 secret=shh", req)
		}

		_ = json.NewEncoder(w).Encode(lookupResponse{
			Status: "active",
			Token: &lookupTokenField{
				ID:                  tokenID,
				TenantID:            tenantID,
				SellerID:            sellerID,
				Scopes:              []string{"products:read", "orders:read"},
				IssuedByAuth0UserID: "auth0|user-42",
			},
		})
	})
	jwt := pkgmw.NewJWTMiddleware(context.Background(), pkgmw.JWTConfig{})

	var captured struct {
		tenant   tenant.Context
		apiToken Context
		ok       bool
	}
	mw := OrJWT(jwt, loader, "sk_live_")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, err := tenant.FromContext(r.Context())
		if err != nil {
			t.Fatalf("tenant.FromContext: %v", err)
		}
		ac, ok := FromContext(r.Context())
		captured.tenant = tc
		captured.apiToken = ac
		captured.ok = ok
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/seller/products", nil)
	req.Header.Set("Authorization", "Bearer sk_live_abc123_shh")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", rec.Code, rec.Body.String())
	}
	if !captured.ok {
		t.Fatal("apitoken.Context should be present")
	}
	if captured.tenant.TenantID != tenantID {
		t.Errorf("tenant id = %s, want %s", captured.tenant.TenantID, tenantID)
	}
	if captured.tenant.SellerID == nil || *captured.tenant.SellerID != sellerID {
		t.Errorf("seller id = %v, want %s", captured.tenant.SellerID, sellerID)
	}
	if captured.tenant.UserID != "auth0|user-42" {
		t.Errorf("user id = %q, want auth0|user-42", captured.tenant.UserID)
	}
	if len(captured.tenant.Roles) != 1 || captured.tenant.Roles[0] != "seller" {
		t.Errorf("roles = %v, want [seller]", captured.tenant.Roles)
	}
	if captured.apiToken.ID != tokenID {
		t.Errorf("token id = %s, want %s", captured.apiToken.ID, tokenID)
	}
	if !captured.apiToken.HasScope("products:read") {
		t.Error("expected products:read scope")
	}
	if captured.apiToken.HasScope("orders:write") {
		t.Error("unexpected orders:write scope")
	}
}

// Non-active responses from the auth service must become 401. We test
// all four non-active statuses because they're all one statement
// difference from each other in the middleware and a future editor
// might accidentally treat "revoked" as an error class.
func TestOrJWT_NonActiveStatuses_Return401(t *testing.T) {
	for _, status := range []string{"revoked", "expired", "not_found", "invalid"} {
		t.Run(status, func(t *testing.T) {
			loader := newTestLoader(t, func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(lookupResponse{Status: status})
			})
			jwt := pkgmw.NewJWTMiddleware(context.Background(), pkgmw.JWTConfig{})

			mw := OrJWT(jwt, loader, "sk_live_")
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("inner handler should not run")
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/seller/products", nil)
			req.Header.Set("Authorization", "Bearer sk_live_abc_shh")
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status %s → http %d, want 401", status, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "invalid api token") {
				t.Errorf("body = %s, expected uniform invalid token message", rec.Body.String())
			}
		})
	}
}

// Malformed token (missing separator) must 401 before reaching the
// loader — this is a defence-in-depth check since a well-formed request
// with a bad separator could otherwise waste a network hop per attempt.
func TestOrJWT_MalformedToken_Returns401(t *testing.T) {
	loader := newTestLoader(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("loader must not be called for a structurally invalid token")
	})
	jwt := pkgmw.NewJWTMiddleware(context.Background(), pkgmw.JWTConfig{})

	mw := OrJWT(jwt, loader, "sk_live_")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler should not run")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/seller/products", nil)
	req.Header.Set("Authorization", "Bearer sk_live_nodelim")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
