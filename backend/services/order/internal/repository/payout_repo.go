package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// PayoutRepository handles persistence of payouts.
type PayoutRepository struct {
	pool *pgxpool.Pool
}

// NewPayoutRepository creates a new PayoutRepository.
func NewPayoutRepository(pool *pgxpool.Pool) *PayoutRepository {
	return &PayoutRepository{pool: pool}
}

// Create inserts a new payout record.
func (r *PayoutRepository) Create(ctx context.Context, tenantID uuid.UUID, payout *domain.Payout) error {
	payout.ID = uuid.New()
	payout.TenantID = tenantID

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO order_svc.payouts
			 (id, tenant_id, seller_id, order_id, amount, currency, stripe_transfer_id, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING created_at`,
			payout.ID, payout.TenantID, payout.SellerID, payout.OrderID,
			payout.Amount, payout.Currency, payout.StripeTransferID, payout.Status,
		).Scan(&payout.CreatedAt)
	})
}

// GetByOrderID retrieves a payout by its associated order ID.
func (r *PayoutRepository) GetByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Payout, error) {
	var p domain.Payout
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, order_id, amount, currency, stripe_transfer_id, status, created_at, completed_at
			 FROM order_svc.payouts WHERE order_id = $1 AND tenant_id = $2`,
			orderID, tenantID,
		).Scan(
			&p.ID, &p.TenantID, &p.SellerID, &p.OrderID,
			&p.Amount, &p.Currency, &p.StripeTransferID, &p.Status, &p.CreatedAt, &p.CompletedAt,
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
		return nil, fmt.Errorf("get payout by order id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &p, nil
}

// ListBySeller returns paginated payouts for a specific seller.
func (r *PayoutRepository) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Payout, int, error) {
	var payouts []domain.Payout
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM order_svc.payouts WHERE tenant_id = $1 AND seller_id = $2`,
			tenantID, sellerID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count payouts: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, seller_id, order_id, amount, currency, stripe_transfer_id, status, created_at, completed_at
			 FROM order_svc.payouts WHERE tenant_id = $1 AND seller_id = $2
			 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
			tenantID, sellerID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list payouts: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var p domain.Payout
			if err := rows.Scan(
				&p.ID, &p.TenantID, &p.SellerID, &p.OrderID,
				&p.Amount, &p.Currency, &p.StripeTransferID, &p.Status, &p.CreatedAt, &p.CompletedAt,
			); err != nil {
				return fmt.Errorf("scan payout: %w", err)
			}
			payouts = append(payouts, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return payouts, total, nil
}

// UpdateStatus updates the status (and optionally the stripe transfer ID) of a payout.
func (r *PayoutRepository) UpdateStatus(ctx context.Context, tenantID, payoutID uuid.UUID, status string, stripeTransferID *string) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE order_svc.payouts
			 SET status = $3, stripe_transfer_id = COALESCE($4, stripe_transfer_id),
			     completed_at = CASE WHEN $3 = 'completed' THEN NOW() ELSE completed_at END
			 WHERE id = $1 AND tenant_id = $2`,
			payoutID, tenantID, status, stripeTransferID,
		)
		if err != nil {
			return fmt.Errorf("update payout status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("payout not found")
		}
		return nil
	})
}
