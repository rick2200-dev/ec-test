package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
	"github.com/Riku-KANO/ec-test/services/search/internal/engine"
)

// ---- test double ----

type mockSearchEngine struct {
	searchFn       func(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error)
	indexProductFn func(ctx context.Context, product domain.ProductEvent) error
	deleteProductFn func(ctx context.Context, tenantID, productID uuid.UUID) error
}

func (m *mockSearchEngine) Search(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, req)
	}
	return &domain.SearchResult{Products: []domain.ProductHit{}, Facets: []domain.Facet{}}, nil
}

func (m *mockSearchEngine) IndexProduct(ctx context.Context, product domain.ProductEvent) error {
	if m.indexProductFn != nil {
		return m.indexProductFn(ctx, product)
	}
	return nil
}

func (m *mockSearchEngine) DeleteProduct(ctx context.Context, tenantID, productID uuid.UUID) error {
	if m.deleteProductFn != nil {
		return m.deleteProductFn(ctx, tenantID, productID)
	}
	return nil
}

// compile-time interface check
var _ engine.SearchEngine = (*mockSearchEngine)(nil)

func newSearchSvc(eng *mockSearchEngine) *SearchService {
	return NewSearchService(eng)
}

// ---- Search ----

func TestSearch_Success(t *testing.T) {
	svc := newSearchSvc(&mockSearchEngine{})

	result, err := svc.Search(context.Background(), domain.SearchRequest{
		TenantID: uuid.New(),
		Query:    "shirt",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestSearch_MissingTenantID(t *testing.T) {
	svc := newSearchSvc(&mockSearchEngine{})

	_, err := svc.Search(context.Background(), domain.SearchRequest{Query: "shirt"})
	if !errors.Is(err, domain.ErrMissingTenantID) {
		t.Errorf("want ErrMissingTenantID, got %v", err)
	}
}

func TestSearch_LimitClampedToDefault(t *testing.T) {
	var capturedLimit int
	eng := &mockSearchEngine{
		searchFn: func(_ context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
			capturedLimit = req.Limit
			return &domain.SearchResult{}, nil
		},
	}
	svc := newSearchSvc(eng)

	// Limit 0 → default 20
	_, _ = svc.Search(context.Background(), domain.SearchRequest{TenantID: uuid.New(), Limit: 0})
	if capturedLimit != 20 {
		t.Errorf("limit = %d, want 20 (default)", capturedLimit)
	}

	// Limit > 100 → default 20
	_, _ = svc.Search(context.Background(), domain.SearchRequest{TenantID: uuid.New(), Limit: 200})
	if capturedLimit != 20 {
		t.Errorf("limit = %d, want 20 (over-max clamped)", capturedLimit)
	}
}

func TestSearch_NegativeOffsetClampedToZero(t *testing.T) {
	var capturedOffset int
	eng := &mockSearchEngine{
		searchFn: func(_ context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
			capturedOffset = req.Offset
			return &domain.SearchResult{}, nil
		},
	}
	svc := newSearchSvc(eng)

	_, _ = svc.Search(context.Background(), domain.SearchRequest{TenantID: uuid.New(), Limit: 10, Offset: -5})
	if capturedOffset != 0 {
		t.Errorf("offset = %d, want 0", capturedOffset)
	}
}

func TestSearch_StatusDefaultsToActive(t *testing.T) {
	var capturedStatus string
	eng := &mockSearchEngine{
		searchFn: func(_ context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
			capturedStatus = req.Status
			return &domain.SearchResult{}, nil
		},
	}
	svc := newSearchSvc(eng)

	_, _ = svc.Search(context.Background(), domain.SearchRequest{TenantID: uuid.New(), Limit: 10})
	if capturedStatus != "active" {
		t.Errorf("status = %q, want %q", capturedStatus, "active")
	}
}

func TestSearch_StatusNotOverriddenWhenSet(t *testing.T) {
	var capturedStatus string
	eng := &mockSearchEngine{
		searchFn: func(_ context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
			capturedStatus = req.Status
			return &domain.SearchResult{}, nil
		},
	}
	svc := newSearchSvc(eng)

	_, _ = svc.Search(context.Background(), domain.SearchRequest{
		TenantID: uuid.New(),
		Limit:    10,
		Status:   "draft",
	})
	if capturedStatus != "draft" {
		t.Errorf("status = %q, want %q", capturedStatus, "draft")
	}
}

func TestSearch_NilResultsReplacedWithEmpty(t *testing.T) {
	eng := &mockSearchEngine{
		searchFn: func(_ context.Context, _ domain.SearchRequest) (*domain.SearchResult, error) {
			return &domain.SearchResult{Products: nil, Facets: nil}, nil
		},
	}
	svc := newSearchSvc(eng)

	result, err := svc.Search(context.Background(), domain.SearchRequest{TenantID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Products == nil {
		t.Error("Products should not be nil")
	}
	if result.Facets == nil {
		t.Error("Facets should not be nil")
	}
}

func TestSearch_EngineError(t *testing.T) {
	eng := &mockSearchEngine{
		searchFn: func(_ context.Context, _ domain.SearchRequest) (*domain.SearchResult, error) {
			return nil, errors.New("engine down")
		},
	}
	svc := newSearchSvc(eng)

	_, err := svc.Search(context.Background(), domain.SearchRequest{TenantID: uuid.New()})

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *apperrors.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Errorf("status = %d, want 500", appErr.Status)
	}
}

// ---- Suggest ----

func TestSuggest_MissingTenantID(t *testing.T) {
	svc := newSearchSvc(&mockSearchEngine{})

	_, err := svc.Suggest(context.Background(), domain.SearchRequest{Query: "sh"})
	if !errors.Is(err, domain.ErrMissingTenantID) {
		t.Errorf("want ErrMissingTenantID, got %v", err)
	}
}

func TestSuggest_ForcesLimitAndOffset(t *testing.T) {
	var capturedReq domain.SearchRequest
	eng := &mockSearchEngine{
		searchFn: func(_ context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
			capturedReq = req
			return &domain.SearchResult{}, nil
		},
	}
	svc := newSearchSvc(eng)

	_, _ = svc.Suggest(context.Background(), domain.SearchRequest{
		TenantID: uuid.New(),
		Query:    "sh",
		Limit:    50,
		Offset:   100,
	})
	if capturedReq.Limit != 10 {
		t.Errorf("limit = %d, want 10", capturedReq.Limit)
	}
	if capturedReq.Offset != 0 {
		t.Errorf("offset = %d, want 0", capturedReq.Offset)
	}
	if capturedReq.Status != "active" {
		t.Errorf("status = %q, want %q", capturedReq.Status, "active")
	}
}

func TestSuggest_FacetsAlwaysEmpty(t *testing.T) {
	eng := &mockSearchEngine{
		searchFn: func(_ context.Context, _ domain.SearchRequest) (*domain.SearchResult, error) {
			return &domain.SearchResult{
				Products: []domain.ProductHit{{Name: "shirt"}},
				Facets:   []domain.Facet{{Field: "category"}},
			}, nil
		},
	}
	svc := newSearchSvc(eng)

	result, err := svc.Suggest(context.Background(), domain.SearchRequest{TenantID: uuid.New(), Query: "sh"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facets) != 0 {
		t.Errorf("Facets = %v, want empty (suggestions never return facets)", result.Facets)
	}
	if len(result.Products) == 0 {
		t.Error("Products should not be empty")
	}
}

// ---- IndexProduct / DeleteProduct ----

func TestIndexProduct_DelegatesToEngine(t *testing.T) {
	called := false
	eng := &mockSearchEngine{
		indexProductFn: func(_ context.Context, _ domain.ProductEvent) error {
			called = true
			return nil
		},
	}
	svc := newSearchSvc(eng)

	if err := svc.IndexProduct(context.Background(), domain.ProductEvent{ID: uuid.New()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("engine.IndexProduct was not called")
	}
}

func TestDeleteProduct_DelegatesToEngine(t *testing.T) {
	called := false
	eng := &mockSearchEngine{
		deleteProductFn: func(_ context.Context, _, _ uuid.UUID) error {
			called = true
			return nil
		},
	}
	svc := newSearchSvc(eng)

	if err := svc.DeleteProduct(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("engine.DeleteProduct was not called")
	}
}
