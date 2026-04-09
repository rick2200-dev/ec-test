-- Buyer subscription plans: defines the plan tiers available for buyers within a tenant.
CREATE TABLE auth_svc.buyer_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id),
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

-- Buyer subscriptions: tracks which plan each buyer is currently on.
CREATE TABLE auth_svc.buyer_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id),
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

-- Indexes
CREATE INDEX idx_buyer_plans_tenant ON auth_svc.buyer_plans(tenant_id);
CREATE INDEX idx_buyer_subscriptions_tenant ON auth_svc.buyer_subscriptions(tenant_id);
CREATE INDEX idx_buyer_subscriptions_buyer ON auth_svc.buyer_subscriptions(tenant_id, buyer_auth0_id);
CREATE INDEX idx_buyer_subscriptions_plan ON auth_svc.buyer_subscriptions(plan_id);
CREATE INDEX idx_buyer_subscriptions_status ON auth_svc.buyer_subscriptions(tenant_id, status);

-- Row-Level Security
ALTER TABLE auth_svc.buyer_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_subscriptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON auth_svc.buyer_plans
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON auth_svc.buyer_subscriptions
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Seed default Buyer Premium plan for every existing tenant.
INSERT INTO auth_svc.buyer_plans (tenant_id, name, slug, price_amount, price_currency, features, status)
SELECT id, 'Premium', 'buyer-premium', 300, 'JPY',
       '{"free_shipping": true}'::jsonb,
       'active'
FROM auth_svc.tenants;

-- Add shipping_fee column to orders.
ALTER TABLE order_svc.orders ADD COLUMN shipping_fee BIGINT NOT NULL DEFAULT 0;
