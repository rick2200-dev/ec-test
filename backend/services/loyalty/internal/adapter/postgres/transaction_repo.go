package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// TransactionRepository is the postgres implementation of
// TransactionStore backed by loyalty_svc.point_transactions.
type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

// Insert appends a ledger row. A UNIQUE violation on the idempotency key
// surfaces as port.ErrDuplicateIdempotencyKey so the app layer can
// treat the call as already-done.
func (r *TransactionRepository) Insert(ctx context.Context, t *domain.Transaction) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO loyalty_svc.point_transactions
			   (id, buyer_auth0_id, type, amount, balance_after,
			    source_type, source_id, reservation_id, expires_at, note)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 RETURNING created_at`,
			t.ID, t.BuyerAuth0ID, string(t.Type), t.Amount, t.BalanceAfter,
			string(t.SourceType), t.SourceID, t.ReservationID, t.ExpiresAt, t.Note,
		).Scan(&t.CreatedAt)
		if err != nil {
			// pgx surfaces "23505 unique_violation" via the SQLSTATE. We
			// match on the constraint name to avoid catching unrelated
			// conflicts (e.g. a future UNIQUE on id).
			if isDuplicateIdempotencyKey(err) {
				return port.ErrDuplicateIdempotencyKey
			}
			return fmt.Errorf("insert point transaction: %w", err)
		}
		return nil
	})
}

func (r *TransactionRepository) ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Transaction, int, error) {
	var (
		results []domain.Transaction
		total   int
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM loyalty_svc.point_transactions
			 WHERE buyer_auth0_id = $1`,
			buyerAuth0ID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count transactions: %w", err)
		}
		if total == 0 {
			return nil
		}

		rows, err := tx.Query(ctx,
			`SELECT id, buyer_auth0_id, type, amount, balance_after,
			        source_type, source_id, reservation_id, expires_at, note, created_at
			 FROM loyalty_svc.point_transactions
			 WHERE buyer_auth0_id = $1
			 ORDER BY created_at DESC
			 LIMIT $2 OFFSET $3`,
			buyerAuth0ID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("query transactions: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var (
				t             domain.Transaction
				typ, srcType  string
				reservationID *string
				note          *string
			)
			if err := rows.Scan(
				&t.ID, &t.BuyerAuth0ID, &typ, &t.Amount, &t.BalanceAfter,
				&srcType, &t.SourceID, &reservationID, &t.ExpiresAt, &note, &t.CreatedAt,
			); err != nil {
				return fmt.Errorf("scan transaction: %w", err)
			}
			t.Type = domain.TransactionType(typ)
			t.SourceType = domain.SourceType(srcType)
			if reservationID != nil {
				parsed, parseErr := parseUUID(*reservationID)
				if parseErr != nil {
					return fmt.Errorf("parse reservation_id: %w", parseErr)
				}
				t.ReservationID = &parsed
			}
			if note != nil {
				t.Note = *note
			}
			results = append(results, t)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *TransactionRepository) GetByIdempotency(ctx context.Context, sourceType domain.SourceType, sourceID string, txType domain.TransactionType) (*domain.Transaction, error) {
	var (
		t             domain.Transaction
		typ, srcType  string
		reservationID *string
		note          *string
		found         bool
	)
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, buyer_auth0_id, type, amount, balance_after,
			        source_type, source_id, reservation_id, expires_at, note, created_at
			 FROM loyalty_svc.point_transactions
			 WHERE source_type = $1 AND source_id = $2 AND type = $3`,
			string(sourceType), sourceID, string(txType),
		).Scan(
			&t.ID, &t.BuyerAuth0ID, &typ, &t.Amount, &t.BalanceAfter,
			&srcType, &t.SourceID, &reservationID, &t.ExpiresAt, &note, &t.CreatedAt,
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
		return nil, fmt.Errorf("get transaction by idempotency: %w", err)
	}
	if !found {
		return nil, nil
	}
	t.Type = domain.TransactionType(typ)
	t.SourceType = domain.SourceType(srcType)
	if reservationID != nil {
		parsed, parseErr := parseUUID(*reservationID)
		if parseErr != nil {
			return nil, fmt.Errorf("parse reservation_id: %w", parseErr)
		}
		t.ReservationID = &parsed
	}
	if note != nil {
		t.Note = *note
	}
	return &t, nil
}

// ListByBuyerChronological returns every ledger row for a buyer in
// true insertion order. Used by the expiration reaper to replay the
// full ledger and compute per-earn-row remaining balance.
//
// Order is driven by ledger_seq (migration 004): created_at defaults
// to NOW() which collides for rows written inside the same tx, and
// the id fallback (UUID) is random. Both combined would break FIFO
// accounting for any path that writes multiple ledger rows in one
// transaction (today that includes CommitPointsReservation, which
// emits both a redeem and an earn row).
func (r *TransactionRepository) ListByBuyerChronological(ctx context.Context, buyerAuth0ID string) ([]domain.Transaction, error) {
	var results []domain.Transaction
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, buyer_auth0_id, type, amount, balance_after,
			        source_type, source_id, reservation_id, expires_at, note, created_at
			 FROM loyalty_svc.point_transactions
			 WHERE buyer_auth0_id = $1
			 ORDER BY ledger_seq ASC`,
			buyerAuth0ID,
		)
		if err != nil {
			return fmt.Errorf("query chronological: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var (
				t             domain.Transaction
				typ, srcType  string
				reservationID *string
				note          *string
			)
			if err := rows.Scan(
				&t.ID, &t.BuyerAuth0ID, &typ, &t.Amount, &t.BalanceAfter,
				&srcType, &t.SourceID, &reservationID, &t.ExpiresAt, &note, &t.CreatedAt,
			); err != nil {
				return fmt.Errorf("scan transaction: %w", err)
			}
			t.Type = domain.TransactionType(typ)
			t.SourceType = domain.SourceType(srcType)
			if reservationID != nil {
				parsed, parseErr := parseUUID(*reservationID)
				if parseErr != nil {
					return fmt.Errorf("parse reservation_id: %w", parseErr)
				}
				t.ReservationID = &parsed
			}
			if note != nil {
				t.Note = *note
			}
			results = append(results, t)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ListBuyersWithExpiredUnprocessedEarn returns up to `limit` distinct
// buyer_auth0_ids that have at least one earn row where expires_at has
// passed and no corresponding expire row has been written yet. LEFT
// JOIN + IS NULL tends to plan better than NOT EXISTS here because the
// partial index on expiring earns is small, and the planner can hash-
// anti-join the expire rows by source_id.
func (r *TransactionRepository) ListBuyersWithExpiredUnprocessedEarn(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	var buyers []string
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT DISTINCT pt.buyer_auth0_id
			 FROM loyalty_svc.point_transactions pt
			 LEFT JOIN loyalty_svc.point_transactions ex
			     ON ex.source_type = 'expiration_job'
			    AND ex.source_id   = pt.id::text
			    AND ex.type        = 'expire'
			 WHERE pt.type        = 'earn'
			   AND pt.expires_at IS NOT NULL
			   AND pt.expires_at  < $1
			   AND ex.id IS NULL
			 LIMIT $2`,
			now, limit,
		)
		if err != nil {
			return fmt.Errorf("query expired earn buyers: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var b string
			if err := rows.Scan(&b); err != nil {
				return fmt.Errorf("scan buyer: %w", err)
			}
			buyers = append(buyers, b)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return buyers, nil
}

// isDuplicateIdempotencyKey recognizes the unique-violation on the
// (source_type, source_id, type) index. pgx wraps the error, so we
// string-match rather than import the pgconn error type at this layer.
func isDuplicateIdempotencyKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "point_transactions_idem_unique") ||
		(strings.Contains(msg, "duplicate key") && strings.Contains(msg, "point_transactions"))
}
