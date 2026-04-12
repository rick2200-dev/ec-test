package repository

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresViewRefresher refreshes materialized views backed by Postgres.
// It satisfies service.ViewRefresher via structural typing.
type PostgresViewRefresher struct {
	pool *pgxpool.Pool
}

// NewPostgresViewRefresher creates a new PostgresViewRefresher.
func NewPostgresViewRefresher(pool *pgxpool.Pool) *PostgresViewRefresher {
	return &PostgresViewRefresher{pool: pool}
}

// RefreshPopularProducts refreshes the catalog_svc.popular_products
// materialized view concurrently.
func (r *PostgresViewRefresher) RefreshPopularProducts(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY catalog_svc.popular_products")
	if err != nil {
		slog.Error("failed to refresh popular_products view", "error", err)
	}
	return err
}
