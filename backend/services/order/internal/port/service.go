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
	CreateOrder(ctx context.Context, tenantID uuid.UUID, input domain.CreateOrderInput) (*domain.OrderWithLines, string, error)
	CreateCheckout(ctx context.Context, tenantID uuid.UUID, input domain.CheckoutInput) (*domain.CheckoutResult, error)
	HandlePaymentSuccess(ctx context.Context, stripePaymentIntentID string) error
	CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*PurchaseCheckResult, error)
	GetOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error)
	ListBuyerOrders(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error)
	ListSellerOrders(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error)
	UpdateOrderStatus(ctx context.Context, tenantID, sellerID, orderID uuid.UUID, status string) error
	ListPayouts(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error)
	ListCommissionRules(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.CommissionRule, int, error)
	CreateCommissionRule(ctx context.Context, tenantID uuid.UUID, rule *domain.CommissionRule) error
}
