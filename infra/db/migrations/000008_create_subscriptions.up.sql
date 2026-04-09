-- Subscription plans: defines the plan tiers available within a tenant.
CREATE TABLE auth_svc.subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id),
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

-- Seller subscriptions: tracks which plan each seller is currently on.
CREATE TABLE auth_svc.seller_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id),
    seller_id UUID NOT NULL REFERENCES auth_svc.sellers(id),
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

-- Indexes
CREATE INDEX idx_subscription_plans_tenant ON auth_svc.subscription_plans(tenant_id);
CREATE INDEX idx_subscription_plans_tier ON auth_svc.subscription_plans(tenant_id, tier);
CREATE INDEX idx_seller_subscriptions_tenant ON auth_svc.seller_subscriptions(tenant_id);
CREATE INDEX idx_seller_subscriptions_seller ON auth_svc.seller_subscriptions(seller_id);
CREATE INDEX idx_seller_subscriptions_plan ON auth_svc.seller_subscriptions(plan_id);
CREATE INDEX idx_seller_subscriptions_status ON auth_svc.seller_subscriptions(tenant_id, status);

-- Row-Level Security
ALTER TABLE auth_svc.subscription_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.seller_subscriptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON auth_svc.subscription_plans
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON auth_svc.seller_subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Seed default plans for every existing tenant.
-- Free plan
INSERT INTO auth_svc.subscription_plans (tenant_id, name, slug, tier, price_amount, price_currency, features, status)
SELECT id, 'Free', 'free', 0, 0, 'JPY',
       '{"max_products": 10, "search_boost": 1.0, "featured_slots": 0, "promoted_results": 0}'::jsonb,
       'active'
FROM auth_svc.tenants;

-- Standard plan
INSERT INTO auth_svc.subscription_plans (tenant_id, name, slug, tier, price_amount, price_currency, features, status)
SELECT id, 'Standard', 'standard', 1, 9800, 'JPY',
       '{"max_products": 50, "search_boost": 1.5, "featured_slots": 2, "promoted_results": 0}'::jsonb,
       'active'
FROM auth_svc.tenants;

-- Premium plan
INSERT INTO auth_svc.subscription_plans (tenant_id, name, slug, tier, price_amount, price_currency, features, status)
SELECT id, 'Premium', 'premium', 2, 29800, 'JPY',
       '{"max_products": -1, "search_boost": 2.5, "featured_slots": 5, "promoted_results": 3}'::jsonb,
       'active'
FROM auth_svc.tenants;

-- Assign all existing sellers to the Free plan.
INSERT INTO auth_svc.seller_subscriptions (tenant_id, seller_id, plan_id, status)
SELECT s.tenant_id, s.id, p.id, 'active'
FROM auth_svc.sellers s
JOIN auth_svc.subscription_plans p ON p.tenant_id = s.tenant_id AND p.slug = 'free';
