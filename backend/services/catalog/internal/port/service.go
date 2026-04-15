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
	// CreateCategory creates a new category.
	CreateCategory(ctx context.Context, c *domain.Category) error
	// ListCategories returns all categories.
	ListCategories(ctx context.Context) ([]domain.Category, error)
	// UpdateCategory persists changes to an existing category.
	UpdateCategory(ctx context.Context, c *domain.Category) error

	// CreateProduct creates a product along with its initial SKUs in a single operation.
	CreateProduct(ctx context.Context, p *domain.Product, skus []domain.SKU) error
	// GetProduct retrieves a product with all its SKUs by slug.
	GetProduct(ctx context.Context, slug string) (*domain.ProductWithSKUs, error)
	// GetProductByID retrieves a product by its UUID without SKUs.
	GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	// ListProducts returns a paginated, filtered list of products.
	ListProducts(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int, error)
	// UpdateProduct persists changes to a product's metadata.
	UpdateProduct(ctx context.Context, p *domain.Product) error
	// UpdateProductStatus changes the publish/archive status of a product.
	UpdateProductStatus(ctx context.Context, id uuid.UUID, status domain.ProductStatus) error
	// ArchiveProduct sets the product's status to archived, hiding it from buyers.
	ArchiveProduct(ctx context.Context, id uuid.UUID) error

	// CreateSKU adds a new SKU to an existing product.
	CreateSKU(ctx context.Context, sku *domain.SKU) error
	// GetSKU retrieves a SKU by its UUID.
	GetSKU(ctx context.Context, id uuid.UUID) (*domain.SKU, error)
	// GetSKUWithProductName retrieves a SKU together with its parent product's name and seller ID.
	// Used by the cart service to snapshot price and display metadata at add-to-cart time.
	GetSKUWithProductName(ctx context.Context, id uuid.UUID) (*SKULookup, error)
	// ListSKUs returns all SKUs belonging to the given product.
	ListSKUs(ctx context.Context, productID uuid.UUID) ([]domain.SKU, error)
	// UpdateSKU persists changes to an existing SKU.
	UpdateSKU(ctx context.Context, sku *domain.SKU) error
	// UpdateSKUStatus changes the active/archive status of a SKU.
	UpdateSKUStatus(ctx context.Context, id uuid.UUID, status domain.ProductStatus) error
	// BatchGetSKUMappings resolves a batch of SKU ids to their (product_id,
	// seller_id). Unknown ids are silently omitted.
	BatchGetSKUMappings(ctx context.Context, ids []uuid.UUID) ([]SKUMapping, error)
}
