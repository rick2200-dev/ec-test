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
	defer tx.Rollback(ctx)

	// Set the tenant_id session variable for Row-Level Security policies.
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID.String())); err != nil {
		return fmt.Errorf("set tenant_id: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// TenantQuery executes a single query within a tenant-scoped transaction.
func TenantQuery(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, query string, args ...any) (pgx.Rows, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID.String())); err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("set tenant_id: %w", err)
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}

	// Note: caller must close rows, which will also commit the transaction
	return rows, nil
}
