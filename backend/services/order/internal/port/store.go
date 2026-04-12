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
	Create(ctx context.Context, tenantID uuid.UUID, order *domain.Order, lines []domain.OrderLine) error
	CreateCheckoutBatch(ctx context.Context, tenantID uuid.UUID, items []domain.CheckoutBatchItem) error
	GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error)
	HasPurchasedSKU(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*domain.PurchaseSKURecord, error)
	ListByBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error)
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error)
	UpdateStatus(ctx context.Context, tenantID, orderID uuid.UUID, status string) error
	SetPaid(ctx context.Context, tenantID, orderID uuid.UUID, paidAt time.Time, stripePaymentIntentID string) error
	FindAllByStripePaymentIntentID(ctx context.Context, paymentIntentID string) ([]domain.Order, error)
	SetStripePaymentIntentID(ctx context.Context, tenantID, orderID uuid.UUID, paymentIntentID string) error
}

// CommissionStore is the driven port for commission rule persistence.
// *repository.CommissionRepository satisfies this interface.
type CommissionStore interface {
	GetApplicableRule(ctx context.Context, tenantID, sellerID uuid.UUID, categoryID *uuid.UUID) (*domain.CommissionRule, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.CommissionRule, int, error)
	Create(ctx context.Context, tenantID uuid.UUID, rule *domain.CommissionRule) error
}

// PayoutStore is the driven port for payout persistence.
// *repository.PayoutRepository satisfies this interface.
type PayoutStore interface {
	GetByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Payout, error)
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error)
	UpdateStatus(ctx context.Context, tenantID, payoutID uuid.UUID, status string, stripeTransferID *string) error
}

// StripePayments is the Stripe client driven port.
// *stripe.Client satisfies this interface.
type StripePayments interface {
	//nolint:staticcheck // SA1019: legacy single-seller path intentionally uses the deprecated Destination Charges API
	CreatePaymentIntent(amount int64, currency string, sellerStripeAccountID string, metadata map[string]string) (paymentIntentID, clientSecret string, err error)
	CreatePlatformPaymentIntent(amount int64, currency string, metadata map[string]string) (paymentIntentID, clientSecret string, err error)
	CreateTransfer(amount int64, currency string, sellerStripeAccountID string, paymentIntentID string) (transferID string, err error)
}

// BuyerSubscriptionChecker is the driven port for buyer subscription lookups.
// *httpclient.BuyerSubscriptionClient satisfies this interface.
type BuyerSubscriptionChecker interface {
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
