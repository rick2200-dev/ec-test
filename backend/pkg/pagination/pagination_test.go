package pagination_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Riku-KANO/ec-test/pkg/pagination"
)

func TestFromRequest_Defaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items", nil)
	p := pagination.FromRequest(r)
	if p.Limit != pagination.DefaultLimit {
		t.Errorf("expected default limit %d, got %d", pagination.DefaultLimit, p.Limit)
	}
	if p.Offset != 0 {
		t.Errorf("expected default offset 0, got %d", p.Offset)
	}
}

func TestFromRequest_ValidParams(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=10&offset=20", nil)
	p := pagination.FromRequest(r)
	if p.Limit != 10 {
		t.Errorf("expected limit 10, got %d", p.Limit)
	}
	if p.Offset != 20 {
		t.Errorf("expected offset 20, got %d", p.Offset)
	}
}

func TestFromRequest_LimitAboveMax(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=999", nil)
	p := pagination.FromRequest(r)
	if p.Limit != pagination.MaxLimit {
		t.Errorf("expected limit capped at %d, got %d", pagination.MaxLimit, p.Limit)
	}
}

func TestFromRequest_NonNumericLimit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=abc", nil)
	p := pagination.FromRequest(r)
	if p.Limit != pagination.DefaultLimit {
		t.Errorf("expected default limit %d for non-numeric input, got %d", pagination.DefaultLimit, p.Limit)
	}
}

func TestFromRequest_NegativeLimit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?limit=-5", nil)
	p := pagination.FromRequest(r)
	if p.Limit != pagination.DefaultLimit {
		t.Errorf("expected default limit %d for negative input, got %d", pagination.DefaultLimit, p.Limit)
	}
}

func TestFromRequest_NegativeOffset(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?offset=-1", nil)
	p := pagination.FromRequest(r)
	if p.Offset != 0 {
		t.Errorf("expected offset 0 for negative input, got %d", p.Offset)
	}
}

func TestFromRequest_ZeroOffset(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/items?offset=0", nil)
	p := pagination.FromRequest(r)
	if p.Offset != 0 {
		t.Errorf("expected offset 0, got %d", p.Offset)
	}
}
