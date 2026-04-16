package app

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	catalogv1 "github.com/Riku-KANO/ec-test/services/catalog/api/gen/go/catalog/v1"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
	"github.com/Riku-KANO/ec-test/services/catalog/internal/port"
)

const productEventsTopic = "product-events"

// CatalogService implements business logic for catalog operations.
type CatalogService struct {
	categories port.CategoryStore
	products   port.ProductStore
	skus       port.SKUStore
	outbox     port.OutboxStore
	txRunner   port.TxRunner
}

// NewCatalogService creates a new CatalogService. The outbox store + tx
// runner replace the former direct Pub/Sub publisher: product lifecycle
// events are now appended to catalog_svc.outbox_events inside the same Tx
// as the product write, and drained to Pub/Sub by a separate relay worker.
func NewCatalogService(
	categories port.CategoryStore,
	products port.ProductStore,
	skus port.SKUStore,
	outbox port.OutboxStore,
	txRunner port.TxRunner,
) *CatalogService {
	return &CatalogService{
		categories: categories,
		products:   products,
		skus:       skus,
		outbox:     outbox,
		txRunner:   txRunner,
	}
}

// mustProtoJSON serializes a proto message using snake_case field names so
// consumers see the same wire format as JSON-struct publishers. Panics are
// impossible in practice since every proto payload here is structurally
// valid; marshal errors are swallowed and logged.
func mustProtoJSON(m proto.Message) json.RawMessage {
	b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(m)
	if err != nil {
		slog.Warn("failed to marshal proto event payload", "error", err)
		return json.RawMessage("{}")
	}
	return b
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
	if err := s.txRunner.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.products.Create(txCtx, p); err != nil {
			return apperrors.Internal("failed to create product", err)
		}
		for i := range skus {
			skus[i].ProductID = p.ID
			skus[i].SellerID = p.SellerID
			skus[i].Status = domain.StatusDraft
			if err := s.skus.Create(txCtx, &skus[i]); err != nil {
				return apperrors.Internal("failed to create sku", err)
			}
		}
		return s.outbox.AppendOutboxEvent(txCtx, port.OutboxEvent{
			EventType: "product.created",
			Topic:     productEventsTopic,
			Payload: mustProtoJSON(&catalogv1.ProductCreated{
				Id:          p.ID.String(),
				SellerId:    p.SellerID.String(),
				Name:        p.Name,
				Slug:        p.Slug,
				Description: p.Description,
				Status:      string(p.Status),
			}),
		})
	}); err != nil {
		return err
	}

	slog.Info("product created", "id", p.ID, "slug", p.Slug, "sku_count", len(skus))
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

	// HTTP handlers only populate the fields the caller can change (name,
	// slug, description, attributes, category_id) and leave seller_id +
	// status unset. Backfill from the existing row so the event payload
	// reflects the full current state rather than zero values.
	p.SellerID = existing.SellerID
	if p.Status == "" {
		p.Status = existing.Status
	}

	if err := s.txRunner.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.products.Update(txCtx, p); err != nil {
			return apperrors.Internal("failed to update product", err)
		}
		return s.outbox.AppendOutboxEvent(txCtx, port.OutboxEvent{
			EventType: "product.updated",
			Topic:     productEventsTopic,
			Payload: mustProtoJSON(&catalogv1.ProductUpdated{
				Id:          p.ID.String(),
				SellerId:    p.SellerID.String(),
				Name:        p.Name,
				Slug:        p.Slug,
				Description: p.Description,
				Status:      string(p.Status),
			}),
		})
	}); err != nil {
		return err
	}

	slog.Info("product updated", "id", p.ID)
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

	if err := s.txRunner.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.products.UpdateStatus(txCtx, id, status); err != nil {
			return apperrors.Internal("failed to update product status", err)
		}
		// Archive transitions are the delete signal for downstream consumers
		// (search, recommend); distinct event type so they can evict their
		// projections without re-parsing the status field.
		if status == domain.StatusArchived {
			return s.outbox.AppendOutboxEvent(txCtx, port.OutboxEvent{
				EventType: "product.deleted",
				Topic:     productEventsTopic,
				Payload: mustProtoJSON(&catalogv1.ProductDeleted{
					Id: id.String(),
				}),
			})
		}
		return s.outbox.AppendOutboxEvent(txCtx, port.OutboxEvent{
			EventType: "product.updated",
			Topic:     productEventsTopic,
			Payload: mustProtoJSON(&catalogv1.ProductUpdated{
				Id:          id.String(),
				SellerId:    existing.SellerID.String(),
				Name:        existing.Name,
				Slug:        existing.Slug,
				Description: existing.Description,
				Status:      string(status),
			}),
		})
	}); err != nil {
		return err
	}

	slog.Info("product status updated", "id", id, "status", status)
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

// BatchGetSKUMappings resolves a batch of SKU ids to (sku_id, product_id,
// seller_id) triples. Unknown ids are silently omitted from the response so
// callers must tolerate length mismatch. Intended for internal callers
// (order at checkout time) that snapshot product_id onto order lines.
func (s *CatalogService) BatchGetSKUMappings(ctx context.Context, ids []uuid.UUID) ([]port.SKUMapping, error) {
	mappings, err := s.skus.BatchGetMappings(ctx, ids)
	if err != nil {
		return nil, apperrors.Internal("failed to batch get sku mappings", err)
	}
	return mappings, nil
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
