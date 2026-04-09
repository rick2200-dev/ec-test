package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/repository"
	stripeClient "github.com/Riku-KANO/ec-test/services/order/internal/stripe"
)

// OrderService implements order business logic.
type OrderService struct {
	orderRepo          *repository.OrderRepository
	commissionRepo     *repository.CommissionRepository
	payoutRepo         *repository.PayoutRepository
	stripe             *stripeClient.Client
	publisher          pubsub.Publisher
	buyerSubClient     *BuyerSubscriptionClient
	defaultShippingFee int64
}

// NewOrderService creates a new OrderService.
func NewOrderService(
	orderRepo *repository.OrderRepository,
	commissionRepo *repository.CommissionRepository,
	payoutRepo *repository.PayoutRepository,
	stripe *stripeClient.Client,
	publisher pubsub.Publisher,
	buyerSubClient *BuyerSubscriptionClient,
	defaultShippingFee int64,
) *OrderService {
	return &OrderService{
		orderRepo:          orderRepo,
		commissionRepo:     commissionRepo,
		payoutRepo:         payoutRepo,
		stripe:             stripe,
		publisher:          publisher,
		buyerSubClient:     buyerSubClient,
		defaultShippingFee: defaultShippingFee,
	}
}

// CreateOrder creates a new order with Stripe PaymentIntent.
func (s *OrderService) CreateOrder(ctx context.Context, tenantID uuid.UUID, input domain.CreateOrderInput) (*domain.OrderWithLines, string, error) {
	if len(input.Lines) == 0 {
		return nil, "", apperrors.BadRequest("order must have at least one line item")
	}

	// 1. Calculate subtotal from lines.
	var subtotal int64
	var lines []domain.OrderLine
	for _, li := range input.Lines {
		lineTotal := li.UnitPrice * int64(li.Quantity)
		subtotal += lineTotal
		lines = append(lines, domain.OrderLine{
			SKUID:       li.SKUID,
			ProductName: li.ProductName,
			SKUCode:     li.SKUCode,
			Quantity:    li.Quantity,
			UnitPrice:   li.UnitPrice,
			LineTotal:   lineTotal,
		})
	}

	// 2. Find applicable commission rule.
	// For MVP, category_id is nil (applies to all categories).
	rule, err := s.commissionRepo.GetApplicableRule(ctx, tenantID, input.SellerID, nil)
	if err != nil {
		return nil, "", apperrors.Internal("failed to get commission rule", err)
	}

	// 3. Calculate commission amount.
	var commissionAmount int64
	if rule != nil {
		commissionAmount = subtotal * int64(rule.RateBps) / 10000
	}

	// 4. Determine shipping fee based on buyer's subscription status.
	var shippingFee int64 = s.defaultShippingFee
	if hasFree, err := s.buyerSubClient.HasFreeShipping(ctx, tenantID, input.BuyerAuth0ID); err != nil {
		slog.Warn("failed to check buyer subscription, charging standard shipping", "error", err)
	} else if hasFree {
		shippingFee = 0
	}

	// 5. Calculate total (buyer pays subtotal + shipping; commission is deducted from seller's share).
	totalAmount := subtotal + shippingFee

	currency := input.Currency
	if currency == "" {
		currency = "jpy"
	}

	// 6. Create Stripe PaymentIntent.
	metadata := map[string]string{
		"tenant_id": tenantID.String(),
		"seller_id": input.SellerID.String(),
	}

	// For MVP, use seller_id as a placeholder for connected account ID.
	// In production, you'd look up the seller's Stripe connected account ID.
	piID, clientSecret, err := s.stripe.CreatePaymentIntent(
		totalAmount,
		currency,
		input.SellerID.String(), // placeholder: should be seller's Stripe connected account ID
		metadata,
	)
	if err != nil {
		return nil, "", apperrors.Internal("failed to create payment intent", err)
	}

	// 7. Save order + lines to DB.
	order := &domain.Order{
		SellerID:              input.SellerID,
		BuyerAuth0ID:          input.BuyerAuth0ID,
		Status:                domain.StatusPending,
		SubtotalAmount:        subtotal,
		ShippingFee:           shippingFee,
		CommissionAmount:      commissionAmount,
		TotalAmount:           totalAmount,
		Currency:              currency,
		ShippingAddress:       input.ShippingAddress,
		StripePaymentIntentID: &piID,
	}

	if err := s.orderRepo.Create(ctx, tenantID, order, lines); err != nil {
		return nil, "", apperrors.Internal("failed to create order", err)
	}

	result := &domain.OrderWithLines{
		Order: *order,
		Lines: lines,
	}

	slog.Info("order created", "order_id", order.ID, "tenant_id", tenantID, "total", totalAmount)

	pubsub.PublishEvent(ctx, s.publisher, tenantID, "order.created", "order-events", map[string]any{
		"order_id":       order.ID.String(),
		"seller_id":      order.SellerID.String(),
		"buyer_auth0_id": order.BuyerAuth0ID,
		"total_amount":   totalAmount,
		"currency":       currency,
	})

	// 7. Return order + Stripe client secret.
	return result, clientSecret, nil
}

// HandlePaymentSuccess handles a successful payment from Stripe.
func (s *OrderService) HandlePaymentSuccess(ctx context.Context, stripePaymentIntentID string) error {
	// Find order by payment intent ID (cross-tenant lookup for webhook).
	order, err := s.orderRepo.FindByStripePaymentIntentID(ctx, stripePaymentIntentID)
	if err != nil {
		return apperrors.Internal("failed to find order by payment intent", err)
	}
	if order == nil {
		return apperrors.NotFound("order not found for payment intent: " + stripePaymentIntentID)
	}

	// Update status to paid.
	now := time.Now()
	if err := s.orderRepo.SetPaid(ctx, order.TenantID, order.ID, now, stripePaymentIntentID); err != nil {
		return apperrors.Internal("failed to update order to paid", err)
	}

	slog.Info("order marked as paid", "order_id", order.ID, "payment_intent", stripePaymentIntentID)

	pubsub.PublishEvent(ctx, s.publisher, order.TenantID, "order.paid", "order-events", map[string]any{
		"order_id":                order.ID.String(),
		"seller_id":               order.SellerID.String(),
		"buyer_auth0_id":          order.BuyerAuth0ID,
		"total_amount":            order.TotalAmount,
		"stripe_payment_intent_id": stripePaymentIntentID,
	})

	return nil
}

// GetOrder retrieves an order with its lines.
func (s *OrderService) GetOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error) {
	order, err := s.orderRepo.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, apperrors.Internal("failed to get order", err)
	}
	if order == nil {
		return nil, apperrors.NotFound("order not found")
	}
	return order, nil
}

// ListBuyerOrders returns paginated orders for a buyer.
func (s *OrderService) ListBuyerOrders(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error) {
	orders, total, err := s.orderRepo.ListByBuyer(ctx, tenantID, buyerAuth0ID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list buyer orders", err)
	}
	return orders, total, nil
}

// ListSellerOrders returns paginated orders for a seller.
func (s *OrderService) ListSellerOrders(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error) {
	orders, total, err := s.orderRepo.ListBySeller(ctx, tenantID, sellerID, status, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list seller orders", err)
	}
	return orders, total, nil
}

// UpdateOrderStatus updates the status of an order (for seller: processing, shipped, delivered).
func (s *OrderService) UpdateOrderStatus(ctx context.Context, tenantID, orderID uuid.UUID, status string) error {
	// Validate allowed status transitions.
	switch status {
	case domain.StatusProcessing, domain.StatusShipped, domain.StatusDelivered, domain.StatusCompleted, domain.StatusCancelled:
		// valid
	default:
		return apperrors.BadRequest(fmt.Sprintf("invalid status: %s", status))
	}

	if err := s.orderRepo.UpdateStatus(ctx, tenantID, orderID, status); err != nil {
		return apperrors.Internal("failed to update order status", err)
	}

	if status == domain.StatusShipped {
		pubsub.PublishEvent(ctx, s.publisher, tenantID, "order.shipped", "order-events", map[string]any{
			"order_id": orderID.String(),
		})
	}

	return nil
}

// ListPayouts returns paginated payouts for a seller.
func (s *OrderService) ListPayouts(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error) {
	payouts, total, err := s.payoutRepo.ListBySeller(ctx, tenantID, sellerID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list payouts", err)
	}
	return payouts, total, nil
}

// ListCommissionRules returns paginated commission rules.
func (s *OrderService) ListCommissionRules(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.CommissionRule, int, error) {
	rules, total, err := s.commissionRepo.List(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list commission rules", err)
	}
	return rules, total, nil
}

// CreateCommissionRule creates a new commission rule.
func (s *OrderService) CreateCommissionRule(ctx context.Context, tenantID uuid.UUID, rule *domain.CommissionRule) error {
	if err := s.commissionRepo.Create(ctx, tenantID, rule); err != nil {
		return apperrors.Internal("failed to create commission rule", err)
	}
	return nil
}
