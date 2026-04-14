CREATE TABLE auth_svc.sellers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth0_org_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    stripe_account_id VARCHAR(255),
    commission_rate_bps INT NOT NULL DEFAULT 1000,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth_svc.seller_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL REFERENCES auth_svc.sellers(id),
    auth0_user_id VARCHAR(255) NOT NULL UNIQUE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_sellers_status ON auth_svc.sellers(status);
CREATE INDEX idx_seller_users_seller ON auth_svc.seller_users(seller_id);
