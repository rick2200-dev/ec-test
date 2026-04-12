package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
)

// CatalogUseCase is the driving port (inbound) for catalog operations.
// Handlers and the gRPC server depend on this interface;
// *service.CatalogService satisfies it.
type CatalogUseCase interface {
	CreateCategory(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error
	ListCategories(ctx context.Context, tenantID uuid.UUID) ([]domain.Category, error)
	UpdateCategory(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error

	CreateProduct(ctx context.Context, tenantID uuid.UUID, p *domain.Product, skus []domain.SKU) error
	GetProduct(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.ProductWithSKUs, error)
	GetProductByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error)
	ListProducts(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int, error)
	UpdateProduct(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error
	UpdateProductStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error
	ArchiveProduct(ctx context.Context, tenantID, id uuid.UUID) error

	CreateSKU(ctx context.Context, tenantID uuid.UUID, sku *domain.SKU) error
	GetSKU(ctx context.Context, tenantID, id uuid.UUID) (*domain.SKU, error)
	GetSKUWithProductName(ctx context.Context, tenantID, id uuid.UUID) (*SKULookup, error)
	ListSKUs(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.SKU, error)
	UpdateSKU(ctx context.Context, tenantID uuid.UUID, sku *domain.SKU) error
	UpdateSKUStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error
}
