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

// Payout status constants.
const (
	PayoutStatusPending   = "pending"
	PayoutStatusCompleted = "completed"
	PayoutStatusFailed    = "failed"
)

// Order represents a marketplace order.
type Order struct {
	ID                    uuid.UUID       `json:"id"`
	TenantID              uuid.UUID       `json:"tenant_id"`
	SellerID              uuid.UUID       `json:"seller_id"`
	SellerName            string          `json:"seller_name"`
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
	ProductID   uuid.UUID `json:"product_id"`
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

// CheckoutInput holds the data needed for a multi-seller checkout. The order
// service groups the flat Lines list by seller_id, creates one Order per
// seller in a single transaction, and then issues a single PaymentIntent
// covering the whole cart.
type CheckoutInput struct {
	BuyerAuth0ID    string              `json:"buyer_auth0_id"`
	Lines           []CheckoutLineInput `json:"lines"`
	ShippingAddress json.RawMessage     `json:"shipping_address"`
	Currency        string              `json:"currency"`
}

// CheckoutLineInput is one line in a checkout request, carrying the seller_id
// so the order service can group by seller, plus price snapshots captured at
// add-to-cart time.
type CheckoutLineInput struct {
	SKUID       uuid.UUID `json:"sku_id"`
	SellerID    uuid.UUID `json:"seller_id"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
	ProductName string    `json:"product_name"`
	SKUCode     string    `json:"sku_code"`
}

// CheckoutResult is what CreateCheckout returns: the created orders (one per
// seller) sharing a single Stripe PaymentIntent.
type CheckoutResult struct {
	Orders                []OrderWithLines `json:"orders"`
	StripeClientSecret    string           `json:"stripe_client_secret"`
	StripePaymentIntentID string           `json:"stripe_payment_intent_id"`
	TotalAmount           int64            `json:"total_amount"`
	Currency              string           `json:"currency"`
}
