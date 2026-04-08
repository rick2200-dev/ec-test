package engine

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
)

// PostgresEngine implements RecommendEngine using PostgreSQL queries.
type PostgresEngine struct {
	pool *pgxpool.Pool
}

// NewPostgresEngine creates a new PostgreSQL-based recommendation engine.
func NewPostgresEngine(pool *pgxpool.Pool) *PostgresEngine {
	return &PostgresEngine{pool: pool}
}

// Recommend dispatches to the appropriate query based on recommendation type.
func (e *PostgresEngine) Recommend(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
	switch req.Type {
	case domain.Popular:
		return e.popular(ctx, req)
	case domain.Similar:
		return e.similar(ctx, req)
	case domain.PersonalizedForYou:
		return e.personalizedForYou(ctx, req)
	case domain.FrequentlyBoughtTogether:
		return e.frequentlyBoughtTogether(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported recommendation type: %s", req.Type)
	}
}

// popular returns products ordered by purchase_count + view_count*0.1 from the
// materialized view.
func (e *PostgresEngine) popular(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
	query := `
		SELECT pp.product_id, pp.tenant_id, pp.seller_id, pp.name, pp.slug,
		       COALESCE(p.price_amount, 0), COALESCE(p.price_currency, 'JPY'),
		       (pp.purchase_count + pp.view_count * 0.1) AS score
		FROM catalog_svc.popular_products pp
		JOIN catalog_svc.products p ON p.id = pp.product_id AND p.tenant_id = pp.tenant_id
		WHERE pp.tenant_id = $1
		ORDER BY score DESC
		LIMIT $2
	`

	rows, err := e.pool.Query(ctx, query, req.TenantID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("query popular products: %w", err)
	}
	defer rows.Close()

	var products []domain.RecommendedProduct
	for rows.Next() {
		var rp domain.RecommendedProduct
		if err := rows.Scan(
			&rp.ID, &rp.TenantID, &rp.SellerID, &rp.Name, &rp.Slug,
			&rp.PriceAmount, &rp.PriceCurrency, &rp.Score,
		); err != nil {
			return nil, fmt.Errorf("scan popular product: %w", err)
		}
		rp.Reason = "Popular product"
		products = append(products, rp)
	}

	return &domain.RecommendResponse{Products: products}, nil
}

// similar finds products in the same categories as the given product, excluding
// the product itself, ordered by popularity.
func (e *PostgresEngine) similar(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
	if req.ProductID == nil {
		return nil, fmt.Errorf("product_id is required for similar recommendations")
	}

	query := `
		SELECT DISTINCT p.id, p.tenant_id, p.seller_id, p.name, p.slug,
		       p.price_amount, p.price_currency,
		       COALESCE(pp.purchase_count + pp.view_count * 0.1, 0) AS score
		FROM catalog_svc.products p
		JOIN catalog_svc.product_categories pc ON pc.product_id = p.id
		JOIN catalog_svc.product_categories pc2 ON pc2.category_id = pc.category_id
		LEFT JOIN catalog_svc.popular_products pp ON pp.product_id = p.id AND pp.tenant_id = p.tenant_id
		WHERE pc2.product_id = $1
		  AND p.id != $1
		  AND p.tenant_id = $2
		  AND p.status = 'active'
		ORDER BY score DESC
		LIMIT $3
	`

	rows, err := e.pool.Query(ctx, query, req.ProductID, req.TenantID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("query similar products: %w", err)
	}
	defer rows.Close()

	var products []domain.RecommendedProduct
	for rows.Next() {
		var rp domain.RecommendedProduct
		if err := rows.Scan(
			&rp.ID, &rp.TenantID, &rp.SellerID, &rp.Name, &rp.Slug,
			&rp.PriceAmount, &rp.PriceCurrency, &rp.Score,
		); err != nil {
			return nil, fmt.Errorf("scan similar product: %w", err)
		}
		rp.Reason = "Similar product based on shared categories"
		products = append(products, rp)
	}

	return &domain.RecommendResponse{Products: products}, nil
}

// personalizedForYou finds products in categories the user has viewed but not
// purchased, weighted by view frequency.
func (e *PostgresEngine) personalizedForYou(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
	query := `
		WITH user_viewed_categories AS (
			SELECT pc.category_id, COUNT(*) AS view_weight
			FROM catalog_svc.user_events ue
			JOIN catalog_svc.product_categories pc ON pc.product_id = ue.product_id
			WHERE ue.tenant_id = $1
			  AND ue.user_id = $2
			  AND ue.event_type = 'product_viewed'
			GROUP BY pc.category_id
		),
		user_purchased AS (
			SELECT DISTINCT ue.product_id
			FROM catalog_svc.user_events ue
			WHERE ue.tenant_id = $1
			  AND ue.user_id = $2
			  AND ue.event_type = 'purchased'
		)
		SELECT p.id, p.tenant_id, p.seller_id, p.name, p.slug,
		       p.price_amount, p.price_currency,
		       SUM(uvc.view_weight) AS score
		FROM catalog_svc.products p
		JOIN catalog_svc.product_categories pc ON pc.product_id = p.id
		JOIN user_viewed_categories uvc ON uvc.category_id = pc.category_id
		LEFT JOIN user_purchased up ON up.product_id = p.id
		WHERE p.tenant_id = $1
		  AND p.status = 'active'
		  AND up.product_id IS NULL
		GROUP BY p.id, p.tenant_id, p.seller_id, p.name, p.slug, p.price_amount, p.price_currency
		ORDER BY score DESC
		LIMIT $3
	`

	rows, err := e.pool.Query(ctx, query, req.TenantID, req.UserID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("query personalized products: %w", err)
	}
	defer rows.Close()

	var products []domain.RecommendedProduct
	for rows.Next() {
		var rp domain.RecommendedProduct
		if err := rows.Scan(
			&rp.ID, &rp.TenantID, &rp.SellerID, &rp.Name, &rp.Slug,
			&rp.PriceAmount, &rp.PriceCurrency, &rp.Score,
		); err != nil {
			return nil, fmt.Errorf("scan personalized product: %w", err)
		}
		rp.Reason = "Based on your browsing history"
		products = append(products, rp)
	}

	return &domain.RecommendResponse{Products: products}, nil
}

// frequentlyBoughtTogether finds products that other buyers also purchased
// alongside the given product.
func (e *PostgresEngine) frequentlyBoughtTogether(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
	if req.ProductID == nil {
		return nil, fmt.Errorf("product_id is required for frequently_bought_together recommendations")
	}

	query := `
		WITH co_buyers AS (
			SELECT DISTINCT ue.user_id
			FROM catalog_svc.user_events ue
			WHERE ue.tenant_id = $1
			  AND ue.product_id = $2
			  AND ue.event_type = 'purchased'
		)
		SELECT p.id, p.tenant_id, p.seller_id, p.name, p.slug,
		       p.price_amount, p.price_currency,
		       COUNT(DISTINCT ue2.user_id)::float8 AS score
		FROM catalog_svc.user_events ue2
		JOIN co_buyers cb ON cb.user_id = ue2.user_id
		JOIN catalog_svc.products p ON p.id = ue2.product_id AND p.tenant_id = ue2.tenant_id
		WHERE ue2.tenant_id = $1
		  AND ue2.event_type = 'purchased'
		  AND ue2.product_id != $2
		  AND p.status = 'active'
		GROUP BY p.id, p.tenant_id, p.seller_id, p.name, p.slug, p.price_amount, p.price_currency
		ORDER BY score DESC
		LIMIT $3
	`

	rows, err := e.pool.Query(ctx, query, req.TenantID, req.ProductID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("query frequently bought together: %w", err)
	}
	defer rows.Close()

	var products []domain.RecommendedProduct
	for rows.Next() {
		var rp domain.RecommendedProduct
		if err := rows.Scan(
			&rp.ID, &rp.TenantID, &rp.SellerID, &rp.Name, &rp.Slug,
			&rp.PriceAmount, &rp.PriceCurrency, &rp.Score,
		); err != nil {
			return nil, fmt.Errorf("scan frequently bought product: %w", err)
		}
		rp.Reason = "Frequently bought together"
		products = append(products, rp)
	}

	return &domain.RecommendResponse{Products: products}, nil
}

// RecordEvent inserts a user behavior event into the user_events table.
func (e *PostgresEngine) RecordEvent(ctx context.Context, event domain.UserEvent) error {
	query := `
		INSERT INTO catalog_svc.user_events (tenant_id, user_id, event_type, product_id)
		VALUES ($1, $2, $3, $4)
	`

	_, err := e.pool.Exec(ctx, query, event.TenantID, event.UserID, event.EventType, event.ProductID)
	if err != nil {
		return fmt.Errorf("insert user event: %w", err)
	}
	return nil
}
