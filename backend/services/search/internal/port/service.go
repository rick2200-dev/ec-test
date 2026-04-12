// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the search service.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
)

// SearchUseCase is the driving port (inbound) for search operations.
// Handlers depend on this interface; *service.SearchService satisfies it.
type SearchUseCase interface {
	// Search executes a full-text search against the product index and returns
	// matching products with pagination metadata.
	Search(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error)
	// Suggest returns autocomplete suggestions for the given search query.
	Suggest(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error)
	// IndexProduct indexes or re-indexes a product event (insert/update) in the search backend.
	IndexProduct(ctx context.Context, product domain.ProductEvent) error
	// DeleteProduct removes a product from the search index.
	DeleteProduct(ctx context.Context, tenantID, productID uuid.UUID) error
}
