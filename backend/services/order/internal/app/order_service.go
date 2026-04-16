package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	orderv1 "github.com/Riku-KANO/ec-test/services/order/api/gen/go/order/v1"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/port"
)

// OrderService implements order business logic.
type OrderService struct {
	orderRepo          port.OrderStore
	commissionRepo     port.CommissionStore
	payoutRepo         port.PayoutStore
	stripe             port.StripePayments
	publisher          pubsub.Publisher
	buyerSubClient     port.BuyerSubscriptionChecker
	sellerLookup       port.SellerLookup
	skuLookup          port.SKULookup
	defaultShippingFee int64
}

// NewOrderService creates a new OrderService. sellerLookup and skuLookup
// replace the former cross-schema reads of auth_svc.sellers /
// catalog_svc.skus; they are called before each insert transaction to
// resolve the seller_name + product_id snapshots that the order row
// persists. See Phase 1.2 in the refactor plan.
func NewOrderService(
	orderRepo port.OrderStore,
	commissionRepo port.CommissionStore,
	payoutRepo port.PayoutStore,
	stripe port.StripePayments,
	publisher pubsub.Publisher,
	buyerSubClient port.BuyerSubscriptionChecker,
	sellerLookup port.SellerLookup,
	skuLookup port.SKULookup,
	defaultShippingFee int64,
) *OrderService {
	return &OrderService{
		orderRepo:          orderRepo,
		commissionRepo:     commissionRepo,
		payoutRepo:         payoutRepo,
		stripe:             stripe,
		publisher:          publisher,
		buyerSubClient:     buyerSubClient,
		sellerLookup:       sellerLookup,
		skuLookup:          skuLookup,
		defaultShippingFee: defaultShippingFee,
	}
}

// CreateOrder creates a new order with Stripe PaymentIntent.
func (s *OrderService) CreateOrder(ctx context.Context, input domain.CreateOrderInput) (*domain.OrderWithLines, string, error) {
	if len(input.Lines) == 0 {
		return nil, "", domain.ErrEmptyOrder
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
	rule, err := s.commissionRepo.GetApplicableRule(ctx, input.SellerID, nil)
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
	if hasFree, err := s.buyerSubClient.HasFreeShipping(ctx, input.BuyerAuth0ID); err != nil {
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

	// 7. Resolve seller_name + product_id snapshots from the owning
	// services. These used to be cross-schema SELECTs in the order repo;
	// Phase 1.2 moved the lookups here so order_svc no longer needs GRANT
	// on auth_svc / catalog_svc.
	sellerNames, err := s.sellerLookup.BatchGetSellerNames(ctx, []uuid.UUID{input.SellerID})
	if err != nil {
		return nil, "", apperrors.Internal("resolve seller name", err)
	}
	skuIDs := make([]uuid.UUID, 0, len(lines))
	for _, l := range lines {
		skuIDs = append(skuIDs, l.SKUID)
	}
	productIDs, err := s.skuLookup.BatchGetSKUProductIDs(ctx, skuIDs)
	if err != nil {
		return nil, "", apperrors.Internal("resolve sku product ids", err)
	}
	for i := range lines {
		pid, ok := productIDs[lines[i].SKUID]
		if !ok {
			return nil, "", apperrors.BadRequest(fmt.Sprintf("sku %s has no matching product", lines[i].SKUID))
		}
		lines[i].ProductID = pid
	}

	// 8. Save order + lines to DB.
	order := &domain.Order{
		SellerID:              input.SellerID,
		SellerName:            sellerNames[input.SellerID],
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

	if err := s.orderRepo.Create(ctx, order, lines); err != nil {
		return nil, "", apperrors.Internal("failed to create order", err)
	}

	result := &domain.OrderWithLines{
		Order: *order,
		Lines: lines,
	}

	slog.Info("order created", "order_id", order.ID, "total", totalAmount)

	pubsub.PublishProtoEvent(ctx, s.publisher, domain.EventTypeOrderCreated, "order-events", &orderv1.OrderCreated{
		OrderId:      order.ID.String(),
		SellerId:     order.SellerID.String(),
		BuyerAuth0Id: order.BuyerAuth0ID,
		TotalAmount:  totalAmount,
		Currency:     currency,
	})

	// 7. Return order + Stripe client secret.
	return result, clientSecret, nil
}

// CreateCheckout creates one Order (plus a pending Payout) per seller for a
// multi-seller cart, all inside a single transaction, then issues a single
// Stripe PaymentIntent covering the full cart. The returned orders all share
// the same stripe_payment_intent_id, which is the grouping key the webhook
// handler uses to distribute funds via per-seller Transfers.
func (s *OrderService) CreateCheckout(ctx context.Context, input domain.CheckoutInput) (*domain.CheckoutResult, error) {
	if len(input.Lines) == 0 {
		return nil, domain.ErrEmptyOrder
	}
	if input.BuyerAuth0ID == "" {
		return nil, domain.ErrBuyerRequired
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
			return nil, domain.ErrInvalidQuantity
		}
		g, ok := groups[line.SellerID]
		if !ok {
			g = &sellerGroup{sellerID: line.SellerID}
			groups[line.SellerID] = g
			groupOrder = append(groupOrder, line.SellerID)
		}
		g.lines = append(g.lines, line)
	}

	// 2. Determine shipping fee per order based on buyer subscription.
	shippingFeePerOrder := s.defaultShippingFee
	if hasFree, err := s.buyerSubClient.HasFreeShipping(ctx, input.BuyerAuth0ID); err != nil {
		slog.Warn("failed to check buyer subscription, charging standard shipping", "error", err)
	} else if hasFree {
		shippingFeePerOrder = 0
	}

	// 3. Build an (Order, Lines, Payout) tuple for each seller group and
	//    compute the cart-wide total for the PaymentIntent.
	batch := make([]domain.CheckoutBatchItem, 0, len(groupOrder))
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

		rule, err := s.commissionRepo.GetApplicableRule(ctx, sellerID, nil)
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

		batch = append(batch, domain.CheckoutBatchItem{
			Order:  order,
			Lines:  lines,
			Payout: payout,
		})
	}

	// 4. Resolve seller_name + product_id snapshots via the owning services'
	// RPCs before writing. Replaces former cross-schema reads. See Phase 1.2.
	sellerIDSet := make(map[uuid.UUID]struct{}, len(batch))
	skuIDSet := make(map[uuid.UUID]struct{})
	for _, item := range batch {
		sellerIDSet[item.Order.SellerID] = struct{}{}
		for _, l := range item.Lines {
			skuIDSet[l.SKUID] = struct{}{}
		}
	}
	sellerIDs := make([]uuid.UUID, 0, len(sellerIDSet))
	for id := range sellerIDSet {
		sellerIDs = append(sellerIDs, id)
	}
	skuIDs := make([]uuid.UUID, 0, len(skuIDSet))
	for id := range skuIDSet {
		skuIDs = append(skuIDs, id)
	}
	sellerNames, err := s.sellerLookup.BatchGetSellerNames(ctx, sellerIDs)
	if err != nil {
		return nil, apperrors.Internal("resolve seller names", err)
	}
	productIDs, err := s.skuLookup.BatchGetSKUProductIDs(ctx, skuIDs)
	if err != nil {
		return nil, apperrors.Internal("resolve sku product ids", err)
	}
	for _, item := range batch {
		item.Order.SellerName = sellerNames[item.Order.SellerID]
		for i := range item.Lines {
			pid, ok := productIDs[item.Lines[i].SKUID]
			if !ok {
				return nil, apperrors.BadRequest(fmt.Sprintf("sku %s has no matching product", item.Lines[i].SKUID))
			}
			item.Lines[i].ProductID = pid
		}
	}

	// 5. Insert all orders + pending payouts atomically.
	if err := s.orderRepo.CreateCheckoutBatch(ctx, batch); err != nil {
		return nil, apperrors.Internal("failed to create checkout batch", err)
	}

	// 5. Create one PaymentIntent on the platform account (Separate Charges
	//    and Transfers). Funds land on the platform; per-seller Transfers
	//    will be created by the webhook once payment succeeds.
	metadata := map[string]string{
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
		if err := s.orderRepo.SetStripePaymentIntentID(ctx, batch[i].Order.ID, piID); err != nil {
			return nil, apperrors.Internal("failed to attach payment intent to order", err)
		}
		batch[i].Order.StripePaymentIntentID = &piID
	}

	// 7. Publish order.created for each order in the checkout.
	for i := range batch {
		o := batch[i].Order
		pubsub.PublishProtoEvent(ctx, s.publisher, domain.EventTypeOrderCreated, "order-events", &orderv1.OrderCreated{
			OrderId:               o.ID.String(),
			SellerId:              o.SellerID.String(),
			BuyerAuth0Id:          o.BuyerAuth0ID,
			TotalAmount:           o.TotalAmount,
			Currency:              o.Currency,
			StripePaymentIntentId: piID,
		})
	}

	slog.Info("checkout created",
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

		// 1. Mark the order paid. Guarded on status='pending' — see
		//    repository.ErrOrderNotPending. This is the P1 guardrail
		//    against a redelivered payment_intent.succeeded webhook
		//    reverting a cancelled order back to paid (and therefore
		//    triggering a Stripe Transfer against funds that have
		//    already been refunded and reversed).
		if err := s.orderRepo.SetPaid(ctx, order.ID, now, stripePaymentIntentID); err != nil {
			if errors.Is(err, domain.ErrOrderNotPending) {
				slog.Info("skipping payment success for order that is not in pending status",
					"order_id", order.ID,
					"order_status", order.Status,
					"payment_intent", stripePaymentIntentID,
				)
				continue
			}
			return apperrors.Internal("failed to update order to paid", err)
		}

		// 2. Locate the pending payout we inserted during checkout.
		payout, err := s.payoutRepo.GetByOrderID(ctx, order.ID)
		if err != nil {
			return apperrors.Internal("failed to get payout for order", err)
		}
		if payout == nil {
			slog.Warn("no payout record found for order, skipping transfer",
				"order_id", order.ID, "payment_intent", stripePaymentIntentID)
			continue
		}
		// 2b. Belt-and-braces payout status guard.
		if payout.Status != domain.PayoutStatusPending {
			slog.Info("skipping payment success for non-pending payout",
				"order_id", order.ID,
				"payout_id", payout.ID,
				"payout_status", payout.Status,
				"payment_intent", stripePaymentIntentID,
			)
			continue
		}

		// 3. Resolve the seller's connected Stripe account id.
		sellerStripeAccountID := getSellerStripeAccountID(order.SellerID)

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
			if failErr := s.payoutRepo.UpdateStatus(ctx, payout.ID, domain.PayoutStatusFailed, nil); failErr != nil {
				slog.Error("failed to mark payout failed", "error", failErr, "payout_id", payout.ID)
			}
			pubsub.PublishProtoEvent(ctx, s.publisher, domain.EventTypePayoutFailed, "payout-events", &orderv1.PayoutFailed{
				PayoutId: payout.ID.String(),
				OrderId:  order.ID.String(),
				SellerId: order.SellerID.String(),
				Error:    transferErr.Error(),
			})
			continue
		}

		// 5. Mark the payout completed with the transfer id.
		if err := s.payoutRepo.UpdateStatus(ctx, payout.ID, domain.PayoutStatusCompleted, &transferID); err != nil {
			return apperrors.Internal("failed to mark payout completed", err)
		}

		slog.Info("order marked paid and transfer created",
			"order_id", order.ID,
			"payment_intent", stripePaymentIntentID,
			"transfer_id", transferID,
		)

		// 6. Publish order.paid and payout.completed. Line items are
		// attached so recommend can record one purchased event per product
		// rather than falling back to the useless order-keyed signal.
		lineItems, linesErr := s.loadPaidLineItems(ctx, order.ID)
		if linesErr != nil {
			slog.Warn("failed to load line items for order.paid event; publishing without them",
				"order_id", order.ID, "error", linesErr)
		}
		pubsub.PublishProtoEvent(ctx, s.publisher, domain.EventTypeOrderPaid, "order-events", &orderv1.OrderPaid{
			OrderId:               order.ID.String(),
			SellerId:              order.SellerID.String(),
			BuyerAuth0Id:          order.BuyerAuth0ID,
			TotalAmount:           order.TotalAmount,
			StripePaymentIntentId: stripePaymentIntentID,
			ShippingAddressJson:   string(order.ShippingAddress),
			LineItems:             lineItems,
		})
		pubsub.PublishProtoEvent(ctx, s.publisher, domain.EventTypePayoutCompleted, "payout-events", &orderv1.PayoutCompleted{
			PayoutId:         payout.ID.String(),
			OrderId:          order.ID.String(),
			SellerId:         order.SellerID.String(),
			Amount:           payout.Amount,
			Currency:         payout.Currency,
			StripeTransferId: transferID,
		})
	}

	return nil
}

// getSellerStripeAccountID returns the seller's Stripe connected account id.
// This is currently a stub that synthesizes an id from the seller uuid. The
// real implementation must look up sellers.stripe_account_id via the auth
// service (or a shared seller lookup API). Tracked as a known limitation in
// docs/payment.md.
func getSellerStripeAccountID(sellerID uuid.UUID) string {
	return "acct_stub_" + sellerID.String()
}

// loadPaidLineItems fetches the order lines for the given order and returns
// the slim per-line snapshot attached to order.paid events so downstream
// consumers (recommend, analytics) can attribute purchases to products
// without calling back to the order service.
func (s *OrderService) loadPaidLineItems(ctx context.Context, orderID uuid.UUID) ([]*orderv1.PaidLineItem, error) {
	owl, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("load order lines: %w", err)
	}
	if owl == nil {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	out := make([]*orderv1.PaidLineItem, 0, len(owl.Lines))
	for _, l := range owl.Lines {
		out = append(out, &orderv1.PaidLineItem{
			ProductId: l.ProductID.String(),
			SkuId:     l.SKUID.String(),
			Quantity:  int32(l.Quantity),
		})
	}
	return out, nil
}

// CheckPurchase verifies whether the given buyer has a paid-or-later order
// containing the given SKU from the given seller. Used by the inquiry service
// before allowing a buyer to open a new thread. Failure to find a matching
// purchase is NOT an error — it returns Purchased=false so the caller can
// respond with 403 to the buyer.
func (s *OrderService) CheckPurchase(ctx context.Context, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*port.PurchaseCheckResult, error) {
	if buyerAuth0ID == "" {
		return nil, domain.ErrBuyerRequired
	}
	rec, err := s.orderRepo.HasPurchasedSKU(ctx, buyerAuth0ID, sellerID, skuID)
	if err != nil {
		return nil, apperrors.Internal("failed to check purchase", err)
	}
	if rec == nil {
		return &port.PurchaseCheckResult{Purchased: false}, nil
	}
	return &port.PurchaseCheckResult{
		Purchased:       true,
		EarliestOrderID: rec.OrderID,
		ProductName:     rec.ProductName,
		SKUCode:         rec.SKUCode,
	}, nil
}

// GetOrder retrieves an order with its lines.
func (s *OrderService) GetOrder(ctx context.Context, orderID uuid.UUID) (*domain.OrderWithLines, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, apperrors.Internal("failed to get order", err)
	}
	if order == nil {
		return nil, domain.ErrOrderNotFound
	}
	return order, nil
}

// ListBuyerOrders returns paginated orders for a buyer.
func (s *OrderService) ListBuyerOrders(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error) {
	orders, total, err := s.orderRepo.ListByBuyer(ctx, buyerAuth0ID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list buyer orders", err)
	}
	return orders, total, nil
}

// ListSellerOrders returns paginated orders for a seller.
func (s *OrderService) ListSellerOrders(ctx context.Context, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error) {
	orders, total, err := s.orderRepo.ListBySeller(ctx, sellerID, status, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list seller orders", err)
	}
	return orders, total, nil
}

// UpdateOrderStatus updates the status of an order (for seller: processing,
// shipped, delivered, etc.).
//
// Seller ownership is enforced BEFORE any write: the handler passes the
// authenticated seller id, and this method fetches the order and
// verifies seller_id matches before delegating to the repository.
// Mismatches return 404 (not 403) to avoid leaking the existence of
// another seller's order.
func (s *OrderService) UpdateOrderStatus(ctx context.Context, sellerID, orderID uuid.UUID, status string) error {
	// Validate allowed status transitions.
	switch status {
	case domain.StatusProcessing, domain.StatusShipped, domain.StatusDelivered, domain.StatusCompleted, domain.StatusCancelled:
		// valid
	default:
		return fmt.Errorf("%w: %s", domain.ErrInvalidOrderStatus, status)
	}

	// Ownership check: load the order and confirm this seller owns it
	// before the repository UPDATE.
	existing, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return apperrors.Internal("failed to load order", err)
	}
	if existing == nil {
		return domain.ErrOrderNotFound
	}
	if existing.SellerID != sellerID {
		return domain.ErrOrderNotFound
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, status); err != nil {
		return apperrors.Internal("failed to update order status", err)
	}

	if status == domain.StatusShipped {
		pubsub.PublishProtoEvent(ctx, s.publisher, domain.EventTypeOrderShipped, "order-events", &orderv1.OrderShipped{
			OrderId: orderID.String(),
		})
	}

	return nil
}

// ListPayouts returns paginated payouts for a seller.
func (s *OrderService) ListPayouts(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error) {
	payouts, total, err := s.payoutRepo.ListBySeller(ctx, sellerID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list payouts", err)
	}
	return payouts, total, nil
}

// ListCommissionRules returns paginated commission rules.
func (s *OrderService) ListCommissionRules(ctx context.Context, limit, offset int) ([]domain.CommissionRule, int, error) {
	rules, total, err := s.commissionRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list commission rules", err)
	}
	return rules, total, nil
}

// CreateCommissionRule creates a new commission rule.
func (s *OrderService) CreateCommissionRule(ctx context.Context, rule *domain.CommissionRule) error {
	if err := s.commissionRepo.Create(ctx, rule); err != nil {
		return apperrors.Internal("failed to create commission rule", err)
	}
	return nil
}
