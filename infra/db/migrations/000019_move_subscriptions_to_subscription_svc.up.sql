-- Phase 2 of auth refactor: relocate seller/buyer plan + subscription tables
-- from auth_svc to a dedicated subscription_svc schema. The REST / gRPC
-- surface is preserved — callers (gateway, order) are rewired in application
-- code separately.
--
-- Ordering note (why everything is in one migration):
--   1. The new subscription_svc.* tables go under FORCE ROW LEVEL SECURITY
--      like every other tenant-scoped table (see 000015).
--   2. catalog_svc.seller_plan_boost joins those tables across all tenants,
--      so it can only be created + refreshed by a role that holds BYPASSRLS.
--   3. If the MV were created by `ecmarket` (the migrator + app role) after
--      FORCE RLS is applied, CREATE MATERIALIZED VIEW would silently observe
--      an empty join — or fail outright. Splitting the bypass-role setup
--      into a follow-up migration means a rerun of the initial migration
--      never becomes valid on its own.
--   Therefore: enable FORCE RLS → create the BYPASSRLS role → create the
--   MV under that role → define the SECURITY DEFINER refresh function,
--   all within this migration.

CREATE SCHEMA IF NOT EXISTS subscription_svc;

-- ---------- Seller subscription plans ----------
CREATE TABLE subscription_svc.subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    tier INT NOT NULL DEFAULT 0,
    price_amount BIGINT NOT NULL DEFAULT 0,
    price_currency VARCHAR(3) NOT NULL DEFAULT 'JPY',
    features JSONB NOT NULL DEFAULT '{}',
    stripe_price_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE TABLE subscription_svc.seller_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    seller_id UUID NOT NULL,
    plan_id UUID NOT NULL REFERENCES subscription_svc.subscription_plans(id),
    stripe_subscription_id VARCHAR(255),
    stripe_customer_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, seller_id)
);

-- ---------- Buyer subscription plans ----------
CREATE TABLE subscription_svc.buyer_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    price_amount BIGINT NOT NULL DEFAULT 0,
    price_currency VARCHAR(3) NOT NULL DEFAULT 'JPY',
    features JSONB NOT NULL DEFAULT '{}',
    stripe_price_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE TABLE subscription_svc.buyer_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    buyer_auth0_id VARCHAR(255) NOT NULL,
    plan_id UUID NOT NULL REFERENCES subscription_svc.buyer_plans(id),
    stripe_subscription_id VARCHAR(255),
    stripe_customer_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, buyer_auth0_id)
);

-- ---------- Indexes ----------
CREATE INDEX idx_sub_plans_tenant       ON subscription_svc.subscription_plans(tenant_id);
CREATE INDEX idx_sub_plans_tier         ON subscription_svc.subscription_plans(tenant_id, tier);
CREATE INDEX idx_seller_subs_tenant     ON subscription_svc.seller_subscriptions(tenant_id);
CREATE INDEX idx_seller_subs_seller     ON subscription_svc.seller_subscriptions(seller_id);
CREATE INDEX idx_seller_subs_plan       ON subscription_svc.seller_subscriptions(plan_id);
CREATE INDEX idx_seller_subs_status     ON subscription_svc.seller_subscriptions(tenant_id, status);

CREATE INDEX idx_buyer_plans_tenant     ON subscription_svc.buyer_plans(tenant_id);
CREATE INDEX idx_buyer_subs_tenant      ON subscription_svc.buyer_subscriptions(tenant_id);
CREATE INDEX idx_buyer_subs_buyer       ON subscription_svc.buyer_subscriptions(tenant_id, buyer_auth0_id);
CREATE INDEX idx_buyer_subs_plan        ON subscription_svc.buyer_subscriptions(plan_id);
CREATE INDEX idx_buyer_subs_status      ON subscription_svc.buyer_subscriptions(tenant_id, status);

-- ---------- Copy data from auth_svc ----------
-- Runs before FORCE RLS is applied, so the migrator role (ecmarket) can
-- read auth_svc.* and write into subscription_svc.* without bypass.
INSERT INTO subscription_svc.subscription_plans
    (id, tenant_id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at)
SELECT id, tenant_id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
FROM auth_svc.subscription_plans;

INSERT INTO subscription_svc.seller_subscriptions
    (id, tenant_id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
     current_period_start, current_period_end, canceled_at, created_at, updated_at)
SELECT id, tenant_id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
       current_period_start, current_period_end, canceled_at, created_at, updated_at
FROM auth_svc.seller_subscriptions;

INSERT INTO subscription_svc.buyer_plans
    (id, tenant_id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at)
SELECT id, tenant_id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
FROM auth_svc.buyer_plans;

INSERT INTO subscription_svc.buyer_subscriptions
    (id, tenant_id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
     current_period_start, current_period_end, canceled_at, created_at, updated_at)
SELECT id, tenant_id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
       current_period_start, current_period_end, canceled_at, created_at, updated_at
FROM auth_svc.buyer_subscriptions;

-- ---------- Row-Level Security (mirrors auth_svc + 000015 FORCE) ----------
ALTER TABLE subscription_svc.subscription_plans     ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscription_svc.seller_subscriptions   ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscription_svc.buyer_plans            ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscription_svc.buyer_subscriptions    ENABLE ROW LEVEL SECURITY;

ALTER TABLE subscription_svc.subscription_plans     FORCE ROW LEVEL SECURITY;
ALTER TABLE subscription_svc.seller_subscriptions   FORCE ROW LEVEL SECURITY;
ALTER TABLE subscription_svc.buyer_plans            FORCE ROW LEVEL SECURITY;
ALTER TABLE subscription_svc.buyer_subscriptions    FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON subscription_svc.subscription_plans
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON subscription_svc.seller_subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON subscription_svc.buyer_plans
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON subscription_svc.buyer_subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- ---------- BYPASSRLS helper role for the cross-tenant materialized view ----
-- catalog_svc.seller_plan_boost spans every tenant, so neither its CREATE
-- nor its REFRESH can run under a role that is subject to the FORCE RLS
-- policies above. We create a dedicated NOLOGIN BYPASSRLS role, grant the
-- migrator/app role membership so `SET LOCAL ROLE` succeeds, and grant the
-- narrow SELECTs the MV actually needs. The application role never gets
-- BYPASSRLS itself — only EXECUTE on the SECURITY DEFINER refresh function.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ecmarket_rls_bypass') THEN
        CREATE ROLE ecmarket_rls_bypass NOLOGIN BYPASSRLS;
    ELSE
        ALTER ROLE ecmarket_rls_bypass BYPASSRLS;
    END IF;
END
$$;

GRANT ecmarket_rls_bypass TO ecmarket;

GRANT USAGE ON SCHEMA subscription_svc, catalog_svc, auth_svc TO ecmarket_rls_bypass;
GRANT SELECT ON subscription_svc.seller_subscriptions  TO ecmarket_rls_bypass;
GRANT SELECT ON subscription_svc.subscription_plans    TO ecmarket_rls_bypass;
GRANT SELECT ON auth_svc.sellers                       TO ecmarket_rls_bypass;

-- ---------- Rebuild catalog_svc.seller_plan_boost ----------
-- The old MV joined auth_svc.seller_subscriptions and
-- auth_svc.subscription_plans. Drop it and recreate it against
-- subscription_svc.* under the bypass role so the CREATE observes all
-- tenants and the owner has permission to REFRESH going forward.
DROP MATERIALIZED VIEW IF EXISTS catalog_svc.seller_plan_boost;

SET LOCAL ROLE ecmarket_rls_bypass;

CREATE MATERIALIZED VIEW catalog_svc.seller_plan_boost AS
SELECT
    s.tenant_id,
    s.id AS seller_id,
    COALESCE(sp.tier, 0) AS plan_tier,
    COALESCE(sp.slug, 'free') AS plan_slug,
    COALESCE((sp.features->>'search_boost')::float, 1.0) AS search_boost,
    COALESCE((sp.features->>'promoted_results')::int, 0) AS promoted_results
FROM auth_svc.sellers s
LEFT JOIN subscription_svc.seller_subscriptions ss
    ON ss.seller_id = s.id AND ss.tenant_id = s.tenant_id AND ss.status = 'active'
LEFT JOIN subscription_svc.subscription_plans sp
    ON sp.id = ss.plan_id AND sp.tenant_id = s.tenant_id;

CREATE UNIQUE INDEX idx_seller_plan_boost_pk
    ON catalog_svc.seller_plan_boost(tenant_id, seller_id);

RESET ROLE;

GRANT SELECT ON catalog_svc.seller_plan_boost TO ecmarket;

-- ---------- SECURITY DEFINER refresh wrapper --------------------------------
-- The application calls this instead of REFRESH MATERIALIZED VIEW directly.
-- Owned by the bypass role so it executes with BYPASSRLS regardless of the
-- caller. search_path is pinned to pg_catalog to neutralise the usual
-- SECURITY DEFINER escalation surface.
CREATE OR REPLACE FUNCTION catalog_svc.refresh_seller_plan_boost()
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY catalog_svc.seller_plan_boost;
END;
$$;

ALTER FUNCTION catalog_svc.refresh_seller_plan_boost() OWNER TO ecmarket_rls_bypass;

REVOKE ALL ON FUNCTION catalog_svc.refresh_seller_plan_boost() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION catalog_svc.refresh_seller_plan_boost() TO ecmarket;

-- ---------- Drop originals ----------
DROP TABLE auth_svc.buyer_subscriptions;
DROP TABLE auth_svc.buyer_plans;
DROP TABLE auth_svc.seller_subscriptions;
DROP TABLE auth_svc.subscription_plans;
