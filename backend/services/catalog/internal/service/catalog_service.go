package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/repository"
)

// CatalogService implements business logic for catalog operations.
type CatalogService struct {
	categories *repository.CategoryRepository
	products   *repository.ProductRepository
	skus       *repository.SKURepository
	publisher  pubsub.Publisher
}

// NewCatalogService creates a new CatalogService.
func NewCatalogService(
	categories *repository.CategoryRepository,
	products *repository.ProductRepository,
	skus *repository.SKURepository,
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
func (s *CatalogService) publishEvent(ctx context.Context, tenantID uuid.UUID, eventType, topic string, data any) {
	pubsub.PublishEvent(ctx, s.publisher, tenantID, eventType, topic, data)
}

// --- Category operations ---

// CreateCategory creates a new category.
func (s *CatalogService) CreateCategory(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error {
	// Check slug uniqueness within tenant.
	existing, err := s.categories.GetBySlug(ctx, tenantID, c.Slug)
	if err != nil {
		return apperrors.Internal("failed to check category slug", err)
	}
	if existing != nil {
		return apperrors.Conflict("category slug already exists")
	}

	if err := s.categories.Create(ctx, tenantID, c); err != nil {
		return apperrors.Internal("failed to create category", err)
	}

	slog.Info("category created", "id", c.ID, "tenant_id", tenantID, "slug", c.Slug)
	return nil
}

// ListCategories returns all categories for a tenant.
func (s *CatalogService) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]domain.Category, error) {
	categories, err := s.categories.List(ctx, tenantID)
	if err != nil {
		return nil, apperrors.Internal("failed to list categories", err)
	}
	return categories, nil
}

// UpdateCategory updates an existing category.
func (s *CatalogService) UpdateCategory(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error {
	existing, err := s.categories.GetByID(ctx, tenantID, c.ID)
	if err != nil {
		return apperrors.Internal("failed to get category", err)
	}
	if existing == nil {
		return apperrors.NotFound("category not found")
	}

	if err := s.categories.Update(ctx, tenantID, c); err != nil {
		return apperrors.Internal("failed to update category", err)
	}

	slog.Info("category updated", "id", c.ID, "tenant_id", tenantID)
	return nil
}

// --- Product operations ---

// CreateProduct creates a new product with optional SKUs.
func (s *CatalogService) CreateProduct(ctx context.Context, tenantID uuid.UUID, p *domain.Product, skus []domain.SKU) error {
	// Validate seller context.
	tc, err := tenant.FromContext(ctx)
	if err == nil && tc.SellerID != nil {
		p.SellerID = *tc.SellerID
	}

	if p.SellerID == uuid.Nil {
		return apperrors.BadRequest("seller_id is required")
	}

	// Check slug uniqueness.
	existing, err := s.products.GetBySlug(ctx, tenantID, p.Slug)
	if err != nil {
		return apperrors.Internal("failed to check product slug", err)
	}
	if existing != nil {
		return apperrors.Conflict("product slug already exists")
	}

	p.Status = domain.StatusDraft
	if err := s.products.Create(ctx, tenantID, p); err != nil {
		return apperrors.Internal("failed to create product", err)
	}

	// Create associated SKUs.
	for i := range skus {
		skus[i].ProductID = p.ID
		skus[i].SellerID = p.SellerID
		skus[i].Status = domain.StatusDraft
		if err := s.skus.Create(ctx, tenantID, &skus[i]); err != nil {
			return apperrors.Internal("failed to create sku", err)
		}
	}

	slog.Info("product created", "id", p.ID, "tenant_id", tenantID, "slug", p.Slug, "sku_count", len(skus))

	s.publishEvent(ctx, tenantID, "product.created", "product-events", map[string]any{
		"product_id": p.ID.String(),
		"seller_id":  p.SellerID.String(),
		"name":       p.Name,
		"status":     string(p.Status),
		"slug":       p.Slug,
	})

	return nil
}

// GetProduct retrieves a product with its SKUs by slug.
func (s *CatalogService) GetProduct(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.ProductWithSKUs, error) {
	p, err := s.products.GetWithSKUsBySlug(ctx, tenantID, slug)
	if err != nil {
		return nil, apperrors.Internal("failed to get product", err)
	}
	if p == nil {
		return nil, apperrors.NotFound("product not found")
	}
	return p, nil
}

// GetProductByID retrieves a product by its ID.
func (s *CatalogService) GetProductByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	p, err := s.products.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get product", err)
	}
	if p == nil {
		return nil, apperrors.NotFound("product not found")
	}
	return p, nil
}

// ListProducts returns a filtered list of products.
// Buyers see only active products; sellers see their own products in any status.
func (s *CatalogService) ListProducts(ctx context.Context, filter repository.ProductFilter, limit, offset int) ([]domain.Product, int, error) {
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
func (s *CatalogService) UpdateProduct(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error {
	existing, err := s.products.GetByID(ctx, tenantID, p.ID)
	if err != nil {
		return apperrors.Internal("failed to get product", err)
	}
	if existing == nil {
		return apperrors.NotFound("product not found")
	}

	// Verify seller ownership.
	tc, tcErr := tenant.FromContext(ctx)
	if tcErr == nil && tc.SellerID != nil && existing.SellerID != *tc.SellerID {
		return apperrors.Forbidden("not authorized to update this product")
	}

	if err := s.products.Update(ctx, tenantID, p); err != nil {
		return apperrors.Internal("failed to update product", err)
	}

	slog.Info("product updated", "id", p.ID, "tenant_id", tenantID)

	s.publishEvent(ctx, tenantID, "product.updated", "product-events", map[string]any{
		"product_id": p.ID.String(),
		"seller_id":  p.SellerID.String(),
		"name":       p.Name,
		"status":     string(p.Status),
		"slug":       p.Slug,
	})

	return nil
}

// UpdateProductStatus changes a product's status.
func (s *CatalogService) UpdateProductStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error {
	existing, err := s.products.GetByID(ctx, tenantID, id)
	if err != nil {
		return apperrors.Internal("failed to get product", err)
	}
	if existing == nil {
		return apperrors.NotFound("product not found")
	}

	// Verify seller ownership.
	tc, tcErr := tenant.FromContext(ctx)
	if tcErr == nil && tc.SellerID != nil && existing.SellerID != *tc.SellerID {
		return apperrors.Forbidden("not authorized to update this product")
	}

	if err := s.products.UpdateStatus(ctx, tenantID, id, status); err != nil {
		return apperrors.Internal("failed to update product status", err)
	}

	slog.Info("product status updated", "id", id, "tenant_id", tenantID, "status", status)

	s.publishEvent(ctx, tenantID, "product.updated", "product-events", map[string]any{
		"product_id": id.String(),
		"seller_id":  existing.SellerID.String(),
		"name":       existing.Name,
		"status":     string(status),
		"slug":       existing.Slug,
	})

	return nil
}

// ArchiveProduct sets a product's status to archived.
func (s *CatalogService) ArchiveProduct(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.UpdateProductStatus(ctx, tenantID, id, domain.StatusArchived)
}

// --- SKU operations ---

// CreateSKU creates a new SKU for a product.
func (s *CatalogService) CreateSKU(ctx context.Context, tenantID uuid.UUID, sku *domain.SKU) error {
	// Verify product exists.
	product, err := s.products.GetByID(ctx, tenantID, sku.ProductID)
	if err != nil {
		return apperrors.Internal("failed to verify product", err)
	}
	if product == nil {
		return apperrors.NotFound("product not found")
	}

	sku.SellerID = product.SellerID
	sku.Status = domain.StatusDraft
	if err := s.skus.Create(ctx, tenantID, sku); err != nil {
		return apperrors.Internal("failed to create sku", err)
	}

	slog.Info("sku created", "id", sku.ID, "product_id", sku.ProductID, "tenant_id", tenantID)
	return nil
}

// GetSKU retrieves a SKU by its ID.
func (s *CatalogService) GetSKU(ctx context.Context, tenantID, id uuid.UUID) (*domain.SKU, error) {
	sku, err := s.skus.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get sku", err)
	}
	if sku == nil {
		return nil, apperrors.NotFound("sku not found")
	}
	return sku, nil
}

// SKULookup is the shape returned by GetSKUWithProductName, used by the
// cart service to snapshot price and display metadata at add-to-cart time.
type SKULookup struct {
	SKUID         uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"product_id"`
	SellerID      uuid.UUID `json:"seller_id"`
	ProductName   string    `json:"product_name"`
	SKUCode       string    `json:"sku_code"`
	PriceAmount   int64     `json:"price_amount"`
	PriceCurrency string    `json:"price_currency"`
	Status        string    `json:"status"`
}

// GetSKUWithProductName returns a SKU joined with its product name.
// Intended for intra-cluster callers (e.g. cart service) that need the
// full purchasable snapshot in one round-trip.
func (s *CatalogService) GetSKUWithProductName(ctx context.Context, tenantID, id uuid.UUID) (*SKULookup, error) {
	sku, err := s.skus.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get sku", err)
	}
	if sku == nil {
		return nil, apperrors.NotFound("sku not found")
	}

	product, err := s.products.GetByID(ctx, tenantID, sku.ProductID)
	if err != nil {
		return nil, apperrors.Internal("failed to get product for sku", err)
	}
	if product == nil {
		return nil, apperrors.NotFound("product not found for sku")
	}

	return &SKULookup{
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
func (s *CatalogService) ListSKUs(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.SKU, error) {
	skus, err := s.skus.List(ctx, tenantID, productID)
	if err != nil {
		return nil, apperrors.Internal("failed to list skus", err)
	}
	return skus, nil
}

// UpdateSKU updates a SKU's details.
func (s *CatalogService) UpdateSKU(ctx context.Context, tenantID uuid.UUID, sku *domain.SKU) error {
	existing, err := s.skus.GetByID(ctx, tenantID, sku.ID)
	if err != nil {
		return apperrors.Internal("failed to get sku", err)
	}
	if existing == nil {
		return apperrors.NotFound("sku not found")
	}

	if err := s.skus.Update(ctx, tenantID, sku); err != nil {
		return apperrors.Internal("failed to update sku", err)
	}

	slog.Info("sku updated", "id", sku.ID, "tenant_id", tenantID)
	return nil
}

// UpdateSKUStatus changes a SKU's status.
func (s *CatalogService) UpdateSKUStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error {
	existing, err := s.skus.GetByID(ctx, tenantID, id)
	if err != nil {
		return apperrors.Internal("failed to get sku", err)
	}
	if existing == nil {
		return apperrors.NotFound("sku not found")
	}

	if err := s.skus.UpdateStatus(ctx, tenantID, id, status); err != nil {
		return apperrors.Internal("failed to update sku status", err)
	}

	slog.Info("sku status updated", "id", id, "tenant_id", tenantID, "status", status)
	return nil
}
