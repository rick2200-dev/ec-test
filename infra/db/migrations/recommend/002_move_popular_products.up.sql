-- Phase 2.2 (partial): rebuild popular_products in recommend_svc so the
-- matview reads from recommend_svc.user_events (where recommend now writes
-- its behavior events) instead of the abandoned catalog_svc.user_events.
--
-- On a fresh recommend-only DB (Phase 3 split), catalog_svc doesn't exist.
-- Create an empty placeholder matview instead — migration 005 drops and
-- recreates the matview against recommend_svc.products (no catalog refs),
-- so this placeholder is only visible between 002 and 005.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc') THEN
        CREATE MATERIALIZED VIEW IF NOT EXISTS recommend_svc.popular_products AS
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
        CREATE MATERIALIZED VIEW IF NOT EXISTS recommend_svc.popular_products AS
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
