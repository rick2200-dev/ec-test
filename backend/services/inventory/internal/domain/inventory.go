package domain

import (
	"time"

	"github.com/google/uuid"
)

// MovementType represents the type of a stock movement.
type MovementType string

const (
	MovementReceived MovementType = "received"
	MovementReserved MovementType = "reserved"
	MovementReleased MovementType = "released"
	MovementSold     MovementType = "sold"
	MovementAdjusted MovementType = "adjusted"
)

// Inventory represents the stock level of a single SKU for a tenant.
type Inventory struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	SKUID             uuid.UUID `json:"sku_id"`
	SellerID          uuid.UUID `json:"seller_id"`
	QuantityAvailable int       `json:"quantity_available"`
	QuantityReserved  int       `json:"quantity_reserved"`
	LowStockThreshold int       `json:"low_stock_threshold"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CancellationLine is a SKU+quantity pair representing one line item in an
// order cancellation. Used by ReleaseForOrderCancellation so the caller
// (the order-cancelled subscriber) does not need to import the repository.
type CancellationLine struct {
	SKUID    uuid.UUID
	Quantity int
}

// StockMovement records a change in stock for auditing.
type StockMovement struct {
	ID            uuid.UUID    `json:"id"`
	TenantID      uuid.UUID    `json:"tenant_id"`
	SKUID         uuid.UUID    `json:"sku_id"`
	MovementType  MovementType `json:"movement_type"`
	Quantity      int          `json:"quantity"`
	ReferenceType string       `json:"reference_type,omitempty"`
	ReferenceID   string       `json:"reference_id,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}
