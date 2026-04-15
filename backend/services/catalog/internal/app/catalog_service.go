package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/port"
)

// CatalogService implements business logic for catalog operations.
type CatalogService struct {
	categories port.CategoryStore
	products   port.ProductStore
	skus       port.SKUStore
	publisher  pubsub.Publisher
}

// NewCatalogService creates a new CatalogService.
func NewCatalogService(
	categories port.CategoryStore,
	products port.ProductStore,
	skus port.SKUStore,
	publisher pubsub.Publisher,
) *CatalogService {
	return &CatalogService{
		categories: categories,
		products:   products,
		skus:       skus,
		publisher:  publisher,
	}
}

// publishEvent publishes an event if the publisher is configured.
func (s *CatalogService) publishEvent(ctx context.Context, eventType, topic string, data any) {
	pubsub.PublishEvent(ctx, s.publisher, eventType, topic, data)
}

// --- Category operations ---

// CreateCategory creates a new category.
func (s *CatalogService) CreateCategory(ctx context.Context, c *domain.Category) error {
	// Check slug uniqueness.
	existing, err := s.categories.GetBySlug(ctx, c.Slug)
	if err != nil {
		return apperrors.Internal("failed to check category slug", err)
	}
	if existing != nil {
		return domain.ErrCategorySlugConflict
	}

	if err := s.categories.Create(ctx, c); err != nil {
		return apperrors.Internal("failed to create category", err)
	}

	slog.Info("category created", "id", c.ID, "slug", c.Slug)
	return nil
}

// ListCategories returns all categories.
func (s *CatalogService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	categories, err := s.categories.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list categories", err)
	}
	return categories, nil
}

// UpdateCategory updates an existing category.
func (s *CatalogService) UpdateCategory(ctx context.Context, c *domain.Category) error {
	existing, err := s.categories.GetByID(ctx, c.ID)
	if err != nil {
		return apperrors.Internal("failed to get category", err)
	}
	if existing == nil {
		return domain.ErrCategoryNotFound
	}

	if err := s.categories.Update(ctx, c); err != nil {
		return apperrors.Internal("failed to update category", err)
	}

	slog.Info("category updated", "id", c.ID)
	return nil
}

// --- Product operations ---

// CreateProduct creates a new product with optional SKUs.
func (s *CatalogService) CreateProduct(ctx context.Context, p *domain.Product, skus []domain.SKU) error {
	// Validate seller context.
	tc, err := tenant.FromContext(ctx)
	if err == nil && tc.SellerID != nil {
		p.SellerID = *tc.SellerID
	}

	if p.SellerID == uuid.Nil {
		return domain.ErrSellerRequired
	}

	// Check slug uniqueness.
	existing, err := s.products.GetBySlug(ctx, p.Slug)
	if err != nil {
		return apperrors.Internal("failed to check product slug", err)
	}
	if existing != nil {
		return domain.ErrProductSlugConflict
	}

	p.Status = domain.StatusDraft
	if err := s.products.Create(ctx, p); err != nil {
		return apperrors.Internal("failed to create product", err)
	}

	// Create associated SKUs.
	for i := range skus {
		skus[i].ProductID = p.ID
		skus[i].SellerID = p.SellerID
		skus[i].Status = domain.StatusDraft
		if err := s.skus.Create(ctx, &skus[i]); err != nil {
			return apperrors.Internal("failed to create sku", err)
		}
	}

	slog.Info("product created", "id", p.ID, "slug", p.Slug, "sku_count", len(skus))

	s.publishEvent(ctx, "product.created", "product-events", map[string]any{
		"product_id": p.ID.String(),
		"seller_id":  p.SellerID.String(),
		"name":       p.Name,
		"status":     string(p.Status),
		"slug":       p.Slug,
	})

	return nil
}

// GetProduct retrieves a product with its SKUs by slug.
func (s *CatalogService) GetProduct(ctx context.Context, slug string) (*domain.ProductWithSKUs, error) {
	p, err := s.products.GetWithSKUsBySlug(ctx, slug)
	if err != nil {
		return nil, apperrors.Internal("failed to get product", err)
	}
	if p == nil {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

// GetProductByID retrieves a product by its ID.
func (s *CatalogService) GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	p, err := s.products.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get product", err)
	}
	if p == nil {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

// ListProducts returns a filtered list of products.
// Buyers see only active products; sellers see their own products in any status.
func (s *CatalogService) ListProducts(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int, error) {
	// If the caller is a seller, scope to their own products.
	tc, err := tenant.FromContext(ctx)
	if err == nil && tc.SellerID != nil {
		filter.SellerID = tc.SellerID
	} else {
		// Buyers only see active products.
		active := domain.StatusActive
		filter.Status = &active
	}

	products, total, err := s.products.List(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list products", err)
	}
	return products, total, nil
}

// UpdateProduct updates a product's details.
func (s *CatalogService) UpdateProduct(ctx context.Context, p *domain.Product) error {
	existing, err := s.products.GetByID(ctx, p.ID)
	if err != nil {
		return apperrors.Internal("failed to get product", err)
	}
	if existing == nil {
		return domain.ErrProductNotFound
	}

	// Verify seller ownership.
	tc, tcErr := tenant.FromContext(ctx)
	if tcErr == nil && tc.SellerID != nil && existing.SellerID != *tc.SellerID {
		return domain.ErrNotProductOwner
	}

	if err := s.products.Update(ctx, p); err != nil {
		return apperrors.Internal("failed to update product", err)
	}

	slog.Info("product updated", "id", p.ID)

	s.publishEvent(ctx, "product.updated", "product-events", map[string]any{
		"product_id": p.ID.String(),
		"seller_id":  p.SellerID.String(),
		"name":       p.Name,
		"status":     string(p.Status),
		"slug":       p.Slug,
	})

	return nil
}

// UpdateProductStatus changes a product's status.
func (s *CatalogService) UpdateProductStatus(ctx context.Context, id uuid.UUID, status domain.ProductStatus) error {
	existing, err := s.products.GetByID(ctx, id)
	if err != nil {
		return apperrors.Internal("failed to get product", err)
	}
	if existing == nil {
		return domain.ErrProductNotFound
	}

	// Verify seller ownership.
	tc, tcErr := tenant.FromContext(ctx)
	if tcErr == nil && tc.SellerID != nil && existing.SellerID != *tc.SellerID {
		return domain.ErrNotProductOwner
	}

	if err := s.products.UpdateStatus(ctx, id, status); err != nil {
		return apperrors.Internal("failed to update product status", err)
	}

	slog.Info("product status updated", "id", id, "status", status)

	s.publishEvent(ctx, "product.updated", "product-events", map[string]any{
		"product_id": id.String(),
		"seller_id":  existing.SellerID.String(),
		"name":       existing.Name,
		"status":     string(status),
		"slug":       existing.Slug,
	})

	return nil
}

// ArchiveProduct sets a product's status to archived.
func (s *CatalogService) ArchiveProduct(ctx context.Context, id uuid.UUID) error {
	return s.UpdateProductStatus(ctx, id, domain.StatusArchived)
}

// --- SKU operations ---

// CreateSKU creates a new SKU for a product.
func (s *CatalogService) CreateSKU(ctx context.Context, sku *domain.SKU) error {
	// Verify product exists.
	product, err := s.products.GetByID(ctx, sku.ProductID)
	if err != nil {
		return apperrors.Internal("failed to verify product", err)
	}
	if product == nil {
		return domain.ErrProductNotFound
	}

	sku.SellerID = product.SellerID
	sku.Status = domain.StatusDraft
	if err := s.skus.Create(ctx, sku); err != nil {
		return apperrors.Internal("failed to create sku", err)
	}

	slog.Info("sku created", "id", sku.ID, "product_id", sku.ProductID)
	return nil
}

// GetSKU retrieves a SKU by its ID.
func (s *CatalogService) GetSKU(ctx context.Context, id uuid.UUID) (*domain.SKU, error) {
	sku, err := s.skus.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get sku", err)
	}
	if sku == nil {
		return nil, domain.ErrSKUNotFound
	}
	return sku, nil
}

// GetSKUWithProductName returns a SKU joined with its product name.
// Intended for intra-cluster callers (e.g. cart service) that need the
// full purchasable snapshot in one round-trip.
func (s *CatalogService) GetSKUWithProductName(ctx context.Context, id uuid.UUID) (*port.SKULookup, error) {
	sku, err := s.skus.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get sku", err)
	}
	if sku == nil {
		return nil, domain.ErrSKUNotFound
	}

	product, err := s.products.GetByID(ctx, sku.ProductID)
	if err != nil {
		return nil, apperrors.Internal("failed to get product for sku", err)
	}
	if product == nil {
		return nil, apperrors.NotFound("product not found for sku")
	}

	return &port.SKULookup{
		SKUID:         sku.ID,
		ProductID:     sku.ProductID,
		SellerID:      sku.SellerID,
		ProductName:   product.Name,
		SKUCode:       sku.SKUCode,
		PriceAmount:   sku.PriceAmount,
		PriceCurrency: sku.PriceCurrency,
		Status:        string(sku.Status),
	}, nil
}

// ListSKUs returns all SKUs for a product.
func (s *CatalogService) ListSKUs(ctx context.Context, productID uuid.UUID) ([]domain.SKU, error) {
	skus, err := s.skus.List(ctx, productID)
	if err != nil {
		return nil, apperrors.Internal("failed to list skus", err)
	}
	return skus, nil
}

// UpdateSKU updates a SKU's details.
func (s *CatalogService) UpdateSKU(ctx context.Context, sku *domain.SKU) error {
	existing, err := s.skus.GetByID(ctx, sku.ID)
	if err != nil {
		return apperrors.Internal("failed to get sku", err)
	}
	if existing == nil {
		return domain.ErrSKUNotFound
	}

	if err := s.skus.Update(ctx, sku); err != nil {
		return apperrors.Internal("failed to update sku", err)
	}

	slog.Info("sku updated", "id", sku.ID)
	return nil
}

// UpdateSKUStatus changes a SKU's status.
func (s *CatalogService) UpdateSKUStatus(ctx context.Context, id uuid.UUID, status domain.ProductStatus) error {
	existing, err := s.skus.GetByID(ctx, id)
	if err != nil {
		return apperrors.Internal("failed to get sku", err)
	}
	if existing == nil {
		return domain.ErrSKUNotFound
	}

	if err := s.skus.UpdateStatus(ctx, id, status); err != nil {
		return apperrors.Internal("failed to update sku status", err)
	}

	slog.Info("sku status updated", "id", id, "status", status)
	return nil
}
