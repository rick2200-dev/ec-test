-- Materialized view for fast search-time lookup of seller plan boost factors.
-- On a fresh catalog-only DB (Phase 3 split), auth_svc doesn't exist.
-- Create a placeholder matview instead — 007 drops it unconditionally
-- (search owns its own projection now).

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth_svc') THEN
        CREATE MATERIALIZED VIEW IF NOT EXISTS catalog_svc.seller_plan_boost AS
        SELECT
            s.id AS seller_id,
            COALESCE(sp.tier, 0) AS plan_tier,
            COALESCE(sp.slug, 'free') AS plan_slug,
            COALESCE((sp.features->>'search_boost')::float, 1.0) AS search_boost,
            COALESCE((sp.features->>'promoted_results')::int, 0) AS promoted_results
        FROM auth_svc.sellers s
        LEFT JOIN auth_svc.seller_subscriptions ss
            ON ss.seller_id = s.id AND ss.status = 'active'
        LEFT JOIN auth_svc.subscription_plans sp
            ON sp.id = ss.plan_id;
    ELSE
        CREATE MATERIALIZED VIEW IF NOT EXISTS catalog_svc.seller_plan_boost AS
        SELECT
            NULL::UUID AS seller_id,
            0 AS plan_tier,
            'free'::VARCHAR AS plan_slug,
            1.0::DOUBLE PRECISION AS search_boost,
            0 AS promoted_results
        WHERE FALSE;
    END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_seller_plan_boost_pk
    ON catalog_svc.seller_plan_boost(seller_id);
