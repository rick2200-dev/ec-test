-- Platform admins: per-tenant administrator hierarchy.
-- Roles: super_admin (can manage other admins) > admin > support.
CREATE TABLE auth_svc.platform_admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id) ON DELETE CASCADE,
    auth0_user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'support'
        CHECK (role IN ('super_admin','admin','support')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, auth0_user_id)
);

CREATE INDEX idx_platform_admins_tenant_role
    ON auth_svc.platform_admins(tenant_id, role);

ALTER TABLE auth_svc.platform_admins ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON auth_svc.platform_admins
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Enforce seller_users.role to be one of the known values and add an index
-- supporting "count by role within seller" lookups used by safeguards.
ALTER TABLE auth_svc.seller_users
    ADD CONSTRAINT seller_users_role_check
    CHECK (role IN ('owner','admin','member'));

CREATE INDEX idx_seller_users_role
    ON auth_svc.seller_users(tenant_id, seller_id, role);

-- RBAC audit log: every grant/revoke/role-change/transfer-ownership event.
CREATE TABLE auth_svc.rbac_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth_svc.tenants(id) ON DELETE CASCADE,
    actor_auth0_user_id VARCHAR(255) NOT NULL,
    target_auth0_user_id VARCHAR(255) NOT NULL,
    scope VARCHAR(50) NOT NULL
        CHECK (scope IN ('seller_user','platform_admin')),
    scope_id UUID,
    action VARCHAR(50) NOT NULL
        CHECK (action IN ('grant','revoke','role_change','transfer_ownership')),
    before_role VARCHAR(50),
    after_role VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rbac_audit_tenant_created
    ON auth_svc.rbac_audit_log(tenant_id, created_at DESC);

ALTER TABLE auth_svc.rbac_audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON auth_svc.rbac_audit_log
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
