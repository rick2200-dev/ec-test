// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the catalog service.
package port

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
)

// OutboxEvent is a catalog domain event destined for the "product-events"
// topic. It is written to catalog_svc.outbox_events inside the same Tx as
// the product mutation so publication is guaranteed even if the process
// crashes before a direct Pub/Sub call would have returned.
type OutboxEvent struct {
	ID        uuid.UUID
	EventType string
	Topic     string
	Payload   json.RawMessage
}

// OutboxStore persists outbox rows + lets the relay worker claim them.
// The persistence side (AppendOutboxEvent) is called from the catalog app
// service under a Tx; claim/markPublished live in the relay worker and
// stay package-private to internal/adapter/pubsub.
type OutboxStore interface {
	// AppendOutboxEvent inserts a new row. If the caller has a Tx in
	// context the insert joins it; otherwise a short auto-commit Tx is
	// opened on the pool.
	AppendOutboxEvent(ctx context.Context, e OutboxEvent) error
}

// TxRunner is the driven port for running database transactions. Reuses
// pkg/database.PoolTxRunner in production.
type TxRunner interface {
	RunTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// CategoryStore is the driven port for category persistence.
// *repository.CategoryRepository satisfies this interface.
type CategoryStore interface {
	// Create persists a new category.
	Create(ctx context.Context, c *domain.Category) error
	// GetByID retrieves a category by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	// GetBySlug retrieves a category by its URL-friendly slug.
	GetBySlug(ctx context.Context, slug string) (*domain.Category, error)
	// List returns all categories (no pagination; category trees are typically small).
	List(ctx context.Context) ([]domain.Category, error)
	// Update persists changes to an existing category.
	Update(ctx context.Context, c *domain.Category) error
}

// ProductStore is the driven port for product persistence.
// *repository.ProductRepository satisfies this interface.
type ProductStore interface {
	// Create persists a new product.
	Create(ctx context.Context, p *domain.Product) error
	// GetByID retrieves a product by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	// GetBySlug retrieves a product by its URL-friendly slug.
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)
	// GetWithSKUsBySlug retrieves a product together with all its SKUs by slug.
	GetWithSKUsBySlug(ctx context.Context, slug string) (*domain.ProductWithSKUs, error)
	// List returns a paginated, filtered list of products.
	List(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int, error)
	// Update persists changes to an existing product's metadata.
	Update(ctx context.Context, p *domain.Product) error
	// UpdateStatus changes the publish/archive status of a product.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ProductStatus) error
	// GetWithSKUs retrieves a product together with all its SKUs by product ID.
	GetWithSKUs(ctx context.Context, productID uuid.UUID) (*domain.ProductWithSKUs, error)
}

// SKUStore is the driven port for SKU persistence.
// *repository.SKURepository satisfies this interface.
type SKUStore interface {
	// Create persists a new SKU.
	Create(ctx context.Context, s *domain.SKU) error
	// GetByID retrieves a SKU by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SKU, error)
	// List returns all SKUs belonging to the given product.
	List(ctx context.Context, productID uuid.UUID) ([]domain.SKU, error)
	// Update persists changes to an existing SKU.
	Update(ctx context.Context, s *domain.SKU) error
	// UpdateStatus changes the active/archive status of a SKU.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ProductStatus) error
	// BatchGetMappings returns (sku_id, product_id, seller_id) for the given
	// SKU ids. Unknown ids are silently omitted. Used by the BatchGetSKUs RPC
	// so service-to-service callers can snapshot product_id without reading
	// catalog_svc directly.
	BatchGetMappings(ctx context.Context, ids []uuid.UUID) ([]SKUMapping, error)
	// CheapestPrice returns (price_amount, price_currency, true) for the
	// lowest-price SKU of the given product, or (0, "", false) if the
	// product has no SKUs. Used by the outbox publisher to snapshot the
	// representative price onto product.created / product.updated events.
	CheapestPrice(ctx context.Context, productID uuid.UUID) (int64, string, bool, error)
}

// SKUMapping is the minimal (sku_id, product_id, seller_id) triple exposed
// over BatchGetSKUs RPC.
type SKUMapping struct {
	SKUID     uuid.UUID
	ProductID uuid.UUID
	SellerID  uuid.UUID
}

// SKULookup is the shape returned by GetSKUWithProductName, used by the
// cart service to snapshot price and display metadata at add-to-cart time.
// Placed in port so both the app layer and the internal handler share the type.
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
