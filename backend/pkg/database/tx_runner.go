package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolTxRunner wraps a *pgxpool.Pool and implements the consumer-side TxRunner
// interface defined in each service that needs transactions.
// Pass &database.PoolTxRunner{Pool: pool} wherever a TxRunner is expected.
type PoolTxRunner struct {
	Pool *pgxpool.Pool
}

// RunTx begins a transaction, embeds it in the context, and delegates to TxCtx.
// Repository methods that receive the returned context can extract the
// transaction via TxFromContext and join it without taking pgx.Tx as an
// explicit parameter.
func (r *PoolTxRunner) RunTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return TxCtx(ctx, r.Pool, fn)
}
