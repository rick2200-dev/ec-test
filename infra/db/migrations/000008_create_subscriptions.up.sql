-- Subscription plans: defines the plan tiers available.
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

-- Seller subscriptions: tracks which plan each seller is currently on.
CREATE TABLE auth_svc.seller_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL REFERENCES auth_svc.sellers(id) UNIQUE,
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

-- Indexes
CREATE INDEX idx_subscription_plans_tier ON auth_svc.subscription_plans(tier);
CREATE INDEX idx_seller_subscriptions_seller ON auth_svc.seller_subscriptions(seller_id);
CREATE INDEX idx_seller_subscriptions_plan ON auth_svc.seller_subscriptions(plan_id);
CREATE INDEX idx_seller_subscriptions_status ON auth_svc.seller_subscriptions(status);

-- Seed default plans.
-- Free plan
INSERT INTO auth_svc.subscription_plans (name, slug, tier, price_amount, price_currency, features, status)
VALUES ('Free', 'free', 0, 0, 'JPY',
        '{"max_products": 10, "search_boost": 1.0, "featured_slots": 0, "promoted_results": 0}'::jsonb,
        'active');

-- Standard plan
INSERT INTO auth_svc.subscription_plans (name, slug, tier, price_amount, price_currency, features, status)
VALUES ('Standard', 'standard', 1, 9800, 'JPY',
        '{"max_products": 50, "search_boost": 1.5, "featured_slots": 2, "promoted_results": 0}'::jsonb,
        'active');

-- Premium plan
INSERT INTO auth_svc.subscription_plans (name, slug, tier, price_amount, price_currency, features, status)
VALUES ('Premium', 'premium', 2, 29800, 'JPY',
        '{"max_products": -1, "search_boost": 2.5, "featured_slots": 5, "promoted_results": 3}'::jsonb,
        'active');

-- Assign all existing sellers to the Free plan.
INSERT INTO auth_svc.seller_subscriptions (seller_id, plan_id, status)
SELECT s.id, p.id, 'active'
FROM auth_svc.sellers s
JOIN auth_svc.subscription_plans p ON p.slug = 'free';
