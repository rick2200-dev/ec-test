-- Buyer subscription plans: defines the plan tiers available for buyers.
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

-- Buyer subscriptions: tracks which plan each buyer is currently on.
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

-- Indexes
CREATE INDEX idx_buyer_subscriptions_buyer ON auth_svc.buyer_subscriptions(buyer_auth0_id);
CREATE INDEX idx_buyer_subscriptions_plan ON auth_svc.buyer_subscriptions(plan_id);
CREATE INDEX idx_buyer_subscriptions_status ON auth_svc.buyer_subscriptions(status);

-- Seed default Buyer Premium plan.
INSERT INTO auth_svc.buyer_plans (name, slug, price_amount, price_currency, features, status)
VALUES ('Premium', 'buyer-premium', 300, 'JPY',
        '{"free_shipping": true}'::jsonb,
        'active');

-- Add shipping_fee column to orders.
ALTER TABLE order_svc.orders ADD COLUMN shipping_fee BIGINT NOT NULL DEFAULT 0;
