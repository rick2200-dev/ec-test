package handler

import (
	"encoding/json"
	"time"
)

// orderLineJSON is the shape that order-svc returns today for a line inside
// an OrderWithLines JSON body. Mirrors
// backend/services/order/internal/domain/order.go OrderLine.
type orderLineJSON struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	OrderID     string    `json:"order_id"`
	SKUID       string    `json:"sku_id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	SKUCode     string    `json:"sku_code"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
	LineTotal   int64     `json:"line_total"`
	CreatedAt   time.Time `json:"created_at"`
}

// orderDetailJSON is the shape that order-svc returns for GET /orders/{id}
// (OrderWithLines). The gateway decodes the upstream response into this
// struct and then enriches each line via catalog gRPC before returning.
type orderDetailJSON struct {
	ID                    string          `json:"id"`
	TenantID              string          `json:"tenant_id"`
	SellerID              string          `json:"seller_id"`
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
	Lines                 []orderLineJSON `json:"lines"`
}

// enrichedLineJSON is the order-detail line shape the gateway returns to the
// buyer frontend. It layers product_slug / image_url / is_deleted on top of
// the snapshotted fields from order_lines.
//
// Invariant: product_name is ALWAYS the snapshot captured at checkout time
// — never the current catalog name. This keeps history immutable from the
// buyer's perspective.
type enrichedLineJSON struct {
	ID          string `json:"id"`
	SKUID       string `json:"sku_id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	SKUCode     string `json:"sku_code"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	LineTotal   int64  `json:"line_total"`
	// ImageURL is the current primary product image URL, or empty string
	// if the product has no image or has been deleted/archived.
	ImageURL string `json:"image_url"`
	// ProductSlug is the current product slug (for linking to the product
	// page), or empty string if the product is unavailable.
	ProductSlug string `json:"product_slug"`
	// IsDeleted is true when the catalog product is missing, archived, or
	// fails to load. The frontend uses this to render a "deleted" badge
	// and suppress the product-page link.
	IsDeleted bool `json:"is_deleted"`
}

// orderDetailResponseJSON is the response body for GET /api/v1/buyer/orders/{id}.
// All summary fields from the original order are kept so the frontend does
// not need a second round-trip; lines are replaced with enriched lines.
type orderDetailResponseJSON struct {
	ID                    string             `json:"id"`
	TenantID              string             `json:"tenant_id"`
	SellerID              string             `json:"seller_id"`
	SellerName            string             `json:"seller_name"`
	Status                string             `json:"status"`
	SubtotalAmount        int64              `json:"subtotal_amount"`
	ShippingFee           int64              `json:"shipping_fee"`
	CommissionAmount      int64              `json:"commission_amount"`
	TotalAmount           int64              `json:"total_amount"`
	Currency              string             `json:"currency"`
	ShippingAddress       json.RawMessage    `json:"shipping_address"`
	StripePaymentIntentID *string            `json:"stripe_payment_intent_id,omitempty"`
	PaidAt                *time.Time         `json:"paid_at,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
	Lines                 []enrichedLineJSON `json:"lines"`
}
