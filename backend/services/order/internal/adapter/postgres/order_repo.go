package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// ErrOrderNotPending is re-exported from domain for callers that may import
// this package directly. New code should prefer domain.ErrOrderNotPending.
//
// Deprecated: use domain.ErrOrderNotPending.
var ErrOrderNotPending = domain.ErrOrderNotPending

// OrderRepository handles persistence of orders and order lines.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository creates a new OrderRepository.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Create inserts a new order and its lines within a single transaction.
// The caller is responsible for resolving order.SellerName (via the auth
// service's /internal/sellers/batch-get) and each line.ProductID (via
// catalog's BatchGetSKUs RPC) BEFORE invoking this method. Phase 1.2
// removed the cross-schema lookups that used to live here so order_svc
// no longer depends on auth_svc / catalog_svc schemas at runtime.
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order, lines []domain.OrderLine) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		return insertOrderTx(ctx, tx, order, lines)
	})
}

// insertOrderTx inserts a single order plus its lines using an existing
// transaction. Callers are responsible for the surrounding Tx and for
// populating order.SellerName and line.ProductID via lookupSellerNames /
// lookupSKUProductIDs before calling this. Shared by Create (single-seller,
// deprecated) and CreateCheckoutBatch (multi-seller).
func insertOrderTx(ctx context.Context, tx pgx.Tx, order *domain.Order, lines []domain.OrderLine) error {
	order.ID = uuid.New()

	err := tx.QueryRow(ctx,
		`INSERT INTO order_svc.orders
		 (id, seller_id, seller_name, buyer_auth0_id, status, subtotal_amount, shipping_fee, commission_amount, total_amount, currency, shipping_address, stripe_payment_intent_id,
		  coupon_discount_amount, point_discount_amount, coupon_id, coupon_reservation_id, point_reservation_id, points_earned)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		 RETURNING created_at, updated_at`,
		order.ID, order.SellerID, order.SellerName, order.BuyerAuth0ID, order.Status,
		order.SubtotalAmount, order.ShippingFee, order.CommissionAmount, order.TotalAmount, order.Currency,
		order.ShippingAddress, order.StripePaymentIntentID,
		order.CouponDiscountAmount, order.PointDiscountAmount, order.CouponID, order.CouponReservationID, order.PointReservationID, order.PointsEarned,
	).Scan(&order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	for i := range lines {
		lines[i].ID = uuid.New()
		lines[i].OrderID = order.ID

		err := tx.QueryRow(ctx,
			`INSERT INTO order_svc.order_lines
			 (id, order_id, sku_id, product_id, product_name, sku_code, quantity, unit_price, line_total)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING created_at`,
			lines[i].ID, lines[i].OrderID, lines[i].SKUID, lines[i].ProductID,
			lines[i].ProductName, lines[i].SKUCode, lines[i].Quantity,
			lines[i].UnitPrice, lines[i].LineTotal,
		).Scan(&lines[i].CreatedAt)
		if err != nil {
			return fmt.Errorf("insert order line: %w", err)
		}
	}

	return nil
}

// CheckoutBatchItem is re-exported from domain for backwards compatibility.
// New code should prefer domain.CheckoutBatchItem.
//
// Deprecated: use domain.CheckoutBatchItem.
type CheckoutBatchItem = domain.CheckoutBatchItem

// CreateCheckoutBatch inserts a list of orders (one per seller) together
// with a matching pending payout for each, inside a single transaction.
// All IDs are assigned by this method. If any insert fails, the entire
// batch rolls back — so a multi-seller checkout either creates every order
// cleanly or leaves the database untouched.
//
// Callers must resolve order.SellerName + line.ProductID for every item
// before calling (Phase 1.2 moved those lookups out of the repo).
func (r *OrderRepository) CreateCheckoutBatch(ctx context.Context, items []CheckoutBatchItem) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		for _, item := range items {
			if err := insertOrderTx(ctx, tx, item.Order, item.Lines); err != nil {
				return err
			}

			// Insert a matching pending payout. stripe_transfer_id stays NULL
			// until the webhook creates a Stripe Transfer on payment success.
			item.Payout.ID = uuid.New()
			item.Payout.OrderID = item.Order.ID
			item.Payout.Status = domain.PayoutStatusPending

			err := tx.QueryRow(ctx,
				`INSERT INTO order_svc.payouts
				 (id, seller_id, order_id, amount, currency, stripe_transfer_id, status)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)
				 RETURNING created_at`,
				item.Payout.ID, item.Payout.SellerID, item.Payout.OrderID,
				item.Payout.Amount, item.Payout.Currency, item.Payout.StripeTransferID, item.Payout.Status,
			).Scan(&item.Payout.CreatedAt)
			if err != nil {
				return fmt.Errorf("insert payout: %w", err)
			}
		}
		return nil
	})
}

// GetByID retrieves an order with its lines.
func (r *OrderRepository) GetByID(ctx context.Context, orderID uuid.UUID) (*domain.OrderWithLines, error) {
	var result domain.OrderWithLines
	var found bool

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, seller_id, seller_name, buyer_auth0_id, status,
			        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
			        shipping_address, stripe_payment_intent_id, paid_at, cancelled_at, cancellation_reason,
			        created_at, updated_at,
			        coupon_discount_amount, point_discount_amount, coupon_id,
			        coupon_reservation_id, point_reservation_id, points_earned,
			        paid_event_published_at
			 FROM order_svc.orders WHERE id = $1`,
			orderID,
		).Scan(
			&result.ID, &result.SellerID, &result.SellerName, &result.BuyerAuth0ID, &result.Status,
			&result.SubtotalAmount, &result.ShippingFee, &result.CommissionAmount, &result.TotalAmount, &result.Currency,
			&result.ShippingAddress, &result.StripePaymentIntentID, &result.PaidAt, &result.CancelledAt, &result.CancellationReason,
			&result.CreatedAt, &result.UpdatedAt,
			&result.CouponDiscountAmount, &result.PointDiscountAmount, &result.CouponID,
			&result.CouponReservationID, &result.PointReservationID, &result.PointsEarned,
			&result.PaidEventPublishedAt,
		)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true

		rows, err := tx.Query(ctx,
			`SELECT id, order_id, sku_id, product_id, product_name, sku_code, quantity, unit_price, line_total, created_at
			 FROM order_svc.order_lines WHERE order_id = $1
			 ORDER BY created_at ASC`,
			orderID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var l domain.OrderLine
			if err := rows.Scan(
				&l.ID, &l.OrderID, &l.SKUID, &l.ProductID, &l.ProductName,
				&l.SKUCode, &l.Quantity, &l.UnitPrice, &l.LineTotal, &l.CreatedAt,
			); err != nil {
				return err
			}
			result.Lines = append(result.Lines, l)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &result, nil
}

// PurchaseSKURecord is re-exported from domain for backwards compatibility.
// New code should prefer domain.PurchaseSKURecord.
//
// Deprecated: use domain.PurchaseSKURecord.
type PurchaseSKURecord = domain.PurchaseSKURecord

// HasPurchasedSKU reports whether the given buyer has a paid-or-later order
// with the given seller containing the given SKU. Returns the earliest such
// order's id plus the product/sku name snapshot from the order line. Returns
// (nil, nil) when no matching purchase exists. Statuses considered "purchased"
// are paid, processing, shipped, delivered, and completed.
func (r *OrderRepository) HasPurchasedSKU(ctx context.Context, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*PurchaseSKURecord, error) {
	var rec PurchaseSKURecord
	var found bool

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT o.id, ol.product_name, ol.sku_code
			   FROM order_svc.orders o
			   JOIN order_svc.order_lines ol
			        ON ol.order_id = o.id
			  WHERE o.buyer_auth0_id = $1
			    AND o.seller_id      = $2
			    AND ol.sku_id        = $3
			    AND o.status IN ('paid','processing','shipped','delivered','completed')
			  ORDER BY o.created_at ASC
			  LIMIT 1`,
			buyerAuth0ID, sellerID, skuID,
		).Scan(&rec.OrderID, &rec.ProductName, &rec.SKUCode)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("check purchased sku: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &rec, nil
}

// ListByBuyer returns paginated orders for a specific buyer.
func (r *OrderRepository) ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error) {
	var orders []domain.Order
	var total int

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM order_svc.orders WHERE buyer_auth0_id = $1`,
			buyerAuth0ID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count buyer orders: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT id, seller_id, seller_name, buyer_auth0_id, status,
			        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
			        shipping_address, stripe_payment_intent_id, paid_at, cancelled_at, cancellation_reason,
			        created_at, updated_at,
			        coupon_discount_amount, point_discount_amount, coupon_id,
			        coupon_reservation_id, point_reservation_id, points_earned,
			        paid_event_published_at
			 FROM order_svc.orders WHERE buyer_auth0_id = $1
			 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			buyerAuth0ID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list buyer orders: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var o domain.Order
			if err := rows.Scan(
				&o.ID, &o.SellerID, &o.SellerName, &o.BuyerAuth0ID, &o.Status,
				&o.SubtotalAmount, &o.ShippingFee, &o.CommissionAmount, &o.TotalAmount, &o.Currency,
				&o.ShippingAddress, &o.StripePaymentIntentID, &o.PaidAt, &o.CancelledAt, &o.CancellationReason,
				&o.CreatedAt, &o.UpdatedAt,
				&o.CouponDiscountAmount, &o.PointDiscountAmount, &o.CouponID,
				&o.CouponReservationID, &o.PointReservationID, &o.PointsEarned,
				&o.PaidEventPublishedAt,
			); err != nil {
				return fmt.Errorf("scan order: %w", err)
			}
			orders = append(orders, o)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListBySeller returns paginated orders for a specific seller, optionally filtered by status.
func (r *OrderRepository) ListBySeller(ctx context.Context, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error) {
	var orders []domain.Order
	var total int

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		conditions := "seller_id = $1"
		args := []any{sellerID}
		argIdx := 2

		if status != "" {
			conditions += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, status)
			argIdx++
		}

		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM order_svc.orders WHERE %s", conditions)
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return fmt.Errorf("count seller orders: %w", err)
		}

		query := fmt.Sprintf(
			`SELECT id, seller_id, seller_name, buyer_auth0_id, status,
			        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
			        shipping_address, stripe_payment_intent_id, paid_at, cancelled_at, cancellation_reason,
			        created_at, updated_at,
			        coupon_discount_amount, point_discount_amount, coupon_id,
			        coupon_reservation_id, point_reservation_id, points_earned,
			        paid_event_published_at
			 FROM order_svc.orders WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
			conditions, argIdx, argIdx+1,
		)
		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("list seller orders: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var o domain.Order
			if err := rows.Scan(
				&o.ID, &o.SellerID, &o.SellerName, &o.BuyerAuth0ID, &o.Status,
				&o.SubtotalAmount, &o.ShippingFee, &o.CommissionAmount, &o.TotalAmount, &o.Currency,
				&o.ShippingAddress, &o.StripePaymentIntentID, &o.PaidAt, &o.CancelledAt, &o.CancellationReason,
				&o.CreatedAt, &o.UpdatedAt,
				&o.CouponDiscountAmount, &o.PointDiscountAmount, &o.CouponID,
				&o.CouponReservationID, &o.PointReservationID, &o.PointsEarned,
				&o.PaidEventPublishedAt,
			); err != nil {
				return fmt.Errorf("scan order: %w", err)
			}
			orders = append(orders, o)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// UpdateStatus changes the status of an order.
func (r *OrderRepository) UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE order_svc.orders SET status = $2, updated_at = NOW()
			 WHERE id = $1`,
			orderID, status,
		)
		if err != nil {
			return fmt.Errorf("update order status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("order not found")
		}
		return nil
	})
}

// SetPaid marks a pending order as paid with the payment timestamp and Stripe
// payment intent ID. The UPDATE is guarded on status='pending' so a redelivered
// payment_intent.succeeded webhook (either an idempotent retry on an already
// paid order, or a replay arriving after the order was cancelled) cannot flip
// the row. Returns ErrOrderNotPending in that case so HandlePaymentSuccess
// can skip transfer creation — see the P1 guard in order_service.go.
func (r *OrderRepository) SetPaid(ctx context.Context, orderID uuid.UUID, paidAt time.Time, stripePaymentIntentID string) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE order_svc.orders
			 SET status = $2, paid_at = $3, stripe_payment_intent_id = $4, updated_at = NOW()
			 WHERE id = $1 AND status = $5`,
			orderID, domain.StatusPaid, paidAt, stripePaymentIntentID, domain.StatusPending,
		)
		if err != nil {
			return fmt.Errorf("set order paid: %w", err)
		}
		if tag.RowsAffected() == 0 {
			// Distinguish "row missing" (hard error) from "row present but
			// not in pending" (idempotent no-op) so the caller can skip
			// downstream transfer creation without treating replay as a
			// hard failure.
			var sentinel int
			qerr := tx.QueryRow(ctx,
				`SELECT 1 FROM order_svc.orders WHERE id = $1`,
				orderID,
			).Scan(&sentinel)
			if errors.Is(qerr, pgx.ErrNoRows) {
				return fmt.Errorf("order not found")
			}
			if qerr != nil {
				return fmt.Errorf("check order existence after SetPaid no-op: %w", qerr)
			}
			return domain.ErrOrderNotPending
		}
		return nil
	})
}

// FindByStripePaymentIntentID finds an order by payment intent ID.
func (r *OrderRepository) FindByStripePaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.Order, error) {
	var o domain.Order
	err := r.pool.QueryRow(ctx,
		`SELECT id, seller_id, seller_name, buyer_auth0_id, status,
		        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
		        shipping_address, stripe_payment_intent_id, paid_at, cancelled_at, cancellation_reason,
		        created_at, updated_at,
		        coupon_discount_amount, point_discount_amount, coupon_id,
		        coupon_reservation_id, point_reservation_id, points_earned,
		        paid_event_published_at
		 FROM order_svc.orders WHERE stripe_payment_intent_id = $1`,
		paymentIntentID,
	).Scan(
		&o.ID, &o.SellerID, &o.SellerName, &o.BuyerAuth0ID, &o.Status,
		&o.SubtotalAmount, &o.ShippingFee, &o.CommissionAmount, &o.TotalAmount, &o.Currency,
		&o.ShippingAddress, &o.StripePaymentIntentID, &o.PaidAt, &o.CancelledAt, &o.CancellationReason,
		&o.CreatedAt, &o.UpdatedAt,
		&o.CouponDiscountAmount, &o.PointDiscountAmount, &o.CouponID,
		&o.CouponReservationID, &o.PointReservationID, &o.PointsEarned,
		&o.PaidEventPublishedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find order by payment intent: %w", err)
	}
	return &o, nil
}

// FindAllByStripePaymentIntentID returns every order sharing a payment intent
// ID. Used by the webhook handler: a single Stripe PaymentIntent maps to N
// orders (one per seller) in a multi-seller checkout.
func (r *OrderRepository) FindAllByStripePaymentIntentID(ctx context.Context, paymentIntentID string) ([]domain.Order, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, seller_id, seller_name, buyer_auth0_id, status,
		        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
		        shipping_address, stripe_payment_intent_id, paid_at, cancelled_at, cancellation_reason,
		        created_at, updated_at,
		        coupon_discount_amount, point_discount_amount, coupon_id,
		        coupon_reservation_id, point_reservation_id, points_earned,
		        paid_event_published_at
		 FROM order_svc.orders WHERE stripe_payment_intent_id = $1
		 ORDER BY created_at ASC`,
		paymentIntentID,
	)
	if err != nil {
		return nil, fmt.Errorf("find orders by payment intent: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(
			&o.ID, &o.SellerID, &o.SellerName, &o.BuyerAuth0ID, &o.Status,
			&o.SubtotalAmount, &o.ShippingFee, &o.CommissionAmount, &o.TotalAmount, &o.Currency,
			&o.ShippingAddress, &o.StripePaymentIntentID, &o.PaidAt, &o.CancelledAt, &o.CancellationReason,
			&o.CreatedAt, &o.UpdatedAt,
			&o.CouponDiscountAmount, &o.PointDiscountAmount, &o.CouponID,
			&o.CouponReservationID, &o.PointReservationID, &o.PointsEarned,
			&o.PaidEventPublishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// SetStripePaymentIntentID updates the payment intent id for an already-created
// order. Used by CreateCheckout which inserts orders first and then stamps the
// Stripe PaymentIntent id once it has been created for the whole checkout.
func (r *OrderRepository) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE order_svc.orders
			 SET stripe_payment_intent_id = $2, updated_at = NOW()
			 WHERE id = $1`,
			orderID, paymentIntentID,
		)
		if err != nil {
			return fmt.Errorf("set stripe payment intent id: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("order not found")
		}
		return nil
	})
}

// MarkPaidEventPublished stamps paid_event_published_at with the
// current timestamp if it was previously NULL. Returns (true, nil)
// when the column was just set (caller just published events) and
// (false, nil) on a replay where the column was already populated
// (caller should not re-publish). The WHERE clause enforces the
// once-only semantics at the DB level even under concurrent handler
// replays.
func (r *OrderRepository) MarkPaidEventPublished(ctx context.Context, orderID uuid.UUID) (bool, error) {
	var marked bool
	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE order_svc.orders
			 SET paid_event_published_at = NOW(), updated_at = NOW()
			 WHERE id = $1 AND paid_event_published_at IS NULL`,
			orderID,
		)
		if err != nil {
			return fmt.Errorf("mark paid event published: %w", err)
		}
		marked = tag.RowsAffected() > 0
		return nil
	})
	return marked, err
}

// ClaimPaidEventPublish acquires a TTL-bounded claim on the paid
// publish slot for this order. A single CTE atomically attempts the
// UPDATE and reports the post-update published-flag state, so the
// caller can distinguish "someone else already published" (clean
// skip) from "someone else is currently publishing" (retryable).
//
// Outcomes:
//   - acquired=true                                  — proceed to publish
//   - acquired=false, alreadyPublished=true          — skip (done)
//   - acquired=false, alreadyPublished=false         — another handler
//     owns a fresh claim. Caller must surface a retryable error so
//     Stripe redelivers after the TTL elapses; returning cleanly
//     here would strand the publish if the claim-holder later fails.
//
// Stale claims (older than ttl) are transparently re-acquired so a
// crashed mid-publish handler doesn't block the publish forever.
func (r *OrderRepository) ClaimPaidEventPublish(ctx context.Context, orderID uuid.UUID, ttl time.Duration) (acquired, alreadyPublished bool, err error) {
	err = database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		// One round-trip: the UPDATE CTE only returns a row when we
		// win the claim; the subsequent SELECT reads the final
		// published-flag state (reflects the UPDATE if it ran, or
		// the pre-existing value if it didn't). This is what makes
		// "already_published" cheap to detect without an extra query.
		return tx.QueryRow(ctx,
			`WITH upd AS (
			     UPDATE order_svc.orders
			        SET paid_event_claim_at = NOW(), updated_at = NOW()
			      WHERE id = $1
			        AND paid_event_published_at IS NULL
			        AND (paid_event_claim_at IS NULL
			             OR paid_event_claim_at < NOW() - ($2::bigint * INTERVAL '1 second'))
			  RETURNING id
			 )
			 SELECT
			     EXISTS (SELECT 1 FROM upd)                                           AS acquired,
			     COALESCE((SELECT paid_event_published_at IS NOT NULL
			                 FROM order_svc.orders WHERE id = $1), false)             AS already_published`,
			orderID, int64(ttl.Seconds()),
		).Scan(&acquired, &alreadyPublished)
	})
	return acquired, alreadyPublished, err
}

// SetPointsEarned records the loyalty points earned on order payment.
// Called from HandlePaymentSuccess after loyalty-svc confirms the
// ledger write. Safe to call multiple times — the value is the
// authoritative totals from loyalty-svc, not an accumulator.
func (r *OrderRepository) SetPointsEarned(ctx context.Context, orderID uuid.UUID, amount int64) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE order_svc.orders
			 SET points_earned = $2, updated_at = NOW()
			 WHERE id = $1`,
			orderID, amount,
		)
		if err != nil {
			return fmt.Errorf("set points earned: %w", err)
		}
		return nil
	})
}
