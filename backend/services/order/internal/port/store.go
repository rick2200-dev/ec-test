// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the order service.
package port

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// OrderStore is the driven port for order persistence.
// *repository.OrderRepository satisfies this interface.
type OrderStore interface {
	// Create persists a new order together with its line items in a single operation.
	Create(ctx context.Context, tenantID uuid.UUID, order *domain.Order, lines []domain.OrderLine) error
	// CreateCheckoutBatch persists multiple orders from a multi-seller cart checkout in one call.
	CreateCheckoutBatch(ctx context.Context, tenantID uuid.UUID, items []domain.CheckoutBatchItem) error
	// GetByID retrieves an order with its line items by order ID within the tenant.
	GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error)
	// HasPurchasedSKU returns the earliest purchase record for the buyer/SKU combination,
	// or nil if no paid order exists.
	HasPurchasedSKU(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*domain.PurchaseSKURecord, error)
	// ListByBuyer returns a paginated list of orders placed by the buyer.
	ListByBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error)
	// ListBySeller returns a paginated list of orders received by the seller, optionally filtered by status.
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error)
	// UpdateStatus sets the fulfillment status of an order (e.g. "pending" → "shipped" → "delivered").
	UpdateStatus(ctx context.Context, tenantID, orderID uuid.UUID, status string) error
	// SetPaid records the payment confirmation timestamp and Stripe payment intent ID for an order.
	SetPaid(ctx context.Context, tenantID, orderID uuid.UUID, paidAt time.Time, stripePaymentIntentID string) error
	// FindAllByStripePaymentIntentID returns all orders associated with the given Stripe payment intent.
	FindAllByStripePaymentIntentID(ctx context.Context, paymentIntentID string) ([]domain.Order, error)
	// SetStripePaymentIntentID links a Stripe payment intent ID to an order before payment is confirmed.
	SetStripePaymentIntentID(ctx context.Context, tenantID, orderID uuid.UUID, paymentIntentID string) error
}

// CommissionStore is the driven port for commission rule persistence.
// *repository.CommissionRepository satisfies this interface.
type CommissionStore interface {
	// GetApplicableRule returns the most specific commission rule that applies to the seller
	// and optional category. Category-scoped rules take precedence over seller-wide rules.
	GetApplicableRule(ctx context.Context, tenantID, sellerID uuid.UUID, categoryID *uuid.UUID) (*domain.CommissionRule, error)
	// List returns a paginated list of all commission rules for the tenant.
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.CommissionRule, int, error)
	// Create persists a new commission rule.
	Create(ctx context.Context, tenantID uuid.UUID, rule *domain.CommissionRule) error
}

// PayoutStore is the driven port for payout persistence.
// *repository.PayoutRepository satisfies this interface.
type PayoutStore interface {
	// GetByOrderID retrieves the payout record associated with the given order.
	GetByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Payout, error)
	// ListBySeller returns a paginated list of payouts for the seller.
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error)
	// UpdateStatus updates the status of a payout and optionally sets the Stripe transfer ID.
	UpdateStatus(ctx context.Context, tenantID, payoutID uuid.UUID, status string, stripeTransferID *string) error
}

// StripePayments is the Stripe client driven port.
// *stripe.Client satisfies this interface.
type StripePayments interface {
	//nolint:staticcheck // SA1019: legacy single-seller path intentionally uses the deprecated Destination Charges API
	// CreatePaymentIntent creates a single-seller payment intent using the Destination Charges API.
	CreatePaymentIntent(amount int64, currency string, sellerStripeAccountID string, metadata map[string]string) (paymentIntentID, clientSecret string, err error)
	// CreatePlatformPaymentIntent creates a platform-level payment intent for multi-seller checkouts.
	// Funds are held on the platform account and transferred to sellers after payment capture.
	CreatePlatformPaymentIntent(amount int64, currency string, metadata map[string]string) (paymentIntentID, clientSecret string, err error)
	// CreateTransfer initiates a Stripe Connect transfer to a seller's account after payment capture.
	CreateTransfer(amount int64, currency string, sellerStripeAccountID string, paymentIntentID string) (transferID string, err error)
}

// BuyerSubscriptionChecker is the driven port for buyer subscription lookups.
// *httpclient.BuyerSubscriptionClient satisfies this interface.
type BuyerSubscriptionChecker interface {
	// HasFreeShipping reports whether the buyer currently holds an active plan that grants free shipping.
	HasFreeShipping(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (bool, error)
}

// PurchaseCheckResult is returned by CheckPurchase. Placed in port so both
// the app layer (order_service.CheckPurchase) and the handler can share the
// type without a circular import.
type PurchaseCheckResult struct {
	Purchased       bool      `json:"purchased"`
	EarliestOrderID uuid.UUID `json:"earliest_order_id,omitempty"`
	ProductName     string    `json:"product_name,omitempty"`
	SKUCode         string    `json:"sku_code,omitempty"`
}
