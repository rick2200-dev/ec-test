package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Riku-KANO/ec-test/pkg/middleware"
)

func TestRequireInternalToken_ValidToken(t *testing.T) {
	secret := "my-secret-token"
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireInternalToken(secret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/internal/something", nil)
	req.Header.Set("X-Internal-Token", secret)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequireInternalToken_InvalidToken(t *testing.T) {
	secret := "my-secret-token"
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	handler := middleware.RequireInternalToken(secret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/internal/something", nil)
	req.Header.Set("X-Internal-Token", "wrong-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireInternalToken_MissingToken(t *testing.T) {
	secret := "my-secret-token"
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	handler := middleware.RequireInternalToken(secret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/internal/something", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireInternalToken_EmptySecret(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	handler := middleware.RequireInternalToken("")(inner)
	req := httptest.NewRequest(http.MethodGet, "/internal/something", nil)
	req.Header.Set("X-Internal-Token", "any-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestRequireInternalToken_EmptySecretNoHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	handler := middleware.RequireInternalToken("")(inner)
	req := httptest.NewRequest(http.MethodGet, "/internal/something", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestRequireInternalToken_DifferentLengthToken(t *testing.T) {
	secret := "short"
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not have been called")
	})

	handler := middleware.RequireInternalToken(secret)(inner)
	req := httptest.NewRequest(http.MethodGet, "/internal/something", nil)
	req.Header.Set("X-Internal-Token", "much-longer-token-value")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
