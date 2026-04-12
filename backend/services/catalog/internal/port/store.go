// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the catalog service.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
)

// CategoryStore is the driven port for category persistence.
// *repository.CategoryRepository satisfies this interface.
type CategoryStore interface {
	Create(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Category, error)
	GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Category, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.Category, error)
	Update(ctx context.Context, tenantID uuid.UUID, c *domain.Category) error
}

// ProductStore is the driven port for product persistence.
// *repository.ProductRepository satisfies this interface.
type ProductStore interface {
	Create(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error)
	GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Product, error)
	GetWithSKUsBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.ProductWithSKUs, error)
	List(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error
}

// SKUStore is the driven port for SKU persistence.
// *repository.SKURepository satisfies this interface.
type SKUStore interface {
	Create(ctx context.Context, tenantID uuid.UUID, s *domain.SKU) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SKU, error)
	List(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.SKU, error)
	Update(ctx context.Context, tenantID uuid.UUID, s *domain.SKU) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.ProductStatus) error
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
