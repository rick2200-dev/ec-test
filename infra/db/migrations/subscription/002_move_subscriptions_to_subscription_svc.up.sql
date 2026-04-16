-- Creates the subscription_svc schema and its 4 tables. Idempotent so
-- it works both on the shared-cluster DB (where an earlier version of
-- this migration already created these tables and copied data over from
-- auth_svc) and on a fresh subscription-only DB (Phase 3 split).
--
-- The old up-script also copied rows from auth_svc.*, dropped the
-- catalog_svc.seller_plan_boost matview, and dropped the auth_svc
-- originals. Those cross-schema parts are obsolete — catalog/007 drops
-- the matview, and auth_svc subscription tables no longer exist on any
-- live DB. So this version just owns schema + tables + seed data.

CREATE SCHEMA IF NOT EXISTS subscription_svc;

-- ---------- Seller subscription plans ----------
CREATE TABLE IF NOT EXISTS subscription_svc.subscription_plans (
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

CREATE TABLE IF NOT EXISTS subscription_svc.seller_subscriptions (
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
CREATE TABLE IF NOT EXISTS subscription_svc.buyer_plans (
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

CREATE TABLE IF NOT EXISTS subscription_svc.buyer_subscriptions (
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
CREATE INDEX IF NOT EXISTS idx_sub_plans_tier         ON subscription_svc.subscription_plans(tier);
CREATE INDEX IF NOT EXISTS idx_seller_subs_seller     ON subscription_svc.seller_subscriptions(seller_id);
CREATE INDEX IF NOT EXISTS idx_seller_subs_plan       ON subscription_svc.seller_subscriptions(plan_id);
CREATE INDEX IF NOT EXISTS idx_seller_subs_status     ON subscription_svc.seller_subscriptions(status);

CREATE INDEX IF NOT EXISTS idx_buyer_subs_buyer       ON subscription_svc.buyer_subscriptions(buyer_auth0_id);
CREATE INDEX IF NOT EXISTS idx_buyer_subs_plan        ON subscription_svc.buyer_subscriptions(plan_id);
CREATE INDEX IF NOT EXISTS idx_buyer_subs_status      ON subscription_svc.buyer_subscriptions(status);

-- ---------- Seed default plans (idempotent) ----------
INSERT INTO subscription_svc.subscription_plans (name, slug, tier, price_amount, price_currency, features, status)
VALUES ('Free', 'free', 0, 0, 'JPY',
        '{"max_products": 10, "search_boost": 1.0, "featured_slots": 0, "promoted_results": 0}'::jsonb,
        'active')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subscription_svc.subscription_plans (name, slug, tier, price_amount, price_currency, features, status)
VALUES ('Standard', 'standard', 1, 9800, 'JPY',
        '{"max_products": 50, "search_boost": 1.5, "featured_slots": 2, "promoted_results": 0}'::jsonb,
        'active')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO subscription_svc.subscription_plans (name, slug, tier, price_amount, price_currency, features, status)
VALUES ('Premium', 'premium', 2, 29800, 'JPY',
        '{"max_products": -1, "search_boost": 2.5, "featured_slots": 5, "promoted_results": 3}'::jsonb,
        'active')
ON CONFLICT (slug) DO NOTHING;
