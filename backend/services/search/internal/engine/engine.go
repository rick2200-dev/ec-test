package engine

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
)

// SearchEngine defines the interface for search backends.
type SearchEngine interface {
	// Search executes a product search query and returns results.
	Search(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error)

	// IndexProduct indexes or updates a product in the search backend.
	IndexProduct(ctx context.Context, product domain.ProductEvent) error

	// DeleteProduct removes a product from the search index.
	DeleteProduct(ctx context.Context, tenantID uuid.UUID, productID uuid.UUID) error
}
