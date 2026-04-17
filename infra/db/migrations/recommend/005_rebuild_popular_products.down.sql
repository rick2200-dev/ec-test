-- Restore the previous definition. On a split DB without catalog_svc,
-- create a placeholder (same as 002 does for fresh DBs).
DROP MATERIALIZED VIEW IF EXISTS recommend_svc.popular_products;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc') THEN
        CREATE MATERIALIZED VIEW recommend_svc.popular_products AS
        SELECT
            p.id AS product_id,
            p.seller_id,
            p.name,
            p.slug,
            pr.price_amount,
            pr.price_currency,
            COUNT(DISTINCT ue.id) FILTER (WHERE ue.event_type = 'purchased')       AS purchase_count,
            COUNT(DISTINCT ue.id) FILTER (WHERE ue.event_type = 'product_viewed')  AS view_count,
            COUNT(DISTINCT ue.id)                                                  AS total_events
        FROM catalog_svc.products p
        LEFT JOIN LATERAL (
            SELECT price_amount, price_currency
            FROM catalog_svc.skus
            WHERE product_id = p.id
            ORDER BY price_amount ASC
            LIMIT 1
        ) pr ON TRUE
        LEFT JOIN recommend_svc.user_events ue ON ue.product_id = p.id
        WHERE p.status = 'active'
        GROUP BY p.id, p.seller_id, p.name, p.slug, pr.price_amount, pr.price_currency;
    ELSE
        CREATE MATERIALIZED VIEW recommend_svc.popular_products AS
        SELECT
            NULL::UUID AS product_id,
            NULL::UUID AS seller_id,
            NULL::VARCHAR(500) AS name,
            NULL::VARCHAR(500) AS slug,
            NULL::BIGINT AS price_amount,
            NULL::VARCHAR(3) AS price_currency,
            0::BIGINT AS purchase_count,
            0::BIGINT AS view_count,
            0::BIGINT AS total_events
        WHERE FALSE;
    END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_recommend_popular_products
    ON recommend_svc.popular_products (product_id);

ALTER MATERIALIZED VIEW recommend_svc.popular_products OWNER TO recommend_role;
