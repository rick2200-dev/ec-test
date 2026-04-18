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
	// PayoutStatusReversed is set when a seller-approved cancellation
	// request has triggered a Stripe Transfer Reversal against this payout.
	PayoutStatusReversed = "reversed"
)

// Order represents a marketplace order.
//
// CouponDiscountAmount / PointDiscountAmount are the per-order shares of
// the coupon and point redemptions applied at checkout. They are stamped
// when the order is created and never mutate afterwards. total_amount =
// subtotal + shipping_fee - coupon_discount - point_discount.
//
// CouponReservationID / PointReservationID are the opaque handles the
// order service holds onto between checkout and Stripe webhook. Only
// the "anchor" order within a multi-seller cart carries the reservation
// IDs — they point to a single cart-wide reservation in coupon-svc /
// loyalty-svc. Commit / Release go against those handles.
type Order struct {
	ID                    uuid.UUID       `json:"id"`
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
	CancelledAt           *time.Time      `json:"cancelled_at,omitempty"`
	CancellationReason    *string         `json:"cancellation_reason,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`

	CouponDiscountAmount int64      `json:"coupon_discount_amount"`
	PointDiscountAmount  int64      `json:"point_discount_amount"`
	CouponID             *uuid.UUID `json:"coupon_id,omitempty"`
	CouponReservationID  *uuid.UUID `json:"coupon_reservation_id,omitempty"`
	PointReservationID   *uuid.UUID `json:"point_reservation_id,omitempty"`
	PointsEarned         int64      `json:"points_earned"`

	// PaidEventPublishedAt is the once-per-order guard for emitting
	// `order.paid` + `payout.completed` to Pub/Sub. Stamped by
	// HandlePaymentSuccess the first time the publish succeeds, then
	// checked on every retry so a webhook replay doesn't double-emit
	// but also doesn't drop the events when Commit failures force the
	// handler to return before reaching the original publish site.
	PaidEventPublishedAt *time.Time `json:"paid_event_published_at,omitempty"`
}

// TotalDiscount returns the sum of coupon and point discounts applied.
// Useful for event payloads and analytics; the individual fields stay
// separate on the order row so audits can tell the two apart.
func (o *Order) TotalDiscount() int64 { return o.CouponDiscountAmount + o.PointDiscountAmount }

// OrderLine represents a line item within an order.
type OrderLine struct {
	ID          uuid.UUID `json:"id"`
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
	SellerID         uuid.UUID  `json:"seller_id"`
	OrderID          uuid.UUID  `json:"order_id"`
	Amount           int64      `json:"amount"`
	Currency         string     `json:"currency"`
	StripeTransferID *string    `json:"stripe_transfer_id,omitempty"`
	StripeReversalID *string    `json:"stripe_reversal_id,omitempty"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	ReversedAt       *time.Time `json:"reversed_at,omitempty"`
}

// CreateOrderInput holds the data needed to create a new order.
type CreateOrderInput struct {
	SellerID        uuid.UUID        `json:"seller_id"`
	BuyerAuth0ID    string           `json:"buyer_auth0_id"`
	Lines           []OrderLineInput `json:"lines"`
	ShippingAddress json.RawMessage  `json:"shipping_address"`
	Currency        string           `json:"currency"`
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
//
// CouponCode and PointsToRedeem are optional — feature flags on the
// order service env decide whether they are honored. When off, non-empty
// values are rejected with 400 so clients see a clear failure rather
// than a silent drop.
type CheckoutInput struct {
	BuyerAuth0ID    string              `json:"buyer_auth0_id"`
	Lines           []CheckoutLineInput `json:"lines"`
	ShippingAddress json.RawMessage     `json:"shipping_address"`
	Currency        string              `json:"currency"`
	CouponCode      string              `json:"coupon_code,omitempty"`
	PointsToRedeem  int64               `json:"points_to_redeem,omitempty"`
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
	Orders                  []OrderWithLines `json:"orders"`
	StripeClientSecret      string           `json:"stripe_client_secret"`
	StripePaymentIntentID   string           `json:"stripe_payment_intent_id"`
	TotalAmount             int64            `json:"total_amount"`
	Currency                string           `json:"currency"`
	SubtotalBeforeDiscounts int64            `json:"subtotal_before_discounts"`
	CouponDiscountAmount    int64            `json:"coupon_discount_amount"`
	PointDiscountAmount     int64            `json:"point_discount_amount"`
	AppliedCouponCode       string           `json:"applied_coupon_code,omitempty"`
}

// CheckoutBatchItem is one (order, lines, payout) tuple for a single seller
// inside a multi-seller checkout. Used by OrderStore.CreateCheckoutBatch to
// insert all items atomically in a single tenant transaction.
type CheckoutBatchItem struct {
	Order  *Order
	Lines  []OrderLine
	Payout *Payout
}

// CanBeCancelled reports whether the order's current status allows a
// cancellation request to be opened or approved. This is the Go-side
// pre-check; the repository SQL guard uses cancellableStatuses() (in the
// cancellation package) to re-enforce the same constraint atomically.
func (o *Order) CanBeCancelled() bool {
	switch o.Status {
	case StatusPending, StatusPaid, StatusProcessing:
		return true
	}
	return false
}

// PurchaseSKURecord is the minimal information returned by OrderStore.HasPurchasedSKU
// when a matching purchase exists. It carries the earliest paid order plus the
// product/SKU snapshot captured on that order line so callers (inquiry service)
// can seed a new thread without a separate catalog lookup.
type PurchaseSKURecord struct {
	OrderID     uuid.UUID
	ProductName string
	SKUCode     string
}
