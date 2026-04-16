-- Phase 2.2 (partial): rebuild popular_products in recommend_svc so the
-- matview reads from recommend_svc.user_events (where recommend now writes
-- its behavior events) instead of the abandoned catalog_svc.user_events.
-- Also price_amount / price_currency are pulled from catalog_svc.skus (the
-- real price carrier; catalog_svc.products has none), using the cheapest
-- SKU as the representative price for the product.
--
-- The catalog_svc.popular_products matview created by
-- catalog/003_create_recommendations stays in place for rollback safety
-- and is dropped in a follow-up once this schema is promoted everywhere.

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

CREATE UNIQUE INDEX IF NOT EXISTS idx_recommend_popular_products
    ON recommend_svc.popular_products (product_id);

-- REFRESH MATERIALIZED VIEW CONCURRENTLY requires the caller to own the
-- view. Transfer ownership to recommend_role so the service (not the
-- ecmarket superuser) can schedule refreshes.
ALTER MATERIALIZED VIEW recommend_svc.popular_products OWNER TO recommend_role;
