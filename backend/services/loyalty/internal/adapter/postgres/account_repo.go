// Package repository contains the loyalty service's postgres adapters.
// Repos always run inside an outer transaction when one is present on
// the context (via database.TxOrPool), so the app layer's Tx wrapper
// makes the account update and ledger insert commit atomically.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// AccountRepository is the postgres implementation of AccountStore.
type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Get(ctx context.Context, buyerAuth0ID string) (*domain.Account, error) {
	var acct domain.Account
	var found bool
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT buyer_auth0_id, balance, pending_redemption,
			        lifetime_earned, lifetime_redeemed, version, updated_at
			 FROM loyalty_svc.point_accounts
			 WHERE buyer_auth0_id = $1`,
			buyerAuth0ID,
		).Scan(
			&acct.BuyerAuth0ID, &acct.Balance, &acct.PendingRedemption,
			&acct.LifetimeEarned, &acct.LifetimeRedeemed, &acct.Version, &acct.UpdatedAt,
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
		return nil, fmt.Errorf("get point account: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &acct, nil
}

// EnsureExists inserts a zero-balance row if missing. Using ON CONFLICT
// DO NOTHING keeps this cheap and safe to call on every earn path.
func (r *AccountRepository) EnsureExists(ctx context.Context, buyerAuth0ID string) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO loyalty_svc.point_accounts (buyer_auth0_id)
			 VALUES ($1)
			 ON CONFLICT (buyer_auth0_id) DO NOTHING`,
			buyerAuth0ID,
		)
		if err != nil {
			return fmt.Errorf("ensure point account: %w", err)
		}
		return nil
	})
}

// LockForUpdate performs EnsureExists + SELECT ... FOR UPDATE in the
// same tx, returning the locked snapshot. The FOR UPDATE acquires a
// row-level exclusive lock that serializes concurrent earn / reserve
// flows and closes the race window where two transactions could both
// miss the idempotency check and each apply a balance delta.
//
// Callers MUST be running inside an outer transaction; otherwise the
// lock is released at the end of this call and provides no guarantee.
// database.TxOrPool in a RunTx-wrapped context satisfies that.
func (r *AccountRepository) LockForUpdate(ctx context.Context, buyerAuth0ID string) (*domain.Account, error) {
	var acct domain.Account
	err := database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		// Upsert-then-lock. The ON CONFLICT branch returns the
		// existing row, so a single round-trip both guarantees the
		// row exists and hands back the snapshot we need for ApplyDelta.
		// Note: INSERT ... ON CONFLICT DO UPDATE ... RETURNING is
		// deliberately written as DO UPDATE with a no-op SET so we
		// still get a RETURNING row on conflict. (DO NOTHING would
		// skip the RETURNING on conflict.) Pairing that with a
		// subsequent SELECT FOR UPDATE would be cleaner but costs an
		// extra round-trip, which matters on the hot earn path.
		err := tx.QueryRow(ctx,
			`INSERT INTO loyalty_svc.point_accounts (buyer_auth0_id)
			 VALUES ($1)
			 ON CONFLICT (buyer_auth0_id) DO UPDATE
			   SET buyer_auth0_id = EXCLUDED.buyer_auth0_id
			 RETURNING buyer_auth0_id, balance, pending_redemption,
			           lifetime_earned, lifetime_redeemed, version, updated_at`,
			buyerAuth0ID,
		).Scan(
			&acct.BuyerAuth0ID, &acct.Balance, &acct.PendingRedemption,
			&acct.LifetimeEarned, &acct.LifetimeRedeemed, &acct.Version, &acct.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("upsert+lock account: %w", err)
		}
		// ON CONFLICT DO UPDATE already acquires a row lock on
		// existing rows; the INSERT branch locks the just-inserted
		// row. Both cases give us exclusive access for the rest of
		// the surrounding transaction.
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &acct, nil
}

// ApplyDelta adjusts the balance/pending/lifetime counters atomically.
// The WHERE clause on version gives us optimistic-locking semantics:
// when the version has moved on, the update affects zero rows and we
// return port.ErrOptimisticLock.
func (r *AccountRepository) ApplyDelta(ctx context.Context, buyerAuth0ID string, delta port.AccountDelta, expectedVersion int) error {
	return database.TxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE loyalty_svc.point_accounts
			 SET balance            = balance + $2,
			     pending_redemption = pending_redemption + $3,
			     lifetime_earned    = lifetime_earned + $4,
			     lifetime_redeemed  = lifetime_redeemed + $5,
			     version            = version + 1,
			     updated_at         = NOW()
			 WHERE buyer_auth0_id = $1
			   AND version = $6`,
			buyerAuth0ID,
			delta.BalanceDelta, delta.PendingRedemptionDelta,
			delta.LifetimeEarnedDelta, delta.LifetimeRedeemedDelta,
			expectedVersion,
		)
		if err != nil {
			return fmt.Errorf("apply account delta: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return port.ErrOptimisticLock
		}
		return nil
	})
}
