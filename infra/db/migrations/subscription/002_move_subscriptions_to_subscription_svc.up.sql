-- Phase 2 of auth refactor: relocate seller/buyer plan + subscription tables
-- from auth_svc to a dedicated subscription_svc schema. The REST / gRPC
-- surface is preserved — callers (gateway, order) are rewired in application
-- code separately.

CREATE SCHEMA IF NOT EXISTS subscription_svc;

-- ---------- Seller subscription plans ----------
CREATE TABLE subscription_svc.subscription_plans (
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

CREATE TABLE subscription_svc.seller_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL UNIQUE,
    plan_id UUID NOT NULL REFERENCES subscription_svc.subscription_plans(id),
    stripe_subscription_id VARCHAR(255),
    stripe_customer_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------- Buyer subscription plans ----------
CREATE TABLE subscription_svc.buyer_plans (
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

CREATE TABLE subscription_svc.buyer_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_auth0_id VARCHAR(255) NOT NULL UNIQUE,
    plan_id UUID NOT NULL REFERENCES subscription_svc.buyer_plans(id),
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
CREATE INDEX idx_sub_plans_tier         ON subscription_svc.subscription_plans(tier);
CREATE INDEX idx_seller_subs_seller     ON subscription_svc.seller_subscriptions(seller_id);
CREATE INDEX idx_seller_subs_plan       ON subscription_svc.seller_subscriptions(plan_id);
CREATE INDEX idx_seller_subs_status     ON subscription_svc.seller_subscriptions(status);

CREATE INDEX idx_buyer_subs_buyer       ON subscription_svc.buyer_subscriptions(buyer_auth0_id);
CREATE INDEX idx_buyer_subs_plan        ON subscription_svc.buyer_subscriptions(plan_id);
CREATE INDEX idx_buyer_subs_status      ON subscription_svc.buyer_subscriptions(status);

-- ---------- Copy data from auth_svc ----------
INSERT INTO subscription_svc.subscription_plans
    (id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at)
SELECT id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
FROM auth_svc.subscription_plans;

INSERT INTO subscription_svc.seller_subscriptions
    (id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
     current_period_start, current_period_end, canceled_at, created_at, updated_at)
SELECT id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
       current_period_start, current_period_end, canceled_at, created_at, updated_at
FROM auth_svc.seller_subscriptions;

INSERT INTO subscription_svc.buyer_plans
    (id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at)
SELECT id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
FROM auth_svc.buyer_plans;

INSERT INTO subscription_svc.buyer_subscriptions
    (id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
     current_period_start, current_period_end, canceled_at, created_at, updated_at)
SELECT id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
       current_period_start, current_period_end, canceled_at, created_at, updated_at
FROM auth_svc.buyer_subscriptions;

-- ---------- Rebuild catalog_svc.seller_plan_boost ----------
-- Drop the old MV (joined auth_svc) and recreate it against subscription_svc.
DROP MATERIALIZED VIEW IF EXISTS catalog_svc.seller_plan_boost;

CREATE MATERIALIZED VIEW catalog_svc.seller_plan_boost AS
SELECT
    s.id AS seller_id,
    COALESCE(sp.tier, 0) AS plan_tier,
    COALESCE(sp.slug, 'free') AS plan_slug,
    COALESCE((sp.features->>'search_boost')::float, 1.0) AS search_boost,
    COALESCE((sp.features->>'promoted_results')::int, 0) AS promoted_results
FROM auth_svc.sellers s
LEFT JOIN subscription_svc.seller_subscriptions ss
    ON ss.seller_id = s.id AND ss.status = 'active'
LEFT JOIN subscription_svc.subscription_plans sp
    ON sp.id = ss.plan_id;

CREATE UNIQUE INDEX idx_seller_plan_boost_pk
    ON catalog_svc.seller_plan_boost(seller_id);

-- ---------- SECURITY DEFINER refresh wrapper --------------------------------
-- The application calls this instead of REFRESH MATERIALIZED VIEW directly.
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

-- ---------- Drop originals ----------
DROP TABLE auth_svc.buyer_subscriptions;
DROP TABLE auth_svc.buyer_plans;
DROP TABLE auth_svc.seller_subscriptions;
DROP TABLE auth_svc.subscription_plans;
