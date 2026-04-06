CREATE TABLE auth_svc.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_svc.sellers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id),
    auth0_org_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    stripe_account_id VARCHAR(255),
    commission_rate_bps INT NOT NULL DEFAULT 1000,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE TABLE auth_svc.seller_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id),
    seller_id UUID NOT NULL REFERENCES auth_svc.sellers(id),
    auth0_user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, auth0_user_id)
);

-- Indexes
CREATE INDEX idx_sellers_tenant ON auth_svc.sellers(tenant_id);
CREATE INDEX idx_sellers_status ON auth_svc.sellers(tenant_id, status);
CREATE INDEX idx_seller_users_tenant ON auth_svc.seller_users(tenant_id);
CREATE INDEX idx_seller_users_seller ON auth_svc.seller_users(seller_id);

-- Row-Level Security
ALTER TABLE auth_svc.sellers ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.seller_users ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON auth_svc.sellers
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON auth_svc.seller_users
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
