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

// RefreshPopularProducts refreshes the recommend_svc.popular_products
// materialized view concurrently. The matview aggregates user events
// (owned by recommend) against catalog_svc.products + skus; running the
// refresh from the recommend service keeps the rebuild cadence tied to
// the team that consumes the output.
func (r *PostgresViewRefresher) RefreshPopularProducts(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY recommend_svc.popular_products")
	if err != nil {
		slog.Error("failed to refresh popular_products view", "error", err)
	}
	return err
}
