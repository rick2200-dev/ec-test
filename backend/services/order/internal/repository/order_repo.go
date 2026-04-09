package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// OrderRepository handles persistence of orders and order lines.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository creates a new OrderRepository.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Create inserts a new order and its lines within a single tenant-scoped transaction.
func (r *OrderRepository) Create(ctx context.Context, tenantID uuid.UUID, order *domain.Order, lines []domain.OrderLine) error {
	order.ID = uuid.New()
	order.TenantID = tenantID

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO order_svc.orders
			 (id, tenant_id, seller_id, buyer_auth0_id, status, subtotal_amount, shipping_fee, commission_amount, total_amount, currency, shipping_address, stripe_payment_intent_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			 RETURNING created_at, updated_at`,
			order.ID, order.TenantID, order.SellerID, order.BuyerAuth0ID, order.Status,
			order.SubtotalAmount, order.ShippingFee, order.CommissionAmount, order.TotalAmount, order.Currency,
			order.ShippingAddress, order.StripePaymentIntentID,
		).Scan(&order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return fmt.Errorf("insert order: %w", err)
		}

		for i := range lines {
			lines[i].ID = uuid.New()
			lines[i].TenantID = tenantID
			lines[i].OrderID = order.ID

			err := tx.QueryRow(ctx,
				`INSERT INTO order_svc.order_lines
				 (id, tenant_id, order_id, sku_id, product_name, sku_code, quantity, unit_price, line_total)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				 RETURNING created_at`,
				lines[i].ID, lines[i].TenantID, lines[i].OrderID, lines[i].SKUID,
				lines[i].ProductName, lines[i].SKUCode, lines[i].Quantity,
				lines[i].UnitPrice, lines[i].LineTotal,
			).Scan(&lines[i].CreatedAt)
			if err != nil {
				return fmt.Errorf("insert order line: %w", err)
			}
		}

		return nil
	})
}

// GetByID retrieves an order with its lines.
func (r *OrderRepository) GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error) {
	var result domain.OrderWithLines
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, buyer_auth0_id, status,
			        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
			        shipping_address, stripe_payment_intent_id, paid_at, created_at, updated_at
			 FROM order_svc.orders WHERE id = $1 AND tenant_id = $2`,
			orderID, tenantID,
		).Scan(
			&result.ID, &result.TenantID, &result.SellerID, &result.BuyerAuth0ID, &result.Status,
			&result.SubtotalAmount, &result.ShippingFee, &result.CommissionAmount, &result.TotalAmount, &result.Currency,
			&result.ShippingAddress, &result.StripePaymentIntentID, &result.PaidAt, &result.CreatedAt, &result.UpdatedAt,
		)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true

		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, order_id, sku_id, product_name, sku_code, quantity, unit_price, line_total, created_at
			 FROM order_svc.order_lines WHERE order_id = $1 AND tenant_id = $2
			 ORDER BY created_at ASC`,
			orderID, tenantID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var l domain.OrderLine
			if err := rows.Scan(
				&l.ID, &l.TenantID, &l.OrderID, &l.SKUID, &l.ProductName,
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

// ListByBuyer returns paginated orders for a specific buyer.
func (r *OrderRepository) ListByBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]domain.Order, int, error) {
	var orders []domain.Order
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM order_svc.orders WHERE tenant_id = $1 AND buyer_auth0_id = $2`,
			tenantID, buyerAuth0ID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count buyer orders: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, seller_id, buyer_auth0_id, status,
			        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
			        shipping_address, stripe_payment_intent_id, paid_at, created_at, updated_at
			 FROM order_svc.orders WHERE tenant_id = $1 AND buyer_auth0_id = $2
			 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
			tenantID, buyerAuth0ID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list buyer orders: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var o domain.Order
			if err := rows.Scan(
				&o.ID, &o.TenantID, &o.SellerID, &o.BuyerAuth0ID, &o.Status,
				&o.SubtotalAmount, &o.ShippingFee, &o.CommissionAmount, &o.TotalAmount, &o.Currency,
				&o.ShippingAddress, &o.StripePaymentIntentID, &o.PaidAt, &o.CreatedAt, &o.UpdatedAt,
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
func (r *OrderRepository) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, status string, limit, offset int) ([]domain.Order, int, error) {
	var orders []domain.Order
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		conditions := "tenant_id = $1 AND seller_id = $2"
		args := []any{tenantID, sellerID}
		argIdx := 3

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
			`SELECT id, tenant_id, seller_id, buyer_auth0_id, status,
			        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
			        shipping_address, stripe_payment_intent_id, paid_at, created_at, updated_at
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
				&o.ID, &o.TenantID, &o.SellerID, &o.BuyerAuth0ID, &o.Status,
				&o.SubtotalAmount, &o.ShippingFee, &o.CommissionAmount, &o.TotalAmount, &o.Currency,
				&o.ShippingAddress, &o.StripePaymentIntentID, &o.PaidAt, &o.CreatedAt, &o.UpdatedAt,
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
func (r *OrderRepository) UpdateStatus(ctx context.Context, tenantID, orderID uuid.UUID, status string) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE order_svc.orders SET status = $3, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			orderID, tenantID, status,
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

// SetPaid marks an order as paid with the payment timestamp and Stripe payment intent ID.
func (r *OrderRepository) SetPaid(ctx context.Context, tenantID, orderID uuid.UUID, paidAt time.Time, stripePaymentIntentID string) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE order_svc.orders
			 SET status = $3, paid_at = $4, stripe_payment_intent_id = $5, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			orderID, tenantID, domain.StatusPaid, paidAt, stripePaymentIntentID,
		)
		if err != nil {
			return fmt.Errorf("set order paid: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("order not found")
		}
		return nil
	})
}

// GetByStripePaymentIntentID finds an order by its Stripe payment intent ID.
func (r *OrderRepository) GetByStripePaymentIntentID(ctx context.Context, tenantID uuid.UUID, paymentIntentID string) (*domain.Order, error) {
	var o domain.Order
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, buyer_auth0_id, status,
			        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
			        shipping_address, stripe_payment_intent_id, paid_at, created_at, updated_at
			 FROM order_svc.orders WHERE stripe_payment_intent_id = $1 AND tenant_id = $2`,
			paymentIntentID, tenantID,
		).Scan(
			&o.ID, &o.TenantID, &o.SellerID, &o.BuyerAuth0ID, &o.Status,
			&o.SubtotalAmount, &o.ShippingFee, &o.CommissionAmount, &o.TotalAmount, &o.Currency,
			&o.ShippingAddress, &o.StripePaymentIntentID, &o.PaidAt, &o.CreatedAt, &o.UpdatedAt,
		)
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
		return nil, fmt.Errorf("get order by payment intent: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &o, nil
}

// FindByStripePaymentIntentID finds an order across all tenants by payment intent ID.
// This is used by webhook handlers where tenant context may not be available.
func (r *OrderRepository) FindByStripePaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.Order, error) {
	var o domain.Order
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, seller_id, buyer_auth0_id, status,
		        subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
		        shipping_address, stripe_payment_intent_id, paid_at, created_at, updated_at
		 FROM order_svc.orders WHERE stripe_payment_intent_id = $1`,
		paymentIntentID,
	).Scan(
		&o.ID, &o.TenantID, &o.SellerID, &o.BuyerAuth0ID, &o.Status,
		&o.SubtotalAmount, &o.ShippingFee, &o.CommissionAmount, &o.TotalAmount, &o.Currency,
		&o.ShippingAddress, &o.StripePaymentIntentID, &o.PaidAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find order by payment intent: %w", err)
	}
	return &o, nil
}
