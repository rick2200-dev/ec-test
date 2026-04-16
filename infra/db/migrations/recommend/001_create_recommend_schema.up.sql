-- Phase 2.2 (partial): recommend takes ownership of its own user_events
-- table instead of writing into catalog_svc.user_events. The legacy
-- catalog_svc.user_events created by catalog/003_create_recommendations
-- stays in place for rollback safety — a future migration can drop it once
-- all environments have promoted this change.
--
-- The popular_products materialized view in catalog_svc still reads from
-- catalog_svc.user_events today; that matview will be replaced (or moved)
-- in a follow-up step.

CREATE SCHEMA IF NOT EXISTS recommend_svc;

CREATE TABLE IF NOT EXISTS recommend_svc.user_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    VARCHAR(255) NOT NULL,
    event_type VARCHAR(50)  NOT NULL,
    product_id UUID         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recommend_user_events_user    ON recommend_svc.user_events (user_id);
CREATE INDEX IF NOT EXISTS idx_recommend_user_events_product ON recommend_svc.user_events (product_id);
CREATE INDEX IF NOT EXISTS idx_recommend_user_events_type    ON recommend_svc.user_events (event_type);
CREATE INDEX IF NOT EXISTS idx_recommend_user_events_created ON recommend_svc.user_events (created_at);
