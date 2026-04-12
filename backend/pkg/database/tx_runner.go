package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolTxRunner wraps a *pgxpool.Pool and implements the consumer-side TxRunner
// interface defined in each service that needs tenant-scoped transactions.
// Pass &database.PoolTxRunner{Pool: pool} wherever a TxRunner is expected.
type PoolTxRunner struct {
	Pool *pgxpool.Pool
}

// RunTenantTx begins a tenant-scoped transaction, embeds it in the context,
// and delegates to TenantTxCtx. Repository methods that receive the returned
// context can extract the transaction via TxFromContext and join it without
// taking pgx.Tx as an explicit parameter.
func (r *PoolTxRunner) RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error {
	return TenantTxCtx(ctx, r.Pool, tenantID, fn)
}
