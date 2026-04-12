package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// OrderUseCase is the driving port (inbound) for order operations.
// Handlers and the gRPC server depend on this interface;
// *service.OrderService satisfies it.
type OrderUseCase interface {
	// CreateOrder creates a single-seller order with an associated Stripe payment intent.
	// Returns the order with its lines and the Stripe client secret for front-end confirmation.
	CreateOrder(ctx context.Context, tenantID uuid.UUID, input domain.CreateOrderInput) (*domain.OrderWithLines, string, error)
	// CreateCheckout processes a multi-seller cart checkout, creating one order per seller
	// under a shared Stripe payment intent, and returns the client secret.
	CreateCheckout(ctx context.Context, tenantID uuid.UUID, input domain.CheckoutInput) (*domain.CheckoutResult, error)
	// HandlePaymentSuccess confirms payment for all orders linked to the Stripe payment intent,
	// triggers inventory sold-confirmation, and initiates seller payouts.
	HandlePaymentSuccess(ctx context.Context, stripePaymentIntentID string) error
	// CheckPurchase reports whether the buyer has an existing paid order for the given SKU from the seller.
	CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*PurchaseCheckResult, error)
	// GetOrder retrieves an order with its line items.
	GetOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error)
	// ListBuyerOrders returns a paginated list of orders placed by the buyer.
	ListBuyerOrders(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error)
	// ListSellerOrders returns a paginated list of orders received by the seller, optionally filtered by status.
	ListSellerOrders(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error)
	// UpdateOrderStatus updates the fulfillment status of an order; only the owning seller may perform this.
	UpdateOrderStatus(ctx context.Context, tenantID, sellerID, orderID uuid.UUID, status string) error
	// ListPayouts returns a paginated list of seller payouts.
	ListPayouts(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error)
	// ListCommissionRules returns a paginated list of platform commission rules.
	ListCommissionRules(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.CommissionRule, int, error)
	// CreateCommissionRule adds a new commission rule for the tenant.
	CreateCommissionRule(ctx context.Context, tenantID uuid.UUID, rule *domain.CommissionRule) error
}
