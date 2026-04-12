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
	Search(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error)
	Suggest(ctx context.Context, req domain.SearchRequest) (*domain.SearchResult, error)
	IndexProduct(ctx context.Context, product domain.ProductEvent) error
	DeleteProduct(ctx context.Context, tenantID, productID uuid.UUID) error
}
