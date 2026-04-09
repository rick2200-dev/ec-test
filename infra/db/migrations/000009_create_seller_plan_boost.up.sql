-- Materialized view for fast search-time lookup of seller plan boost factors.
-- Lives in catalog_svc so the search engine can JOIN without cross-schema complexity.
CREATE MATERIALIZED VIEW catalog_svc.seller_plan_boost AS
SELECT
    s.tenant_id,
    s.id AS seller_id,
    COALESCE(sp.tier, 0) AS plan_tier,
    COALESCE(sp.slug, 'free') AS plan_slug,
    COALESCE((sp.features->>'search_boost')::float, 1.0) AS search_boost,
    COALESCE((sp.features->>'promoted_results')::int, 0) AS promoted_results
FROM auth_svc.sellers s
LEFT JOIN auth_svc.seller_subscriptions ss
    ON ss.seller_id = s.id AND ss.tenant_id = s.tenant_id AND ss.status = 'active'
LEFT JOIN auth_svc.subscription_plans sp
    ON sp.id = ss.plan_id AND sp.tenant_id = s.tenant_id;

-- Unique index required for REFRESH MATERIALIZED VIEW CONCURRENTLY.
CREATE UNIQUE INDEX idx_seller_plan_boost_pk
    ON catalog_svc.seller_plan_boost(tenant_id, seller_id);
