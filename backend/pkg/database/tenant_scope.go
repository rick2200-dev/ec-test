package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantTx begins a transaction with the tenant_id session variable set for RLS.
// All queries within this transaction will be scoped to the given tenant.
func TenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// On the commit path Rollback is a no-op and returns
	// pgx.ErrTxClosed; we don't care about that. On an error path the
	// caller already has the real error from fn(tx).
	defer func() { _ = tx.Rollback(ctx) }()

	// Set the tenant_id session variable for Row-Level Security policies.
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID.String())); err != nil {
		return fmt.Errorf("set tenant_id: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
