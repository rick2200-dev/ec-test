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
	// CreateCategory creates a new category within the tenant.
	CreateCategory(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error
	// ListCategories returns all categories for the tenant.
	ListCategories(ctx context.Context, tenantID uuid.UUID) ([]domain.Category, error)
	// UpdateCategory persists changes to an existing category.
	UpdateCategory(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error

	// CreateProduct creates a product along with its initial SKUs in a single operation.
	CreateProduct(ctx context.Context, tenantID uuid.UUID, p *domain.Product, skus []domain.SKU) error
	// GetProduct retrieves a product with all its SKUs by slug.
	GetProduct(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.ProductWithSKUs, error)
	// GetProductByID retrieves a product by its UUID without SKUs.
	GetProductByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error)
	// ListProducts returns a paginated, filtered list of products.
	ListProducts(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int, error)
	// UpdateProduct persists changes to a product's metadata.
	UpdateProduct(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error
	// UpdateProductStatus changes the publish/archive status of a product.
	UpdateProductStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error
	// ArchiveProduct sets the product's status to archived, hiding it from buyers.
	ArchiveProduct(ctx context.Context, tenantID, id uuid.UUID) error

	// CreateSKU adds a new SKU to an existing product.
	CreateSKU(ctx context.Context, tenantID uuid.UUID, sku *domain.SKU) error
	// GetSKU retrieves a SKU by its UUID.
	GetSKU(ctx context.Context, tenantID, id uuid.UUID) (*domain.SKU, error)
	// GetSKUWithProductName retrieves a SKU together with its parent product's name and seller ID.
	// Used by the cart service to snapshot price and display metadata at add-to-cart time.
	GetSKUWithProductName(ctx context.Context, tenantID, id uuid.UUID) (*SKULookup, error)
	// ListSKUs returns all SKUs belonging to the given product.
	ListSKUs(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.SKU, error)
	// UpdateSKU persists changes to an existing SKU.
	UpdateSKU(ctx context.Context, tenantID uuid.UUID, sku *domain.SKU) error
	// UpdateSKUStatus changes the active/archive status of a SKU.
	UpdateSKUStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error
}
