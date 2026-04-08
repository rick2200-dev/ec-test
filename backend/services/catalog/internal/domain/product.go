package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProductStatus represents the lifecycle state of a product or SKU.
type ProductStatus string

const (
	StatusDraft    ProductStatus = "draft"
	StatusActive   ProductStatus = "active"
	StatusArchived ProductStatus = "archived"
)

// Category represents a product category within a tenant.
type Category struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	SortOrder int        `json:"sort_order"`
	CreatedAt time.Time  `json:"created_at"`
}

// Product represents a product listing within a tenant marketplace.
type Product struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	SellerID    uuid.UUID       `json:"seller_id"`
	CategoryID  *uuid.UUID      `json:"category_id,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description"`
	Status      ProductStatus   `json:"status"`
	Attributes  json.RawMessage `json:"attributes,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// SKU represents a stock-keeping unit (variant) of a product.
type SKU struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	ProductID     uuid.UUID       `json:"product_id"`
	SellerID      uuid.UUID       `json:"seller_id"`
	SKUCode       string          `json:"sku_code"`
	PriceAmount   int64           `json:"price_amount"`
	PriceCurrency string          `json:"price_currency"`
	Attributes    json.RawMessage `json:"attributes,omitempty"`
	Status        ProductStatus   `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// ProductWithSKUs embeds a Product along with its associated SKUs.
type ProductWithSKUs struct {
	Product
	SKUs []SKU `json:"skus"`
}
