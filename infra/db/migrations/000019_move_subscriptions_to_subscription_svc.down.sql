-- Inverse of 000019 up: recreate auth_svc subscription tables, copy data back,
-- rebuild catalog_svc.seller_plan_boost against auth_svc, then drop
-- subscription_svc.

-- ---------- Tear down the refresh function + MV first ----------
DROP FUNCTION IF EXISTS catalog_svc.refresh_seller_plan_boost();
DROP MATERIALIZED VIEW IF EXISTS catalog_svc.seller_plan_boost;

-- ---------- Recreate auth_svc tables ----------
CREATE TABLE auth_svc.subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    tier INT NOT NULL DEFAULT 0,
    price_amount BIGINT NOT NULL DEFAULT 0,
    price_currency VARCHAR(3) NOT NULL DEFAULT 'JPY',
    features JSONB NOT NULL DEFAULT '{}',
    stripe_price_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_svc.seller_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL UNIQUE,
    plan_id UUID NOT NULL REFERENCES auth_svc.subscription_plans(id),
    stripe_subscription_id VARCHAR(255),
    stripe_customer_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_svc.buyer_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    price_amount BIGINT NOT NULL DEFAULT 0,
    price_currency VARCHAR(3) NOT NULL DEFAULT 'JPY',
    features JSONB NOT NULL DEFAULT '{}',
    stripe_price_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_svc.buyer_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_auth0_id VARCHAR(255) NOT NULL UNIQUE,
    plan_id UUID NOT NULL REFERENCES auth_svc.buyer_plans(id),
    stripe_subscription_id VARCHAR(255),
    stripe_customer_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------- Indexes ----------
CREATE INDEX idx_auth_sub_plans_tier       ON auth_svc.subscription_plans(tier);
CREATE INDEX idx_auth_seller_subs_seller   ON auth_svc.seller_subscriptions(seller_id);
CREATE INDEX idx_auth_seller_subs_plan     ON auth_svc.seller_subscriptions(plan_id);
CREATE INDEX idx_auth_seller_subs_status   ON auth_svc.seller_subscriptions(status);
CREATE INDEX idx_auth_buyer_subs_buyer     ON auth_svc.buyer_subscriptions(buyer_auth0_id);
CREATE INDEX idx_auth_buyer_subs_plan      ON auth_svc.buyer_subscriptions(plan_id);
CREATE INDEX idx_auth_buyer_subs_status    ON auth_svc.buyer_subscriptions(status);

-- ---------- Copy data back ----------
INSERT INTO auth_svc.subscription_plans
    (id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at)
SELECT id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
FROM subscription_svc.subscription_plans;

INSERT INTO auth_svc.seller_subscriptions
    (id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
     current_period_start, current_period_end, canceled_at, created_at, updated_at)
SELECT id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
       current_period_start, current_period_end, canceled_at, created_at, updated_at
FROM subscription_svc.seller_subscriptions;

INSERT INTO auth_svc.buyer_plans
    (id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at)
SELECT id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
FROM subscription_svc.buyer_plans;

INSERT INTO auth_svc.buyer_subscriptions
    (id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
     current_period_start, current_period_end, canceled_at, created_at, updated_at)
SELECT id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
       current_period_start, current_period_end, canceled_at, created_at, updated_at
FROM subscription_svc.buyer_subscriptions;

-- ---------- Rebuild catalog_svc.seller_plan_boost against auth_svc ----------
CREATE MATERIALIZED VIEW catalog_svc.seller_plan_boost AS
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

CREATE UNIQUE INDEX idx_seller_plan_boost_pk
    ON catalog_svc.seller_plan_boost(seller_id);

GRANT SELECT ON catalog_svc.seller_plan_boost TO ecmarket;

-- Recreate the SECURITY DEFINER refresh wrapper so app code keeps working
-- after a rollback. Executable by ecmarket.
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

REVOKE ALL ON FUNCTION catalog_svc.refresh_seller_plan_boost() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION catalog_svc.refresh_seller_plan_boost() TO ecmarket;

-- ---------- Drop subscription_svc ----------
DROP TABLE subscription_svc.buyer_subscriptions;
DROP TABLE subscription_svc.buyer_plans;
DROP TABLE subscription_svc.seller_subscriptions;
DROP TABLE subscription_svc.subscription_plans;
DROP SCHEMA subscription_svc;
