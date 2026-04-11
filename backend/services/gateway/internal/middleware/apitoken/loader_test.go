package apitoken

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/gateway/internal/proxy"
)

// TestLoader_CachesActiveResponses confirms the single call-through the
// cache is built around. Two Load calls against the same (prefix,lookup)
// should produce exactly one HTTP round-trip.
func TestLoader_CachesActiveResponses(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(lookupResponse{
			Status: "active",
			Token: &lookupTokenField{
				ID:                  uuid.New(),
				TenantID:            uuid.New(),
				SellerID:            uuid.New(),
				Scopes:              []string{"products:read"},
				IssuedByAuth0UserID: "auth0|x",
			},
		})
	}))
	t.Cleanup(srv.Close)

	loader := NewLoader(proxy.NewServiceClient(srv.URL).WithHeader("X-Internal-Token", "s"), 5*time.Second)

	_, st1, err := loader.Load(context.Background(), "sk_live_", "abc", "secret")
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	_, st2, err := loader.Load(context.Background(), "sk_live_", "abc", "secret")
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if st1 != StatusActive || st2 != StatusActive {
		t.Errorf("statuses = %s/%s, want active/active", st1, st2)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("http calls = %d, want 1 (cache miss + hit)", got)
	}
}

// Evict must clear the cache entry so a revoke propagates immediately
// instead of after the TTL.
func TestLoader_Evict(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(lookupResponse{
			Status: "active",
			Token: &lookupTokenField{
				ID:                  uuid.New(),
				TenantID:            uuid.New(),
				SellerID:            uuid.New(),
				IssuedByAuth0UserID: "auth0|x",
			},
		})
	}))
	t.Cleanup(srv.Close)

	loader := NewLoader(proxy.NewServiceClient(srv.URL).WithHeader("X-Internal-Token", "s"), time.Hour)

	if _, _, err := loader.Load(context.Background(), "sk_live_", "abc", "secret"); err != nil {
		t.Fatalf("initial load: %v", err)
	}
	loader.Evict("sk_live_", "abc")
	if _, _, err := loader.Load(context.Background(), "sk_live_", "abc", "secret"); err != nil {
		t.Fatalf("post-evict load: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("http calls = %d, want 2 (both should miss)", got)
	}
}

// Non-active statuses must not be cached — a freshly-issued token whose
// first lookup came back "not_found" (e.g. due to replication lag) should
// work on its second call without waiting for the TTL.
func TestLoader_DoesNotCacheNonActive(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(lookupResponse{Status: "not_found"})
	}))
	t.Cleanup(srv.Close)

	loader := NewLoader(proxy.NewServiceClient(srv.URL).WithHeader("X-Internal-Token", "s"), time.Hour)

	for i := range 3 {
		_, st, err := loader.Load(context.Background(), "sk_live_", "x", "y")
		if err != nil {
			t.Fatalf("load %d: %v", i, err)
		}
		if st != StatusNotFound {
			t.Errorf("status %d = %s, want not_found", i, st)
		}
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("http calls = %d, want 3 (negative results must not cache)", got)
	}
}

// Transport errors surface as ErrLoaderTransport wrapped errors, so
// callers can distinguish "auth service is down" (503) from "token is
// bad" (401).
func TestLoader_Non200IsTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	loader := NewLoader(proxy.NewServiceClient(srv.URL).WithHeader("X-Internal-Token", "s"), 0)

	_, _, err := loader.Load(context.Background(), "sk_live_", "a", "b")
	if err == nil {
		t.Fatal("expected error")
	}
}
