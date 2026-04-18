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

// publishClaimTTL caps how long a HandlePaymentSuccess pass can hold
// the paid-event publish slot before another concurrent delivery may
// re-acquire it. Must comfortably exceed the 95th-percentile publish
// latency (Pub/Sub publish + marshal + DB roundtrip). Two minutes is
// generous for the current footprint and still fast enough that a
// crashed publisher doesn't block downstream consumers for long.
const publishClaimTTL = 2 * time.Minute

// OrderService implements order business logic.
//
// couponReserver / pointReserver / enableCoupons / enableLoyalty are
// the discount integration seams added alongside the coupon + loyalty
// MVP. Nil reservers combined with the feature flags off keeps the
// service usable in environments where either subsystem is not yet
// deployed — the non-nil check short-circuits cleanly so tests and
// local dev don't need stubs.
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

	couponReserver port.CouponReserver
	pointReserver  port.PointReserver
	enableCoupons  bool
	enableLoyalty  bool
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

// WithCouponReserver injects the coupon-svc client and turns the
// coupon feature on. No-op if r is nil.
func (s *OrderService) WithCouponReserver(r port.CouponReserver, enabled bool) *OrderService {
	s.couponReserver = r
	s.enableCoupons = enabled && r != nil
	return s
}

// WithPointReserver injects the loyalty-svc client and turns the
// loyalty feature on. Enables the earn-on-paid path independently of
// the redeem-at-checkout path, which may not be wired yet.
func (s *OrderService) WithPointReserver(r port.PointReserver, enabled bool) *OrderService {
	s.pointReserver = r
	s.enableLoyalty = enabled && r != nil
	return s
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

	// Feature-flag gating. A disabled feature with a non-empty input
	// fails fast rather than silently dropping the discount.
	if input.CouponCode != "" && !s.enableCoupons {
		return nil, domain.ErrFeatureDisabled
	}
	if input.PointsToRedeem > 0 && !s.enableLoyalty {
		return nil, domain.ErrFeatureDisabled
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
	//    record pre-discount subtotals in order.
	batch := make([]domain.CheckoutBatchItem, 0, len(groupOrder))
	subtotals := make([]int64, 0, len(groupOrder))
	var cartSubtotal int64
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
			// MVP: coupons are platform-issued → commission is computed
			// on the pre-discount subtotal so the seller's revenue is
			// unaffected by the discount. A future seller-issued coupon
			// phase flips this to post-discount on a per-order basis.
			commissionAmount = subtotal * int64(rule.RateBps) / 10000
		}

		order := &domain.Order{
			SellerID:         sellerID,
			BuyerAuth0ID:     input.BuyerAuth0ID,
			Status:           domain.StatusPending,
			SubtotalAmount:   subtotal,
			ShippingFee:      shippingFeePerOrder,
			CommissionAmount: commissionAmount,
			Currency:         currency,
			ShippingAddress:  input.ShippingAddress,
			// TotalAmount is stamped after discount distribution below.
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
		subtotals = append(subtotals, subtotal)
		cartSubtotal += subtotal
	}

	// 3b. Reserve the coupon + points with the external services. The
	// cleanup closure releases whichever reservation succeeded if a
	// later step fails; a success flag at the end of CreateCheckout
	// disables it so only the real failure paths release.
	var (
		couponReservation *port.CouponReservation
		pointReservation  *port.PointReservation
		checkoutSucceeded bool
	)
	defer func() {
		if checkoutSucceeded {
			return
		}
		if couponReservation != nil && s.couponReserver != nil {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.couponReserver.Release(releaseCtx, couponReservation.ReservationID, "checkout_failed"); err != nil {
				slog.Warn("failed to release coupon reservation after checkout failure", "reservation_id", couponReservation.ReservationID, "error", err)
			}
		}
		if pointReservation != nil && s.pointReserver != nil {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.pointReserver.Release(releaseCtx, pointReservation.ReservationID, "checkout_failed"); err != nil {
				slog.Warn("failed to release point reservation after checkout failure", "reservation_id", pointReservation.ReservationID, "error", err)
			}
		}
	}()

	if input.CouponCode != "" {
		sellerSubs := make([]port.SellerSubtotal, len(batch))
		for i, item := range batch {
			sellerSubs[i] = port.SellerSubtotal{SellerID: item.Order.SellerID, Subtotal: subtotals[i]}
		}
		r, err := s.couponReserver.Reserve(ctx, input.CouponCode, input.BuyerAuth0ID, sellerSubs, currency)
		if err != nil {
			return nil, err // domain sentinel or infra error; handler translates
		}
		couponReservation = r
	}
	if input.PointsToRedeem > 0 {
		// Clamp requested points to what the cart can actually absorb.
		// Without this, DistributeDiscount silently caps each
		// per-seller share at its subtotal but the full reservation
		// still gets burned on Commit — the buyer would lose more
		// points than the discount they actually received.
		var couponDiscountSoFar int64
		if couponReservation != nil {
			couponDiscountSoFar = couponReservation.DiscountAmount
		}
		maxApplicable := cartSubtotal - couponDiscountSoFar
		if maxApplicable < 0 {
			maxApplicable = 0
		}
		effective := input.PointsToRedeem
		if effective > maxApplicable {
			effective = maxApplicable
		}
		if effective > 0 {
			r, err := s.pointReserver.Reserve(ctx, input.BuyerAuth0ID, effective)
			if err != nil {
				return nil, err
			}
			pointReservation = r
		}
	}

	// 3c. Distribute the discounts across seller orders proportionally
	// to subtotal. Rounding remainders go to the last non-zero bucket
	// so sum(shares) == discount exactly.
	var couponDiscount, pointDiscount int64
	if couponReservation != nil {
		couponDiscount = couponReservation.DiscountAmount
	}
	if pointReservation != nil {
		pointDiscount = pointReservation.Amount
	}
	couponShares := domain.DistributeDiscount(couponDiscount, subtotals)
	pointShares := domain.DistributeDiscount(pointDiscount, subtotals)

	// 3d. Apply shares to each order + compute per-order total. Stamp
	// the reservation IDs on the first (anchor) order only, which
	// drives the single Commit/Release round-trip on the webhook path.
	var cartTotal int64
	for i := range batch {
		batch[i].Order.CouponDiscountAmount = couponShares[i]
		batch[i].Order.PointDiscountAmount = pointShares[i]
		orderTotal := batch[i].Order.SubtotalAmount + batch[i].Order.ShippingFee - couponShares[i] - pointShares[i]
		if orderTotal < 0 {
			orderTotal = 0
		}
		batch[i].Order.TotalAmount = orderTotal
		cartTotal += orderTotal
	}
	// Stamp coupon identifiers on EVERY order that actually received
	// a discount share. Originally only the anchor order (batch[0])
	// carried these so the webhook Commit could use a single round-
	// trip, but cancellation runs per-order: if a non-anchor order is
	// cancelled, its OrderCancelled event would omit coupon_id and
	// the coupon subscriber would skip the refund entirely. Making
	// every discounted order carry the IDs lets the subscriber
	// compute per-order partial refunds against the single shared
	// redemption row.
	//
	// Commit still uses only the anchor's reservation_id — coupon-svc
	// writes one redemption per cart, keyed on (coupon_id, anchor
	// order_id). Per-order refunds find that row via
	// coupon_reservation_id on the event payload.
	if couponReservation != nil {
		cid := couponReservation.CouponID
		rid := couponReservation.ReservationID
		for i := range batch {
			if couponShares[i] > 0 {
				batch[i].Order.CouponID = &cid
				batch[i].Order.CouponReservationID = &rid
			}
		}
	}
	// Point redemption stays anchor-only: loyalty refund uses the
	// per-order point_discount_amount from the event payload, and
	// the refund ledger row is keyed on (order_cancelled, order_id,
	// refund) — no lookup by reservation_id is needed.
	if pointReservation != nil && len(batch) > 0 {
		rid := pointReservation.ReservationID
		batch[0].Order.PointReservationID = &rid
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

	// 7. Publish order.created for each order in the checkout. The
	// event now carries the per-order discount split + cart-wide
	// subtotal_before_discounts so downstream analytics (and the
	// loyalty earn-on-paid subscriber) can compute against the
	// pre-discount base.
	for i := range batch {
		o := batch[i].Order
		pubsub.PublishProtoEvent(ctx, s.publisher, domain.EventTypeOrderCreated, "order-events", &orderv1.OrderCreated{
			OrderId:                 o.ID.String(),
			SellerId:                o.SellerID.String(),
			BuyerAuth0Id:            o.BuyerAuth0ID,
			TotalAmount:             o.TotalAmount,
			Currency:                o.Currency,
			StripePaymentIntentId:   piID,
			CouponDiscountAmount:    o.CouponDiscountAmount,
			PointDiscountAmount:     o.PointDiscountAmount,
			SubtotalBeforeDiscounts: o.SubtotalAmount,
		})
	}

	slog.Info("checkout created",
		"buyer_auth0_id", input.BuyerAuth0ID,
		"order_count", len(batch),
		"total", cartTotal,
		"payment_intent", piID,
		"coupon_discount", couponDiscount,
		"point_discount", pointDiscount,
	)

	// 8. Shape the return value.
	result := &domain.CheckoutResult{
		Orders:                  make([]domain.OrderWithLines, 0, len(batch)),
		StripeClientSecret:      clientSecret,
		StripePaymentIntentID:   piID,
		TotalAmount:             cartTotal,
		Currency:                currency,
		SubtotalBeforeDiscounts: cartSubtotal,
		CouponDiscountAmount:    couponDiscount,
		PointDiscountAmount:     pointDiscount,
		AppliedCouponCode:       input.CouponCode,
	}
	for i := range batch {
		result.Orders = append(result.Orders, domain.OrderWithLines{
			Order: *batch[i].Order,
			Lines: batch[i].Lines,
		})
	}
	// Disable the failure-path release: the orders are durable and
	// the reservations will resolve to committed/released via the
	// webhook handler.
	checkoutSucceeded = true
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
		// Cancelled orders short-circuit entirely: refund path already
		// reversed any Stripe transfer. For any other non-pending
		// status (paid, processing, …) we keep going so a retried
		// webhook can redo missed Commit / earn work — those downstream
		// calls are idempotent via DB UNIQUE keys, and the payout
		// guard below prevents a double Stripe Transfer.
		if order.Status == domain.StatusCancelled {
			slog.Info("skipping payment success for cancelled order",
				"order_id", order.ID,
				"payment_intent", stripePaymentIntentID,
			)
			continue
		}
		if err := s.orderRepo.SetPaid(ctx, order.ID, now, stripePaymentIntentID); err != nil {
			if errors.Is(err, domain.ErrOrderNotPending) {
				// Already transitioned past pending — safe to fall
				// through to the retry-idempotent Commit path.
				slog.Info("order already paid, re-running downstream commits if needed",
					"order_id", order.ID,
					"order_status", order.Status,
					"payment_intent", stripePaymentIntentID,
				)
			} else {
				return apperrors.Internal("failed to update order to paid", err)
			}
		}

		// The webhook handler is split into three sections so a
		// retried Stripe delivery can complete whatever portion
		// failed on the first pass:
		//
		//   (A) One-shot Stripe transfer — gated by payout.Status =
		//       pending. A retry with a completed payout short-
		//       circuits this block entirely; it never creates a
		//       second Stripe Transfer. failed / reversed / other
		//       terminal payout states abort the whole iteration
		//       because the order is no longer in a "payout still
		//       to settle" shape — proceeding would either mis-
		//       attribute discounts to a failed order or re-apply
		//       them to a reversed one.
		//
		//   (B) Idempotent downstream commits — coupon redemption
		//       commit and loyalty redeem + earn. Every step is
		//       safe to retry: the downstream services dedupe by
		//       DB UNIQUE constraints. Failures here return an
		//       error so Stripe replays the webhook.
		//
		//   (C) One-time event publish — gated by
		//       orders.paid_event_published_at, NOT by payout status.
		//       The flag makes the publish both "at most once"
		//       (avoid duplicate notifications, etc.) and "at
		//       least once" (guarantees shipping / notification /
		//       loyalty-earn fan-in fire even when a prior attempt
		//       got past the transfer but died before publish).

		payout, err := s.payoutRepo.GetByOrderID(ctx, order.ID)
		if err != nil {
			return apperrors.Internal("failed to get payout for order", err)
		}
		if payout == nil {
			slog.Warn("no payout record found for order, skipping transfer",
				"order_id", order.ID, "payment_intent", stripePaymentIntentID)
			continue
		}

		// Payout-state allowlist. Only `pending` (first delivery)
		// and `completed` (retry after a successful transfer) are
		// compatible with the happy-path handler. `failed` and
		// `reversed` require manual recovery and must not trigger
		// coupon / loyalty commit work.
		switch payout.Status {
		case domain.PayoutStatusPending, domain.PayoutStatusCompleted:
			// fall through to the handler body
		default:
			slog.Warn("payout in terminal non-success state; skipping handler body",
				"order_id", order.ID,
				"payout_id", payout.ID,
				"payout_status", payout.Status,
				"payment_intent", stripePaymentIntentID,
			)
			continue
		}

		// --- (A) One-shot transfer block ---
		var (
			transferID      string
			transferPending = payout.Status == domain.PayoutStatusPending
		)
		if transferPending {
			sellerStripeAccountID := getSellerStripeAccountID(order.SellerID)
			var transferErr error
			transferID, transferErr = s.stripe.CreateTransfer(
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
				// Payout has failed but the buyer's money is still
				// on the platform; don't proceed to Commit the
				// reservation because the payout state is in a
				// manual-recovery situation that ops will unstick.
				continue
			}
			if err := s.payoutRepo.UpdateStatus(ctx, payout.ID, domain.PayoutStatusCompleted, &transferID); err != nil {
				return apperrors.Internal("failed to mark payout completed", err)
			}
			slog.Info("order marked paid and transfer created",
				"order_id", order.ID,
				"payment_intent", stripePaymentIntentID,
				"transfer_id", transferID,
			)
		} else {
			// Retry / replay: payout already completed on a prior
			// delivery. Skip the Stripe Transfer (avoid duplicate
			// money movement) and fall through to (B) and (C) so
			// previously-failed commits or publishes can recover.
			slog.Info("payout already completed; retrying downstream commits / publish",
				"order_id", order.ID,
				"payout_id", payout.ID,
				"payment_intent", stripePaymentIntentID,
			)
			if payout.StripeTransferID != nil {
				transferID = *payout.StripeTransferID
			}
		}

		// --- (B) Idempotent downstream commits ---
		// (B.1) Coupon redemption commit. Dedupes by
		// (coupon_id, order_id) UNIQUE.
		if order.CouponReservationID != nil && s.couponReserver != nil {
			if _, err := s.couponReserver.Commit(ctx, *order.CouponReservationID, order.ID, stripePaymentIntentID); err != nil {
				slog.Error("coupon commit failed; returning error so Stripe retries the webhook",
					"error", err, "order_id", order.ID, "reservation_id", *order.CouponReservationID)
				return apperrors.Internal("failed to commit coupon reservation", err)
			}
		}
		// (B.2) Loyalty commit (redeem + earn). Dedupes by
		// (reservation_commit, reservation_id, redeem) and
		// (order_paid, order_id, earn) UNIQUE.
		if s.pointReserver != nil && s.enableLoyalty {
			subtotalForEarn := order.SubtotalAmount
			earnResult, err := s.pointReserver.Commit(ctx, order.PointReservationID, order.BuyerAuth0ID, order.ID.String(), stripePaymentIntentID, subtotalForEarn, order.Currency)
			if err != nil {
				slog.Error("loyalty commit failed; returning error so Stripe retries the webhook",
					"error", err, "order_id", order.ID)
				return apperrors.Internal("failed to commit loyalty reservation / earn", err)
			}
			if earnResult != nil && earnResult.Earned > 0 {
				// Best-effort mirror of the earned total onto the
				// order row for display. The loyalty ledger is
				// authoritative; a slip here is non-critical.
				if err := s.orderRepo.SetPointsEarned(ctx, order.ID, earnResult.Earned); err != nil {
					slog.Warn("failed to persist points_earned on order", "error", err, "order_id", order.ID, "earned", earnResult.Earned)
				}
				order.PointsEarned = earnResult.Earned
			}
		}

		// --- (C) Claim → Publish → Mark ---
		// Closes two separate reliability windows:
		//
		//   C.1 Claim: ClaimPaidEventPublish atomically serializes
		//     concurrent handler runs with a TTL-bounded row lock.
		//     Only the winner proceeds to C.2. Losers skip — either
		//     the current claim-holder is publishing (and will mark
		//     on success), or the row is already fully published.
		//     The TTL self-heals a crashed publisher: once
		//     publishClaimTTL elapses another attempt may re-claim.
		//
		//   C.2 Publish: PublishProtoEventWithErr returns the broker
		//     error instead of swallowing it. If either event fails
		//     we return an error so Stripe redelivers the webhook —
		//     the next attempt re-claims (after TTL) and re-publishes.
		//     Subscribers are idempotent by order_id so a duplicate
		//     caused by a racing stale-claim recovery is tolerable;
		//     a DROPPED event is not.
		//
		//   C.3 Mark: MarkPaidEventPublished stamps the final "done"
		//     flag, which makes C.1 fail on all future attempts and
		//     short-circuits the whole block.
		if order.PaidEventPublishedAt != nil {
			continue
		}
		claimed, err := s.orderRepo.ClaimPaidEventPublish(ctx, order.ID, publishClaimTTL)
		if err != nil {
			return apperrors.Internal("claim paid event publish slot", err)
		}
		if !claimed {
			slog.Info("paid-event publish claim held by another handler; skipping",
				"order_id", order.ID, "payment_intent", stripePaymentIntentID)
			continue
		}

		lineItems, linesErr := s.loadPaidLineItems(ctx, order.ID)
		if linesErr != nil {
			slog.Warn("failed to load line items for order.paid event; publishing without them",
				"order_id", order.ID, "error", linesErr)
		}
		if err := pubsub.PublishProtoEventWithErr(ctx, s.publisher, domain.EventTypeOrderPaid, "order-events", &orderv1.OrderPaid{
			OrderId:                 order.ID.String(),
			SellerId:                order.SellerID.String(),
			BuyerAuth0Id:            order.BuyerAuth0ID,
			TotalAmount:             order.TotalAmount,
			StripePaymentIntentId:   stripePaymentIntentID,
			ShippingAddressJson:     string(order.ShippingAddress),
			LineItems:               lineItems,
			CouponDiscountAmount:    order.CouponDiscountAmount,
			PointDiscountAmount:     order.PointDiscountAmount,
			SubtotalBeforeDiscounts: order.SubtotalAmount,
			PointsEarned:            order.PointsEarned,
		}); err != nil {
			slog.Error("failed to publish order.paid; returning error so Stripe retries",
				"error", err, "order_id", order.ID)
			return apperrors.Internal("publish order.paid", err)
		}
		if err := pubsub.PublishProtoEventWithErr(ctx, s.publisher, domain.EventTypePayoutCompleted, "payout-events", &orderv1.PayoutCompleted{
			PayoutId:         payout.ID.String(),
			OrderId:          order.ID.String(),
			SellerId:         order.SellerID.String(),
			Amount:           payout.Amount,
			Currency:         payout.Currency,
			StripeTransferId: transferID,
		}); err != nil {
			slog.Error("failed to publish payout.completed; returning error so Stripe retries",
				"error", err, "order_id", order.ID)
			return apperrors.Internal("publish payout.completed", err)
		}
		// Both publishes succeeded → stamp the done flag. A failure
		// here means the next Stripe retry will republish (idempotent
		// at subscriber-level), which is still safer than dropping.
		if _, err := s.orderRepo.MarkPaidEventPublished(ctx, order.ID); err != nil {
			slog.Error("failed to mark paid_event_published_at; next retry may republish",
				"error", err, "order_id", order.ID)
		}
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
