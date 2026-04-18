package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

type RedemptionRepository struct {
	pool *pgxpool.Pool
}

func NewRedemptionRepository(pool *pgxpool.Pool) *RedemptionRepository {
	return &RedemptionRepository{pool: pool}
}

func (r *RedemptionRepository) Insert(ctx context.Context, red *domain.CouponRedemption) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO coupon_svc.coupon_redemptions
			   (id, coupon_id, buyer_auth0_id, order_id, discount_applied, reservation_id)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 RETURNING committed_at`,
			red.ID, red.CouponID, red.BuyerAuth0ID, red.OrderID, red.DiscountApplied, red.ReservationID,
		).Scan(&red.CommittedAt)
		if err != nil {
			// The UNIQUE (coupon_id, order_id) index protects idempotency
			// on Stripe webhook replays. App treats this as
			// "already committed" and swaps in the existing row.
			if isDuplicateRedemption(err) {
				return port.ErrDuplicateRedemption
			}
			return fmt.Errorf("insert redemption: %w", err)
		}
		return nil
	})
}

func (r *RedemptionRepository) GetByCouponAndOrder(ctx context.Context, couponID, orderID uuid.UUID) (*domain.CouponRedemption, error) {
	var (
		red   domain.CouponRedemption
		found bool
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, coupon_id, buyer_auth0_id, order_id, discount_applied, reservation_id, committed_at
			 FROM coupon_svc.coupon_redemptions
			 WHERE coupon_id = $1 AND order_id = $2`,
			couponID, orderID,
		).Scan(&red.ID, &red.CouponID, &red.BuyerAuth0ID, &red.OrderID, &red.DiscountApplied, &red.ReservationID, &red.CommittedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &red, nil
}

func (r *RedemptionRepository) ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.CouponRedemption, int, error) {
	var (
		rows  []domain.CouponRedemption
		total int
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM coupon_svc.coupon_redemptions WHERE buyer_auth0_id = $1`,
			buyerAuth0ID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count redemptions: %w", err)
		}
		if total == 0 {
			return nil
		}
		q, err := tx.Query(ctx,
			`SELECT id, coupon_id, buyer_auth0_id, order_id, discount_applied, reservation_id, committed_at
			 FROM coupon_svc.coupon_redemptions
			 WHERE buyer_auth0_id = $1
			 ORDER BY committed_at DESC
			 LIMIT $2 OFFSET $3`,
			buyerAuth0ID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("query redemptions: %w", err)
		}
		defer q.Close()
		for q.Next() {
			var red domain.CouponRedemption
			if err := q.Scan(&red.ID, &red.CouponID, &red.BuyerAuth0ID, &red.OrderID, &red.DiscountApplied, &red.ReservationID, &red.CommittedAt); err != nil {
				return err
			}
			rows = append(rows, red)
		}
		return q.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *RedemptionRepository) CountByBuyerAndCoupon(ctx context.Context, buyerAuth0ID string, couponID uuid.UUID) (int, error) {
	var count int
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM coupon_svc.coupon_redemptions
			 WHERE buyer_auth0_id = $1 AND coupon_id = $2`,
			buyerAuth0ID, couponID,
		).Scan(&count)
	})
	if err != nil {
		return 0, fmt.Errorf("count by buyer and coupon: %w", err)
	}
	return count, nil
}

func (r *RedemptionRepository) StatsByCoupon(ctx context.Context, couponID uuid.UUID) (int, int64, error) {
	var (
		count int
		total int64
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT COUNT(*), COALESCE(SUM(discount_applied), 0)
			 FROM coupon_svc.coupon_redemptions
			 WHERE coupon_id = $1`,
			couponID,
		).Scan(&count, &total)
	})
	if err != nil {
		return 0, 0, fmt.Errorf("stats by coupon: %w", err)
	}
	return count, total, nil
}

// MarkRefunded stamps refunded_at + refunded_reason on a redemption.
// The WHERE filters out already-refunded rows so a replayed cancel
// returns (false, nil) without clobbering the original timestamp.
func (r *RedemptionRepository) MarkRefunded(ctx context.Context, redemptionID uuid.UUID, reason string) (bool, error) {
	var affected bool
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE coupon_svc.coupon_redemptions
			 SET refunded_at = NOW(), refunded_reason = $2
			 WHERE id = $1 AND refunded_at IS NULL`,
			redemptionID, reason,
		)
		if err != nil {
			return fmt.Errorf("mark redemption refunded: %w", err)
		}
		affected = tag.RowsAffected() > 0
		return nil
	})
	return affected, err
}

// isDuplicateRedemption recognizes the (coupon_id, order_id) UNIQUE
// violation. Matching on the constraint name keeps us from catching an
// unrelated UNIQUE (e.g. a future PK on id).
func isDuplicateRedemption(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "coupon_redemptions_coupon_order_unique") ||
		(strings.Contains(msg, "duplicate key") && strings.Contains(msg, "coupon_redemptions"))
}
