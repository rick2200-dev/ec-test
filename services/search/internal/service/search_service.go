package service

import (
	"context"
	"log/slog"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
	"github.com/Riku-KANO/ec-test/services/search/internal/engine"
	"github.com/google/uuid"
)

// SearchService provides business logic for product search.
type SearchService struct {
	engine engine.SearchEngine
}

// NewSearchService creates a new SearchService.
func NewSearchService(eng engine.SearchEngine) *SearchService {
	return &SearchService{engine: eng}
}

// Search validates the request and delegates to the search engine.
func (s *SearchService) Search(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
	// tenant_id is always required
	if req.TenantID == uuid.Nil {
		return nil, apperrors.BadRequest("tenant_id is required")
	}

	// Enforce defaults
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Buyers should only see active products
	if req.Status == "" {
		req.Status = "active"
	}

	result, err := s.engine.Search(ctx, req)
	if err != nil {
		slog.Error("search failed", "error", err, "query", req.Query, "tenant_id", req.TenantID)
		return nil, apperrors.Internal("search failed", err)
	}

	if result.Products == nil {
		result.Products = []domain.ProductHit{}
	}
	if result.Facets == nil {
		result.Facets = []domain.Facet{}
	}

	return result, nil
}

// Suggest returns search suggestions based on a prefix query.
func (s *SearchService) Suggest(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error) {
	if req.TenantID == uuid.Nil {
		return nil, apperrors.BadRequest("tenant_id is required")
	}

	// Suggestions return a small number of results
	req.Limit = 10
	req.Offset = 0
	req.Status = "active"

	result, err := s.engine.Search(ctx, req)
	if err != nil {
		slog.Error("suggest failed", "error", err, "query", req.Query, "tenant_id", req.TenantID)
		return nil, apperrors.Internal("suggest failed", err)
	}

	if result.Products == nil {
		result.Products = []domain.ProductHit{}
	}
	// No facets for suggestions
	result.Facets = []domain.Facet{}

	return result, nil
}

// IndexProduct delegates indexing to the search engine.
func (s *SearchService) IndexProduct(ctx context.Context, product domain.ProductEvent) error {
	return s.engine.IndexProduct(ctx, product)
}

// DeleteProduct delegates deletion to the search engine.
func (s *SearchService) DeleteProduct(ctx context.Context, tenantID, productID uuid.UUID) error {
	return s.engine.DeleteProduct(ctx, tenantID, productID)
}
