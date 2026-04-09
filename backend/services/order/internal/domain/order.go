package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Order status constants.
const (
	StatusPending    = "pending"
	StatusPaid       = "paid"
	StatusProcessing = "processing"
	StatusShipped    = "shipped"
	StatusDelivered  = "delivered"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"
)

// Order represents a marketplace order.
type Order struct {
	ID                    uuid.UUID       `json:"id"`
	TenantID              uuid.UUID       `json:"tenant_id"`
	SellerID              uuid.UUID       `json:"seller_id"`
	BuyerAuth0ID          string          `json:"buyer_auth0_id"`
	Status                string          `json:"status"`
	SubtotalAmount        int64           `json:"subtotal_amount"`
	ShippingFee           int64           `json:"shipping_fee"`
	CommissionAmount      int64           `json:"commission_amount"`
	TotalAmount           int64           `json:"total_amount"`
	Currency              string          `json:"currency"`
	ShippingAddress       json.RawMessage `json:"shipping_address"`
	StripePaymentIntentID *string         `json:"stripe_payment_intent_id,omitempty"`
	PaidAt                *time.Time      `json:"paid_at,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// OrderLine represents a line item within an order.
type OrderLine struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	OrderID     uuid.UUID `json:"order_id"`
	SKUID       uuid.UUID `json:"sku_id"`
	ProductName string    `json:"product_name"`
	SKUCode     string    `json:"sku_code"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
	LineTotal   int64     `json:"line_total"`
	CreatedAt   time.Time `json:"created_at"`
}

// OrderWithLines embeds an Order along with its line items.
type OrderWithLines struct {
	Order
	Lines []OrderLine `json:"lines"`
}

// CommissionRule defines how commission is calculated for a seller/category.
type CommissionRule struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	SellerID   *uuid.UUID `json:"seller_id,omitempty"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	RateBps    int        `json:"rate_bps"`
	Priority   int        `json:"priority"`
	ValidFrom  time.Time  `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Payout represents a payout to a seller.
type Payout struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	SellerID         uuid.UUID  `json:"seller_id"`
	OrderID          uuid.UUID  `json:"order_id"`
	Amount           int64      `json:"amount"`
	Currency         string     `json:"currency"`
	StripeTransferID *string    `json:"stripe_transfer_id,omitempty"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// CreateOrderInput holds the data needed to create a new order.
type CreateOrderInput struct {
	SellerID        uuid.UUID       `json:"seller_id"`
	BuyerAuth0ID    string          `json:"buyer_auth0_id"`
	Lines           []OrderLineInput `json:"lines"`
	ShippingAddress json.RawMessage `json:"shipping_address"`
	Currency        string          `json:"currency"`
}

// OrderLineInput holds the data for a single line when creating an order.
type OrderLineInput struct {
	SKUID       uuid.UUID `json:"sku_id"`
	ProductName string    `json:"product_name"`
	SKUCode     string    `json:"sku_code"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
}
