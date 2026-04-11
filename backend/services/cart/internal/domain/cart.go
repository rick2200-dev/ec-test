package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Cart is the buyer's shopping cart, persisted in Redis as a JSON blob.
// A cart belongs to a single (tenant_id, buyer_auth0_id) pair and can
// contain items from multiple sellers (Amazon-style multi-seller cart).
type Cart struct {
	TenantID     uuid.UUID  `json:"tenant_id"`
	BuyerAuth0ID string     `json:"buyer_auth0_id"`
	Items        []CartItem `json:"items"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CartItem is a single SKU in the cart. Price and name snapshots are
// captured at add-to-cart time so the buyer sees a stable price while
// browsing. Checkout re-validates against the catalog.
type CartItem struct {
	SKUID               uuid.UUID `json:"sku_id"`
	SellerID            uuid.UUID `json:"seller_id"`
	Quantity            int       `json:"quantity"`
	UnitPriceSnapshot   int64     `json:"unit_price_snapshot"`
	Currency            string    `json:"currency"`
	ProductNameSnapshot string    `json:"product_name_snapshot"`
	SKUCodeSnapshot     string    `json:"sku_code_snapshot"`
	AddedAt             time.Time `json:"added_at"`
}

// Total returns the subtotal of all items in the cart's currency.
// For multi-currency carts this returns the raw sum and should not be
// used directly — use per-seller groupings instead.
func (c *Cart) Total() int64 {
	var sum int64
	for _, item := range c.Items {
		sum += item.UnitPriceSnapshot * int64(item.Quantity)
	}
	return sum
}

// IsEmpty reports whether the cart has no items.
func (c *Cart) IsEmpty() bool {
	return len(c.Items) == 0
}

// FindItem returns the index of an item by SKU, or -1 if not present.
func (c *Cart) FindItem(skuID uuid.UUID) int {
	for i, item := range c.Items {
		if item.SKUID == skuID {
			return i
		}
	}
	return -1
}

// CheckoutInput is the payload passed to the order service to create
// orders from a cart. ShippingAddress is opaque JSON forwarded verbatim.
type CheckoutInput struct {
	BuyerAuth0ID        string          `json:"buyer_auth0_id"`
	Currency            string          `json:"currency"`
	ShippingAddressJSON json.RawMessage `json:"shipping_address"`
	Lines               []CheckoutLine  `json:"lines"`
}

// CheckoutLine carries one SKU across the cart→order boundary, including
// the price snapshot that the order service will re-validate.
type CheckoutLine struct {
	SKUID               uuid.UUID `json:"sku_id"`
	SellerID            uuid.UUID `json:"seller_id"`
	Quantity            int       `json:"quantity"`
	UnitPriceSnapshot   int64     `json:"unit_price_snapshot"`
	ProductNameSnapshot string    `json:"product_name_snapshot"`
	SKUCodeSnapshot     string    `json:"sku_code_snapshot"`
}

// CheckoutResult is what the order service returns from /internal/checkouts.
type CheckoutResult struct {
	OrderIDs              []uuid.UUID `json:"order_ids"`
	StripeClientSecret    string      `json:"stripe_client_secret"`
	StripePaymentIntentID string      `json:"stripe_payment_intent_id"`
	TotalAmount           int64       `json:"total_amount"`
	Currency              string      `json:"currency"`
}
