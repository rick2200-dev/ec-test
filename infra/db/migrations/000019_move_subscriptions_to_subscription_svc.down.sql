-- Inverse of 000019 up: recreate auth_svc subscription tables, copy data back,
-- rebuild catalog_svc.seller_plan_boost against auth_svc, then drop
-- subscription_svc.
--
-- Ordering mirrors up: the materialized view is a cross-tenant projection
-- over FORCE-RLS tables, so the bypass role is reused here for both reading
-- subscription_svc.* (to copy data back) and for creating the MV against
-- auth_svc.*.

-- ---------- Tear down the refresh function + MV first ----------
-- The function and MV are owned by ecmarket_rls_bypass; drop them before
-- touching the source tables so we don't leave dangling references.
DROP FUNCTION IF EXISTS catalog_svc.refresh_seller_plan_boost();
DROP MATERIALIZED VIEW IF EXISTS catalog_svc.seller_plan_boost;

-- ---------- Recreate auth_svc tables ----------
CREATE TABLE auth_svc.subscription_plans (
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

CREATE TABLE auth_svc.seller_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    seller_id UUID NOT NULL,
    plan_id UUID NOT NULL REFERENCES auth_svc.subscription_plans(id),
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

CREATE TABLE auth_svc.buyer_plans (
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

CREATE TABLE auth_svc.buyer_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    buyer_auth0_id VARCHAR(255) NOT NULL,
    plan_id UUID NOT NULL REFERENCES auth_svc.buyer_plans(id),
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
CREATE INDEX idx_auth_sub_plans_tenant     ON auth_svc.subscription_plans(tenant_id);
CREATE INDEX idx_auth_sub_plans_tier       ON auth_svc.subscription_plans(tenant_id, tier);
CREATE INDEX idx_auth_seller_subs_tenant   ON auth_svc.seller_subscriptions(tenant_id);
CREATE INDEX idx_auth_seller_subs_seller   ON auth_svc.seller_subscriptions(seller_id);
CREATE INDEX idx_auth_seller_subs_plan     ON auth_svc.seller_subscriptions(plan_id);
CREATE INDEX idx_auth_seller_subs_status   ON auth_svc.seller_subscriptions(tenant_id, status);
CREATE INDEX idx_auth_buyer_plans_tenant   ON auth_svc.buyer_plans(tenant_id);
CREATE INDEX idx_auth_buyer_subs_tenant    ON auth_svc.buyer_subscriptions(tenant_id);
CREATE INDEX idx_auth_buyer_subs_buyer     ON auth_svc.buyer_subscriptions(tenant_id, buyer_auth0_id);
CREATE INDEX idx_auth_buyer_subs_plan      ON auth_svc.buyer_subscriptions(plan_id);
CREATE INDEX idx_auth_buyer_subs_status    ON auth_svc.buyer_subscriptions(tenant_id, status);

-- ---------- Copy data back ----------
-- subscription_svc.* is still under FORCE RLS at this point, so a plain
-- SELECT by `ecmarket` would return zero rows. Switch to the bypass role
-- for the duration of the copy, then reset. (Grants on subscription_svc.*
-- to the bypass role were created in up; they still hold here.)
SET LOCAL ROLE ecmarket_rls_bypass;

INSERT INTO auth_svc.subscription_plans
    (id, tenant_id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at)
SELECT id, tenant_id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
FROM subscription_svc.subscription_plans;

INSERT INTO auth_svc.seller_subscriptions
    (id, tenant_id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
     current_period_start, current_period_end, canceled_at, created_at, updated_at)
SELECT id, tenant_id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
       current_period_start, current_period_end, canceled_at, created_at, updated_at
FROM subscription_svc.seller_subscriptions;

INSERT INTO auth_svc.buyer_plans
    (id, tenant_id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at)
SELECT id, tenant_id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
FROM subscription_svc.buyer_plans;

INSERT INTO auth_svc.buyer_subscriptions
    (id, tenant_id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
     current_period_start, current_period_end, canceled_at, created_at, updated_at)
SELECT id, tenant_id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
       current_period_start, current_period_end, canceled_at, created_at, updated_at
FROM subscription_svc.buyer_subscriptions;

RESET ROLE;

-- ---------- RLS on auth_svc.* (matches original pre-Phase-2 state) ----------
ALTER TABLE auth_svc.subscription_plans     ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.seller_subscriptions   ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_plans            ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_subscriptions    ENABLE ROW LEVEL SECURITY;

ALTER TABLE auth_svc.subscription_plans     FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.seller_subscriptions   FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_plans            FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_subscriptions    FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON auth_svc.subscription_plans
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON auth_svc.seller_subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON auth_svc.buyer_plans
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
CREATE POLICY tenant_isolation ON auth_svc.buyer_subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- ---------- Rebuild catalog_svc.seller_plan_boost against auth_svc ----------
-- Must be created under the bypass role for the same reason as in up —
-- FORCE RLS on auth_svc.seller_subscriptions / subscription_plans would
-- otherwise hide every tenant's rows from the owning role.
GRANT SELECT ON auth_svc.seller_subscriptions TO ecmarket_rls_bypass;
GRANT SELECT ON auth_svc.subscription_plans   TO ecmarket_rls_bypass;

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
LEFT JOIN auth_svc.seller_subscriptions ss
    ON ss.seller_id = s.id AND ss.tenant_id = s.tenant_id AND ss.status = 'active'
LEFT JOIN auth_svc.subscription_plans sp
    ON sp.id = ss.plan_id AND sp.tenant_id = s.tenant_id;

CREATE UNIQUE INDEX idx_seller_plan_boost_pk
    ON catalog_svc.seller_plan_boost(tenant_id, seller_id);

RESET ROLE;

GRANT SELECT ON catalog_svc.seller_plan_boost TO ecmarket;

-- Recreate the SECURITY DEFINER refresh wrapper so app code keeps working
-- after a rollback. Owned by the bypass role, executable by ecmarket.
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

-- ---------- Drop subscription_svc ----------
DROP TABLE subscription_svc.buyer_subscriptions;
DROP TABLE subscription_svc.buyer_plans;
DROP TABLE subscription_svc.seller_subscriptions;
DROP TABLE subscription_svc.subscription_plans;
DROP SCHEMA subscription_svc;

-- We intentionally leave the `ecmarket_rls_bypass` role in place. Dropping
-- it requires reassigning every object it still owns (future migrations
-- may add more SECURITY DEFINER helpers). The role is harmless when unused.
