-- Buyers: minimal profile record created on first Auth0 login via the
-- POST /internal/buyers/upsert endpoint. Downstream services still key off
-- the Auth0 `sub` claim directly; this table exists so profile data
-- (email / display name) is queryable without hitting Auth0 every time.
CREATE TABLE auth_svc.buyers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id),
    auth0_sub VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, auth0_sub)
);

CREATE INDEX idx_buyers_tenant ON auth_svc.buyers(tenant_id);

-- Row-Level Security — same pattern as sellers/seller_users.
ALTER TABLE auth_svc.buyers ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON auth_svc.buyers
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
