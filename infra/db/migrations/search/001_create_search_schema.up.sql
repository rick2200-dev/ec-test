-- Phase 2.2 (completion): search owns its own seller_plan_boost projection
-- instead of reading the catalog_svc materialized view. The view in
-- catalog_svc is dropped by a separate catalog migration once this table
-- is consuming subscription-events; the refresh() function call from the
-- subscription service has been replaced with event-driven updates.

CREATE SCHEMA IF NOT EXISTS search_svc;

CREATE TABLE IF NOT EXISTS search_svc.seller_plan_boost (
    seller_id         UUID PRIMARY KEY,
    plan_tier         INT          NOT NULL DEFAULT 0,
    plan_slug         VARCHAR(100) NOT NULL DEFAULT 'free',
    search_boost      DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    promoted_results  INT          NOT NULL DEFAULT 0,
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Backfill from the current (soon-to-be-dropped) catalog_svc view so
-- search queries see the same data during the transition. Next deploy
-- after a full promotion can drop the catalog view.
INSERT INTO search_svc.seller_plan_boost (seller_id, plan_tier, plan_slug, search_boost, promoted_results)
SELECT seller_id, plan_tier, plan_slug, search_boost, promoted_results
FROM catalog_svc.seller_plan_boost
ON CONFLICT (seller_id) DO NOTHING;
