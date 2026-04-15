-- Platform admins: administrator hierarchy.
-- Roles: super_admin (can manage other admins) > admin > support.
CREATE TABLE auth_svc.platform_admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth0_user_id VARCHAR(255) NOT NULL UNIQUE,
    role VARCHAR(50) NOT NULL DEFAULT 'support'
        CHECK (role IN ('super_admin','admin','support')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_platform_admins_role
    ON auth_svc.platform_admins(role);

-- Enforce seller_users.role to be one of the known values and add an index
-- supporting "count by role within seller" lookups used by safeguards.
ALTER TABLE auth_svc.seller_users
    ADD CONSTRAINT seller_users_role_check
    CHECK (role IN ('owner','admin','member'));

CREATE INDEX idx_seller_users_role
    ON auth_svc.seller_users(seller_id, role);

-- RBAC audit log: every grant/revoke/role-change/transfer-ownership event.
CREATE TABLE auth_svc.rbac_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

CREATE INDEX idx_rbac_audit_created
    ON auth_svc.rbac_audit_log(created_at DESC);
