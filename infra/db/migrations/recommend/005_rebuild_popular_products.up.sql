-- Phase 2.2: rebuild popular_products against the local recommend_svc
-- projection now that Phase 2.2's product slice has landed. Removes the
-- last cross-schema JOIN from the matview (previously it read
-- catalog_svc.products + catalog_svc.skus for name / slug / price).

DROP MATERIALIZED VIEW IF EXISTS recommend_svc.popular_products;

CREATE MATERIALIZED VIEW recommend_svc.popular_products AS
SELECT
    p.id AS product_id,
    p.seller_id,
    p.name,
    p.slug,
    p.cheapest_price_amount  AS price_amount,
    p.cheapest_price_currency AS price_currency,
    COUNT(DISTINCT ue.id) FILTER (WHERE ue.event_type = 'purchased')      AS purchase_count,
    COUNT(DISTINCT ue.id) FILTER (WHERE ue.event_type = 'product_viewed') AS view_count,
    COUNT(DISTINCT ue.id)                                                 AS total_events
FROM recommend_svc.products p
LEFT JOIN recommend_svc.user_events ue ON ue.product_id = p.id
WHERE p.status = 'active'
GROUP BY p.id, p.seller_id, p.name, p.slug, p.cheapest_price_amount, p.cheapest_price_currency;

CREATE UNIQUE INDEX IF NOT EXISTS idx_recommend_popular_products
    ON recommend_svc.popular_products (product_id);

-- Refresh ownership stays with recommend_role so the service can run
-- REFRESH MATERIALIZED VIEW CONCURRENTLY without superuser.
ALTER MATERIALIZED VIEW recommend_svc.popular_products OWNER TO recommend_role;
