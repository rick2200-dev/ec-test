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
			`SELECT id, coupon_id, buyer_auth0_id, order_id, discount_applied, refunded_amount,
			        reservation_id, committed_at, refunded_at, refunded_reason
			 FROM coupon_svc.coupon_redemptions
			 WHERE coupon_id = $1 AND order_id = $2`,
			couponID, orderID,
		).Scan(
			&red.ID, &red.CouponID, &red.BuyerAuth0ID, &red.OrderID,
			&red.DiscountApplied, &red.RefundedAmount,
			&red.ReservationID, &red.CommittedAt, &red.RefundedAt, &red.RefundedReason,
		)
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

// GetByReservationID loads the cart's redemption via its reservation
// foreign key. Used by cancellation: each seller order in the cart
// carries the same coupon_reservation_id on its OrderCancelled event,
// so any of them can find the redemption without knowing the anchor
// order_id.
func (r *RedemptionRepository) GetByReservationID(ctx context.Context, reservationID uuid.UUID) (*domain.CouponRedemption, error) {
	var (
		red   domain.CouponRedemption
		found bool
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, coupon_id, buyer_auth0_id, order_id, discount_applied, refunded_amount,
			        reservation_id, committed_at, refunded_at, refunded_reason
			 FROM coupon_svc.coupon_redemptions
			 WHERE reservation_id = $1`,
			reservationID,
		).Scan(
			&red.ID, &red.CouponID, &red.BuyerAuth0ID, &red.OrderID,
			&red.DiscountApplied, &red.RefundedAmount,
			&red.ReservationID, &red.CommittedAt, &red.RefundedAt, &red.RefundedReason,
		)
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
			`SELECT id, coupon_id, buyer_auth0_id, order_id, discount_applied, refunded_amount,
			        reservation_id, committed_at, refunded_at, refunded_reason
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
			if err := q.Scan(
				&red.ID, &red.CouponID, &red.BuyerAuth0ID, &red.OrderID,
				&red.DiscountApplied, &red.RefundedAmount,
				&red.ReservationID, &red.CommittedAt, &red.RefundedAt, &red.RefundedReason,
			); err != nil {
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

// ApplyPartialRefund increments refunded_amount by the caller-supplied
// share. It caps the new total at discount_applied (defense against a
// caller passing a too-large share) and, on the transition that brings
// refunded_amount equal to discount_applied, stamps refunded_at so the
// app layer can release the coupon seat exactly once.
//
// Returns:
//   - applied:        the amount actually added to refunded_amount
//     (may be smaller than share when another concurrent call or the
//     cap already exhausted the remaining discount)
//   - fullyRefunded:  whether this call brought the redemption to a
//     fully-refunded state (→ caller decrements usage_count)
//   - alreadyFull:    refunded_at was already set before this call
//     (replayed cancel delivery — idempotent no-op)
//
// The whole update is a single atomic UPDATE ... RETURNING, so racing
// cancel deliveries are serialized at the row level.
func (r *RedemptionRepository) ApplyPartialRefund(ctx context.Context, redemptionID uuid.UUID, share int64, reason string) (applied int64, fullyRefunded, alreadyFull bool, err error) {
	if share <= 0 {
		return 0, false, false, nil
	}
	err = database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		// Use LEAST() to cap the increment at what's still unrefunded
		// on the row; a single UPDATE is safer than read-modify-write
		// under concurrent cancel deliveries.
		var (
			oldRefunded int64
			newRefunded int64
			applied_    int64
			fully       bool
			wasFull     bool
		)
		rowErr := tx.QueryRow(ctx,
			`WITH prev AS (
			     SELECT discount_applied, refunded_amount, refunded_at IS NOT NULL AS was_full
			       FROM coupon_svc.coupon_redemptions
			      WHERE id = $1
			 ),
			 upd AS (
			     UPDATE coupon_svc.coupon_redemptions
			        SET refunded_amount = LEAST(refunded_amount + $2, discount_applied),
			            refunded_at     = CASE
			                WHEN refunded_at IS NULL
			                     AND LEAST(refunded_amount + $2, discount_applied) >= discount_applied
			                THEN NOW()
			                ELSE refunded_at
			            END,
			            refunded_reason = CASE
			                WHEN refunded_at IS NULL
			                     AND LEAST(refunded_amount + $2, discount_applied) >= discount_applied
			                THEN $3
			                ELSE refunded_reason
			            END
			      WHERE id = $1 AND refunded_at IS NULL
			  RETURNING refunded_amount, refunded_at IS NOT NULL AS now_full
			 )
			 SELECT prev.refunded_amount,
			        COALESCE(upd.refunded_amount, prev.refunded_amount),
			        COALESCE(upd.now_full, false),
			        prev.was_full
			   FROM prev LEFT JOIN upd ON TRUE`,
			redemptionID, share, reason,
		).Scan(&oldRefunded, &newRefunded, &fully, &wasFull)
		if rowErr != nil {
			return fmt.Errorf("apply partial refund: %w", rowErr)
		}
		alreadyFull = wasFull
		if !wasFull {
			applied_ = newRefunded - oldRefunded
			fullyRefunded = fully
		}
		applied = applied_
		return nil
	})
	return applied, fullyRefunded, alreadyFull, err
}

// InsertRefundEvent writes a per-(redemption, cancelled_order) row.
// UNIQUE (redemption_id, cancelled_order_id) makes this the arbiter
// of per-order refund idempotency: a redelivered order.cancelled for
// the same order hits the constraint and the caller returns
// AlreadyRefunded without re-applying the share.
func (r *RedemptionRepository) InsertRefundEvent(ctx context.Context, redemptionID, cancelledOrderID uuid.UUID, shareApplied int64, reason string) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO coupon_svc.coupon_refund_events
			   (redemption_id, cancelled_order_id, share_applied, reason)
			 VALUES ($1, $2, $3, $4)`,
			redemptionID, cancelledOrderID, shareApplied, reason,
		)
		if err != nil {
			if isDuplicateRefundEvent(err) {
				return port.ErrDuplicateRefundEvent
			}
			return fmt.Errorf("insert refund event: %w", err)
		}
		return nil
	})
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

// isDuplicateRefundEvent recognizes the (redemption_id, cancelled_order_id)
// UNIQUE violation on coupon_refund_events. Narrows the string match
// to the expected constraint name so unrelated PK conflicts surface
// as the real error.
func isDuplicateRefundEvent(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "coupon_refund_events_per_order_unique") ||
		(strings.Contains(msg, "duplicate key") && strings.Contains(msg, "coupon_refund_events"))
}
