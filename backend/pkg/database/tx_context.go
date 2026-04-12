package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

// WithTx returns a child context that carries tx. Repository methods call
// TxFromContext to reuse an outer tenant-scoped transaction started by the
// service layer, instead of opening a new independent transaction.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext retrieves the pgx.Tx stored by WithTx, if any.
// Returns (nil, false) when no transaction is in the context.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// TenantTxCtx is the context-propagating form of TenantTx. It embeds the
// opened transaction in the context before calling fn, so repository methods
// that receive the context can extract the tx via TxFromContext and join the
// outer transaction without receiving pgx.Tx as an explicit parameter.
//
// This is the preferred form for service-layer TxRunner interfaces because it
// keeps pgx out of service/port signatures.
func TenantTxCtx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(ctx context.Context) error) error {
	return TenantTx(ctx, pool, tenantID, func(tx pgx.Tx) error {
		return fn(WithTx(ctx, tx))
	})
}
