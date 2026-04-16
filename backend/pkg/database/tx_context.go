package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

// WithTx returns a child context that carries tx. Repository methods call
// TxFromContext to reuse an outer transaction started by the service layer,
// instead of opening a new independent transaction.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext retrieves the pgx.Tx stored by WithTx, if any.
// Returns (nil, false) when no transaction is in the context.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// TxCtx is the context-propagating form of Tx. It embeds the opened
// transaction in the context before calling fn, so repository methods that
// receive the context can extract the tx via TxFromContext and join the outer
// transaction without receiving pgx.Tx as an explicit parameter.
//
// This is the preferred form for service-layer TxRunner interfaces because it
// keeps pgx out of service/port signatures.
func TxCtx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	return Tx(ctx, pool, func(tx pgx.Tx) error {
		return fn(WithTx(ctx, tx))
	})
}

// TxOrPool runs fn either in the caller's outer transaction (when one is
// carried on the context) or in a short auto-commit transaction opened on
// the pool. Repositories use this to stay agnostic of whether the service
// layer has wrapped multiple repo calls in a single Tx — critical for the
// outbox pattern where one business write and one outbox insert must
// commit atomically.
func TxOrPool(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	if tx, ok := TxFromContext(ctx); ok {
		return fn(tx)
	}
	return Tx(ctx, pool, fn)
}
