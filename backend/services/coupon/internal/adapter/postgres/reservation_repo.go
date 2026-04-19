package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// ReservationRepository is the postgres implementation of ReservationStore.
type ReservationRepository struct {
	pool *pgxpool.Pool
}

func NewReservationRepository(pool *pgxpool.Pool) *ReservationRepository {
	return &ReservationRepository{pool: pool}
}

func (r *ReservationRepository) Insert(ctx context.Context, res *domain.CouponReservation) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO coupon_svc.coupon_reservations
			   (id, coupon_id, buyer_auth0_id, order_candidate_id,
			    stripe_payment_intent_id, discount_amount, currency,
			    applicable_seller_id,
			    status, expires_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			 RETURNING created_at`,
			res.ID, res.CouponID, res.BuyerAuth0ID, res.OrderCandidateID,
			res.StripePaymentIntentID, res.DiscountAmount, res.Currency,
			res.ApplicableSellerID,
			string(res.Status), res.ExpiresAt,
		).Scan(&res.CreatedAt)
	})
}

func (r *ReservationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CouponReservation, error) {
	var (
		res    domain.CouponReservation
		status string
		found  bool
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, coupon_id, buyer_auth0_id, order_candidate_id,
			        stripe_payment_intent_id, discount_amount, currency,
			        applicable_seller_id,
			        status, expires_at, created_at, committed_at, released_at
			 FROM coupon_svc.coupon_reservations WHERE id = $1`,
			id,
		).Scan(
			&res.ID, &res.CouponID, &res.BuyerAuth0ID, &res.OrderCandidateID,
			&res.StripePaymentIntentID, &res.DiscountAmount, &res.Currency,
			&res.ApplicableSellerID,
			&status, &res.ExpiresAt, &res.CreatedAt, &res.CommittedAt, &res.ReleasedAt,
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
		return nil, fmt.Errorf("get reservation: %w", err)
	}
	if !found {
		return nil, nil
	}
	res.Status = domain.ReservationStatus(status)
	return &res, nil
}

// UpdateStatus uses a CASE so `committed_at` / `released_at` get
// stamped only on the matching transition. Returns false when the row
// wasn't in the expected state (caller treats as terminal/lost-race).
func (r *ReservationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, expect, next domain.ReservationStatus) (bool, error) {
	var affected bool
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE coupon_svc.coupon_reservations
			 SET status = $2,
			     committed_at = CASE WHEN $2 = 'committed' THEN NOW() ELSE committed_at END,
			     released_at  = CASE WHEN $2 IN ('released','expired') THEN NOW() ELSE released_at END
			 WHERE id = $1 AND status = $3`,
			id, string(next), string(expect),
		)
		if err != nil {
			return fmt.Errorf("update reservation status: %w", err)
		}
		affected = tag.RowsAffected() > 0
		return nil
	})
	if err != nil {
		return false, err
	}
	return affected, nil
}

// ClaimExpired is the reaper entrypoint. RETURNING gets both the row
// PK and the coupon_id so the caller can decrement usage_count in the
// same tx without a separate lookup.
func (r *ReservationRepository) ClaimExpired(ctx context.Context, limit int) ([]port.ReservationExpiry, error) {
	var out []port.ReservationExpiry
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`WITH claimed AS (
			     SELECT id FROM coupon_svc.coupon_reservations
			     WHERE status = 'pending' AND expires_at < NOW()
			     ORDER BY expires_at ASC
			     LIMIT $1
			     FOR UPDATE SKIP LOCKED
			 )
			 UPDATE coupon_svc.coupon_reservations cr
			 SET status = 'expired', released_at = NOW()
			 FROM claimed
			 WHERE cr.id = claimed.id
			 RETURNING cr.id, cr.coupon_id`,
			limit,
		)
		if err != nil {
			return fmt.Errorf("claim expired reservations: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var e port.ReservationExpiry
			if err := rows.Scan(&e.ReservationID, &e.CouponID); err != nil {
				return err
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	return out, err
}

func (r *ReservationRepository) CountPendingByCoupon(ctx context.Context, couponID uuid.UUID) (int, error) {
	var count int
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM coupon_svc.coupon_reservations
			 WHERE coupon_id = $1 AND status = 'pending'`,
			couponID,
		).Scan(&count)
	})
	if err != nil {
		return 0, fmt.Errorf("count pending reservations: %w", err)
	}
	return count, nil
}
