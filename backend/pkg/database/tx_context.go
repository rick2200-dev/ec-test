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
