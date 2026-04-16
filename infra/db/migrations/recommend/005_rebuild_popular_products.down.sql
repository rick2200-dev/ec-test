-- Restore the previous definition (reads catalog_svc.products + skus).
DROP MATERIALIZED VIEW IF EXISTS recommend_svc.popular_products;

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

CREATE UNIQUE INDEX IF NOT EXISTS idx_recommend_popular_products
    ON recommend_svc.popular_products (product_id);

ALTER MATERIALIZED VIEW recommend_svc.popular_products OWNER TO recommend_role;
