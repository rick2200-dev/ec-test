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
	shippingFee := s.defaultShippingFee
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
	// Legacy single-seller path deliberately uses the deprecated Destination
	// Charges API; multi-seller checkouts go through CreateCheckout, which
	// uses CreatePlatformPaymentIntent + CreateTransfer instead.
	//nolint:staticcheck // SA1019: legacy single-seller CreateOrder intentionally uses the deprecated Destination Charges API
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

// CreateCheckout creates one Order (plus a pending Payout) per seller for a
// multi-seller cart, all inside a single tenant transaction, then issues a
// single Stripe PaymentIntent covering the full cart. The returned orders
// all share the same stripe_payment_intent_id, which is the grouping key
// the webhook handler uses to distribute funds via per-seller Transfers.
func (s *OrderService) CreateCheckout(ctx context.Context, tenantID uuid.UUID, input domain.CheckoutInput) (*domain.CheckoutResult, error) {
	if len(input.Lines) == 0 {
		return nil, apperrors.BadRequest("checkout must have at least one line item")
	}
	if input.BuyerAuth0ID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id is required")
	}

	currency := input.Currency
	if currency == "" {
		currency = "jpy"
	}

	// 1. Group lines by seller_id, preserving input order so the output is
	//    deterministic (needed for tests and predictable user-facing lists).
	type sellerGroup struct {
		sellerID uuid.UUID
		lines    []domain.CheckoutLineInput
	}
	var groupOrder []uuid.UUID
	groups := make(map[uuid.UUID]*sellerGroup)
	for _, line := range input.Lines {
		if line.Quantity <= 0 {
			return nil, apperrors.BadRequest("quantity must be positive")
		}
		g, ok := groups[line.SellerID]
		if !ok {
			g = &sellerGroup{sellerID: line.SellerID}
			groups[line.SellerID] = g
			groupOrder = append(groupOrder, line.SellerID)
		}
		g.lines = append(g.lines, line)
	}

	// 2. Determine shipping fee per order based on buyer subscription. The
	//    fee is charged once per order (one per seller); premium buyers get
	//    free shipping on every order in the checkout.
	shippingFeePerOrder := s.defaultShippingFee
	if hasFree, err := s.buyerSubClient.HasFreeShipping(ctx, tenantID, input.BuyerAuth0ID); err != nil {
		slog.Warn("failed to check buyer subscription, charging standard shipping", "error", err)
	} else if hasFree {
		shippingFeePerOrder = 0
	}

	// 3. Build an (Order, Lines, Payout) tuple for each seller group and
	//    compute the cart-wide total for the PaymentIntent.
	batch := make([]repository.CheckoutBatchItem, 0, len(groupOrder))
	var cartTotal int64
	for _, sellerID := range groupOrder {
		group := groups[sellerID]

		var subtotal int64
		lines := make([]domain.OrderLine, 0, len(group.lines))
		for _, li := range group.lines {
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

		rule, err := s.commissionRepo.GetApplicableRule(ctx, tenantID, sellerID, nil)
		if err != nil {
			return nil, apperrors.Internal("failed to get commission rule", err)
		}
		var commissionAmount int64
		if rule != nil {
			commissionAmount = subtotal * int64(rule.RateBps) / 10000
		}

		orderTotal := subtotal + shippingFeePerOrder
		cartTotal += orderTotal

		order := &domain.Order{
			SellerID:         sellerID,
			BuyerAuth0ID:     input.BuyerAuth0ID,
			Status:           domain.StatusPending,
			SubtotalAmount:   subtotal,
			ShippingFee:      shippingFeePerOrder,
			CommissionAmount: commissionAmount,
			TotalAmount:      orderTotal,
			Currency:         currency,
			ShippingAddress:  input.ShippingAddress,
			// StripePaymentIntentID is stamped after Stripe call below.
		}

		payout := &domain.Payout{
			SellerID: sellerID,
			Amount:   subtotal - commissionAmount,
			Currency: currency,
		}

		batch = append(batch, repository.CheckoutBatchItem{
			Order:  order,
			Lines:  lines,
			Payout: payout,
		})
	}

	// 4. Insert all orders + pending payouts atomically.
	if err := s.orderRepo.CreateCheckoutBatch(ctx, tenantID, batch); err != nil {
		return nil, apperrors.Internal("failed to create checkout batch", err)
	}

	// 5. Create one PaymentIntent on the platform account (Separate Charges
	//    and Transfers). Funds land on the platform; per-seller Transfers
	//    will be created by the webhook once payment succeeds.
	metadata := map[string]string{
		"tenant_id":      tenantID.String(),
		"buyer_auth0_id": input.BuyerAuth0ID,
		"order_count":    fmt.Sprintf("%d", len(batch)),
	}
	piID, clientSecret, err := s.stripe.CreatePlatformPaymentIntent(cartTotal, currency, metadata)
	if err != nil {
		// Orders were already inserted; without a PI they are unusable.
		// The caller should retry checkout; stale pending orders will be
		// cleaned up by a future reaper or surface to the buyer as pending.
		return nil, apperrors.Internal("failed to create payment intent", err)
	}

	// 6. Stamp the PI id on every order we just created.
	for i := range batch {
		if err := s.orderRepo.SetStripePaymentIntentID(ctx, tenantID, batch[i].Order.ID, piID); err != nil {
			return nil, apperrors.Internal("failed to attach payment intent to order", err)
		}
		batch[i].Order.StripePaymentIntentID = &piID
	}

	// 7. Publish order.created for each order in the checkout.
	for i := range batch {
		o := batch[i].Order
		pubsub.PublishEvent(ctx, s.publisher, tenantID, "order.created", "order-events", map[string]any{
			"order_id":                 o.ID.String(),
			"seller_id":                o.SellerID.String(),
			"buyer_auth0_id":           o.BuyerAuth0ID,
			"total_amount":             o.TotalAmount,
			"currency":                 o.Currency,
			"stripe_payment_intent_id": piID,
		})
	}

	slog.Info("checkout created",
		"tenant_id", tenantID,
		"buyer_auth0_id", input.BuyerAuth0ID,
		"order_count", len(batch),
		"total", cartTotal,
		"payment_intent", piID,
	)

	// 8. Shape the return value.
	result := &domain.CheckoutResult{
		Orders:                make([]domain.OrderWithLines, 0, len(batch)),
		StripeClientSecret:    clientSecret,
		StripePaymentIntentID: piID,
		TotalAmount:           cartTotal,
		Currency:              currency,
	}
	for i := range batch {
		result.Orders = append(result.Orders, domain.OrderWithLines{
			Order: *batch[i].Order,
			Lines: batch[i].Lines,
		})
	}
	return result, nil
}

// HandlePaymentSuccess handles a successful payment from Stripe. A single
// PaymentIntent may cover multiple orders (one per seller) from a multi-seller
// checkout, so this iterates over every matching order and creates a Stripe
// Transfer to each seller's connected account.
func (s *OrderService) HandlePaymentSuccess(ctx context.Context, stripePaymentIntentID string) error {
	orders, err := s.orderRepo.FindAllByStripePaymentIntentID(ctx, stripePaymentIntentID)
	if err != nil {
		return apperrors.Internal("failed to find orders by payment intent", err)
	}
	if len(orders) == 0 {
		return apperrors.NotFound("orders not found for payment intent: " + stripePaymentIntentID)
	}

	now := time.Now()
	for i := range orders {
		order := &orders[i]

		// 1. Mark the order paid.
		if err := s.orderRepo.SetPaid(ctx, order.TenantID, order.ID, now, stripePaymentIntentID); err != nil {
			return apperrors.Internal("failed to update order to paid", err)
		}

		// 2. Locate the pending payout we inserted during checkout.
		payout, err := s.payoutRepo.GetByOrderID(ctx, order.TenantID, order.ID)
		if err != nil {
			return apperrors.Internal("failed to get payout for order", err)
		}
		if payout == nil {
			slog.Warn("no payout record found for order, skipping transfer",
				"order_id", order.ID, "payment_intent", stripePaymentIntentID)
			continue
		}

		// 3. Resolve the seller's connected Stripe account id. This is a
		//    stub until sellers are onboarded through Stripe Connect; see
		//    docs/payment.md (known limitations) for the real lookup path.
		sellerStripeAccountID := getSellerStripeAccountID(order.TenantID, order.SellerID)

		// 4. Create the Stripe Transfer on the platform-held funds.
		transferID, transferErr := s.stripe.CreateTransfer(
			payout.Amount,
			payout.Currency,
			sellerStripeAccountID,
			stripePaymentIntentID,
		)
		if transferErr != nil {
			slog.Error("failed to create stripe transfer",
				"error", transferErr,
				"order_id", order.ID,
				"payout_id", payout.ID,
				"amount", payout.Amount,
			)
			if failErr := s.payoutRepo.UpdateStatus(ctx, order.TenantID, payout.ID, domain.PayoutStatusFailed, nil); failErr != nil {
				slog.Error("failed to mark payout failed", "error", failErr, "payout_id", payout.ID)
			}
			pubsub.PublishEvent(ctx, s.publisher, order.TenantID, "payout.failed", "payout-events", map[string]any{
				"payout_id": payout.ID.String(),
				"order_id":  order.ID.String(),
				"seller_id": order.SellerID.String(),
				"error":     transferErr.Error(),
			})
			continue
		}

		// 5. Mark the payout completed with the transfer id.
		if err := s.payoutRepo.UpdateStatus(ctx, order.TenantID, payout.ID, domain.PayoutStatusCompleted, &transferID); err != nil {
			return apperrors.Internal("failed to mark payout completed", err)
		}

		slog.Info("order marked paid and transfer created",
			"order_id", order.ID,
			"payment_intent", stripePaymentIntentID,
			"transfer_id", transferID,
		)

		// 6. Publish order.paid and payout.completed.
		pubsub.PublishEvent(ctx, s.publisher, order.TenantID, "order.paid", "order-events", map[string]any{
			"order_id":                 order.ID.String(),
			"seller_id":                order.SellerID.String(),
			"buyer_auth0_id":           order.BuyerAuth0ID,
			"total_amount":             order.TotalAmount,
			"stripe_payment_intent_id": stripePaymentIntentID,
		})
		pubsub.PublishEvent(ctx, s.publisher, order.TenantID, "payout.completed", "payout-events", map[string]any{
			"payout_id":          payout.ID.String(),
			"order_id":           order.ID.String(),
			"seller_id":          order.SellerID.String(),
			"amount":             payout.Amount,
			"currency":           payout.Currency,
			"stripe_transfer_id": transferID,
		})
	}

	return nil
}

// getSellerStripeAccountID returns the seller's Stripe connected account id.
// This is currently a stub that synthesizes an id from the seller uuid. The
// real implementation must look up sellers.stripe_account_id via the auth
// service (or a shared seller lookup API). Tracked as a known limitation in
// docs/payment.md.
func getSellerStripeAccountID(_, sellerID uuid.UUID) string {
	return "acct_stub_" + sellerID.String()
}

// PurchaseCheckResult is returned by CheckPurchase when the caller is asking
// whether a buyer has purchased a specific SKU from a specific seller.
type PurchaseCheckResult struct {
	Purchased       bool      `json:"purchased"`
	EarliestOrderID uuid.UUID `json:"earliest_order_id,omitempty"`
	ProductName     string    `json:"product_name,omitempty"`
	SKUCode         string    `json:"sku_code,omitempty"`
}

// CheckPurchase verifies whether the given buyer has a paid-or-later order
// containing the given SKU from the given seller. Used by the inquiry service
// before allowing a buyer to open a new thread. Failure to find a matching
// purchase is NOT an error — it returns Purchased=false so the caller can
// respond with 403 to the buyer.
func (s *OrderService) CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*PurchaseCheckResult, error) {
	if buyerAuth0ID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id is required")
	}
	rec, err := s.orderRepo.HasPurchasedSKU(ctx, tenantID, buyerAuth0ID, sellerID, skuID)
	if err != nil {
		return nil, apperrors.Internal("failed to check purchase", err)
	}
	if rec == nil {
		return &PurchaseCheckResult{Purchased: false}, nil
	}
	return &PurchaseCheckResult{
		Purchased:       true,
		EarliestOrderID: rec.OrderID,
		ProductName:     rec.ProductName,
		SKUCode:         rec.SKUCode,
	}, nil
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
