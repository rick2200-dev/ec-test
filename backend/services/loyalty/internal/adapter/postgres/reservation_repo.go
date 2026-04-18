package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// ReservationRepository backs the loyalty_svc.point_reservations table.
type ReservationRepository struct {
	pool *pgxpool.Pool
}

func NewReservationRepository(pool *pgxpool.Pool) *ReservationRepository {
	return &ReservationRepository{pool: pool}
}

func (r *ReservationRepository) Insert(ctx context.Context, res *domain.Reservation) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO loyalty_svc.point_reservations
			   (id, buyer_auth0_id, amount, order_candidate_id,
			    stripe_payment_intent_id, status, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING created_at`,
			res.ID, res.BuyerAuth0ID, res.Amount, res.OrderCandidateID,
			res.StripePaymentIntentID, string(res.Status), res.ExpiresAt,
		).Scan(&res.CreatedAt)
	})
}

func (r *ReservationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Reservation, error) {
	var (
		res    domain.Reservation
		status string
		found  bool
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, buyer_auth0_id, amount, order_candidate_id,
			        stripe_payment_intent_id, status, expires_at,
			        created_at, committed_at, released_at
			 FROM loyalty_svc.point_reservations WHERE id = $1`,
			id,
		).Scan(
			&res.ID, &res.BuyerAuth0ID, &res.Amount, &res.OrderCandidateID,
			&res.StripePaymentIntentID, &status, &res.ExpiresAt,
			&res.CreatedAt, &res.CommittedAt, &res.ReleasedAt,
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

// UpdateStatus stamps committed_at / released_at in the same update
// so callers don't have to sequence two writes.
func (r *ReservationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, expect, next domain.ReservationStatus) (bool, error) {
	var affected bool
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE loyalty_svc.point_reservations
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

// ClaimExpired flips stale pending rows to expired and returns
// (id, buyer, amount) so the reaper can adjust pending_redemption in
// one pass. SKIP LOCKED avoids thrashing with concurrent reapers.
func (r *ReservationRepository) ClaimExpired(ctx context.Context, limit int) ([]port.ReservationExpiry, error) {
	var out []port.ReservationExpiry
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`WITH claimed AS (
			     SELECT id FROM loyalty_svc.point_reservations
			     WHERE status = 'pending' AND expires_at < NOW()
			     ORDER BY expires_at ASC
			     LIMIT $1
			     FOR UPDATE SKIP LOCKED
			 )
			 UPDATE loyalty_svc.point_reservations r
			 SET status = 'expired', released_at = NOW()
			 FROM claimed
			 WHERE r.id = claimed.id
			 RETURNING r.id, r.buyer_auth0_id, r.amount`,
			limit,
		)
		if err != nil {
			return fmt.Errorf("claim expired reservations: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var e port.ReservationExpiry
			if err := rows.Scan(&e.ReservationID, &e.BuyerAuth0ID, &e.Amount); err != nil {
				return err
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	return out, err
}
